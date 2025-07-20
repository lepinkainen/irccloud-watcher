package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"irccloud-watcher/internal/storage"
	"irccloud-watcher/internal/utils"

	"github.com/gorilla/websocket"
)

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
}

// NewIRCCloudClient creates a new IRCCloudClient.
func NewIRCCloudClient(db *storage.DB) *IRCCloudClient {
	return &IRCCloudClient{
		db: db,
	}
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
	Type     string                 `json:"type"`
	Chan     string                 `json:"chan"`
	From     string                 `json:"from"`
	Msg      string                 `json:"msg"`
	Time     int64                  `json:"time"`
	EID      int64                  `json:"eid"`
	BID      int                    `json:"bid"`
	Server   string                 `json:"server"`
	Nick     string                 `json:"nick"`
	Hostmask string                 `json:"hostmask"`
	Ops      map[string]interface{} `json:"ops"`
	Self     bool                   `json:"self"`
}

// OOBInclude is a message that contains a URL to the backlog.
type OOBInclude struct {
	URL string `json:"url"`
}

// Connect connects to the IRCCloud WebSocket API.
func (c *IRCCloudClient) Connect(email, password string) error {
	// Step 1: Authenticate and get authentication response.
	authResp, err := c.authenticate(email, password)
	if err != nil {
		return fmt.Errorf("could not authenticate: %w", err)
	}

	// Store the session and API host for later use
	c.session = authResp.Session
	c.apiHost = authResp.APIHost

	// Step 2: Connect to the WebSocket API.
	log.Println("🌐 Step 3: Connecting to WebSocket...")
	// Build dynamic WebSocket URL from authentication response
	wsURL := c.buildWebSocketURL(authResp)
	log.Printf("🌐 WebSocket URL: %s", wsURL)

	header := http.Header{}
	header.Add("Origin", "https://www.irccloud.com")
	header.Add("User-Agent", "irccloud-watcher/0.1.0")
	header.Add("Cookie", "session="+authResp.Session) // Manually add the session cookie

	log.Printf("🌐 WebSocket headers: %v", header)
	log.Printf("🌐 Session key length: %d", len(authResp.Session))

	dialer := &websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  45 * time.Second,
		EnableCompression: true, // Enable compression as suggested in docs
	}

	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			log.Printf("❌ WebSocket handshake failed with status: %s", resp.Status)
			if location := resp.Header.Get("Location"); location != "" {
				log.Printf("❌ Redirect location: %s", location)
			}
			log.Printf("❌ Response headers: %v", resp.Header)
			errorBody, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				log.Printf("❌ WebSocket response body: %s", string(errorBody))
			}
		}
		return fmt.Errorf("could not connect to websocket: %w", err)
	}
	c.conn = conn

	log.Println("✅ WebSocket connection established!")
	return nil
}

// Close closes the WebSocket connection.
func (c *IRCCloudClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Run starts the client and listens for messages.
func (c *IRCCloudClient) Run(channels, ignoredChannels []string) {
	// Store filtering parameters in the client
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

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				// TODO: Implement reconnection logic
				return
			}

			var ircMsg IRCMessage
			if err := json.Unmarshal(message, &ircMsg); err != nil {
				log.Println("unmarshal error:", err)
				continue
			}

			if ircMsg.Type == "oob_include" {
				var oob OOBInclude
				if err := json.Unmarshal(message, &oob); err != nil {
					log.Println("unmarshal oob error:", err)
					continue
				}
				log.Printf("🔍 Received oob_include with URL: %s", oob.URL)
				if err := c.processBacklog(oob.URL); err != nil {
					log.Println("error processing backlog:", err)
				}
				continue
			}

			// Accept message if not ignored and either no channels specified (accept all) or channel is in allowed list
			if ircMsg.Type == "buffer_msg" && !c.ignoredChannelSet[ircMsg.Chan] && (len(c.channels) == 0 || c.channelSet[ircMsg.Chan]) {
				cleanedMsg := utils.CleanIRCMessage(ircMsg.Msg)
				log.Printf("Received message in %s from %s: %s", ircMsg.Chan, ircMsg.From, cleanedMsg)

				msgTime := time.Unix(0, ircMsg.Time*1000)

				dbMsg := &storage.Message{
					Channel:      ircMsg.Chan,
					Timestamp:    msgTime,
					Sender:       ircMsg.From,
					Message:      cleanedMsg,
					Date:         msgTime.Format("2006-01-02"),
					IRCCloudTime: ircMsg.Time,
				}

				if err := c.db.InsertMessage(dbMsg); err != nil {
					log.Printf("Error inserting message into DB: %v", err)
				}
				c.lastSeenEID = ircMsg.Time
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			// Send a heartbeat to keep the connection alive and report the last seen EID
			heartbeat := map[string]interface{}{"_method": "heartbeat", "_reqid": time.Now().Unix(), "last_seen_eid": c.lastSeenEID}
			if err := c.conn.WriteJSON(heartbeat); err != nil {
				log.Println("heartbeat error:", err)
			}
		case <-interrupt:
			log.Println("interrupt received, closing connection")
			err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
			}
			time.Sleep(time.Second)
			return
		}
	}
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
		log.Printf("Received backlog message in %s from %s: %s", ircMsg.Chan, ircMsg.From, cleanedMsg)

		msgTime := time.Unix(0, ircMsg.Time*1000)

		dbMsg := &storage.Message{
			Channel:      ircMsg.Chan,
			Timestamp:    msgTime,
			Sender:       ircMsg.From,
			Message:      cleanedMsg,
			Date:         msgTime.Format("2006-01-02"),
			IRCCloudTime: ircMsg.Time,
		}

		if err := c.db.InsertMessage(dbMsg); err != nil {
			log.Printf("Error inserting backlog message into DB: %v", err)
		}
	}

	log.Println("Finished processing backlog")
	return nil
}

// debugLogRequest logs HTTP request details when debug mode is enabled
func debugLogRequest(method, url string, headers http.Header) {
	if os.Getenv("IRCCLOUD_DEBUG") == "true" {
		log.Printf("🔍 %s %s", method, url)
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
