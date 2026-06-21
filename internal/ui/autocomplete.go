package ui

import (
	"fmt"
	"strings"
)

type commandSuggestion struct {
	Name        string
	Description string
}

var slashCommands = []commandSuggestion{
	{"/help", "Show this help message"},
	{"/clear", "Clear chat history"},
	{"/history", "Get or set the maximum history size"},
	{"/url", "Get or set the LLM base URL"},
	{"/provider", "Add a new LLM provider"},
	{"/connect", "Switch to a stored provider"},
	{"/system", "Get or set the system prompt"},
	{"/copy", "Copy last response or query to clipboard"},
	{"/quit", "Quit the application"},
	{"/exit", "Quit the application"},
}

func matchingCommands(input string) []commandSuggestion {
	if !strings.HasPrefix(input, "/") {
		return nil
	}
	var matches []commandSuggestion
	for _, cmd := range slashCommands {
		if strings.HasPrefix(cmd.Name, input) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// updateAutocomplete recomputes the suggestion list based on the current
// input. The dropdown is shown only when the input is a partial command
// (starts with "/" and contains no spaces).
func (m *Model) updateAutocomplete() {
	if strings.HasPrefix(m.Input, "/") && !strings.Contains(m.Input, " ") {
		m.AutocompleteSuggestions = matchingCommands(m.Input)
	} else {
		m.AutocompleteSuggestions = nil
	}
	if m.AutocompleteIndex >= len(m.AutocompleteSuggestions) {
		m.AutocompleteIndex = 0
	}
}

func (m Model) autocompleteVisible() bool {
	return len(m.AutocompleteSuggestions) > 0
}

// renderAutocomplete returns the lines for the command dropdown overlay.
func (m Model) renderAutocomplete(width int) []string {
	suggestions := m.AutocompleteSuggestions
	if len(suggestions) == 0 {
		return nil
	}

	maxVisible := 8
	startIdx := 0
	if m.AutocompleteIndex >= maxVisible {
		startIdx = m.AutocompleteIndex - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(suggestions) {
		endIdx = len(suggestions)
	}
	visible := suggestions[startIdx:endIdx]

	var lines []string

	title := " Commands "
	titleLen := visualLength(title)
	borderLen := width - 2 - titleLen
	if borderLen < 0 {
		borderLen = 0
	}
	left := borderLen / 2
	right := borderLen - left
	lines = append(lines, colorGrey+"┌"+strings.Repeat("─", left)+colorReset+
		colorCyan+title+colorReset+
		colorGrey+strings.Repeat("─", right)+"┐"+colorReset)

	for i, s := range visible {
		actualIdx := startIdx + i
		marker := "  "
		if actualIdx == m.AutocompleteIndex {
			marker = colorOrange + "▸ " + colorReset
		}
		content := fmt.Sprintf("  %s%-14s %s", marker, s.Name, colorGrey+s.Description+colorReset)
		lines = append(lines, padModalLine(content, width))
	}

	lines = append(lines, modalBottomLine(width))

	return lines
}
