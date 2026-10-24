package termtest

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestCancelledCSIKeepsNextCharacter(t *testing.T) {
	// CAN (0x18) and SUB (0x1A) abort a control sequence. The next printable
	// byte is text, not the sequence's final byte.
	for _, abort := range []string{"\x18", "\x1a"} {
		s := NewScreen(20, 2)
		s.Write([]byte("\x1b[12" + abort + "Hello"))
		assert.Equal(t, "Hello", s.Row(0), "abort %q", abort)
	}

	// ESC inside a control sequence starts a new sequence.
	s := NewScreen(20, 2)
	s.Write([]byte("\x1b[12\x1b[1;3HHello"))
	assert.Equal(t, "  Hello", s.Row(0))
}

func TestNewScreenZeroSize(t *testing.T) {
	s := NewScreen(0, 0)
	// None of these may panic.
	s.Write([]byte("abc\r\n\x1b[2K\x1b[J"))
	s.ClearLine()
	s.SetCursor(5, 5)
	w, h := s.Size()
	assert.True(t, w >= 1 && h >= 1)
}
