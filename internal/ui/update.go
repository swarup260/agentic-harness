package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/llm"
)

func waitForMsg(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m Model) submitQuery() tea.Cmd {
	m.LlmClient.SubmitQuery(m.Ctx, m.SystemPrompt, m.CurrentQuery, m.MsgChan)
	return waitForMsg(m.MsgChan)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.PasteMsg:
		m.Input += msg.Content
		return m, nil

	case tea.ClipboardMsg:
		m.Input += msg.Content
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+v":
			return m, tea.ReadClipboard

		case "ctrl+x":
			if m.Input != "" {
				textToCopy := m.Input
				m.Input = ""
				return m, tea.SetClipboard(textToCopy)
			}
			return m, nil

		case "alt+c":
			if m.Input != "" {
				return m, tea.SetClipboard(m.Input)
			}
			return m, nil

		case "ctrl+up":
			m.MaxHistorySize++
			m.HistoryIndex = len(m.History)
			return m, nil

		case "ctrl+down":
			if m.MaxHistorySize > 1 {
				m.MaxHistorySize--
				if len(m.History) > m.MaxHistorySize {
					m.History = m.History[len(m.History)-m.MaxHistorySize:]
				}
				m.HistoryIndex = len(m.History)
			}
			return m, nil

		case "up":
			if len(m.History) > 0 {
				if m.HistoryIndex == len(m.History) {
					m.SavedInput = m.Input
				}
				if m.HistoryIndex > 0 {
					m.HistoryIndex--
					m.Input = m.History[m.HistoryIndex].Query
				}
			}
			return m, nil

		case "pgup", "pageup", "shift+up", "ctrl+u", "alt+up":
			if m.AutoScroll {
				m.ScrollOffset = m.getScrollBottom()
				m.AutoScroll = false
			}
			m.ScrollOffset = m.ScrollOffset - 3
			if m.ScrollOffset < 0 {
				m.ScrollOffset = 0
			}
			return m, nil

		case "down":
			if len(m.History) > 0 {
				if m.HistoryIndex < len(m.History) {
					m.HistoryIndex++
					if m.HistoryIndex == len(m.History) {
						m.Input = m.SavedInput
					} else {
						m.Input = m.History[m.HistoryIndex].Query
					}
				}
			}
			return m, nil

		case "pgdn", "pgdown", "pagedown", "shift+down", "ctrl+d", "alt+down":
			if m.AutoScroll {
				return m, nil
			}
			m.ScrollOffset = m.ScrollOffset + 3
			maxScroll := m.getScrollBottom()
			if m.ScrollOffset >= maxScroll {
				m.ScrollOffset = maxScroll
				m.AutoScroll = true
			}
			return m, nil

		case "enter":
			trimmed := strings.TrimSpace(m.Input)
			if trimmed != "" {
				if strings.HasPrefix(trimmed, "/") {
					return m.handleSlashCommand(trimmed)
				}

				if m.Loading {
					m.QueryQueue = append(m.QueryQueue, trimmed)
					m.Input = ""
					m.SavedInput = ""
					m.HistoryIndex = len(m.History)
					m.AutoScroll = true
					return m, nil
				}

				m.Loading = true
				m.AutoScroll = true
				m.StartTime = time.Now()
				m.CurrentQuery = m.Input
				m.Input = ""
				m.RawStreamBuffer = ""
				m.Ttft = 0
				m.TotalTime = 0
				m.PromptTokens = 0
				m.CompletionTokens = 0
				m.MsgChan = make(chan tea.Msg, 500)
				m.SavedInput = ""

				cmd := m.submitQuery()
				return m, cmd
			}
			return m, nil

		case "backspace":
			if len(m.Input) > 0 {
				runes := []rune(m.Input)
				m.Input = string(runes[:len(runes)-1])
			}
			return m, nil

		case "esc":
			m.Input = ""
			return m, nil

		default:
			if msg.Text != "" {
				m.Input += msg.Text
			} else if msg.String() == "space" {
				m.Input += " "
			}
			return m, nil
		}

	case tea.MouseWheelMsg:
		mEvent := msg.Mouse()
		switch mEvent.Button {
		case tea.MouseWheelUp:
			if m.AutoScroll {
				m.ScrollOffset = m.getScrollBottom()
				m.AutoScroll = false
			}
			m.ScrollOffset = m.ScrollOffset - 1
			if m.ScrollOffset < 0 {
				m.ScrollOffset = 0
			}
		case tea.MouseWheelDown:
			if m.AutoScroll {
				return m, nil
			}
			m.ScrollOffset = m.ScrollOffset + 1
			maxScroll := m.getScrollBottom()
			if m.ScrollOffset >= maxScroll {
				m.ScrollOffset = maxScroll
				m.AutoScroll = true
			}
		}
		return m, nil

	case llm.TokenMsg:
		m.RawStreamBuffer += msg.Token
		return m, waitForMsg(m.MsgChan)

	case llm.UsageMsg:
		m.PromptTokens = msg.PromptTokens
		m.CompletionTokens = msg.CompletionTokens
		m.Ttft = msg.TTFT
		m.TotalTime = msg.TotalTime
		return m, waitForMsg(m.MsgChan)

	case llm.ErrMsg:
		m.Loading = false
		thought, resp, _ := llm.ParseStream(m.RawStreamBuffer)
		m.History = append(m.History, ChatMessage{
			Query:            m.CurrentQuery,
			Response:         resp + fmt.Sprintf("\n[Error: %v]", msg.Err),
			Thought:          thought,
			TTFT:             m.Ttft,
			TotalTime:        time.Since(m.StartTime),
			PromptTokens:     m.PromptTokens,
			CompletionTokens: m.CompletionTokens,
		})
		if len(m.History) > m.MaxHistorySize {
			m.History = m.History[len(m.History)-m.MaxHistorySize:]
		}
		m.HistoryIndex = len(m.History)

		if len(m.QueryQueue) > 0 {
			nextQuery := m.QueryQueue[0]
			m.QueryQueue = m.QueryQueue[1:]

			m.Loading = true
			m.AutoScroll = true
			m.StartTime = time.Now()
			m.CurrentQuery = nextQuery
			m.RawStreamBuffer = ""
			m.Ttft = 0
			m.TotalTime = 0
			m.PromptTokens = 0
			m.CompletionTokens = 0
			m.MsgChan = make(chan tea.Msg, 500)

			cmd := m.submitQuery()
			return m, cmd
		}
		return m, nil

	case llm.DoneMsg:
		m.Loading = false
		thought, resp, _ := llm.ParseStream(m.RawStreamBuffer)
		m.History = append(m.History, ChatMessage{
			Query:            m.CurrentQuery,
			Response:         resp,
			Thought:          thought,
			TTFT:             m.Ttft,
			TotalTime:        time.Since(m.StartTime),
			PromptTokens:     m.PromptTokens,
			CompletionTokens: m.CompletionTokens,
		})
		if len(m.History) > m.MaxHistorySize {
			m.History = m.History[len(m.History)-m.MaxHistorySize:]
		}
		m.HistoryIndex = len(m.History)

		if len(m.QueryQueue) > 0 {
			nextQuery := m.QueryQueue[0]
			m.QueryQueue = m.QueryQueue[1:]

			m.Loading = true
			m.AutoScroll = true
			m.StartTime = time.Now()
			m.CurrentQuery = nextQuery
			m.RawStreamBuffer = ""
			m.Ttft = 0
			m.TotalTime = 0
			m.PromptTokens = 0
			m.CompletionTokens = 0
			m.MsgChan = make(chan tea.Msg, 500)

			cmd := m.submitQuery()
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}
