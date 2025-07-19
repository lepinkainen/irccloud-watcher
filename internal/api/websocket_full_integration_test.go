//go:build integration

package api

import (
	"os"
	"testing"
	"time"

	"irccloud-watcher/internal/storage"
)

func TestCompleteFlow(t *testing.T) {
	email := os.Getenv("IRCCLOUD_EMAIL")
	password := os.Getenv("IRCCLOUD_PASSWORD")

	if email == "" || password == "" {
		t.Skip("Integration test requires IRCCLOUD_EMAIL and IRCCLOUD_PASSWORD")
	}

	// Test database setup
	db, err := storage.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Test client creation and connection
	client := NewIRCCloudClient(db)
	err = client.Connect(email, password)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	t.Log("Successfully connected to IRCCloud WebSocket API")

	// Test brief message reception
	done := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		done <- true
	}()

	// Start the client in a goroutine for a brief test
	go client.Run([]string{"#test"})

	<-done

	// The test completes successfully if we get here without errors
	t.Log("Integration test completed successfully")
}
