package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"irccloud-watcher/internal/config"
	"irccloud-watcher/internal/storage"
)

// Test data
var sampleTokenResponse = `{"success":true,"token":"1752956700.e40fd00eac68b2e0b0979ba8bae1469f"}`
var sampleLoginResponse = `{
    "success":true,
    "session":"2.73dcfacebec8df39d9affb7d4a58e556",
    "uid":305680,
    "api_host":"https://api-2.irccloud.com",
    "websocket_host":"api-2.irccloud.com",
    "websocket_path":"/websocket/2",
    "url":""
}`

var sampleErrorResponse = `{"success":false,"message":"Invalid credentials"}`

func TestFormtokenRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Length") != "0" {
			t.Errorf("Expected Content-Length: 0, got %s", r.Header.Get("Content-Length"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json, got %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "irccloud-watcher/0.1.0" {
			t.Errorf("Expected correct User-Agent, got %s", r.Header.Get("User-Agent"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleTokenResponse))
	}))
	defer server.Close()

	// This test would require refactoring authenticate to be testable
	// For now, we test the response parsing logic
	var tokenResp struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
	}

	err := json.Unmarshal([]byte(sampleTokenResponse), &tokenResp)
	if err != nil {
		t.Fatalf("Failed to parse token response: %v", err)
	}

	if !tokenResp.Success {
		t.Error("Expected success to be true")
	}

	if tokenResp.Token == "" {
		t.Error("Expected token to be non-empty")
	}
}

func TestLoginResponseParsing(t *testing.T) {
	authResp, err := parseAPIResponse([]byte(sampleLoginResponse), 200)
	if err != nil {
		t.Fatalf("Failed to parse login response: %v", err)
	}

	if !authResp.Success {
		t.Error("Expected success to be true")
	}

	if authResp.Session == "" {
		t.Error("Expected session to be non-empty")
	}

	if authResp.UID != 305680 {
		t.Errorf("Expected UID 305680, got %d", authResp.UID)
	}

	if authResp.WebSocketHost != "api-2.irccloud.com" {
		t.Errorf("Expected WebSocket host api-2.irccloud.com, got %s", authResp.WebSocketHost)
	}

	if authResp.WebSocketPath != "/websocket/2" {
		t.Errorf("Expected WebSocket path /websocket/2, got %s", authResp.WebSocketPath)
	}
}

func TestErrorResponseParsing(t *testing.T) {
	_, err := parseAPIResponse([]byte(sampleErrorResponse), 401)
	if err == nil {
		t.Fatal("Expected error when parsing error response")
	}

	authErr, ok := err.(*AuthError)
	if !ok {
		t.Fatalf("Expected AuthError, got %T", err)
	}

	if authErr.Type != "api_error" {
		t.Errorf("Expected error type api_error, got %s", authErr.Type)
	}

	if authErr.Message != "Invalid credentials" {
		t.Errorf("Expected error message 'Invalid credentials', got %s", authErr.Message)
	}

	if authErr.Status != 401 {
		t.Errorf("Expected status 401, got %d", authErr.Status)
	}
}

func TestWebSocketURLBuilding(t *testing.T) {
	tests := []struct {
		name     string
		response AuthResponse
		expected string
	}{
		{
			name: "normal response",
			response: AuthResponse{
				WebSocketHost: "api-2.irccloud.com",
				WebSocketPath: "/websocket/2",
			},
			expected: "wss://api-2.irccloud.com/websocket/2?since_id=0&stream_id=0",
		},
		{
			name: "different api server",
			response: AuthResponse{
				WebSocketHost: "api-5.irccloud.com",
				WebSocketPath: "/websocket/5",
			},
			expected: "wss://api-5.irccloud.com/websocket/5?since_id=0&stream_id=0",
		},
		{
			name:     "empty response uses fallback",
			response: AuthResponse{},
			expected: "wss://www.irccloud.com/?since_id=0&stream_id=0",
		},
		{
			name: "missing host uses fallback",
			response: AuthResponse{
				WebSocketPath: "/websocket/2",
			},
			expected: "wss://www.irccloud.com/?since_id=0&stream_id=0",
		},
		{
			name: "missing path uses fallback",
			response: AuthResponse{
				WebSocketHost: "api-2.irccloud.com",
			},
			expected: "wss://www.irccloud.com/?since_id=0&stream_id=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewIRCCloudClient(nil)
			result := client.buildWebSocketURL(&tt.response)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsSensitiveHeader(t *testing.T) {
	tests := []struct {
		header    string
		sensitive bool
	}{
		{"authorization", true},
		{"Authorization", true},
		{"AUTHORIZATION", true},
		{"cookie", true},
		{"Cookie", true},
		{"x-auth-formtoken", true},
		{"X-Auth-Formtoken", true},
		{"user-agent", false},
		{"content-type", false},
		{"accept", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := isSensitiveHeader(tt.header)
			if result != tt.sensitive {
				t.Errorf("Expected %s to be sensitive=%t, got %t", tt.header, tt.sensitive, result)
			}
		})
	}
}

func TestIRCMessageStructure(t *testing.T) {
	sampleMessage := `{
		"type": "buffer_msg",
		"chan": "#test",
		"from": "testuser",
		"msg": "Hello world",
		"time": 1234567890000000,
		"eid": 123456789,
		"bid": 12345,
		"server": "irc.example.com",
		"nick": "testuser",
		"hostmask": "testuser@example.com",
		"self": false
	}`

	var ircMsg IRCMessage
	err := json.Unmarshal([]byte(sampleMessage), &ircMsg)
	if err != nil {
		t.Fatalf("Failed to parse IRC message: %v", err)
	}

	if ircMsg.Type != "buffer_msg" {
		t.Errorf("Expected type buffer_msg, got %s", ircMsg.Type)
	}

	if ircMsg.Chan != "#test" {
		t.Errorf("Expected channel #test, got %s", ircMsg.Chan)
	}

	if ircMsg.From != "testuser" {
		t.Errorf("Expected from testuser, got %s", ircMsg.From)
	}

	if ircMsg.Msg != "Hello world" {
		t.Errorf("Expected message 'Hello world', got %s", ircMsg.Msg)
	}

	if ircMsg.Time != 1234567890000000 {
		t.Errorf("Expected time 1234567890000000, got %d", ircMsg.Time)
	}

	if ircMsg.EID != 123456789 {
		t.Errorf("Expected EID 123456789, got %d", ircMsg.EID)
	}

	if ircMsg.Self != false {
		t.Errorf("Expected self false, got %t", ircMsg.Self)
	}
}

func TestDebugLogging(t *testing.T) {
	// Test that debug logging doesn't crash when environment variable is not set
	headers := http.Header{}
	headers.Set("User-Agent", "test")
	headers.Set("Authorization", "secret")

	// This should not panic
	debugLogRequest("GET", "https://example.com", headers)

	// Set debug mode and test again
	os.Setenv("IRCCLOUD_DEBUG", "true")
	defer os.Unsetenv("IRCCLOUD_DEBUG")

	// This should not panic and should log output
	debugLogRequest("GET", "https://example.com", headers)

	// Test response logging
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
	}
	body := []byte(`{"test": "response"}`)
	debugLogResponse(resp, body)
}

func TestMalformedJSONHandling(t *testing.T) {
	malformedJSON := `{"success":true,"missing_closing_brace"`

	_, err := parseAPIResponse([]byte(malformedJSON), 200)
	if err == nil {
		t.Error("Expected error when parsing malformed JSON")
	}

	if !strings.Contains(err.Error(), "could not parse response") {
		t.Errorf("Expected parse error message, got: %v", err)
	}
}

// Test connection state management
func TestConnectionState(t *testing.T) {
	client := NewIRCCloudClient(nil)

	// Test initial state
	if client.getState() != StateDisconnected {
		t.Errorf("Expected initial state to be StateDisconnected, got %s", client.getState())
	}

	// Test state transitions
	client.setState(StateConnecting)
	if client.getState() != StateConnecting {
		t.Errorf("Expected state to be StateConnecting, got %s", client.getState())
	}

	client.setState(StateConnected)
	if client.getState() != StateConnected {
		t.Errorf("Expected state to be StateConnected, got %s", client.getState())
	}

	client.setState(StateReconnecting)
	if client.getState() != StateReconnecting {
		t.Errorf("Expected state to be StateReconnecting, got %s", client.getState())
	}

	client.setState(StateError)
	if client.getState() != StateError {
		t.Errorf("Expected state to be StateError, got %s", client.getState())
	}
}

// Test connection state string representation
func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateDisconnected, "disconnected"},
		{StateConnecting, "connecting"},
		{StateConnected, "connected"},
		{StateReconnecting, "reconnecting"},
		{StateError, "error"},
		{ConnectionState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.state.String())
			}
		})
	}
}

// Test exponential backoff calculation
func TestCalculateBackoffDelay(t *testing.T) {
	client := NewIRCCloudClient(nil)
	client.SetConnectionConfig(&config.ConnectionConfig{
		InitialRetryDelay: "1s",
		MaxRetryDelay:     "30s",
		BackoffMultiplier: 2.0,
	})

	tests := []struct {
		retryCount    int
		expectedDelay time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second},  // Capped at max delay
		{10, 30 * time.Second}, // Still capped at max delay
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.retryCount+'0')), func(t *testing.T) {
			client.retryCount = tt.retryCount
			delay := client.calculateBackoffDelay()
			if delay != tt.expectedDelay {
				t.Errorf("For retry count %d, expected delay %v, got %v", tt.retryCount, tt.expectedDelay, delay)
			}
		})
	}
}

// Test backoff with invalid config values
func TestCalculateBackoffDelayWithInvalidConfig(t *testing.T) {
	client := NewIRCCloudClient(nil)
	client.SetConnectionConfig(&config.ConnectionConfig{
		InitialRetryDelay: "invalid",
		MaxRetryDelay:     "invalid",
		BackoffMultiplier: 2.0,
	})
	client.retryCount = 1

	// Should use fallback values when config is invalid
	delay := client.calculateBackoffDelay()
	expectedDelay := 2 * time.Second // 1s * 2^1 (using fallback initial delay)
	if delay != expectedDelay {
		t.Errorf("Expected fallback delay %v, got %v", expectedDelay, delay)
	}
}

// Test connection configuration defaults
func TestConnectionConfigDefaults(t *testing.T) {
	connConfig := &config.ConnectionConfig{
		HeartbeatInterval: "30s",
		MaxRetryAttempts:  10,
		InitialRetryDelay: "1s",
		MaxRetryDelay:     "5m",
		BackoffMultiplier: 2.0,
		ConnectionTimeout: "45s",
		PingInterval:      "60s",
	}

	// For now, test that the config struct exists and can be used
	client := NewIRCCloudClient(nil)
	client.SetConnectionConfig(connConfig)

	// Test that the client doesn't panic with empty config
	if client.connConfig == nil {
		t.Error("Connection config should not be nil after setting")
	}

	// Test that config values are properly set
	if client.connConfig.MaxRetryAttempts != 10 {
		t.Errorf("Expected MaxRetryAttempts to be 10, got %d", client.connConfig.MaxRetryAttempts)
	}

	if client.connConfig.BackoffMultiplier != 2.0 {
		t.Errorf("Expected BackoffMultiplier to be 2.0, got %f", client.connConfig.BackoffMultiplier)
	}
}

// Test message processing
func TestProcessMessage(t *testing.T) {
	// Create a temporary database for testing
	db, err := storage.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	client := NewIRCCloudClient(db)
	client.channelSet = map[string]bool{"#test": true}
	client.ignoredChannelSet = map[string]bool{"#ignored": true}

	// Test valid buffer message
	validMessage := `{
		"type": "buffer_msg",
		"chan": "#test",
		"from": "testuser",
		"msg": "Hello world",
		"time": 1634567890000000
	}`

	err = client.processMessage([]byte(validMessage))
	if err != nil {
		t.Errorf("Failed to process valid message: %v", err)
	}

	// Test ignored channel message
	ignoredMessage := `{
		"type": "buffer_msg",
		"chan": "#ignored",
		"from": "testuser",
		"msg": "This should be ignored",
		"time": 1634567890000000
	}`

	err = client.processMessage([]byte(ignoredMessage))
	if err != nil {
		t.Errorf("Failed to process ignored message: %v", err)
	}

	// Test invalid JSON
	invalidMessage := `{"invalid": json}`
	err = client.processMessage([]byte(invalidMessage))
	if err == nil {
		t.Error("Expected error when processing invalid JSON")
	}
}

func TestDebugMode(t *testing.T) {
	// Create in-memory database
	db, err := storage.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	// Create client
	client := NewIRCCloudClient(db)

	// Test with debug mode disabled (default)
	client.SetDebugMode(false)
	if client.debugMode {
		t.Error("Debug mode should be disabled by default")
	}

	// Test enabling debug mode
	client.SetDebugMode(true)
	if !client.debugMode {
		t.Error("Debug mode should be enabled after SetDebugMode(true)")
	}

	// Test disabling debug mode
	client.SetDebugMode(false)
	if client.debugMode {
		t.Error("Debug mode should be disabled after SetDebugMode(false)")
	}
}
