package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/swarup260/agent-harness-loop/internal/llm"
)

func (m Model) getWrappedLines(width int) []string {
	var lines []string

	for _, msg := range m.History {
		// User Query (wrapped to prevent layout overflow)
		queryLines := wrapText(msg.Query, width-10)
		if len(queryLines) > 0 {
			lines = append(lines, colorGrey+"> "+colorReset+queryLines[0])
			for qIdx := 1; qIdx < len(queryLines); qIdx++ {
				lines = append(lines, "  "+queryLines[qIdx])
			}
		}
		lines = append(lines, "") // space after query

		// Thought block if present
		if msg.Thought != "" {
			thoughtTimeStr := fmt.Sprintf("%.1fs", msg.TotalTime.Seconds())
			if msg.TTFT > 0 {
				thoughtTimeStr = fmt.Sprintf("%.1fs", msg.TTFT.Seconds())
			}
			lines = append(lines, colorOrange+"+ Thought: "+thoughtTimeStr+colorReset)
			lines = append(lines, "")
		}

		// Assistant Response (rendered as Markdown)
		respLines := renderMarkdown(msg.Response, width-8)
		for _, rl := range respLines {
			lines = append(lines, "  "+rl)
		}
		lines = append(lines, "") // space

		if strings.HasPrefix(msg.Query, "/") {
			footerText := fmt.Sprintf("%s■%s Command", colorOrange, colorReset)
			lines = append(lines, "  "+footerText)
			lines = append(lines, "")
			continue
		}

		// Message Footer
		footerTime := fmt.Sprintf("%.1fs", msg.TotalTime.Seconds())
		if msg.TotalTime >= time.Minute {
			mins := int(msg.TotalTime.Minutes())
			secs := int(msg.TotalTime.Seconds()) % 60
			footerTime = fmt.Sprintf("%dm %ds", mins, secs)
		}

		// Prompt speed calculation
		speedStr := "0.0 tok/s"
		if msg.TTFT > 0 && msg.PromptTokens > 0 {
			speed := float64(msg.PromptTokens) / msg.TTFT.Seconds()
			speedStr = fmt.Sprintf("%.1f tok/s", speed)
		} else if msg.TotalTime > 0 && msg.PromptTokens > 0 {
			speed := float64(msg.PromptTokens) / msg.TotalTime.Seconds()
			speedStr = fmt.Sprintf("%.1f tok/s", speed)
		}

		footerText := fmt.Sprintf("%s■%s Query · LLM (%s) · %s", colorBlue, colorReset, speedStr, footerTime)
		lines = append(lines, "  "+footerText)
		lines = append(lines, "")
	}

	// Add current active streaming query/response
	if m.Loading {
		// User Query (wrapped to prevent layout overflow)
		queryLines := wrapText(m.CurrentQuery, width-10)
		if len(queryLines) > 0 {
			lines = append(lines, colorGrey+"> "+colorReset+queryLines[0])
			for qIdx := 1; qIdx < len(queryLines); qIdx++ {
				lines = append(lines, "  "+queryLines[qIdx])
			}
		}
		lines = append(lines, "")

		thought, resp, isThinking := llm.ParseStream(m.RawStreamBuffer)

		if thought != "" {
			if isThinking {
				lines = append(lines, colorOrange+"+ Thinking..."+colorReset)
				thoughtLines := wrapText(thought, width-8)
				for _, tl := range thoughtLines {
					lines = append(lines, "  "+colorGrey+tl+colorReset)
				}
				lines = append(lines, "")
			} else {
				thoughtTimeStr := fmt.Sprintf("%.1fs", m.Ttft.Seconds())
				if m.Ttft == 0 {
					thoughtTimeStr = fmt.Sprintf("%.1fs", time.Since(m.StartTime).Seconds())
				}
				lines = append(lines, colorOrange+"+ Thought: "+thoughtTimeStr+colorReset)
				lines = append(lines, "")
			}
		} else if resp == "" {
			lines = append(lines, colorOrange+"+ LLM is thinking..."+colorReset)
			lines = append(lines, "")
		}

		if resp != "" {
			// Assistant Response (rendered as Markdown during stream)
			respLines := renderMarkdown(resp, width-8)
			for _, rl := range respLines {
				lines = append(lines, "  "+rl)
			}
		}
	}

	// Show queued/waiting messages
	for qIdx, queuedQuery := range m.QueryQueue {
		wrapped := wrapText(queuedQuery, width-10)
		if len(wrapped) > 0 {
			lines = append(lines, colorGrey+fmt.Sprintf("> (Waiting in queue #%d): %s", qIdx+1, wrapped[0])+colorReset)
			for j := 1; j < len(wrapped); j++ {
				lines = append(lines, colorGrey+"  "+wrapped[j]+colorReset)
			}
			lines = append(lines, "")
		}
	}

	return lines
}

func (m Model) getScrollBottom() int {
	viewportHeight := m.Height - 10
	if viewportHeight < 3 {
		viewportHeight = 3
	}
	wrappedLines := m.getWrappedLines(m.Width)
	maxScroll := len(wrappedLines) - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (m Model) View() tea.View {
	W := m.Width
	H := m.Height

	viewportHeight := H - 10
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	wrappedLines := m.getWrappedLines(W)

	scrollOffset := m.ScrollOffset
	if m.AutoScroll {
		scrollOffset = m.getScrollBottom()
	}

	var sb strings.Builder

	// 1. Header (fixed 2 lines)
	sb.WriteString(colorBold)
	sb.WriteString(colorCyan)
	sb.WriteString(" 🚀 LLM Stock Analysis Harness")
	sb.WriteString(colorReset)
	sb.WriteString(" (")
	sb.WriteString(m.LlmClient.URL())
	sb.WriteString(")\n")
	sb.WriteString(colorGrey)
	sb.WriteString(strings.Repeat("─", W))
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	// 2. Output Box (with ANSI style resets at end of lines to prevent styling bleed)
	for i := 0; i < viewportHeight; i++ {
		lineIdx := scrollOffset + i
		if lineIdx >= 0 && lineIdx < len(wrappedLines) {
			sb.WriteString(wrappedLines[lineIdx])
			sb.WriteString(colorReset)
		}
		sb.WriteString("\n")
	}

	// 3. Input Box (bordered)
	sb.WriteString(colorGrey)
	sb.WriteString("┌")
	sb.WriteString(strings.Repeat("─", W-2))
	sb.WriteString("┐")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	promptLabel := colorCyan + "Prompt" + colorReset + colorGrey + " · LLM " + colorReset

	inputText := m.Input
	cursorStr := "█"

	inputLineContent := "  " + promptLabel + " " + inputText + cursorStr

	contentLen := visualLength(inputLineContent)
	paddingLen := W - 2 - contentLen
	if paddingLen < 0 {
		paddingLen = 0
	}

	sb.WriteString(colorGrey)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString(inputLineContent)
	sb.WriteString(strings.Repeat(" ", paddingLen))
	sb.WriteString(colorGrey)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")
	sb.WriteString(colorGrey)
	sb.WriteString("└")
	sb.WriteString(strings.Repeat("─", W-2))
	sb.WriteString("┘")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	// 4. Bottom status
	var statsParts []string
	if m.TotalTime > 0 {
		statsParts = append(statsParts, fmt.Sprintf("Total: %.1fs", m.TotalTime.Seconds()))
	} else if m.Loading {
		elapsed := time.Since(m.StartTime)
		statsParts = append(statsParts, fmt.Sprintf("Total: %.1fs", elapsed.Seconds()))
	}

	if len(m.QueryQueue) > 0 {
		statsParts = append(statsParts, fmt.Sprintf("Queued: %d", len(m.QueryQueue)))
	}

	if m.Ttft > 0 {
		statsParts = append(statsParts, fmt.Sprintf("TTFT: %.1fs", m.Ttft.Seconds()))
	}

	if m.PromptTokens > 0 {
		if m.Ttft > 0 {
			speed := float64(m.PromptTokens) / m.Ttft.Seconds()
			statsParts = append(statsParts, fmt.Sprintf("Prompt Speed: %.1f tok/s", speed))
		}
		statsParts = append(statsParts, fmt.Sprintf("Prompt: %d tok", m.PromptTokens))
	}

	statsParts = append(statsParts, fmt.Sprintf("Buffer: %d (ctrl+↑/↓)", m.MaxHistorySize))
	statsParts = append(statsParts, "ctrl+c quit")
	statsText := strings.Join(statsParts, " · ")

	statsLen := visualLength(statsText)
	statsPadding := W - statsLen - 2
	if statsPadding < 0 {
		statsPadding = 0
	}
	sb.WriteString(strings.Repeat(" ", statsPadding))
	sb.WriteString(colorGrey)
	sb.WriteString(statsText)
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	v := tea.NewView(sb.String() + "\x1b[?1000h\x1b[?1006h")
	v.AltScreen = true
	v.MouseMode = tea.MouseModeNone
	return v
}
