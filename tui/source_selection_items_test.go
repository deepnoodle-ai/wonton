package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/runewidth"
)

// columnOf is the screen column where sub starts in a rendered row.
func columnOf(row, sub string) int {
	return runewidth.StringWidth(row[:strings.Index(row, sub)])
}

// A viewport with a gap between items shows a blank row between them, and
// a selection across them copies that blank row as a blank line, the way a
// terminal would.
func TestSelectionAcrossItemsCopiesTheGapAsBlankLines(t *testing.T) {
	for _, gap := range []int{0, 1, 2} {
		want := "first" + strings.Repeat("\n", gap+1) + "second"

		source := &sourceItems{text: []string{"first", "second"}}
		s := &ViewportState{}
		renderViewport(t, s, source, 20, 10, gap)
		s.BeginSelection(2, 0)
		s.ExtendSelection(8, gap+1)
		s.EndSelection()
		assert.Equal(t, s.SelectedText(), want, "source items, gap %d", gap)

		legacy := &textItems{text: []string{"first", "second"}}
		s = &ViewportState{}
		renderViewport(t, s, legacy, 20, 10, gap)
		s.BeginSelection(0, 0)
		s.ExtendSelection(6, gap+1)
		s.EndSelection()
		assert.Equal(t, s.SelectedText(), want, "legacy items, gap %d", gap)
	}
}

// An item the selection only touches at its edge adds no blank lines.
func TestSelectionEndingInAGapAddsNoTrailingBlankLines(t *testing.T) {
	source := &sourceItems{text: []string{"first", "second", "third"}}
	s := &ViewportState{}
	renderViewport(t, s, source, 20, 10, 1)
	// Row 3 is the gap below "second": the drag ends at its end.
	s.BeginSelection(2, 0)
	s.ExtendSelection(4, 3)
	s.EndSelection()
	assert.Equal(t, s.SelectedText(), "first\n\nsecond")

	// Backwards from inside "third" to the gap above it.
	s.BeginSelection(5, 4)
	s.ExtendSelection(4, 3)
	s.EndSelection()
	assert.Equal(t, s.SelectedText(), "thi")
}

// Markdown draws the spaces between words itself, so they have no source
// bytes of their own. The highlight still runs through them: a selection is
// one continuous band on each row, not a row of separate words.
func TestSourceHighlightRunsThroughSpacesBetweenWords(t *testing.T) {
	s, v := newSourceViewport(t, "alpha beta gamma", true, 30)
	s.BeginSelection(2, 0)
	s.ExtendSelection(18, 0)
	s.EndSelection()
	assert.Equal(t, s.SelectedText(), "alpha beta gamma")

	var cells []string
	screen := renderViewport(t, s, v, 30, 20, 0)
	for x := range 30 {
		if screen.Cell(x, 0).Style.Reverse {
			cells = append(cells, string(screen.Cell(x, 0).Char))
		}
	}
	assert.Equal(t, strings.Join(cells, ""), "alpha beta gamma")
}

// The highlight does not spread onto decoration outside the selected text:
// a list bullet before the first selected word stays unhighlighted.
func TestSourceHighlightLeavesLeadingDecorationAlone(t *testing.T) {
	s, v := newSourceViewport(t, "- one two", true, 30)
	screen := renderViewport(t, s, v, 30, 20, 0)
	row := strings.TrimRight(strings.Split(screen.Text(), "\n")[0], " ")
	s.BeginSelection(columnOf(row, "one"), 0)
	s.ExtendSelection(runewidth.StringWidth(row), 0)
	s.EndSelection()
	screen = renderViewport(t, s, v, 30, 20, 0)
	assert.Equal(t, reversedRuns(screen, 30, 20), []string{"one two"})
}

// A press on decoration, such as a list bullet, starts the selection at the
// text after it when the drag goes forward, and ends it at the text before
// it when the drag goes backward. Before, both directions snapped to the
// previous text, so a forward drag copied the bullet's line break and syntax
// without highlighting them.
func TestSourceDragFromDecorationSnapsIntoTheSelection(t *testing.T) {
	s, v := newSourceViewport(t, "- one\n- two", true, 30)
	screen := renderViewport(t, s, v, 30, 20, 0)
	lines := strings.Split(screen.Text(), "\n")
	two := -1
	for y, line := range lines {
		if strings.Contains(line, "two") {
			two = y
		}
	}
	if two < 1 {
		t.Fatalf("no second item:\n%s", screen.Text())
	}
	bullet := strings.IndexFunc(lines[two], func(r rune) bool { return r != ' ' })
	end := columnOf(lines[two], "two") + 3

	// Forward from the bullet.
	s.BeginSelection(bullet, two)
	s.ExtendSelection(end, two)
	s.EndSelection()
	assert.Equal(t, s.SelectedText(), "two")

	// Backward onto the bullet.
	s.BeginSelection(end, two)
	s.ExtendSelection(bullet, two)
	s.EndSelection()
	assert.Equal(t, s.SelectedText(), "two")

	// Backward from the bullet up into the first item.
	s.BeginSelection(bullet, two)
	s.ExtendSelection(columnOf(lines[0], "one"), 0)
	s.EndSelection()
	assert.Equal(t, s.SelectedText(), "one")

	// A drag that stays on decoration selects nothing.
	s.BeginSelection(bullet, two)
	s.ExtendSelection(0, two)
	s.EndSelection()
	assert.False(t, s.HasSelection(), "decoration alone is not a selection")
}
