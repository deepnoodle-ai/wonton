package termtest

// This file models the scroll region (DECSTBM, "CSI top ; bottom r"): the
// rows that line feeds, index, reverse index, scroll up and down, and
// insert and delete line act on. With no region set, the region is the
// whole screen. Behavior follows xterm and ghostty.

// setScrollRegion handles "CSI top ; bottom r". Parameters are 1-based; a
// missing or zero parameter means the screen edge, and a bottom past the
// screen means the last row. A region with fewer than two rows is
// ignored. A valid region moves the cursor home.
func (s *Screen) setScrollRegion(args []int) {
	top := getParam(args, 0, 1) - 1
	bottom := min(getParam(args, 1, s.height), s.height) - 1
	if top >= bottom {
		return
	}
	s.top, s.bottom = top, bottom
	s.cursorHome()
}

// resetScrollRegion makes the whole screen the scroll region.
func (s *Screen) resetScrollRegion() {
	s.top, s.bottom = 0, s.height-1
}

// cursorHome moves the cursor to the top-left corner, which is the top of
// the scroll region in origin mode.
func (s *Screen) cursorHome() {
	s.wrapPending = false
	s.cursorX = 0
	s.cursorY = 0
	if s.originMode {
		s.cursorY = s.top
	}
}

// setCursorRow moves the cursor to a 1-based row from CUP, HVP, or VPA. In
// origin mode the row counts from the top of the scroll region and the
// cursor stays inside the region.
func (s *Screen) setCursorRow(row int) {
	if s.originMode {
		s.cursorY = clamp(s.top+row-1, s.top, s.bottom)
		return
	}
	s.cursorY = clamp(row-1, 0, s.height-1)
}

// cursorUp moves the cursor up n rows. A cursor inside the scroll region
// stops at its top margin; a cursor above the region stops at row 0.
func (s *Screen) cursorUp(n int) {
	limit := 0
	if s.cursorY >= s.top {
		limit = s.top
	}
	s.cursorY = max(limit, s.cursorY-n)
}

// cursorDown moves the cursor down n rows. A cursor inside the scroll
// region stops at its bottom margin; a cursor below the region stops at
// the last row.
func (s *Screen) cursorDown(n int) {
	limit := s.height - 1
	if s.cursorY <= s.bottom {
		limit = s.bottom
	}
	s.cursorY = min(limit, s.cursorY+n)
}

// index moves the cursor down one row (LF, IND, NEL, autowrap). At the
// bottom of the scroll region it scrolls the region up instead. Below the
// region the cursor moves down to the last row and never scrolls.
func (s *Screen) index() {
	switch {
	case s.cursorY == s.bottom:
		s.scrollRegionUp(1)
	case s.cursorY < s.height-1:
		s.cursorY++
	}
}

// reverseIndex moves the cursor up one row (RI). At the top of the scroll
// region it scrolls the region down instead.
func (s *Screen) reverseIndex() {
	switch {
	case s.cursorY == s.top:
		s.scrollRegionDown(1)
	case s.cursorY > 0:
		s.cursorY--
	}
}

// scrollRegionUp scrolls the rows of the scroll region up by n. New rows
// at the bottom are erase cells. When the region starts at the top of the
// primary screen, the rows that leave it go to scrollback. The alternate
// screen has no scrollback.
func (s *Screen) scrollRegionUp(n int) {
	n = min(n, s.bottom-s.top+1)
	if n <= 0 {
		return
	}
	if s.top == 0 && !s.alt.active {
		for y := 0; y < n; y++ {
			s.scrollback = append(s.scrollback, s.Row(y))
		}
	}
	s.shiftRowsUp(s.top, s.bottom, n)
}

// scrollRegionDown scrolls the rows of the scroll region down by n. New
// rows at the top are erase cells. Rows that leave the bottom are dropped.
func (s *Screen) scrollRegionDown(n int) {
	s.shiftRowsDown(s.top, s.bottom, n)
}

// insertLines inserts n erase rows at the cursor row (IL). Rows from the
// cursor to the bottom of the scroll region move down; rows pushed past
// the bottom margin are dropped. The cursor moves to column 0 and a
// deferred wrap is cancelled. Outside the region it does nothing, not even
// move the cursor, as in xterm and ghostty.
func (s *Screen) insertLines(n int) {
	if s.cursorY < s.top || s.cursorY > s.bottom {
		return
	}
	s.shiftRowsDown(s.cursorY, s.bottom, n)
	s.cursorX = 0
	s.wrapPending = false
}

// deleteLines deletes n rows at the cursor row (DL). Rows below move up
// within the scroll region, and erase rows fill in at the bottom margin.
// Deleted rows never go to scrollback. The cursor moves to column 0 and a
// deferred wrap is cancelled. Outside the region it does nothing.
func (s *Screen) deleteLines(n int) {
	if s.cursorY < s.top || s.cursorY > s.bottom {
		return
	}
	s.shiftRowsUp(s.cursorY, s.bottom, n)
	s.cursorX = 0
	s.wrapPending = false
}

// shiftRowsUp moves rows top..bottom up by n and fills the freed rows at
// the bottom with erase cells.
func (s *Screen) shiftRowsUp(top, bottom, n int) {
	n = min(n, bottom-top+1)
	if n <= 0 {
		return
	}
	copy(s.cells[top:bottom+1], s.cells[top+n:bottom+1])
	for y := bottom - n + 1; y <= bottom; y++ {
		s.cells[y] = s.eraseRow()
	}
}

// shiftRowsDown moves rows top..bottom down by n and fills the freed rows
// at the top with erase cells.
func (s *Screen) shiftRowsDown(top, bottom, n int) {
	n = min(n, bottom-top+1)
	if n <= 0 {
		return
	}
	copy(s.cells[top+n:bottom+1], s.cells[top:bottom+1-n])
	for y := top; y < top+n; y++ {
		s.cells[y] = s.eraseRow()
	}
}
