package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTerminal_EnableDisableAlternateScreen(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 5, buf)

	term.EnableAlternateScreen()
	term.EnableAlternateScreen()
	assert.Equal(t, buf.String(), "\033[?1049h")
	assert.True(t, term.altScreen)

	buf.Reset()
	term.DisableAlternateScreen()
	term.DisableAlternateScreen()
	assert.Equal(t, buf.String(), "\033[?1049l")
	assert.True(t, !term.altScreen)
}

func TestTerminal_EnableDisableBracketedPaste(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 5, buf)

	term.EnableBracketedPaste()
	term.EnableBracketedPaste()
	assert.Equal(t, buf.String(), "\033[?2004h")
	assert.True(t, term.bracketedPaste)

	buf.Reset()
	term.DisableBracketedPaste()
	term.DisableBracketedPaste()
	assert.Equal(t, buf.String(), "\033[?2004l")
	assert.True(t, !term.bracketedPaste)
}

func TestTerminal_EnableDisableMouseTracking(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 5, buf)

	term.EnableMouseTracking()
	assert.Equal(t, buf.String(), "\033[?1006h\033[?1000h\033[?1003h")
	assert.Equal(t, term.MouseMode(), MouseModeTracking)

	buf.Reset()
	term.DisableMouseTracking()
	assert.Equal(t, buf.String(), "\033[?1000l\033[?1002l\033[?1003l\033[?1006l")
	assert.Equal(t, term.MouseMode(), MouseModeOff)
}

func TestTerminal_EnableMouseButtons(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 5, buf)

	term.EnableMouseButtons()
	assert.Equal(t, buf.String(), "\033[?1006h\033[?1000h")
	assert.Equal(t, term.MouseMode(), MouseModeButtons)

	buf.Reset()
	term.DisableMouseTracking()
	assert.Equal(t, buf.String(), "\033[?1000l\033[?1002l\033[?1003l\033[?1006l")
	assert.Equal(t, term.MouseMode(), MouseModeOff)
}

func TestTerminal_EnableMouseDrag(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 5, buf)

	// Drag mode adds ?1002 (motion while held) but never ?1003 (hover).
	term.EnableMouseDrag()
	assert.Equal(t, buf.String(), "\033[?1006h\033[?1000h\033[?1002h")
	assert.Equal(t, term.MouseMode(), MouseModeDrag)

	buf.Reset()
	term.DisableMouseTracking()
	assert.Equal(t, buf.String(), "\033[?1000l\033[?1002l\033[?1003l\033[?1006l")
	assert.Equal(t, term.MouseMode(), MouseModeOff)
}

func TestTerminal_EnableDisableEnhancedKeyboard(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 5, buf)

	term.EnableEnhancedKeyboard()
	term.EnableEnhancedKeyboard()
	assert.Equal(t, buf.String(), "\033[>1u")
	assert.True(t, term.IsKittyProtocolEnabled())

	buf.Reset()
	term.DisableEnhancedKeyboard()
	term.DisableEnhancedKeyboard()
	assert.Equal(t, buf.String(), "\033[<u")
	assert.True(t, !term.IsKittyProtocolEnabled())
}

func TestTerminal_KittySupportAccessors(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})
	assert.True(t, !term.IsKittyProtocolSupported())
	term.kittySupported = true
	assert.True(t, term.IsKittyProtocolSupported())
}

func TestTerminal_EnableDisableRawMode_TestMode(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})

	err := term.EnableRawMode()
	assert.NoError(t, err)
	assert.True(t, term.rawMode)

	err = term.DisableRawMode()
	assert.NoError(t, err)
	assert.True(t, !term.rawMode)
}

func TestTerminal_RefreshSize_NoOpInTestMode(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})
	calls := 0
	term.OnResize(func(width, height int) {
		calls++
	})

	err := term.RefreshSize()
	assert.NoError(t, err)
	assert.Equal(t, term.width, 10)
	assert.Equal(t, term.height, 5)
	assert.Equal(t, calls, 0)
}

func TestTerminal_MoveCursorClamp(t *testing.T) {
	term := NewTestTerminal(5, 4, &bytes.Buffer{})
	term.MoveCursor(2, 2)

	term.MoveCursorUp(10)
	assert.Equal(t, term.virtualY, 0)

	term.MoveCursorLeft(10)
	assert.Equal(t, term.virtualX, 0)

	term.MoveCursorDown(10)
	assert.Equal(t, term.virtualY, 3)

	term.MoveCursorRight(10)
	assert.Equal(t, term.virtualX, 4)
}

func TestTerminal_BypassInputUpdatesBuffers(t *testing.T) {
	term := NewTestTerminal(3, 2, &bytes.Buffer{})
	term.BypassInput("a\nb\nc")

	assert.Equal(t, term.backBuffer[0][0].Char, 'b')
	assert.Equal(t, term.backBuffer[1][0].Char, 'c')
	assert.Equal(t, term.frontBuffer[0][0].Char, 'b')
	assert.Equal(t, term.frontBuffer[1][0].Char, 'c')
	assert.Equal(t, term.virtualX, 1)
	assert.Equal(t, term.virtualY, 1)
}

func TestTerminal_BypassInputPreservesGraphemeClusters(t *testing.T) {
	term := NewTestTerminal(10, 2, &bytes.Buffer{})
	term.BypassInput("#\uFE0F\u20E3A")

	assert.Equal(t, '#', term.backBuffer[0][0].Char)
	assert.Equal(t, "\uFE0F\u20E3", term.backBuffer[0][0].Trailing)
	assert.Equal(t, 2, term.backBuffer[0][0].Width)
	assert.Equal(t, true, term.backBuffer[0][1].Continuation)
	assert.Equal(t, 'A', term.backBuffer[0][2].Char)

	assert.Equal(t, '#', term.frontBuffer[0][0].Char)
	assert.Equal(t, "\uFE0F\u20E3", term.frontBuffer[0][0].Trailing)
	assert.Equal(t, 'A', term.frontBuffer[0][2].Char)
	assert.Equal(t, 3, term.virtualX)
	assert.Equal(t, 0, term.virtualY)
}

func TestTerminal_FlushIsWrappedInSynchronizedOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 5, buf)

	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	assert.NoError(t, frame.PrintStyled(0, 0, "hi", NewStyle()))
	assert.NoError(t, term.EndFrame(frame))

	out := buf.String()
	// The whole repaint arrives between the markers, so a terminal that
	// implements DEC 2026 shows it in one piece instead of tearing.
	assert.True(t, strings.HasPrefix(out, "\033[?2026h"), "flush must begin with sync-on, got %q", out)
	assert.True(t, strings.HasSuffix(out, "\033[?2026l"), "flush must end with sync-off, got %q", out)
	assert.Contains(t, out, "hi")
}

func TestTerminal_InvalidateForcesAFullRedraw(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 2, buf)

	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	assert.NoError(t, frame.PrintStyled(0, 0, "hi", NewStyle()))
	assert.NoError(t, term.EndFrame(frame))

	// Nothing changed, so a second identical frame sends no cells.
	buf.Reset()
	frame, err = term.BeginFrame()
	assert.NoError(t, err)
	assert.NoError(t, frame.PrintStyled(0, 0, "hi", NewStyle()))
	assert.NoError(t, term.EndFrame(frame))
	assert.False(t, strings.Contains(buf.String(), "hi"), "an unchanged frame should re-send nothing")

	// After something else has written to the screen, the terminal's picture of
	// it is worthless and every cell has to go out again.
	term.Invalidate()
	buf.Reset()
	frame, err = term.BeginFrame()
	assert.NoError(t, err)
	assert.NoError(t, frame.PrintStyled(0, 0, "hi", NewStyle()))
	assert.NoError(t, term.EndFrame(frame))
	assert.Contains(t, buf.String(), "hi")
}
