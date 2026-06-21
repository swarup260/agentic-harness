package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/openai/openai-go/v3"
	"github.com/swarup260/agent-harness-loop/internal/llm"
)

type toolResult struct {
	CallID string
	Name   string
	Result string
	Err    error
}

type toolResultMsg struct {
	Results []toolResult
}

func waitForMsg(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m Model) submitQuery() tea.Cmd {
	m.LlmClient.SubmitQuery(m.Ctx, m.Messages, m.OpenAITools, m.MsgChan)
	return waitForMsg(m.MsgChan)
}

// buildMessages assembles the conversation history (system + prior turns +
// current query) as OpenAI message params. Slash-command entries in history
// are skipped since they are not part of the LLM conversation.
func (m Model) buildMessages() []openai.ChatCompletionMessageParamUnion {
	msgs := []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(m.SystemPrompt)}

	start := 0
	if len(m.History) > m.MaxHistorySize {
		start = len(m.History) - m.MaxHistorySize
	}
	for i := start; i < len(m.History); i++ {
		h := m.History[i]
		if strings.HasPrefix(h.Query, "/") {
			continue
		}
		msgs = append(msgs, openai.UserMessage(h.Query))
		if h.Response != "" {
			msgs = append(msgs, openai.AssistantMessage(h.Response))
		}
	}

	msgs = append(msgs, openai.UserMessage(m.CurrentQuery))
	return msgs
}

// startQuery resets state for a new query, builds the message list and tools,
// and kicks off the streaming request.
func (m Model) startQuery(query string) (Model, tea.Cmd) {
	m.Loading = true
	m.AutoScroll = true
	m.StartTime = time.Now()
	m.CurrentQuery = query
	m.Input = ""
	m.SavedInput = ""
	m.RawStreamBuffer = ""
	m.ToolEvents = nil
	m.ToolCallCount = 0
	m.Ttft = 0
	m.TotalTime = 0
	m.PromptTokens = 0
	m.CompletionTokens = 0
	m.MsgChan = make(chan tea.Msg, 500)
	m.Messages = m.buildMessages()
	m.OpenAITools = m.Registry.ToOpenAITools()
	cmd := m.submitQuery()
	return m, cmd
}

// executeTools runs each requested tool call and returns a toolResultMsg.
func (m Model) executeTools(calls []llm.ToolCall) tea.Cmd {
	registry := m.Registry
	ctx := m.Ctx
	return func() tea.Msg {
		results := make([]toolResult, len(calls))
		for i, call := range calls {
			tool, ok := registry.Get(call.Name)
			if !ok {
				results[i] = toolResult{CallID: call.ID, Name: call.Name, Err: fmt.Errorf("unknown tool: %s", call.Name)}
				continue
			}
			result, err := tool.Execute(ctx, call.Arguments)
			results[i] = toolResult{CallID: call.ID, Name: call.Name, Result: result, Err: err}
		}
		return toolResultMsg{Results: results}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.PasteMsg:
		if m.ModalMode == "provider_form" {
			m.setActiveFieldValue(m.activeFieldValue() + msg.Content)
		} else {
			m.Input += msg.Content
			m.updateAutocomplete()
		}
		return m, nil

	case tea.ClipboardMsg:
		if m.ModalMode == "provider_form" {
			m.setActiveFieldValue(m.activeFieldValue() + msg.Content)
		} else {
			m.Input += msg.Content
			m.updateAutocomplete()
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.ModalMode != "" {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m.handleModalKey(msg)
		}

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
			if m.autocompleteVisible() {
				if m.AutocompleteIndex > 0 {
					m.AutocompleteIndex--
				}
				return m, nil
			}
			if len(m.History) > 0 {
				if m.HistoryIndex == len(m.History) {
					m.SavedInput = m.Input
				}
				if m.HistoryIndex > 0 {
					m.HistoryIndex--
					m.Input = m.History[m.HistoryIndex].Query
				}
			}
			m.updateAutocomplete()
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
			if m.autocompleteVisible() {
				if m.AutocompleteIndex < len(m.AutocompleteSuggestions)-1 {
					m.AutocompleteIndex++
				}
				return m, nil
			}
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
			m.updateAutocomplete()
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
			if m.autocompleteVisible() && len(m.Input) > 1 && !strings.Contains(m.Input, " ") {
				m.Input = m.AutocompleteSuggestions[m.AutocompleteIndex].Name
				m.AutocompleteSuggestions = nil
			}
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

				return m.startQuery(m.Input)
			}
			return m, nil

		case "tab":
			if m.autocompleteVisible() {
				m.Input = m.AutocompleteSuggestions[m.AutocompleteIndex].Name
				m.AutocompleteSuggestions = nil
			}
			return m, nil

		case "backspace":
			if len(m.Input) > 0 {
				runes := []rune(m.Input)
				m.Input = string(runes[:len(runes)-1])
			}
			m.updateAutocomplete()
			return m, nil

		case "esc":
			m.Input = ""
			m.AutocompleteSuggestions = nil
			return m, nil

		default:
			if msg.Text != "" {
				m.Input += msg.Text
			} else if msg.String() == "space" {
				m.Input += " "
			}
			m.updateAutocomplete()
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

	case llm.ToolCallMsg:
		asst := openai.ChatCompletionAssistantMessageParam{}
		if msg.Content != "" {
			asst.Content.OfString = openai.String(msg.Content)
		}
		for _, call := range msg.Calls {
			asst.ToolCalls = append(asst.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: call.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Arguments: call.Arguments,
						Name:      call.Name,
					},
				},
			})
			m.ToolEvents = append(m.ToolEvents, fmt.Sprintf("🔧 Calling %s(%s)", call.Name, call.Arguments))
		}
		m.Messages = append(m.Messages, openai.ChatCompletionMessageParamUnion{OfAssistant: &asst})

		m.ToolCallCount++
		if m.ToolCallCount > maxToolRounds {
			m.Loading = false
			thought, resp, _ := llm.ParseStream(m.RawStreamBuffer)
			m.History = append(m.History, ChatMessage{
				Query:            m.CurrentQuery,
				Response:         resp + "\n[Tool call limit reached]",
				Thought:          thought,
				ToolEvents:       m.ToolEvents,
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
				return m.startQuery(nextQuery)
			}
			return m, nil
		}

		return m, m.executeTools(msg.Calls)

	case toolResultMsg:
		for _, r := range msg.Results {
			content := r.Result
			if r.Err != nil {
				content = fmt.Sprintf("Error: %v", r.Err)
			}
			m.Messages = append(m.Messages, openai.ToolMessage(content, r.CallID))
		}
		m.RawStreamBuffer = ""
		m.MsgChan = make(chan tea.Msg, 500)
		cmd := m.submitQuery()
		return m, cmd

	case llm.ErrMsg:
		m.Loading = false
		thought, resp, _ := llm.ParseStream(m.RawStreamBuffer)
		m.History = append(m.History, ChatMessage{
			Query:            m.CurrentQuery,
			Response:         resp + fmt.Sprintf("\n[Error: %v]", msg.Err),
			Thought:          thought,
			ToolEvents:       m.ToolEvents,
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
			return m.startQuery(nextQuery)
		}
		return m, nil

	case llm.DoneMsg:
		m.Loading = false
		thought, resp, _ := llm.ParseStream(m.RawStreamBuffer)
		m.History = append(m.History, ChatMessage{
			Query:            m.CurrentQuery,
			Response:         resp,
			Thought:          thought,
			ToolEvents:       m.ToolEvents,
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
			return m.startQuery(nextQuery)
		}
		return m, nil
	}

	return m, nil
}
