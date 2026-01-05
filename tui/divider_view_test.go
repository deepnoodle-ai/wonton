package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestDivider_Creation(t *testing.T) {
	d := Divider()
	assert.NotNil(t, d)
	assert.Equal(t, '─', d.char)
	assert.Equal(t, "", d.title)
}

func TestDivider_Char(t *testing.T) {
	tests := []struct {
		name string
		char rune
	}{
		{"equals sign", '═'},
		{"asterisk", '*'},
		{"dash", '-'},
		{"underscore", '_'},
		{"tilde", '~'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Divider().Char(tt.char)
			assert.Equal(t, tt.char, d.char)
		})
	}
}

func TestDivider_Char_Chaining(t *testing.T) {
	d := Divider().Char('═')
	assert.NotNil(t, d)
	assert.Equal(t, '═', d.char)
}

func TestDivider_Fg(t *testing.T) {
	d := Divider().Fg(ColorRed)
	assert.NotNil(t, d)
	assert.Equal(t, ColorRed, d.style.Foreground)
}

func TestDivider_Fg_Chaining(t *testing.T) {
	d := Divider().Fg(ColorBlue).Fg(ColorGreen)
	assert.Equal(t, ColorGreen, d.style.Foreground)
}

func TestDivider_Style(t *testing.T) {
	customStyle := NewStyle().WithForeground(ColorYellow).WithBold()
	d := Divider().Style(customStyle)

	assert.NotNil(t, d)
	assert.Equal(t, ColorYellow, d.style.Foreground)
	assert.True(t, d.style.Bold)
}

func TestDivider_Style_Chaining(t *testing.T) {
	style1 := NewStyle().WithForeground(ColorRed)
	style2 := NewStyle().WithForeground(ColorBlue).WithDim()

	d := Divider().Style(style1).Style(style2)
	assert.Equal(t, ColorBlue, d.style.Foreground)
	assert.True(t, d.style.Dim)
}

func TestDivider_Title(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"simple title", "Section"},
		{"with spaces", "My Section"},
		{"empty string", ""},
		{"unicode", "日本語"},
		{"long title", "This is a very long section title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Divider().Title(tt.title)
			assert.Equal(t, tt.title, d.title)
		})
	}
}

func TestDivider_Title_Chaining(t *testing.T) {
	d := Divider().Title("First").Title("Second")
	assert.Equal(t, "Second", d.title)
}

func TestDivider_Bold(t *testing.T) {
	d := Divider().Bold()
	assert.NotNil(t, d)
	assert.True(t, d.style.Bold)
}

func TestDivider_Bold_Chaining(t *testing.T) {
	d := Divider().Bold().Fg(ColorRed)
	assert.True(t, d.style.Bold)
	assert.Equal(t, ColorRed, d.style.Foreground)
}

func TestDivider_Dim(t *testing.T) {
	d := Divider().Dim()
	assert.NotNil(t, d)
	assert.True(t, d.style.Dim)
}

func TestDivider_Dim_Chaining(t *testing.T) {
	d := Divider().Dim().Fg(ColorBlue)
	assert.True(t, d.style.Dim)
	assert.Equal(t, ColorBlue, d.style.Foreground)
}

func TestDivider_Bold_And_Dim(t *testing.T) {
	d := Divider().Bold().Dim()
	assert.True(t, d.style.Bold)
	assert.True(t, d.style.Dim)
}

func TestDivider_MethodChaining(t *testing.T) {
	d := Divider().
		Char('═').
		Fg(ColorCyan).
		Title("Test Section").
		Bold()

	assert.Equal(t, '═', d.char)
	assert.Equal(t, ColorCyan, d.style.Foreground)
	assert.Equal(t, "Test Section", d.title)
	assert.True(t, d.style.Bold)
}

func TestDivider_NotFlexible(t *testing.T) {
	d := Divider()
	// Dividers should NOT implement Flexible - they have fixed height (1 row)
	// and fill width via size(), not flex distribution.
	// This matches CSS behavior where <hr> has flex-grow: 0.
	_, ok := interface{}(d).(Flexible)
	assert.False(t, ok, "dividerView should not implement Flexible")
}

func TestDivider_Size_NoTitle(t *testing.T) {
	d := Divider()

	t.Run("with max width", func(t *testing.T) {
		w, h := d.size(80, 10)
		assert.Equal(t, 80, w)
		assert.Equal(t, 1, h)
	})

	t.Run("zero max width", func(t *testing.T) {
		w, h := d.size(0, 10)
		assert.Equal(t, 1, w)
		assert.Equal(t, 1, h)
	})

	t.Run("small max width", func(t *testing.T) {
		w, h := d.size(5, 10)
		assert.Equal(t, 5, w)
		assert.Equal(t, 1, h)
	})
}

func TestDivider_Size_WithTitle(t *testing.T) {
	d := Divider().Title("Section")

	t.Run("with max width", func(t *testing.T) {
		w, h := d.size(80, 10)
		assert.Equal(t, 80, w)
		assert.Equal(t, 1, h)
	})

	t.Run("zero max width", func(t *testing.T) {
		w, h := d.size(0, 10)
		// Should request width based on title + padding
		assert.True(t, w > 7) // "Section" is 7 chars + 4 padding
		assert.Equal(t, 1, h)
	})
}

func TestDivider_Render_SimpleLine(t *testing.T) {
	var buf strings.Builder
	d := Divider()

	err := Print(d, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain the divider character
	assert.True(t, strings.Contains(output, "─"), "output should contain divider character")
}

func TestDivider_Render_CustomChar(t *testing.T) {
	var buf strings.Builder
	d := Divider().Char('═')

	err := Print(d, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "═"), "output should contain custom divider character")
}

func TestDivider_Render_WithTitle(t *testing.T) {
	var buf strings.Builder
	d := Divider().Title("Section")

	err := Print(d, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain both the title and divider characters
	assert.True(t, strings.Contains(output, "Section"), "output should contain title text")
	assert.True(t, strings.Contains(output, "─"), "output should contain divider character")
}

func TestDivider_Render_WithTitle_NarrowWidth(t *testing.T) {
	var buf strings.Builder
	d := Divider().Title("Very Long Title That Won't Fit")

	err := Print(d, PrintConfig{Width: 10, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain truncated title
	assert.True(t, len(output) > 0, "output should not be empty")
}

func TestDivider_Render_Colored(t *testing.T) {
	var buf strings.Builder
	d := Divider().Fg(ColorRed)

	err := Print(d, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI color escape codes
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape codes")
}

func TestDivider_Render_Bold(t *testing.T) {
	var buf strings.Builder
	d := Divider().Bold()

	err := Print(d, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI bold escape code
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape codes")
}

func TestDivider_Render_Dim(t *testing.T) {
	var buf strings.Builder
	d := Divider().Dim()

	err := Print(d, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI dim escape code
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape codes")
}

func TestDivider_Render_ZeroWidth(t *testing.T) {
	var buf strings.Builder
	d := Divider()

	err := Print(d, PrintConfig{Width: 0, Output: &buf})
	assert.NoError(t, err)
	// Should not panic with zero width
}

func TestDivider_Render_InStack(t *testing.T) {
	var buf strings.Builder
	view := Stack(
		Text("Header"),
		Divider(),
		Text("Content"),
		Divider().Char('═'),
		Text("Footer"),
	)

	err := Print(view, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Header"), "should contain header")
	assert.True(t, strings.Contains(output, "Content"), "should contain content")
	assert.True(t, strings.Contains(output, "Footer"), "should contain footer")
	assert.True(t, strings.Contains(output, "─"), "should contain default divider")
	assert.True(t, strings.Contains(output, "═"), "should contain custom divider")
}

func TestDivider_Render_WithTitleAndCustomChar(t *testing.T) {
	var buf strings.Builder
	d := Divider().Char('═').Title("Section")

	err := Print(d, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Section"), "should contain title")
	assert.True(t, strings.Contains(output, "═"), "should contain custom divider char")
}

func TestDivider_Render_Styled(t *testing.T) {
	var buf strings.Builder
	style := NewStyle().WithForeground(ColorMagenta).WithBold()
	d := Divider().Style(style).Title("Styled")

	err := Print(d, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Styled"), "should contain title")
	assert.True(t, strings.Contains(output, "\033["), "should contain ANSI codes")
}

func TestDivider_DefaultStyle(t *testing.T) {
	d := Divider()
	// Default style should have bright black foreground
	assert.Equal(t, ColorBrightBlack, d.style.Foreground)
}

func TestDivider_MultipleStyles(t *testing.T) {
	var buf strings.Builder
	view := Stack(
		Divider().Fg(ColorRed).Title("Red"),
		Divider().Fg(ColorGreen).Title("Green"),
		Divider().Fg(ColorBlue).Title("Blue"),
	)

	err := Print(view, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Red"), "should contain Red")
	assert.True(t, strings.Contains(output, "Green"), "should contain Green")
	assert.True(t, strings.Contains(output, "Blue"), "should contain Blue")
}

func TestHeaderBar_NotFlexible(t *testing.T) {
	h := HeaderBar("Title")
	// Header bars should NOT implement Flexible - they have fixed height (1 row)
	// and fill width via size(), not flex distribution.
	_, ok := interface{}(h).(Flexible)
	assert.False(t, ok, "headerBarView should not implement Flexible")
}

func TestStatusBar_NotFlexible(t *testing.T) {
	s := StatusBar("Status")
	// Status bars should NOT implement Flexible - they have fixed height (1 row)
	// and fill width via size(), not flex distribution.
	_, ok := interface{}(s).(Flexible)
	assert.False(t, ok, "statusBar (headerBarView) should not implement Flexible")
}
