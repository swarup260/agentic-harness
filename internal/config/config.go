package config

// Config represents the application settings.
type Config struct {
	LLMURL         string
	SystemPrompt   string
	MaxHistorySize int
}

// DefaultConfig returns the default configuration settings.
func DefaultConfig() *Config {
	return &Config{
		LLMURL:         "http://0.0.0.0:8080",
		SystemPrompt:   "You are a senior stock analyst with 20+ years of experience. You always output the final answer in bullet points.",
		MaxHistorySize: 5,
	}
}
