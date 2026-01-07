package terminal

import (
	"bytes"
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
	assert.True(t, term.mouseEnabled)

	buf.Reset()
	term.DisableMouseTracking()
	assert.Equal(t, buf.String(), "\033[?1000l\033[?1003l\033[?1006l")
	assert.True(t, !term.mouseEnabled)
}

func TestTerminal_EnableMouseButtons(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(10, 5, buf)

	term.EnableMouseButtons()
	assert.Equal(t, buf.String(), "\033[?1006h\033[?1000h")
	assert.True(t, term.mouseEnabled)

	buf.Reset()
	term.DisableMouseTracking()
	assert.Equal(t, buf.String(), "\033[?1000l\033[?1003l\033[?1006l")
	assert.True(t, !term.mouseEnabled)
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
