package main

import (
	"strings"
	"testing"
)

func TestRenderInline(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "This is **bold** text",
			expected: "This is \033[1mbold\033[22m text",
		},
		{
			input:    "This is *italic* text",
			expected: "This is \033[3mitalic\033[23m text",
		},
		{
			input:    "This is `code` block",
			expected: "This is \033[38;5;208mcode\033[39m block",
		},
		{
			input:    "Mixed **bold** and *italic* and `code`",
			expected: "Mixed \033[1mbold\033[22m and \033[3mitalic\033[23m and \033[38;5;208mcode\033[39m",
		},
	}

	for _, tc := range tests {
		got := renderInline(tc.input)
		if got != tc.expected {
			t.Errorf("renderInline(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestIsNumberedList(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1. First item", true},
		{"12. Twelfth item", true},
		{"1.Not list", false},
		{"List item", false},
		{"- Bullet", false},
	}

	for _, tc := range tests {
		got := isNumberedList(tc.input)
		if got != tc.expected {
			t.Errorf("isNumberedList(%q) = %v; want %v", tc.input, got, tc.expected)
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	// Test bullet list wrapping and bullets
	bulletInput := "- Buy carrots and check prices at market"
	bulletOutput := renderMarkdown(bulletInput, 40)
	if len(bulletOutput) == 0 {
		t.Fatalf("Expected non-empty output for bullet list item")
	}
	if !strings.Contains(bulletOutput[0], "• ") {
		t.Errorf("Expected first line to contain bullet character, got: %q", bulletOutput[0])
	}

	// Test heading H1
	h1Input := "# Stock Analysis"
	h1Output := renderMarkdown(h1Input, 40)
	if len(h1Output) < 2 {
		t.Fatalf("Expected at least two lines for H1 heading (heading + underline)")
	}
	if !strings.Contains(h1Output[0], "Stock Analysis") {
		t.Errorf("Expected H1 text to contain heading, got: %q", h1Output[0])
	}

	// Test code blocks
	codeBlockInput := "```\nfmt.Println(\"Hello\")\n```"
	codeBlockOutput := renderMarkdown(codeBlockInput, 40)
	if len(codeBlockOutput) < 3 {
		t.Fatalf("Expected code block output to contain at least 3 lines (top, code, bottom)")
	}
	if !strings.Contains(codeBlockOutput[0], "┌") {
		t.Errorf("Expected top border '┌', got: %q", codeBlockOutput[0])
	}
	if !strings.Contains(codeBlockOutput[len(codeBlockOutput)-1], "└") {
		t.Errorf("Expected bottom border '└', got: %q", codeBlockOutput[len(codeBlockOutput)-1])
	}
}
