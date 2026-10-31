package termtest

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
// Erase in display with parameter 3 (CSI 3 J) and a reset (ESC c) empty the
// scrollback. Scrollback is text only: it keeps no styles.
func (s *Screen) Scrollback() []string {
	if len(s.scrollback) == 0 {
		return nil
	}
	return append([]string(nil), s.scrollback...)
}
