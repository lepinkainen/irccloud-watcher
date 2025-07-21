package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"irccloud-watcher/internal/config"
	"irccloud-watcher/internal/storage"
	"irccloud-watcher/internal/utils"

	"github.com/gorilla/websocket"
)

// ConnectionState represents the current state of the WebSocket connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateError
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// IRCCloudClient is a client for the IRCCloud API.
type IRCCloudClient struct {
	conn              *websocket.Conn
	db                *storage.DB
	lastSeenEID       int64
	session           string
	apiHost           string
	channels          []string
	ignoredChannels   []string
	channelSet        map[string]bool
	ignoredChannelSet map[string]bool

	// Connection management
	connConfig      *config.ConnectionConfig
	state           ConnectionState
	stateMutex      sync.RWMutex
	retryCount      int
	lastConnectTime time.Time
	ctx             context.Context
	cancelFunc      context.CancelFunc

	// Authentication cache
	authResp *AuthResponse
	email    string
	password string

	// Debug mode
	debugMode bool
}

// NewIRCCloudClient creates a new IRCCloudClient.
func NewIRCCloudClient(db *storage.DB) *IRCCloudClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &IRCCloudClient{
		db:         db,
		state:      StateDisconnected,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// setState safely updates the connection state
func (c *IRCCloudClient) setState(state ConnectionState) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	if c.state != state {
		log.Printf("🔄 Connection state: %s -> %s", c.state, state)
		c.state = state
	}
}

// getState safely reads the connection state
func (c *IRCCloudClient) getState() ConnectionState {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state
}

// SetConnectionConfig sets the connection configuration
func (c *IRCCloudClient) SetConnectionConfig(cfg *config.ConnectionConfig) {
	c.connConfig = cfg
}

// SetDebugMode enables or disables debug mode for printing raw messages
func (c *IRCCloudClient) SetDebugMode(debug bool) {
	c.debugMode = debug
}

// AuthResponse is the response from the IRCCloud authentication endpoint.
type AuthResponse struct {
	Success       bool   `json:"success"`
	Session       string `json:"session"`
	UID           int    `json:"uid"`
	APIHost       string `json:"api_host"`
	WebSocketHost string `json:"websocket_host"`
	WebSocketPath string `json:"websocket_path"`
	URL           string `json:"url"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// AuthError represents authentication-related errors.
type AuthError struct {
	Type    string
	Message string
	Status  int
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error [%s]: %s (status: %d)", e.Type, e.Message, e.Status)
}

// IRCMessage represents a message from the IRCCloud WebSocket.
type IRCMessage struct {
	Type     string         `json:"type"`
	Chan     string         `json:"chan"`
	From     string         `json:"from"`
	Msg      string         `json:"msg"`
	Time     int64          `json:"time"`
	EID      int64          `json:"eid"`
	BID      int            `json:"bid"`
	Server   string         `json:"server"`
	Nick     string         `json:"nick"`
	Hostmask string         `json:"hostmask"`
	Ops      map[string]any `json:"ops"`
	Self     bool           `json:"self"`
}

// OOBInclude is a message that contains a URL to the backlog.
type OOBInclude struct {
	URL string `json:"url"`
}

// Connect connects to the IRCCloud WebSocket API with retry logic.
func (c *IRCCloudClient) Connect(email, password string) error {
	// Store credentials for reconnection
	c.email = email
	c.password = password

	return c.connectWithRetry()
}

// connectWithRetry implements exponential backoff retry logic
func (c *IRCCloudClient) connectWithRetry() error {
	c.retryCount = 0

	for c.retryCount < c.connConfig.MaxRetryAttempts {
		if c.ctx.Err() != nil {
			return fmt.Errorf("connection cancelled")
		}

		if c.retryCount > 0 {
			c.setState(StateReconnecting)
			delay := c.calculateBackoffDelay()
			log.Printf("🔄 Retry attempt %d/%d in %v", c.retryCount+1, c.connConfig.MaxRetryAttempts, delay)

			select {
			case <-time.After(delay):
			case <-c.ctx.Done():
				return fmt.Errorf("connection cancelled during retry")
			}
		} else {
			c.setState(StateConnecting)
		}

		err := c.attemptConnection()
		if err == nil {
			c.setState(StateConnected)
			c.retryCount = 0
			c.lastConnectTime = time.Now()
			log.Println("✅ WebSocket connection established!")
			return nil
		}

		log.Printf("❌ Connection attempt failed: %v", err)
		c.retryCount++

		if c.retryCount >= c.connConfig.MaxRetryAttempts {
			c.setState(StateError)
			return fmt.Errorf("failed to connect after %d attempts: %w", c.connConfig.MaxRetryAttempts, err)
		}
	}

	return fmt.Errorf("connection failed")
}

// attemptConnection tries to establish a single connection
func (c *IRCCloudClient) attemptConnection() error {
	// Step 1: Authenticate if we don't have a cached auth response or it's stale
	if c.authResp == nil || time.Since(c.lastConnectTime) > 30*time.Minute {
		log.Println("🔐 Authenticating...")
		authResp, err := c.authenticate(c.email, c.password)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		c.authResp = authResp
		c.session = authResp.Session
		c.apiHost = authResp.APIHost
	}

	// Step 2: Connect to the WebSocket API
	log.Println("🌐 Connecting to WebSocket...")
	wsURL := c.buildWebSocketURL(c.authResp)
	log.Printf("🌐 WebSocket URL: %s", wsURL)

	header := http.Header{}
	header.Add("Origin", "https://www.irccloud.com")
	header.Add("User-Agent", "irccloud-watcher/0.1.0")
	header.Add("Cookie", "session="+c.authResp.Session)

	// Parse connection timeout
	timeout, err := time.ParseDuration(c.connConfig.ConnectionTimeout)
	if err != nil {
		timeout = 45 * time.Second
	}

	dialer := &websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  timeout,
		EnableCompression: true,
	}

	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			log.Printf("❌ WebSocket handshake failed with status: %s", resp.Status)
			if location := resp.Header.Get("Location"); location != "" {
				log.Printf("❌ Redirect location: %s", location)
			}
			errorBody, readErr := io.ReadAll(resp.Body)
			if readErr == nil && len(errorBody) < 500 {
				log.Printf("❌ WebSocket response body: %s", string(errorBody))
			}
		}
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	// Close any existing connection
	if c.conn != nil {
		c.conn.Close()
	}

	c.conn = conn
	return nil
}

// calculateBackoffDelay calculates the delay for exponential backoff
func (c *IRCCloudClient) calculateBackoffDelay() time.Duration {
	initialDelay, err := time.ParseDuration(c.connConfig.InitialRetryDelay)
	if err != nil {
		initialDelay = time.Second
	}

	maxDelay, err := time.ParseDuration(c.connConfig.MaxRetryDelay)
	if err != nil {
		maxDelay = 5 * time.Minute
	}

	// Calculate exponential backoff: initial * (multiplier ^ retryCount)
	delay := time.Duration(float64(initialDelay) * math.Pow(c.connConfig.BackoffMultiplier, float64(c.retryCount)))

	// Cap at maximum delay
	delay = min(delay, maxDelay)

	return delay
}

// Close closes the WebSocket connection and cancels reconnection attempts.
func (c *IRCCloudClient) Close() {
	c.setState(StateDisconnected)
	c.cancelFunc() // Cancel any ongoing operations

	if c.conn != nil {
		if err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			log.Printf("⚠️ Error writing close message: %v", err)
		}
		c.conn.Close()
		c.conn = nil
	}
}

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
			log.Printf("🔍 Processing message: Channel=%s, From=%s, Time=%d, Converted=%s", ircMsg.Chan, ircMsg.From, ircMsg.Time, msgTime.Format(time.RFC3339))
		}

		log.Printf("%s <%s> %s", ircMsg.Chan, ircMsg.From, cleanedMsg)

		dbMsg := &storage.Message{
			Channel:      ircMsg.Chan,
			Timestamp:    msgTime,
			Sender:       ircMsg.From,
			Message:      cleanedMsg,
			Date:         msgTime.Format("2006-01-02"),
			IRCCloudTime: ircMsg.Time,
		}

		if err := c.db.InsertMessage(dbMsg); err != nil {
			log.Printf("❌ Error inserting message into DB: %v", err)
			return fmt.Errorf("error inserting message into DB: %w", err)
		}

		if os.Getenv("IRCCLOUD_DEBUG") == "true" {
			log.Printf("✅ Message stored successfully: ID will be auto-generated")
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

// authenticate authenticates with the IRCCloud API and returns the full authentication response.
func (c *IRCCloudClient) authenticate(email, password string) (*AuthResponse, error) {
	log.Printf("🔐 Starting authentication for email: %s", email)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second, Jar: jar}

	// Step 1: Get an auth-formtoken
	log.Println("📡 Step 1: Requesting auth-formtoken...")
	tokenURL := "https://www.irccloud.com/chat/auth-formtoken"
	req, err := http.NewRequest("POST", tokenURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("could not create token request: %w", err)
	}
	req.Header.Set("User-Agent", "irccloud-watcher/0.1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Length", "0")

	debugLogRequest("POST", tokenURL, req.Header)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not perform token request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("📡 Token request response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		errorBody, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			log.Printf("❌ Token request error response body: %s", string(errorBody))
		}
		return nil, fmt.Errorf("token request failed with status: %s", resp.Status)
	}

	type TokenResponse struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read token response body: %w", err)
	}

	debugLogResponse(resp, body)

	var tokenResp TokenResponse
	if parseErr := json.Unmarshal(body, &tokenResp); parseErr != nil {
		log.Printf("❌ Failed to parse token response: %s", string(body))
		return nil, fmt.Errorf("could not parse token response: %w", parseErr)
	}

	log.Printf("✅ Token received successfully: %t, Token length: %d", tokenResp.Success, len(tokenResp.Token))
	if !tokenResp.Success {
		return nil, fmt.Errorf("token request unsuccessful")
	}

	// Step 2: Log in with email, password, and token
	log.Println("🔑 Step 2: Logging in with credentials...")
	loginURL := "https://www.irccloud.com/chat/login"
	data := url.Values{}
	data.Set("email", email)
	data.Set("password", password)
	data.Set("token", tokenResp.Token)

	req, err = http.NewRequest("POST", loginURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("could not create login request: %w", err)
	}

	req.Header.Set("X-Auth-Formtoken", tokenResp.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "irccloud-watcher/0.1.0")
	req.Header.Set("Accept", "application/json")

	debugLogRequest("POST", loginURL, req.Header)
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not perform login request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("🔑 Login response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		errorBody, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			log.Printf("❌ Login request error response body: %s", string(errorBody))
		}
		return nil, fmt.Errorf("login failed with status: %s", resp.Status)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read login response body: %w", err)
	}

	debugLogResponse(resp, body)

	authResp, err := parseAPIResponse(body, resp.StatusCode)
	if err != nil {
		log.Printf("❌ Authentication failed: %v", err)
		return nil, err
	}

	log.Printf("✅ Login success: %t, Session length: %d", authResp.Success, len(authResp.Session))
	log.Printf("🎉 Authentication completed successfully!")
	log.Printf("🌐 WebSocket details - Host: %s, Path: %s", authResp.WebSocketHost, authResp.WebSocketPath)
	return authResp, nil
}

// buildWebSocketURL constructs the WebSocket URL from authentication response
func (c *IRCCloudClient) buildWebSocketURL(authResp *AuthResponse) string {
	if authResp.WebSocketHost != "" && authResp.WebSocketPath != "" {
		baseURL := fmt.Sprintf("wss://%s%s", authResp.WebSocketHost, authResp.WebSocketPath)
		// Add query parameters
		u, err := url.Parse(baseURL)
		if err != nil {
			log.Printf("⚠️ Error parsing WebSocket URL, using fallback: %v", err)
			return "wss://www.irccloud.com/?since_id=0&stream_id=0"
		}
		q := u.Query()
		q.Set("since_id", "0")
		q.Set("stream_id", "0")
		u.RawQuery = q.Encode()
		return u.String()
	}

	// Fallback to original URL
	log.Println("⚠️ Using fallback WebSocket URL")
	return "wss://www.irccloud.com/?since_id=0&stream_id=0"
}

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
			log.Printf("🔍 Processing backlog message: Channel=%s, From=%s, Time=%d, Converted=%s", ircMsg.Chan, ircMsg.From, ircMsg.Time, msgTime.Format(time.RFC3339))
		}

		log.Printf("%s <%s> %s", ircMsg.Chan, ircMsg.From, cleanedMsg)

		dbMsg := &storage.Message{
			Channel:      ircMsg.Chan,
			Timestamp:    msgTime,
			Sender:       ircMsg.From,
			Message:      cleanedMsg,
			Date:         msgTime.Format("2006-01-02"),
			IRCCloudTime: ircMsg.Time,
		}

		if err := c.db.InsertMessage(dbMsg); err != nil {
			log.Printf("❌ Error inserting backlog message into DB: %v", err)
		} else if os.Getenv("IRCCLOUD_DEBUG") == "true" {
			log.Printf("✅ Backlog message stored successfully")
		}
	}

	log.Println("Finished processing backlog")
	return nil
}

// debugLogRequest logs HTTP request details when debug mode is enabled
func debugLogRequest(method, requestURL string, headers http.Header) {
	if os.Getenv("IRCCLOUD_DEBUG") == "true" {
		log.Printf("🔍 %s %s", method, requestURL)
		for key, values := range headers {
			if !isSensitiveHeader(key) {
				log.Printf("🔍   %s: %s", key, strings.Join(values, ", "))
			}
		}
	}
}

// debugLogResponse logs HTTP response details when debug mode is enabled
func debugLogResponse(resp *http.Response, body []byte) {
	if os.Getenv("IRCCLOUD_DEBUG") == "true" {
		log.Printf("🔍 Response: %s", resp.Status)
		if len(body) > 200 {
			log.Printf("🔍 Body: %s...", string(body[:200]))
		} else {
			log.Printf("🔍 Body: %s", string(body))
		}
	}
}

// isSensitiveHeader checks if a header contains sensitive information
func isSensitiveHeader(key string) bool {
	sensitive := []string{"authorization", "cookie", "x-auth-formtoken"}
	for _, s := range sensitive {
		if strings.EqualFold(key, s) {
			return true
		}
	}
	return false
}

// parseAPIResponse parses API responses and handles errors properly
func parseAPIResponse(body []byte, statusCode int) (*AuthResponse, error) {
	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("could not parse response: %w", err)
	}

	if !authResp.Success {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			// If we can't parse the error response, use a generic message
			return nil, &AuthError{
				Type:    "api_error",
				Message: "Authentication failed",
				Status:  statusCode,
			}
		}
		return nil, &AuthError{
			Type:    "api_error",
			Message: errResp.Message,
			Status:  statusCode,
		}
	}

	return &authResp, nil
}
