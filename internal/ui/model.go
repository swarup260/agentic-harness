package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/config"
	"github.com/swarup260/agent-harness-loop/internal/llm"
)

// ChatMessage represents a single message-response transaction in history.
type ChatMessage struct {
	Query            string
	Response         string
	Thought          string
	TTFT             time.Duration
	TotalTime        time.Duration
	PromptTokens     int
	CompletionTokens int
}

// Model represents the state of the Bubble Tea application.
type Model struct {
	Config       *config.Config
	LlmClient    *llm.Client
	Ctx          context.Context
	History      []ChatMessage
	SystemPrompt string

	Input   string
	Loading bool

	CurrentQuery    string
	RawStreamBuffer string

	// Metrics for active request
	StartTime        time.Time
	Ttft             time.Duration
	TotalTime        time.Duration
	PromptTokens     int
	CompletionTokens int

	// Viewport / scrolling
	Width        int
	Height       int
	ScrollOffset int
	AutoScroll   bool

	// History cycling
	HistoryIndex   int
	SavedInput     string
	MaxHistorySize int

	// Communication channel for async LLM stream
	MsgChan chan tea.Msg

	// Queue for pending queries
	QueryQueue []string
}

// NewModel initializes the Bubble Tea model with config and llm client.
func NewModel(cfg *config.Config, llmClient *llm.Client) Model {
	return Model{
		Config:         cfg,
		LlmClient:      llmClient,
		Ctx:            context.Background(),
		AutoScroll:     true,
		Width:          80, // default until WindowSizeMsg
		Height:         24, // default until WindowSizeMsg
		MaxHistorySize: cfg.MaxHistorySize,
		SystemPrompt:   cfg.SystemPrompt,
	}
}

// Init initializes the model's commands on start.
func (m Model) Init() tea.Cmd {
	return nil
}
