package api

import (
	"log"
	"net/http"
	"os"
	"strings"
)

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
