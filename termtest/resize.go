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
	// continued is set on the row that wrapped text continues on. It
	// usually matches the row above being wrapped, but insert and delete
	// line end it on the rows they move while the row above keeps its
	// wrap, as in ghostty. Resize counts these rows to keep the rows below
	// the cursor.
	continued bool
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

// markContinued records that row y continues a soft-wrapped row.
func (s *Screen) markContinued(y int) {
	s.metaFor(y).continued = true
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

// contentEnd returns the column after the last cell of the row that holds
// content: a printed cell, or a cell that differs from an erased one.
func (r gridRow) contentEnd() int {
	for x := len(r.cells) - 1; x >= 0; x-- {
		if r.cells[x] != blankCell() || r.printedAt(x) {
			return x + 1
		}
	}
	return 0
}

// lineLen returns how many cells of the row belong to its logical line. A
// soft-wrapped row is full, less a pad cell left by a wide character.
func (r gridRow) lineLen() int {
	if !r.wrapped() {
		return r.contentEnd()
	}
	last := len(r.cells) - 1
	if r.meta.spacerHead && r.cells[last] == blankCell() && !r.printedAt(last) {
		return last
	}
	return len(r.cells)
}

func (r gridRow) wrapped() bool { return r.meta != nil && r.meta.wrapped }

func (r gridRow) printedAt(x int) bool {
	return r.meta != nil && r.meta.printed != nil && x < len(r.meta.printed) && r.meta.printed[x]
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
//     next row whole. No half character is left. A cursor or saved cursor
//     on it stays on the blank cell it leaves at the end of the row, as in
//     ghostty.
//   - The cursor and the saved cursor (ESC 7, CSI s) stay on the same
//     character. One whose row leaves the top goes to the home position.
//     If the cursor waited to wrap (it wrote the last column), it goes to
//     the second column of the top row instead, when the width allows. At
//     width 1 it stays in the first column and still waits to wrap.
//     A saved cursor in the blank cells after a line's text stays on the
//     row where the text of its old row now starts, as in ghostty. Its
//     column is cut to fit that row, so it can land on text.
//   - With fewer rows, blank rows below the cursor go first, then rows
//     leave the top for the scrollback (see Scrollback). With more rows,
//     rows come back from the scrollback if the cursor is on the last
//     row; otherwise blank rows are added at the bottom.
//   - Soft-wrapped lines in the scrollback reflow too, also a line that
//     starts in the scrollback and ends on the screen. When wider rows
//     leave room on the screen, rows come back from the scrollback, and
//     the cursor keeps as many rows below it as before.
//   - The scroll region resets to the whole screen. Origin mode stays.
//
// The alternate screen (see IsAlternateScreen) does not reflow, as in
// ghostty: each row is cut at the new right edge, a wide character whose
// second half falls off the edge becomes blank, and widening adds blank
// cells. It has no scrollback, so rows that leave its top are lost. The
// primary screen behind it still reflows, so the shell screen that comes
// back when the application quits is the reflowed one.
//
// Width and height under 1 are treated as 1, as in NewScreen. Resizing to
// the current size changes nothing. The current style, any partly written
// escape sequence, and a pending wrap carry over.
//
// A test of a program that queries its size also tells the program the
// new size, after the Resize. For a wonton terminal.Terminal, that is
// SetSize with the same size.
func (s *Screen) Resize(width, height int) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
// marks[0] is the cursor and marks[1] the saved cursor. The primary screen
// reflows soft-wrapped lines, together with its scrollback; the alternate
// screen cuts each row at the new width and has no scrollback.
//
// Like the cursor, the saved cursor stays on its character, as ghostty
// tracks it, and goes home if its row leaves the top.
func (s *Screen) resizeGrid(width, height int, primary bool, marks ...mark) []mark {
	g := &resizer{width: s.width, height: s.height}
	// A reflow can change every scrollback row, so it takes them all.
	// Without one, rows only move between the screen and the end of the
	// scrollback, and a screen of the new height takes back at most height
	// rows, so the older rows stay where they are, untouched.
	reflow := primary && width != s.width
	var kept []gridRow
	if primary {
		from := 0
		if !reflow {
			from = max(len(s.scrollback)-height, 0)
		}
		kept = s.scrollback[:from]
		g.rows = append(g.rows, s.scrollback[from:]...)
	}
	g.top = len(g.rows)
	for y := 0; y < s.height; y++ {
		g.rows = append(g.rows, gridRow{cells: s.cells[y], meta: s.meta(y)})
	}
	for _, m := range marks {
		m.y += g.top
		g.marks = append(g.marks, m)
	}

	// Like ghostty, change the row count before the reflow when the
	// screen narrows and after it when the screen widens. Both steps use
	// the cursor row from before the resize, as ghostty does.
	cursorY := marks[0].y
	switch {
	case width == s.width:
		g.fitHeight(height, cursorY)
	case !primary:
		g.truncate(width)
		g.fitHeight(height, cursorY)
	case width < s.width:
		g.fitHeight(height, cursorY)
		g.reflow(width, cursorY)
	default:
		g.reflow(width, cursorY)
		g.fitHeight(height, cursorY)
	}

	if primary {
		// kept is a prefix of the old scrollback and g.rows holds copies
		// of the rows after it, so appending in place overwrites nothing
		// still needed.
		s.scrollback = append(kept, g.rows[:g.top]...)
	}
	s.width, s.height = width, height
	s.cells = make([][]Cell, height)
	s.rowInfo = nil
	for y, row := range g.rows[g.top : g.top+height] {
		s.cells[y] = row.cells
		if row.meta != nil {
			if s.rowInfo == nil {
				s.rowInfo = make(map[*Cell]*rowMeta)
			}
			s.rowInfo[&s.cells[y][0]] = row.meta
		}
	}
	moved := make([]mark, len(g.marks))
	for i, m := range g.marks {
		if m.y -= g.top; m.y < 0 {
			// The mark's row left the top: ghostty puts it at home. The
			// cursor keeps the pending wrap it had before the resize,
			// which the next lines turn into a move to the next cell, as
			// ghostty does.
			m = mark{pending: i == 0 && marks[0].pending}
		}
		if m.pending && m.x < width-1 {
			m.x, m.pending = m.x+1, false
		}
		m.x = clamp(m.x, 0, width-1)
		m.y = clamp(m.y, 0, height-1)
		m.pending = m.pending && m.x == width-1
		moved[i] = m
	}
	return moved
}

// resizer holds the rows of one screen while Resize works on them: the
// scrollback rows (primary screen only), then the screen rows from index
// top. Mark rows are indexes into rows; marks[0] is the cursor.
type resizer struct {
	rows          []gridRow
	top           int
	width, height int
	marks         []mark
}

// truncate cuts or pads every row to width, without reflow.
func (g *resizer) truncate(width int) {
	for i, r := range g.rows {
		g.rows[i] = r.truncated(width)
	}
	if width > g.width {
		for i, m := range g.marks {
			if m.pending {
				// The cell after the mark now exists, so the wrap is no
				// longer pending: the mark moves onto that cell.
				g.marks[i].x, g.marks[i].pending = m.x+1, false
			}
		}
	}
	g.width = width
}

// fitHeight changes the number of screen rows without reflow, as ghostty
// does. Fewer rows: blank rows at the bottom go first, then rows leave the
// top for the scrollback. More rows: if the cursor was on the last row,
// rows come back from the scrollback; otherwise, and when the scrollback
// runs out, blank rows are added at the bottom. cursorY is the cursor's
// screen row before the resize.
func (g *resizer) fitHeight(height, cursorY int) {
	switch {
	case height < g.height:
		for n := g.height - height; n > 0 && g.droppable(len(g.rows)-1) && g.rows[len(g.rows)-1].blank(); n-- {
			g.rows = g.rows[:len(g.rows)-1]
		}
		g.top = max(g.top, len(g.rows)-height)
	case height > g.height && cursorY >= g.height-1:
		g.top = max(0, len(g.rows)-height)
	}
	g.padRows(height)
	g.height = height
}

// reflow rewraps every logical line, scrollback included, at width. Like
// ghostty, it drops blank rows at the end, shows the last rows, and then
// adds blank rows at the bottom so the cursor keeps as many rows below it
// as it had, less the rows that new soft wraps above it take. cursorY is
// the cursor's screen row before the resize, which ghostty counts from
// even when the row count already changed.
func (g *resizer) reflow(width, cursorY int) {
	below := max(g.height-1-cursorY, 0)
	cursor := g.marks[0].y
	onScreen := cursor >= g.top
	wrapsBefore := 0
	if onScreen {
		wrapsBefore = continuations(g.rows, g.top, cursor)
	}

	g.rows, g.marks = reflowRows(g.rows, width, g.marks)
	g.width = width
	for len(g.rows) > 1 && g.droppable(len(g.rows)-1) && g.rows[len(g.rows)-1].contentEnd() == 0 &&
		!g.rows[len(g.rows)-1].wrapped() {
		g.rows = g.rows[:len(g.rows)-1]
	}
	g.top = 0
	g.padRows(g.height)
	g.top = len(g.rows) - g.height

	cursor = g.marks[0].y
	if onScreen && cursor >= g.top {
		wrapsNow := continuations(g.rows, g.top, cursor)
		have := g.height - 1 - (cursor - g.top)
		for n := below - max(wrapsNow-wrapsBefore, 0) - have; n > 0; n-- {
			g.rows = append(g.rows, gridRow{cells: blankRow(width)})
		}
		g.top = len(g.rows) - g.height
	}
}

// droppable reports whether row y may be dropped from the bottom: no mark
// (cursor or saved cursor) is on it or below it.
func (g *resizer) droppable(y int) bool {
	for _, m := range g.marks {
		if m.y >= y {
			return false
		}
	}
	return true
}

// continuations counts the rows from..to that continue a soft-wrapped row.
func continuations(rows []gridRow, from, to int) int {
	n := 0
	for y := from; y <= to && y < len(rows); y++ {
		if rows[y].meta != nil && rows[y].meta.continued {
			n++
		}
	}
	return n
}

// padRows adds blank rows at the bottom until the screen has height rows.
func (g *resizer) padRows(height int) {
	for len(g.rows)-g.top < height {
		g.rows = append(g.rows, gridRow{cells: blankRow(g.width)})
	}
}

// truncated returns the row cut or padded to width, in a new slice. A wide
// character whose second half falls off the new edge becomes blank. When
// the width is unchanged, the row keeps its metadata; otherwise it keeps
// only its printed marks, since a cut row no longer wraps.
func (r gridRow) truncated(width int) gridRow {
	old := len(r.cells)
	cells := blankRow(width)
	copy(cells, r.cells)
	if width < old && cells[width-1].Width == 2 {
		cells[width-1] = blankCell()
	}
	if r.meta == nil || width == old {
		return gridRow{cells: cells, meta: r.meta}
	}
	cut := &rowMeta{}
	if r.meta.printed != nil {
		cut.printed = make([]bool, width)
		copy(cut.printed, r.meta.printed)
	}
	return gridRow{cells: cells, meta: cut}
}

// gridRow is one row with its metadata: a scrollback row, or a row of the
// screen being built by Resize.
type gridRow struct {
	cells []Cell
	meta  *rowMeta
}

// blank reports whether the row has no text, so a shorter screen can drop
// it. Like ghostty, it counts printed cells (spaces too) as text, but not
// cells that only have a background color from an erase.
func (r gridRow) blank() bool {
	if r.wrapped() {
		return false
	}
	for x, c := range r.cells {
		if (c.Char != ' ' && c.Char != 0) || c.Trailing != "" || r.printedAt(x) {
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

// reflowRows rewraps every logical line of rows at width and returns the
// new rows with the new mark positions; marks[0] is the cursor. It does not
// fit the rows to a height.
func reflowRows(src []gridRow, width int, marks []mark) (rows []gridRow, moved []mark) {
	moved = make([]mark, len(marks))
	rows = make([]gridRow, 0, len(src))
	// The glyph and offset buffers are reused for every line, since
	// layoutLine does not keep them.
	var glyphs []lineGlyph
	offs := make([]int, len(marks))
	for y := 0; y < len(src); {
		// Collect the rows of one logical line.
		first := y
		for y < len(src)-1 && src[y].wrapped() {
			y++
		}
		last := y
		y++

		glyphs = glyphs[:0]
		lineLen := 0
		lastStart := -1 // where the last row starts in the line, if it ends the line
		for i := range offs {
			offs[i] = -1
		}
		for r := first; r <= last; r++ {
			row := src[r]
			n := row.lineLen()
			if r == last {
				n = row.contentEnd()
				if !row.wrapped() {
					lastStart = lineLen
				}
			}
			for x := 0; x < n; x++ {
				c := row.cells[x]
				if c.Continuation {
					continue
				}
				glyphs = append(glyphs, lineGlyph{
					cell:    c,
					width:   max(c.Width, 1),
					printed: row.printedAt(x),
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

		l := layoutLine(glyphs, width, marks, offs, lastStart)
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
	rows  []gridRow
	marks []mark // row within the line and column of each mark on it
}

// layoutLine places the glyphs of one logical line on rows of the given
// width. offs[i] is the offset of marks[i] in the line, or -1 if the mark
// is not on it. The line gets blank rows down to the cursor (marks[0]) if
// the cursor is past the end of the text.
//
// lastStart is the offset where the line's last row starts, or -1 if that
// row soft-wraps. A saved cursor in the blank cells after the text of that
// row is cut as ghostty's reflowRow cuts a tracked pin: its column in the
// old row is cut to the cells left after the text that comes before that
// row on the new rows. So it never moves to the rows that the cursor adds
// below the text.
func layoutLine(glyphs []lineGlyph, width int, marks []mark, offs []int, lastStart int) laidOutLine {
	out := laidOutLine{marks: make([]mark, len(marks))}
	row := gridRow{cells: blankRow(width)}
	col := 0
	// freeCol is the first free column after the glyphs that come before
	// the line's last row: ghostty's dst_x when its reflow reaches that row.
	// A wide character that starts the row and then wraps does not change
	// it, because ghostty cuts the pin before it writes that character.
	freeCol := 0

	// Where each offset of the line landed: its row, its column, and the
	// last column of the glyph that covers it. Only the marks need it, so
	// a line without one, nearly every line, skips it.
	type pos struct{ row, col, end int }
	var at map[int]pos
	for _, off := range offs {
		if off >= 0 {
			at = map[int]pos{}
			break
		}
	}
	for _, g := range glyphs {
		c, w := g.cell, g.width
		if w > width {
			// A wide character never fits a one-column row.
			c, w = blankCell(), 1
		}
		spacer := -1 // the pad cell left when a wide character wraps
		if col+w > width {
			if row.meta == nil {
				row.meta = &rowMeta{}
			}
			row.meta.wrapped, row.meta.spacerHead = true, col < width
			if col < width {
				spacer = col
			}
			out.rows = append(out.rows, row)
			row = gridRow{cells: blankRow(width), meta: &rowMeta{continued: true}}
			col = 0
		}
		for i := 0; at != nil && i < g.width; i++ {
			at[g.off+i] = pos{len(out.rows), min(col+i, col+w-1), col + w - 1}
		}
		if at != nil && spacer >= 0 {
			// A mark on a wide character that wraps stays on the pad cell
			// it leaves behind, as ghostty moves its pins.
			at[g.off] = pos{len(out.rows) - 1, spacer, spacer}
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
		if lastStart >= 0 && g.off < lastStart {
			freeCol = col
		}
	}
	out.rows = append(out.rows, row)
	lineLen := 0
	if len(glyphs) > 0 {
		g := glyphs[len(glyphs)-1]
		lineLen = g.off + g.width
	}
	if freeCol >= width {
		// The text before the last row filled its new row, so ghostty's
		// reflow has a pending wrap and the last row starts on the next.
		freeCol = 0
	}

	for i, off := range offs {
		if off < 0 {
			continue
		}
		saved := i > 0 && lastStart >= 0 && off >= lineLen
		if saved {
			off = lastStart + min(off-lastStart, width-1-freeCol)
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
			// A saved cursor keeps its pending wrap for resizeGrid.
			if saved {
				m.pending = marks[i].pending
			} else if marks[i].pending {
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
		// The cells up to a cursor past the end of the text belong to the
		// line, as in ghostty, so the rows they fill soft-wrap.
		for len(out.rows) <= out.marks[0].y {
			last := &out.rows[len(out.rows)-1]
			if last.meta == nil {
				last.meta = &rowMeta{}
			}
			last.meta.wrapped, last.meta.spacerHead = true, false
			out.rows = append(out.rows, gridRow{cells: blankRow(width), meta: &rowMeta{continued: true}})
		}
	}
	for i := 1; i < len(offs); i++ {
		if offs[i] >= 0 && out.marks[i].y >= len(out.rows) {
			// Only when the line's last row soft-wraps (lastStart is -1)
			// can a saved cursor fall past the rows. Keep it on the last.
			out.marks[i].y, out.marks[i].x = len(out.rows)-1, width-1
		}
	}
	return out
}
