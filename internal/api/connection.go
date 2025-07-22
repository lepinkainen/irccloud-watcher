package api

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

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
