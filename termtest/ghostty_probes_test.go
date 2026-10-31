package termtest

import (
	"fmt"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// bgSpan is a run of cells on one row, from x0 to x1 inclusive, that have
// the same non-default background color. The color is "p<n>" for palette
// index n, or "#rrggbb".
type bgSpan struct {
	y, x0, x1 int
	color     string
}

// ghosttyProbe is one byte stream and the screen that ghostty shows after
// it. The expected values were recorded from libghostty-vt, ghostty's
// terminal core, by a harness outside this repo that writes the same bytes
// into ghostty and termtest and compares every cell, the cursor, and the
// scrollback text. The results are written in here so that the checks run
// without ghostty or cgo. To add a probe, record ghostty's result for the
// new bytes; do not copy termtest's output.
type ghosttyProbe struct {
	name       string
	size       [2]int // width, height
	in         string
	rows       []string // Row text, trailing spaces trimmed
	cursor     [2]int
	scrollback []string
	bg         []bgSpan // every cell with a non-default background
}

// ghosttyProbes covers scroll regions, background color erase, C0 controls
// inside CSI sequences, and the cursor after IL and DL.
var ghosttyProbes = []ghosttyProbe{
	// region
	{
		name:       "region: LF in top-anchored region adds scrollback",
		size:       [2]int{10, 5},
		in:         "1\r\n2\r\n3\r\n4\r\n5\x1b[1;3r\x1b[3;1H\n\nX",
		rows:       []string{"3", "", "X", "4", "5"},
		cursor:     [2]int{1, 2},
		scrollback: []string{"1", "2"},
	},
	{
		name:       "region: SU in top-anchored region adds scrollback",
		size:       [2]int{10, 5},
		in:         "1\r\n2\r\n3\r\n4\r\n5\x1b[1;3r\x1b[2S",
		rows:       []string{"3", "", "", "4", "5"},
		cursor:     [2]int{0, 0},
		scrollback: []string{"1", "2"},
	},
	{
		name:       "region: SU full screen adds scrollback",
		size:       [2]int{10, 3},
		in:         "1\r\n2\r\n3\x1b[2S",
		rows:       []string{"3", "", ""},
		cursor:     [2]int{1, 2},
		scrollback: []string{"1", "2"},
	},
	{
		name:   "region: DL at top adds no scrollback",
		size:   [2]int{10, 3},
		in:     "1\r\n2\r\n3\x1b[1;1H\x1b[M",
		rows:   []string{"2", "3", ""},
		cursor: [2]int{0, 0},
	},
	{
		name:   "region: region not at top adds no scrollback",
		size:   [2]int{10, 5},
		in:     "1\r\n2\r\n3\r\n4\r\n5\x1b[2;4r\x1b[4;1H\n\nX",
		rows:   []string{"1", "4", "", "X", "5"},
		cursor: [2]int{1, 3},
	},
	{
		name:       "region: autowrap at region bottom scrolls region",
		size:       [2]int{5, 4},
		in:         "1\r\n2\r\n3\r\n4\x1b[1;3r\x1b[3;1Habcdefg",
		rows:       []string{"2", "abcde", "fg", "4"},
		cursor:     [2]int{2, 2},
		scrollback: []string{"1"},
	},
	{
		name:   "region: NEL at region bottom",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[2;3r\x1b[3;4H\x1bEX",
		rows:   []string{"1", "3", "X", "4"},
		cursor: [2]int{1, 2},
	},
	{
		name:   "region: LF below region at last row does not scroll",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[1;2r\x1b[4;1H\nX",
		rows:   []string{"1", "2", "3", "X"},
		cursor: [2]int{1, 3},
	},
	{
		name:   "region: LF below region moves down",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[1;2r\x1b[3;1H\nX",
		rows:   []string{"1", "2", "3", "X"},
		cursor: [2]int{1, 3},
	},
	{
		name:   "region: CUU stops at top margin",
		size:   [2]int{10, 5},
		in:     "\x1b[2;4r\x1b[3;1H\x1b[9AX",
		rows:   []string{"", "X", "", "", ""},
		cursor: [2]int{1, 1},
	},
	{
		name:   "region: CUD stops at bottom margin",
		size:   [2]int{10, 5},
		in:     "\x1b[2;4r\x1b[3;1H\x1b[9BX",
		rows:   []string{"", "", "", "X", ""},
		cursor: [2]int{1, 3},
	},
	{
		name:   "region: CUU above region not stopped",
		size:   [2]int{10, 5},
		in:     "\x1b[3;4r\x1b[2;1H\x1b[9AX",
		rows:   []string{"X", "", "", "", ""},
		cursor: [2]int{1, 0},
	},
	{
		name:   "region: CUD below region not stopped",
		size:   [2]int{10, 5},
		in:     "\x1b[2;3r\x1b[4;1H\x1b[9BX",
		rows:   []string{"", "", "", "", "X"},
		cursor: [2]int{1, 4},
	},
	{
		name:       "region: invalid DECSTBM ignored",
		size:       [2]int{10, 4},
		in:         "1\r\n2\r\n3\r\n4\x1b[3;2r\x1b[4;1H\nX",
		rows:       []string{"2", "3", "4", "X"},
		cursor:     [2]int{1, 3},
		scrollback: []string{"1"},
	},
	{
		name:   "region: DECSTBM top only",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[2r\x1b[4;1H\nX",
		rows:   []string{"1", "3", "4", "X"},
		cursor: [2]int{1, 3},
	},
	{
		name:       "region: DECSTBM reset",
		size:       [2]int{10, 4},
		in:         "1\r\n2\r\n3\r\n4\x1b[2;3r\x1b[r\x1b[4;1H\nX",
		rows:       []string{"2", "3", "4", "X"},
		cursor:     [2]int{1, 3},
		scrollback: []string{"1"},
	},
	{
		name:   "region: DECSTBM bottom beyond screen",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[2;99r\x1b[4;1H\nX",
		rows:   []string{"1", "3", "4", "X"},
		cursor: [2]int{1, 3},
	},
	{
		name:   "region: IL inside region stops at bottom",
		size:   [2]int{10, 5},
		in:     "1\r\n2\r\n3\r\n4\r\n5\x1b[2;4r\x1b[2;3H\x1b[LX",
		rows:   []string{"1", "X", "2", "3", "5"},
		cursor: [2]int{1, 1},
	},
	{
		name:   "region: DL inside region",
		size:   [2]int{10, 5},
		in:     "1\r\n2\r\n3\r\n4\r\n5\x1b[2;4r\x1b[2;3H\x1b[2MX",
		rows:   []string{"1", "X", "", "", "5"},
		cursor: [2]int{1, 1},
	},
	{
		name:   "region: DL outside region ignored",
		size:   [2]int{10, 5},
		in:     "1\r\n2\r\n3\r\n4\r\n5\x1b[2;3r\x1b[5;1H\x1b[M",
		rows:   []string{"1", "2", "3", "4", "5"},
		cursor: [2]int{0, 4},
	},
	{
		name:   "region: SD within region",
		size:   [2]int{10, 5},
		in:     "1\r\n2\r\n3\r\n4\r\n5\x1b[2;4r\x1b[1T",
		rows:   []string{"1", "", "2", "3", "5"},
		cursor: [2]int{0, 0},
	},
	{
		name:   "region: SU larger than region",
		size:   [2]int{10, 5},
		in:     "1\r\n2\r\n3\r\n4\r\n5\x1b[2;4r\x1b[9S",
		rows:   []string{"1", "", "", "", "5"},
		cursor: [2]int{0, 0},
	},
	{
		name:   "region: RI above region at row 0 no scroll",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[2;3r\x1b[1;1H\x1bMX",
		rows:   []string{"X", "2", "3", "4"},
		cursor: [2]int{1, 0},
	},
	{
		name:   "region: origin mode CUP relative and clamped",
		size:   [2]int{10, 5},
		in:     "\x1b[2;4r\x1b[?6h\x1b[2;3HX\x1b[9;9HY",
		rows:   []string{"", "", "  X", "        Y", ""},
		cursor: [2]int{9, 3},
	},
	{
		name:   "region: origin mode VPA",
		size:   [2]int{10, 5},
		in:     "\x1b[2;4r\x1b[?6h\x1b[2dX",
		rows:   []string{"", "", "X", "", ""},
		cursor: [2]int{1, 2},
	},
	{
		name:   "region: origin mode CUU stops at margin",
		size:   [2]int{10, 5},
		in:     "\x1b[2;4r\x1b[?6h\x1b[9AX",
		rows:   []string{"", "X", "", "", ""},
		cursor: [2]int{1, 1},
	},
	{
		name:   "region: origin mode reset homes",
		size:   [2]int{10, 5},
		in:     "\x1b[2;4r\x1b[?6h\x1b[2;2H\x1b[?6lX",
		rows:   []string{"X", "", "", "", ""},
		cursor: [2]int{1, 0},
	},
	{
		name:   "region: DECSTBM in origin mode homes to region top",
		size:   [2]int{10, 5},
		in:     "\x1b[?6h\x1b[2;4rX",
		rows:   []string{"", "X", "", "", ""},
		cursor: [2]int{1, 1},
	},
	{
		name:   "region: DECSC/DECRC saves origin mode",
		size:   [2]int{10, 5},
		in:     "\x1b[2;4r\x1b[?6h\x1b7\x1b[?6l\x1b8\x1b[1;1HX",
		rows:   []string{"", "X", "", "", ""},
		cursor: [2]int{1, 1},
	},
	{
		name:       "region: RIS clears region",
		size:       [2]int{10, 4},
		in:         "1\r\n2\r\n3\r\n4\x1b[2;3r\x1bc1\r\n2\r\n3\r\n4\r\n5",
		rows:       []string{"2", "3", "4", "5"},
		cursor:     [2]int{1, 3},
		scrollback: []string{"1"},
	},
	{
		name:   "region: RIS clears scrollback",
		size:   [2]int{10, 2},
		in:     "1\r\n2\r\n3\x1bcX",
		rows:   []string{"X", ""},
		cursor: [2]int{1, 0},
	},
	{
		name:   "region: ED 3 clears scrollback only",
		size:   [2]int{10, 2},
		in:     "1\r\n2\r\n3\x1b[3JX",
		rows:   []string{"2", "3X"},
		cursor: [2]int{2, 1},
	},
	{
		name:       "region: inline print through region",
		size:       [2]int{10, 6},
		in:         "a\r\nb\r\nc\r\nd\r\nL1\r\nL2\x1b7\x1b[1;4r\x1b[4;1H\x1b[0m\r\nnew1\x1b[0m\r\nnew2\x1b[r\x1b8",
		rows:       []string{"c", "d", "new1", "new2", "L1", "L2"},
		cursor:     [2]int{2, 5},
		scrollback: []string{"a", "b"},
	},
	{
		name:       "region: inline print keeps styled rows",
		size:       [2]int{10, 4},
		in:         "a\r\nb\r\nL1\x1b7\x1b[1;2r\x1b[2;1H\x1b[0m\r\n\x1b[31;44mred\x1b[0m\r\nok\x1b[r\x1b8",
		rows:       []string{"red", "ok", "L1", ""},
		cursor:     [2]int{2, 2},
		scrollback: []string{"a", "b"},
		bg:         []bgSpan{{0, 0, 2, "p4"}},
	},
	// bce
	{
		name:       "bce: SU fills bg",
		size:       [2]int{10, 3},
		in:         "1\r\n2\r\n3\x1b[44m\x1b[S",
		rows:       []string{"2", "3", ""},
		cursor:     [2]int{1, 2},
		scrollback: []string{"1"},
		bg:         []bgSpan{{2, 0, 9, "p4"}},
	},
	{
		name:   "bce: SD fills bg",
		size:   [2]int{10, 3},
		in:     "1\r\n2\r\n3\x1b[44m\x1b[T",
		rows:   []string{"", "1", "2"},
		cursor: [2]int{1, 2},
		bg:     []bgSpan{{0, 0, 9, "p4"}},
	},
	{
		name:   "bce: DL fills bg",
		size:   [2]int{10, 3},
		in:     "1\r\n2\r\n3\x1b[1;1H\x1b[45m\x1b[M",
		rows:   []string{"2", "3", ""},
		cursor: [2]int{0, 0},
		bg:     []bgSpan{{2, 0, 9, "p5"}},
	},
	{
		name:   "bce: ICH fills bg",
		size:   [2]int{10, 2},
		in:     "abcdef\x1b[1;2H\x1b[46m\x1b[2@",
		rows:   []string{"a  bcdef", ""},
		cursor: [2]int{1, 0},
		bg:     []bgSpan{{0, 1, 2, "p6"}},
	},
	{
		name:   "bce: RI scroll fills bg",
		size:   [2]int{10, 3},
		in:     "1\r\n2\r\n3\x1b[1;1H\x1b[41m\x1bM",
		rows:   []string{"", "1", "2"},
		cursor: [2]int{0, 0},
		bg:     []bgSpan{{0, 0, 9, "p1"}},
	},
	{
		name:   "bce: EL 1 fills bg",
		size:   [2]int{10, 2},
		in:     "abcdefghij\x1b[1;5H\x1b[42m\x1b[1K",
		rows:   []string{"     fghij", ""},
		cursor: [2]int{4, 0},
		bg:     []bgSpan{{0, 0, 4, "p2"}},
	},
	{
		name:   "bce: ED 1 fills bg",
		size:   [2]int{10, 2},
		in:     "abcdefghij\r\nabcdefghij\x1b[2;5H\x1b[43m\x1b[1J",
		rows:   []string{"", "     fghij"},
		cursor: [2]int{4, 1},
		bg:     []bgSpan{{0, 0, 9, "p3"}, {1, 0, 4, "p3"}},
	},
	{
		name:   "bce: BCE ignores bold and underline",
		size:   [2]int{10, 2},
		in:     "\x1b[1;4;44mab\x1b[K",
		rows:   []string{"ab", ""},
		cursor: [2]int{2, 0},
		bg:     []bgSpan{{0, 0, 9, "p4"}},
	},
	{
		name:   "bce: BCE with 256 and RGB bg",
		size:   [2]int{10, 2},
		in:     "\x1b[48;5;200ma\x1b[K\r\n\x1b[48;2;1;2;3mb\x1b[K",
		rows:   []string{"a", "b"},
		cursor: [2]int{1, 1},
		bg:     []bgSpan{{0, 0, 9, "p200"}, {1, 0, 9, "#010203"}},
	},
	{
		name:   "bce: region scroll fills bg",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[2;3r\x1b[3;1H\x1b[44m\n",
		rows:   []string{"1", "3", "", "4"},
		cursor: [2]int{0, 2},
		bg:     []bgSpan{{2, 0, 9, "p4"}},
	},
	{
		name:   "bce: DCH after BCE erase keeps bg",
		size:   [2]int{10, 2},
		in:     "\x1b[44mab\x1b[K\x1b[0m\x1b[1;1H\x1b[P",
		rows:   []string{"b", ""},
		cursor: [2]int{0, 0},
		bg:     []bgSpan{{0, 0, 8, "p4"}},
	},
	// c0csi
	{
		name:   "c0csi: LF inside CSI (column 0, so ONLCR does not matter)",
		size:   [2]int{10, 3},
		in:     "abc\r\x1b[2\nCX",
		rows:   []string{"abc", "  X", ""},
		cursor: [2]int{3, 1},
	},
	{
		name:   "c0csi: BS inside CSI",
		size:   [2]int{10, 2},
		in:     "abcd\x1b[\b2CX",
		rows:   []string{"abcd X", ""},
		cursor: [2]int{6, 0},
	},
	{
		name:   "c0csi: HT inside CSI",
		size:   [2]int{20, 2},
		in:     "a\x1b[\t1CX",
		rows:   []string{"a        X", ""},
		cursor: [2]int{10, 0},
	},
	{
		name:   "c0csi: VT inside CSI",
		size:   [2]int{10, 3},
		in:     "abc\x1b[\v1CX",
		rows:   []string{"abc", "    X", ""},
		cursor: [2]int{5, 1},
	},
	{
		name:   "c0csi: BEL inside CSI ignored",
		size:   [2]int{10, 2},
		in:     "abc\x1b[2\aCX",
		rows:   []string{"abc  X", ""},
		cursor: [2]int{6, 0},
	},
	{
		name:   "c0csi: CR inside SGR",
		size:   [2]int{10, 2},
		in:     "abc\x1b[3\r1mX",
		rows:   []string{"Xbc", ""},
		cursor: [2]int{1, 0},
	},
	{
		name:   "c0csi: two C0 inside CSI",
		size:   [2]int{10, 3},
		in:     "abc\x1b[\r\n2CX",
		rows:   []string{"abc", "  X", ""},
		cursor: [2]int{3, 1},
	},
	// ildl
	{
		name:   "ildl: IL moves cursor to column 0",
		size:   [2]int{10, 3},
		in:     "1\r\n2\r\n3\x1b[2;5H\x1b[LX",
		rows:   []string{"1", "X", "2"},
		cursor: [2]int{1, 1},
	},
	{
		name:   "ildl: DL moves cursor to column 0",
		size:   [2]int{10, 3},
		in:     "1\r\n2\r\n3\x1b[1;5H\x1b[MX",
		rows:   []string{"X", "3", ""},
		cursor: [2]int{1, 0},
	},
	{
		name:   "ildl: IL with count moves cursor to column 0",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[2;5H\x1b[2LX",
		rows:   []string{"1", "X", "", "2"},
		cursor: [2]int{1, 1},
	},
	{
		name:   "ildl: IL clears pending wrap",
		size:   [2]int{10, 3},
		in:     "abcdefghij\x1b[LX",
		rows:   []string{"X", "abcdefghij", ""},
		cursor: [2]int{1, 0},
	},
	{
		name:   "ildl: DL clears pending wrap",
		size:   [2]int{10, 3},
		in:     "abcdefghij\r\n2\x1b[1;10HZ\x1b[MX",
		rows:   []string{"X", "", ""},
		cursor: [2]int{1, 0},
	},
	{
		name:   "ildl: IL outside region keeps column",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[2;3r\x1b[4;5H\x1b[LX",
		rows:   []string{"1", "2", "3", "4   X"},
		cursor: [2]int{5, 3},
	},
	{
		name:   "ildl: DL outside region keeps column",
		size:   [2]int{10, 4},
		in:     "1\r\n2\r\n3\r\n4\x1b[2;3r\x1b[4;5H\x1b[MX",
		rows:   []string{"1", "2", "3", "4   X"},
		cursor: [2]int{5, 3},
	},
	{
		name:   "ildl: IL outside region keeps pending wrap",
		size:   [2]int{10, 4},
		in:     "\x1b[1;2r\x1b[4;1Habcdefghij\x1b[LX",
		rows:   []string{"", "", "", "Xbcdefghij"},
		cursor: [2]int{1, 3},
	},
	{
		name:   "ildl: DL outside region keeps pending wrap",
		size:   [2]int{10, 4},
		in:     "\x1b[1;2r\x1b[4;1Habcdefghij\x1b[MX",
		rows:   []string{"", "", "", "Xbcdefghij"},
		cursor: [2]int{1, 3},
	},
}

func TestGhosttyProbes(t *testing.T) {
	for _, p := range ghosttyProbes {
		t.Run(p.name, func(t *testing.T) {
			s := NewScreen(p.size[0], p.size[1])
			s.Write([]byte(p.in))
			assert.Equal(t, rows(s), p.rows, "rows")
			x, y := s.Cursor()
			assert.Equal(t, [2]int{x, y}, p.cursor, "cursor")
			assert.Equal(t, s.Scrollback(), p.scrollback, "scrollback")
			assert.Equal(t, bgSpans(s), p.bg, "background")
		})
	}
}

// bgSpans returns the runs of cells with a non-default background color.
func bgSpans(s *Screen) []bgSpan {
	var out []bgSpan
	w, h := s.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; {
			c := colorName(s.Cell(x, y).Style.Background)
			if c == "" {
				x++
				continue
			}
			x0 := x
			for x < w && colorName(s.Cell(x, y).Style.Background) == c {
				x++
			}
			out = append(out, bgSpan{y, x0, x - 1, c})
		}
	}
	return out
}

// colorName names a color the way the ghostty results do. It returns ""
// for the default color.
func colorName(c Color) string {
	switch c.Type {
	case ColorBasic, Color256:
		return fmt.Sprintf("p%d", c.Value)
	case ColorRGB:
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return ""
}
