package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOllamaClient(t *testing.T) {
	client := NewOllamaClient(nil)

	if client.Name() != "ollama" {
		t.Errorf("expected name to be 'ollama', got %s", client.Name())
	}

	if client.config.BaseURL != "http://localhost:11434" {
		t.Errorf("expected base URL to be 'http://localhost:11434', got %s", client.config.BaseURL)
	}

	if client.config.DefaultModel != "llama3.2" {
		t.Errorf("expected default model to be 'llama3.2', got %s", client.config.DefaultModel)
	}
}

func TestNewOllamaClientWithCustomConfig(t *testing.T) {
	config := &ProviderConfig{
		BaseURL:      "http://custom:8080",
		DefaultModel: "custom-model",
		Timeout:      5 * time.Second,
	}

	client := NewOllamaClient(config)

	if client.baseURL != "http://custom:8080" {
		t.Errorf("expected base URL to be 'http://custom:8080', got %s", client.baseURL)
	}

	if client.config.DefaultModel != "custom-model" {
		t.Errorf("expected default model to be 'custom-model', got %s", client.config.DefaultModel)
	}
}

func TestOllamaClient_Generate_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("expected path /api/generate, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("expected model 'test-model', got %s", req.Model)
		}

		if req.Prompt != "Hello, world!" {
			t.Errorf("expected prompt 'Hello, world!', got %s", req.Prompt)
		}

		resp := OllamaResponse{
			Model:         "test-model",
			Response:      "Hello there!",
			Done:          true,
			TotalDuration: 100000000, // 100ms in nanoseconds
			EvalCount:     5,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ProviderConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client := NewOllamaClient(config)

	req := &GenerateRequest{
		Model:  "test-model",
		Prompt: "Hello, world!",
	}

	ctx := context.Background()
	resp, err := client.Generate(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Text != "Hello there!" {
		t.Errorf("expected text 'Hello there!', got %s", resp.Text)
	}

	if resp.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %s", resp.Model)
	}

	if resp.TokensUsed != 5 {
		t.Errorf("expected 5 tokens used, got %d", resp.TokensUsed)
	}

	if resp.TotalDuration != 100*time.Millisecond {
		t.Errorf("expected total duration 100ms, got %v", resp.TotalDuration)
	}
}

func TestOllamaClient_Generate_WithDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Check that defaults are applied
		if req.Model != "llama3.2" {
			t.Errorf("expected default model 'llama3.2', got %s", req.Model)
		}

		if req.Options == nil {
			t.Fatal("expected options to be set")
		}

		if req.Options.NumPredict != 2048 {
			t.Errorf("expected default max tokens 2048, got %d", req.Options.NumPredict)
		}

		if req.Options.Temperature != 0.7 {
			t.Errorf("expected default temperature 0.7, got %f", req.Options.Temperature)
		}

		resp := OllamaResponse{
			Model:    req.Model,
			Response: "Generated with defaults",
			Done:     true,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ProviderConfig{
		BaseURL:            server.URL,
		Timeout:            5 * time.Second,
		DefaultModel:       "llama3.2",
		DefaultMaxTokens:   2048,
		DefaultTemperature: 0.7,
	}

	client := NewOllamaClient(config)

	req := &GenerateRequest{
		Prompt: "Test prompt",
		// Model, MaxTokens, and Temperature not set - should use defaults
	}

	ctx := context.Background()
	_, err := client.Generate(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOllamaClient_Generate_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(OllamaError{Error: "model not found"})
	}))
	defer server.Close()

	config := &ProviderConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 0, // No retries for this test
	}

	client := NewOllamaClient(config)

	req := &GenerateRequest{
		Model:  "nonexistent-model",
		Prompt: "Hello, world!",
	}

	ctx := context.Background()
	_, err := client.Generate(ctx, req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("expected error to contain 'model not found', got %s", err.Error())
	}
}

func TestOllamaClient_Generate_NilRequest(t *testing.T) {
	client := NewOllamaClient(nil)

	ctx := context.Background()
	_, err := client.Generate(ctx, nil)

	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}

	if !strings.Contains(err.Error(), "request cannot be nil") {
		t.Errorf("expected error to contain 'request cannot be nil', got %s", err.Error())
	}
}

func TestOllamaClient_ListModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("expected path /api/tags, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		resp := OllamaModelsResponse{
			Models: []OllamaModel{
				{Name: "llama3.2"},
				{Name: "codellama"},
				{Name: "mistral"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ProviderConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client := NewOllamaClient(config)

	ctx := context.Background()
	models, err := client.ListModels(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedModels := []string{"llama3.2", "codellama", "mistral"}
	if len(models) != len(expectedModels) {
		t.Errorf("expected %d models, got %d", len(expectedModels), len(models))
	}

	for i, expected := range expectedModels {
		if models[i] != expected {
			t.Errorf("expected model %s at index %d, got %s", expected, i, models[i])
		}
	}
}

func TestOllamaClient_Health_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(OllamaModelsResponse{})
	}))
	defer server.Close()

	config := &ProviderConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client := NewOllamaClient(config)

	ctx := context.Background()
	err := client.Health(ctx)

	if err != nil {
		t.Errorf("expected health check to pass, got error: %v", err)
	}
}

func TestOllamaClient_Health_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &ProviderConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client := NewOllamaClient(config)

	ctx := context.Background()
	err := client.Health(ctx)

	if err == nil {
		t.Fatal("expected health check to fail, got nil error")
	}

	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected error to contain 'status 500', got %s", err.Error())
	}
}

func TestOllamaClient_Close(t *testing.T) {
	client := NewOllamaClient(nil)

	err := client.Close()
	if err != nil {
		t.Errorf("expected Close to return nil, got %v", err)
	}
}
