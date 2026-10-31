package termtest

import "strings"

// Scrollback returns the rows that scrolled off the top of the screen,
// oldest first. Each row is text, as Row returns it.
//
// A row enters scrollback when a line feed, index, next line, autowrap, or
// scroll up (CSI S) scrolls a scroll region that starts at the top row of
// the screen. With no scroll region set, the region is the whole screen, so
// ordinary output that scrolls the screen adds its rows here. Rows that
// leave a region that starts lower, and rows removed by delete line
// (CSI M), are dropped, as they are in xterm and ghostty.
//
// Resize moves rows between the screen and the scrollback, as ghostty does:
// rows pushed off the top go to scrollback, a taller screen takes rows
// back from it when the cursor is on the last row, and a width change
// reflows soft-wrapped lines across both.
//
// Only the primary screen has scrollback. While the alternate screen is
// visible (see IsAlternateScreen), Scrollback returns nil and the primary
// screen's scrollback is kept for when it comes back.
//
// Erase in display with parameter 3 (CSI 3 J) on the primary screen and a
// reset (ESC c) empty the scrollback. Scrollback reports text only; the
// rows keep their styles for when Resize brings them back.
func (s *Screen) Scrollback() []string {
	if len(s.scrollback) == 0 || s.alt.active {
		return nil
	}
	out := make([]string, len(s.scrollback))
	for i, row := range s.scrollback {
		out[i] = rowText(row.cells)
	}
	return out
}

// pushScrollback moves row y, with its metadata, to the end of the
// scrollback. The caller replaces row y in s.cells.
func (s *Screen) pushScrollback(y int) {
	row := gridRow{cells: s.cells[y], meta: s.meta(y)}
	if row.meta != nil {
		delete(s.rowInfo, &s.cells[y][0])
	}
	s.scrollback = append(s.scrollback, row)
}

// rowText returns the text of a row with trailing spaces trimmed.
func rowText(cells []Cell) string {
	var line strings.Builder
	for _, cell := range cells {
		if cell.Continuation {
			continue
		}
		writeCellGlyph(&line, cell)
	}
	return strings.TrimRight(line.String(), " ")
}
