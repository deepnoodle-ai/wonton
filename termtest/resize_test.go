package termtest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// resizeCases were checked against libghostty-vt, ghostty's terminal
// library: each stream, resize, and following write gives the rows and
// cursor that ghostty shows.
var resizeCases = []struct {
	name          string
	w, h          int
	pre           string
	rw, rh        int
	rows          []string
	cursorX       int
	cursorY       int
	post          string
	rowsAfterPost []string
}{
	{"softwrap narrow", 10, 4, "abcdefghijklmno", 5, 4,
		[]string{"abcde", "fghij", "klmno"}, 0, 3,
		"X", []string{"abcde", "fghij", "klmno", "X"}},
	{"softwrap widen", 5, 4, "abcdefghijklmno", 10, 4,
		[]string{"abcdefghij", "klmno"}, 5, 1,
		"X", []string{"abcdefghij", "klmnoX"}},
	{"hardline narrow", 10, 4, "abcdefgh\r\nxy", 5, 4,
		[]string{"abcde", "fgh", "xy"}, 2, 2,
		"X", []string{"abcde", "fgh", "xyX"}},
	{"hardline widen", 5, 4, "abcde\r\nfg", 10, 4,
		[]string{"abcde", "fg"}, 2, 1,
		"X", []string{"abcde", "fgX"}},
	{"narrow pushes top off", 10, 3, "abcdefgh\r\n12345678\r\nzz", 4, 3,
		[]string{"1234", "5678", "zz"}, 2, 2,
		"X", []string{"1234", "5678", "zzX"}},
	{"shrink height blank below cursor", 10, 5, "one\r\ntwo", 10, 3,
		[]string{"one", "two"}, 3, 1,
		"X", []string{"one", "twoX"}},
	{"shrink height cursor at bottom", 10, 5, "1\r\n2\r\n3\r\n4\r\n5", 10, 3,
		[]string{"3", "4", "5"}, 1, 2,
		"X", []string{"3", "4", "5X"}},
	{"shrink height cursor mid", 10, 5, "1\r\n2\r\n3\r\n4\r\n5\x1b[2;1H", 10, 3,
		[]string{"3", "4", "5"}, 0, 0,
		"X", []string{"X", "4", "5"}},
	{"grow height", 10, 3, "1\r\n2", 10, 5,
		[]string{"1", "2"}, 1, 1,
		"X", []string{"1", "2X"}},
	{"grow both", 5, 3, "abcdefg\r\n2", 10, 6,
		[]string{"abcdefg", "2"}, 1, 1,
		"X", []string{"abcdefg", "2X"}},
	{"wide at edge narrow", 6, 3, "abc中文", 4, 3,
		[]string{"abc", "中文"}, 0, 2,
		"X", []string{"abc", "中文", "X"}},
	{"wide at edge hard line", 6, 3, "a中文\r\nx", 4, 3,
		[]string{"a中", "文", "x"}, 1, 2,
		"X", []string{"a中", "文", "xX"}},
	{"wide cut odd", 7, 3, "ab中文d", 3, 3,
		[]string{"ab", "中", "文d"}, 2, 2,
		"X", []string{"中", "文d", "X"}},
	{"cursor in wrapped line", 10, 4, "abcdefghij\x1b[1;8H", 5, 4,
		[]string{"abcde", "fghij"}, 2, 1,
		"X", []string{"abcde", "fgXij"}},
	{"pending wrap narrow", 10, 4, "abcdefghij", 5, 4,
		[]string{"abcde", "fghij"}, 4, 1,
		"X", []string{"abcde", "fghij", "X"}},
	{"pending wrap widen", 5, 4, "abcdefghij", 10, 4,
		[]string{"abcdefghij"}, 9, 0,
		"X", []string{"abcdefghij", "X"}},
	{"cursor past content", 10, 4, "abc\x1b[1;9H", 5, 4,
		[]string{"abc"}, 3, 1,
		"X", []string{"abc", "   X"}},
	{"cursor on blank row past end", 10, 4, "\x1b[2;9H", 5, 4,
		[]string{}, 3, 2,
		"X", []string{"", "", "   X"}},
	{"printed trailing spaces", 10, 4, "ab        \r\ncd", 5, 4,
		[]string{"ab", "", "cd"}, 2, 2,
		"X", []string{"ab", "", "cdX"}},
	{"printed spaces then erase", 10, 4, "abcdef\x1b[1;3H\x1b[2X\x1b[1;5H\x1b[K\r\nzz", 3, 4,
		[]string{"ab", "zz"}, 2, 1,
		"X", []string{"ab", "zzX"}},
	{"trailing fg spaces", 10, 4, "ab\x1b[31m   \x1b[0m\r\ncd", 3, 4,
		[]string{"ab", "", "cd"}, 2, 2,
		"X", []string{"ab", "", "cdX"}},
	{"saved cursor at end of wrapped line", 10, 6, "abcdefghijklmn\x1b7\x1b[6;1H", 5, 6,
		[]string{"fghij", "klmn"}, 0, 5,
		"\x1b8X", []string{"fghij", "klmnX"}},
	{"saved cursor beyond width", 10, 4, "ab\x1b[1;9H\x1b7\x1b[4;1H", 5, 4,
		[]string{"ab"}, 0, 3,
		"\x1b8X", []string{"ab  X"}},
	{"three hard rows cursor top", 10, 3, "abcdefghij\r\nklmnopqrst\r\nuv\x1b[1;1H", 5, 3,
		[]string{"klmno", "pqrst", "uv"}, 0, 0,
		"X", []string{"Xlmno", "pqrst", "uv"}},
	{"shrink both", 10, 5, "abcdefghij\r\n1\r\n2\r\n3", 5, 3,
		[]string{"1", "2", "3"}, 1, 2,
		"X", []string{"1", "2", "3X"}},
	{"empty rows between", 10, 5, "abcdefgh\r\n\r\n\r\nzz", 4, 5,
		[]string{"abcd", "efgh", "", "", "zz"}, 2, 4,
		"X", []string{"abcd", "efgh", "", "", "zzX"}},
	{"EL0 unwraps", 5, 4, "abcdefgh\x1b[1;4H\x1b[K", 10, 4,
		[]string{"abc", "fgh"}, 3, 0,
		"X", []string{"abcX", "fgh"}},
	{"EL2 unwraps", 5, 4, "abcdefgh\x1b[1;1H\x1b[2K", 10, 4,
		[]string{"", "fgh"}, 0, 0,
		"X", []string{"X", "fgh"}},
	{"EL1 keeps wrap", 5, 4, "abcdefgh\x1b[1;5H\x1b[1K", 10, 4,
		[]string{"     fgh"}, 4, 0,
		"X", []string{"    Xfgh"}},
	{"ECH to end unwraps", 5, 4, "abcdefgh\x1b[1;5H\x1b[X", 10, 4,
		[]string{"abcd", "fgh"}, 4, 0,
		"X", []string{"abcdX", "fgh"}},
	{"ECH mid unwraps", 5, 4, "abcdefgh\x1b[1;2H\x1b[X", 10, 4,
		[]string{"a cde", "fgh"}, 1, 0,
		"X", []string{"aXcde", "fgh"}},
	{"DCH unwraps", 5, 4, "abcdefgh\x1b[1;2H\x1b[P", 10, 4,
		[]string{"acde", "fgh"}, 1, 0,
		"X", []string{"aXde", "fgh"}},
	{"ICH keeps wrap", 5, 4, "abcdefgh\x1b[1;2H\x1b[@", 10, 4,
		[]string{"a bcdfgh"}, 1, 0,
		"X", []string{"aXbcdfgh"}},
	{"ED0 unwraps", 5, 4, "abcdefgh\x1b[1;3H\x1b[J", 10, 4,
		[]string{"ab"}, 2, 0,
		"X", []string{"abX"}},
	{"ED1 keeps wrap", 5, 4, "abcdefgh\x1b[1;5H\x1b[1J", 10, 4,
		[]string{"     fgh"}, 4, 0,
		"X", []string{"    Xfgh"}},
	{"ED2 clears", 5, 4, "abcdefgh\x1b[2J\x1b[1;1Hxy\x1b[2;1Hzz", 10, 4,
		[]string{"xy", "zz"}, 2, 1,
		"X", []string{"xy", "zzX"}},
	{"overwrite keeps wrap", 5, 4, "abcdefgh\x1b[1;1HXY", 10, 4,
		[]string{"XYcdefgh"}, 2, 0,
		"X", []string{"XYXdefgh"}},
	{"continuation erased", 5, 4, "abcdefgh\x1b[2;1H\x1b[2K", 10, 4,
		[]string{"abcde"}, 5, 0,
		"X", []string{"abcdeX"}},
	{"IL inside wrapped", 5, 4, "abcdefgh\x1b[2;1H\x1b[L", 10, 4,
		[]string{"abcde", "fgh"}, 5, 0,
		"X", []string{"abcdeX", "fgh"}},
	{"DL wrapped row", 5, 4, "abcdefghijklm\x1b[2;1H\x1b[M", 10, 4,
		[]string{"abcdeklm"}, 5, 0,
		"X", []string{"abcdeXlm"}},
	{"CUP into continuation", 5, 4, "abcdefgh\x1b[2;1HXY", 10, 4,
		[]string{"abcdeXYh"}, 7, 0,
		"X", []string{"abcdeXYX"}},
	{"spacer head", 5, 4, "abcd中x", 10, 4,
		[]string{"abcd中x"}, 7, 0,
		"X", []string{"abcd中xX"}},
	{"spacer head overwritten", 5, 4, "abcd中x\x1b[1;5HZ", 10, 4,
		[]string{"abcdZ中x"}, 5, 0,
		"X", []string{"abcdZX x"}},
	{"RIS", 5, 4, "abcdefgh\x1bcxy", 10, 4,
		[]string{"xy"}, 2, 0,
		"X", []string{"xyX"}},
	{"wrapped row with spaces", 5, 4, "ab   cd", 10, 4,
		[]string{"ab   cd"}, 7, 0,
		"X", []string{"ab   cdX"}},
	{"width 1 wide", 4, 3, "a中b", 1, 3,
		[]string{"a", "", "b"}, 0, 2,
		"X", []string{"", "b", "X"}},
	{"styled text reflows", 10, 3, "\x1b[1;31mabcdefghijkl\x1b[0m", 4, 3,
		[]string{"efgh", "ijkl"}, 0, 2,
		"X", []string{"efgh", "ijkl", "X"}},
	{"grapheme cluster", 6, 3, "ab👍🏽cdé", 3, 3,
		[]string{"ab", "👍🏽c", "dé"}, 2, 2,
		"X", []string{"ab", "👍🏽c", "déX"}},
	{"live region narrow", 20, 6, "scroll\r\n\x1b[2K0123456789abcdef\r\n\x1b[2Kstatus line here", 10, 6,
		[]string{"scroll", "0123456789", "abcdef", "status lin", "e here"}, 6, 4,
		"\x1b[1A\r\x1b[2K0123456789\r\n\x1b[2Kstatus", []string{"scroll", "0123456789", "abcdef", "0123456789", "status"}},
	{"live three lines", 20, 8, "s1\r\ns2\r\n0123456789abcdefghi\r\nshort\r\nmid line 123456", 10, 8,
		[]string{"s1", "s2", "0123456789", "abcdefghi", "short", "mid line 1", "23456"}, 5, 6,
		"X", []string{"s1", "s2", "0123456789", "abcdefghi", "short", "mid line 1", "23456X"}},
	{"full width rows narrow", 10, 4, "aaaaaaaaaa\r\nbbbbbbbbbb\r\ncc", 5, 3,
		[]string{"bbbbb", "bbbbb", "cc"}, 2, 2,
		"X", []string{"bbbbb", "bbbbb", "ccX"}},
	{"narrow then same width height", 10, 4, "abcdefghijkl", 10, 2,
		[]string{"abcdefghij", "kl"}, 2, 1,
		"X", []string{"abcdefghij", "klX"}},
	{"two wraps in a row", 4, 6, "abcdefghijklm", 6, 6,
		[]string{"abcdef", "ghijkl", "m"}, 1, 2,
		"X", []string{"abcdef", "ghijkl", "mX"}},
	{"widen joins three", 3, 6, "abcdefghi", 9, 6,
		[]string{"abcdefghi"}, 8, 0,
		"X", []string{"abcdefghi", "X"}},
	{"widen partial", 3, 6, "abcdefgh", 5, 6,
		[]string{"abcde", "fgh"}, 3, 1,
		"X", []string{"abcde", "fghX"}},
	{"wide widen", 3, 4, "中文字", 6, 4,
		[]string{"中文字"}, 0, 1,
		"X", []string{"中文字", "X"}},

	// The saved cursor stays on its character, as ghostty tracks it.
	{"saved cursor in wrapped line", 10, 4, "abcdefghijklmn\x1b[2;3H\x1b7\x1b[4;1H", 5, 4,
		[]string{"fghij", "klmn"}, 0, 3,
		"\x1b8X", []string{"fghij", "klXn"}},
	{"saved cursor follows reflow", 10, 4, "abcdefghijklmn\x1b[2;3H\x1b7", 5, 4,
		[]string{"abcde", "fghij", "klmn"}, 2, 2,
		"\x1b[4;1H\x1b8X", []string{"abcde", "fghij", "klXn"}},
	{"saved cursor moves up with rows", 10, 5, "1\r\n2\r\n3\x1b7\r\n4\r\n5", 10, 3,
		[]string{"3", "4", "5"}, 1, 2,
		"\x1b8X", []string{"3X", "4", "5"}},
	{"saved cursor row off top goes home", 10, 5, "1\x1b[1;4H\x1b7\x1b[5;1H5", 10, 3,
		[]string{"", "", "5"}, 1, 2,
		"\x1b8X", []string{"X", "", "5"}},
	{"saved pending wrap widen", 5, 4, "abcde\x1b7\r\nx", 10, 4,
		[]string{"abcde", "x"}, 1, 1,
		"\x1b8X", []string{"abcdeX", "x"}},

	// The alternate screen truncates; the primary screen behind it
	// reflows around its own cursor and saved cursor (R-12).
	{"alt narrow truncates", 10, 4, "\x1b[?1049h\x1b[Habcdefghijklmn", 5, 4,
		[]string{"abcde", "klmn"}, 4, 1,
		"X", []string{"abcde", "klmnX"}},
	{"alt widen pads", 5, 4, "\x1b[?1049h\x1b[Habcdefgh", 10, 4,
		[]string{"abcde", "fgh"}, 3, 1,
		"X", []string{"abcde", "fghX"}},
	{"alt wide cut", 6, 3, "\x1b[?1049h\x1b[Habcd中\r\nx", 5, 3,
		[]string{"abcd", "x"}, 1, 1,
		"X", []string{"abcd", "xX"}},
	{"alt wide cut odd", 7, 3, "\x1b[?1049h\x1b[Hab中文d", 4, 3,
		[]string{"ab中"}, 3, 0,
		"X", []string{"ab中", "X"}},
	{"alt cursor clamps", 10, 4, "\x1b[?1049h\x1b[2;9Hab", 5, 4,
		[]string{}, 4, 1,
		"X", []string{"", "", "X"}},
	{"alt pending wrap narrow", 10, 4, "\x1b[?1049h\x1b[Habcdefghij", 5, 4,
		[]string{"abcde"}, 4, 0,
		"X", []string{"abcde", "X"}},
	{"alt pending wrap widen", 5, 4, "\x1b[?1049h\x1b[Habcde", 10, 4,
		[]string{"abcde"}, 5, 0,
		"X", []string{"abcdeX"}},
	{"alt pending wrap same width", 5, 4, "\x1b[?1049h\x1b[Habcde", 5, 3,
		[]string{"abcde"}, 4, 0,
		"X", []string{"abcde", "X"}},
	{"alt shrink height blank below", 10, 5, "\x1b[?1049h\x1b[Hone\r\ntwo", 10, 3,
		[]string{"one", "two"}, 3, 1,
		"X", []string{"one", "twoX"}},
	{"alt shrink height cursor bottom", 10, 5, "\x1b[?1049h\x1b[H1\r\n2\r\n3\r\n4\r\n5", 10, 3,
		[]string{"3", "4", "5"}, 1, 2,
		"X", []string{"3", "4", "5X"}},
	{"alt shrink height cursor mid", 10, 5, "\x1b[?1049h\x1b[H1\r\n2\r\n3\r\n4\r\n5\x1b[2;1H", 10, 3,
		[]string{"3", "4", "5"}, 0, 0,
		"X", []string{"X", "4", "5"}},
	{"alt grow height", 10, 3, "\x1b[?1049h\x1b[H1\r\n2", 10, 5,
		[]string{"1", "2"}, 1, 1,
		"X", []string{"1", "2X"}},
	{"alt shrink both", 10, 5, "\x1b[?1049h\x1b[Habcdefghij\r\n1\r\n2\r\n3", 5, 3,
		[]string{"1", "2", "3"}, 1, 2,
		"X", []string{"1", "2", "3X"}},
	{"alt saved cursor clamps", 10, 4, "\x1b[?1049h\x1b[2;9H\x1b7\x1b[H", 5, 4,
		[]string{}, 0, 0,
		"\x1b8X", []string{"", "    X"}},
	{"alt then primary narrow", 10, 4, "abcdefghijklmn\x1b[?1049h\x1b[Hxyz", 5, 4,
		[]string{"xyz"}, 3, 0,
		"Q\x1b[?1049lX", []string{"abcde", "fghij", "klmnX"}},
	{"alt then primary widen", 5, 4, "abcdefghijk\x1b[?1049h\x1b[Hxyz", 10, 4,
		[]string{"xyz"}, 3, 0,
		"Q\x1b[?1049lX", []string{"abcdefghij", "kX"}},
	{"alt then primary saved mid line", 10, 4, "abcdefghijklmn\x1b[1;3H\x1b[?1049h\x1b[Hxyz", 5, 4,
		[]string{"xyz"}, 3, 0,
		"\x1b[?1049lX", []string{"abXde", "fghij", "klmn"}},
	{"alt then primary pending wrap", 10, 4, "abcdefghij\x1b[?1049h", 5, 4,
		[]string{}, 4, 0,
		"\x1b[?1049lX", []string{"abcde", "fghij", "X"}},
	{"alt then primary shrink height", 10, 5, "1\r\n2\r\n3\r\n4\x1b[?1049h\x1b[Hxyz", 10, 3,
		[]string{"xyz"}, 3, 0,
		"\x1b[?1049lX", []string{"2", "3", "4X"}},
	{"alt then primary rows off top", 10, 3, "abcdefgh\r\n12345678\r\nzz\x1b[?1049h", 4, 3,
		[]string{}, 2, 2,
		"\x1b[?1049lX", []string{"1234", "5678", "zzX"}},
	{"alt clear keeps primary wraps", 5, 4, "abcdefgh\x1b[?1049h\x1b[2J\x1b[Hxyz", 10, 4,
		[]string{"xyz"}, 3, 0,
		"\x1b[?1049lX", []string{"abcdefghX"}},
	{"alt 47 then primary", 10, 4, "abcdefghijklmn\x1b[?47hxyz", 5, 4,
		[]string{"", "    x"}, 4, 1,
		"\x1b[?47lX", []string{"abcde", "fghiX", "klmn"}},
	{"alt 1047 then primary", 10, 4, "abcdefghijklmn\x1b[?1047hxyz", 5, 4,
		[]string{"", "    x"}, 4, 1,
		"\x1b[?1047lX", []string{"abcde", "fghiX", "klmn"}},
	{"primary after alt truncates hidden alt", 10, 4, "\x1b[?47h\x1b[Habcdefghij\x1b[?47l\x1b[Hp", 5, 4,
		[]string{"p"}, 1, 0,
		"\x1b[?47hX", []string{"aXcde"}},
	{"alt 1049 restores reflowed cursor", 10, 4, "abcdefghijklmn\x1b[2;3H\x1b[?1049h", 5, 4,
		[]string{}, 2, 1,
		"\x1b[?1049lX", []string{"abcde", "fghij", "klXn"}},
	{"alt 47 saved cursor in wrapped line", 10, 4, "abcdefghijklmn\x1b[2;3H\x1b7\x1b[4;1H\x1b[?47h", 5, 4,
		[]string{}, 0, 3,
		"\x1b[?47l\x1b8X", []string{"fghij", "klXn"}},
	{"alt 47 primary cursor fits height", 9, 2, "ea\r\n\x1b[?47h\x1b[H", 7, 1,
		[]string{}, 0, 0,
		"\x1b[?47lX", []string{"X"}},
	{"alt 47 primary cursor fits height saved", 9, 2, "ea\r\n\x1b[?47h\x1b[H", 7, 1,
		[]string{}, 0, 0,
		"\x1b[?47l\x1b8X", []string{"X"}},
	{"alt 47 saved cursor follows reflow", 10, 4, "abcdefghijklmn\x1b[2;3H\x1b7\x1b[?47h\x1b[4;1H", 5, 4,
		[]string{}, 0, 3,
		"\x1b[?47l\x1b8X", []string{"abcde", "fghij", "klXn"}},
	{"alt 1049 cursor moved in alt", 10, 4, "abcdefghijklmn\x1b[2;3H\x1b[?1049h\x1b[4;1H", 5, 4,
		[]string{}, 0, 3,
		"\x1b[?1049lX", []string{"abcde", "fghij", "klXn"}},
	{"alt saved pending wrap widen", 5, 4, "\x1b[?1049h\x1b[Habcde\x1b7\r\nx", 10, 4,
		[]string{"abcde", "x"}, 1, 1,
		"\x1b8X", []string{"abcdeX", "x"}},
	{"alt saved pending wrap narrow", 5, 4, "\x1b[?1049h\x1b[Habcde\x1b7\r\nx", 3, 4,
		[]string{"abc", "x"}, 1, 1,
		"\x1b8X", []string{"abc", "X"}},
	{"primary after alt saved cursor", 10, 4, "\x1b[?47h\x1b[2;9H\x1b7\x1b[?47l", 5, 4,
		[]string{}, 3, 2,
		"\x1b[?47h\x1b8X", []string{"", "    X"}},
}

// visibleRows returns the screen rows with trailing blank rows removed.
func visibleRows(s *Screen) []string {
	rows := strings.Split(strings.TrimSuffix(s.Text(), "\n"), "\n")
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	if rows == nil {
		rows = []string{}
	}
	return rows
}

func TestResizeMatchesGhostty(t *testing.T) {
	for _, tc := range resizeCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewScreen(tc.w, tc.h)
			s.Write([]byte(tc.pre))
			s.Resize(tc.rw, tc.rh)

			w, h := s.Size()
			assert.Equal(t, w, tc.rw)
			assert.Equal(t, h, tc.rh)
			want := tc.rows
			if want == nil {
				want = []string{}
			}
			assert.Equal(t, visibleRows(s), want)
			x, y := s.Cursor()
			assert.Equal(t, x, tc.cursorX, "cursor x")
			assert.Equal(t, y, tc.cursorY, "cursor y")

			s.Write([]byte(tc.post))
			assert.Equal(t, visibleRows(s), tc.rowsAfterPost)
		})
	}
}

func TestResizeClampsToOneCell(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("ab"))
	s.Resize(0, -4)
	w, h := s.Size()
	assert.Equal(t, w, 1)
	assert.Equal(t, h, 1)
	x, y := s.Cursor()
	assert.Equal(t, x, 0)
	assert.Equal(t, y, 0)
}

func TestResizeToSameSizeChangesNothing(t *testing.T) {
	s := NewScreen(5, 2)
	s.Write([]byte("abcde"))
	s.Resize(5, 2)
	// The pending wrap survives, so the next character wraps.
	s.Write([]byte("X"))
	assert.Equal(t, visibleRows(s), []string{"abcde", "X"})
}

func TestResizeKeepsStyleAndPartialSequence(t *testing.T) {
	s := NewScreen(10, 2)
	s.Write([]byte("\x1b[1mab\x1b[3"))
	s.Resize(4, 2)
	s.Write([]byte("1mc"))
	cell := s.Cell(2, 0)
	assert.Equal(t, cell.Char, 'c')
	assert.True(t, cell.Style.Bold)
	assert.Equal(t, cell.Style.Foreground.Type, ColorBasic)
	assert.Equal(t, cell.Style.Foreground.Value, uint8(1))
}

func TestResizeKeepsCellStyles(t *testing.T) {
	s := NewScreen(6, 3)
	s.Write([]byte("ab\x1b[32mcdefg"))
	s.Resize(3, 3)
	assert.Equal(t, visibleRows(s), []string{"abc", "def", "g"})
	assert.False(t, s.Cell(1, 0).Style.Foreground.Type == ColorBasic)
	assert.Equal(t, s.Cell(2, 0).Style.Foreground.Type, ColorBasic)
	assert.Equal(t, s.Cell(0, 2).Style.Foreground.Type, ColorBasic)
}

func TestResizeKeepsWideCharactersWhole(t *testing.T) {
	s := NewScreen(5, 3)
	s.Write([]byte("ab中文"))
	s.Resize(3, 3)
	assert.Equal(t, visibleRows(s), []string{"ab", "中", "文"})
	assert.Equal(t, s.Cell(0, 1).Width, 2)
	assert.True(t, s.Cell(1, 1).Continuation)
	assert.Equal(t, s.Cell(2, 0), blankCell())
}

func TestResizeTwiceRoundTrips(t *testing.T) {
	s := NewScreen(12, 4)
	s.Write([]byte("hello wonderful world\r\nnext"))
	s.Resize(5, 8)
	s.Resize(12, 4)
	assert.Equal(t, visibleRows(s), []string{"hello wonder", "ful world", "next"})
	x, y := s.Cursor()
	assert.Equal(t, x, 4)
	assert.Equal(t, y, 2)
}

func TestResizeAfterScrollingKeepsWraps(t *testing.T) {
	s := NewScreen(4, 2)
	for i := 0; i < 50; i++ {
		s.Write([]byte("abcdef\r\n"))
	}
	s.Write([]byte("abcdef"))
	s.Resize(8, 2)
	assert.Equal(t, visibleRows(s), []string{"abcdef"})
	// Rows that scrolled off do not keep their metadata alive.
	assert.True(t, len(s.rowInfo) <= 2*2+1, "rowInfo has %d entries", len(s.rowInfo))
}

func TestResizeAfterLongAltScreenSessionKeepsPrimaryWraps(t *testing.T) {
	s := NewScreen(5, 3)
	s.Write([]byte("abcdefgh\r\n$ "))
	s.Write([]byte("\x1b[?1049h"))
	for i := 0; i < 50; i++ {
		s.Write([]byte("0123456789\r\n")) // wraps and scrolls the alternate screen
	}
	s.Resize(10, 3)
	assert.True(t, s.IsAlternateScreen())
	s.Write([]byte("\x1b[?1049l"))
	assert.Equal(t, visibleRows(s), []string{"abcdefgh", "$"})
	AssertCursor(t, s, 2, 1)
}

func TestResizeAltScreenBlanksCutWideCharacter(t *testing.T) {
	s := NewScreen(6, 2)
	s.Write([]byte("\x1b[?1049h\x1b[Habcd\x1b[1;31m中"))
	s.Resize(5, 2)
	assert.Equal(t, s.Cell(4, 0), blankCell())
	assert.Equal(t, visibleRows(s), []string{"abcd"})
}

// ExampleScreen_Resize shows a soft-wrapped line reflowing when the window
// narrows. A test of a wonton app then tells the app the new size (for a
// terminal.Terminal, with SetSize) and writes the app's redraw.
func ExampleScreen_Resize() {
	screen := NewScreen(20, 4)
	screen.Write([]byte("The quick brown fox jumps\r\n$ "))

	screen.Resize(10, 4)
	fmt.Print(screen.Text())
	x, y := screen.Cursor()
	fmt.Printf("cursor at (%d, %d)\n", x, y)

	// Output:
	// The quick
	// brown fox
	// jumps
	// $
	// cursor at (2, 3)
}
