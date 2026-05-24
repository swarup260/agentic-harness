package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	tea "charm.land/bubbletea/v2"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
	colorGrey   = "\033[90m"
	colorOrange = "\033[38;5;208m"
	colorBlue   = "\033[34m"
	colorGreen  = "\033[32m"
	colorWhite  = "\033[37m"
)

type chatMessage struct {
	Query            string
	Response         string
	Thought          string
	TTFT             time.Duration
	TotalTime        time.Duration
	PromptTokens     int
	CompletionTokens int
}

type model struct {
	llmURL           string
	client           *openai.Client
	ctx              context.Context
	history          []chatMessage

	input            string
	loading          bool

	currentQuery     string
	rawStreamBuffer  string

	// Metrics for active request
	startTime        time.Time
	ttft             time.Duration
	totalTime        time.Duration
	promptTokens     int
	completionTokens int

	// Viewport / scrolling
	width            int
	height           int
	scrollOffset     int
	autoScroll       bool

	// History cycling
	historyIndex     int
	savedInput       string
	maxHistorySize   int

	// Communication channel for async LLM stream
	msgChan          chan tea.Msg
}

// Bubble Tea Message types
type tokenMsg struct {
	token string
}

type usageMsg struct {
	promptTokens     int
	completionTokens int
	ttft             time.Duration
	totalTime        time.Duration
}

type errMsg struct {
	err error
}

type doneMsg struct{}

func initialModel() model {
	llmURL := "http://0.0.0.0:8080"
	client := openai.NewClient(
		option.WithBaseURL(llmURL),
		option.WithAPIKey("sk-no-key"),
	)
	return model{
		llmURL:         llmURL,
		client:         &client,
		ctx:            context.Background(),
		autoScroll:     true,
		width:          80, // default until WindowSizeMsg
		height:         24, // default until WindowSizeMsg
		maxHistorySize: 5,  // default limit of 5 questions stored
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func waitForMsg(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func parseStream(buffer string) (thought string, response string, isThinking bool) {
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

func stripANSI(s string) string {
	var builder strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		builder.WriteByte(s[i])
	}
	return builder.String()
}

func visualLength(s string) int {
	runes := []rune(stripANSI(s))
	return len(runes)
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var lines []string
	paragraphs := strings.Split(text, "\n")
	for _, para := range paragraphs {
		if para == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Split(para, " ")
		var currentLine strings.Builder

		for _, word := range words {
			if word == "" {
				currentLine.WriteString(" ")
				continue
			}

			wordLen := visualLength(word)
			currentLineLen := visualLength(currentLine.String())

			if currentLineLen == 0 {
				currentLine.WriteString(word)
			} else if currentLineLen+1+wordLen <= width {
				currentLine.WriteString(" ")
				currentLine.WriteString(word)
			} else {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}
		if currentLine.Len() > 0 {
			lines = append(lines, currentLine.String())
		}
	}
	return lines
}

func renderInline(text string) string {
	// 1. Inline code: `code` -> colorOrange + code + \033[39m
	reCode := regexp.MustCompile("`([^`]+)`")
	text = reCode.ReplaceAllString(text, colorOrange+"$1"+"\033[39m")

	// 2. Bold: **text** -> \033[1m + text + \033[22m
	reBold1 := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	text = reBold1.ReplaceAllString(text, "\033[1m$1\033[22m")

	reBold2 := regexp.MustCompile(`__([^_]+)__`)
	text = reBold2.ReplaceAllString(text, "\033[1m$1\033[22m")

	// 3. Italic: *text* -> \033[3m + text + \033[23m
	reItalic1 := regexp.MustCompile(`\*([^*]+)\*`)
	text = reItalic1.ReplaceAllString(text, "\033[3m$1\033[23m")

	reItalic2 := regexp.MustCompile(`_([^_]+)_`)
	text = reItalic2.ReplaceAllString(text, "\033[3m$1\033[23m")

	return text
}

func isNumberedList(s string) bool {
	if len(s) < 3 {
		return false
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s)-1 && s[i] == '.' && s[i+1] == ' ' {
		return true
	}
	return false
}

func renderCodeBlock(lines []string, width int) []string {
	var result []string
	if len(lines) == 0 {
		return result
	}

	borderWidth := width - 6
	if borderWidth < 10 {
		borderWidth = 10
	}
	result = append(result, colorGrey+"┌"+strings.Repeat("─", borderWidth)+"┐"+colorReset)

	for _, line := range lines {
		wrapped := wrapText(line, borderWidth-2)
		for _, wl := range wrapped {
			paddingLen := borderWidth - 2 - visualLength(wl)
			if paddingLen < 0 {
				paddingLen = 0
			}
			result = append(result, colorGrey+"│ "+colorReset+wl+strings.Repeat(" ", paddingLen)+colorGrey+" │"+colorReset)
		}
	}

	result = append(result, colorGrey+"└"+strings.Repeat("─", borderWidth)+"┘"+colorReset)
	return result
}

func renderMarkdown(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	rawLines := strings.Split(text, "\n")
	var formattedLines []string

	inCodeBlock := false
	var codeBlockLines []string

	for i := 0; i < len(rawLines); i++ {
		line := rawLines[i]
		trimmedLine := strings.TrimSpace(line)

		// Handle code blocks
		if strings.HasPrefix(trimmedLine, "```") {
			if inCodeBlock {
				inCodeBlock = false
				formattedLines = append(formattedLines, renderCodeBlock(codeBlockLines, width)...)
				codeBlockLines = nil
			} else {
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			codeBlockLines = append(codeBlockLines, line)
			continue
		}

		// Handle block quotes
		if strings.HasPrefix(trimmedLine, ">") {
			content := strings.TrimSpace(trimmedLine[1:])
			wrapped := wrapText(renderInline(content), width-6)
			for _, wl := range wrapped {
				formattedLines = append(formattedLines, colorGrey+"│ "+colorReset+wl)
			}
			continue
		}

		// Handle Headings
		if strings.HasPrefix(trimmedLine, "#") {
			level := 0
			for level < len(trimmedLine) && trimmedLine[level] == '#' {
				level++
			}
			if level > 0 && level < len(trimmedLine) && trimmedLine[level] == ' ' {
				content := strings.TrimSpace(trimmedLine[level:])
				content = renderInline(content)
				headerText := "\033[1m" + colorCyan + content + "\033[22m" + colorReset

				if len(formattedLines) > 0 && formattedLines[len(formattedLines)-1] != "" {
					formattedLines = append(formattedLines, "")
				}

				switch level {
				case 1:
					formattedLines = append(formattedLines, headerText)
					formattedLines = append(formattedLines, colorCyan+strings.Repeat("━", visualLength(content))+colorReset)
				case 2:
					formattedLines = append(formattedLines, headerText)
					formattedLines = append(formattedLines, colorCyan+strings.Repeat("─", visualLength(content))+colorReset)
				default:
					formattedLines = append(formattedLines, headerText)
				}
				continue
			}
		}

		// Handle Horizontal Rules
		if trimmedLine == "---" || trimmedLine == "***" || trimmedLine == "___" {
			formattedLines = append(formattedLines, colorGrey+strings.Repeat("─", width-4)+colorReset)
			continue
		}

		// Handle Bullet List Items
		if strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "* ") || strings.HasPrefix(trimmedLine, "+ ") {
			content := trimmedLine[2:]
			content = renderInline(content)

			wrapped := wrapText(content, width-6)
			if len(wrapped) > 0 {
				formattedLines = append(formattedLines, colorCyan+"• "+colorReset+wrapped[0])
				for j := 1; j < len(wrapped); j++ {
					formattedLines = append(formattedLines, "  "+wrapped[j])
				}
			}
			continue
		}

		// Handle Numbered List Items
		if isNumberedList(trimmedLine) {
			dotIdx := strings.Index(trimmedLine, ".")
			numPrefix := trimmedLine[:dotIdx+2]
			content := trimmedLine[dotIdx+2:]
			content = renderInline(content)

			prefixLen := visualLength(numPrefix)
			wrapped := wrapText(content, width-4-prefixLen)
			if len(wrapped) > 0 {
				formattedLines = append(formattedLines, colorCyan+numPrefix+colorReset+wrapped[0])
				padding := strings.Repeat(" ", prefixLen)
				for j := 1; j < len(wrapped); j++ {
					formattedLines = append(formattedLines, padding+wrapped[j])
				}
			}
			continue
		}

		// Empty line
		if trimmedLine == "" {
			formattedLines = append(formattedLines, "")
			continue
		}

		// Normal paragraph
		content := renderInline(line)
		wrapped := wrapText(content, width-4)
		formattedLines = append(formattedLines, wrapped...)
	}

	if inCodeBlock && len(codeBlockLines) > 0 {
		formattedLines = append(formattedLines, renderCodeBlock(codeBlockLines, width)...)
	}

	return formattedLines
}

func (m model) getWrappedLines(width int) []string {
	var lines []string

	for _, msg := range m.history {
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
	if m.loading {
		// User Query (wrapped to prevent layout overflow)
		queryLines := wrapText(m.currentQuery, width-10)
		if len(queryLines) > 0 {
			lines = append(lines, colorGrey+"> "+colorReset+queryLines[0])
			for qIdx := 1; qIdx < len(queryLines); qIdx++ {
				lines = append(lines, "  "+queryLines[qIdx])
			}
		}
		lines = append(lines, "")

		thought, resp, isThinking := parseStream(m.rawStreamBuffer)

		if thought != "" {
			if isThinking {
				lines = append(lines, colorOrange+"+ Thinking..."+colorReset)
				thoughtLines := wrapText(thought, width-8)
				for _, tl := range thoughtLines {
					lines = append(lines, "  "+colorGrey+tl+colorReset)
				}
				lines = append(lines, "")
			} else {
				thoughtTimeStr := fmt.Sprintf("%.1fs", m.ttft.Seconds())
				if m.ttft == 0 {
					thoughtTimeStr = fmt.Sprintf("%.1fs", time.Since(m.startTime).Seconds())
				}
				lines = append(lines, colorOrange+"+ Thought: "+thoughtTimeStr+colorReset)
				lines = append(lines, "")
			}
		}

		if resp != "" {
			// Assistant Response (rendered as Markdown during stream)
			respLines := renderMarkdown(resp, width-8)
			for _, rl := range respLines {
				lines = append(lines, "  "+rl)
			}
		}
	}

	return lines
}

func (m model) submitQuery() tea.Cmd {
	query := m.currentQuery
	ch := m.msgChan
	client := m.client
	ctx := m.ctx

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a senior stock analyst with 20+ years of experience. You always output the final answer in bullet points."),
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

		stream := client.Chat.Completions.NewStreaming(ctx, params)
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
					ch <- tokenMsg{token: content}
				}
			}

			if event.Usage.TotalTokens > 0 {
				ch <- usageMsg{
					promptTokens:     int(event.Usage.PromptTokens),
					completionTokens: int(event.Usage.CompletionTokens),
					ttft:             ttft,
					totalTime:        time.Since(startTime),
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- errMsg{err: err}
		} else {
			ch <- doneMsg{}
		}
	}()

	return waitForMsg(ch)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+up":
			m.maxHistorySize++
			m.historyIndex = len(m.history)
			return m, nil

		case "ctrl+down":
			if m.maxHistorySize > 1 {
				m.maxHistorySize--
				if len(m.history) > m.maxHistorySize {
					m.history = m.history[len(m.history)-m.maxHistorySize:]
				}
				m.historyIndex = len(m.history)
			}
			return m, nil

		case "up":
			if !m.loading && len(m.history) > 0 {
				if m.historyIndex == len(m.history) {
					m.savedInput = m.input
				}
				if m.historyIndex > 0 {
					m.historyIndex--
					m.input = m.history[m.historyIndex].Query
				}
			}
			return m, nil

		case "pgup", "shift+up":
			m.autoScroll = false
			m.scrollOffset = m.scrollOffset - 3
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return m, nil

		case "down":
			if !m.loading && len(m.history) > 0 {
				if m.historyIndex < len(m.history) {
					m.historyIndex++
					if m.historyIndex == len(m.history) {
						m.input = m.savedInput
					} else {
						m.input = m.history[m.historyIndex].Query
					}
				}
			}
			return m, nil

		case "pgdown", "shift+down":
			m.scrollOffset = m.scrollOffset + 3
			wrappedLines := m.getWrappedLines(m.width)
			viewportHeight := m.height - 10
			if viewportHeight < 3 {
				viewportHeight = 3
			}
			maxScroll := len(wrappedLines) - viewportHeight
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset >= maxScroll {
				m.scrollOffset = maxScroll
				m.autoScroll = true
			}
			return m, nil

		case "enter":
			if !m.loading && strings.TrimSpace(m.input) != "" {
				m.loading = true
				m.autoScroll = true
				m.startTime = time.Now()
				m.currentQuery = m.input
				m.input = ""
				m.rawStreamBuffer = ""
				m.ttft = 0
				m.totalTime = 0
				m.promptTokens = 0
				m.completionTokens = 0
				m.msgChan = make(chan tea.Msg, 500)
				m.savedInput = ""

				cmd := m.submitQuery()
				return m, cmd
			}
			return m, nil

		case "backspace":
			if len(m.input) > 0 {
				runes := []rune(m.input)
				m.input = string(runes[:len(runes)-1])
			}
			return m, nil

		case "esc":
			m.input = ""
			return m, nil

		default:
			keyStr := msg.String()
			if len(keyStr) == 1 && !m.loading {
				m.input += keyStr
			} else if keyStr == "space" && !m.loading {
				m.input += " "
			}
			return m, nil
		}

	case tea.MouseWheelMsg:
		mEvent := msg.Mouse()
		switch mEvent.Button {
		case tea.MouseWheelUp:
			m.autoScroll = false
			m.scrollOffset = m.scrollOffset - 1
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		case tea.MouseWheelDown:
			m.scrollOffset = m.scrollOffset + 1
			wrappedLines := m.getWrappedLines(m.width)
			viewportHeight := m.height - 10
			if viewportHeight < 3 {
				viewportHeight = 3
			}
			maxScroll := len(wrappedLines) - viewportHeight
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset >= maxScroll {
				m.scrollOffset = maxScroll
				m.autoScroll = true
			}
		}
		return m, nil

	case tokenMsg:
		m.rawStreamBuffer += msg.token
		return m, waitForMsg(m.msgChan)

	case usageMsg:
		m.promptTokens = msg.promptTokens
		m.completionTokens = msg.completionTokens
		m.ttft = msg.ttft
		m.totalTime = msg.totalTime
		return m, waitForMsg(m.msgChan)

	case errMsg:
		m.loading = false
		thought, resp, _ := parseStream(m.rawStreamBuffer)
		m.history = append(m.history, chatMessage{
			Query:            m.currentQuery,
			Response:         resp + fmt.Sprintf("\n[Error: %v]", msg.err),
			Thought:          thought,
			TTFT:             m.ttft,
			TotalTime:        time.Since(m.startTime),
			PromptTokens:     m.promptTokens,
			CompletionTokens: m.completionTokens,
		})
		if len(m.history) > m.maxHistorySize {
			m.history = m.history[len(m.history)-m.maxHistorySize:]
		}
		m.historyIndex = len(m.history)
		return m, nil

	case doneMsg:
		m.loading = false
		thought, resp, _ := parseStream(m.rawStreamBuffer)
		m.history = append(m.history, chatMessage{
			Query:            m.currentQuery,
			Response:         resp,
			Thought:          thought,
			TTFT:             m.ttft,
			TotalTime:        time.Since(m.startTime),
			PromptTokens:     m.promptTokens,
			CompletionTokens: m.completionTokens,
		})
		if len(m.history) > m.maxHistorySize {
			m.history = m.history[len(m.history)-m.maxHistorySize:]
		}
		m.historyIndex = len(m.history)
		return m, nil
	}

	return m, nil
}

func (m model) View() tea.View {
	W := m.width
	H := m.height

	viewportHeight := H - 10
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	wrappedLines := m.getWrappedLines(W)

	scrollOffset := m.scrollOffset
	if m.autoScroll {
		scrollOffset = len(wrappedLines) - viewportHeight
		if scrollOffset < 0 {
			scrollOffset = 0
		}
	}

	var sb strings.Builder

	// 1. Header (fixed 2 lines)
	sb.WriteString(colorBold)
	sb.WriteString(colorCyan)
	sb.WriteString(" 🚀 LLM Stock Analysis Harness")
	sb.WriteString(colorReset)
	sb.WriteString(" (")
	sb.WriteString(m.llmURL)
	sb.WriteString(")\n")
	sb.WriteString(colorGrey)
	sb.WriteString(strings.Repeat("─", W))
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	// 2. Output Box (with ANSI style resets at end of lines to prevent styling bleed)
	for i := 0; i < viewportHeight; i++ {
		lineIdx := scrollOffset + i
		if lineIdx >= 0 && lineIdx < len(wrappedLines) {
			sb.WriteString(colorGrey)
			sb.WriteString("│ ")
			sb.WriteString(colorReset)
			sb.WriteString(wrappedLines[lineIdx])
			sb.WriteString(colorReset)
		} else {
			sb.WriteString(colorGrey + "│ " + colorReset)
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

	promptLabel := colorCyan + "Build" + colorReset + colorGrey + " · LLM " + colorReset

	inputText := m.input
	cursorStr := "█"
	if m.loading {
		inputText = colorGrey + "(LLM is thinking...)" + colorReset
		cursorStr = ""
	}

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
	if m.totalTime > 0 {
		statsParts = append(statsParts, fmt.Sprintf("Total: %.1fs", m.totalTime.Seconds()))
	} else if m.loading {
		elapsed := time.Since(m.startTime)
		statsParts = append(statsParts, fmt.Sprintf("Total: %.1fs", elapsed.Seconds()))
	}

	if m.ttft > 0 {
		statsParts = append(statsParts, fmt.Sprintf("TTFT: %.1fs", m.ttft.Seconds()))
	}

	if m.promptTokens > 0 {
		if m.ttft > 0 {
			speed := float64(m.promptTokens) / m.ttft.Seconds()
			statsParts = append(statsParts, fmt.Sprintf("Prompt Speed: %.1f tok/s", speed))
		}
		statsParts = append(statsParts, fmt.Sprintf("Prompt: %d tok", m.promptTokens))
	}

	statsParts = append(statsParts, fmt.Sprintf("Buffer: %d (ctrl+↑/↓)", m.maxHistorySize))
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

	v := tea.NewView(sb.String())
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
