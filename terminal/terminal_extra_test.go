package terminal

import (
	"bytes"
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTerminalRenderFrame_SubFrameBounds(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})
	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	defer term.EndFrame(frame)

	sub := frame.SubFrame(image.Rect(2, 1, 7, 4))
	width, height := sub.Size()
	assert.Equal(t, width, 5)
	assert.Equal(t, height, 3)
	assert.Equal(t, sub.GetBounds(), image.Rect(2, 1, 7, 4))
}

func TestTerminalRenderFrame_SubFrameEmpty(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})
	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	defer term.EndFrame(frame)

	sub := frame.SubFrame(image.Rect(20, 20, 25, 25))
	width, height := sub.Size()
	assert.Equal(t, width, 0)
	assert.Equal(t, height, 0)
	assert.Equal(t, sub.GetBounds(), image.Rectangle{})
}

func TestTerminalRenderFrame_SubFrameTranslation(t *testing.T) {
	term := NewTestTerminal(5, 5, &bytes.Buffer{})
	frame, err := term.BeginFrame()
	assert.NoError(t, err)

	sub := frame.SubFrame(image.Rect(2, 1, 4, 3))
	err = sub.PrintStyled(0, 0, "Z", NewStyle())
	assert.NoError(t, err)

	assert.NoError(t, term.EndFrame(frame))
	assert.Equal(t, term.backBuffer[1][2].Char, 'Z')
}

func TestTerminalRenderFrame_PrintTruncatedNoWrap(t *testing.T) {
	term := NewTestTerminal(3, 2, &bytes.Buffer{})
	frame, err := term.BeginFrame()
	assert.NoError(t, err)

	sub := frame.SubFrame(image.Rect(0, 0, 3, 1))
	err = sub.PrintTruncated(0, 0, "hello", NewStyle())
	assert.NoError(t, err)

	assert.NoError(t, term.EndFrame(frame))
	assert.Equal(t, term.backBuffer[0][0].Char, 'h')
	assert.Equal(t, term.backBuffer[0][1].Char, 'e')
	assert.Equal(t, term.backBuffer[0][2].Char, 'l')
	assert.Equal(t, term.backBuffer[1][0].Char, ' ')
}

func TestTerminal_EndFrame_InvalidFrame(t *testing.T) {
	term := NewTestTerminal(3, 3, &bytes.Buffer{})
	other := NewTestTerminal(3, 3, &bytes.Buffer{})

	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	defer term.EndFrame(frame)

	err = other.EndFrame(frame)
	assert.ErrorIs(t, err, ErrInvalidFrame)
}

func TestTerminal_WriteRaw(t *testing.T) {
	buf := &bytes.Buffer{}
	term := NewTestTerminal(3, 3, buf)

	err := term.WriteRaw([]byte("abc"))
	assert.NoError(t, err)
	assert.Equal(t, buf.String(), "abc")
}

func TestTerminal_WriteRaw_Closed(t *testing.T) {
	term := NewTestTerminal(3, 3, &bytes.Buffer{})
	term.out = nil

	err := term.WriteRaw([]byte("a"))
	assert.ErrorIs(t, err, ErrClosed)
}

func TestTerminal_GetCell(t *testing.T) {
	term := NewTestTerminal(3, 3, &bytes.Buffer{})

	err := term.SetCell(1, 1, 'A', NewStyle())
	assert.NoError(t, err)

	cell := term.GetCell(1, 1)
	assert.Equal(t, cell.Char, 'A')

	assert.Equal(t, term.GetCell(-1, 0).Char, ' ')
	assert.Equal(t, term.GetCell(0, -1).Char, ' ')
	assert.Equal(t, term.GetCell(10, 10).Char, ' ')
}
