package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/openai/openai-go/v3"
	"github.com/swarup260/agent-harness-loop/internal/llm"
)

func TestBuildMessages(t *testing.T) {
	m := createTestModel()
	m.SystemPrompt = "You are a test analyst."
	m.CurrentQuery = "What about TSLA?"
	m.History = []ChatMessage{
		{Query: "What about AAPL?", Response: "AAPL looks good."},
		{Query: "/clear", Response: "History cleared."},
		{Query: "What about MSFT?", Response: "MSFT is strong."},
	}

	msgs := m.buildMessages()

	// Expected: system + user(AAPL) + assistant(AAPL) + user(MSFT) + assistant(MSFT) + user(TSLA)
	// The /clear entry should be skipped.
	if len(msgs) != 6 {
		t.Fatalf("Expected 6 messages, got %d", len(msgs))
	}

	if msgs[0].OfSystem == nil {
		t.Error("Expected first message to be system")
	}
	if msgs[0].OfSystem.Content.OfString.Value != "You are a test analyst." {
		t.Errorf("Expected system prompt, got %q", msgs[0].OfSystem.Content.OfString.Value)
	}

	if msgs[1].OfUser == nil || msgs[1].OfUser.Content.OfString.Value != "What about AAPL?" {
		t.Error("Expected second message to be user query about AAPL")
	}
	if msgs[2].OfAssistant == nil || msgs[2].OfAssistant.Content.OfString.Value != "AAPL looks good." {
		t.Error("Expected third message to be assistant response about AAPL")
	}
	if msgs[3].OfUser == nil || msgs[3].OfUser.Content.OfString.Value != "What about MSFT?" {
		t.Error("Expected fourth message to be user query about MSFT")
	}
	if msgs[4].OfAssistant == nil || msgs[4].OfAssistant.Content.OfString.Value != "MSFT is strong." {
		t.Error("Expected fifth message to be assistant response about MSFT")
	}
	if msgs[5].OfUser == nil || msgs[5].OfUser.Content.OfString.Value != "What about TSLA?" {
		t.Error("Expected sixth message to be current user query about TSLA")
	}
}

func TestBuildMessagesRespectsMaxHistory(t *testing.T) {
	m := createTestModel()
	m.MaxHistorySize = 2
	m.CurrentQuery = "latest"
	m.History = []ChatMessage{
		{Query: "q1", Response: "r1"},
		{Query: "q2", Response: "r2"},
		{Query: "q3", Response: "r3"},
	}

	msgs := m.buildMessages()

	// system + user(q2) + assistant(r2) + user(q3) + assistant(r3) + user(latest) = 6
	// q1 should be dropped (only last 2 history entries kept)
	if len(msgs) != 6 {
		t.Fatalf("Expected 6 messages (system + 2 history pairs + current), got %d", len(msgs))
	}
	if msgs[1].OfUser.Content.OfString.Value != "q2" {
		t.Errorf("Expected first history entry to be q2, got %q", msgs[1].OfUser.Content.OfString.Value)
	}
}

func TestToolCallDispatch(t *testing.T) {
	m := createTestModel()
	m.Loading = true
	m.CurrentQuery = "Price of AAPL?"
	m.Messages = m.buildMessages()
	m.OpenAITools = m.Registry.ToOpenAITools()
	m.MsgChan = make(chan tea.Msg, 500)

	toolCallMsg := llm.ToolCallMsg{
		Content: "Let me check that for you.",
		Calls: []llm.ToolCall{
			{ID: "call_1", Name: "get_stock_info", Arguments: `{"symbol":"AAPL"}`},
		},
	}

	updatedModel, cmd := m.Update(toolCallMsg)
	m2 := updatedModel.(Model)

	// Verify assistant message with tool calls was appended
	lastMsg := m2.Messages[len(m2.Messages)-1]
	if lastMsg.OfAssistant == nil {
		t.Fatal("Expected last message to be assistant message with tool calls")
	}
	if len(lastMsg.OfAssistant.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(lastMsg.OfAssistant.ToolCalls))
	}
	tc := lastMsg.OfAssistant.ToolCalls[0].OfFunction
	if tc.ID != "call_1" {
		t.Errorf("Expected tool call ID 'call_1', got %q", tc.ID)
	}
	if tc.Function.Name != "get_stock_info" {
		t.Errorf("Expected tool name 'get_stock_info', got %q", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"symbol":"AAPL"}` {
		t.Errorf("Expected arguments, got %q", tc.Function.Arguments)
	}

	// Verify content was preserved
	if lastMsg.OfAssistant.Content.OfString.Value != "Let me check that for you." {
		t.Errorf("Expected assistant content, got %q", lastMsg.OfAssistant.Content.OfString.Value)
	}

	// Verify tool events
	if len(m2.ToolEvents) != 1 {
		t.Fatalf("Expected 1 tool event, got %d", len(m2.ToolEvents))
	}
	if !strings.Contains(m2.ToolEvents[0], "get_stock_info") {
		t.Errorf("Expected tool event to mention tool name, got %q", m2.ToolEvents[0])
	}

	// Verify tool call count incremented
	if m2.ToolCallCount != 1 {
		t.Errorf("Expected tool call count 1, got %d", m2.ToolCallCount)
	}

	// Execute the command to get toolResultMsg
	if cmd == nil {
		t.Fatal("Expected non-nil command (executeTools)")
	}
	resultMsg := cmd()
	result, ok := resultMsg.(toolResultMsg)
	if !ok {
		t.Fatalf("Expected toolResultMsg, got %T", resultMsg)
	}
	if len(result.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Err != nil {
		t.Errorf("Expected no error from tool execution, got %v", result.Results[0].Err)
	}
	if !strings.Contains(result.Results[0].Result, "Stock info result") {
		t.Errorf("Expected result to contain 'Stock info result', got %q", result.Results[0].Result)
	}
	if result.Results[0].CallID != "call_1" {
		t.Errorf("Expected call ID 'call_1', got %q", result.Results[0].CallID)
	}
}

func TestToolResultAppendsToolMessage(t *testing.T) {
	m := createTestModel()
	m.Loading = true
	m.CurrentQuery = "Price of AAPL?"
	m.Messages = []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("test"),
		openai.UserMessage("Price of AAPL?"),
	}
	m.OpenAITools = m.Registry.ToOpenAITools()
	m.ToolCallCount = 1

	resultMsg := toolResultMsg{
		Results: []toolResult{
			{CallID: "call_1", Name: "get_stock_info", Result: "price is $150"},
		},
	}

	updatedModel, cmd := m.Update(resultMsg)
	m2 := updatedModel.(Model)

	// Verify tool message was appended
	lastMsg := m2.Messages[len(m2.Messages)-1]
	if lastMsg.OfTool == nil {
		t.Fatal("Expected last message to be a tool message")
	}
	if lastMsg.OfTool.ToolCallID != "call_1" {
		t.Errorf("Expected tool call ID 'call_1', got %q", lastMsg.OfTool.ToolCallID)
	}
	if lastMsg.OfTool.Content.OfString.Value != "price is $150" {
		t.Errorf("Expected tool content 'price is $150', got %q", lastMsg.OfTool.Content.OfString.Value)
	}

	// Verify a new channel was created and a command returned for re-submission
	if cmd == nil {
		t.Error("Expected non-nil command for re-submission")
	}
	if m2.MsgChan == nil {
		t.Error("Expected new MsgChan to be created")
	}
}

func TestToolCallUnknownTool(t *testing.T) {
	m := createTestModel()
	m.Loading = true
	m.CurrentQuery = "test"
	m.Messages = m.buildMessages()
	m.OpenAITools = m.Registry.ToOpenAITools()
	m.MsgChan = make(chan tea.Msg, 500)

	toolCallMsg := llm.ToolCallMsg{
		Calls: []llm.ToolCall{
			{ID: "call_x", Name: "nonexistent_tool", Arguments: "{}"},
		},
	}

	_, cmd := m.Update(toolCallMsg)
	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}
	resultMsg := cmd()
	result, ok := resultMsg.(toolResultMsg)
	if !ok {
		t.Fatalf("Expected toolResultMsg, got %T", resultMsg)
	}
	if result.Results[0].Err == nil {
		t.Error("Expected error for unknown tool")
	}
}

func TestToolLoopCap(t *testing.T) {
	m := createTestModel()
	m.Loading = true
	m.CurrentQuery = "test"
	m.Messages = m.buildMessages()
	m.OpenAITools = m.Registry.ToOpenAITools()
	m.MsgChan = make(chan tea.Msg, 500)
	m.ToolCallCount = maxToolRounds

	toolCallMsg := llm.ToolCallMsg{
		Calls: []llm.ToolCall{
			{ID: "call_1", Name: "get_stock_info", Arguments: `{"symbol":"AAPL"}`},
		},
	}

	updatedModel, cmd := m.Update(toolCallMsg)
	m2 := updatedModel.(Model)

	if m2.Loading {
		t.Error("Expected loading to be false after tool call cap reached")
	}
	if cmd != nil {
		t.Error("Expected nil command after tool call cap reached")
	}
	if len(m2.History) == 0 {
		t.Fatal("Expected history to have an entry")
	}
	last := m2.History[len(m2.History)-1]
	if !strings.Contains(last.Response, "Tool call limit reached") {
		t.Errorf("Expected response to mention tool call limit, got %q", last.Response)
	}
}
