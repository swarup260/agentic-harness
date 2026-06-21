package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/openai/openai-go/v3"
	"github.com/swarup260/agent-harness-loop/internal/config"
	"github.com/swarup260/agent-harness-loop/internal/llm"
	"github.com/swarup260/agent-harness-loop/internal/tools"
)

// maxToolRounds caps consecutive tool-call rounds to prevent infinite loops.
const maxToolRounds = 5

// ChatMessage represents a single message-response transaction in history.
type ChatMessage struct {
	Query            string
	Response         string
	Thought          string
	ToolEvents       []string
	TTFT             time.Duration
	TotalTime        time.Duration
	PromptTokens     int
	CompletionTokens int
}

// Model represents the state of the Bubble Tea application.
type Model struct {
	Config    *config.Config
	LlmClient *llm.Client
	Registry  *tools.Registry
	Ctx       context.Context
	History   []ChatMessage

	SystemPrompt string

	Input   string
	Loading bool

	CurrentQuery    string
	RawStreamBuffer string

	// In-flight conversation state for multi-turn tool-calling
	Messages       []openai.ChatCompletionMessageParamUnion
	OpenAITools    []openai.ChatCompletionToolUnionParam
	ToolCallCount  int
	ToolEvents     []string

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

	// Modal state
	ConfigPath   string
	ModalMode    string // "", "provider_form", "connect_list"
	FormName     string
	FormBaseURL  string
	FormAPIKey   string
	FormField    int // 0=name, 1=baseURL, 2=apiKey
	ConnectNames []string
	ConnectIndex int

	// Autocomplete
	AutocompleteSuggestions []commandSuggestion
	AutocompleteIndex       int
}

// NewModel initializes the Bubble Tea model with config, llm client, and tool registry.
func NewModel(cfg *config.Config, llmClient *llm.Client, registry *tools.Registry, configPath string) Model {
	return Model{
		Config:         cfg,
		LlmClient:      llmClient,
		Registry:       registry,
		Ctx:            context.Background(),
		AutoScroll:     true,
		Width:          80, // default until WindowSizeMsg
		Height:         24, // default until WindowSizeMsg
		MaxHistorySize: cfg.MaxHistorySize,
		SystemPrompt:   cfg.SystemPrompt,
		ConfigPath:     configPath,
	}
}

// Init initializes the model's commands on start.
func (m Model) Init() tea.Cmd {
	return nil
}
