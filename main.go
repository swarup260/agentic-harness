package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/config"
	"github.com/swarup260/agent-harness-loop/internal/llm"
	"github.com/swarup260/agent-harness-loop/internal/ui"
)

func main() {
	cfg := config.DefaultConfig()
	llmClient := llm.NewClient(cfg.LLMURL)
	m := ui.NewModel(cfg, llmClient)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}

	// Restore normal terminal mouse mode upon exit
	fmt.Print("\x1b[?1000l\x1b[?1006l")
}
