package tui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// WrapText wraps the given text to fit within the specified width.
// It respects existing newlines and wraps on word boundaries where possible.
func WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var sb strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			sb.WriteRune('\n')
		}

		if runewidth.StringWidth(line) <= width {
			sb.WriteString(line)
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}

		currentLineLen := 0
		for _, word := range words {
			wordLen := runewidth.StringWidth(word)

			// If a single word is too long, it might overflow or split (simple behavior here: overflow)
			// A better approach for very long words is to split them, but keeping it simple for now.

			spaceLen := 0
			if currentLineLen > 0 {
				spaceLen = 1
			}

			if currentLineLen+spaceLen+wordLen > width {
				if currentLineLen > 0 {
					sb.WriteRune('\n')
					currentLineLen = 0
				}
			}

			if currentLineLen > 0 {
				sb.WriteRune(' ')
				currentLineLen++
			}

			sb.WriteString(word)
			currentLineLen += wordLen
		}
	}

	return sb.String()
}

// AlignText aligns the given text within the specified width.
// It processes each line of the text individually.
func AlignText(text string, width int, align Alignment) string {
	if width <= 0 {
		return text
	}

	var sb strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			sb.WriteRune('\n')
		}

		lineLen := runewidth.StringWidth(line)
		if lineLen >= width {
			sb.WriteString(line)
			continue
		}

		padding := width - lineLen

		switch align {
		case AlignLeft:
			sb.WriteString(line)
			sb.WriteString(strings.Repeat(" ", padding))
		case AlignCenter:
			leftPad := padding / 2
			rightPad := padding - leftPad
			sb.WriteString(strings.Repeat(" ", leftPad))
			sb.WriteString(line)
			sb.WriteString(strings.Repeat(" ", rightPad))
		case AlignRight:
			sb.WriteString(strings.Repeat(" ", padding))
			sb.WriteString(line)
		}
	}

	return sb.String()
}

// MeasureText returns the width and height (number of lines) of the text.
func MeasureText(text string) (width, height int) {
	lines := strings.Split(text, "\n")
	height = len(lines)
	maxW := 0
	for _, line := range lines {
		w := runewidth.StringWidth(line)
		if w > maxW {
			maxW = w
		}
	}
	return maxW, height
}
