package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config stores the application configuration.
// It is loaded from a YAML file.
type Config struct {
	Email             string           `mapstructure:"email"`
	Password          string           `mapstructure:"password"`
	Channels          []string         `mapstructure:"channels"`
	IgnoredChannels   []string         `mapstructure:"ignored_channels"`
	DatabasePath      string           `mapstructure:"database_path"`
	SummaryOutputPath string           `mapstructure:"summary_output_path"`
	SummaryTime       string           `mapstructure:"summary_time"`
	Connection        ConnectionConfig `mapstructure:"connection"`
	LLM               LLMConfig        `mapstructure:"llm"`
}

// ConnectionConfig stores WebSocket connection parameters.
type ConnectionConfig struct {
	HeartbeatInterval string  `mapstructure:"heartbeat_interval"`
	MaxRetryAttempts  int     `mapstructure:"max_retry_attempts"`
	InitialRetryDelay string  `mapstructure:"initial_retry_delay"`
	MaxRetryDelay     string  `mapstructure:"max_retry_delay"`
	BackoffMultiplier float64 `mapstructure:"backoff_multiplier"`
	ConnectionTimeout string  `mapstructure:"connection_timeout"`
	PingInterval      string  `mapstructure:"ping_interval"`
}

// LLMConfig stores LLM provider settings for summary generation.
type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	BaseURL     string  `mapstructure:"base_url"`
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	APIKey      string  `mapstructure:"api_key"`
}

// LoadConfig loads the configuration from the given path.
func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv() // Enable environment variable substitution

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}

	// Override with environment variables if set
	if email := os.Getenv("IRCCLOUD_EMAIL"); email != "" {
		c.Email = email
	}
	if password := os.Getenv("IRCCLOUD_PASSWORD"); password != "" {
		c.Password = password
	}

	// Override LLM API key with environment variable if set
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		c.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" && c.LLM.Provider == "openai" {
		c.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" && c.LLM.Provider == "anthropic" {
		c.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" && c.LLM.Provider == "gemini" {
		c.LLM.APIKey = apiKey
	}

	// Set default connection values if not specified
	setConnectionDefaults(&c.Connection)

	// Validate required fields first (before setting defaults)
	if err := c.ValidateRequired(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set default LLM values if not specified
	setLLMDefaults(&c.LLM)

	// Validate LLM configuration after defaults are set
	if c.LLM.Provider != "" {
		if err := c.validateLLMConfig(); err != nil {
			return nil, fmt.Errorf("LLM configuration validation failed: %w", err)
		}
	}

	return &c, nil
}

// ValidateRequired checks that all required configuration fields are present
func (c *Config) ValidateRequired() error {
	if c.Email == "" {
		return fmt.Errorf("email is required (set in config.yaml or IRCCLOUD_EMAIL environment variable)")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required (set in config.yaml or IRCCLOUD_PASSWORD environment variable)")
	}
	if c.DatabasePath == "" {
		return fmt.Errorf("database_path is required")
	}
	if c.SummaryOutputPath == "" {
		return fmt.Errorf("summary_output_path is required")
	}
	if c.SummaryTime == "" {
		return fmt.Errorf("summary_time is required (cron expression)")
	}

	return nil
}

// Validate checks that all required configuration fields are present and valid
func (c *Config) Validate() error {
	if err := c.ValidateRequired(); err != nil {
		return err
	}

	// Validate LLM configuration if provider is specified
	if c.LLM.Provider != "" {
		if err := c.validateLLMConfig(); err != nil {
			return fmt.Errorf("LLM configuration validation failed: %w", err)
		}
	}

	return nil
}

// validateLLMConfig validates LLM-specific configuration
func (c *Config) validateLLMConfig() error {
	validProviders := map[string]bool{
		"ollama":    true,
		"openai":    true,
		"anthropic": true,
		"gemini":    true,
	}

	if !validProviders[c.LLM.Provider] {
		return fmt.Errorf("unsupported LLM provider: %s (supported: ollama, openai, anthropic, gemini)", c.LLM.Provider)
	}

	if c.LLM.Provider != "ollama" && c.LLM.APIKey == "" {
		return fmt.Errorf("api_key is required for provider %s (set via environment variable or config file)", c.LLM.Provider)
	}

	if c.LLM.Model == "" {
		return fmt.Errorf("model is required for LLM provider %s", c.LLM.Provider)
	}

	if c.LLM.Temperature < 0 || c.LLM.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2, got %f", c.LLM.Temperature)
	}

	if c.LLM.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be greater than 0, got %d", c.LLM.MaxTokens)
	}

	return nil
}

// setConnectionDefaults sets default values for connection configuration
func setConnectionDefaults(c *ConnectionConfig) {
	if c.HeartbeatInterval == "" {
		c.HeartbeatInterval = "30s"
	}
	if c.MaxRetryAttempts == 0 {
		c.MaxRetryAttempts = 10
	}
	if c.InitialRetryDelay == "" {
		c.InitialRetryDelay = "1s"
	}
	if c.MaxRetryDelay == "" {
		c.MaxRetryDelay = "5m"
	}
	if c.BackoffMultiplier == 0 {
		c.BackoffMultiplier = 2.0
	}
	if c.ConnectionTimeout == "" {
		c.ConnectionTimeout = "45s"
	}
	if c.PingInterval == "" {
		c.PingInterval = "60s"
	}
}

// setLLMDefaults sets default values for LLM configuration
func setLLMDefaults(c *LLMConfig) {
	if c.Provider == "" {
		c.Provider = "ollama"
	}
	if c.BaseURL == "" && c.Provider == "ollama" {
		c.BaseURL = "http://localhost:11434"
	}
	if c.Model == "" {
		switch c.Provider {
		case "ollama":
			c.Model = "llama3.2"
		case "openai":
			c.Model = "gpt-4o-mini"
		case "anthropic":
			c.Model = "claude-3-haiku-20240307"
		case "gemini":
			c.Model = "gemini-1.5-flash"
		}
	}
	if c.Temperature == 0 {
		c.Temperature = 0.7
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 1000
	}
}
