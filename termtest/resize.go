package termtest

// rowMeta is what reflow needs to know about a row beyond its cells.
//
// Rows are keyed by their first cell, so the metadata moves with the row
// slice when the screen scrolls or inserts or deletes lines, and a freshly
// allocated row starts with none.
type rowMeta struct {
	// wrapped is set when printed text overflowed this row and continued
	// on the next one (a soft wrap), so reflow joins the two rows.
	wrapped bool
	// spacerHead is set when the row wrapped early because a wide
	// character did not fit, leaving a blank pad cell in the last column
	// that is not part of the text.
	spacerHead bool
	// printed marks cells a program printed. A printed space looks the
	// same as an erased cell, but reflow keeps it as content.
	printed []bool
}

// meta returns the metadata of row y, or nil if the row has none.
func (s *Screen) meta(y int) *rowMeta {
	if s.rowInfo == nil || y < 0 || y >= s.height {
		return nil
	}
	return s.rowInfo[&s.cells[y][0]]
}

// metaFor returns the metadata of row y, creating it if needed.
func (s *Screen) metaFor(y int) *rowMeta {
	if m := s.meta(y); m != nil {
		return m
	}
	if s.rowInfo == nil {
		s.rowInfo = make(map[*Cell]*rowMeta)
	}
	if len(s.rowInfo) > 2*s.height {
		s.pruneMeta()
	}
	m := &rowMeta{}
	s.rowInfo[&s.cells[y][0]] = m
	return m
}

// pruneMeta drops metadata of rows that scrolled off or were deleted.
func (s *Screen) pruneMeta() {
	live := make(map[*Cell]*rowMeta, s.height)
	for y := 0; y < s.height; y++ {
		key := &s.cells[y][0]
		if m, ok := s.rowInfo[key]; ok {
			live[key] = m
		}
	}
	s.rowInfo = live
}

// markPrinted records that cells [x, x+n) of row y were printed.
func (s *Screen) markPrinted(y, x, n int) {
	m := s.metaFor(y)
	if m.printed == nil {
		m.printed = make([]bool, s.width)
	}
	for i := max(x, 0); i < x+n && i < s.width; i++ {
		m.printed[i] = true
	}
}

// markWrapped records that row y soft-wraps into the next row. spacer
// reports that the last cell is a pad left by a wide character that did
// not fit.
func (s *Screen) markWrapped(y int, spacer bool) {
	m := s.metaFor(y)
	m.wrapped = true
	m.spacerHead = spacer
}

// markErased records that cells [from, to) of row y were erased. unwrap
// ends the row's soft wrap, as erasing to the end of a row does in ghostty.
func (s *Screen) markErased(y, from, to int, unwrap bool) {
	m := s.meta(y)
	if m == nil {
		return
	}
	if unwrap {
		m.wrapped = false
		m.spacerHead = false
	}
	if m.printed != nil {
		for i := max(from, 0); i < to && i < s.width; i++ {
			m.printed[i] = false
		}
	}
}

// shiftPrinted moves the printed marks of row y right by n cells from
// column from (left when n is negative), as ICH and DCH move cells.
func (s *Screen) shiftPrinted(y, from, n int) {
	m := s.meta(y)
	if m == nil || m.printed == nil {
		return
	}
	shifted := make([]bool, s.width)
	copy(shifted, m.printed[:max(from, 0)])
	for x := from; x < s.width; x++ {
		if to := x + n; to >= from && to < s.width {
			shifted[to] = m.printed[x]
		}
	}
	m.printed = shifted
}

// contentEnd returns the column after the last cell of row y that holds
// content: a printed cell, or a cell that differs from an erased one.
func (s *Screen) contentEnd(y int) int {
	m := s.meta(y)
	for x := s.width - 1; x >= 0; x-- {
		if s.cells[y][x] != blankCell() || (m != nil && m.printed != nil && m.printed[x]) {
			return x + 1
		}
	}
	return 0
}

// rowLen returns how many cells of row y belong to its logical line. A
// soft-wrapped row is full, less a pad cell left by a wide character.
func (s *Screen) rowLen(y int) int {
	m := s.meta(y)
	if m == nil || !m.wrapped {
		return s.contentEnd(y)
	}
	if m.spacerHead && s.cells[y][s.width-1] == blankCell() &&
		(m.printed == nil || !m.printed[s.width-1]) {
		return s.width - 1
	}
	return s.width
}

// Resize changes the screen size, as a user resizes a terminal window.
// Write the program's redraw after it, as the program would send it after
// SIGWINCH.
//
// Resize reflows text the way ghostty does, and most modern terminals
// (iTerm2, kitty, WezTerm) do the same:
//
//   - Text that wrapped at the right edge is a soft-wrapped line. It
//     rewraps at the new width: narrower moves text onto more rows, wider
//     joins the rows again. A line that ended with a line feed never joins
//     the next one, but it wraps if it no longer fits.
//   - A wide character that does not fit at the end of a row moves to the
//     next row whole. No half character is left.
//   - The cursor and the saved cursor (ESC 7, CSI s) stay on the same
//     character. One whose row leaves the top goes to the home position.
//   - With fewer rows, blank rows below the cursor go first, then rows
//     leave the top. termtest has no scrollback, so those rows are lost: a
//     later Resize to more rows adds blank rows at the bottom, where a
//     real terminal would bring them back.
//
// The alternate screen (see IsAlternateScreen) does not reflow, as in
// ghostty: each row is cut at the new right edge, a wide character whose
// second half falls off the edge becomes blank, and widening adds blank
// cells. The primary screen behind it still reflows, so the shell screen
// that comes back when the application quits is the reflowed one.
//
// Width and height under 1 are treated as 1, as in NewScreen. Resizing to
// the current size changes nothing. The current style, any partly written
// escape sequence, and a pending wrap carry over.
//
// A test of a program that queries its size also tells the program the
// new size, after the Resize. For a wonton terminal.Terminal, that is
// SetSize with the same size.
func (s *Screen) Resize(width, height int) {
	width, height = max(width, 1), max(height, 1)
	if width == s.width && height == s.height {
		return
	}
	oldWidth, oldHeight := s.width, s.height
	if s.alt.hidden != nil {
		// Resize the hidden screen first, as if it were visible, with
		// the cursor it had when it was hidden and its saved cursor.
		s.swapGrids()
		saved := &s.alt.saved
		marks := s.resizeGrid(width, height, s.alt.active,
			s.alt.cursor, mark{saved.x, saved.y, saved.wrap})
		s.alt.cursor = marks[0]
		saved.x, saved.y, saved.wrap = marks[1].x, marks[1].y, marks[1].pending
		s.swapGrids()
		s.width, s.height = oldWidth, oldHeight
	}
	marks := s.resizeGrid(width, height, !s.alt.active,
		mark{s.cursorX, s.cursorY, s.wrapPending}, mark{s.savedX, s.savedY, s.savedWrap})
	s.cursorX, s.cursorY, s.wrapPending = marks[0].x, marks[0].y, marks[0].pending
	s.savedX, s.savedY, s.savedWrap = marks[1].x, marks[1].y, marks[1].pending
	// A resize resets the scroll region to the whole screen, as in ghostty
	// and xterm. Origin mode stays set.
	s.resetScrollRegion()
}

// mark is a cursor position that Resize moves to the new grid.
type mark struct {
	x, y    int
	pending bool // a wrap is pending at the mark
}

// swapGrids swaps the visible grid and its row metadata with the hidden
// grid and its metadata. Calling it twice restores the screen.
func (s *Screen) swapGrids() {
	s.cells, s.alt.hidden = s.alt.hidden, s.cells
	s.rowInfo, s.alt.hiddenInfo = s.alt.hiddenInfo, s.rowInfo
}

// resizeGrid resizes the visible grid and returns where the marks moved.
// marks[0] is the cursor and marks[1] the saved cursor. It reflows
// soft-wrapped lines when reflow is set; otherwise it cuts each row at the
// new width.
//
// Like the cursor, the saved cursor stays on its character, as ghostty
// tracks it, and goes home if its row leaves the top.
func (s *Screen) resizeGrid(width, height int, reflow bool, marks ...mark) []mark {
	var rows []resizedRow
	moved := make([]mark, len(marks))
	if width == s.width || !reflow {
		rows = make([]resizedRow, s.height)
		for y := range rows {
			rows[y] = s.truncatedRow(y, width)
		}
		for i, m := range marks {
			if m.pending && width > s.width {
				// The cell after the mark now exists, so the wrap is no
				// longer pending: the mark moves onto that cell.
				m.x, m.pending = m.x+1, false
			}
			moved[i] = m
		}
	} else {
		rows, moved = s.reflow(width, marks)
	}

	// Fit the rows to the new height: drop blank rows below the cursor,
	// then rows from the top, then pad with blank rows at the bottom.
	for len(rows) > height && len(rows)-1 > moved[0].y && rows[len(rows)-1].blank() {
		rows = rows[:len(rows)-1]
	}
	if drop := len(rows) - height; drop > 0 {
		rows = rows[drop:]
		for i := range moved {
			if moved[i].y -= drop; moved[i].y < 0 {
				// The mark's row left the top: ghostty puts it at home.
				moved[i] = mark{}
			}
		}
	}
	for len(rows) < height {
		rows = append(rows, resizedRow{cells: blankRow(width)})
	}

	s.width, s.height = width, height
	s.cells = make([][]Cell, height)
	s.rowInfo = nil
	for y, row := range rows {
		s.cells[y] = row.cells
		if row.meta != nil {
			if s.rowInfo == nil {
				s.rowInfo = make(map[*Cell]*rowMeta)
			}
			s.rowInfo[&s.cells[y][0]] = row.meta
		}
	}
	for i, m := range moved {
		m.x = clamp(m.x, 0, width-1)
		m.y = clamp(m.y, 0, height-1)
		m.pending = m.pending && m.x == width-1
		moved[i] = m
	}
	return moved
}

// truncatedRow returns row y cut or padded to width, in a new slice. A wide
// character whose second half falls off the new edge becomes blank. When
// the width is unchanged, the row keeps its metadata; otherwise it keeps
// only its printed marks, since a cut row no longer wraps.
func (s *Screen) truncatedRow(y, width int) resizedRow {
	cells := blankRow(width)
	copy(cells, s.cells[y])
	if width < s.width && cells[width-1].Width == 2 {
		cells[width-1] = blankCell()
	}
	m := s.meta(y)
	if m == nil || width == s.width {
		return resizedRow{cells: cells, meta: m}
	}
	cut := &rowMeta{}
	if m.printed != nil {
		cut.printed = make([]bool, width)
		copy(cut.printed, m.printed)
	}
	return resizedRow{cells: cells, meta: cut}
}

// resizedRow is one row of the screen being built by Resize.
type resizedRow struct {
	cells []Cell
	meta  *rowMeta
}

func (r resizedRow) blank() bool {
	if r.meta != nil && r.meta.wrapped {
		return false
	}
	for x, c := range r.cells {
		if c != blankCell() || (r.meta != nil && r.meta.printed != nil && r.meta.printed[x]) {
			return false
		}
	}
	return true
}

func blankRow(width int) []Cell {
	row := make([]Cell, width)
	for x := range row {
		row[x] = blankCell()
	}
	return row
}

// lineGlyph is one glyph of a logical line, at offset off in the line.
type lineGlyph struct {
	cell    Cell
	width   int
	printed bool
	off     int
}

// reflow rewraps every logical line at width and returns the new rows
// with the new mark positions; marks[0] is the cursor. It does not fit the
// rows to a height.
func (s *Screen) reflow(width int, marks []mark) (rows []resizedRow, moved []mark) {
	moved = make([]mark, len(marks))
	for y := 0; y < s.height; {
		// Collect the rows of one logical line.
		first := y
		for y < s.height-1 {
			if m := s.meta(y); m == nil || !m.wrapped {
				break
			}
			y++
		}
		last := y
		y++

		var glyphs []lineGlyph
		lineLen := 0
		offs := make([]int, len(marks))
		for i := range offs {
			offs[i] = -1
		}
		for r := first; r <= last; r++ {
			n := s.rowLen(r)
			if r == last {
				n = s.contentEnd(r)
			}
			m := s.meta(r)
			for x := 0; x < n; x++ {
				c := s.cells[r][x]
				if c.Continuation {
					continue
				}
				glyphs = append(glyphs, lineGlyph{
					cell:    c,
					width:   max(c.Width, 1),
					printed: m != nil && m.printed != nil && m.printed[x],
					off:     lineLen + x,
				})
			}
			for i, mk := range marks {
				if r == mk.y {
					offs[i] = lineLen + mk.x
				}
			}
			lineLen += n
		}

		l := layoutLine(glyphs, width, marks, offs)
		for i, off := range offs {
			if off >= 0 {
				moved[i] = l.marks[i]
				moved[i].y += len(rows)
			}
		}
		rows = append(rows, l.rows...)
	}
	return rows, moved
}

type laidOutLine struct {
	rows  []resizedRow
	marks []mark // row within the line and column of each mark on it
}

// layoutLine places the glyphs of one logical line on rows of the given
// width. offs[i] is the offset of marks[i] in the line, or -1 if the mark
// is not on it. The line gets blank rows down to the cursor (marks[0]) if
// the cursor is past the end of the text.
func layoutLine(glyphs []lineGlyph, width int, marks []mark, offs []int) laidOutLine {
	out := laidOutLine{marks: make([]mark, len(marks))}
	row := resizedRow{cells: blankRow(width)}
	col := 0

	// Where each offset of the line landed: its row, its column, and the
	// last column of the glyph that covers it.
	type pos struct{ row, col, end int }
	at := map[int]pos{}
	for _, g := range glyphs {
		c, w := g.cell, g.width
		if w > width {
			// A wide character never fits a one-column row.
			c, w = blankCell(), 1
		}
		if col+w > width {
			if row.meta == nil {
				row.meta = &rowMeta{}
			}
			row.meta.wrapped, row.meta.spacerHead = true, col < width
			out.rows = append(out.rows, row)
			row = resizedRow{cells: blankRow(width)}
			col = 0
		}
		for i := 0; i < g.width; i++ {
			at[g.off+i] = pos{len(out.rows), min(col+i, col+w-1), col + w - 1}
		}
		row.cells[col] = c
		if w == 2 {
			row.cells[col+1] = continuationCell(c.Style)
		}
		if g.printed {
			if row.meta == nil {
				row.meta = &rowMeta{}
			}
			if row.meta.printed == nil {
				row.meta.printed = make([]bool, width)
			}
			for i := col; i < col+w; i++ {
				row.meta.printed[i] = true
			}
		}
		col += w
	}
	out.rows = append(out.rows, row)
	lineLen := 0
	if len(glyphs) > 0 {
		g := glyphs[len(glyphs)-1]
		lineLen = g.off + g.width
	}

	for i, off := range offs {
		if off < 0 {
			continue
		}
		var m mark
		if p, ok := at[off]; ok {
			m.y, m.x = p.row, p.col
			if marks[i].pending {
				// The mark waits after the glyph it sits on. It keeps
				// waiting if that glyph still ends a row; otherwise it
				// moves to the cell after the glyph.
				if p.end == width-1 {
					m.x, m.pending = p.end, true
				} else {
					m.x = p.end + 1
				}
			}
		} else {
			// The mark is past the end of the text. Keep its distance
			// from the text, wrapping it as if blank cells filled the gap.
			if marks[i].pending {
				off++
			}
			r, c := len(out.rows)-1, col
			if c >= width {
				r, c = r+1, 0
			}
			for o := lineLen; o < off; o++ {
				if c++; c >= width {
					r, c = r+1, 0
				}
			}
			m.y, m.x = r, c
		}
		out.marks[i] = m
	}

	if offs[0] >= 0 {
		for len(out.rows) <= out.marks[0].y {
			out.rows = append(out.rows, resizedRow{cells: blankRow(width)})
		}
	}
	for i := 1; i < len(offs); i++ {
		if offs[i] >= 0 && out.marks[i].y >= len(out.rows) {
			// A saved cursor past the end of the rows stays on the
			// line's last row.
			out.marks[i].y, out.marks[i].x = len(out.rows)-1, width-1
		}
	}
	return out
}
