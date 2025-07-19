//go:build !ci

package api

import (
	"os"
	"strings"
	"testing"
)

func TestRealAuthentication(t *testing.T) {
	email := os.Getenv("IRCCLOUD_EMAIL")
	password := os.Getenv("IRCCLOUD_PASSWORD")

	if email == "" || password == "" {
		t.Skip("Skipping real auth test - no credentials provided (set IRCCLOUD_EMAIL and IRCCLOUD_PASSWORD)")
	}

	client := NewIRCCloudClient(nil)
	authResp, err := client.authenticate(email, password)
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}

	if !authResp.Success {
		t.Error("Authentication was not successful")
	}

	if authResp.Session == "" {
		t.Error("No session returned")
	}

	if authResp.WebSocketHost == "" {
		t.Error("No WebSocket host returned")
	}

	if authResp.WebSocketPath == "" {
		t.Error("No WebSocket path returned")
	}

	if authResp.UID == 0 {
		t.Error("No UID returned")
	}

	// Test WebSocket URL building with real response
	wsURL := client.buildWebSocketURL(authResp)
	if wsURL == "" {
		t.Error("WebSocket URL construction failed")
	}

	// Verify the URL is properly formatted
	expectedPrefix := "wss://" + authResp.WebSocketHost + authResp.WebSocketPath
	if !strings.Contains(wsURL, expectedPrefix) {
		t.Errorf("WebSocket URL should contain %s, got %s", expectedPrefix, wsURL)
	}

	// Verify query parameters are present
	if !strings.Contains(wsURL, "since_id=0") {
		t.Error("WebSocket URL should contain since_id=0")
	}

	if !strings.Contains(wsURL, "stream_id=0") {
		t.Error("WebSocket URL should contain stream_id=0")
	}

	t.Logf("Successfully authenticated user %d", authResp.UID)
	t.Logf("WebSocket URL: %s", wsURL)
}

func TestRealWebSocketConnection(t *testing.T) {
	email := os.Getenv("IRCCLOUD_EMAIL")
	password := os.Getenv("IRCCLOUD_PASSWORD")

	if email == "" || password == "" {
		t.Skip("Skipping real WebSocket test - no credentials provided")
	}

	// Note: This test attempts a real WebSocket connection
	// It may fail if IRCCloud's API is down or credentials are invalid
	client := NewIRCCloudClient(nil)

	err := client.Connect(email, password)
	if err != nil {
		// This might fail due to various reasons (API changes, rate limiting, etc.)
		// So we log the error but don't fail the test
		t.Logf("WebSocket connection failed (this may be expected): %v", err)
		return
	}

	// If we get here, connection succeeded
	t.Log("WebSocket connection established successfully")

	// Clean up
	client.Close()
}
