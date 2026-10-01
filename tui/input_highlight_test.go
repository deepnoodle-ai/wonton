package tui

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// drawTextInput draws ti into a w x h test frame and returns a cell reader.
func drawTextInput(t *testing.T, ti *textInput, w, h int) func(x, y int) Cell {
	t.Helper()
	term := NewTestTerminal(w, h, &bytes.Buffer{})
	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	ti.SetBounds(image.Rect(0, 0, w, h))
	ti.Draw(frame)
	return frame.Cell
}

func ranges(rs ...TextRange) func(string) []TextRange {
	return func(string) []TextRange { return rs }
}

var (
	boldStyle      = NewStyle().WithBold()
	underlineStyle = NewStyle().WithUnderline()
)

func TestInputField_Highlight_StylesRange(t *testing.T) {
	value := "/effort high"
	cmdStyle := NewStyle().WithForeground(ColorCyan).WithBold()
	field := InputField(&value).
		TextStyle(NewStyle().WithItalic()).
		Highlight(func(v string) []TextRange {
			cmd, _, _ := strings.Cut(v, " ")
			return []TextRange{{Start: 0, End: len(cmd), Style: cmdStyle}}
		})
	screen := SprintScreen(field, WithWidth(20))

	termtest.AssertRowContains(t, screen, 0, "/effort high")
	for x := 0; x < len("/effort"); x++ {
		s := screen.Cell(x, 0).Style
		assert.True(t, s.Bold, "cell %d bold", x)
		assert.True(t, s.Italic, "cell %d keeps TextStyle", x)
		assert.Equal(t, uint8(ColorCyan), s.Foreground.Value, "cell %d cyan", x)
	}
	for x := len("/effort"); x < len(value); x++ {
		s := screen.Cell(x, 0).Style
		assert.False(t, s.Bold, "cell %d not bold", x)
		assert.True(t, s.Italic, "cell %d keeps TextStyle", x)
	}
}

func TestTextInput_Highlight_SpansWrap(t *testing.T) {
	ti := newTextInput()
	ti.SetValue("abcdefgh")
	ti.Highlight = ranges(TextRange{Start: 3, End: 7, Style: boldStyle})
	cell := drawTextInput(t, ti, 5, 2)

	assert.Equal(t, 'd', cell(3, 0).Char)
	assert.Equal(t, 'f', cell(0, 1).Char)
	for x, want := range []bool{false, false, false, true, true} {
		assert.Equal(t, want, cell(x, 0).Style.Bold, "row 0 cell %d", x)
	}
	for x, want := range []bool{true, true, false} {
		assert.Equal(t, want, cell(x, 1).Style.Bold, "row 1 cell %d", x)
	}
}

func TestTextInput_Highlight_MultiByteAndWide(t *testing.T) {
	ti := newTextInput()
	value := "前後 é!"
	ti.SetValue(value)
	start := strings.Index(value, "後")
	ti.Highlight = ranges(
		TextRange{Start: start, End: start + len("後"), Style: boldStyle},
		// Starts inside the multi-byte é: the cluster's first byte is not
		// covered, so é stays plain.
		TextRange{Start: strings.Index(value, "é") + 1, End: len(value), Style: underlineStyle},
	)
	cell := drawTextInput(t, ti, 10, 1)

	assert.Equal(t, '前', cell(0, 0).Char)
	assert.False(t, cell(0, 0).Style.Bold)
	assert.Equal(t, '後', cell(2, 0).Char)
	assert.True(t, cell(2, 0).Style.Bold)
	assert.False(t, cell(4, 0).Style.Bold, "space after the wide char")
	assert.Equal(t, 'é', cell(5, 0).Char)
	assert.False(t, cell(5, 0).Style.Underline)
	assert.Equal(t, '!', cell(6, 0).Char)
	assert.True(t, cell(6, 0).Style.Underline)
}

func TestTextInput_Highlight_BadRanges(t *testing.T) {
	ti := newTextInput()
	ti.SetValue("abcd")
	ti.Highlight = ranges(
		TextRange{Start: 1, End: 1000, Style: underlineStyle}, // runs past the end
		TextRange{Start: -5, End: 2, Style: boldStyle},        // starts before 0
		TextRange{Start: 100, End: 200},                       // entirely past the end
		TextRange{Start: 3, End: 1},                           // inverted
		TextRange{Start: -10, End: -2},                        // entirely before 0
	)
	var cell func(x, y int) Cell
	assert.NotPanics(t, func() { cell = drawTextInput(t, ti, 10, 1) })

	assert.True(t, cell(0, 0).Style.Bold)
	assert.False(t, cell(0, 0).Style.Underline)
	assert.True(t, cell(1, 0).Style.Bold && cell(1, 0).Style.Underline, "overlap merges both")
	assert.False(t, cell(2, 0).Style.Bold)
	assert.True(t, cell(3, 0).Style.Underline)
	assert.False(t, cell(4, 0).Style.Underline, "blank cell after the value")
}

func TestTextInput_Highlight_SkipsPastePlaceholders(t *testing.T) {
	ti := newTextInput()
	ti.WithPastePlaceholderMode(true)
	ti.SetValue("ab")
	ti.HandlePaste("x\ny\nz")
	ti.insertAtCursor("cd")
	value := ti.Value()
	assert.Equal(t, "abx\ny\nzcd", value)

	var got string
	ti.Highlight = func(v string) []TextRange {
		got = v
		return []TextRange{
			{Start: len(value) - 2, End: len(value), Style: boldStyle}, // "cd"
			{Start: 0, End: len(value), Style: underlineStyle},
		}
	}
	cell := drawTextInput(t, ti, 40, 1)
	assert.Equal(t, value, got, "Highlight sees the real value, not the placeholder")

	assert.False(t, cell(0, 0).Style.Bold)
	assert.True(t, cell(0, 0).Style.Underline)
	placeholder := "[pasted 3 lines]"
	for x := 2; x < 2+len(placeholder); x++ {
		assert.Equal(t, ti.PasteStyle, cell(x, 0).Style, "placeholder cell %d", x)
	}
	cx := 2 + len(placeholder)
	assert.Equal(t, 'c', cell(cx, 0).Char)
	assert.True(t, cell(cx, 0).Style.Bold && cell(cx+1, 0).Style.Bold)
}

func TestTextInput_Highlight_EmptyAndMasked(t *testing.T) {
	ti := newTextInput()
	ti.WithPlaceholder("type here")
	ti.Highlight = ranges(TextRange{Start: 0, End: 100, Style: boldStyle})
	cell := drawTextInput(t, ti, 20, 1)
	assert.Equal(t, ti.PlaceholderStyle, cell(0, 0).Style)

	ti.SetValue("secret")
	ti.WithMask('*')
	cell = drawTextInput(t, ti, 20, 1)
	assert.Equal(t, '*', cell(0, 0).Char)
	assert.False(t, cell(0, 0).Style.Bold)
}

func TestInputRegistry_SyncsStyleAndHighlightEachRender(t *testing.T) {
	text := "hi"
	reg := &inputRegistryImpl{inputs: make(map[string]*inputState)}
	italic := NewStyle().WithItalic()
	state := reg.Register("in", inputConfig{binding: &text, textStyle: &italic, placeholder: "a"}, nil)
	assert.Equal(t, italic, state.input.Style)
	assert.Nil(t, state.input.Highlight)

	// A later render with a different style, placeholder, and highlight
	// takes effect without recreating the input.
	bold := NewStyle().WithBold()
	state = reg.Register("in", inputConfig{
		binding: &text, textStyle: &bold, placeholder: "b",
		placeholderStyle: &italic,
		highlight:        ranges(TextRange{Start: 0, End: 1, Style: bold}),
	}, nil)
	assert.Equal(t, bold, state.input.Style)
	assert.Equal(t, "b", state.input.Placeholder)
	assert.Equal(t, italic, state.input.PlaceholderStyle)
	assert.Equal(t, 1, len(state.input.Highlight("hi")))

	// Dropping the options restores the defaults.
	state = reg.Register("in", inputConfig{binding: &text}, nil)
	assert.Equal(t, NewStyle(), state.input.Style)
	assert.Equal(t, defaultPlaceholderStyle, state.input.PlaceholderStyle)
	assert.Nil(t, state.input.Highlight)
}

func ExampleInputFieldView_Highlight() {
	commands := map[string]bool{"/effort": true, "/help": true}
	commandStyle := NewStyle().WithForeground(ColorCyan).WithBold()

	var draft string
	// Return this field from the application's View; the function runs on
	// every render, so it always sees the current draft.
	field := InputField(&draft).Highlight(func(value string) []TextRange {
		cmd, _, _ := strings.Cut(value, " ")
		if !commands[cmd] {
			return nil
		}
		return []TextRange{{Start: 0, End: len(cmd), Style: commandStyle}}
	})
	for _, value := range []string{"/effort high", "/nope"} {
		for _, r := range field.highlight(value) {
			fmt.Printf("%q styled in %q\n", value[r.Start:r.End], value)
		}
	}
	// Output:
	// "/effort" styled in "/effort high"
}
