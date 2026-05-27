package ui

import (
	"regexp"
	"strings"
)

// ANSI color escape sequences used for rendering markdown elements.
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
