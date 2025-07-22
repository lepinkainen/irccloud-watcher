package llm

import (
	"testing"
	"time"
)

func TestDefaultProviderConfig(t *testing.T) {
	config := DefaultProviderConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout to be 30s, got %v", config.Timeout)
	}

	if config.RetryAttempts != 3 {
		t.Errorf("expected retry attempts to be 3, got %d", config.RetryAttempts)
	}

	if config.RetryDelay != 1*time.Second {
		t.Errorf("expected retry delay to be 1s, got %v", config.RetryDelay)
	}

	if config.MaxRetryDelay != 10*time.Second {
		t.Errorf("expected max retry delay to be 10s, got %v", config.MaxRetryDelay)
	}

	if config.DefaultMaxTokens != 2048 {
		t.Errorf("expected default max tokens to be 2048, got %d", config.DefaultMaxTokens)
	}

	if config.DefaultTemperature != 0.7 {
		t.Errorf("expected default temperature to be 0.7, got %f", config.DefaultTemperature)
	}
}

func TestGenerateRequest(t *testing.T) {
	req := &GenerateRequest{
		Model:       "test-model",
		Prompt:      "Hello, world!",
		MaxTokens:   100,
		Temperature: 0.8,
		Context:     []string{"previous", "context"},
	}

	if req.Model != "test-model" {
		t.Errorf("expected model to be 'test-model', got %s", req.Model)
	}

	if req.Prompt != "Hello, world!" {
		t.Errorf("expected prompt to be 'Hello, world!', got %s", req.Prompt)
	}

	if req.MaxTokens != 100 {
		t.Errorf("expected max tokens to be 100, got %d", req.MaxTokens)
	}

	if req.Temperature != 0.8 {
		t.Errorf("expected temperature to be 0.8, got %f", req.Temperature)
	}

	if len(req.Context) != 2 {
		t.Errorf("expected context length to be 2, got %d", len(req.Context))
	}
}

func TestGenerateResponse(t *testing.T) {
	resp := &GenerateResponse{
		Text:          "Generated text",
		TokensUsed:    50,
		Model:         "test-model",
		FinishReason:  "stop",
		ResponseTime:  100 * time.Millisecond,
		TotalDuration: 200 * time.Millisecond,
	}

	if resp.Text != "Generated text" {
		t.Errorf("expected text to be 'Generated text', got %s", resp.Text)
	}

	if resp.TokensUsed != 50 {
		t.Errorf("expected tokens used to be 50, got %d", resp.TokensUsed)
	}

	if resp.Model != "test-model" {
		t.Errorf("expected model to be 'test-model', got %s", resp.Model)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("expected finish reason to be 'stop', got %s", resp.FinishReason)
	}

	if resp.ResponseTime != 100*time.Millisecond {
		t.Errorf("expected response time to be 100ms, got %v", resp.ResponseTime)
	}

	if resp.TotalDuration != 200*time.Millisecond {
		t.Errorf("expected total duration to be 200ms, got %v", resp.TotalDuration)
	}
}
