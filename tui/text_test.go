package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{"empty", "", 10, ""},
		{"width zero", "hello world", 0, "hello world"},
		{"negative width", "hello world", -1, "hello world"},
		{"fits exactly", "hello", 5, "hello"},
		{"shorter than width", "hi", 10, "hi"},
		{"wrap single line", "hello world", 6, "hello\nworld"},
		{"wrap multiple words", "one two three four", 8, "one two\nthree\nfour"},
		{"preserves newlines", "hello\nworld", 20, "hello\nworld"},
		{"wrap with newlines", "hello world\nfoo bar", 6, "hello\nworld\nfoo\nbar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapText(tt.text, tt.width)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlignText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		align    Alignment
		expected string
	}{
		{"empty left", "", 10, AlignLeft, "          "}, // empty line gets padded
		{"width zero", "hello", 0, AlignLeft, "hello"},
		{"negative width", "hello", -1, AlignCenter, "hello"},
		{"left align", "hi", 5, AlignLeft, "hi   "},
		{"center align", "hi", 6, AlignCenter, "  hi  "},
		{"center align odd", "hi", 5, AlignCenter, " hi  "},
		{"right align", "hi", 5, AlignRight, "   hi"},
		{"longer than width", "hello", 3, AlignCenter, "hello"},
		{"multiline left", "a\nb", 3, AlignLeft, "a  \nb  "},
		{"multiline center", "a\nb", 3, AlignCenter, " a \n b "},
		{"multiline right", "a\nb", 3, AlignRight, "  a\n  b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AlignText(tt.text, tt.width, tt.align)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMeasureText(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedWidth  int
		expectedHeight int
	}{
		{"empty", "", 0, 1},
		{"single line", "hello", 5, 1},
		{"multiple lines", "hello\nworld", 5, 2},
		{"varying lengths", "a\nab\nabc", 3, 3},
		{"empty line in middle", "a\n\nb", 1, 3},
		{"wide characters", "日本語", 6, 1}, // CJK characters are 2 wide
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := MeasureText(tt.text)
			assert.Equal(t, tt.expectedWidth, width)
			assert.Equal(t, tt.expectedHeight, height)
		})
	}
}
