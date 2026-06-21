package llm

import (
	"context"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Message types returned during LLM stream processing.
type TokenMsg struct {
	Token string
}

type UsageMsg struct {
	PromptTokens     int
	CompletionTokens int
	TTFT             time.Duration
	TotalTime        time.Duration
}

type ErrMsg struct {
	Err error
}

type DoneMsg struct{}

// ToolCall represents a single function call requested by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// ToolCallMsg is sent when the model requests one or more tool calls instead
// of a final response. Content holds any text the model produced alongside
// the tool calls (may be empty).
type ToolCallMsg struct {
	Content string
	Calls   []ToolCall
}

type Client struct {
	openaiClient *openai.Client
	baseURL      string
	apiKey       string
	model        string
	seed         *int64
	temperature  *float64
}

// NewClient creates a new OpenAI-compatible LLM client.
func NewClient(baseURL, apiKey, model string, seed *int64, temperature *float64) *Client {
	c := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
	return &Client{
		openaiClient: &c,
		baseURL:      baseURL,
		apiKey:       apiKey,
		model:        model,
		seed:         seed,
		temperature:  temperature,
	}
}

// URL returns the base URL of the client.
func (c *Client) URL() string {
	return c.baseURL
}

// Model returns the configured model name.
func (c *Client) Model() string {
	return c.model
}

// APIKey returns the configured API key.
func (c *Client) APIKey() string {
	return c.apiKey
}

// SubmitQuery triggers the async streaming completion request and sends
// responses to ch. If tools is non-empty they are offered to the model; when
// the model calls a tool a ToolCallMsg is sent instead of DoneMsg.
func (c *Client) SubmitQuery(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
	tools []openai.ChatCompletionToolUnionParam,
	ch chan<- tea.Msg,
) {
	params := openai.ChatCompletionNewParams{
		Messages: messages,
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	}
	if c.model != "" {
		params.Model = c.model
	}
	if c.seed != nil {
		params.Seed = openai.Int(*c.seed)
	}
	if c.temperature != nil {
		params.Temperature = openai.Float(*c.temperature)
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	go func() {
		startTime := time.Now()
		var ttft time.Duration
		firstToken := true

		var contentBuilder strings.Builder
		toolCalls := map[int64]*ToolCall{}
		var hasToolCalls bool

		stream := c.openaiClient.Chat.Completions.NewStreaming(ctx, params)
		defer stream.Close()

		for stream.Next() {
			event := stream.Current()

			if len(event.Choices) > 0 {
				choice := event.Choices[0]

				if choice.Delta.Content != "" {
					if firstToken {
						ttft = time.Since(startTime)
						firstToken = false
					}
					contentBuilder.WriteString(choice.Delta.Content)
					ch <- TokenMsg{Token: choice.Delta.Content}
				}

				for _, tc := range choice.Delta.ToolCalls {
					if firstToken {
						ttft = time.Since(startTime)
						firstToken = false
					}
					call, exists := toolCalls[tc.Index]
					if !exists {
						call = &ToolCall{}
						toolCalls[tc.Index] = call
					}
					if tc.ID != "" {
						call.ID = tc.ID
					}
					if tc.Function.Name != "" {
						call.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						call.Arguments += tc.Function.Arguments
					}
				}

				if choice.FinishReason == "tool_calls" {
					hasToolCalls = true
				}
			}

			if event.Usage.TotalTokens > 0 {
				ch <- UsageMsg{
					PromptTokens:     int(event.Usage.PromptTokens),
					CompletionTokens: int(event.Usage.CompletionTokens),
					TTFT:             ttft,
					TotalTime:        time.Since(startTime),
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- ErrMsg{Err: err}
			return
		}

		if hasToolCalls || len(toolCalls) > 0 {
			indices := make([]int64, 0, len(toolCalls))
			for idx := range toolCalls {
				indices = append(indices, idx)
			}
			sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })
			calls := make([]ToolCall, 0, len(indices))
			for _, idx := range indices {
				calls = append(calls, *toolCalls[idx])
			}
			ch <- ToolCallMsg{Content: contentBuilder.String(), Calls: calls}
		} else {
			ch <- DoneMsg{}
		}
	}()
}

// ParseStream splits a stream buffer containing "<think>...</think>" tags.
func ParseStream(buffer string) (thought string, response string, isThinking bool) {
	if !strings.Contains(buffer, "<think>") {
		return "", buffer, false
	}

	parts := strings.SplitN(buffer, "<think>", 2)
	afterThink := parts[1]

	if !strings.Contains(afterThink, "</think>") {
		return afterThink, "", true
	}

	thinkParts := strings.SplitN(afterThink, "</think>", 2)
	return thinkParts[0], thinkParts[1], false
}
