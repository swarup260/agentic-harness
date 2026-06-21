package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/config"
	"github.com/swarup260/agent-harness-loop/internal/llm"
	"github.com/swarup260/agent-harness-loop/internal/tools"
	"github.com/swarup260/agent-harness-loop/internal/ui"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	p := cfg.ActiveProviderConfig()
	llmClient := llm.NewClient(p.BaseURL, p.APIKey, p.Model, cfg.Seed, cfg.Temperature)

	registry := tools.NewRegistry()
	registry.Register(&tools.StockInfoTool{})
	registry.Register(&tools.FinancialsTool{})

	m := ui.NewModel(cfg, llmClient, registry, "config.json")

	prog := tea.NewProgram(m)
	if _, err := prog.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("\x1b[?1000l\x1b[?1006l")
}
