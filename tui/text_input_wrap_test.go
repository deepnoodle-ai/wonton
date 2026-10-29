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

	// Place cursor at column 0 of the wrapped second visual line ('k', offset 10).
	ti.CursorPos = 10
	assert.False(t, ti.atFirstVisualLine(), "cursor on wrapped row 1 must not report first visual line")

	// Home on column 0 of row 1 stays at offset 10 instead of jumping to row 0.
	ti.HandleKey(KeyEvent{Key: KeyHome})
	assert.Equal(t, 10, ti.CursorPos)

	// End on column 0 of row 1 moves to the end of row 1 (offset 13).
	ti.HandleKey(KeyEvent{Key: KeyEnd})
	assert.Equal(t, 13, ti.CursorPos)

	// Up from column 0 of row 1 moves to column 0 of row 0 (offset 0).
	ti.CursorPos = 10
	assert.True(t, ti.HandleKey(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, 0, ti.CursorPos)

	// End on a soft-wrapped line stops at the last character on that visual line ('j', offset 9),
	// not the first character of the next visual line ('k', offset 10); typing then inserts before 'j'.
	ti.CursorPos = 3
	ti.HandleKey(KeyEvent{Key: KeyEnd})
	assert.Equal(t, 9, ti.CursorPos)
	ti.HandleKey(KeyEvent{Rune: 'X'})
	assert.Equal(t, "abcdefghiXjklm", ti.Value())
	assert.Equal(t, 10, ti.CursorPos)

	// When row 0 wraps early before a wide char ("abcdefghi" + "中klmnopqr"),
	// pressing Up from the end of row 1 (col 10) must land on row 0 ('i', offset 8),
	// not on '中' (offset 9, which is row 1 col 0).
	ti.SetValue("abcdefghi中klmnopqr")
	assert.True(t, ti.HandleKey(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, 8, ti.CursorPos)
}
