package termtest

// Alternate screen support (DEC private modes 47, 1047 and 1049).
//
// A Screen shows one of two grids, like a terminal window: the primary screen
// or the alternate screen that full-screen applications draw on. The visible
// grid is always s.cells, so every read and write method acts on what a user
// would see. Switching swaps s.cells with the hidden grid kept here, together
// with each screen's saved cursor. The cursor position, the current style and
// the pending wrap belong to the terminal, not to a screen, so they carry over.
//
// The semantics follow xterm and were checked against ghostty:
//
//	?47h    switch to alternate              ?47l    switch to primary
//	?1047h  switch to alternate              ?1047l  clear alternate, switch to primary
//	?1049h  save cursor, switch, clear       ?1049l  switch to primary, restore cursor

// altState holds the hidden grid and the alternate-screen flag.
type altState struct {
	active bool     // the alternate screen is visible
	hidden [][]Cell // the grid that is not visible; nil until the first switch
	saved  savedCursor
}

// savedCursor is the DECSC state of the hidden screen. Each screen has its own.
type savedCursor struct {
	x, y  int
	style Style
	wrap  bool
}

// IsAlternateScreen reports whether the alternate screen is visible. It is
// true after an application sends ESC[?1049h (or ?1047h, ?47h) and false again
// after the matching reset. While it is true, Text, Row, Cell, the assertions
// and snapshots all show the alternate screen; the primary screen comes back
// unchanged when the application leaves it.
func (s *Screen) IsAlternateScreen() bool {
	return s.alt.active
}

// setPrivateModes applies CSI ? Pm h (set) or CSI ? Pm l (reset). Only the
// alternate screen modes change the screen; other private modes are ignored.
func (s *Screen) setPrivateModes(modes []int, set bool) {
	for _, mode := range modes {
		switch mode {
		case 47:
			s.switchScreen(set)
		case 1047:
			if !set && s.alt.active {
				s.clearCells()
				s.wrapPending = false
			}
			s.switchScreen(set)
		case 1049:
			if set {
				s.saveCursor()
				s.switchScreen(true)
				s.clearCells()
			} else {
				s.switchScreen(false)
				s.restoreCursor()
			}
		}
	}
}

// switchScreen makes the alternate screen (alternate true) or the primary
// screen visible. Switching to the screen that is already visible does nothing.
func (s *Screen) switchScreen(alternate bool) {
	if s.alt.active == alternate {
		return
	}
	if s.alt.hidden == nil {
		s.alt.hidden = make([][]Cell, s.height)
		for y := range s.alt.hidden {
			s.alt.hidden[y] = make([]Cell, s.width)
			for x := range s.alt.hidden[y] {
				s.alt.hidden[y][x] = blankCell()
			}
		}
	}
	s.cells, s.alt.hidden = s.alt.hidden, s.cells
	visible := savedCursor{x: s.savedX, y: s.savedY, style: s.savedStyle, wrap: s.savedWrap}
	s.savedX, s.savedY, s.savedStyle, s.savedWrap = s.alt.saved.x, s.alt.saved.y, s.alt.saved.style, s.alt.saved.wrap
	s.alt.saved = visible
	s.alt.active = alternate
}

// resetAltScreen returns to the primary screen and discards the alternate
// screen and its saved cursor (RIS).
func (s *Screen) resetAltScreen() {
	s.switchScreen(false)
	s.alt = altState{}
}

// saveCursor saves the cursor position, style and pending wrap (DECSC).
func (s *Screen) saveCursor() {
	s.savedX = s.cursorX
	s.savedY = s.cursorY
	s.savedStyle = s.style
	s.savedWrap = s.wrapPending
}

// restoreCursor restores the state saved by saveCursor (DECRC). Without a
// save, the cursor goes home with the default style.
func (s *Screen) restoreCursor() {
	s.cursorX = s.savedX
	s.cursorY = s.savedY
	s.style = s.savedStyle
	s.wrapPending = s.savedWrap
}
