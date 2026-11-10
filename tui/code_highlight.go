package tui

import (
	"sync/atomic"

	"github.com/deepnoodle-ai/wonton/runewidth"
)

// CodeHighlighter colors the code of a fenced Markdown code block. It returns
// one slice of segments for each line of code, without the newlines. theme is
// MarkdownTheme.SyntaxTheme. It returns nil when it does not know language;
// the block then uses MarkdownTheme.CodeBlockStyle.
type CodeHighlighter func(code, language, theme string) [][]StyledSegment

var codeHighlighter atomic.Pointer[CodeHighlighter]

// SetCodeHighlighter sets the highlighter that Markdown and MarkdownRenderer
// use for fenced code blocks that name a language. With none set, code blocks
// use MarkdownTheme.CodeBlockStyle. nil removes the highlighter.
//
// The core tui package has no syntax highlighter, so that programs that do
// not show code stay small. Import the chroma-based highlighter for its side
// effect to color code blocks:
//
//	import _ "github.com/deepnoodle-ai/wonton/tui/highlight"
func SetCodeHighlighter(h CodeHighlighter) {
	if h == nil {
		codeHighlighter.Store(nil)
		return
	}
	codeHighlighter.Store(&h)
}

// truncateToWidth truncates a string to fit within a given display width.
// It never splits a grapheme cluster.
func truncateToWidth(s string, maxWidth int) string {
	result, _ := runewidth.FitLeft(s, maxWidth)
	return result
}
