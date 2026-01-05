package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestFill_Creation(t *testing.T) {
	fill := Fill('*')
	assert.NotNil(t, fill)
	assert.Equal(t, '*', fill.char)
}

func TestFill_CreationWithDifferentCharacters(t *testing.T) {
	tests := []struct {
		name string
		char rune
	}{
		{"space", ' '},
		{"asterisk", '*'},
		{"hash", '#'},
		{"equals", '='},
		{"dot", '.'},
		{"unicode block", '█'},
		{"unicode shade", '░'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fill := Fill(tt.char)
			assert.Equal(t, tt.char, fill.char)
		})
	}
}

func TestFill_Fg(t *testing.T) {
	fill := Fill('*').Fg(ColorRed)
	assert.NotNil(t, fill)
	// Verify the style has been set (we can't directly inspect the color,
	// but we can verify the method returns the same instance for chaining)
	assert.NotNil(t, fill.style)
}

func TestFill_FgRGB(t *testing.T) {
	fill := Fill('*').FgRGB(255, 128, 64)
	assert.NotNil(t, fill)
	assert.NotNil(t, fill.style)
}

func TestFill_Bg(t *testing.T) {
	fill := Fill('*').Bg(ColorBlue)
	assert.NotNil(t, fill)
	assert.NotNil(t, fill.style)
}

func TestFill_BgRGB(t *testing.T) {
	fill := Fill('*').BgRGB(64, 128, 255)
	assert.NotNil(t, fill)
	assert.NotNil(t, fill.style)
}

func TestFill_Style(t *testing.T) {
	customStyle := NewStyle().WithForeground(ColorGreen).WithBold()
	fill := Fill('*').Style(customStyle)
	assert.NotNil(t, fill)
	assert.Equal(t, customStyle, fill.style)
}

func TestFill_MethodChaining(t *testing.T) {
	// Test that all methods return the same instance for chaining
	fill := Fill('*')
	result := fill.Fg(ColorRed).Bg(ColorBlue).FgRGB(255, 0, 0).BgRGB(0, 0, 255)
	assert.Equal(t, fill, result)
}

func TestFill_Size(t *testing.T) {
	tests := []struct {
		name      string
		maxWidth  int
		maxHeight int
		wantW     int
		wantH     int
	}{
		{"zero dimensions", 0, 0, 0, 0},
		{"width only", 10, 0, 10, 0},
		{"height only", 0, 5, 0, 5},
		{"small area", 10, 5, 10, 5},
		{"medium area", 40, 20, 40, 20},
		{"large area", 100, 50, 100, 50},
		{"tall and narrow", 5, 100, 5, 100},
		{"wide and short", 100, 5, 100, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fill := Fill('*')
			w, h := fill.size(tt.maxWidth, tt.maxHeight)
			assert.Equal(t, tt.wantW, w)
			assert.Equal(t, tt.wantH, h)
		})
	}
}

func TestFill_Flex(t *testing.T) {
	fill := Fill('*')
	flex := fill.flex()
	assert.Equal(t, 1, flex)
}

func TestFill_FlexWithDifferentConfigurations(t *testing.T) {
	// Fill should always be flexible regardless of configuration
	tests := []struct {
		name string
		fill *fillView
	}{
		{"basic fill", Fill('*')},
		{"styled fill", Fill('*').Fg(ColorRed)},
		{"background fill", Fill(' ').Bg(ColorBlue)},
		{"custom style", Fill('#').Style(NewStyle().WithBold())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flex := tt.fill.flex()
			assert.Equal(t, 1, flex)
		})
	}
}

func TestFill_Render(t *testing.T) {
	var buf strings.Builder
	fill := Fill('*')

	err := Print(fill, PrintConfig{Width: 10, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain asterisks
	assert.True(t, strings.Contains(output, "*"), "output should contain fill character")
}

func TestFill_RenderWithColor(t *testing.T) {
	var buf strings.Builder
	fill := Fill('*').Fg(ColorRed)

	err := Print(fill, PrintConfig{Width: 10, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI escape codes and the fill character
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape code")
	assert.True(t, strings.Contains(output, "*"), "output should contain fill character")
}

func TestFill_RenderWithBackground(t *testing.T) {
	var buf strings.Builder
	fill := Fill(' ').Bg(ColorBlue)

	err := Print(fill, PrintConfig{Width: 10, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI escape codes for background color
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape code")
}

func TestFill_RenderWithBothColors(t *testing.T) {
	var buf strings.Builder
	fill := Fill('█').Fg(ColorGreen).Bg(ColorBlue)

	err := Print(fill, PrintConfig{Width: 10, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI escape codes
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape code")
	assert.True(t, strings.Contains(output, "█"), "output should contain fill character")
}

func TestFill_RenderWithRGB(t *testing.T) {
	var buf strings.Builder
	fill := Fill('*').FgRGB(255, 128, 0).BgRGB(0, 64, 128)

	err := Print(fill, PrintConfig{Width: 10, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI escape codes
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape code")
	assert.True(t, strings.Contains(output, "*"), "output should contain fill character")
}

func TestFill_RenderEmpty(t *testing.T) {
	var buf strings.Builder
	fill := Fill('*')

	// Zero width
	err := Print(fill, PrintConfig{Width: 0, Height: 5, Output: &buf})
	assert.NoError(t, err)

	// Zero height
	buf.Reset()
	err = Print(fill, PrintConfig{Width: 10, Height: 0, Output: &buf})
	assert.NoError(t, err)

	// Both zero
	buf.Reset()
	err = Print(fill, PrintConfig{Width: 0, Height: 0, Output: &buf})
	assert.NoError(t, err)
}

func TestFill_RenderDifferentCharacters(t *testing.T) {
	tests := []struct {
		name string
		char rune
	}{
		{"space", ' '},
		{"asterisk", '*'},
		{"hash", '#'},
		{"equals", '='},
		{"dash", '-'},
		{"underscore", '_'},
		{"unicode block", '█'},
		{"unicode shade light", '░'},
		{"unicode shade medium", '▒'},
		{"unicode shade dark", '▓'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			fill := Fill(tt.char)

			err := Print(fill, PrintConfig{Width: 10, Height: 5, Output: &buf})
			assert.NoError(t, err)

			output := buf.String()
			// For non-space characters, verify they appear in output
			if tt.char != ' ' {
				assert.True(t, strings.ContainsRune(output, tt.char),
					"output should contain fill character")
			}
		})
	}
}

func TestFill_InStack(t *testing.T) {
	var buf strings.Builder
	view := Stack(
		Text("Header"),
		Fill('-'),
		Text("Footer"),
	)

	err := Print(view, PrintConfig{Width: 20, Height: 10, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Header"), "should contain header")
	assert.True(t, strings.Contains(output, "-"), "should contain fill character")
	assert.True(t, strings.Contains(output, "Footer"), "should contain footer")
}

func TestFill_MultipleInStack(t *testing.T) {
	var buf strings.Builder
	view := Stack(
		Text("Section 1"),
		Fill('-'),
		Text("Section 2"),
		Fill('='),
		Text("Section 3"),
	)

	err := Print(view, PrintConfig{Width: 20, Height: 15, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Section 1"), "should contain section 1")
	assert.True(t, strings.Contains(output, "-"), "should contain first fill character")
	assert.True(t, strings.Contains(output, "Section 2"), "should contain section 2")
	assert.True(t, strings.Contains(output, "="), "should contain second fill character")
	assert.True(t, strings.Contains(output, "Section 3"), "should contain section 3")
}

func TestFill_Spacing(t *testing.T) {
	var buf strings.Builder
	view := Stack(
		Text("Line 1"),
		Fill(' ').Bg(ColorBrightBlack),
		Text("Line 2"),
	)

	err := Print(view, PrintConfig{Width: 20, Height: 10, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Line 1"), "should contain first line")
	assert.True(t, strings.Contains(output, "Line 2"), "should contain second line")
}

func TestFill_AsBackground(t *testing.T) {
	// Fill can be used as a colored background
	var buf strings.Builder
	fill := Fill(' ').BgRGB(32, 32, 32)

	err := Print(fill, PrintConfig{Width: 20, Height: 10, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI escape codes
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape code")
}

func TestFill_CustomStyle(t *testing.T) {
	var buf strings.Builder
	customStyle := NewStyle().
		WithForeground(ColorYellow).
		WithBackground(ColorMagenta).
		WithBold()
	fill := Fill('*').Style(customStyle)

	err := Print(fill, PrintConfig{Width: 10, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape code")
	assert.True(t, strings.Contains(output, "*"), "output should contain fill character")
}

func TestFill_LargeDimensions(t *testing.T) {
	var buf strings.Builder
	fill := Fill('.')

	err := Print(fill, PrintConfig{Width: 100, Height: 50, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, len(output) > 0, "output should not be empty for large dimensions")
}

func TestFill_SingleCell(t *testing.T) {
	var buf strings.Builder
	fill := Fill('X')

	err := Print(fill, PrintConfig{Width: 1, Height: 1, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "X"), "output should contain fill character")
}
