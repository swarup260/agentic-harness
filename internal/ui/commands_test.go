package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/config"
	"github.com/swarup260/agent-harness-loop/internal/llm"
)

func createTestModel() Model {
	cfg := config.DefaultConfig()
	llmClient := llm.NewClient(cfg.LLMURL)
	return NewModel(cfg, llmClient)
}

func TestCommands(t *testing.T) {
	// 1. Test /help command
	m := createTestModel()
	m.Input = "/help"

	// Create enter keypress
	msg := tea.KeyPressMsg(tea.Key{Text: "enter"})
	updatedModel, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("Expected cmd to be nil for local commands")
	}

	m2 := updatedModel.(Model)
	if len(m2.History) != 1 {
		t.Fatalf("Expected history length to be 1, got %d", len(m2.History))
	}

	if m2.History[0].Query != "/help" {
		t.Errorf("Expected query to be '/help', got %q", m2.History[0].Query)
	}

	if !strings.Contains(m2.History[0].Response, "Available commands") {
		t.Errorf("Expected response to contain help message, got %q", m2.History[0].Response)
	}

	// 2. Test /system command (get & set)
	m = createTestModel()
	m.Input = "/system"
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[0].Response, "Current system prompt") {
		t.Errorf("Expected response to contain current system prompt, got %q", m2.History[0].Response)
	}

	m = createTestModel()
	m.Input = "/system You are a helpful bot."
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if m2.SystemPrompt != "You are a helpful bot." {
		t.Errorf("Expected systemPrompt to be updated, got %q", m2.SystemPrompt)
	}
	if !strings.Contains(m2.History[0].Response, "You are a helpful bot.") {
		t.Errorf("Expected response to confirm system prompt update, got %q", m2.History[0].Response)
	}

	// 3. Test /url command (get & set)
	m = createTestModel()
	m.Input = "/url"
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[0].Response, "Current LLM base URL") {
		t.Errorf("Expected response to contain current LLM URL, got %q", m2.History[0].Response)
	}

	m = createTestModel()
	m.Input = "/url http://localhost:9090"
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if m2.LlmClient.URL() != "http://localhost:9090" {
		t.Errorf("Expected llmURL to be updated, got %q", m2.LlmClient.URL())
	}
	if !strings.Contains(m2.History[0].Response, "updated to http://localhost:9090") {
		t.Errorf("Expected response to confirm url update, got %q", m2.History[0].Response)
	}

	// 4. Test /history command (get & set)
	m = createTestModel()
	m.Input = "/history"
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[0].Response, "Current history limit is") {
		t.Errorf("Expected response to contain current history limit, got %q", m2.History[0].Response)
	}

	m = createTestModel()
	m.Input = "/history 10"
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if m2.MaxHistorySize != 10 {
		t.Errorf("Expected maxHistorySize to be 10, got %d", m2.MaxHistorySize)
	}

	m = createTestModel()
	m.Input = "/history invalid"
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[0].Response, "Invalid history limit") {
		t.Errorf("Expected warning for invalid history limit, got %q", m2.History[0].Response)
	}

	// 5. Test /clear command
	m = createTestModel()
	m.History = append(m.History, ChatMessage{Query: "hello", Response: "world"})
	m.Input = "/clear"
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if len(m2.History) != 0 {
		t.Errorf("Expected history to be cleared, got %d messages", len(m2.History))
	}

	// 7. Test /copy command
	m = createTestModel()
	m.Input = "/copy"
	updatedModel, clipboardCmd := m.Update(msg)
	if clipboardCmd != nil {
		t.Error("Expected nil command when copying empty history")
	}
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[0].Response, "No response history to copy") {
		t.Errorf("Expected empty history warning, got %q", m2.History[0].Response)
	}

	m = createTestModel()
	m.History = append(m.History, ChatMessage{Query: "test query", Response: "test response"})
	m.Input = "/copy"
	updatedModel, clipboardCmd = m.Update(msg)
	if clipboardCmd == nil {
		t.Error("Expected non-nil command for copying valid response")
	}
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[len(m2.History)-1].Response, "Last assistant response copied to clipboard") {
		t.Errorf("Expected copy confirmation, got %q", m2.History[len(m2.History)-1].Response)
	}

	// Test /copy query
	m = createTestModel()
	m.History = append(m.History, ChatMessage{Query: "test query", Response: "test response"})
	m.Input = "/copy query"
	updatedModel, clipboardCmd = m.Update(msg)
	if clipboardCmd == nil {
		t.Error("Expected non-nil command for copying valid query")
	}
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[len(m2.History)-1].Response, "Last user query copied to clipboard") {
		t.Errorf("Expected query copy confirmation, got %q", m2.History[len(m2.History)-1].Response)
	}

	// Test /copy invalid
	m = createTestModel()
	m.Input = "/copy foo"
	updatedModel, clipboardCmd = m.Update(msg)
	if clipboardCmd != nil {
		t.Error("Expected nil command for invalid copy arg")
	}
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[0].Response, "Invalid argument") {
		t.Errorf("Expected invalid argument error, got %q", m2.History[0].Response)
	}

	// 8. Test /quit and /exit commands
	m = createTestModel()
	m.Input = "/quit"
	_, quitCmd := m.Update(msg)
	if quitCmd == nil {
		t.Error("Expected /quit to return a non-nil command")
	}

	m = createTestModel()
	m.Input = "/exit"
	_, exitCmd := m.Update(msg)
	if exitCmd == nil {
		t.Error("Expected /exit to return a non-nil command")
	}

	// 9. Test unknown command
	m = createTestModel()
	m.Input = "/foo"
	updatedModel, _ = m.Update(msg)
	m2 = updatedModel.(Model)
	if !strings.Contains(m2.History[0].Response, "Unknown command") {
		t.Errorf("Expected response to contain unknown command error, got %q", m2.History[0].Response)
	}
}

func TestPaste(t *testing.T) {
	// Test tea.PasteMsg
	m := createTestModel()
	msg := tea.PasteMsg{Content: "pasted query"}
	updatedModel, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("Expected cmd to be nil for paste message")
	}
	m2 := updatedModel.(Model)
	if m2.Input != "pasted query" {
		t.Errorf("Expected input to be 'pasted query', got %q", m2.Input)
	}

	// Test multi-character tea.KeyPressMsg using msg.Text
	m = createTestModel()
	keyMsg := tea.KeyPressMsg(tea.Key{Text: "pasted multi-character"})
	updatedModel, cmd = m.Update(keyMsg)
	if cmd != nil {
		t.Error("Expected cmd to be nil for keypress message")
	}
	m2 = updatedModel.(Model)
	if m2.Input != "pasted multi-character" {
		t.Errorf("Expected input to be 'pasted multi-character', got %q", m2.Input)
	}
}

func TestClipboardKeybindings(t *testing.T) {
	// Test ctrl+v (paste) triggers ReadClipboard cmd
	m := createTestModel()
	keyMsg := tea.KeyPressMsg(tea.Key{Text: "ctrl+v"})
	updatedModel, cmd := m.Update(keyMsg)
	if cmd == nil {
		t.Error("Expected ctrl+v to return a command")
	}

	// Test tea.ClipboardMsg appends text to input
	m = createTestModel()
	m.Input = "existing "
	clipMsg := tea.ClipboardMsg{Content: "pasted content"}
	updatedModel, cmd = m.Update(clipMsg)
	if cmd != nil {
		t.Error("Expected cmd to be nil for ClipboardMsg")
	}
	m2 := updatedModel.(Model)
	if m2.Input != "existing pasted content" {
		t.Errorf("Expected input to be 'existing pasted content', got %q", m2.Input)
	}

	// Test ctrl+x (cut) clears input and sets clipboard
	m = createTestModel()
	m.Input = "cut me"
	keyMsg = tea.KeyPressMsg(tea.Key{Text: "ctrl+x"})
	updatedModel, cmd = m.Update(keyMsg)
	if cmd == nil {
		t.Error("Expected ctrl+x to return a SetClipboard command")
	}
	m2 = updatedModel.(Model)
	if m2.Input != "" {
		t.Errorf("Expected input to be cleared after cut, got %q", m2.Input)
	}

	// Test alt+c (copy) preserves input and sets clipboard
	m = createTestModel()
	m.Input = "copy me"
	keyMsg = tea.KeyPressMsg(tea.Key{Text: "alt+c"})
	updatedModel, cmd = m.Update(keyMsg)
	if cmd == nil {
		t.Error("Expected alt+c to return a SetClipboard command")
	}
	m2 = updatedModel.(Model)
	if m2.Input != "copy me" {
		t.Errorf("Expected input to remain 'copy me', got %q", m2.Input)
	}
}

type dummyError struct{}

func (dummyError) Error() string {
	return "some error"
}

func TestQueryQueue(t *testing.T) {
	m := createTestModel()
	m.Loading = true // simulate active request
	m.Input = "queued query 1"

	// Press Enter to submit the query while loading
	msg := tea.KeyPressMsg(tea.Key{Text: "enter"})
	updatedModel, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("Expected no command to be returned when query is queued")
	}

	m2 := updatedModel.(Model)
	if len(m2.QueryQueue) != 1 {
		t.Fatalf("Expected 1 query in queue, got %d", len(m2.QueryQueue))
	}
	if m2.QueryQueue[0] != "queued query 1" {
		t.Errorf("Expected first queued query to be 'queued query 1', got %q", m2.QueryQueue[0])
	}
	if m2.Input != "" {
		t.Errorf("Expected input prompt to be cleared after queueing, got %q", m2.Input)
	}

	// Submit another query to queue
	m2.Input = "queued query 2"
	updatedModel, cmd = m2.Update(msg)
	m3 := updatedModel.(Model)
	if len(m3.QueryQueue) != 2 {
		t.Fatalf("Expected 2 queries in queue, got %d", len(m3.QueryQueue))
	}
	if m3.QueryQueue[1] != "queued query 2" {
		t.Errorf("Expected second queued query to be 'queued query 2', got %q", m3.QueryQueue[1])
	}

	// Trigger DoneMsg for current active query, it should start the first queued query
	updatedModel, cmd = m3.Update(llm.DoneMsg{})
	if cmd == nil {
		t.Error("Expected DoneMsg to start the next queued query and return a command")
	}
	m4 := updatedModel.(Model)
	if !m4.Loading {
		t.Error("Expected model to be loading next query")
	}
	if m4.CurrentQuery != "queued query 1" {
		t.Errorf("Expected current query to be 'queued query 1', got %q", m4.CurrentQuery)
	}
	if len(m4.QueryQueue) != 1 {
		t.Errorf("Expected queue size to decrease to 1, got %d", len(m4.QueryQueue))
	}
	if m4.QueryQueue[0] != "queued query 2" {
		t.Errorf("Expected remaining queued query to be 'queued query 2', got %q", m4.QueryQueue[0])
	}

	// Trigger ErrMsg for next query, it should start the last queued query
	updatedModel, cmd = m4.Update(llm.ErrMsg{Err: dummyError{}})
	if cmd == nil {
		t.Error("Expected ErrMsg to start the next queued query and return a command")
	}
	m5 := updatedModel.(Model)
	if !m5.Loading {
		t.Error("Expected model to be loading next query after error")
	}
	if m5.CurrentQuery != "queued query 2" {
		t.Errorf("Expected current query to be 'queued query 2', got %q", m5.CurrentQuery)
	}
	if len(m5.QueryQueue) != 0 {
		t.Errorf("Expected queue to be empty, got %d", len(m5.QueryQueue))
	}

	// Trigger DoneMsg for last query, it should finish loading and not return any command
	updatedModel, cmd = m5.Update(llm.DoneMsg{})
	if cmd != nil {
		t.Error("Expected no command when queue is empty and query finishes")
	}
	m6 := updatedModel.(Model)
	if m6.Loading {
		t.Error("Expected model to stop loading when queue is empty")
	}
}
