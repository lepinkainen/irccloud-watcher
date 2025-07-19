package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config stores the application configuration.
// It is loaded from a YAML file.
type Config struct {
	Email             string   `mapstructure:"email"`
	Password          string   `mapstructure:"password"`
	Channels          []string `mapstructure:"channels"`
	IgnoredChannels   []string `mapstructure:"ignored_channels"`
	DatabasePath      string   `mapstructure:"database_path"`
	SummaryOutputPath string   `mapstructure:"summary_output_path"`
	SummaryTime       string   `mapstructure:"summary_time"`
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

	// Validate required fields
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &c, nil
}

// Validate checks that all required configuration fields are present and valid
func (c *Config) Validate() error {
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
