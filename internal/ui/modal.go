package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/config"
	"github.com/swarup260/agent-harness-loop/internal/llm"
)

// handleModalKey dispatches keypresses to the active modal handler.
func (m Model) handleModalKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch m.ModalMode {
	case "provider_form":
		return m.handleProviderFormKey(msg)
	case "connect_list":
		return m.handleConnectListKey(msg)
	}
	return m, nil
}

func (m Model) handleProviderFormKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeModal()
		return m, nil

	case "enter":
		if m.FormField < 2 {
			m.FormField++
			return m, nil
		}
		m.saveProvider()
		m.closeModal()
		return m, nil

	case "ctrl+v":
		return m, tea.ReadClipboard

	case "ctrl+x":
		if m.activeFieldValue() != "" {
			text := m.activeFieldValue()
			m.setActiveFieldValue("")
			return m, tea.SetClipboard(text)
		}
		return m, nil

	case "alt+c":
		if m.activeFieldValue() != "" {
			return m, tea.SetClipboard(m.activeFieldValue())
		}
		return m, nil

	case "up":
		if m.FormField > 0 {
			m.FormField--
		}
		return m, nil

	case "down":
		if m.FormField < 2 {
			m.FormField++
		}
		return m, nil

	case "backspace":
		m.setActiveFieldValue(removeLastRune(m.activeFieldValue()))
		return m, nil

	default:
		ch := ""
		if msg.Text != "" {
			ch = msg.Text
		} else if msg.String() == "space" {
			ch = " "
		}
		if ch == "" {
			return m, nil
		}
		m.setActiveFieldValue(m.activeFieldValue() + ch)
		return m, nil
	}
}

func (m Model) activeFieldValue() string {
	switch m.FormField {
	case 0:
		return m.FormName
	case 1:
		return m.FormBaseURL
	case 2:
		return m.FormAPIKey
	}
	return ""
}

func (m *Model) setActiveFieldValue(v string) {
	switch m.FormField {
	case 0:
		m.FormName = v
	case 1:
		m.FormBaseURL = v
	case 2:
		m.FormAPIKey = v
	}
}

func (m Model) handleConnectListKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeModal()
		return m, nil

	case "up":
		if m.ConnectIndex > 0 {
			m.ConnectIndex--
		}
		return m, nil

	case "down":
		if m.ConnectIndex < len(m.ConnectNames)-1 {
			m.ConnectIndex++
		}
		return m, nil

	case "enter":
		if len(m.ConnectNames) > 0 && m.ConnectIndex < len(m.ConnectNames) {
			name := m.ConnectNames[m.ConnectIndex]
			m.Config.ActiveProvider = name
			p := m.Config.Providers[name]
			m.LlmClient = llm.NewClient(p.BaseURL, p.APIKey, p.Model, m.Config.Seed, m.Config.Temperature)
		}
		m.closeModal()
		return m, nil
	}
	return m, nil
}

func (m *Model) closeModal() {
	m.ModalMode = ""
	m.FormName = ""
	m.FormBaseURL = ""
	m.FormAPIKey = ""
	m.FormField = 0
	m.ConnectIndex = 0
	m.ConnectNames = nil
}

func (m *Model) saveProvider() {
	if m.FormName == "" {
		return
	}
	if m.Config.Providers == nil {
		m.Config.Providers = map[string]config.ProviderConfig{}
	}
	m.Config.Providers[m.FormName] = config.ProviderConfig{
		BaseURL: m.FormBaseURL,
		APIKey:  m.FormAPIKey,
	}
	m.Config.Save(m.ConfigPath)
}

func removeLastRune(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}

// renderModal returns the lines for the active modal overlay.
func (m Model) renderModal(width int) []string {
	switch m.ModalMode {
	case "provider_form":
		return m.renderProviderForm(width)
	case "connect_list":
		return m.renderConnectList(width)
	}
	return nil
}

func (m Model) renderProviderForm(width int) []string {
	var lines []string

	lines = append(lines, modalTitleLine(" Add Provider ", width))
	lines = append(lines, padModalLine("", width))

	fields := []struct {
		label string
		value string
	}{
		{"Name:", m.FormName},
		{"Base URL:", m.FormBaseURL},
		{"API Key:", m.FormAPIKey},
	}

	for i, f := range fields {
		cursor := ""
		if i == m.FormField {
			cursor = "█"
		}
		content := fmt.Sprintf("  %s %s%s", f.label, f.value, cursor)
		lines = append(lines, padModalLine(content, width))
	}

	lines = append(lines, padModalLine("", width))
	lines = append(lines, padModalLine(colorGrey+"  enter = next/save  ·  esc = cancel"+colorReset, width))
	lines = append(lines, modalBottomLine(width))

	return lines
}

func (m Model) renderConnectList(width int) []string {
	var lines []string

	lines = append(lines, modalTitleLine(" Connect to Provider ", width))
	lines = append(lines, padModalLine("", width))

	for i, name := range m.ConnectNames {
		marker := "  "
		if i == m.ConnectIndex {
			marker = colorOrange + "▸ " + colorReset
		}
		suffix := ""
		if name == m.Config.ActiveProvider {
			suffix = colorGrey + " (active)" + colorReset
		}
		content := fmt.Sprintf("  %s%s%s", marker, name, suffix)
		lines = append(lines, padModalLine(content, width))
	}

	if len(m.ConnectNames) == 0 {
		lines = append(lines, padModalLine(colorGrey+"  No providers configured."+colorReset, width))
	}

	lines = append(lines, padModalLine("", width))
	lines = append(lines, padModalLine(colorGrey+"  enter = connect  ·  esc = cancel"+colorReset, width))
	lines = append(lines, modalBottomLine(width))

	return lines
}

func modalTitleLine(title string, width int) string {
	titleLen := visualLength(title)
	borderLen := width - 2 - titleLen
	if borderLen < 0 {
		borderLen = 0
	}
	left := borderLen / 2
	right := borderLen - left
	return colorGrey + "┌" + strings.Repeat("─", left) + colorReset +
		colorCyan + title + colorReset +
		colorGrey + strings.Repeat("─", right) + "┐" + colorReset
}

func modalBottomLine(width int) string {
	return colorGrey + "└" + strings.Repeat("─", width-2) + "┘" + colorReset
}

func padModalLine(content string, width int) string {
	innerWidth := width - 2
	contentLen := visualLength(content)
	padding := innerWidth - contentLen
	if padding < 0 {
		padding = 0
	}
	return colorGrey + "│" + colorReset + content + strings.Repeat(" ", padding) + colorGrey + "│" + colorReset
}
