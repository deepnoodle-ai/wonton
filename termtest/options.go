package termtest

// ScreenOption configures a new Screen. Pass options to NewScreen, or to a
// helper that makes a screen, such as Capture.Screen or NewRecorder.
type ScreenOption func(*Screen)

// applyOptions applies opts to s in order.
func (s *Screen) applyOptions(opts []ScreenOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
}

// WithRawMode sets whether the screen models a terminal whose tty is in raw
// mode. Use it to test output that a program writes after it puts the tty in
// raw mode, such as tui.InlineApp, the full-screen runtime, or output from
// tui.Print with tui.WithRawMode(true).
//
// In raw mode the tty sends bytes to the terminal unchanged, so a bare line
// feed ("\n") moves the cursor down one row and keeps its column. Only a
// carriage return ("\r") moves the cursor to column 0. Output that forgets a
// "\r" before a new line then shows text that steps to the right, as it does
// in a real terminal.
//
// The default is false: cooked mode, where the tty adds a carriage return
// before each line feed, so a bare "\n" also moves to column 0. All other
// bytes behave the same in both modes. Screen.Reset keeps the mode, because
// a terminal reset does not change the tty.
func WithRawMode(enabled bool) ScreenOption {
	return func(s *Screen) {
		s.rawMode = enabled
	}
}
