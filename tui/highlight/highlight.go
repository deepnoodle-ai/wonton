// Package highlight provides syntax-highlighted views for tui: CodeView for
// source code and UnifiedDiffView for unified diffs. It uses chroma, which
// adds several megabytes to a binary, so it lives outside the core tui
// package.
//
// Importing the package also sets the tui Markdown code highlighter, so
// fenced code blocks in tui.Markdown are colored by language. A program that
// only wants that effect imports it for its side effect:
//
//	import _ "github.com/deepnoodle-ai/wonton/tui/highlight"
//
// Example:
//
//	view := tui.Stack(
//	    tui.Text("main.go").Bold(),
//	    highlight.Code(src, "go").Theme("dracula"),
//	)
package highlight

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/deepnoodle-ai/wonton/tui"
)

func init() {
	tui.SetCodeHighlighter(Markdown)
}

// view lets CodeView and UnifiedDiffView embed a tui.View without exporting
// the field.
type view = tui.View

// Markdown is the tui.CodeHighlighter that this package sets for tui
// Markdown code blocks. It returns nil when chroma has no lexer for
// language. An unknown theme uses chroma's fallback style.
func Markdown(code, language, theme string) [][]tui.StyledSegment {
	lexer := lexers.Get(language)
	if lexer == nil {
		return nil
	}
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return nil
	}
	chromaStyle := chromaTheme(theme)

	var result [][]tui.StyledSegment
	var line []tui.StyledSegment
	for _, token := range iterator.Tokens() {
		style := tokenStyle(chromaStyle.Get(token.Type))
		for i, part := range strings.Split(token.Value, "\n") {
			if i > 0 {
				result = append(result, line)
				line = []tui.StyledSegment{}
			}
			if part != "" {
				line = append(line, tui.StyledSegment{Text: part, Style: style})
			}
		}
	}
	if len(line) > 0 {
		result = append(result, line)
	}
	return result
}

// AvailableThemes returns the names of the syntax highlighting themes.
func AvailableThemes() []string {
	return styles.Names()
}

// AvailableLanguages returns the names of the languages that can be
// highlighted.
func AvailableLanguages() []string {
	return lexers.Names(false)
}

// chromaTheme returns the named chroma style, or chroma's fallback style.
func chromaTheme(name string) *chroma.Style {
	if style := styles.Get(name); style != nil {
		return style
	}
	return styles.Fallback
}

// tokenStyle converts a chroma style entry to a tui.Style. It keeps the
// foreground and text attributes and drops the background, so the terminal
// or panel background shows through.
func tokenStyle(entry chroma.StyleEntry) tui.Style {
	style := tui.NewStyle()
	if entry.Colour.IsSet() {
		style = style.WithFgRGB(tui.RGB{
			R: entry.Colour.Red(),
			G: entry.Colour.Green(),
			B: entry.Colour.Blue(),
		})
	}
	if entry.Bold == chroma.Yes {
		style = style.WithBold()
	}
	if entry.Italic == chroma.Yes {
		style = style.WithItalic()
	}
	if entry.Underline == chroma.Yes {
		style = style.WithUnderline()
	}
	return style
}
