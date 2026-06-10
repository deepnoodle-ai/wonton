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
