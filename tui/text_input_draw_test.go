package tui

import (
	"strings"
	"testing"
)

// cursorCells counts the cells drawn with the input's cursor style.
func cursorCells(cell func(x, y int) Cell, ti *textInput, w, h int) int {
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if cell(x, y).Style == ti.CursorStyle {
				n++
			}
		}
	}
	return n
}

// A masked single-row input must keep its cursor visible at full width.
func TestTextInputMaskedCursorAtFullWidth(t *testing.T) {
	ti := newTextInput().WithMask('*')
	ti.SetFocused(true)
	for range 10 {
		ti.HandleKey(KeyEvent{Rune: 'x'})
	}
	cell := drawTextInput(t, ti, 10, 1)
	if cursorCells(cell, ti, 10, 1) != 1 {
		t.Fatal("masked cursor is not drawn at full width")
	}
}

// A single-line input squeezed to one row by its parent shows the wrapped
// row that holds the cursor, so the user sees what they type.
func TestTextInputOneRowOverflowShowsCursorRow(t *testing.T) {
	ti := newTextInput()
	ti.SetFocused(true)
	ti.SetValue("abcdefghij" + "klm")
	cell := drawTextInput(t, ti, 10, 1)
	if ti.ScrollOffset != 1 {
		t.Fatalf("ScrollOffset = %d, want 1", ti.ScrollOffset)
	}
	if cell(0, 0).Char != 'k' || cursorCells(cell, ti, 10, 1) != 1 {
		t.Fatalf("row shows %q, want the cursor row starting at 'k'", string(cell(0, 0).Char))
	}
}

// A multiline input whose height comes from the parent (MaxHeight == 0)
// must scroll to keep the cursor visible.
func TestTextInputScrollsWithoutMaxHeight(t *testing.T) {
	ti := newTextInput().WithMultilineMode(true)
	ti.SetFocused(true)
	ti.SetValue(strings.Repeat("line\n", 9) + "last")
	cell := drawTextInput(t, ti, 20, 3)
	if ti.ScrollOffset != 7 {
		t.Fatalf("ScrollOffset = %d, want 7", ti.ScrollOffset)
	}
	if cursorCells(cell, ti, 20, 3) != 1 {
		t.Fatal("cursor is not visible")
	}
}
