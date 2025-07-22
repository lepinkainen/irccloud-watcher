package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"irccloud-watcher/internal/config"
	"irccloud-watcher/internal/storage"
	"irccloud-watcher/internal/utils"

	"github.com/gorilla/websocket"
)

// Run starts the client and listens for messages with automatic reconnection.
func (c *IRCCloudClient) Run(channels, ignoredChannels []string, connConfig *config.ConnectionConfig) {
	// Store filtering parameters and connection config
	c.channels = channels
	c.ignoredChannels = ignoredChannels
	c.channelSet = make(map[string]bool)
	for _, ch := range channels {
		c.channelSet[ch] = true
	}
	c.ignoredChannelSet = make(map[string]bool)
	for _, ch := range ignoredChannels {
		c.ignoredChannelSet[ch] = true
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Main loop with reconnection handling
	for {
		select {
		case <-interrupt:
			log.Println("🛑 Interrupt received, shutting down...")
			c.Close()
			return
		case <-c.ctx.Done():
			log.Println("🛑 Context cancelled, shutting down...")
			return
		default:
			switch {
			case c.getState() == StateConnected && c.conn != nil:
				// Run the message loop until connection fails
				if err := c.runMessageLoop(); err != nil {
					log.Printf("❌ Message loop error: %v", err)
					c.setState(StateError)

					// Close broken connection
					if c.conn != nil {
						c.conn.Close()
						c.conn = nil
					}

					// Attempt reconnection
					log.Println("🔄 Attempting to reconnect...")
					if reconnectErr := c.connectWithRetry(); reconnectErr != nil {
						log.Printf("❌ Reconnection failed: %v", reconnectErr)
						if c.retryCount >= c.connConfig.MaxRetryAttempts {
							log.Println("❌ Max retry attempts reached, exiting...")
							return
						}
					}
				}
			case c.getState() == StateDisconnected:
				// Connection was closed externally
				return
			default:
				// Wait a bit before checking again
				time.Sleep(time.Second)
			}
		}
	}
}

// runMessageLoop handles the WebSocket message processing with ping/pong monitoring
func (c *IRCCloudClient) runMessageLoop() error {
	// Parse intervals from config
	heartbeatInterval, err := time.ParseDuration(c.connConfig.HeartbeatInterval)
	if err != nil {
		heartbeatInterval = 30 * time.Second
	}

	pingInterval, err := time.ParseDuration(c.connConfig.PingInterval)
	if err != nil {
		pingInterval = 60 * time.Second
	}

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	// Set up ping/pong handlers
	c.conn.SetPongHandler(func(string) error {
		if os.Getenv("IRCCLOUD_DEBUG") == "true" {
			log.Println("🏓 Received pong")
		}
		return nil
	})

	// Message reading goroutine
	done := make(chan error, 1)
	go func() {
		defer close(done)
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				done <- fmt.Errorf("read error: %w", err)
				return
			}

			if err := c.processMessage(message); err != nil {
				log.Printf("⚠️ Error processing message: %v", err)
				// Continue processing other messages
			}
		}
	}()

	// Main event loop
	for {
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("context cancelled")
		case err := <-done:
			return err
		case <-heartbeatTicker.C:
			if err := c.sendHeartbeat(); err != nil {
				return fmt.Errorf("heartbeat failed: %w", err)
			}
		case <-pingTicker.C:
			if err := c.sendPing(); err != nil {
				return fmt.Errorf("ping failed: %w", err)
			}
		}
	}
}

// processMessage handles individual WebSocket messages
func (c *IRCCloudClient) processMessage(message []byte) error {
	// Print raw message if debug mode is enabled
	if c.debugMode {
		fmt.Printf("RAW: %s\n", string(message))
	}

	var ircMsg IRCMessage
	if err := json.Unmarshal(message, &ircMsg); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	if ircMsg.Type == "oob_include" {
		var oob OOBInclude
		if err := json.Unmarshal(message, &oob); err != nil {
			return fmt.Errorf("unmarshal oob error: %w", err)
		}
		log.Printf("🔍 Received oob_include with URL: %s", oob.URL)
		if err := c.processBacklog(oob.URL); err != nil {
			log.Printf("⚠️ Error processing backlog: %v", err)
		}
		return nil
	}

	// Accept message if not ignored and either no channels specified (accept all) or channel is in allowed list
	if ircMsg.Type == "buffer_msg" && !c.ignoredChannelSet[ircMsg.Chan] && (len(c.channels) == 0 || c.channelSet[ircMsg.Chan]) {
		// Check if we've seen this EID before (skip if duplicate)
		if c.isEIDSeen(ircMsg.EID) {
			if os.Getenv("IRCCLOUD_DEBUG") == "true" {
				log.Printf("🔄 Duplicate message filtered: EID=%d, Channel=%s", ircMsg.EID, ircMsg.Chan)
			}
			return nil
		}

		cleanedMsg := utils.CleanIRCMessage(ircMsg.Msg)

		// Handle timestamp conversion - IRCCloud uses microseconds since Unix epoch
		// Live messages often have timestamp 0, so we use current time as fallback
		var msgTime time.Time
		if ircMsg.Time > 0 {
			// Convert from microseconds to seconds and nanoseconds
			seconds := ircMsg.Time / 1000000
			microseconds := ircMsg.Time % 1000000
			nanoseconds := microseconds * 1000
			msgTime = time.Unix(seconds, nanoseconds)
		} else {
			// Use current time for live messages (timestamp 0 is normal)
			msgTime = time.Now()
		}

		if os.Getenv("IRCCLOUD_DEBUG") == "true" {
			log.Printf("🔍 Processing message: Channel=%s, From=%s, EID=%d, Time=%d, Converted=%s", ircMsg.Chan, ircMsg.From, ircMsg.EID, ircMsg.Time, msgTime.Format(time.RFC3339))
		}

		log.Printf("%s <%s> %s", ircMsg.Chan, ircMsg.From, cleanedMsg)

		dbMsg := &storage.Message{
			Channel:   ircMsg.Chan,
			Timestamp: msgTime,
			Sender:    ircMsg.From,
			Message:   cleanedMsg,
			Date:      msgTime.Format("2006-01-02"),
			EID:       ircMsg.EID,
		}

		if err := c.db.InsertMessage(dbMsg); err != nil {
			log.Printf("❌ Error inserting message into DB: %v", err)
			return fmt.Errorf("error inserting message into DB: %w", err)
		}

		if os.Getenv("IRCCLOUD_DEBUG") == "true" {
			log.Printf("✅ Message stored successfully: EID=%d", ircMsg.EID)
		}

		// Fix: Use EID instead of Time for lastSeenEID tracking
		if ircMsg.EID > c.lastSeenEID {
			c.lastSeenEID = ircMsg.EID
		}
	} else if os.Getenv("IRCCLOUD_DEBUG") == "true" {
		// Debug why message was filtered out
		log.Printf("🚫 Message filtered: Type=%s, Channel=%s, Ignored=%t, ChannelAllowed=%t",
			ircMsg.Type, ircMsg.Chan, c.ignoredChannelSet[ircMsg.Chan],
			(len(c.channels) == 0 || c.channelSet[ircMsg.Chan]))
	}

	return nil
}

// sendHeartbeat sends a heartbeat message to keep the connection alive
func (c *IRCCloudClient) sendHeartbeat() error {
	heartbeat := map[string]any{
		"_method":       "heartbeat",
		"_reqid":        time.Now().Unix(),
		"last_seen_eid": c.lastSeenEID,
	}

	if err := c.conn.WriteJSON(heartbeat); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	if os.Getenv("IRCCLOUD_DEBUG") == "true" {
		log.Println("💓 Heartbeat sent")
	}
	return nil
}

// sendPing sends a WebSocket ping frame
func (c *IRCCloudClient) sendPing() error {
	if err := c.conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}

	if os.Getenv("IRCCLOUD_DEBUG") == "true" {
		log.Println("🏓 Ping sent")
	}
	return nil
}
