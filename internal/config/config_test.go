package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ActiveProvider != "default" {
		t.Errorf("Expected active provider 'default', got %q", cfg.ActiveProvider)
	}
	p, ok := cfg.Providers["default"]
	if !ok {
		t.Fatal("Expected 'default' provider to exist")
	}
	if p.BaseURL != "http://0.0.0.0:8080" {
		t.Errorf("Expected base URL 'http://0.0.0.0:8080', got %q", p.BaseURL)
	}
	if cfg.MaxHistorySize != 5 {
		t.Errorf("Expected max history size 5, got %d", cfg.MaxHistorySize)
	}
	if cfg.Seed == nil || *cfg.Seed != 0 {
		t.Error("Expected seed to be 0")
	}
	if cfg.Temperature == nil || *cfg.Temperature != 0 {
		t.Error("Expected temperature to be 0")
	}
}

func TestActiveProviderConfig(t *testing.T) {
	cfg := DefaultConfig()
	p := cfg.ActiveProviderConfig()
	if p.BaseURL != "http://0.0.0.0:8080" {
		t.Errorf("Expected base URL 'http://0.0.0.0:8080', got %q", p.BaseURL)
	}

	cfg.ActiveProvider = "nonexistent"
	p = cfg.ActiveProviderConfig()
	if p.BaseURL != "" {
		t.Errorf("Expected empty config for nonexistent provider, got %+v", p)
	}
}

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{
		"providers": {
			"openai": {"base_url": "https://api.openai.com/v1", "api_key": "sk-test", "model": "gpt-4o"},
			"local": {"base_url": "http://localhost:11434/v1", "api_key": "ollama", "model": "llama3"}
		},
		"active_provider": "openai",
		"system_prompt": "You are a test bot.",
		"max_history_size": 10,
		"data_sources": {"finnhub": "test-key"}
	}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ActiveProvider != "openai" {
		t.Errorf("Expected active provider 'openai', got %q", cfg.ActiveProvider)
	}
	p := cfg.ActiveProviderConfig()
	if p.APIKey != "sk-test" {
		t.Errorf("Expected API key 'sk-test', got %q", p.APIKey)
	}
	if p.Model != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got %q", p.Model)
	}
	if cfg.MaxHistorySize != 10 {
		t.Errorf("Expected max history size 10, got %d", cfg.MaxHistorySize)
	}
	if cfg.DataSources["finnhub"] != "test-key" {
		t.Error("Expected finnhub key in data sources")
	}
}

func TestLoadMissingFileCreatesTemplate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ActiveProvider != "default" {
		t.Errorf("Expected default active provider, got %q", cfg.ActiveProvider)
	}

	// Verify template was written
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Expected config template to be created")
	}

	// Verify template is valid JSON with expected fields
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var written Config
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("Template is not valid JSON: %v", err)
	}
	if written.ActiveProvider != "default" {
		t.Errorf("Expected template active provider 'default', got %q", written.ActiveProvider)
	}
}

func TestLoadFillsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Minimal config — missing optional fields
	data := `{"providers": {"default": {"base_url": "http://localhost:8080", "api_key": "k", "model": "m"}}}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.SystemPrompt == "" {
		t.Error("Expected system prompt to be filled with default")
	}
	if cfg.MaxHistorySize != 5 {
		t.Errorf("Expected max history size 5, got %d", cfg.MaxHistorySize)
	}
	if cfg.DataSources == nil {
		t.Error("Expected data sources to be initialized")
	}
}
