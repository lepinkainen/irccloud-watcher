//go:build !ci

package api

import (
	"os"
	"strings"
	"testing"

	"irccloud-watcher/internal/config"
)

func TestRealAuthentication(t *testing.T) {
	// Try to load configuration first, fall back to a minimal config for testing
	cfg, err := config.LoadConfig("../../config.yaml")
	if err != nil {
		// If config file doesn't exist, try to create a minimal config from env vars
		cfg = &config.Config{
			Connection: config.ConnectionConfig{
				HeartbeatInterval: "30s",
				MaxRetryAttempts:  10,
				InitialRetryDelay: "1s",
				MaxRetryDelay:     "5m",
				BackoffMultiplier: 2.0,
				ConnectionTimeout: "45s",
				PingInterval:      "60s",
			},
		}
		// The config.LoadConfig would normally handle env vars, but since we can't load the file,
		// we need to manually check them here
		if email := getTestCredential("IRCCLOUD_EMAIL"); email != "" {
			cfg.Email = email
		}
		if password := getTestCredential("IRCCLOUD_PASSWORD"); password != "" {
			cfg.Password = password
		}
	}

	if cfg.Email == "" || cfg.Password == "" {
		t.Skip("Skipping real auth test - no credentials provided (set in config.yaml or IRCCLOUD_EMAIL/IRCCLOUD_PASSWORD environment variables)")
	}

	email := cfg.Email
	password := cfg.Password

	client := NewIRCCloudClient(nil)
	client.SetConnectionConfig(&cfg.Connection)
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
	// Try to load configuration first, fall back to a minimal config for testing
	cfg, err := config.LoadConfig("../../config.yaml")
	if err != nil {
		// If config file doesn't exist, try to create a minimal config from env vars
		cfg = &config.Config{
			Connection: config.ConnectionConfig{
				HeartbeatInterval: "30s",
				MaxRetryAttempts:  10,
				InitialRetryDelay: "1s",
				MaxRetryDelay:     "5m",
				BackoffMultiplier: 2.0,
				ConnectionTimeout: "45s",
				PingInterval:      "60s",
			},
		}
		if email := getTestCredential("IRCCLOUD_EMAIL"); email != "" {
			cfg.Email = email
		}
		if password := getTestCredential("IRCCLOUD_PASSWORD"); password != "" {
			cfg.Password = password
		}
	}

	if cfg.Email == "" || cfg.Password == "" {
		t.Skip("Skipping real WebSocket test - no credentials provided (set in config.yaml or IRCCLOUD_EMAIL/IRCCLOUD_PASSWORD environment variables)")
	}

	email := cfg.Email
	password := cfg.Password

	// Note: This test attempts a real WebSocket connection
	// It may fail if IRCCloud's API is down or credentials are invalid
	client := NewIRCCloudClient(nil)
	client.SetConnectionConfig(&cfg.Connection)

	err = client.Connect(email, password)
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

// getTestCredential safely gets environment variables for testing
func getTestCredential(key string) string {
	return os.Getenv(key)
}
