package tui

import (
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Regression: a grapheme whose display width is wider than the available line
// width must not produce a blank leading line. The wrap check skips the wrap
// when the cursor is already at the line's left edge; the oversized cluster
// draws (and gets clipped) on the current line instead.
func TestTextInput_OversizedClusterDoesNotBlankLeadingLine(t *testing.T) {
	t.Run("countVisualLines width=1 cjk char", func(t *testing.T) {
		ti := newTextInput()
		s := "中"
		ti.SetValue(s)
		assert.Equal(t, 1, ti.countVisualLines(1))
	})

	t.Run("countVisualLines width=2 three-em dash", func(t *testing.T) {
		ti := newTextInput()
		s := "\u2E3B" // THREE-EM DASH, width 4
		ti.SetValue(s)
		// Width 2, cluster width 4: the cluster is too wide for the line but
		// it must count as exactly 1 visual line (drawn and clipped on the
		// current line), not 2 (blank line + cluster line).
		assert.Equal(t, 1, ti.countVisualLines(2))
	})

	t.Run("countVisualLines ascii then oversized cjk", func(t *testing.T) {
		ti := newTextInput()
		s := "ab中"
		ti.SetValue(s)
		// width=2: "ab" fills line 1 (x=2), CJK (width 2) wraps to line 2,
		// total 2 visual lines.
		assert.Equal(t, 2, ti.countVisualLines(2))
	})

	t.Run("calcWrappedHeight oversized single cluster", func(t *testing.T) {
		// width=1 with a CJK char: before the fix this returned 2 (blank line
		// + cluster line); after the fix it returns 1.
		assert.Equal(t, 1, calcWrappedHeight("中", 1))
		assert.Equal(t, 1, calcWrappedHeight("\u2E3B", 2))
	})

	t.Run("calcWrappedHeight ascii run then oversized", func(t *testing.T) {
		// width=2, "ab" + CJK: line 1 holds "ab", line 2 holds CJK.
		assert.Equal(t, 2, calcWrappedHeight("ab中", 2))
	})

	t.Run("multi-rune grapheme cluster counts as one cell group", func(t *testing.T) {
		// "👍🏽" is an emoji + skin-tone modifier: two runes, one grapheme
		// cluster, display width 2. The wrap logic must treat it as a single
		// unit so an oversized cluster does not produce a blank leading line
		// and a fitting cluster does not break across lines mid-cluster.
		cluster := "👍🏽"

		ti := newTextInput()
		ti.SetValue(cluster)
		// width=1: the cluster (width 2) is oversized for the line but must
		// count as exactly 1 visual line (drawn and clipped), exercising
		// multi-rune grapheme clustering in countVisualLines.
		assert.Equal(t, 1, ti.countVisualLines(1))

		// calcWrappedHeight should agree: oversized cluster => 1 line.
		assert.Equal(t, 1, calcWrappedHeight(cluster, 1))
		// "ab" + cluster at width 4: "ab" (x=2) then cluster (width 2) fits on
		// line 1; with width 3, cluster wraps to line 2.
		assert.Equal(t, 2, calcWrappedHeight("ab"+cluster, 3))
	})
}

func TestTextInput_WrappedLineNavigation(t *testing.T) {
	ti := newTextInput().WithMultilineMode(true)
	ti.SetFocused(true)
	ti.SetBounds(image.Rect(0, 0, 10, 3))
	ti.SetValue("abcdefghijklm") // row 0: "abcdefghij" (0..10), row 1: "klm" (10..13)

	// Offset 10 ('k') draws at column 0 of the wrapped second row.
	ti.CursorPos = 10
	assert.False(t, ti.atFirstVisualLine(), "cursor on wrapped row 1 must not report first visual line")

	// Up from column 0 of row 1 moves to column 0 of row 0 (offset 0).
	assert.True(t, ti.HandleKey(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, ti.CursorPos, 0)

	// Home and End move within the line between newlines, across soft wraps,
	// so typing after End appends to the line.
	ti.CursorPos = 3
	ti.HandleKey(KeyEvent{Key: KeyEnd})
	assert.Equal(t, ti.CursorPos, 13)
	ti.HandleKey(KeyEvent{Rune: 'X'})
	assert.Equal(t, ti.Value(), "abcdefghijklmX")
	ti.HandleKey(KeyEvent{Key: KeyHome})
	assert.Equal(t, ti.CursorPos, 0)

	// With a hard newline, Home and End stop at it.
	ti.SetValue("abcdefghijklm\nxyz")
	ti.CursorPos = 3
	ti.HandleKey(KeyEvent{Key: KeyEnd})
	assert.Equal(t, ti.CursorPos, 13)
	ti.CursorPos = 16
	ti.HandleKey(KeyEvent{Key: KeyHome})
	assert.Equal(t, ti.CursorPos, 14)

	// When row 0 wraps early before a wide char ("abcdefghi" + "中klmnopqr"),
	// pressing Up from the end of row 1 (col 10) must land on row 0 ('i', offset 8),
	// not on '中' (offset 9, which is row 1 col 0).
	ti.SetValue("abcdefghi中klmnopqr")
	assert.True(t, ti.HandleKey(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, ti.CursorPos, 8)
}

// A CRLF is one grapheme cluster, and it breaks the line as LF does. The
// cursor stops before the CR or after the LF, never between them, and the
// value keeps its CRLF.
func TestTextInput_CRLFBreaksLine(t *testing.T) {
	const value = "ab\r\ncd" // row 0: "ab" (0..2), CRLF (2..4), row 1: "cd" (4..6)

	t.Run("layout", func(t *testing.T) {
		ti := newTextInput().WithMultilineMode(true)
		ti.SetValue(value)
		assert.Equal(t, 2, ti.countVisualLines(10))
		assert.Equal(t, 2, calcWrappedHeight(value, 10))
		assert.Equal(t, []lineRange{{0, 2}, {4, 6}}, ti.getVisualLines(value))

		// Wrapping restarts after the CRLF: "abc" fills row 0 at width 3,
		// and "de" starts row 1 instead of wrapping after one cell.
		assert.Equal(t, 2, calcWrappedHeight("abc\r\nde", 3))
	})

	t.Run("cursor position", func(t *testing.T) {
		ti := newTextInput().WithMultilineMode(true)
		ti.SetValue(value)
		for _, tc := range []struct{ pos, x, y int }{
			{0, 0, 0}, {2, 2, 0}, {4, 0, 1}, {6, 2, 1},
		} {
			ti.CursorPos = tc.pos
			x, y := ti.getCursorXY(0, 0, 10)
			assert.Equal(t, [2]int{tc.x, tc.y}, [2]int{x, y}, "cursor at %d", tc.pos)
		}
	})

	t.Run("navigation", func(t *testing.T) {
		ti := newTextInput().WithMultilineMode(true)
		ti.SetFocused(true)
		ti.SetBounds(image.Rect(0, 0, 10, 3))
		ti.SetValue(value)

		// Left and Right step over the CRLF as one cluster.
		ti.CursorPos = 4
		ti.HandleKey(KeyEvent{Key: KeyArrowLeft})
		assert.Equal(t, 2, ti.CursorPos)
		ti.HandleKey(KeyEvent{Key: KeyArrowRight})
		assert.Equal(t, 4, ti.CursorPos)

		// Up and Down move between the rows, keeping the column.
		ti.CursorPos = 5
		assert.False(t, ti.atFirstVisualLine())
		assert.True(t, ti.HandleKey(KeyEvent{Key: KeyArrowUp}))
		assert.Equal(t, 1, ti.CursorPos)
		assert.True(t, ti.HandleKey(KeyEvent{Key: KeyArrowDown}))
		assert.Equal(t, 5, ti.CursorPos)
		assert.True(t, ti.atLastVisualLine())

		// Up from the end of row 1 stops before the CR.
		ti.CursorPos = 6
		assert.True(t, ti.HandleKey(KeyEvent{Key: KeyArrowUp}))
		assert.Equal(t, 2, ti.CursorPos)

		// Home and End stop at the CRLF on either side of it.
		ti.CursorPos = 5
		ti.HandleKey(KeyEvent{Key: KeyHome})
		assert.Equal(t, 4, ti.CursorPos)
		ti.HandleKey(KeyEvent{Key: KeyEnd})
		assert.Equal(t, 6, ti.CursorPos)
		ti.CursorPos = 1
		ti.HandleKey(KeyEvent{Key: KeyEnd})
		assert.Equal(t, 2, ti.CursorPos)
		ti.HandleKey(KeyEvent{Key: KeyHome})
		assert.Equal(t, 0, ti.CursorPos)

		// Typing at the start of row 1 keeps the CRLF intact.
		ti.CursorPos = 4
		ti.HandleKey(KeyEvent{Rune: 'X'})
		assert.Equal(t, "ab\r\nXcd", ti.Value())
	})
}
