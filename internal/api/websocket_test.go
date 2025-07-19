package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
