package llm

import (
	"context"
	"time"
)

// GenerateRequest represents a request to generate text from an LLM.
type GenerateRequest struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float64
	Context     []string
}

// GenerateResponse represents a response from an LLM generation request.
type GenerateResponse struct {
	Text          string
	TokensUsed    int
	Model         string
	FinishReason  string
	ResponseTime  time.Duration
	TotalDuration time.Duration
}

// Provider defines the interface for different LLM services.
type Provider interface {
	// Generate generates text using the LLM.
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// ListModels returns available models for this provider.
	ListModels(ctx context.Context) ([]string, error)

	// Health checks if the provider is available and responding.
	Health(ctx context.Context) error

	// Name returns the name of this provider.
	Name() string

	// Close cleans up any resources used by the provider.
	Close() error
}

// ProviderConfig holds common configuration for LLM providers.
type ProviderConfig struct {
	BaseURL            string
	Timeout            time.Duration
	RetryAttempts      int
	RetryDelay         time.Duration
	MaxRetryDelay      time.Duration
	DefaultModel       string
	DefaultMaxTokens   int
	DefaultTemperature float64
}

// DefaultProviderConfig returns default configuration values.
func DefaultProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		Timeout:            30 * time.Second,
		RetryAttempts:      3,
		RetryDelay:         1 * time.Second,
		MaxRetryDelay:      10 * time.Second,
		DefaultMaxTokens:   2048,
		DefaultTemperature: 0.7,
	}
}
