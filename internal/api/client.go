package api

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"irccloud-watcher/internal/config"
	"irccloud-watcher/internal/storage"

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

	// EID deduplication cache
	eidCache      map[int64]bool
	eidCacheMutex sync.RWMutex
	maxCacheSize  int
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

// TokenResponse is the response from the auth-formtoken endpoint.
type TokenResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
}

// NewIRCCloudClient creates a new IRCCloudClient.
func NewIRCCloudClient(db *storage.DB) *IRCCloudClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &IRCCloudClient{
		db:           db,
		state:        StateDisconnected,
		ctx:          ctx,
		cancelFunc:   cancel,
		eidCache:     make(map[int64]bool),
		maxCacheSize: 10000, // Keep track of last 10k EIDs
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

// isEIDSeen checks if an EID has been seen before and marks it as seen
func (c *IRCCloudClient) isEIDSeen(eid int64) bool {
	c.eidCacheMutex.Lock()
	defer c.eidCacheMutex.Unlock()

	if c.eidCache[eid] {
		return true
	}

	// Add to cache
	c.eidCache[eid] = true

	// If cache is getting too large, clean it up (simple FIFO-ish cleanup)
	if len(c.eidCache) > c.maxCacheSize {
		// Remove roughly 20% of entries to avoid frequent cleanups
		toRemove := c.maxCacheSize / 5
		count := 0
		for k := range c.eidCache {
			if count >= toRemove {
				break
			}
			delete(c.eidCache, k)
			count++
		}
		if os.Getenv("IRCCLOUD_DEBUG") == "true" {
			log.Printf("🧹 EID cache cleanup: removed %d entries, %d remaining", toRemove, len(c.eidCache))
		}
	}

	return false
}
