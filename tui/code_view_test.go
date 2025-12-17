package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestCodeHighlight(t *testing.T) {
	code := `func main() {
    fmt.Println("Hello")
}`
	view := Code(code, "go")
	view.highlight()

	assert.True(t, len(view.highlighted) > 0, "should have highlighted lines")
}

func TestCodeLineNumbers(t *testing.T) {
	view := Code("line1\nline2\nline3", "text")

	// With line numbers
	view.LineNumbers(true)
	w := view.lineNumberWidth()
	assert.True(t, w > 0, "should have line number width")

	// Without line numbers
	view.LineNumbers(false)
	w = view.lineNumberWidth()
	assert.Equal(t, 0, w, "should have no line number width")
}

func TestCodeStartLine(t *testing.T) {
	view := Code("line1\nline2", "text")
	view.StartLine(10)
	assert.Equal(t, 10, view.startLine)
}

func TestCodeTheme(t *testing.T) {
	view := Code("code", "go")
	view.Theme("dracula")
	assert.Equal(t, "dracula", view.theme)
}

func TestCodeSize(t *testing.T) {
	view := Code("hello\nworld", "text").LineNumbers(false)
	w, h := view.size(100, 100)
	assert.Equal(t, 5, w, "width should be max line length")
	assert.Equal(t, 2, h, "height should be number of lines")
}

func TestCodeSizeWithLineNumbers(t *testing.T) {
	view := Code("hello\nworld", "text").LineNumbers(true)
	w, h := view.size(100, 100)
	assert.True(t, w > 5, "width should include line numbers")
	assert.Equal(t, 2, h)
}

func TestCodeFixedHeight(t *testing.T) {
	view := Code("1\n2\n3\n4\n5", "text").Height(3)
	_, h := view.size(100, 100)
	assert.Equal(t, 3, h)
}

func TestCodeGetLineCount(t *testing.T) {
	view := Code("a\nb\nc\nd", "text")
	assert.Equal(t, 4, view.GetLineCount())
}

func TestAvailableThemes(t *testing.T) {
	themes := AvailableThemes()
	assert.True(t, len(themes) > 0, "should have themes available")

	// Check for common themes
	found := false
	for _, theme := range themes {
		if theme == "monokai" {
			found = true
			break
		}
	}
	assert.True(t, found, "monokai theme should be available")
}

func TestAvailableLanguages(t *testing.T) {
	langs := AvailableLanguages()
	assert.True(t, len(langs) > 0, "should have languages available")
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		n        int
		width    int
		expected string
	}{
		{1, 3, "  1"},
		{10, 3, " 10"},
		{100, 3, "100"},
		{1000, 3, "1000"},
		{0, 2, " 0"},
	}

	for _, tt := range tests {
		result := padLeft(tt.n, tt.width)
		assert.Equal(t, tt.expected, result, "padLeft(%d, %d)", tt.n, tt.width)
	}
}

func TestTruncateToWidth(t *testing.T) {
	tests := []struct {
		input    string
		maxWidth int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 3, "hel"},
		{"hello", 0, ""},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncateToWidth(tt.input, tt.maxWidth)
		assert.Equal(t, tt.expected, result, "truncateToWidth(%q, %d)", tt.input, tt.maxWidth)
	}
}
