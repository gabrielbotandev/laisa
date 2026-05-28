package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTranscript builds the scrollable conversation text.
func renderTranscript(messages []ChatMessage, viewportWidth int) string {
	textWidth := transcriptContentWidth(viewportWidth)

	var b strings.Builder
	if convoPadTop > 0 && len(messages) > 0 {
		b.WriteString(strings.Repeat("\n", convoPadTop))
	}

	for i, msg := range messages {
		if i > 0 {
			b.WriteByte('\n')
			if convoMessageGap > 0 {
				b.WriteString(strings.Repeat("\n", convoMessageGap))
			}
		}
		switch msg.Role {
		case "user":
			b.WriteString(renderUserBlock(msg.Content, textWidth, viewportWidth))
		case "assistant":
			if msg.Content != "" {
				b.WriteString(renderAssistantBlock(msg.Content, textWidth, viewportWidth))
			}
		default:
			b.WriteString(renderPlainBlock(msg.Content, textWidth, viewportWidth))
		}
	}

	out := strings.TrimRight(b.String(), "\n")
	if convoPadBottom > 0 && len(messages) > 0 {
		out += strings.Repeat("\n", convoPadBottom)
	}
	return out
}

func renderUserBlock(content string, textWidth, blockWidth int) string {
	lineList := strings.Split(wrapPlain(content, textWidth), "\n")
	var parts []string
	for _, line := range lineList {
		parts = append(parts, styleUserBlock.Width(blockWidth).Render(line))
	}
	if len(parts) == 0 {
		return styleUserBlock.Width(blockWidth).Render("")
	}
	return strings.Join(parts, "\n")
}

func renderAssistantBlock(content string, textWidth, blockWidth int) string {
	return styleAssistant.Width(blockWidth).Render(wrapPlain(content, textWidth))
}

func renderPlainBlock(content string, textWidth, blockWidth int) string {
	return styleAssistant.Width(blockWidth).Render(wrapPlain(content, textWidth))
}

func wrapPlain(text string, width int) string {
	if text == "" {
		return ""
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, wrapLine(paragraph, width)...)
	}
	return strings.Join(lines, "\n")
}

func wrapLine(line string, width int) []string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}

	var (
		lines      []string
		current    strings.Builder
		currentLen int
	)

	flush := func() {
		if current.Len() > 0 {
			lines = append(lines, current.String())
			current.Reset()
			currentLen = 0
		}
	}

	for _, word := range words {
		wordLen := lipgloss.Width(word)
		if wordLen > width {
			if currentLen > 0 {
				flush()
			}
			for _, chunk := range splitLongWord(word, width) {
				lines = append(lines, chunk)
			}
			continue
		}
		extra := wordLen
		if currentLen > 0 {
			extra++
		}
		if currentLen+extra > width {
			flush()
		}
		if currentLen > 0 {
			current.WriteByte(' ')
			currentLen++
		}
		current.WriteString(word)
		currentLen += wordLen
	}
	flush()
	return lines
}

func splitLongWord(word string, width int) []string {
	var chunks []string
	runes := []rune(word)
	for len(runes) > 0 {
		n := width
		if n > len(runes) {
			n = len(runes)
		}
		chunks = append(chunks, string(runes[:n]))
		runes = runes[n:]
	}
	return chunks
}
