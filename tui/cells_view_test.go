package tui

import (
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func textRows(lines ...string) [][]Cell {
	rows := make([][]Cell, len(lines))
	for y, line := range lines {
		for _, r := range line {
			rows[y] = append(rows[y], Cell{Char: r, Width: 1})
		}
	}
	return rows
}

func TestGridSelectionText(t *testing.T) {
	rows := textRows(
		"first line  ",
		"second line ",
		"third line  ",
	)
	// Dragged backwards: from "line" on row 2 up to "line" on row 0.
	sel := GridSelection{Anchor: image.Pt(3, 2), Cursor: image.Pt(6, 0)}
	start, end := sel.Ordered()
	assert.Equal(t, start, image.Pt(6, 0))
	assert.Equal(t, end, image.Pt(3, 2))
	assert.Equal(t, sel.Text(rows, nil), "line\nsecond line\nthir")

	assert.True(t, sel.Contains(6, 0))
	assert.False(t, sel.Contains(5, 0))
	assert.True(t, sel.Contains(11, 1))
	assert.True(t, sel.Contains(3, 2))
	assert.False(t, sel.Contains(4, 2))

	// A press that has not moved selects nothing.
	none := GridSelection{Anchor: image.Pt(2, 1), Cursor: image.Pt(2, 1)}
	assert.True(t, none.Empty())
	assert.Equal(t, none.Text(rows, nil), "")
	assert.False(t, none.Contains(2, 1))
}

func TestGridSelectionWideCharacters(t *testing.T) {
	rows := [][]Cell{{
		{Char: 'a', Width: 1},
		{Char: '日', Width: 2}, {Continuation: true},
		{Char: 'b', Width: 1},
		{}, // never written: a space
		{Char: 'c', Width: 1},
	}}
	// Starting on the second half of 日 takes the whole character.
	sel := GridSelection{Anchor: image.Pt(2, 0), Cursor: image.Pt(5, 0)}
	assert.Equal(t, sel.Text(rows, nil), "日b c")
}

func TestCellsViewSelect(t *testing.T) {
	rows := textRows("abcd", "efgh")
	sel := GridSelection{Anchor: image.Pt(2, 0), Cursor: image.Pt(0, 1)}
	screen := SprintScreen(Cells(rows).Select(sel), WithWidth(4), WithHeight(2))
	assert.Equal(t, screen.Row(0), "abcd")
	for _, c := range []struct {
		x, y     int
		selected bool
	}{{1, 0, false}, {2, 0, true}, {3, 0, true}, {0, 1, true}, {1, 1, false}} {
		assert.Equal(t, screen.Cell(c.x, c.y).Style.Reverse, c.selected, "cell %d,%d", c.x, c.y)
	}
}

func TestGridSelectionJoinsWrappedRows(t *testing.T) {
	rows := textRows(
		"see https:",
		"//example.",
		"com now   ",
		"next      ",
	)
	sel := GridSelection{Anchor: image.Pt(4, 0), Cursor: image.Pt(9, 3)}
	wrapped := []bool{true, true, false, false}
	assert.Equal(t, sel.Text(rows, wrapped), "https://example.com now\nnext")

	// A space at the end of a wrapped row is part of the line.
	rows = textRows("one two ", "three   ")
	sel = GridSelection{Anchor: image.Pt(0, 0), Cursor: image.Pt(7, 1)}
	assert.Equal(t, sel.Text(rows, []bool{true}), "one two three")
}
