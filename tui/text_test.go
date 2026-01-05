package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
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

// Render tests using termtest with SprintScreen helper

func TestText_Render_Simple(t *testing.T) {
	v := Text("Hello, World!")
	screen := SprintScreen(v, PrintConfig{Width: 30})
	termtest.AssertRow(t, screen, 0, "Hello, World!")
}

func TestText_Render_Multiline(t *testing.T) {
	v := Text("Line 1\nLine 2\nLine 3")
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRow(t, screen, 0, "Line 1")
	termtest.AssertRow(t, screen, 1, "Line 2")
	termtest.AssertRow(t, screen, 2, "Line 3")
}

func TestText_Render_Bold(t *testing.T) {
	v := Text("Bold Text").Bold()
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRowContains(t, screen, 0, "Bold Text")

	// Check that text has bold style
	cell := screen.Cell(0, 0)
	assert.True(t, cell.Style.Bold)
}

func TestText_Render_Italic(t *testing.T) {
	v := Text("Italic Text").Italic()
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRowContains(t, screen, 0, "Italic Text")

	cell := screen.Cell(0, 0)
	assert.True(t, cell.Style.Italic)
}

func TestText_Render_Underline(t *testing.T) {
	v := Text("Underlined").Underline()
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRowContains(t, screen, 0, "Underlined")

	cell := screen.Cell(0, 0)
	assert.True(t, cell.Style.Underline)
}

func TestText_Render_Dim(t *testing.T) {
	v := Text("Dimmed").Dim()
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRowContains(t, screen, 0, "Dimmed")

	cell := screen.Cell(0, 0)
	assert.True(t, cell.Style.Dim)
}

func TestText_Render_ForegroundColor(t *testing.T) {
	v := Text("Red").Fg(ColorRed)
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRowContains(t, screen, 0, "Red")

	// Check foreground color is set
	cell := screen.Cell(0, 0)
	assert.Equal(t, termtest.ColorBasic, cell.Style.Foreground.Type)
}

func TestText_Render_BackgroundColor(t *testing.T) {
	v := Text("Highlighted").Bg(ColorYellow)
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRowContains(t, screen, 0, "Highlighted")

	// Check background color is set
	cell := screen.Cell(0, 0)
	assert.Equal(t, termtest.ColorBasic, cell.Style.Background.Type)
}

func TestText_Render_CombinedStyles(t *testing.T) {
	v := Text("Styled").Bold().Italic().Underline().Fg(ColorCyan)
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRowContains(t, screen, 0, "Styled")

	cell := screen.Cell(0, 0)
	assert.True(t, cell.Style.Bold)
	assert.True(t, cell.Style.Italic)
	assert.True(t, cell.Style.Underline)
}

func TestText_Render_Printf(t *testing.T) {
	v := Text("Count: %d, Name: %s", 42, "Test")
	screen := SprintScreen(v, PrintConfig{Width: 40})

	termtest.AssertRowContains(t, screen, 0, "Count: 42")
	termtest.AssertRowContains(t, screen, 0, "Name: Test")
}

func TestText_Render_WideCharacters(t *testing.T) {
	v := Text("日本語")
	screen := SprintScreen(v, PrintConfig{Width: 20})

	termtest.AssertRowContains(t, screen, 0, "日本語")
}

func TestText_Render_Emoji(t *testing.T) {
	v := Text("Hello 🎉 World")
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRowContains(t, screen, 0, "Hello")
	termtest.AssertRowContains(t, screen, 0, "World")
}

func TestText_Render_WrappingDisabled(t *testing.T) {
	// Text without explicit wrapping should not wrap
	v := Text("Short text that fits")
	screen := SprintScreen(v, PrintConfig{Width: 30})

	termtest.AssertRow(t, screen, 0, "Short text that fits")
}

func TestText_Render_Empty(t *testing.T) {
	v := Text("")
	screen := SprintScreen(v, PrintConfig{Width: 20})

	// Empty text should render without error
	termtest.AssertRow(t, screen, 0, "")
}

func TestWrappedText_NoLeadingSpace(t *testing.T) {
	// Test WrapText directly first
	wrapped := WrapText("hello world foo", 10)
	expected := "hello\nworld foo"
	if wrapped != expected {
		t.Errorf("WrapText mismatch:\ngot:  %q\nwant: %q", wrapped, expected)
	}

	// Check that wrapped lines don't start with spaces
	lines := strings.Split(wrapped, "\n")
	for i, line := range lines {
		if len(line) > 0 && line[0] == ' ' {
			t.Errorf("WrapText line %d starts with space: %q", i, line)
		}
	}
}

func TestWrappedText_MultipleWraps(t *testing.T) {
	// Test WrapText with more wraps
	wrapped := WrapText("one two three four five", 10)
	lines := strings.Split(wrapped, "\n")

	for i, line := range lines {
		if len(line) > 0 && line[0] == ' ' {
			t.Errorf("WrapText line %d starts with space: %q", i, line)
		}
	}

	// Verify all words are present
	if !strings.Contains(wrapped, "one") || !strings.Contains(wrapped, "five") {
		t.Errorf("Missing content in wrapped text: %q", wrapped)
	}
}
