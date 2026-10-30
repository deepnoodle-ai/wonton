package termtest

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// --- SGR colon sub-parameter handling ---

func TestSGRColon256ColorFollowedByParam(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[38:5:196;1mX"))

	cell := s.Cell(0, 0)
	assert.Equal(t, Color256, cell.Style.Foreground.Type)
	assert.Equal(t, uint8(196), cell.Style.Foreground.Value)
	assert.True(t, cell.Style.Bold)
	assert.False(t, cell.Style.Blink) // 5 is a sub-parameter, not SGR blink
}

func TestSGRColonRGBWithColorspace(t *testing.T) {
	// ITU T.416 form with the (empty) colorspace ID slot: 38:2::r:g:b
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[38:2::128:64:32mX"))

	cell := s.Cell(0, 0)
	assert.Equal(t, ColorRGB, cell.Style.Foreground.Type)
	assert.Equal(t, uint8(128), cell.Style.Foreground.R)
	assert.Equal(t, uint8(64), cell.Style.Foreground.G)
	assert.Equal(t, uint8(32), cell.Style.Foreground.B)
}

func TestSGRColonUnderlineOff(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1;4m\x1b[4:0mX"))

	cell := s.Cell(0, 0)
	assert.False(t, cell.Style.Underline) // 4:0 disables underline
	assert.True(t, cell.Style.Bold)       // and must not reset other attributes
}

func TestSGRColonCurlyUnderline(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[4:3mX"))
	assert.True(t, s.Cell(0, 0).Style.Underline)
}

func TestSGRPrivateSequenceIgnored(t *testing.T) {
	// xterm modifyOtherKeys (CSI > 4;2 m) must not be misread as SGR reset
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1m\x1b[>4;2mX"))
	assert.True(t, s.Cell(0, 0).Style.Bold)
}

// --- Control characters ---

func TestBackspaceMovesCursor(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("ab\b\bcd"))
	assert.Equal(t, "cd", s.Row(0))
}

func TestBackspaceStopsAtMargin(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("a\b\b\bX"))
	assert.Equal(t, "X", s.Row(0))

	x, y := s.Cursor()
	assert.Equal(t, 1, x)
	assert.Equal(t, 0, y)
}

func TestBellIsIgnored(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("ab\a"))
	assert.Equal(t, "ab", s.Row(0))
	assert.Equal(t, "ab\n\n\n\n\n", s.Text())
}

func TestVerticalTabAndFormFeedMoveDown(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("A\vB\fC"))
	assert.Equal(t, "A", s.Row(0))
	assert.Equal(t, " B", s.Row(1))
	assert.Equal(t, "  C", s.Row(2))
}

// --- Erase in Display cursor behavior ---

func TestEraseDisplayKeepsCursor(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("hello\x1b[2;3H\x1b[2J"))

	x, y := s.Cursor()
	assert.Equal(t, 2, x) // ED must not move the cursor
	assert.Equal(t, 1, y)

	s.Write([]byte("X"))
	assert.Equal(t, "  X", s.Row(1))
}

// --- Wide glyph halves on erase ---

func TestClearToEndOfLineOnContinuationCell(t *testing.T) {
	s := NewScreen(10, 2)
	s.Write([]byte("日AB"))
	s.SetCursor(1, 0) // continuation half of 日
	s.Write([]byte("\x1b[K"))

	assert.Equal(t, "", s.Row(0)) // the lead half must be cleared too
}

func TestClearToStartOfLineOnLeadCell(t *testing.T) {
	s := NewScreen(10, 2)
	s.Write([]byte("日AB"))
	s.SetCursor(0, 0) // lead half of 日
	s.Write([]byte("\x1b[1K"))

	// The orphaned continuation cell must be cleared so columns line up:
	// A is at screen column 2, so the row text is "  AB".
	assert.Equal(t, "  AB", s.Row(0))
	assert.False(t, s.Cell(1, 0).Continuation)
}

// --- Deferred wrap (pending wrap state) ---

func TestDeferredWrapCursorStaysOnLastColumn(t *testing.T) {
	s := NewScreen(10, 5)
	s.Write([]byte("0123456789"))

	x, y := s.Cursor()
	assert.Equal(t, 9, x) // cursor reports the last column, not width
	assert.Equal(t, 0, y)

	// The next printable wraps to the following line
	s.Write([]byte("X"))
	assert.Equal(t, "0123456789", s.Row(0))
	assert.Equal(t, "X", s.Row(1))
}

func TestDeferredWrapCursorBack(t *testing.T) {
	s := NewScreen(10, 5)
	s.Write([]byte("0123456789\x1b[1DX"))
	// CUB from the (pending-wrap) last column moves to column 8
	assert.Equal(t, "01234567X9", s.Row(0))
}

func TestDeferredWrapCanceledByCarriageReturn(t *testing.T) {
	s := NewScreen(10, 5)
	s.Write([]byte("0123456789\rX"))
	assert.Equal(t, "X123456789", s.Row(0))
}

// --- Split writes ---

func TestSplitUTF8AcrossWrites(t *testing.T) {
	s := NewScreen(20, 5)
	data := []byte("日本")
	s.Write(data[:4]) // splits the second rune
	s.Write(data[4:])
	assert.Equal(t, "日本", s.Row(0))
}

func TestSplitCSIAcrossWrites(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("hi\x1b[1"))
	s.Write([]byte("mB"))
	assert.Equal(t, "hiB", s.Row(0))
	assert.True(t, s.Cell(2, 0).Style.Bold)
}

func TestSplitEscAcrossWrites(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("a\x1b"))
	s.Write([]byte("[2Cb"))
	assert.Equal(t, "a  b", s.Row(0))
}

func TestSplitOSCAcrossWrites(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b]0;tit"))
	s.Write([]byte("le\x07ok"))
	assert.Equal(t, "ok", s.Row(0))
}

// --- NEL, DCS, ESC 7/8 ---

func TestNextLineScrollsAtBottom(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("A\nB\nC"))
	s.Write([]byte("\x1bED"))

	assert.Equal(t, "B", s.Row(0))
	assert.Equal(t, "C", s.Row(1))
	assert.Equal(t, "D", s.Row(2))
}

func TestDCSPayloadNotRendered(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1bP1$r0m\x1b\\OK"))
	assert.Equal(t, "OK", s.Row(0))
}

func TestAPCPayloadNotRendered(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b_Gpayload\x1b\\OK"))
	assert.Equal(t, "OK", s.Row(0))
}

func TestSaveRestoreCursorIncludesStyle(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1m\x1b7\x1b[0m\x1b8X"))
	assert.True(t, s.Cell(0, 0).Style.Bold) // DECRC restores attributes
}

// --- Recorder reset ---

func TestRecorderResetClearsStyle(t *testing.T) {
	r := NewRecorder(20, 5)
	r.Write([]byte("\x1b[1m"))
	r.Reset()
	r.Write([]byte("X"))
	assert.False(t, r.Screen().Cell(0, 0).Style.Bold)
}

// --- Diff ---

func TestDiffTrailingNewline(t *testing.T) {
	d := Diff("a\n", "a")
	if d == "" {
		t.Fatal("Diff returned empty for strings differing only in trailing newline")
	}
	assert.Contains(t, d, "No newline at end of file")
}

func TestDiffShowsRealContextLines(t *testing.T) {
	expected := "one\ntwo\nthree\nfour\n"
	actual := "one\ntwo\nCHANGED\nfour\n"
	d := Diff(expected, actual)

	assert.Contains(t, d, "- three")
	assert.Contains(t, d, "+ CHANGED")
	assert.Contains(t, d, "  two") // real context content, not a placeholder
	if strings.Contains(d, "(context)") {
		t.Errorf("diff still contains placeholder context lines:\n%s", d)
	}
}

func TestDiffInsertedLineDoesNotCascade(t *testing.T) {
	expected := "a\nb\nc\n"
	actual := "new\na\nb\nc\n"
	d := Diff(expected, actual)

	assert.Contains(t, d, "+ new")
	if strings.Contains(d, "- a") {
		t.Errorf("inserted line cascaded into spurious changes:\n%s", d)
	}
}

func TestDiffEqualStrings(t *testing.T) {
	assert.Equal(t, "", Diff("same\n", "same\n"))
}

// --- Grapheme clusters split across writes & zero-width handling ---

func TestClusterSplitAcrossWrites(t *testing.T) {
	cases := []struct {
		name       string
		writes     []string
		wantRow    string
		wantCursor int
		wantGlyph0 string
		wantWidth0 int
	}{
		{"combining mark", []string{"e", "\u0301x"}, "e\u0301x", 2, "e\u0301", 1},
		{"VS16 widens", []string{"\u2764", "\uFE0Fx"}, "\u2764\uFE0Fx", 3, "\u2764\uFE0F", 2},
		{"ZWJ sequence", []string{"\U0001F468\u200D", "\U0001F469x"}, "\U0001F468\u200D\U0001F469x", 3, "\U0001F468\u200D\U0001F469", 2},
		{"flag", []string{"\U0001F1EF", "\U0001F1F5x"}, "\U0001F1EF\U0001F1F5x", 3, "\U0001F1EF\U0001F1F5", 2},
		{"around SGR", []string{"e\x1b[31m\u0301x"}, "e\u0301x", 2, "e\u0301", 1},
		{"after wide", []string{"世", "\u0301x"}, "世\u0301x", 3, "世\u0301", 2},
		{"after space", []string{" ", "\u0301x"}, " \u0301x", 2, " \u0301", 1},
		// A blank cell reached by a cursor move joins too, as in gif.TerminalScreen.
		{"after cursor move", []string{"\x1b[C", "\u0301x"}, " \u0301x", 2, " \u0301", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewScreen(20, 3)
			for _, w := range tc.writes {
				s.Write([]byte(w))
			}
			assert.Equal(t, tc.wantRow, s.Row(0))
			x, _ := s.Cursor()
			assert.Equal(t, tc.wantCursor, x)
			assert.Equal(t, tc.wantGlyph0, s.CellGlyph(0, 0))
			assert.Equal(t, tc.wantWidth0, s.Cell(0, 0).Width)
		})
	}
}

func TestZeroWidthClusterDoesNotClobberCell(t *testing.T) {
	s := NewScreen(10, 2)
	s.Write([]byte("AB\rA\u200B"))
	assert.Equal(t, "AB", s.Row(0))
	assert.Equal(t, 'B', s.Cell(1, 0).Char)
}

func TestWideClusterOnOneColumnScreen(t *testing.T) {
	s := NewScreen(1, 2)
	s.Write([]byte("世A"))
	assert.Equal(t, "A", s.Row(0))
	assert.Equal(t, "", s.Row(1))
}

// --- CSI private markers, intermediates, DECSED/DECSEL, ESC intermediates ---

func TestCSIPrivateLessThanPrefixIgnored(t *testing.T) {
	s := NewScreen(20, 3)
	s.Write([]byte("\x1b[1mHello\x1b[<u\x1b[<0;10;5M\x1b[<0;10;5m!"))
	assert.Equal(t, "Hello!", s.Row(0))
	assert.True(t, s.Cell(5, 0).Style.Bold)
	assert.False(t, s.Cell(5, 0).Style.Blink)
}

func TestCSIIntermediatesIgnoredAndKeepDeferredWrap(t *testing.T) {
	s := NewScreen(5, 3)
	// Fill row 0 so wrapPending is true, then send DECSCUSR (CSI 2 SP q) and
	// ECMA-48 SR (CSI 1 SP A); neither may move the cursor or cancel wrapPending.
	s.Write([]byte("12345\x1b[2 q\x1b[1 AX"))
	assert.Equal(t, "12345", s.Row(0))
	assert.Equal(t, "X", s.Row(1))
}

func TestDECSelectiveErase(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("abc\r\ndef\x1b[1;1H\x1b[?2J"))
	assert.Equal(t, "", s.Row(0))
	assert.Equal(t, "", s.Row(1))

	s.Write([]byte("hello\x1b[1;3H\x1b[?0K"))
	assert.Equal(t, "he", s.Row(0))
}

func TestESCIntermediateSequencesConsumed(t *testing.T) {
	s := NewScreen(20, 3)
	// DECALN (ESC # 8), S7C1T (ESC SP F), UTF-8 charset (ESC % G)
	s.Write([]byte("\x1b#8\x1b F\x1b"))
	s.Write([]byte("%GHello"))
	assert.Equal(t, "Hello", s.Row(0))
}

func TestOversizedStringSequenceSkippedAcrossWrites(t *testing.T) {
	s := NewScreen(20, 3)
	chunk := []byte(strings.Repeat("A", 64*1024))
	s.Write([]byte("\x1bPq"))
	for i := 0; i < 24; i++ { // 1.5 MiB, more than maxPendingBytes
		s.Write(chunk)
	}
	s.Write([]byte("\x1b"))
	s.Write([]byte("\\o"))

	s.Write([]byte("\x1b]52;c;"))
	for i := 0; i < 24; i++ {
		s.Write(chunk)
	}
	s.Write([]byte("\x07k"))

	assert.Equal(t, "ok", s.Row(0))
	assert.Len(t, s.pending, 0)
}

// TestSplitAtEveryByte checks that splitting a stream at any byte gives the
// same screen as writing it whole. The stream covers OSC with BEL and ST, BEL
// inside a DCS payload, CAN and SUB aborts, an OSC ended by the ESC of a
// following CSI, intermediate escapes, multi-byte UTF-8, and a combining mark.
func TestSplitAtEveryByte(t *testing.T) {
	const stream = "a\x1b]0;title\x07b\x1b]2;t\x1b\\c\x1bP1$r\x07x\x1b\\d\x1b_G\x18e" +
		"\x1b]0;t\x1b[31mf\x1b[12\x1ag\x1b(Bh\x1b#8i\u00e9jk\u0301"

	whole := NewScreen(20, 3)
	whole.Write([]byte(stream))
	assert.Equal(t, whole.Row(0), "abcdefghi\u00e9jk\u0301")
	assert.Equal(t, whole.Cell(5, 0).Style.Foreground, Color{Type: ColorBasic, Value: 1})

	for k := 1; k < len(stream); k++ {
		s := NewScreen(20, 3)
		s.Write([]byte(stream[:k]))
		s.Write([]byte(stream[k:]))
		assert.Equal(t, s.cells, whole.cells, "split at byte %d", k)
	}

	s := NewScreen(20, 3)
	for i := 0; i < len(stream); i++ {
		s.Write([]byte{stream[i]})
	}
	assert.Equal(t, s.cells, whole.cells, "byte at a time")
}

func TestKittyKeyboardPopSequenceIgnored(t *testing.T) {
	// Kitty keyboard pop sequence (\x1b[<u) must not be interpreted as CSI u (restoreCursor).
	s := NewScreen(20, 5)
	s.Write([]byte("Hello\x1b[<u!"))
	assert.Equal(t, "Hello!", s.Row(0))
}

func TestZeroWidthClusterDoesNotOverwriteCell(t *testing.T) {
	s := NewScreen(5, 2)
	s.Write([]byte("AB\u200BC"))
	assert.Equal(t, "ABC", s.Row(0))

	// Zero-width cluster at the right margin must not trigger deferred wrap or overwrite the last cell.
	s2 := NewScreen(3, 2)
	s2.Write([]byte("XYZ\u200B"))
	assert.Equal(t, "XYZ", s2.Row(0))
	assert.Equal(t, "", s2.Row(1))
}

func TestNewScreenClampsNonPositiveDimensions(t *testing.T) {
	s := NewScreen(0, -1)
	w, h := s.Size()
	assert.Equal(t, 1, w)
	assert.Equal(t, 1, h)
	s.SetCursor(5, 5)
	s.Write([]byte("A"))
	assert.Equal(t, "A", s.Row(0))
}
