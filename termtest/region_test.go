package termtest

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// rows returns every row of s as text.
func rows(s *Screen) []string {
	_, h := s.Size()
	out := make([]string, h)
	for y := range out {
		out[y] = s.Row(y)
	}
	return out
}

func numbered(w, h int) *Screen {
	s := NewScreen(w, h)
	for y := 1; y <= h; y++ {
		if y > 1 {
			s.WriteString("\r\n")
		}
		s.WriteString(string(rune('0' + y)))
	}
	return s
}

func TestScrollRegionLineFeed(t *testing.T) {
	s := numbered(10, 5)
	s.Write([]byte("\x1b[2;4r"))
	x, y := s.Cursor()
	assert.Equal(t, [2]int{x, y}, [2]int{0, 0}, "DECSTBM moves the cursor home")

	s.Write([]byte("\x1b[4;1H\nX"))
	assert.Equal(t, rows(s), []string{"1", "3", "4", "X", "5"})
	assert.Equal(t, len(s.Scrollback()), 0, "a region below row 1 drops departing rows")
}

func TestScrollRegionIgnoresInvalid(t *testing.T) {
	for _, seq := range []string{"\x1b[3;2r", "\x1b[3;3r"} {
		s := numbered(10, 4)
		s.Write([]byte("abc" + seq + "\x1b[4;1H\nX"))
		assert.Equal(t, rows(s), []string{"2", "3", "4abc", "X"}, seq)
	}
}

func TestScrollRegionDefaults(t *testing.T) {
	t.Run("top only", func(t *testing.T) {
		s := numbered(10, 4)
		s.Write([]byte("\x1b[2r\x1b[4;1H\nX"))
		assert.Equal(t, rows(s), []string{"1", "3", "4", "X"})
	})
	t.Run("reset", func(t *testing.T) {
		s := numbered(10, 4)
		s.Write([]byte("\x1b[2;3r\x1b[r\x1b[4;1H\nX"))
		assert.Equal(t, rows(s), []string{"2", "3", "4", "X"})
	})
	t.Run("bottom past the screen", func(t *testing.T) {
		s := numbered(10, 4)
		s.Write([]byte("\x1b[2;99r\x1b[4;1H\nX"))
		assert.Equal(t, rows(s), []string{"1", "3", "4", "X"})
	})
}

func TestScrollRegionIndexAndReverseIndex(t *testing.T) {
	s := numbered(10, 4)
	s.Write([]byte("\x1b[2;3r\x1b[3;1H\x1bDX"))
	assert.Equal(t, rows(s), []string{"1", "3", "X", "4"})

	s = numbered(10, 4)
	s.Write([]byte("\x1b[2;3r\x1b[2;1H\x1bMX"))
	assert.Equal(t, rows(s), []string{"1", "X", "2", "4"})

	s = numbered(10, 4)
	s.Write([]byte("\x1b[2;3r\x1b[3;4H\x1bEX"))
	assert.Equal(t, rows(s), []string{"1", "3", "X", "4"})
}

func TestScrollRegionAutowrap(t *testing.T) {
	s := numbered(5, 4)
	s.Write([]byte("\x1b[1;3r\x1b[3;1Habcdefg"))
	assert.Equal(t, rows(s), []string{"2", "abcde", "fg", "4"})
	assert.Equal(t, s.Scrollback(), []string{"1"})
}

func TestScrollRegionCursorOutside(t *testing.T) {
	s := numbered(10, 4)
	s.Write([]byte("\x1b[1;2r\x1b[3;1H\nX"))
	assert.Equal(t, rows(s), []string{"1", "2", "3", "X"}, "below the region LF moves down")

	s.Write([]byte("\nY"))
	assert.Equal(t, rows(s), []string{"1", "2", "3", "Y"}, "on the last row LF does not scroll")
}

func TestScrollRegionCursorMargins(t *testing.T) {
	cases := []struct {
		name, seq string
		x, y      int
	}{
		{"CUU stops at top margin", "\x1b[2;4r\x1b[3;1H\x1b[9A", 0, 1},
		{"CUD stops at bottom margin", "\x1b[2;4r\x1b[3;1H\x1b[9B", 0, 3},
		{"CUU above the region", "\x1b[3;4r\x1b[2;1H\x1b[9A", 0, 0},
		{"CUD below the region", "\x1b[2;3r\x1b[4;1H\x1b[9B", 0, 4},
		{"CNL stops at bottom margin", "\x1b[2;4r\x1b[3;5H\x1b[9E", 0, 3},
		{"CPL stops at top margin", "\x1b[2;4r\x1b[3;5H\x1b[9F", 0, 1},
	}
	for _, tc := range cases {
		s := NewScreen(10, 5)
		s.Write([]byte(tc.seq))
		x, y := s.Cursor()
		assert.Equal(t, [2]int{x, y}, [2]int{tc.x, tc.y}, tc.name)
	}
}

func TestScrollRegionInsertDeleteLines(t *testing.T) {
	s := numbered(10, 5)
	s.Write([]byte("\x1b[2;4r\x1b[2;3H\x1b[LX"))
	assert.Equal(t, rows(s), []string{"1", "X", "2", "3", "5"}, "IL stops at the bottom margin and homes the column")

	s = numbered(10, 5)
	s.Write([]byte("\x1b[2;4r\x1b[2;3H\x1b[2MX"))
	assert.Equal(t, rows(s), []string{"1", "X", "", "", "5"})

	s = numbered(10, 5)
	s.Write([]byte("\x1b[2;3r\x1b[5;1H\x1b[M\x1b[L"))
	assert.Equal(t, rows(s), []string{"1", "2", "3", "4", "5"}, "IL and DL outside the region do nothing")
}

func TestScrollUpDown(t *testing.T) {
	s := numbered(10, 4)
	s.Write([]byte("\x1b[1S\x1b[4;1H\x1b[2T"))
	assert.Equal(t, rows(s), []string{"", "", "2", "3"})
	assert.Equal(t, s.Scrollback(), []string{"1"})

	s = numbered(10, 5)
	s.Write([]byte("\x1b[2;4r\x1b[1T"))
	assert.Equal(t, rows(s), []string{"1", "", "2", "3", "5"})

	s = numbered(10, 5)
	s.Write([]byte("\x1b[2;4r\x1b[9S"))
	assert.Equal(t, rows(s), []string{"1", "", "", "", "5"})
	assert.Equal(t, len(s.Scrollback()), 0)
}

func TestOriginMode(t *testing.T) {
	s := NewScreen(10, 5)
	s.Write([]byte("\x1b[2;4r\x1b[?6h\x1b[2;3HX\x1b[9;9HY"))
	assert.Equal(t, rows(s), []string{"", "", "  X", "        Y", ""})

	s.Write([]byte("\x1b[1dZ"))
	assert.Equal(t, s.Row(1), "         Z", "VPA counts from the region top")

	s.Write([]byte("\x1b[?6l"))
	x, y := s.Cursor()
	assert.Equal(t, [2]int{x, y}, [2]int{0, 0}, "leaving origin mode homes the cursor")

	s = NewScreen(10, 5)
	s.Write([]byte("\x1b[?6h\x1b[2;4r"))
	x, y = s.Cursor()
	assert.Equal(t, [2]int{x, y}, [2]int{0, 1}, "DECSTBM in origin mode homes to the region top")
}

func TestOriginModeSavedWithCursor(t *testing.T) {
	s := NewScreen(10, 5)
	s.Write([]byte("\x1b[2;4r\x1b[?6h\x1b7\x1b[?6l\x1b8\x1b[1;1HX"))
	assert.Equal(t, s.Row(1), "X")
}

func TestResetClearsRegionAndScrollback(t *testing.T) {
	s := numbered(10, 4)
	s.Write([]byte("\x1b[2;3r\x1b[?6h\x1bc"))
	assert.Equal(t, len(s.Scrollback()), 0)
	s.Write([]byte("1\r\n2\r\n3\r\n4\r\n5"))
	assert.Equal(t, rows(s), []string{"2", "3", "4", "5"})
	assert.Equal(t, s.Scrollback(), []string{"1"})
}

// The sequence InlineApp writes to print above a live region: save the
// cursor, set a region above the live region, add rows at its bottom,
// reset the region, and restore the cursor.
func TestInlinePrintThroughRegion(t *testing.T) {
	s := NewScreen(10, 6)
	s.Write([]byte("a\r\nb\r\nc\r\nd\r\nL1\r\nL2"))
	s.Write([]byte("\x1b7\x1b[1;4r\x1b[4;1H\x1b[0m\r\nnew1\x1b[0m\r\n\x1b[31;44mnew2\x1b[0m\x1b[r\x1b8"))

	assert.Equal(t, rows(s), []string{"c", "d", "new1", "new2", "L1", "L2"})
	assert.Equal(t, s.Scrollback(), []string{"a", "b"})
	x, y := s.Cursor()
	assert.Equal(t, [2]int{x, y}, [2]int{2, 5})
	assert.Equal(t, s.Cell(0, 3).Style.Background, Color{Type: ColorBasic, Value: 4})
	assert.Equal(t, s.Cell(5, 3).Style, Style{}, "rows scrolled in after SGR 0 have the default background")
}
