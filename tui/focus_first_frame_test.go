package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// renderFirstFrame renders view once with a fresh focus manager, as the
// runtime does on frame 0, and returns the terminal.
func renderFirstFrame(t *testing.T, view View) *Terminal {
	t.Helper()
	var buf bytes.Buffer
	terminal := NewTestTerminal(30, 6, &buf)
	frame, err := terminal.BeginFrame()
	if err != nil {
		t.Fatalf("BeginFrame: %v", err)
	}
	ctx := NewRenderContext(frame, 0).WithFocusManager(NewFocusManager()).withRegistries(newRegistries())
	view.size(30, 6)
	view.render(ctx)
	terminal.EndFrame(frame)
	return terminal
}

// The auto-focused widget must draw its focused style on the first frame.
func TestFirstFrameFocusStyle(t *testing.T) {
	val := ""
	term := renderFirstFrame(t, InputField(&val).Label("Name").Bordered())
	assert.Equal(t, ColorCyan, term.GetCell(0, 0).Style.Foreground)

	content := "a\nb"
	term = renderFirstFrame(t, TextArea(&content).Bordered().Size(10, 4))
	assert.Equal(t, ColorCyan, term.GetCell(0, 0).Style.Foreground)

	focus := NewStyle().WithForeground(ColorRed)
	in := Input(&val).FocusStyle(focus)
	renderFirstFrame(t, in)
	assert.Equal(t, focus, in.reg.inputs.inputs[in.id].input.Style)
}

// End scrolls to the bottom even without current-line highlighting.
func TestTextAreaEndScrollsToBottom(t *testing.T) {
	ta := TextArea(nil).Content(strings.Repeat("x\n", 19)+"x").Size(30, 6)
	handler := &textAreaFocusHandler{area: ta}
	assert.True(t, handler.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	renderFirstFrame(t, ta)              // render clamps scrollY to the last page
	assert.Equal(t, 14, ta.getScrollY()) // 20 lines, 6 visible
}
