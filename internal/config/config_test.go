package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfigValid(t *testing.T) {
	// Create temporary config file
	configContent := `
email: "test@example.com"
password: "testpassword"
channels:
  - "#test1"
  - "#test2"
ignored_channels:
  - "#spam"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
llm:
  provider: "ollama"
  base_url: "http://localhost:11434"
  model: "llama3.2"
  temperature: 0.7
  max_tokens: 1000
`

	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Test basic fields
	if config.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", config.Email)
	}
	if config.Password != "testpassword" {
		t.Errorf("Expected password 'testpassword', got '%s'", config.Password)
	}
	if len(config.Channels) != 2 {
		t.Errorf("Expected 2 channels, got %d", len(config.Channels))
	}
	if config.Channels[0] != "#test1" {
		t.Errorf("Expected first channel '#test1', got '%s'", config.Channels[0])
	}
	if len(config.IgnoredChannels) != 1 {
		t.Errorf("Expected 1 ignored channel, got %d", len(config.IgnoredChannels))
	}
	if config.IgnoredChannels[0] != "#spam" {
		t.Errorf("Expected ignored channel '#spam', got '%s'", config.IgnoredChannels[0])
	}

	// Test LLM configuration
	if config.LLM.Provider != "ollama" {
		t.Errorf("Expected LLM provider 'ollama', got '%s'", config.LLM.Provider)
	}
	if config.LLM.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected LLM base URL 'http://localhost:11434', got '%s'", config.LLM.BaseURL)
	}
	if config.LLM.Model != "llama3.2" {
		t.Errorf("Expected LLM model 'llama3.2', got '%s'", config.LLM.Model)
	}
	if config.LLM.Temperature != 0.7 {
		t.Errorf("Expected LLM temperature 0.7, got %f", config.LLM.Temperature)
	}
	if config.LLM.MaxTokens != 1000 {
		t.Errorf("Expected LLM max_tokens 1000, got %d", config.LLM.MaxTokens)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Create minimal config file
	configContent := `
email: "test@example.com"
password: "testpassword"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
`

	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Test default LLM values
	if config.LLM.Provider != "ollama" {
		t.Errorf("Expected default LLM provider 'ollama', got '%s'", config.LLM.Provider)
	}
	if config.LLM.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected default LLM base URL 'http://localhost:11434', got '%s'", config.LLM.BaseURL)
	}
	if config.LLM.Model != "llama3.2" {
		t.Errorf("Expected default LLM model 'llama3.2', got '%s'", config.LLM.Model)
	}
	if config.LLM.Temperature != 0.7 {
		t.Errorf("Expected default LLM temperature 0.7, got %f", config.LLM.Temperature)
	}
	if config.LLM.MaxTokens != 1000 {
		t.Errorf("Expected default LLM max_tokens 1000, got %d", config.LLM.MaxTokens)
	}

	// Test default connection values
	if config.Connection.HeartbeatInterval != "30s" {
		t.Errorf("Expected default heartbeat interval '30s', got '%s'", config.Connection.HeartbeatInterval)
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("IRCCLOUD_EMAIL", "env@example.com")
	os.Setenv("IRCCLOUD_PASSWORD", "envpassword")
	os.Setenv("LLM_API_KEY", "env-api-key")
	defer func() {
		os.Unsetenv("IRCCLOUD_EMAIL")
		os.Unsetenv("IRCCLOUD_PASSWORD")
		os.Unsetenv("LLM_API_KEY")
	}()

	configContent := `
email: "config@example.com"
password: "configpassword"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
llm:
  provider: "openai"
  api_key: "config-api-key"
`

	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Test environment variable overrides
	if config.Email != "env@example.com" {
		t.Errorf("Expected email from env 'env@example.com', got '%s'", config.Email)
	}
	if config.Password != "envpassword" {
		t.Errorf("Expected password from env 'envpassword', got '%s'", config.Password)
	}
	if config.LLM.APIKey != "env-api-key" {
		t.Errorf("Expected API key from env 'env-api-key', got '%s'", config.LLM.APIKey)
	}
}

func TestProviderSpecificEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		envVar   string
		envValue string
	}{
		{"OpenAI", "openai", "OPENAI_API_KEY", "openai-key"},
		{"Anthropic", "anthropic", "ANTHROPIC_API_KEY", "anthropic-key"},
		{"Gemini", "gemini", "GEMINI_API_KEY", "gemini-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			configContent := `
email: "test@example.com"
password: "testpassword"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
llm:
  provider: "` + tt.provider + `"
`

			tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
				t.Fatalf("Failed to write config: %v", writeErr)
			}
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			if config.LLM.APIKey != tt.envValue {
				t.Errorf("Expected API key '%s', got '%s'", tt.envValue, config.LLM.APIKey)
			}
		})
	}
}

func TestValidationRequired(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectedError string
	}{
		{
			name: "Missing email",
			configContent: `
password: "testpassword"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
`,
			expectedError: "email is required",
		},
		{
			name: "Missing password",
			configContent: `
email: "test@example.com"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
`,
			expectedError: "password is required",
		},
		{
			name: "Missing database_path",
			configContent: `
email: "test@example.com"
password: "testpassword"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
`,
			expectedError: "database_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, writeErr := tmpFile.WriteString(tt.configContent); writeErr != nil {
				t.Fatalf("Failed to write config: %v", writeErr)
			}
			tmpFile.Close()

			_, err = LoadConfig(tmpFile.Name())
			if err == nil {
				t.Fatalf("Expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestLLMValidation(t *testing.T) {
	tests := []struct {
		name          string
		llmConfig     string
		expectedError string
	}{
		{
			name: "Invalid provider",
			llmConfig: `
llm:
  provider: "invalid"
  model: "test-model"
`,
			expectedError: "unsupported LLM provider: invalid",
		},
		{
			name: "Missing API key for non-ollama provider",
			llmConfig: `
llm:
  provider: "openai"
  model: "gpt-4"
`,
			expectedError: "api_key is required for provider openai",
		},
		{
			name: "Invalid temperature too low",
			llmConfig: `
llm:
  provider: "ollama"
  model: "llama3.2"
  temperature: -0.1
`,
			expectedError: "temperature must be between 0 and 2",
		},
		{
			name: "Invalid temperature too high",
			llmConfig: `
llm:
  provider: "ollama"
  model: "llama3.2"
  temperature: 2.1
`,
			expectedError: "temperature must be between 0 and 2",
		},
		{
			name: "Invalid max_tokens",
			llmConfig: `
llm:
  provider: "ollama"
  model: "llama3.2"
  max_tokens: -1
`,
			expectedError: "max_tokens must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configContent := `
email: "test@example.com"
password: "testpassword"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
` + tt.llmConfig

			tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
				t.Fatalf("Failed to write config: %v", writeErr)
			}
			tmpFile.Close()

			_, err = LoadConfig(tmpFile.Name())
			if err == nil {
				t.Fatalf("Expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestLLMDefaultModels(t *testing.T) {
	tests := []struct {
		provider      string
		expectedModel string
	}{
		{"ollama", "llama3.2"},
		{"openai", "gpt-4o-mini"},
		{"anthropic", "claude-3-haiku-20240307"},
		{"gemini", "gemini-1.5-flash"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			configContent := `
email: "test@example.com"
password: "testpassword"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
llm:
  provider: "` + tt.provider + `"
`
			if tt.provider != "ollama" {
				configContent += `  api_key: "test-key"`
			}

			tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
				t.Fatalf("Failed to write config: %v", writeErr)
			}
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			if config.LLM.Model != tt.expectedModel {
				t.Errorf("Expected default model '%s' for provider '%s', got '%s'", tt.expectedModel, tt.provider, config.LLM.Model)
			}
		})
	}
}

func TestValidLLMConfiguration(t *testing.T) {
	configContent := `
email: "test@example.com"
password: "testpassword"
database_path: "test.db"
summary_output_path: "/tmp/summary.txt"
summary_time: "0 6 * * *"
llm:
  provider: "ollama"
  base_url: "http://localhost:11434"
  model: "llama3.2"
  temperature: 0.8
  max_tokens: 1500
`

	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Validate that configuration loads without errors
	if config.LLM.Provider != "ollama" {
		t.Errorf("Expected provider 'ollama', got '%s'", config.LLM.Provider)
	}
	if config.LLM.Temperature != 0.8 {
		t.Errorf("Expected temperature 0.8, got %f", config.LLM.Temperature)
	}
	if config.LLM.MaxTokens != 1500 {
		t.Errorf("Expected max_tokens 1500, got %d", config.LLM.MaxTokens)
	}
}
