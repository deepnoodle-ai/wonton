package terminal

import (
	"io"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestSetSizeResizesAndCallsCallbacks(t *testing.T) {
	term := NewTestTerminal(80, 24, io.Discard)
	var calls [][2]int
	unsub := term.OnResize(func(w, h int) { calls = append(calls, [2]int{w, h}) })
	defer unsub()

	term.SetSize(40, 10)
	w, h := term.Size()
	assert.Equal(t, w, 40)
	assert.Equal(t, h, 10)
	assert.Equal(t, calls, [][2]int{{40, 10}})

	// The same size, or a size under one cell, changes nothing.
	term.SetSize(40, 10)
	term.SetSize(0, 10)
	term.SetSize(40, -1)
	w, h = term.Size()
	assert.Equal(t, w, 40)
	assert.Equal(t, h, 10)
	assert.Equal(t, len(calls), 1)
}

func TestSetSizeRedrawsEveryCellAtTheNewSize(t *testing.T) {
	var out strings.Builder
	term := NewTestTerminal(10, 2, &out)
	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	frame.PrintStyled(0, 0, "hello", NewStyle())
	assert.NoError(t, term.EndFrame(frame))

	term.SetSize(6, 2)
	out.Reset()
	frame, err = term.BeginFrame()
	assert.NoError(t, err)
	w, _ := frame.Size()
	assert.Equal(t, w, 6)
	frame.PrintStyled(0, 0, "hello", NewStyle())
	assert.NoError(t, term.EndFrame(frame))
	// The front buffer was invalidated, so the unchanged text is sent again.
	assert.Contains(t, out.String(), "hello")
}
