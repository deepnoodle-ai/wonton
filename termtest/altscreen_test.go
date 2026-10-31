package termtest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// TestAlternateScreenMatchesGhostty pins the alternate screen modes (?47,
// ?1047, ?1049) and the saved-cursor rules they share with DECSC. The expected
// rows and cursor came from libghostty-vt (commit 9bfa8ee) on a 10x3 screen,
// so the ghostty comparison runs with go test and needs no harness.
func TestAlternateScreenMatchesGhostty(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		rows   []string
		cx, cy int
	}{
		{"1049 enter shows blank alt", "primary\x1b[?1049h", []string{"", "", ""}, 7, 0},
		{"1049 enter keeps cursor col", "primary\x1b[?1049hX", []string{"       X", "", ""}, 8, 0},
		{"1049 round trip restores primary", "primary\x1b[?1049h\x1b[2JALT\x1b[?1049l", []string{"primary", "", ""}, 7, 0},
		{"1049 exit restores cursor", "ab\x1b[?1049h\x1b[3;5HALT\x1b[?1049lX", []string{"abX", "", ""}, 3, 0},
		{"1049 reenter alt is cleared", "\x1b[?1049hALT\x1b[?1049l\x1b[?1049h", []string{"", "", ""}, 0, 0},
		{"1049 exit restores style", "\x1b[31ma\x1b[?1049h\x1b[32mb\x1b[?1049lX", []string{"aX", "", ""}, 2, 0},
		{"1047 enter no cursor save", "ab\x1b[?1047hX", []string{"  X", "", ""}, 3, 0},
		{"1047 round trip", "primary\x1b[?1047hALT\x1b[?1047lX", []string{"primary  X", "", ""}, 9, 0},
		{"1047 reenter alt", "\x1b[?1047hALT\x1b[?1047l\x1b[?1047h", []string{"", "", ""}, 3, 0},
		{"1047 enter keeps old alt content", "\x1b[?47hALT\x1b[?47l\x1b[?1047h", []string{"ALT", "", ""}, 3, 0},
		{"47 round trip", "primary\x1b[?47hALT\x1b[?47lX", []string{"primary", "X", ""}, 1, 1},
		{"47 reenter alt keeps content", "\x1b[?47hALT\x1b[?47l\x1b[?47h", []string{"ALT", "", ""}, 3, 0},
		{"47 cursor moves on alt carry back", "ab\x1b[?47h\x1b[3;5HZ\x1b[?47lX", []string{"ab", "", "     X"}, 6, 2},
		{"1049 h twice", "p\x1b[?1049hA\x1b[?1049hB", []string{"  B", "", ""}, 3, 0},
		{"1049 l without h", "ab\x1b[3;5H\x1b[?1049lX", []string{"Xb", "", ""}, 1, 0},
		{"1049 l twice", "p\x1b[?1049hA\x1b[?1049l\x1b[?1049lX", []string{"pX", "", ""}, 2, 0},
		{"mixed 1049h 47l", "p\x1b[?1049hA\x1b[?47lX", []string{"p X", "", ""}, 3, 0},
		{"mixed 47h 1049l", "p\x1b[?47hA\x1b[?1049lX", []string{"X", "", ""}, 1, 0},
		{"DECSC per screen", "ab\x1b7\x1b[?47h\x1b[3;3H\x1b7\x1b[?47l\x1b8X", []string{"abX", "", ""}, 3, 0},
		{"DECSC in alt restored in alt", "ab\x1b[?47h\x1b[3;3H\x1b7\x1b[?47l\x1b[?47h\x1b[H\x1b8X", []string{"", "", "  X"}, 3, 2},
		{"RIS exits alt", "p\x1b[?1049hA\x1bcX", []string{"X", "", ""}, 1, 0},
		{"multi param 1049;25", "p\x1b[?25;1049hA\x1b[?1049;25lX", []string{"pX", "", ""}, 2, 0},
		{"pending wrap across switch", "abcdefghij\x1b[?1049hX", []string{"", "X", ""}, 1, 1},
		{"pending wrap across 1049 exit", "\x1b[?1049habcdefghij\x1b[?1049lX", []string{"X", "", ""}, 1, 0},
		{"scroll in alt", "\x1b[?1049h1\r\n2\r\n3\r\n4\x1b[?1049l", []string{"", "", ""}, 0, 0},
		{"1049 save shares DECSC slot", "a\x1b7\x1b[2;4H\x1b[?1049h\x1b[?1049l\x1b[3;8H\x1b8X", []string{"a", "   X", ""}, 4, 1},
		{"alt DECSC initial", "ab\x1b[?1049h\x1b[3;5H\x1b8X", []string{"X", "", ""}, 1, 0},
		{"CSI s shares slot", "a\x1b[s\x1b[3;8H\x1b8X", []string{"aX", "", ""}, 2, 0},
		{"47 SGR carried", "\x1b[31m\x1b[?47hA\x1b[?47lX", []string{" X", "", ""}, 2, 0},
		{"1049 ED in alt keeps primary", "pp\x1b[?1049h\x1b[2J\x1b[?1049l", []string{"pp", "", ""}, 2, 0},
		{"1049 restores pending wrap", "abcdefghij\x1b[?1049h\x1b[2;2HA\x1b[?1049lX", []string{"abcdefghij", "X", ""}, 1, 1},
		{"DECSC restores pending wrap", "abcdefghij\x1b7\x1b[2;2HA\x1b8X", []string{"abcdefghij", "XA", ""}, 1, 1},
		{"47h keeps pending wrap", "abcdefghij\x1b[?47hX", []string{"", "X", ""}, 1, 1},
		{"1047h keeps pending wrap", "abcdefghij\x1b[?1047hX", []string{"", "X", ""}, 1, 1},
		{"1047l on primary clears nothing", "abc\x1b[?1047lX", []string{"abcX", "", ""}, 4, 0},
		{"47l from primary", "abcdefghij\x1b[?47lX", []string{"abcdefghij", "X", ""}, 1, 1},
		{"1049h from alt via 47", "p\x1b[?47hA\x1b[?1049h\x1b[?1049lX", []string{"X", "", ""}, 1, 0},
		{"CSI s saves pending wrap", "abcdefghij\x1b[s\x1b[2;2HA\x1b[uX", []string{"abcdefghij", "XA", ""}, 1, 1},
		{"CSI s then ESC 8 wrap", "abcdefghij\x1b[s\x1b[2;2HA\x1b8X", []string{"abcdefghij", "XA", ""}, 1, 1},
		{"ESC 7 then CSI u wrap", "abcdefghij\x1b7\x1b[2;2HA\x1b[uX", []string{"abcdefghij", "XA", ""}, 1, 1},
		{"1049 styled frame then exit", "$ ok\r\n\x1b[?1049h\x1b[H\x1b[44;1mAPP\x1b[0m\x1b[?1049lnext", []string{"$ ok", "next", ""}, 4, 1},
		{"1049 restore keeps saved SGR", "\x1b[31ma\x1b[?1049h\x1b[32mb\x1b[?1049lX", []string{"aX", "", ""}, 2, 0},
		{"ANSI mode 47 (no ?) ignored", "p\x1b[47hA", []string{"pA", "", ""}, 2, 0},
		{"nested 1049h restores first save", "a\r\nb\x1b[?1049h\x1b[3;6H\x1b[?1049h\x1b[?1049lX", []string{"a", "bX", ""}, 2, 1},
		{"nested 1049h saves into alt slot", "a\r\nb\x1b[?1049h\x1b[3;6H\x1b[?1049h\x1b[H\x1b8X", []string{"", "", "     X"}, 6, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewScreen(10, 3)
			s.Write([]byte(tc.data))
			for y, want := range tc.rows {
				assert.Equal(t, want, strings.TrimRight(s.Row(y), " "), "row %d", y)
			}
			x, y := s.Cursor()
			assert.Equal(t, [2]int{tc.cx, tc.cy}, [2]int{x, y}, "cursor")
		})
	}
}

func TestAlternateScreenFlag(t *testing.T) {
	s := NewScreen(10, 3)
	assert.False(t, s.IsAlternateScreen())
	for _, mode := range []string{"47", "1047", "1049"} {
		s.Write([]byte("\x1b[?" + mode + "h"))
		assert.True(t, s.IsAlternateScreen(), mode)
		s.Write([]byte("\x1b[?" + mode + "l"))
		assert.False(t, s.IsAlternateScreen(), mode)
	}
	s.Write([]byte("\x1b[?25;1049h"))
	assert.True(t, s.IsAlternateScreen(), "multiple parameters")
	s.Write([]byte("\x1bc"))
	assert.False(t, s.IsAlternateScreen(), "RIS returns to the primary screen")
	s.Write([]byte("\x1b[?1049h"))
	s.Reset()
	assert.False(t, s.IsAlternateScreen(), "Reset returns to the primary screen")
	s.Write([]byte("\x1b[47h"))
	assert.False(t, s.IsAlternateScreen(), "ANSI mode 47 is not the alternate screen")
}

// Everything a test reads shows the visible screen: cells, styles, Text,
// Contains, and equality with another screen.
func TestAlternateScreenReadsShowVisibleScreen(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("\x1b[1mshell\x1b[0m"))
	s.Write([]byte("\x1b[?1049h\x1b[H\x1b[44mAPP\x1b[0m"))

	assert.Equal(t, "APP", strings.TrimRight(s.Row(0), " "))
	assert.False(t, s.Contains("shell"))
	assert.Equal(t, ColorBasic, s.Cell(0, 0).Style.Background.Type)
	assert.False(t, s.Cell(0, 0).Style.Bold)

	want := NewScreen(10, 3)
	want.Write([]byte("\x1b[44mAPP"))
	assert.True(t, EqualStyled(want, s), "compares the visible screen")

	s.Write([]byte("\x1b[?1049l"))
	assert.Equal(t, "shell", strings.TrimRight(s.Row(0), " "))
	assert.True(t, s.Cell(0, 0).Style.Bold, "primary styles come back")
	assert.False(t, s.Contains("APP"))
}

// SetCell and Clear act on the visible screen and leave the hidden one alone.
func TestAlternateScreenWritesGoToVisibleScreen(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("shell"))
	s.Write([]byte("\x1b[?1049h"))
	s.SetCell(0, 1, 'Z', Style{})
	s.Clear()
	s.SetCell(0, 2, 'Q', Style{})
	s.Write([]byte("\x1b[?1049l"))
	assert.Equal(t, "shell", strings.TrimRight(s.Row(0), " "))
	assert.Equal(t, "", strings.TrimRight(s.Row(1), " "))
	assert.Equal(t, "", strings.TrimRight(s.Row(2), " "))
}

// A sequence split across writes still switches screens.
func TestAlternateScreenSplitWrite(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("shell\x1b[?10"))
	s.Write([]byte("49hAPP\x1b[?1049"))
	assert.True(t, s.IsAlternateScreen())
	s.Write([]byte("l"))
	assert.Equal(t, "shell", strings.TrimRight(s.Row(0), " "))
}

// A full-screen application draws on the alternate screen. When it leaves,
// the shell screen comes back unchanged, with the cursor where it was.
func ExampleScreen_IsAlternateScreen() {
	screen := NewScreen(20, 3)
	screen.Write([]byte("$ myapp\r\n"))

	screen.Write([]byte("\x1b[?1049h\x1b[HInbox (3)")) // the app starts
	fmt.Println(screen.IsAlternateScreen(), screen.Row(0))

	screen.Write([]byte("\x1b[?1049l")) // the app quits
	x, y := screen.Cursor()
	fmt.Println(screen.IsAlternateScreen(), screen.Row(0), x, y)
	// Output:
	// true Inbox (3)
	// false $ myapp 0 1
}

// A cursor restore also restores the style saved with it: ?1049l, ESC 8
// and CSI u all bring back the red that was active at the save, so the X
// written after the restore is red, not green. Checked against ghostty.
func TestCursorRestoreRestoresStyle(t *testing.T) {
	red := Color{Type: ColorBasic, Value: 1}
	tests := []struct{ name, data string }{
		{"?1049l", "\x1b[31ma\x1b[?1049h\x1b[32mb\x1b[?1049lX"},
		{"ESC 8", "\x1b[31ma\x1b7\x1b[32mb\x1b8X"},
		{"CSI u", "\x1b[31ma\x1b[s\x1b[32mb\x1b[uX"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewScreen(10, 3)
			s.Write([]byte(tc.data))
			assert.Equal(t, "X", s.CellGlyph(1, 0))
			assert.Equal(t, red, s.Cell(1, 0).Style.Foreground)
		})
	}
}
