package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

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
