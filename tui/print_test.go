package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestPrint_SimpleText(t *testing.T) {
	var buf strings.Builder
	view := Text("Hello, World!")

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Hello, World!"), "output should contain text")
}

func TestPrint_StyledText(t *testing.T) {
	var buf strings.Builder
	view := Text("Bold").Bold()

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI bold escape code
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape code")
	assert.True(t, strings.Contains(output, "Bold"), "output should contain text")
}

func TestPrint_ColoredText(t *testing.T) {
	var buf strings.Builder
	view := Text("Red").Fg(ColorRed)

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI color escape code
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape code")
	assert.True(t, strings.Contains(output, "Red"), "output should contain text")
}

func TestPrint_Stack(t *testing.T) {
	var buf strings.Builder
	view := Stack(
		Text("Line 1"),
		Text("Line 2"),
		Text("Line 3"),
	)

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Line 1"), "output should contain Line 1")
	assert.True(t, strings.Contains(output, "Line 2"), "output should contain Line 2")
	assert.True(t, strings.Contains(output, "Line 3"), "output should contain Line 3")

	// Should have multiple lines
	lines := strings.Split(output, "\n")
	assert.True(t, len(lines) >= 3, "output should have at least 3 lines")
}

func TestSprint(t *testing.T) {
	view := Text("Sprint test")

	output := Sprint(view, PrintConfig{Width: 80})
	assert.True(t, strings.Contains(output, "Sprint test"), "output should contain text")
}

func TestFprint(t *testing.T) {
	var buf strings.Builder
	view := Text("Fprint test")

	err := Fprint(&buf, view, PrintConfig{Width: 80})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(buf.String(), "Fprint test"), "output should contain text")
}

func TestPrint_WithHeight(t *testing.T) {
	var buf strings.Builder
	view := Text("Fixed height")

	err := Print(view, PrintConfig{Width: 80, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Fixed height"), "output should contain text")
}

func TestPrint_Bordered(t *testing.T) {
	var buf strings.Builder
	view := Bordered(Text("Bordered")).Border(&SingleBorder)

	err := Print(view, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Bordered"), "output should contain text")
	// Should contain border characters
	assert.True(t, strings.Contains(output, "─"), "output should contain border character")
}

func TestPrint_Empty(t *testing.T) {
	var buf strings.Builder
	view := Empty()

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)
	// Should not error on empty view
}

func TestLivePrinter_Update(t *testing.T) {
	var buf strings.Builder
	lp := NewLivePrinter(PrintConfig{Width: 40, Output: &buf})

	// First update
	err := lp.Update(Text("First"))
	assert.NoError(t, err)
	assert.True(t, strings.Contains(buf.String(), "First"), "should contain first text")

	// Second update should contain cursor movement
	err = lp.Update(Text("Second"))
	assert.NoError(t, err)
	assert.True(t, strings.Contains(buf.String(), "Second"), "should contain second text")
	assert.True(t, strings.Contains(buf.String(), "\033["), "should contain ANSI codes")

	lp.Stop()
}

func TestLivePrinter_Clear(t *testing.T) {
	var buf strings.Builder
	lp := NewLivePrinter(PrintConfig{Width: 40, Output: &buf})

	lp.Update(Text("Content"))
	lp.Clear()
	lp.Stop()

	// Should contain clear sequence
	assert.True(t, strings.Contains(buf.String(), "\033[0J"), "should contain clear sequence")
}

func TestLive_Convenience(t *testing.T) {
	var buf strings.Builder
	count := 0

	err := Live(func(update func(View)) {
		for i := 0; i < 3; i++ {
			update(Text("Count: %d", i))
			count++
		}
	}, PrintConfig{Width: 40, Output: &buf})

	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.True(t, strings.Contains(buf.String(), "Count:"), "should contain text")
}
