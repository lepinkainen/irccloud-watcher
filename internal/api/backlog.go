package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"irccloud-watcher/internal/storage"
	"irccloud-watcher/internal/utils"
)

func (c *IRCCloudClient) processBacklog(backlogURL string) error {
	// The backlog URL is just a path, we need to prepend the correct API host
	if !strings.HasPrefix(backlogURL, "http") {
		if c.apiHost != "" {
			// APIHost already includes the protocol (https://)
			backlogURL = c.apiHost + backlogURL
		} else {
			// Fallback to www.irccloud.com if no API host is available
			backlogURL = "https://www.irccloud.com" + backlogURL
		}
	}

	log.Printf("🔍 Requesting backlog from URL: %s", backlogURL)

	req, err := http.NewRequest("GET", backlogURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("could not create backlog request: %w", err)
	}

	req.Header.Set("User-Agent", "irccloud-watcher/0.1.0")
	req.Header.Set("Cookie", "session="+c.session)
	req.Header.Set("Accept-Encoding", "gzip")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not perform backlog request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backlog request failed with status: %s", resp.Status)
	}

	var reader io.Reader = resp.Body

	// Check if the response is gzipped
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("could not create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	var backlogMessages []IRCMessage
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&backlogMessages); err != nil {
		return fmt.Errorf("could not decode backlog messages: %w", err)
	}

	log.Printf("Processing %d backlog messages", len(backlogMessages))

	for _, ircMsg := range backlogMessages {
		// Skip message if ignored or not in allowed channels
		if ircMsg.Type != "buffer_msg" || c.ignoredChannelSet[ircMsg.Chan] || (len(c.channels) > 0 && !c.channelSet[ircMsg.Chan]) {
			continue
		}

		// Check if we've seen this EID before (skip if duplicate)
		if c.isEIDSeen(ircMsg.EID) {
			if os.Getenv("IRCCLOUD_DEBUG") == "true" {
				log.Printf("🔄 Duplicate backlog message filtered: EID=%d, Channel=%s", ircMsg.EID, ircMsg.Chan)
			}
			continue
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
			log.Printf("🔍 Processing backlog message: Channel=%s, From=%s, EID=%d, Time=%d, Converted=%s", ircMsg.Chan, ircMsg.From, ircMsg.EID, ircMsg.Time, msgTime.Format(time.RFC3339))
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
			log.Printf("❌ Error inserting backlog message into DB: %v", err)
		} else if os.Getenv("IRCCLOUD_DEBUG") == "true" {
			log.Printf("✅ Backlog message stored successfully: EID=%d", ircMsg.EID)
		}
	}

	log.Println("Finished processing backlog")
	return nil
}
