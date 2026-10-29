package gif

import (
	"fmt"
	"slices"
	"testing"
)

// cont marks a continuation cell in an expected row.
const cont = "<cont>"

// graphemeCases are clusters whose cell layout matches a terminal's: each
// takes one cell, or a wide cell plus a continuation.
var graphemeCases = []struct {
	name       string
	input      string
	wantCursor int
	wantRow    []string // the first wantCursor cells
}{
	{"CJK then ASCII", "世A", 3, []string{"世", cont, "A"}},
	{"combining acute", "e\u0301x", 2, []string{"e\u0301", "x"}},
	{"heart VS16", "\u2764\uFE0Fx", 3, []string{"\u2764\uFE0F", cont, "x"}},
	{"hash keycap", "#\uFE0F\u20E3x", 3, []string{"#\uFE0F\u20E3", cont, "x"}},
	{"JP flag", "\U0001F1EF\U0001F1F5x", 3, []string{"\U0001F1EF\U0001F1F5", cont, "x"}},
	{
		"family of four",
		"\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466x",
		3,
		[]string{"\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466", cont, "x"},
	},
	{"stray combining mark", "\u0301A", 1, []string{"A"}},
}

func TestTerminalScreen_Graphemes(t *testing.T) {
	for _, tc := range graphemeCases {
		t.Run(tc.name, func(t *testing.T) {
			screen := NewTerminalScreen(20, 3)
			screen.WriteString(tc.input, White, Black)
			checkRow(t, screen, 0, tc.wantCursor, tc.wantRow)
		})
	}
}

func TestEmulator_Graphemes(t *testing.T) {
	for _, tc := range graphemeCases {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEmulator(20, 3)
			em.ProcessOutput(tc.input)
			checkRow(t, em.Screen(), 0, tc.wantCursor, tc.wantRow)
		})
	}
}

// TestEmulator_ClusterSplitAcrossWrites covers a cluster that arrives in
// pieces, across writes or around an escape sequence. The later runes join
// the cell before the cursor, as in a terminal.
func TestEmulator_ClusterSplitAcrossWrites(t *testing.T) {
	cases := []struct {
		name       string
		writes     []string
		wantCursor int
		wantRow    []string
	}{
		{"combining mark", []string{"e", "\u0301x"}, 2, []string{"e\u0301", "x"}},
		{"VS16 widens", []string{"\u2764", "\uFE0Fx"}, 3, []string{"\u2764\uFE0F", cont, "x"}},
		{"ZWJ sequence", []string{"\U0001F468\u200D", "\U0001F469x"}, 3, []string{"\U0001F468\u200D\U0001F469", cont, "x"}},
		{"flag", []string{"\U0001F1EF", "\U0001F1F5x"}, 3, []string{"\U0001F1EF\U0001F1F5", cont, "x"}},
		{"around SGR", []string{"e\x1b[31m\u0301x"}, 2, []string{"e\u0301", "x"}},
		{"after wide", []string{"世", "\u0301x"}, 3, []string{"世\u0301", cont, "x"}},
		{"no join", []string{"世", "界"}, 4, []string{"世", cont, "界", cont}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEmulator(20, 3)
			for _, w := range tc.writes {
				em.ProcessOutput(w)
			}
			checkRow(t, em.Screen(), 0, tc.wantCursor, tc.wantRow)
		})
	}
}

// TestEmulator_JoinAtRightEdgeStaysInPlace checks that a symbol in the last
// column keeps its place when a later VS16 would make it two columns wide.
// It joins the VS16 but stays one column wide instead of wrapping.
func TestEmulator_JoinAtRightEdgeStaysInPlace(t *testing.T) {
	em := NewEmulator(5, 2)
	em.ProcessOutput("abcd❤")
	em.ProcessOutput("️")
	screen := em.Screen()
	checkRow(t, screen, 0, 5, []string{"a", "b", "c", "d", "❤️"})
	checkRow(t, screen, 1, -1, []string{" ", " ", " ", " ", " "})
	if screen.CursorY != 0 {
		t.Errorf("cursorY = %d, want 0", screen.CursorY)
	}
}

func TestEmulator_JoinedClusterKeepsBaseColors(t *testing.T) {
	em := NewEmulator(20, 3)
	em.ProcessOutput("e\x1b[31m\u0301")
	if got := em.Screen().Cells[0][0].FG; got != White {
		t.Errorf("FG = %v, want the base's %v", got, White)
	}
}

func TestEmulator_WideClusterWrapsAtRightEdge(t *testing.T) {
	em := NewEmulator(5, 3)
	em.ProcessOutput("abcd世")
	screen := em.Screen()
	checkRow(t, screen, 0, -1, []string{"a", "b", "c", "d", " "})
	checkRow(t, screen, 1, 2, []string{"世", cont})
	if screen.CursorY != 1 {
		t.Errorf("cursorY = %d, want 1", screen.CursorY)
	}
}

func TestEmulator_WideClusterFillsLastColumns(t *testing.T) {
	em := NewEmulator(4, 3)
	em.ProcessOutput("ab世")
	checkRow(t, em.Screen(), 0, 4, []string{"a", "b", "世", cont})
}

func TestEmulator_WideClusterOnOneColumnScreen(t *testing.T) {
	em := NewEmulator(1, 2)
	em.ProcessOutput("世A")
	checkRow(t, em.Screen(), 0, 1, []string{"A"})
}

// TestEmulator_OverwriteWideHalf checks that writing over either half of a
// wide cluster blanks the other half instead of leaving half a glyph.
func TestEmulator_OverwriteWideHalf(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantRow []string
	}{
		{"narrow over lead", "世\rA", []string{"A", " ", " ", " "}},
		{"narrow over continuation", "世\x1b[2GA", []string{" ", "A", " ", " "}},
		{"wide over continuation", "世世\x1b[2G界", []string{" ", "界", cont, " "}},
		{"erase from continuation", "世世\x1b[2G\x1b[K", []string{" ", " ", " ", " "}},
		{"erase to continuation", "世世\x1b[2G\x1b[1K", []string{" ", " ", "世", cont}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEmulator(4, 2)
			em.ProcessOutput(tc.input)
			checkRow(t, em.Screen(), 0, -1, tc.wantRow)
		})
	}
}

func TestEmulator_ResizeCutsWideCluster(t *testing.T) {
	em := NewEmulator(4, 2)
	em.ProcessOutput("a世")
	em.Resize(2, 2)
	checkRow(t, em.Screen(), 0, -1, []string{"a", " "})
}

func TestEmulator_InvalidUTF8TakesOneCell(t *testing.T) {
	em := NewEmulator(10, 2)
	em.ProcessOutput("a\xffb")
	screen := em.Screen()
	if screen.CursorX != 3 {
		t.Fatalf("cursorX = %d, want 3", screen.CursorX)
	}
	if got := screen.Cells[0][2].Char; got != 'b' {
		t.Errorf("cell 2 = %q, want 'b'", got)
	}
}

// ExampleEmulator_wideCharacters shows a wide character taking two cells,
// and an accent written separately joining its base letter.
func ExampleEmulator_wideCharacters() {
	em := NewEmulator(10, 1)
	em.ProcessOutput("世A")
	em.ProcessOutput(" e")
	em.ProcessOutput("́") // combining acute accent, in a later write

	screen := em.Screen()
	fmt.Println("cursor:", screen.CursorX)
	for x, cell := range screen.Cells[0][:screen.CursorX] {
		if cell.Continuation {
			fmt.Printf("%d: (continuation)\n", x)
			continue
		}
		fmt.Printf("%d: %+q width %d\n", x, cell.Text(), cell.Width)
	}
	// Output:
	// cursor: 5
	// 0: "\u4e16" width 2
	// 1: (continuation)
	// 2: "A" width 1
	// 3: " " width 1
	// 4: "e\u0301" width 1
}

// TestTerminalRenderer_CombiningMarkStaysInCell checks that a combining mark
// is drawn over its base, not in the cell after it.
func TestTerminalRenderer_CombiningMarkStaysInCell(t *testing.T) {
	render := func(s string) (base, next int) {
		screen := NewTerminalScreen(2, 1)
		screen.WriteString(s, White, Black)
		tr := NewTerminalRendererWithOptions(screen, RendererOptions{FontSize: 14})
		tr.RenderFrame(10)
		img := tr.GIF().images[0]
		for y := 0; y < tr.cellHeight; y++ {
			for x := 0; x < 2*tr.cellWidth; x++ {
				if r, _, _, _ := img.At(x, y).RGBA(); r > 0x4000 {
					if x < tr.cellWidth {
						base++
					} else {
						next++
					}
				}
			}
		}
		return base, next
	}
	plainBase, _ := render("e")
	base, next := render("é")
	if base <= plainBase {
		t.Errorf("base cell lit pixels = %d, want more than plain e's %d", base, plainBase)
	}
	if next != 0 {
		t.Errorf("next cell lit pixels = %d, want 0", next)
	}
}

// checkRow compares the first n cells of a row, or the whole row when n is
// -1, against want, and the cursor column against n when n >= 0. It also
// checks that each cell's Width agrees with the continuation after it.
func checkRow(t *testing.T, screen *TerminalScreen, row, n int, want []string) {
	t.Helper()
	if n >= 0 && screen.CursorX != n {
		t.Errorf("cursorX = %d, want %d", screen.CursorX, n)
	}
	if n < 0 {
		n = screen.Width
	}
	cells := screen.Cells[row][:n]
	got := make([]string, len(cells))
	for x, c := range cells {
		if c.Continuation {
			got[x] = cont
			continue
		}
		got[x] = c.Text()
		wantWidth := 1
		if x+1 < len(screen.Cells[row]) && screen.Cells[row][x+1].Continuation {
			wantWidth = 2
		}
		if c.Width != wantWidth {
			t.Errorf("row %d cell %d (%+q): Width = %d, want %d", row, x, got[x], c.Width, wantWidth)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("row %d:\n got: %+q\nwant: %+q", row, got, want)
	}
}
