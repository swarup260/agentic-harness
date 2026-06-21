package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ProviderConfig holds connection details for a single OpenAI-compatible endpoint.
type ProviderConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// Config represents the application settings loaded from config.json.
type Config struct {
	Providers      map[string]ProviderConfig `json:"providers"`
	ActiveProvider string                    `json:"active_provider"`
	SystemPrompt   string                    `json:"system_prompt"`
	MaxHistorySize int                       `json:"max_history_size"`
	Seed           *int64                    `json:"seed,omitempty"`
	Temperature    *float64                  `json:"temperature,omitempty"`
	DataSources    map[string]string         `json:"data_sources"`
}

// DefaultConfig returns the default configuration settings.
func DefaultConfig() *Config {
	seed := int64(0)
	temp := 0.0
	return &Config{
		Providers: map[string]ProviderConfig{
			"default": {
				BaseURL: "http://0.0.0.0:8080",
				APIKey:  "sk-no-key",
				Model:   "",
			},
		},
		ActiveProvider: "default",
		SystemPrompt:   "You are a senior stock analyst with 20+ years of experience. You always output the final answer in bullet points.",
		MaxHistorySize: 5,
		Seed:           &seed,
		Temperature:    &temp,
		DataSources:    map[string]string{},
	}
}

// Save writes the configuration to a JSON file.
func (c *Config) Save(path string) error {
	return writeTemplate(path, c)
}

// ActiveProviderConfig returns the ProviderConfig for the active provider.
func (c *Config) ActiveProviderConfig() ProviderConfig {
	if p, ok := c.Providers[c.ActiveProvider]; ok {
		return p
	}
	return ProviderConfig{}
}

// Load reads configuration from a JSON file. If the file does not exist, a
// template is written with default values and those defaults are returned.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if writeErr := writeTemplate(path, cfg); writeErr != nil {
				return cfg, fmt.Errorf("failed to write config template: %w", writeErr)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.fillDefaults()
	return &cfg, nil
}

func (c *Config) fillDefaults() {
	def := DefaultConfig()
	if c.Providers == nil || len(c.Providers) == 0 {
		c.Providers = def.Providers
	}
	if c.ActiveProvider == "" {
		c.ActiveProvider = "default"
	}
	if c.SystemPrompt == "" {
		c.SystemPrompt = def.SystemPrompt
	}
	if c.MaxHistorySize == 0 {
		c.MaxHistorySize = def.MaxHistorySize
	}
	if c.DataSources == nil {
		c.DataSources = map[string]string{}
	}
}

func writeTemplate(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0600)
}
