package tui

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTextArea_Constructor(t *testing.T) {
	t.Run("with binding", func(t *testing.T) {
		content := "hello"
		ta := TextArea(&content)

		assert.NotNil(t, ta)
		assert.Equal(t, &content, ta.binding)
		assert.Equal(t, 40, ta.width)
		assert.Equal(t, 10, ta.height)
		assert.Equal(t, "(empty)", ta.emptyPlaceholder)
	})

	t.Run("with nil binding", func(t *testing.T) {
		ta := TextArea(nil)

		assert.NotNil(t, ta)
		assert.Nil(t, ta.binding)
		assert.Equal(t, "", ta.id)
	})
}

func TestTextArea_ID(t *testing.T) {
	ta := TextArea(nil)
	result := ta.ID("my-text-area")

	assert.Equal(t, ta, result, "should return self for chaining")
	assert.Equal(t, "my-text-area", ta.id)
}

func TestTextArea_Content(t *testing.T) {
	ta := TextArea(nil)
	result := ta.Content("static content")

	assert.Equal(t, ta, result)
	assert.Equal(t, "static content", ta.content)
}

func TestTextArea_ScrollY(t *testing.T) {
	scrollPos := 5
	ta := TextArea(nil)
	result := ta.ScrollY(&scrollPos)

	assert.Equal(t, ta, result)
	assert.Equal(t, &scrollPos, ta.scrollY)
}

func TestTextArea_Dimensions(t *testing.T) {
	t.Run("Width", func(t *testing.T) {
		ta := TextArea(nil)
		result := ta.Width(80)

		assert.Equal(t, ta, result)
		assert.Equal(t, 80, ta.width)
	})

	t.Run("Height", func(t *testing.T) {
		ta := TextArea(nil)
		result := ta.Height(20)

		assert.Equal(t, ta, result)
		assert.Equal(t, 20, ta.height)
	})

	t.Run("Size", func(t *testing.T) {
		ta := TextArea(nil)
		result := ta.Size(100, 50)

		assert.Equal(t, ta, result)
		assert.Equal(t, 100, ta.width)
		assert.Equal(t, 50, ta.height)
	})
}

func TestTextArea_Title(t *testing.T) {
	ta := TextArea(nil)
	result := ta.Title("My Title")

	assert.Equal(t, ta, result)
	assert.Equal(t, "My Title", ta.title)
}

func TestTextArea_TitleStyle(t *testing.T) {
	ta := TextArea(nil)
	style := NewStyle().WithForeground(ColorRed)
	result := ta.TitleStyle(style)

	assert.Equal(t, ta, result)
	assert.Equal(t, style, ta.titleStyle)
}

func TestTextArea_FocusTitleStyle(t *testing.T) {
	ta := TextArea(nil)
	style := NewStyle().WithForeground(ColorGreen)
	result := ta.FocusTitleStyle(style)

	assert.Equal(t, ta, result)
	assert.NotNil(t, ta.focusTitleStyle)
	assert.Equal(t, style, *ta.focusTitleStyle)
}

func TestTextArea_TextStyle(t *testing.T) {
	ta := TextArea(nil)
	style := NewStyle().WithForeground(ColorBlue)
	result := ta.TextStyle(style)

	assert.Equal(t, ta, result)
	assert.Equal(t, style, ta.textStyle)
}

func TestTextArea_EmptyPlaceholder(t *testing.T) {
	ta := TextArea(nil)
	result := ta.EmptyPlaceholder("No content here")

	assert.Equal(t, ta, result)
	assert.Equal(t, "No content here", ta.emptyPlaceholder)
}

func TestTextArea_EmptyStyle(t *testing.T) {
	ta := TextArea(nil)
	style := NewStyle().WithForeground(ColorBrightBlack)
	result := ta.EmptyStyle(style)

	assert.Equal(t, ta, result)
	assert.Equal(t, style, ta.emptyStyle)
}

func TestTextArea_Bordered(t *testing.T) {
	ta := TextArea(nil)
	result := ta.Bordered()

	assert.Equal(t, ta, result)
	assert.True(t, ta.bordered)
	assert.NotNil(t, ta.border)
	assert.Equal(t, &RoundedBorder, ta.border)
}

func TestTextArea_Border(t *testing.T) {
	ta := TextArea(nil)
	result := ta.Border(&DoubleBorder)

	assert.Equal(t, ta, result)
	assert.True(t, ta.bordered)
	assert.Equal(t, &DoubleBorder, ta.border)
}

func TestTextArea_BorderFg(t *testing.T) {
	ta := TextArea(nil)
	result := ta.BorderFg(ColorYellow)

	assert.Equal(t, ta, result)
	assert.Equal(t, ColorYellow, ta.borderFg)
}

func TestTextArea_FocusBorderFg(t *testing.T) {
	ta := TextArea(nil)
	result := ta.FocusBorderFg(ColorCyan)

	assert.Equal(t, ta, result)
	assert.Equal(t, ColorCyan, ta.focusBorderFg)
	assert.True(t, ta.hasFocusBorder)
}

func TestTextArea_LeftBorderOnly(t *testing.T) {
	ta := TextArea(nil)
	result := ta.LeftBorderOnly()

	assert.Equal(t, ta, result)
	assert.True(t, ta.leftBorderOnly)
	assert.True(t, ta.bordered)
	assert.NotNil(t, ta.border)
}

func TestTextArea_LineNumbers(t *testing.T) {
	ta := TextArea(nil)

	result := ta.LineNumbers(true)
	assert.Equal(t, ta, result)
	assert.True(t, ta.showLineNumbers)

	ta.LineNumbers(false)
	assert.False(t, ta.showLineNumbers)
}

func TestTextArea_LineNumberStyle(t *testing.T) {
	ta := TextArea(nil)
	style := NewStyle().WithForeground(ColorMagenta)
	result := ta.LineNumberStyle(style)

	assert.Equal(t, ta, result)
	assert.Equal(t, style, ta.lineNumberStyle)
}

func TestTextArea_LineNumberFg(t *testing.T) {
	ta := TextArea(nil)
	result := ta.LineNumberFg(ColorGreen)

	assert.Equal(t, ta, result)
	assert.Equal(t, ColorGreen, ta.lineNumberFg)
	assert.True(t, ta.hasLineNumberFg)
}

func TestTextArea_HighlightCurrentLine(t *testing.T) {
	ta := TextArea(nil)

	result := ta.HighlightCurrentLine(true)
	assert.Equal(t, ta, result)
	assert.True(t, ta.highlightCurrentLine)

	ta.HighlightCurrentLine(false)
	assert.False(t, ta.highlightCurrentLine)
}

func TestTextArea_CurrentLineStyle(t *testing.T) {
	ta := TextArea(nil)
	style := NewStyle().WithBackground(ColorBlue)
	result := ta.CurrentLineStyle(style)

	assert.Equal(t, ta, result)
	assert.Equal(t, style, ta.currentLineStyle)
	assert.True(t, ta.hasCurrentLineStyle)
}

func TestTextArea_CursorLine(t *testing.T) {
	cursorLine := 5
	ta := TextArea(nil)
	result := ta.CursorLine(&cursorLine)

	assert.Equal(t, ta, result)
	assert.Equal(t, &cursorLine, ta.cursorLine)
}

func TestTextArea_getContent(t *testing.T) {
	t.Run("with binding", func(t *testing.T) {
		content := "bound content"
		ta := TextArea(&content)
		ta.content = "static content"

		assert.Equal(t, "bound content", ta.getContent())
	})

	t.Run("without binding", func(t *testing.T) {
		ta := TextArea(nil)
		ta.content = "static content"

		assert.Equal(t, "static content", ta.getContent())
	})
}

func TestTextArea_ScrollYAccessors(t *testing.T) {
	t.Run("with external scrollY", func(t *testing.T) {
		scrollPos := 10
		ta := TextArea(nil).ScrollY(&scrollPos)

		assert.Equal(t, 10, ta.getScrollY())

		ta.setScrollY(15)
		assert.Equal(t, 15, scrollPos)
		assert.Equal(t, 15, ta.getScrollY())
	})

	t.Run("with internal scrollY", func(t *testing.T) {
		ta := TextArea(nil)
		ta.internal = 5

		assert.Equal(t, 5, ta.getScrollY())

		ta.setScrollY(20)
		assert.Equal(t, 20, ta.internal)
		assert.Equal(t, 20, ta.getScrollY())
	})
}

func TestTextArea_CursorLineAccessors(t *testing.T) {
	t.Run("with external cursorLine", func(t *testing.T) {
		cursorLine := 3
		ta := TextArea(nil).CursorLine(&cursorLine)

		assert.Equal(t, 3, ta.getCursorLine())

		ta.setCursorLine(7)
		assert.Equal(t, 7, cursorLine)
		assert.Equal(t, 7, ta.getCursorLine())
	})

	t.Run("with internal cursorLine", func(t *testing.T) {
		ta := TextArea(nil)
		ta.internalCursorLine = 2

		assert.Equal(t, 2, ta.getCursorLine())

		ta.setCursorLine(8)
		assert.Equal(t, 8, ta.internalCursorLine)
		assert.Equal(t, 8, ta.getCursorLine())
	})
}

func TestTextArea_size(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		maxWidth   int
		maxHeight  int
		expectW    int
		expectH    int
	}{
		{"no constraints", 40, 10, 0, 0, 40, 10},
		{"within constraints", 40, 10, 100, 100, 40, 10},
		{"width exceeds max", 40, 10, 30, 100, 30, 10},
		{"height exceeds max", 40, 10, 100, 5, 40, 5},
		{"both exceed max", 40, 10, 20, 5, 20, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := TextArea(nil).Size(tt.width, tt.height)
			w, h := ta.size(tt.maxWidth, tt.maxHeight)
			assert.Equal(t, tt.expectW, w)
			assert.Equal(t, tt.expectH, h)
		})
	}
}

func TestTextArea_lineNumberWidth(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		show     bool
		expected int
	}{
		{"disabled", "line1\nline2", false, 0},
		{"single digit", "line1\nline2", true, 2},    // "1 "
		{"double digit", "1\n2\n3\n4\n5\n6\n7\n8\n9\n10", true, 3}, // "10 "
		{"triple digit", func() string {
			s := ""
			for i := range 100 {
				if i > 0 {
					s += "\n"
				}
				s += "x"
			}
			return s
		}(), true, 4}, // "100 "
		{"empty content", "", true, 2}, // "1 " for the single empty line
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := TextArea(nil).Content(tt.content).LineNumbers(tt.show)
			assert.Equal(t, tt.expected, ta.lineNumberWidth())
		})
	}
}

func TestTextArea_ChainedConfiguration(t *testing.T) {
	content := "test content"
	scrollY := 5
	cursorLine := 2

	ta := TextArea(&content).
		ID("my-textarea").
		Width(80).
		Height(20).
		Title("My Title").
		Bordered().
		BorderFg(ColorYellow).
		FocusBorderFg(ColorCyan).
		LineNumbers(true).
		HighlightCurrentLine(true).
		ScrollY(&scrollY).
		CursorLine(&cursorLine)

	assert.Equal(t, "my-textarea", ta.id)
	assert.Equal(t, 80, ta.width)
	assert.Equal(t, 20, ta.height)
	assert.Equal(t, "My Title", ta.title)
	assert.True(t, ta.bordered)
	assert.Equal(t, ColorYellow, ta.borderFg)
	assert.Equal(t, ColorCyan, ta.focusBorderFg)
	assert.True(t, ta.hasFocusBorder)
	assert.True(t, ta.showLineNumbers)
	assert.True(t, ta.highlightCurrentLine)
	assert.Equal(t, &scrollY, ta.scrollY)
	assert.Equal(t, &cursorLine, ta.cursorLine)
}

// textAreaFocusHandler tests

func TestTextAreaFocusHandler_FocusID(t *testing.T) {
	content := "test"
	ta := TextArea(&content).ID("test-area")
	handler := &textAreaFocusHandler{area: ta}

	assert.Equal(t, "test-area", handler.FocusID())
}

func TestTextAreaFocusHandler_FocusedState(t *testing.T) {
	handler := &textAreaFocusHandler{}

	assert.False(t, handler.IsFocused())

	handler.SetFocused(true)
	assert.True(t, handler.IsFocused())

	handler.SetFocused(false)
	assert.False(t, handler.IsFocused())
}

func TestTextAreaFocusHandler_FocusBounds(t *testing.T) {
	bounds := image.Rect(10, 20, 100, 80)
	handler := &textAreaFocusHandler{bounds: bounds}

	assert.Equal(t, bounds, handler.FocusBounds())
}

func TestTextAreaFocusHandler_HandleKeyEvent(t *testing.T) {
	t.Run("arrow up scrolls", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3")
		ta.internal = 2
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})

		assert.True(t, handled)
		assert.Equal(t, 1, ta.getScrollY())
	})

	t.Run("arrow up at top doesn't go negative", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2")
		ta.internal = 0
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})

		assert.False(t, handled)
		assert.Equal(t, 0, ta.getScrollY())
	})

	t.Run("arrow down scrolls", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3")
		ta.internal = 0
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})

		assert.True(t, handled)
		assert.Equal(t, 1, ta.getScrollY())
	})

	t.Run("page up scrolls by 5", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3")
		ta.internal = 10
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyPageUp})

		assert.True(t, handled)
		assert.Equal(t, 5, ta.getScrollY())
	})

	t.Run("page up doesn't go negative", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3")
		ta.internal = 3
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyPageUp})

		assert.True(t, handled)
		assert.Equal(t, 0, ta.getScrollY())
	})

	t.Run("page down scrolls by 5", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3")
		ta.internal = 0
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyPageDown})

		assert.True(t, handled)
		assert.Equal(t, 5, ta.getScrollY())
	})

	t.Run("home scrolls to top", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3")
		ta.internal = 10
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyHome})

		assert.True(t, handled)
		assert.Equal(t, 0, ta.getScrollY())
	})

	t.Run("end is handled", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3")
		ta.internal = 0
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyEnd})

		assert.True(t, handled)
	})

	t.Run("unhandled key", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2")
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyEscape})

		assert.False(t, handled)
	})
}

func TestTextAreaFocusHandler_HandleKeyEvent_WithHighlightCurrentLine(t *testing.T) {
	t.Run("arrow up moves cursor line", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3").HighlightCurrentLine(true)
		ta.internalCursorLine = 2
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})

		assert.True(t, handled)
		assert.Equal(t, 1, ta.getCursorLine())
	})

	t.Run("arrow up at first line doesn't go negative", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2").HighlightCurrentLine(true)
		ta.internalCursorLine = 0
		ta.internal = 5
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})

		// When cursor is at line 0 but scroll is > 0, it scrolls instead
		assert.True(t, handled)
		assert.Equal(t, 0, ta.getCursorLine())
		assert.Equal(t, 4, ta.getScrollY())
	})

	t.Run("arrow down moves cursor line", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3").HighlightCurrentLine(true)
		ta.internalCursorLine = 0
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})

		assert.True(t, handled)
		assert.Equal(t, 1, ta.getCursorLine())
	})

	t.Run("arrow down at last line", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3").HighlightCurrentLine(true)
		ta.internalCursorLine = 2 // last line (0-indexed)
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})

		// At last line, it just scrolls
		assert.True(t, handled)
		assert.Equal(t, 2, ta.getCursorLine())
	})

	t.Run("page up moves cursor by 5", func(t *testing.T) {
		ta := TextArea(nil).Content("0\n1\n2\n3\n4\n5\n6\n7\n8\n9").HighlightCurrentLine(true)
		ta.internalCursorLine = 8
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyPageUp})

		assert.True(t, handled)
		assert.Equal(t, 3, ta.getCursorLine())
	})

	t.Run("page down moves cursor by 5", func(t *testing.T) {
		ta := TextArea(nil).Content("0\n1\n2\n3\n4\n5\n6\n7\n8\n9").HighlightCurrentLine(true)
		ta.internalCursorLine = 2
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyPageDown})

		assert.True(t, handled)
		assert.Equal(t, 7, ta.getCursorLine())
	})

	t.Run("home moves cursor to first line", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3").HighlightCurrentLine(true)
		ta.internalCursorLine = 2
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyHome})

		assert.True(t, handled)
		assert.Equal(t, 0, ta.getCursorLine())
		assert.Equal(t, 0, ta.getScrollY())
	})

	t.Run("end moves cursor to last line", func(t *testing.T) {
		ta := TextArea(nil).Content("line1\nline2\nline3").HighlightCurrentLine(true).Size(40, 10)
		ta.internalCursorLine = 0
		handler := &textAreaFocusHandler{area: ta}

		handled := handler.HandleKeyEvent(KeyEvent{Key: KeyEnd})

		assert.True(t, handled)
		assert.Equal(t, 2, ta.getCursorLine()) // last line is index 2
	})
}

func TestTextAreaFocusHandler_AutoScroll(t *testing.T) {
	t.Run("auto-scrolls up when cursor moves above viewport", func(t *testing.T) {
		ta := TextArea(nil).Content("0\n1\n2\n3\n4\n5").HighlightCurrentLine(true)
		ta.internalCursorLine = 3
		ta.internal = 3 // viewport starts at line 3
		handler := &textAreaFocusHandler{area: ta}

		handler.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})

		assert.Equal(t, 2, ta.getCursorLine())
		assert.Equal(t, 2, ta.getScrollY()) // scrolled to follow cursor
	})
}

// render tests

func TestTextArea_Render_EmptyContent(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Size(40, 5)

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "(empty)"), "should show empty placeholder")
}

func TestTextArea_Render_CustomEmptyPlaceholder(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Size(40, 5).EmptyPlaceholder("Nothing here")

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Nothing here"), "should show custom placeholder")
}

func TestTextArea_Render_WithContent(t *testing.T) {
	var buf strings.Builder
	content := "Hello World\nLine 2\nLine 3"
	ta := TextArea(&content).Size(40, 5)

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Hello World"), "should contain content")
	assert.True(t, strings.Contains(output, "Line 2"), "should contain line 2")
}

func TestTextArea_Render_WithStaticContent(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Static content").Size(40, 5)

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Static content"), "should contain static content")
}

func TestTextArea_Render_Bordered(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Test").Size(20, 5).Bordered()

	err := Print(ta, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Test"), "should contain content")
	// Should contain border characters (rounded border uses curved corners)
	assert.True(t, strings.Contains(output, "─"), "should contain horizontal border")
}

func TestTextArea_Render_BorderedWithTitle(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Test").Size(30, 5).Bordered().Title("My Title")

	err := Print(ta, WithWidth(30), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Test"), "should contain content")
	assert.True(t, strings.Contains(output, "My Title"), "should contain title")
}

func TestTextArea_Render_WithLineNumbers(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Line A\nLine B\nLine C").Size(40, 5).LineNumbers(true)

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "1"), "should contain line number 1")
	assert.True(t, strings.Contains(output, "2"), "should contain line number 2")
	assert.True(t, strings.Contains(output, "Line A"), "should contain content")
}

func TestTextArea_Render_LeftBorderOnly(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Test content").Size(30, 5).LeftBorderOnly()

	err := Print(ta, WithWidth(30), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Test content"), "should contain content")
	// Should contain vertical border on left side
	assert.True(t, strings.Contains(output, "│"), "should contain left border")
}

func TestTextArea_Render_ZeroSize(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ta := TextArea(nil).Content("Test").Size(40, 10)

	// Create a zero-size context
	ctx := NewRenderContext(frame, 0)
	subCtx := ctx.SubContext(image.Rect(0, 0, 0, 0))

	// Should not panic with zero size
	ta.render(subCtx)
}

func TestTextArea_Render_UpdatesScrollPosition(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	scrollY := 5
	ta := TextArea(nil).Content("Line 1\nLine 2\nLine 3").Size(40, 10).ScrollY(&scrollY)

	ctx := NewRenderContext(frame, 0)
	subCtx := ctx.SubContext(image.Rect(0, 0, 40, 10))

	ta.render(subCtx)

	// Scroll position should be updated (clamped if needed)
	assert.True(t, scrollY >= 0, "scroll position should be non-negative")
}

func TestTextArea_Render_RegistersFocusHandler(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	// Clear focus manager state
	focusManager.Clear()

	ta := TextArea(nil).Content("Test").Size(40, 10).ID("test-textarea")

	ctx := NewRenderContext(frame, 0)
	subCtx := ctx.SubContext(image.Rect(0, 0, 40, 10))

	ta.render(subCtx)

	// Focus handler should be registered
	handlers := focusManager.focusables
	found := false
	for _, h := range handlers {
		if h.FocusID() == "test-textarea" {
			found = true
			break
		}
	}
	assert.True(t, found, "should register focus handler")
}

// renderBordered tests

func TestTextArea_RenderBordered_FullBorder(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)

	ta := TextArea(nil).Content("Test content").Size(30, 10).Bordered().Title("Title")

	ctx := NewRenderContext(frame, 0)
	subCtx := ctx.SubContext(image.Rect(0, 0, 30, 10))

	ta.render(subCtx)

	// End frame to flush output
	terminal.EndFrame(frame)

	output := buf.String()
	// Should have rendered with border characters
	assert.True(t, len(output) > 0, "should have rendered output")
}

func TestTextArea_RenderBordered_LeftBorderOnly(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)

	ta := TextArea(nil).Content("Test content").Size(30, 10).LeftBorderOnly()

	ctx := NewRenderContext(frame, 0)
	subCtx := ctx.SubContext(image.Rect(0, 0, 30, 10))

	ta.render(subCtx)

	terminal.EndFrame(frame)

	output := buf.String()
	assert.True(t, len(output) > 0, "should have rendered output")
}

func TestTextArea_RenderBordered_SmallWidth(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	// Very small width should still render without panic
	ta := TextArea(nil).Content("Test").Size(3, 3).Bordered().Title("T")

	ctx := NewRenderContext(frame, 0)
	subCtx := ctx.SubContext(image.Rect(0, 0, 3, 3))

	ta.render(subCtx)
}

func TestTextArea_RenderBordered_TitleTruncation(t *testing.T) {
	var buf strings.Builder
	// Very long title should be truncated
	ta := TextArea(nil).Content("Test").Size(15, 5).Bordered().Title("Very Long Title That Should Be Truncated")

	err := Print(ta, WithWidth(15), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	// Should not panic and should render something
	output := buf.String()
	assert.True(t, len(output) > 0, "should have rendered output")
}

func TestTextArea_RenderBordered_MinimalHeight(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	// Height of 1 is edge case
	ta := TextArea(nil).Content("Test").Size(20, 1).Bordered()

	ctx := NewRenderContext(frame, 0)
	subCtx := ctx.SubContext(image.Rect(0, 0, 20, 1))

	ta.render(subCtx)
}

func TestTextArea_RenderBordered_WithCustomBorderStyle(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Test").Size(20, 5).Border(&DoubleBorder)

	err := Print(ta, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	// Double border uses ═ for horizontal
	assert.True(t, strings.Contains(output, "═"), "should contain double border horizontal")
}

func TestTextArea_RenderBordered_WithBorderFg(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Test").Size(20, 5).Bordered().BorderFg(ColorRed)

	err := Print(ta, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	// Should contain ANSI color codes for red
	assert.True(t, strings.Contains(output, "\033["), "should contain ANSI escape codes")
}

func TestTextArea_RenderBordered_NoBorderNil(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	// Test with bordered=true but border=nil (shouldn't happen normally but test edge case)
	ta := TextArea(nil).Content("Test").Size(20, 5)
	ta.bordered = true
	ta.border = nil

	ctx := NewRenderContext(frame, 0)
	subCtx := ctx.SubContext(image.Rect(0, 0, 20, 5))

	// Should not panic - it checks for border != nil
	ta.render(subCtx)
}

func TestTextArea_Render_WithHighlightCurrentLine(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Line 1\nLine 2\nLine 3").Size(40, 5).HighlightCurrentLine(true)

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Line 1"), "should contain content")
	// Should have ANSI codes for background color on current line
	assert.True(t, strings.Contains(output, "\033["), "should contain ANSI escape codes")
}

func TestTextArea_Render_WithCustomCurrentLineStyle(t *testing.T) {
	var buf strings.Builder
	ta := TextArea(nil).Content("Line 1\nLine 2").Size(40, 5).
		HighlightCurrentLine(true).
		CurrentLineStyle(NewStyle().WithBackground(ColorBlue))

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, len(output) > 0, "should have rendered output")
}

func TestTextArea_Render_EmptyLines(t *testing.T) {
	var buf strings.Builder
	// Content with empty lines
	ta := TextArea(nil).Content("Line 1\n\nLine 3").Size(40, 5)

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Line 1"), "should contain line 1")
	assert.True(t, strings.Contains(output, "Line 3"), "should contain line 3")
}

func TestTextArea_Render_WithLineNumbersAndHighlight(t *testing.T) {
	var buf strings.Builder
	cursorLine := 1
	ta := TextArea(nil).Content("A\nB\nC").Size(40, 5).
		LineNumbers(true).
		HighlightCurrentLine(true).
		CursorLine(&cursorLine)

	err := Print(ta, WithWidth(40), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "1"), "should contain line number")
	assert.True(t, strings.Contains(output, "A"), "should contain content")
}
