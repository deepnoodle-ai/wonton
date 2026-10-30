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

func TestTextInputWrappedCursorAtLineStart(t *testing.T) {
	t.Run("cursor on first char of wrapped line", func(t *testing.T) {
		ti := newTextInput()
		ti.SetFocused(true)
		ti.SetValue("abcdefghijklm")
		ti.CursorPos = 10 // 'k', drawn at row 1 col 0

		cell := drawTextInput(t, ti, 10, 2)
		if cell(0, 1).Char != 'k' || cell(0, 1).Style != ti.CursorStyle {
			t.Fatalf("cell(0,1) = %q style=%v, want cursor on 'k'", string(cell(0, 1).Char), cell(0, 1).Style)
		}
		if cursorCells(cell, ti, 10, 2) != 1 {
			t.Fatalf("cursorCells = %d, want 1", cursorCells(cell, ti, 10, 2))
		}

		// In a 1-row viewport, keeping the cursor on 'k' must stay on row 1.
		cell = drawTextInput(t, ti, 10, 1)
		if ti.ScrollOffset != 1 {
			t.Fatalf("ScrollOffset = %d, want 1", ti.ScrollOffset)
		}
		if cell(0, 0).Char != 'k' || cursorCells(cell, ti, 10, 1) != 1 {
			t.Fatalf("1-row cell(0,0) = %q cursorCells=%d, want cursor on 'k'", string(cell(0, 0).Char), cursorCells(cell, ti, 10, 1))
		}
	})

	t.Run("cursor on wide char that wraps to next line", func(t *testing.T) {
		ti := newTextInput()
		ti.SetFocused(true)
		ti.SetValue("abcdefghi中")
		ti.CursorPos = 9 // '中' (width 2) wraps to row 1 col 0 in a 10-col input

		if gotLine, gotX := ti.getCursorLine(10), ti.getCursorXInLine(10); gotLine != 1 || gotX != 0 {
			t.Fatalf("cursor at (line=%d, x=%d), want (line=1, x=0)", gotLine, gotX)
		}

		// In a 1-row viewport, auto-scroll must scroll to row 1 so '中' and the cursor are visible.
		cell := drawTextInput(t, ti, 10, 1)
		if ti.ScrollOffset != 1 {
			t.Fatalf("ScrollOffset = %d, want 1", ti.ScrollOffset)
		}
		if cell(0, 0).Char != '中' || cell(0, 0).Style != ti.CursorStyle {
			t.Fatalf("cell(0,0) = %q style=%v, want cursor on '中'", string(cell(0, 0).Char), cell(0, 0).Style)
		}
	})
}

// A CRLF draws as a line break, and the cursor on it draws as a blank cell
// at the end of the row, as it does on an LF.
func TestTextInputDrawsCRLFAsLineBreak(t *testing.T) {
	ti := newTextInput().WithMultilineMode(true)
	ti.SetFocused(true)
	ti.SetValue("ab\r\ncd")
	ti.CursorPos = 2
	cell := drawTextInput(t, ti, 10, 3)
	rows := []string{"", "", ""}
	for y := range rows {
		for x := 0; x < 10; x++ {
			rows[y] += string(cell(x, y).Char)
		}
	}
	if got := strings.Join(rows, "|"); got != "ab        |cd        |          " {
		t.Fatalf("rows = %q, want \"ab\" and \"cd\" on separate rows", got)
	}
	if cell(2, 0).Style != ti.CursorStyle || cursorCells(cell, ti, 10, 3) != 1 {
		t.Fatalf("cursor is not drawn at the end of row 0")
	}
	if ti.Value() != "ab\r\ncd" {
		t.Fatalf("Value() = %q, want the CRLF kept", ti.Value())
	}
}

// Text that exactly fills the width leaves the cursor in the last column.
func TestTextInputCursorAtExactFullWidth(t *testing.T) {
	ti := newTextInput()
	ti.SetFocused(true)
	ti.SetValue("abcdefghij")
	cell := drawTextInput(t, ti, 10, 1)
	if cursorCells(cell, ti, 10, 1) != 1 {
		t.Fatal("cursor is not drawn when the text fills the width")
	}
}

// MaxLength also limits pasted text and Shift+Enter newlines.
func TestTextInputMaxLengthLimitsPasteAndNewline(t *testing.T) {
	ti := newTextInput().WithMultilineMode(true)
	ti.MaxLength = 5
	ti.HandlePaste("abc\ndefgh")
	if ti.Value() != "abc\nd" {
		t.Fatalf("Value() = %q, want the paste cut to 5 runes", ti.Value())
	}
	ti.HandleKey(KeyEvent{Key: KeyEnter, Shift: true})
	if ti.Value() != "abc\nd" {
		t.Fatalf("Value() = %q, want Shift+Enter ignored at MaxLength", ti.Value())
	}
	if !ti.HandlePaste("x") || ti.Value() != "abc\nd" {
		t.Fatalf("Value() = %q, want a paste at MaxLength consumed and ignored", ti.Value())
	}
}
