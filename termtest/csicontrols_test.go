package termtest

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestC0ControlsInsideCSI(t *testing.T) {
	cases := []struct {
		name, seq string
		row       int
		want      string
	}{
		{"CR", "abc\x1b[2\rCX", 0, "abX"},
		{"LF", "abc\r\x1b[2\nCX", 1, "  X"},
		{"BS", "abcd\x1b[\x082CX", 0, "abcd X"},
		{"HT", "a\x1b[\t1CX", 0, "a        X"},
		{"VT", "abc\x1b[\x0b1CX", 1, "    X"},
		{"BEL ignored", "abc\x1b[2\x07CX", 0, "abc  X"},
		{"DEL ignored", "abc\x1b[2\x7fCX", 0, "abc  X"},
		{"CR inside SGR", "abc\x1b[3\r1mX", 0, "Xbc"},
		{"two controls", "abc\x1b[\r\n2CX", 1, "  X"},
	}
	for _, tc := range cases {
		s := NewScreen(20, 3)
		s.Write([]byte(tc.seq))
		assert.Equal(t, s.Row(tc.row), tc.want, tc.name)
	}
}

// A control inside a CSI split across writes runs once.
func TestC0ControlInsideSplitCSI(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("abc\x1b[2\n"))
	s.Write([]byte("CX"))
	assert.Equal(t, s.Row(1), "  X")
	_, y := s.Cursor()
	assert.Equal(t, y, 1)
}

// CAN aborts the sequence; controls before it still run.
func TestC0ControlBeforeCAN(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("abc\x1b[2\r\x18X"))
	assert.Equal(t, s.Row(0), "Xbc")
}
