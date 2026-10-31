package termtest

// eraseCell returns the cell that erase, insert, delete, and scroll
// operations leave behind. Like xterm and ghostty (background color erase,
// BCE), it takes the current background color and no other attribute, so
// "ESC[44m ESC[K" paints the rest of the line blue but does not underline
// or bold it.
func (s *Screen) eraseCell() Cell {
	return Cell{Char: ' ', Width: 1, Style: Style{Background: s.style.Background}}
}

// eraseRow returns a new row filled with erase cells.
func (s *Screen) eraseRow() []Cell {
	row := make([]Cell, s.width)
	s.eraseRange(row, 0, s.width)
	return row
}

// eraseRange fills row[from:to] with erase cells.
func (s *Screen) eraseRange(row []Cell, from, to int) {
	cell := s.eraseCell()
	for x := max(from, 0); x < to && x < len(row); x++ {
		row[x] = cell
	}
}
