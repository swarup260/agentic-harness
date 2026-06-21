package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestMatchingCommands(t *testing.T) {
	// Exact prefix match
	matches := matchingCommands("/h")
	names := suggestionNames(matches)
	if len(names) != 2 {
		t.Fatalf("Expected 2 matches for '/h', got %d: %v", len(names), names)
	}
	if names[0] != "/help" || names[1] != "/history" {
		t.Errorf("Expected [/help /history], got %v", names)
	}

	// Just slash matches all
	matches = matchingCommands("/")
	if len(matches) != len(slashCommands) {
		t.Errorf("Expected %d matches for '/', got %d", len(slashCommands), len(matches))
	}

	// No match
	matches = matchingCommands("/xyz")
	if len(matches) != 0 {
		t.Errorf("Expected 0 matches for '/xyz', got %d", len(matches))
	}

	// Non-slash input returns nil
	matches = matchingCommands("hello")
	if matches != nil {
		t.Error("Expected nil for non-slash input")
	}
}

func TestUpdateAutocompleteShowsOnSlash(t *testing.T) {
	m := createTestModel()
	m.Input = "/h"
	m.updateAutocomplete()

	if !m.autocompleteVisible() {
		t.Error("Expected autocomplete to be visible after typing '/h'")
	}
	if len(m.AutocompleteSuggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(m.AutocompleteSuggestions))
	}
}

func TestUpdateAutocompleteHidesOnSpace(t *testing.T) {
	m := createTestModel()
	m.Input = "/history 10"
	m.updateAutocomplete()

	if m.autocompleteVisible() {
		t.Error("Expected autocomplete hidden when input has a space")
	}
}

func TestUpdateAutocompleteHidesOnNonSlash(t *testing.T) {
	m := createTestModel()
	m.Input = "what stocks?"
	m.updateAutocomplete()

	if m.autocompleteVisible() {
		t.Error("Expected autocomplete hidden for non-slash input")
	}
}

func TestUpdateAutocompleteResetsIndex(t *testing.T) {
	m := createTestModel()
	m.AutocompleteIndex = 5
	m.Input = "/h"
	m.updateAutocomplete()

	if m.AutocompleteIndex != 0 {
		t.Errorf("Expected index reset to 0, got %d", m.AutocompleteIndex)
	}
}

func TestTypingShowsDropdown(t *testing.T) {
	m := createTestModel()

	// Type "/"
	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "/"}))
	m = updatedModel.(Model)
	if !m.autocompleteVisible() {
		t.Error("Expected dropdown visible after typing '/'")
	}

	// Type "h"
	updatedModel, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "h"}))
	m = updatedModel.(Model)
	if !m.autocompleteVisible() {
		t.Error("Expected dropdown visible after typing '/h'")
	}
	if len(m.AutocompleteSuggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(m.AutocompleteSuggestions))
	}
}

func TestUpDownNavigatesDropdown(t *testing.T) {
	m := createTestModel()
	m.Input = "/"
	m.updateAutocomplete()

	// Should have all commands, index at 0
	if m.AutocompleteIndex != 0 {
		t.Fatalf("Expected initial index 0, got %d", m.AutocompleteIndex)
	}

	// Down moves to index 1
	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "down"}))
	m = updatedModel.(Model)
	if m.AutocompleteIndex != 1 {
		t.Errorf("Expected index 1 after down, got %d", m.AutocompleteIndex)
	}

	// Down again
	updatedModel, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "down"}))
	m = updatedModel.(Model)
	if m.AutocompleteIndex != 2 {
		t.Errorf("Expected index 2 after down, got %d", m.AutocompleteIndex)
	}

	// Up moves back
	updatedModel, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "up"}))
	m = updatedModel.(Model)
	if m.AutocompleteIndex != 1 {
		t.Errorf("Expected index 1 after up, got %d", m.AutocompleteIndex)
	}
}

func TestUpDownClampsAtBounds(t *testing.T) {
	m := createTestModel()
	m.Input = "/"
	m.updateAutocomplete()

	// Up at index 0 stays at 0
	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "up"}))
	m = updatedModel.(Model)
	if m.AutocompleteIndex != 0 {
		t.Errorf("Expected index 0 (clamped), got %d", m.AutocompleteIndex)
	}

	// Go to last item
	lastIdx := len(m.AutocompleteSuggestions) - 1
	m.AutocompleteIndex = lastIdx

	// Down at last index stays
	updatedModel, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "down"}))
	m = updatedModel.(Model)
	if m.AutocompleteIndex != lastIdx {
		t.Errorf("Expected index %d (clamped), got %d", lastIdx, m.AutocompleteIndex)
	}
}

func TestTabCompletesSelected(t *testing.T) {
	m := createTestModel()
	m.Input = "/h"
	m.updateAutocomplete()

	// Select /history (index 1)
	m.AutocompleteIndex = 1

	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "tab"}))
	m = updatedModel.(Model)

	if m.Input != "/history" {
		t.Errorf("Expected input '/history' after tab, got %q", m.Input)
	}
	if m.autocompleteVisible() {
		t.Error("Expected dropdown hidden after tab completion")
	}
}

func TestEnterAutoCompletesAndSubmits(t *testing.T) {
	m := createTestModel()
	m.Input = "/h"
	m.updateAutocomplete()

	// First match is /help, pressing enter should complete and run it
	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	m2 := updatedModel.(Model)

	if len(m2.History) == 0 {
		t.Fatal("Expected history to have an entry")
	}
	last := m2.History[len(m2.History)-1]
	if last.Query != "/help" {
		t.Errorf("Expected query '/help' (auto-completed), got %q", last.Query)
	}
	if last.Response == "" {
		t.Error("Expected non-empty response (command should have run)")
	}
}

func TestEnterDoesNotAutoCompleteWithSpace(t *testing.T) {
	m := createTestModel()
	m.Input = "/system You are a bot."
	m.updateAutocomplete()

	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	m2 := updatedModel.(Model)

	last := m2.History[len(m2.History)-1]
	if last.Query != "/system You are a bot." {
		t.Errorf("Expected original input preserved, got %q", last.Query)
	}
}

func TestBackspaceUpdatesDropdown(t *testing.T) {
	m := createTestModel()
	m.Input = "/he"
	m.updateAutocomplete()

	// /he matches only /help
	if len(m.AutocompleteSuggestions) != 1 {
		t.Fatalf("Expected 1 suggestion for '/he', got %d", len(m.AutocompleteSuggestions))
	}

	// Backspace to "/h" — should now match /help and /history
	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "backspace"}))
	m = updatedModel.(Model)
	if len(m.AutocompleteSuggestions) != 2 {
		t.Errorf("Expected 2 suggestions after backspace to '/h', got %d", len(m.AutocompleteSuggestions))
	}
}

func TestEscClearsDropdown(t *testing.T) {
	m := createTestModel()
	m.Input = "/h"
	m.updateAutocomplete()

	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "esc"}))
	m = updatedModel.(Model)

	if m.Input != "" {
		t.Errorf("Expected input cleared, got %q", m.Input)
	}
	if m.autocompleteVisible() {
		t.Error("Expected dropdown hidden after esc")
	}
}

func TestUpDownHistoryWhenDropdownHidden(t *testing.T) {
	m := createTestModel()
	m.History = append(m.History, ChatMessage{Query: "what stocks?", Response: "buy AAPL"})
	m.HistoryIndex = len(m.History)

	// Input doesn't start with /, so dropdown is hidden
	m.Input = ""

	// Up should cycle history, not crash
	updatedModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "up"}))
	m = updatedModel.(Model)
	if m.Input != "what stocks?" {
		t.Errorf("Expected history query, got %q", m.Input)
	}
}

func TestRenderAutocomplete(t *testing.T) {
	m := createTestModel()
	m.Input = "/h"
	m.updateAutocomplete()

	lines := m.renderAutocomplete(80)
	if len(lines) == 0 {
		t.Fatal("Expected non-empty render output")
	}

	// Should have top border + 2 items + bottom border = 4 lines
	if len(lines) != 4 {
		t.Fatalf("Expected 4 lines (border + 2 items + border), got %d", len(lines))
	}

	// First line should be a top border
	if !containsStr(lines[0], "┌") {
		t.Error("Expected top border in first line")
	}

	// Last line should be a bottom border
	if !containsStr(lines[len(lines)-1], "└") {
		t.Error("Expected bottom border in last line")
	}
}

func suggestionNames(suggestions []commandSuggestion) []string {
	names := make([]string, len(suggestions))
	for i, s := range suggestions {
		names[i] = s.Name
	}
	return names
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOfStr(s, substr) >= 0)
}

func indexOfStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
