package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/runewidth"
)

func panelTheme() MarkdownTheme {
	theme := DefaultMarkdownTheme()
	theme.CodeBlockStyle = NewStyle().WithBgRGB(RGB{R: 40, G: 40, B: 40})
	return theme
}

func lineWidth(line StyledLine) int {
	w := line.Indent
	for _, seg := range line.Segments {
		w += runewidth.StringWidth(seg.Text)
	}
	return w
}

// assertPanel checks that lines form a solid rectangle in the panel background.
func assertPanel(t *testing.T, lines []StyledLine, indent, width int) {
	t.Helper()
	for i, line := range lines {
		assert.Equal(t, line.Indent, indent, "line %d indent", i)
		assert.Equal(t, lineWidth(line), indent+width, "line %d width", i)
		for _, seg := range line.Segments {
			assert.NotNil(t, seg.Style.BgRGB, "line %d segment %q has no background", i, seg.Text)
			assert.Equal(t, *seg.Style.BgRGB, RGB{R: 40, G: 40, B: 40})
			assert.False(t, strings.ContainsAny(seg.Text, "\n\t"), "line %d segment %q", i, seg.Text)
		}
	}
}

func TestMarkdownRenderer_CodeBlockWithoutBackgroundUnchanged(t *testing.T) {
	renderer := NewMarkdownRenderer().WithMaxWidth(40)
	result, err := renderer.Render("```\nx := 1\n```\n")
	assert.NoError(t, err)
	assert.Equal(t, len(result.Lines), 1)
	assert.Equal(t, result.Lines[0].Indent, 4)
	assert.Equal(t, result.Lines[0].Segments[0].Text, "x := 1\n")
}

func TestMarkdownRenderer_CodeBlockPanelFillsMaxWidth(t *testing.T) {
	renderer := NewMarkdownRenderer().WithTheme(panelTheme()).WithMaxWidth(30)
	result, err := renderer.Render("```\nshort\n\nlonger line\n```\n\nafter")
	assert.NoError(t, err)
	assertPanel(t, result.Lines[:3], 0, 30)
	text := renderToPlainText(result)
	assert.Equal(t, text, " short"+strings.Repeat(" ", 24)+"\n"+strings.Repeat(" ", 30)+"\n"+" longer line"+strings.Repeat(" ", 18)+"\n\nafter\n")
}

func TestMarkdownRenderer_CodeBlockPanelHighlightedKeepsBackground(t *testing.T) {
	renderer := NewMarkdownRenderer().WithTheme(panelTheme()).WithMaxWidth(40)
	result, err := renderer.Render("```go\nfunc main() {\n\tprintln(\"hi\")\n}\n```\n")
	assert.NoError(t, err)
	assert.Equal(t, len(result.Lines), 3)
	assertPanel(t, result.Lines, 0, 40)
	var colored bool
	for _, seg := range result.Lines[0].Segments {
		if seg.Text == "func" && seg.Style.FgRGB != nil {
			colored = true
		}
	}
	assert.True(t, colored, "syntax highlighting foreground was lost")
	assert.Contains(t, renderToPlainText(result), "    println")
}

func TestMarkdownRenderer_CodeBlockPanelWithoutMaxWidthUsesWidestLine(t *testing.T) {
	renderer := NewMarkdownRenderer().WithTheme(panelTheme())
	result, err := renderer.Render("    a\n    abcdef\n")
	assert.NoError(t, err)
	assertPanel(t, result.Lines, 0, 8)
}

func TestMarkdownRenderer_CodeBlockPanelNested(t *testing.T) {
	renderer := NewMarkdownRenderer().WithTheme(panelTheme()).WithMaxWidth(30)
	result, err := renderer.Render("- item\n\n  ```\n  code\n  ```\n")
	assert.NoError(t, err)
	var panel []StyledLine
	for _, line := range result.Lines {
		if len(line.Segments) > 0 && line.Segments[0].Style.BgRGB != nil {
			panel = append(panel, line)
		}
	}
	assert.Equal(t, len(panel), 1)
	assertPanel(t, panel, 2, 28)
}

func TestMarkdownRenderer_CodeBlockPanelWithSourceOffset(t *testing.T) {
	view := Markdown("```go\nx := 1\n```\n", nil).Theme(panelTheme()).SourceOffset(0)
	view.renderContent(20)
	assertPanel(t, view.rendered.Lines, 0, 20)
	assert.True(t, view.rendered.sourceValid)
}

func TestMarkdownRenderer_CodeBlockLanguageLabel(t *testing.T) {
	theme := panelTheme()
	theme.CodeBlockShowLanguage = true
	renderer := NewMarkdownRenderer().WithTheme(theme).WithMaxWidth(20)
	result, err := renderer.Render("```go\nx := 1\n```\n")
	assert.NoError(t, err)
	assertPanel(t, result.Lines, 0, 20)
	assert.Equal(t, renderToPlainText(result), " x := 1          go \n")
	label := result.Lines[0].Segments[len(result.Lines[0].Segments)-2]
	assert.Equal(t, label.Text, "go")
	assert.Equal(t, label.Style.Foreground, ColorBrightBlack)
}

func TestMarkdownRenderer_CodeBlockLanguageLabelWithoutBackground(t *testing.T) {
	theme := DefaultMarkdownTheme()
	theme.CodeBlockShowLanguage = true
	renderer := NewMarkdownRenderer().WithTheme(theme)
	result, err := renderer.Render("```text\nab\nabcdef\n```\n")
	assert.NoError(t, err)
	assert.Equal(t, renderToPlainText(result), "    ab  text\n    abcdef\n")
}

func TestMarkdownRenderer_CodeBlockLanguageLabelSkippedWhenTooWide(t *testing.T) {
	theme := panelTheme()
	theme.CodeBlockShowLanguage = true
	renderer := NewMarkdownRenderer().WithTheme(theme).WithMaxWidth(12)
	result, err := renderer.Render("```python\nprint(1)\n```\n")
	assert.NoError(t, err)
	assert.NotContains(t, renderToPlainText(result), "python")
	assertPanel(t, result.Lines, 0, 12)
}

func TestMarkdownRenderer_CodeBlockPanelUsesBlockForegroundForPlainTokens(t *testing.T) {
	theme := panelTheme()
	theme.CodeBlockStyle = theme.CodeBlockStyle.WithForeground(ColorWhite)
	renderer := NewMarkdownRenderer().WithTheme(theme).WithMaxWidth(30)
	result, err := renderer.Render("```go\nx := 1\n```\n")
	assert.NoError(t, err)
	assertPanel(t, result.Lines, 0, 30)
	for _, seg := range result.Lines[0].Segments {
		if seg.Style.FgRGB == nil && strings.TrimSpace(seg.Text) != "" {
			assert.Equal(t, seg.Style.Foreground, ColorWhite, "token %q", seg.Text)
		}
	}
}
