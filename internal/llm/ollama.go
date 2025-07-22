package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaClient implements the Provider interface for Ollama.
type OllamaClient struct {
	config     *ProviderConfig
	httpClient *http.Client
	baseURL    string
}

// OllamaRequest represents a request to the Ollama API.
type OllamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options *OllamaOptions `json:"options,omitempty"`
	Context []int          `json:"context,omitempty"`
}

// OllamaOptions represents optional parameters for Ollama requests.
type OllamaOptions struct {
	NumPredict  int     `json:"num_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// OllamaResponse represents a response from the Ollama API.
type OllamaResponse struct {
	Model              string `json:"model"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// OllamaModel represents a model returned by the /api/tags endpoint.
type OllamaModel struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    struct {
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// OllamaModelsResponse represents the response from /api/tags.
type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaError represents an error response from Ollama.
type OllamaError struct {
	Error string `json:"error"`
}

// NewOllamaClient creates a new OllamaClient.
func NewOllamaClient(config *ProviderConfig) *OllamaClient {
	if config == nil {
		config = DefaultProviderConfig()
	}

	// Set Ollama defaults if not specified
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	if config.DefaultModel == "" {
		config.DefaultModel = "llama3.2"
	}

	return &OllamaClient{
		config:  config,
		baseURL: strings.TrimSuffix(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Name returns the provider name.
func (c *OllamaClient) Name() string {
	return "ollama"
}

// Generate generates text using the Ollama API.
func (c *OllamaClient) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	model := req.Model
	if model == "" {
		model = c.config.DefaultModel
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = c.config.DefaultMaxTokens
	}

	temperature := req.Temperature
	if temperature <= 0 {
		temperature = c.config.DefaultTemperature
	}

	ollamaReq := &OllamaRequest{
		Model:  model,
		Prompt: req.Prompt,
		Stream: false,
		Options: &OllamaOptions{
			NumPredict:  maxTokens,
			Temperature: temperature,
		},
	}

	return c.generateWithRetry(ctx, ollamaReq)
}

// generateWithRetry performs the generation with retry logic.
func (c *OllamaClient) generateWithRetry(ctx context.Context, req *OllamaRequest) (*GenerateResponse, error) {
	var lastErr error
	retryDelay := c.config.RetryDelay

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retrying
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
				// Exponential backoff
				retryDelay *= 2
				if retryDelay > c.config.MaxRetryDelay {
					retryDelay = c.config.MaxRetryDelay
				}
			}
		}

		resp, err := c.generate(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on context cancellation or certain errors
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.config.RetryAttempts+1, lastErr)
}

// generate performs a single generation request.
func (c *OllamaClient) generate(ctx context.Context, req *OllamaRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var ollamaErr OllamaError
		if json.Unmarshal(body, &ollamaErr) == nil && ollamaErr.Error != "" {
			return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, ollamaErr.Error)
		}
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	totalDuration := time.Duration(0)
	if ollamaResp.TotalDuration > 0 {
		totalDuration = time.Duration(ollamaResp.TotalDuration) * time.Nanosecond
	}

	tokenCount := ollamaResp.EvalCount
	if tokenCount == 0 && ollamaResp.Response != "" {
		// Rough estimate if not provided
		tokenCount = len(strings.Fields(ollamaResp.Response))
	}

	return &GenerateResponse{
		Text:          ollamaResp.Response,
		TokensUsed:    tokenCount,
		Model:         ollamaResp.Model,
		FinishReason:  "stop", // Ollama doesn't provide this explicitly
		ResponseTime:  responseTime,
		TotalDuration: totalDuration,
	}, nil
}

// ListModels returns available models from the Ollama instance.
func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(modelsResp.Models))
	for i, model := range modelsResp.Models {
		models[i] = model.Name
	}

	return models, nil
}

// Health checks if the Ollama instance is available.
func (c *OllamaClient) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ollama instance not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama instance returned status %d", resp.StatusCode)
	}

	return nil
}

// Close cleans up resources (no-op for HTTP client).
func (c *OllamaClient) Close() error {
	return nil
}
