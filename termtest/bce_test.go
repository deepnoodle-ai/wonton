package termtest

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

var blue = Color{Type: ColorBasic, Value: 4}

func TestBackgroundColorErase(t *testing.T) {
	cases := []struct {
		name string
		w, h int
		seq  string
		x, y int // a cell that the operation filled
	}{
		{"EL", 10, 2, "\x1b[44mab\x1b[K", 5, 0},
		{"EL 1", 10, 2, "abcdefghij\x1b[1;5H\x1b[44m\x1b[1K", 2, 0},
		{"ED 2", 10, 2, "\x1b[44m\x1b[2J", 3, 1},
		{"ED 1", 10, 2, "abcdefghij\r\nabcdefghij\x1b[2;5H\x1b[44m\x1b[1J", 2, 0},
		{"ECH", 10, 2, "abcdef\x1b[1;2H\x1b[44m\x1b[3X", 2, 0},
		{"DCH", 10, 2, "abcdefghij\x1b[1;3H\x1b[44m\x1b[2P", 9, 0},
		{"ICH", 10, 2, "abcdef\x1b[1;2H\x1b[44m\x1b[2@", 2, 0},
		{"IL", 10, 3, "1\r\n2\r\n3\x1b[2;1H\x1b[44m\x1b[L", 4, 1},
		{"DL", 10, 3, "1\r\n2\r\n3\x1b[1;1H\x1b[44m\x1b[M", 4, 2},
		{"SU", 10, 3, "1\r\n2\r\n3\x1b[44m\x1b[S", 4, 2},
		{"SD", 10, 3, "1\r\n2\r\n3\x1b[44m\x1b[T", 4, 0},
		{"LF scroll", 10, 3, "1\r\n2\r\n3\x1b[44m\r\n", 4, 2},
		{"RI scroll", 10, 3, "1\r\n2\r\n3\x1b[1;1H\x1b[44m\x1bM", 4, 0},
	}
	for _, tc := range cases {
		s := NewScreen(tc.w, tc.h)
		s.Write([]byte(tc.seq))
		assert.Equal(t, s.Cell(tc.x, tc.y), Cell{Char: ' ', Width: 1, Style: Style{Background: blue}}, tc.name)
	}
}

func TestBackgroundColorEraseOnlyBackground(t *testing.T) {
	s := NewScreen(10, 2)
	s.Write([]byte("\x1b[1;4;31;44mab\x1b[K"))
	assert.Equal(t, s.Cell(5, 0).Style, Style{Background: blue})
}

func TestBackgroundColorEraseShiftsWithDCH(t *testing.T) {
	s := NewScreen(10, 2)
	s.Write([]byte("\x1b[44mab\x1b[K\x1b[0m\x1b[1;1H\x1b[P"))
	assert.Equal(t, s.Cell(1, 0).Style.Background, blue, "an erased cell keeps its background when it shifts")
	assert.Equal(t, s.Cell(9, 0).Style, Style{}, "the new cell takes the current (default) background")
}

func TestResetErasesWithDefaultBackground(t *testing.T) {
	s := NewScreen(10, 2)
	s.Write([]byte("\x1b[44mab\x1bc"))
	assert.Equal(t, s.Cell(0, 0), Cell{Char: ' ', Width: 1})
}
