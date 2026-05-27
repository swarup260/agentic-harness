package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/llm"
)

// handleSlashCommand parses and executes a slash command.
func (m Model) handleSlashCommand(trimmed string) (Model, tea.Cmd) {
	// Add command to history
	m.History = append(m.History, ChatMessage{
		Query: m.Input,
	})

	parts := strings.Fields(trimmed)
	cmdName := parts[0]
	var responseText string

	switch cmdName {
	case "/help":
		responseText = "Available commands:\n" +
			"• `/help` - Show this help message\n" +
			"• `/clear` - Clear chat history\n" +
			"• `/history [limit]` - Get or set the maximum history size\n" +
			"• `/url [new_url]` - Get or set the LLM base URL\n" +
			"• `/system [prompt]` - Get or set the system prompt\n" +
			"• `/copy [response/query]` - Copy last response or query to clipboard\n" +
			"• `/quit` - Quit the application\n\n" +
			"Keyboard shortcuts:\n" +
			"• `ctrl+c` - Quit\n" +
			"• `ctrl+v` - Paste clipboard to prompt\n" +
			"• `ctrl+x` - Cut prompt to clipboard\n" +
			"• `alt+c` - Copy prompt to clipboard\n" +
			"• `up` / `down` - Cycle through query history\n" +
			"• `ctrl+up` / `ctrl+down` - Increase/decrease history buffer limit\n" +
			"• `pgup` / `pgdn` (or `ctrl+u`/`ctrl+d`) - Scroll view"

	case "/quit", "/exit":
		return m, tea.Quit

	case "/clear":
		m.History = nil
		m.Input = ""
		m.SavedInput = ""
		m.HistoryIndex = 0
		m.AutoScroll = true
		return m, nil

	case "/history":
		if len(parts) < 2 {
			responseText = fmt.Sprintf("Current history limit is %d.", m.MaxHistorySize)
		} else {
			val, err := strconv.Atoi(parts[1])
			if err != nil || val <= 0 {
				responseText = "Invalid history limit. Please specify a positive integer."
			} else {
				m.MaxHistorySize = val
				if len(m.History) > m.MaxHistorySize {
					m.History = m.History[len(m.History)-m.MaxHistorySize:]
				}
				responseText = fmt.Sprintf("History limit set to %d.", val)
			}
		}

	case "/url":
		if len(parts) < 2 {
			responseText = fmt.Sprintf("Current LLM base URL is %s.", m.LlmClient.URL())
		} else {
			newURL := parts[1]
			m.LlmClient = llm.NewClient(newURL)
			responseText = fmt.Sprintf("LLM base URL updated to %s.", newURL)
		}

	case "/system":
		if len(parts) < 2 {
			responseText = fmt.Sprintf("Current system prompt:\n%s", m.SystemPrompt)
		} else {
			newPrompt := strings.TrimSpace(strings.TrimPrefix(trimmed, "/system"))
			m.SystemPrompt = newPrompt
			responseText = fmt.Sprintf("System prompt updated to:\n%s", newPrompt)
		}

	case "/copy":
		var textToCopy string
		var copyType = "response"

		if len(parts) >= 2 {
			arg := strings.ToLower(parts[1])
			if arg == "query" || arg == "prompt" {
				copyType = "query"
			} else if arg == "response" || arg == "answer" {
				copyType = "response"
			} else {
				responseText = "Invalid argument. Use `/copy response` or `/copy query`."
				if len(m.History) > 0 {
					m.History[len(m.History)-1].Response = responseText
				}
				m.Input = ""
				m.SavedInput = ""
				m.HistoryIndex = len(m.History)
				m.AutoScroll = true
				return m, nil
			}
		}

		if copyType == "query" {
			for i := len(m.History) - 2; i >= 0; i-- {
				if !strings.HasPrefix(m.History[i].Query, "/") {
					textToCopy = m.History[i].Query
					break
				}
			}
			if textToCopy == "" {
				for i := len(m.History) - 2; i >= 0; i-- {
					textToCopy = m.History[i].Query
					break
				}
			}
		} else {
			for i := len(m.History) - 2; i >= 0; i-- {
				if !strings.HasPrefix(m.History[i].Query, "/") && m.History[i].Response != "" {
					textToCopy = m.History[i].Response
					break
				}
			}
			if textToCopy == "" {
				for i := len(m.History) - 2; i >= 0; i-- {
					if m.History[i].Response != "" {
						textToCopy = m.History[i].Response
						break
					}
				}
			}
		}

		var clipboardCmd tea.Cmd
		if textToCopy != "" {
			if copyType == "query" {
				responseText = "Last user query copied to clipboard!"
			} else {
				responseText = "Last assistant response copied to clipboard!"
			}
			clipboardCmd = tea.SetClipboard(textToCopy)
		} else {
			if copyType == "query" {
				responseText = "No query history to copy."
			} else {
				responseText = "No response history to copy."
			}
		}

		if len(m.History) > 0 {
			m.History[len(m.History)-1].Response = responseText
		}

		m.Input = ""
		m.SavedInput = ""
		if len(m.History) > m.MaxHistorySize {
			m.History = m.History[len(m.History)-m.MaxHistorySize:]
		}
		m.HistoryIndex = len(m.History)
		m.AutoScroll = true
		return m, clipboardCmd

	default:
		responseText = fmt.Sprintf("Unknown command: %s. Type `/help` for available commands.", cmdName)
	}

	if len(m.History) > 0 && m.History[len(m.History)-1].Query == m.Input {
		m.History[len(m.History)-1].Response = responseText
	}

	m.Input = ""
	m.SavedInput = ""
	if len(m.History) > m.MaxHistorySize {
		m.History = m.History[len(m.History)-m.MaxHistorySize:]
	}
	m.HistoryIndex = len(m.History)
	m.AutoScroll = true
	return m, nil
}
