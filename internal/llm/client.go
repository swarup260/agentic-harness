package llm

import (
	"context"
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

type Client struct {
	openaiClient *openai.Client
	url          string
}

// NewClient creates a new LLM client wrapper.
func NewClient(url string) *Client {
	c := openai.NewClient(
		option.WithBaseURL(url),
		option.WithAPIKey("sk-no-key"),
	)
	return &Client{
		openaiClient: &c,
		url:          url,
	}
}

// URL returns the base URL of the client.
func (c *Client) URL() string {
	return c.url
}

// SubmitQuery triggers the async streaming completion request and sends responses to ch.
func (c *Client) SubmitQuery(ctx context.Context, systemPrompt, query string, ch chan<- tea.Msg) {
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(query),
		},
		Seed:        openai.Int(0),
		Temperature: openai.Float(0),
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	}

	go func() {
		startTime := time.Now()
		var ttft time.Duration
		var firstToken bool = true

		stream := c.openaiClient.Chat.Completions.NewStreaming(ctx, params)
		defer stream.Close()

		for stream.Next() {
			event := stream.Current()

			if firstToken && len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
				ttft = time.Since(startTime)
				firstToken = false
			}

			if len(event.Choices) > 0 {
				content := event.Choices[0].Delta.Content
				if content != "" {
					ch <- TokenMsg{Token: content}
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
