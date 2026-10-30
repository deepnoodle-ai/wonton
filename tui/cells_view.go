package tui

import (
	"image"
	"strings"
)

// CellsView draws a grid of terminal cells as they are: characters, styles,
// wide characters and all. It is how a TUI shows a screen it did not lay out
// itself, such as a snapshot of a terminal running another program.
type CellsView struct {
	rows        [][]Cell
	cursorX     int
	cursorY     int
	cursor      bool
	cursorStyle *Style
	sel         GridSelection
	selStyle    *Style
}

// Cells draws rows of cells, row 0 at the top. A wide character's second
// column is a continuation cell, as in terminal.Cell; a cell with no
// character draws as a space. The view asks for the grid's size; cells past
// the edge of the space it is given are clipped.
func Cells(rows [][]Cell) *CellsView {
	return &CellsView{rows: rows}
}

// Cursor draws a block cursor on the cell at (x, y), by reversing the cell's
// colors unless CursorStyle is set.
func (v *CellsView) Cursor(x, y int) *CellsView {
	v.cursorX, v.cursorY, v.cursor = x, y, true
	return v
}

// CursorStyle sets the style the cursor cell is drawn in, in place of
// reversing it.
func (v *CellsView) CursorStyle(s Style) *CellsView {
	v.cursorStyle = &s
	return v
}

// Select highlights a selection, by reversing the selected cells' colors
// unless SelectStyle is set. An empty selection highlights nothing.
func (v *CellsView) Select(sel GridSelection) *CellsView {
	v.sel = sel
	return v
}

// SelectStyle sets the style selected cells are drawn in, in place of
// reversing them.
func (v *CellsView) SelectStyle(s Style) *CellsView {
	v.selStyle = &s
	return v
}

func (v *CellsView) size(maxWidth, maxHeight int) (int, int) {
	w := 0
	for _, row := range v.rows {
		w = max(w, len(row))
	}
	h := len(v.rows)
	if maxWidth > 0 {
		w = min(w, maxWidth)
	}
	if maxHeight > 0 {
		h = min(h, maxHeight)
	}
	return w, h
}

func (v *CellsView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	for y := 0; y < height && y < len(v.rows); y++ {
		row := v.rows[y]
		for x := 0; x < width && x < len(row); x++ {
			c := row[x]
			if c.Continuation {
				continue
			}
			char := c.Char
			if char == 0 {
				char = ' '
			}
			if c.Trailing == "" && c.Width <= 1 {
				ctx.SetCell(x, y, char, c.Style)
				continue
			}
			if x+c.Width > width {
				// Half a wide character does not fit: leave the column blank.
				ctx.SetCell(x, y, ' ', c.Style)
				continue
			}
			ctx.PrintTruncated(x, y, string(char)+c.Trailing, c.Style)
		}
	}
	if !v.sel.Empty() {
		start, end := v.sel.Ordered()
		for y := max(start.Y, 0); y <= end.Y && y < height; y++ {
			x0, x1 := v.sel.span(y, width)
			for x0 > 0 && x0 < width && ctx.Cell(x0, y).Continuation {
				x0-- // highlight the wide character Text copies
			}
			for x := x0; x < x1; x++ {
				cell := ctx.Cell(x, y)
				if cell.Continuation {
					continue
				}
				style := cell.Style
				if v.selStyle != nil {
					style = *v.selStyle
				} else {
					style.Reverse = !style.Reverse
				}
				ctx.RestyleCell(x, y, style)
			}
		}
	}
	if v.cursor && v.cursorX >= 0 && v.cursorY >= 0 && v.cursorX < width && v.cursorY < height {
		style := ctx.Cell(v.cursorX, v.cursorY).Style
		if v.cursorStyle != nil {
			style = *v.cursorStyle
		} else {
			style.Reverse = !style.Reverse
		}
		ctx.RestyleCell(v.cursorX, v.cursorY, style)
	}
}

// GridSelection is a selection on a grid of cells, made the way a terminal
// makes one: from the cell where a drag started, the Anchor, to the cell
// under the pointer, the Cursor. Both are included, along with the rest of
// the first row, every row in between, and the start of the last row.
//
// The application keeps one while the user drags: set Anchor on the press,
// move Cursor as the pointer moves, and pass it to CellsView.Select to draw
// it. Text returns what it covers.
type GridSelection struct {
	Anchor image.Point
	Cursor image.Point
}

// Empty reports whether the selection covers nothing, which is the case
// until the pointer moves off the cell where the drag started.
func (s GridSelection) Empty() bool { return s.Anchor == s.Cursor }

// Ordered returns the ends of the selection in reading order.
func (s GridSelection) Ordered() (start, end image.Point) {
	a, c := s.Anchor, s.Cursor
	if c.Y < a.Y || (c.Y == a.Y && c.X < a.X) {
		return c, a
	}
	return a, c
}

// Contains reports whether the cell at (x, y) is selected.
func (s GridSelection) Contains(x, y int) bool {
	if s.Empty() {
		return false
	}
	x0, x1 := s.span(y, x+1)
	return x >= x0 && x < x1
}

// span returns the columns [x0, x1) selected on row y of a row width wide.
func (s GridSelection) span(y, width int) (int, int) {
	start, end := s.Ordered()
	if y < start.Y || y > end.Y {
		return 0, 0
	}
	x0, x1 := 0, width
	if y == start.Y {
		x0 = max(start.X, 0)
	}
	if y == end.Y {
		x1 = min(end.X+1, width)
	}
	return x0, max(x0, x1)
}

// Text returns the selected text in rows: one line per row, without the
// blanks at the end of each line, joined by newlines. A wide character is
// included whole if any of its columns is selected.
//
// wrapped marks the rows that continue on the next row, as a terminal wraps
// a line too long for its width; nil means none do. A wrapped row joins the
// next with no line break and keeps its spaces, so a long command or URL
// copies as the one line it was.
func (s GridSelection) Text(rows [][]Cell, wrapped []bool) string {
	if s.Empty() {
		return ""
	}
	start, end := s.Ordered()
	var b strings.Builder
	for y := max(start.Y, 0); y <= end.Y && y < len(rows); y++ {
		row := rows[y]
		x0, x1 := s.span(y, len(row))
		for x0 > 0 && x0 < len(row) && row[x0].Continuation {
			x0-- // back to the wide character this column belongs to
		}
		var line strings.Builder
		for x := x0; x < x1; x++ {
			c := row[x]
			if c.Continuation {
				continue
			}
			if c.Char == 0 {
				line.WriteByte(' ')
				continue
			}
			line.WriteRune(c.Char)
			line.WriteString(c.Trailing)
		}
		if y < end.Y && y < len(wrapped) && wrapped[y] {
			b.WriteString(line.String())
			continue
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		if y < end.Y && y+1 < len(rows) {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
