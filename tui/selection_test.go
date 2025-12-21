package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTextPosition_Before(t *testing.T) {
	tests := []struct {
		name     string
		p        TextPosition
		other    TextPosition
		expected bool
	}{
		{"same position", TextPosition{0, 0}, TextPosition{0, 0}, false},
		{"before on same line", TextPosition{0, 5}, TextPosition{0, 10}, true},
		{"after on same line", TextPosition{0, 10}, TextPosition{0, 5}, false},
		{"earlier line", TextPosition{0, 5}, TextPosition{1, 0}, true},
		{"later line", TextPosition{1, 5}, TextPosition{0, 10}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.Before(tt.other)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTextSelection_Normalized(t *testing.T) {
	// Forward selection
	sel := TextSelection{
		Start:  TextPosition{0, 5},
		End:    TextPosition{2, 10},
		Active: true,
	}
	start, end := sel.Normalized()
	assert.Equal(t, TextPosition{0, 5}, start)
	assert.Equal(t, TextPosition{2, 10}, end)

	// Backward selection
	sel = TextSelection{
		Start:  TextPosition{2, 10},
		End:    TextPosition{0, 5},
		Active: true,
	}
	start, end = sel.Normalized()
	assert.Equal(t, TextPosition{0, 5}, start)
	assert.Equal(t, TextPosition{2, 10}, end)
}

func TestTextSelection_IsEmpty(t *testing.T) {
	// Not active
	sel := TextSelection{Active: false}
	assert.True(t, sel.IsEmpty())

	// Active but same position
	sel = TextSelection{
		Start:  TextPosition{0, 5},
		End:    TextPosition{0, 5},
		Active: true,
	}
	assert.True(t, sel.IsEmpty())

	// Active with different positions
	sel = TextSelection{
		Start:  TextPosition{0, 5},
		End:    TextPosition{0, 10},
		Active: true,
	}
	assert.False(t, sel.IsEmpty())
}

func TestTextSelection_Contains(t *testing.T) {
	sel := TextSelection{
		Start:  TextPosition{1, 5},
		End:    TextPosition{3, 10},
		Active: true,
	}

	tests := []struct {
		name     string
		pos      TextPosition
		expected bool
	}{
		{"before selection", TextPosition{0, 0}, false},
		{"at start", TextPosition{1, 5}, true},
		{"middle of first line", TextPosition{1, 8}, true},
		{"middle line", TextPosition{2, 5}, true},
		{"last line before end", TextPosition{3, 5}, true},
		{"at end (exclusive)", TextPosition{3, 10}, false},
		{"after selection", TextPosition{4, 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sel.Contains(tt.pos)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTextSelection_LineRange(t *testing.T) {
	sel := TextSelection{
		Start:  TextPosition{1, 5},
		End:    TextPosition{3, 10},
		Active: true,
	}

	tests := []struct {
		name      string
		line      int
		lineLen   int
		wantStart int
		wantEnd   int
	}{
		{"before selection", 0, 20, 0, 0},
		{"first line", 1, 20, 5, 20},      // from col 5 to end of line
		{"middle line", 2, 15, 0, 15},     // entire line
		{"last line", 3, 20, 0, 10},       // from start to col 10
		{"after selection", 4, 20, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := sel.LineRange(tt.line, tt.lineLen)
			assert.Equal(t, tt.wantStart, start, "start column")
			assert.Equal(t, tt.wantEnd, end, "end column")
		})
	}
}

func TestExtractSelectedText_SingleLine(t *testing.T) {
	lines := []string{
		"Hello, world!",
		"Second line",
		"Third line",
	}

	sel := TextSelection{
		Start:  TextPosition{0, 7},
		End:    TextPosition{0, 12},
		Active: true,
	}

	result := ExtractSelectedText(lines, sel)
	assert.Equal(t, "world", result)
}

func TestExtractSelectedText_MultiLine(t *testing.T) {
	lines := []string{
		"First line",
		"Second line",
		"Third line",
	}

	sel := TextSelection{
		Start:  TextPosition{0, 6},
		End:    TextPosition{2, 5},
		Active: true,
	}

	result := ExtractSelectedText(lines, sel)
	expected := "line\nSecond line\nThird"
	assert.Equal(t, expected, result)
}

func TestExtractSelectedText_EmptySelection(t *testing.T) {
	lines := []string{"Hello, world!"}

	// Not active
	sel := TextSelection{Active: false}
	assert.Equal(t, "", ExtractSelectedText(lines, sel))

	// Same position
	sel = TextSelection{
		Start:  TextPosition{0, 5},
		End:    TextPosition{0, 5},
		Active: true,
	}
	assert.Equal(t, "", ExtractSelectedText(lines, sel))
}

func TestExtractSelectedText_BackwardSelection(t *testing.T) {
	lines := []string{"Hello, world!"}

	// End before Start (backward drag)
	sel := TextSelection{
		Start:  TextPosition{0, 12},
		End:    TextPosition{0, 7},
		Active: true,
	}

	result := ExtractSelectedText(lines, sel)
	assert.Equal(t, "world", result)
}

func TestSelectWord(t *testing.T) {
	lines := []string{
		"Hello, world!",
		"foo_bar123 test",
	}

	tests := []struct {
		name      string
		pos       TextPosition
		wantStart int
		wantEnd   int
	}{
		{"word at start", TextPosition{0, 2}, 0, 5},      // "Hello"
		{"word after comma", TextPosition{0, 8}, 7, 12}, // "world"
		{"underscore word", TextPosition{1, 4}, 0, 10},  // "foo_bar123"
		{"after underscore", TextPosition{1, 12}, 11, 15}, // "test"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sel := SelectWord(lines, tt.pos)
			assert.True(t, sel.Active)
			assert.Equal(t, tt.pos.Line, sel.Start.Line)
			assert.Equal(t, tt.wantStart, sel.Start.Col, "start col")
			assert.Equal(t, tt.wantEnd, sel.End.Col, "end col")
		})
	}
}

func TestSelectLine(t *testing.T) {
	lines := []string{
		"First line",
		"Second line",
		"Third line",
	}

	sel := SelectLine(lines, 1)
	assert.True(t, sel.Active)
	assert.Equal(t, 1, sel.Start.Line)
	assert.Equal(t, 0, sel.Start.Col)
	assert.Equal(t, 1, sel.End.Line)
	assert.Equal(t, 11, sel.End.Col) // len("Second line")

	// Invalid line
	sel = SelectLine(lines, 10)
	assert.False(t, sel.Active)
}

func TestScreenToTextPosition(t *testing.T) {
	lines := []string{
		"Hello",
		"World",
		"Test",
	}

	tests := []struct {
		name            string
		screenX, screenY int
		scrollY          int
		lineNumWidth     int
		wantLine         int
		wantCol          int
	}{
		{"first char", 0, 0, 0, 0, 0, 0},
		{"middle of first line", 3, 0, 0, 0, 0, 3},
		{"second line", 2, 1, 0, 0, 1, 2},
		{"with scroll", 2, 0, 1, 0, 1, 2},      // scrolled down 1 line
		{"with line numbers", 5, 0, 0, 3, 0, 2}, // 3-char line number width
		{"beyond line end", 10, 0, 0, 0, 0, 5},  // clamps to line length
		{"beyond last line", 0, 10, 0, 0, 2, 0}, // clamps to last line
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := ScreenToTextPosition(tt.screenX, tt.screenY, tt.scrollY, tt.lineNumWidth, lines)
			assert.Equal(t, tt.wantLine, pos.Line, "line")
			assert.Equal(t, tt.wantCol, pos.Col, "col")
		})
	}
}

func TestScreenXToByteOffset_WideChars(t *testing.T) {
	// Test with wide characters (emoji, CJK)
	s := "Hello 🌍 World"

	// 'H' is at screen X 0
	assert.Equal(t, 0, screenXToByteOffset(s, 0))

	// ' ' before emoji is at screen X 5
	assert.Equal(t, 5, screenXToByteOffset(s, 5))

	// Emoji takes 2 screen columns, so 'W' starts at screen X 9
	// (Hello=5, space=1, emoji=2, space=1 = 9)
	offset := screenXToByteOffset(s, 9)
	// Skip the emoji bytes (4 bytes for 🌍) + spaces
	assert.Equal(t, byte('W'), s[offset])
}
