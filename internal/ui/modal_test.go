package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/config"
)

func TestProviderFormModalOpen(t *testing.T) {
	m := createTestModel()
	m.Input = "/provider"

	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	m2 := updatedModel.(Model)

	if m2.ModalMode != "provider_form" {
		t.Errorf("Expected modal mode 'provider_form', got %q", m2.ModalMode)
	}
	if m2.FormField != 0 {
		t.Errorf("Expected form field 0 (name), got %d", m2.FormField)
	}
}

func TestProviderFormTypingAndNavigation(t *testing.T) {
	m := createTestModel()
	m.ModalMode = "provider_form"
	m.FormField = 0

	// Type name
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "m"}))
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "y"}))
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "provider"}))
	if m.FormName != "myprovider" {
		t.Errorf("Expected form name 'myprovider', got %q", m.FormName)
	}

	// Enter advances to field 1 (base URL)
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	if m.FormField != 1 {
		t.Errorf("Expected form field 1 after enter, got %d", m.FormField)
	}

	// Type base URL
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "http://localhost:11434/v1"}))
	if m.FormBaseURL != "http://localhost:11434/v1" {
		t.Errorf("Expected base URL, got %q", m.FormBaseURL)
	}

	// Enter advances to field 2 (API key)
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	if m.FormField != 2 {
		t.Errorf("Expected form field 2 after enter, got %d", m.FormField)
	}

	// Type API key
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "sk-secret"}))
	if m.FormAPIKey != "sk-secret" {
		t.Errorf("Expected API key 'sk-secret', got %q", m.FormAPIKey)
	}
}

func TestProviderFormSaveOnFinalEnter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := config.DefaultConfig()
	m := createTestModel()
	m.Config = cfg
	m.ConfigPath = path
	m.ModalMode = "provider_form"
	m.FormName = "ollama"
	m.FormBaseURL = "http://localhost:11434/v1"
	m.FormAPIKey = "ollama-key"
	m.FormField = 2

	// Final enter saves and closes
	m2, _ := m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "enter"}))

	if m2.ModalMode != "" {
		t.Errorf("Expected modal closed after save, got %q", m2.ModalMode)
	}

	p, ok := m2.Config.Providers["ollama"]
	if !ok {
		t.Fatal("Expected 'ollama' provider to be saved in config")
	}
	if p.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("Expected base URL, got %q", p.BaseURL)
	}
	if p.APIKey != "ollama-key" {
		t.Errorf("Expected API key, got %q", p.APIKey)
	}

	// Verify it persisted to JSON
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	if _, ok := loaded.Providers["ollama"]; !ok {
		t.Error("Expected 'ollama' provider in persisted JSON")
	}
}

func TestProviderFormEscCancels(t *testing.T) {
	m := createTestModel()
	m.ModalMode = "provider_form"
	m.FormName = "test"
	m.FormBaseURL = "http://x"
	m.FormAPIKey = "key"
	m.FormField = 1

	updatedModel, _ := m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "esc"}))
	m2 := updatedModel

	if m2.ModalMode != "" {
		t.Errorf("Expected modal closed on esc, got %q", m2.ModalMode)
	}
	if m2.FormName != "" || m2.FormBaseURL != "" || m2.FormAPIKey != "" {
		t.Error("Expected form fields to be cleared on esc")
	}

	// Verify nothing was saved
	if _, ok := m2.Config.Providers["test"]; ok {
		t.Error("Expected provider NOT to be saved on esc")
	}
}

func TestProviderFormBackspace(t *testing.T) {
	m := createTestModel()
	m.ModalMode = "provider_form"
	m.FormField = 0
	m.FormName = "abc"

	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "backspace"}))
	if m.FormName != "ab" {
		t.Errorf("Expected 'ab' after backspace, got %q", m.FormName)
	}
}

func TestProviderFormUpDownNavigation(t *testing.T) {
	m := createTestModel()
	m.ModalMode = "provider_form"
	m.FormField = 0

	// Down moves to field 1
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "down"}))
	if m.FormField != 1 {
		t.Errorf("Expected field 1 after down, got %d", m.FormField)
	}

	// Down again moves to field 2
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "down"}))
	if m.FormField != 2 {
		t.Errorf("Expected field 2 after down, got %d", m.FormField)
	}

	// Down at max stays at 2
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "down"}))
	if m.FormField != 2 {
		t.Errorf("Expected field 2 (max), got %d", m.FormField)
	}

	// Up moves back to 1
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "up"}))
	if m.FormField != 1 {
		t.Errorf("Expected field 1 after up, got %d", m.FormField)
	}
}

func TestConnectListModalOpen(t *testing.T) {
	m := createTestModel()
	m.Input = "/connect"

	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	m2 := updatedModel.(Model)

	if m2.ModalMode != "connect_list" {
		t.Errorf("Expected modal mode 'connect_list', got %q", m2.ModalMode)
	}
	if len(m2.ConnectNames) == 0 {
		t.Error("Expected connect list to be populated with provider names")
	}
}

func TestConnectListNavigation(t *testing.T) {
	m := createTestModel()
	m.Config.Providers = map[string]config.ProviderConfig{
		"alpha": {BaseURL: "http://a", APIKey: "k1"},
		"beta":  {BaseURL: "http://b", APIKey: "k2"},
		"gamma": {BaseURL: "http://c", APIKey: "k3"},
	}
	m.ModalMode = "connect_list"
	m.ConnectNames = []string{"alpha", "beta", "gamma"}
	m.ConnectIndex = 0

	// Down moves to index 1
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "down"}))
	if m.ConnectIndex != 1 {
		t.Errorf("Expected index 1 after down, got %d", m.ConnectIndex)
	}

	// Down again to index 2
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "down"}))
	if m.ConnectIndex != 2 {
		t.Errorf("Expected index 2 after down, got %d", m.ConnectIndex)
	}

	// Down at max stays
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "down"}))
	if m.ConnectIndex != 2 {
		t.Errorf("Expected index 2 (max), got %d", m.ConnectIndex)
	}

	// Up back to 1
	m, _ = m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "up"}))
	if m.ConnectIndex != 1 {
		t.Errorf("Expected index 1 after up, got %d", m.ConnectIndex)
	}
}

func TestConnectListSelectSwitchesProvider(t *testing.T) {
	m := createTestModel()
	m.Config.Providers = map[string]config.ProviderConfig{
		"default": {BaseURL: "http://0.0.0.0:8080", APIKey: "sk-no-key", Model: ""},
		"ollama":  {BaseURL: "http://localhost:11434/v1", APIKey: "ollama", Model: "llama3"},
	}
	m.Config.ActiveProvider = "default"
	m.ModalMode = "connect_list"
	m.ConnectNames = []string{"default", "ollama"}
	m.ConnectIndex = 1 // select ollama

	updatedModel, _ := m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	m2 := updatedModel

	if m2.ModalMode != "" {
		t.Error("Expected modal closed after connect")
	}
	if m2.Config.ActiveProvider != "ollama" {
		t.Errorf("Expected active provider 'ollama', got %q", m2.Config.ActiveProvider)
	}
	if m2.LlmClient.URL() != "http://localhost:11434/v1" {
		t.Errorf("Expected client URL updated, got %q", m2.LlmClient.URL())
	}
	if m2.LlmClient.Model() != "llama3" {
		t.Errorf("Expected client model updated, got %q", m2.LlmClient.Model())
	}
}

func TestConnectListEscCancels(t *testing.T) {
	m := createTestModel()
	m.Config.ActiveProvider = "default"
	m.ModalMode = "connect_list"
	m.ConnectNames = []string{"default"}
	m.ConnectIndex = 0

	updatedModel, _ := m.handleModalKey(tea.KeyPressMsg(tea.Key{Text: "esc"}))
	m2 := updatedModel

	if m2.ModalMode != "" {
		t.Error("Expected modal closed on esc")
	}
	if m2.Config.ActiveProvider != "default" {
		t.Error("Expected active provider unchanged on esc")
	}
}

func TestModalInterceptsKeys(t *testing.T) {
	m := createTestModel()
	m.ModalMode = "provider_form"
	m.FormField = 0

	// Regular typing should go to the form, not the input
	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "x"}))
	m2 := updatedModel.(Model)

	if m2.FormName != "x" {
		t.Errorf("Expected 'x' in form name, got %q", m2.FormName)
	}
	if m2.Input != "" {
		t.Errorf("Expected input to remain empty during modal, got %q", m2.Input)
	}
}

func TestModalCtrlCStillQuits(t *testing.T) {
	m := createTestModel()
	m.ModalMode = "provider_form"

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Text: "ctrl+c"}))
	if cmd == nil {
		t.Error("Expected ctrl+c to return quit command even during modal")
	}
}
