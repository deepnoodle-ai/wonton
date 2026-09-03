package tui

// RunOption is a functional option for configuring Run.
type RunOption func(*runConfig)

type runConfig struct {
	fps             int
	alternateScreen bool
	hideCursor      bool
	mouseMode       mouseMode
	bracketedPaste  bool
	pasteTabWidth   int
	backslashEnter  bool
	inputSource     InputSource
}

// mouseMode selects how much the terminal reports. Each level is a superset of
// the one above it, and more reporting means more input to decode: a terminal in
// mouseHover sends an event for every pixel of pointer movement over the window.
type mouseMode int

const (
	mouseOff    mouseMode = iota
	mouseButton           // ?1000: press, release, wheel
	mouseDrag             // ?1002: the above, plus motion while a button is held
	mouseHover            // ?1003: the above, plus motion with no button held
)

func defaultRunConfig() runConfig {
	return runConfig{
		fps:             30,
		alternateScreen: true,
		hideCursor:      true,
		mouseMode:       mouseOff,
		pasteTabWidth:   0,
	}
}

// WithInput sets a custom input source for the runtime.
// This is primarily used for testing.
func WithInput(source InputSource) RunOption {
	return func(c *runConfig) {
		c.inputSource = source
	}
}

// WithFPS sets the frames per second for TickEvents.
// Default is 30 FPS. Use 60 for smoother animations.
func WithFPS(fps int) RunOption {
	return func(c *runConfig) {
		if fps > 0 {
			c.fps = fps
		}
	}
}

// WithAlternateScreen controls whether to use the alternate screen buffer.
// When enabled (default), the terminal switches to a separate buffer and
// restores the original content on exit.
func WithAlternateScreen(enabled bool) RunOption {
	return func(c *runConfig) {
		c.alternateScreen = enabled
	}
}

// WithHideCursor controls whether to hide the cursor during rendering.
// Default is true. Set to false if your application manages cursor visibility.
func WithHideCursor(hide bool) RunOption {
	return func(c *runConfig) {
		c.hideCursor = hide
	}
}

// WithMouseTracking enables full mouse event tracking, hover included.
// When enabled, the application will receive MouseEvent events for every
// pointer movement over the window. Prefer WithMouseDrag unless the app
// actually paints hover state.
func WithMouseTracking(enabled bool) RunOption {
	return func(c *runConfig) {
		c.mouseMode = pickMouseMode(c.mouseMode, mouseHover, enabled)
	}
}

// WithMouseButtons enables mouse button and wheel events without any motion
// reporting. The quietest mode that still delivers clicks.
func WithMouseButtons(enabled bool) RunOption {
	return func(c *runConfig) {
		c.mouseMode = pickMouseMode(c.mouseMode, mouseButton, enabled)
	}
}

// WithMouseDrag enables button, wheel, and drag events — motion while a button
// is held, but not hover. This is what drag-to-select needs.
func WithMouseDrag(enabled bool) RunOption {
	return func(c *runConfig) {
		c.mouseMode = pickMouseMode(c.mouseMode, mouseDrag, enabled)
	}
}

// pickMouseMode keeps the most capable mode any option asked for, so the order
// options are passed in does not decide what the terminal reports.
func pickMouseMode(current, want mouseMode, enabled bool) mouseMode {
	if !enabled {
		if current == want {
			return mouseOff
		}
		return current
	}
	if want > current {
		return want
	}
	return current
}

// WithBracketedPaste enables bracketed paste mode.
// When enabled, the terminal can distinguish pasted text from typed text,
// allowing proper handling of multi-line pastes.
func WithBracketedPaste(enabled bool) RunOption {
	return func(c *runConfig) {
		c.bracketedPaste = enabled
	}
}

// WithPasteTabWidth configures how tabs in pasted content are handled.
// If width is 0 (default), tabs are preserved as-is.
// If width > 0, each tab is converted to that many spaces.
func WithPasteTabWidth(width int) RunOption {
	return func(c *runConfig) {
		c.pasteTabWidth = width
	}
}

// WithBackslashEnter enables synthesizing Shift+Enter from a backslash
// immediately followed by Enter — a fallback for terminals without the
// Kitty keyboard protocol, useful for chat-style apps where Shift+Enter
// inserts a newline.
//
// Disabled by default: when enabled, every typed backslash is delayed
// briefly while the runtime waits to see if Enter follows, and a backslash
// followed quickly by Enter is rewritten to Shift+Enter (losing the
// backslash). Only enable this for applications that need the Shift+Enter
// affordance.
func WithBackslashEnter(enabled bool) RunOption {
	return func(c *runConfig) {
		c.backslashEnter = enabled
	}
}

// Run is the simplest way to start a Wonton application.
// It creates a terminal, configures it, runs the application, and cleans up.
//
// The app parameter must implement Application.
//
// Example:
//
//	type MyApp struct {
//	    count int
//	}
//
//	func (app *MyApp) View() tui.View {
//	    return tui.Stack(
//	        tui.Text("Count: %d", app.count),
//	        tui.Clickable("[+]", func() { app.count++ }),
//	    )
//	}
//
//	func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
//	    if key, ok := event.(tui.KeyEvent); ok && key.Rune == 'q' {
//	        return []tui.Cmd{tui.Quit()}
//	    }
//	    return nil
//	}
//
//	func main() {
//	    if err := tui.Run(&MyApp{}); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// Options can be passed to customize behavior:
//
//	tui.Run(&MyApp{},
//	    tui.WithFPS(60),
//	    tui.WithMouseTracking(true),
//	)
func Run(app Application, opts ...RunOption) error {
	// Apply options
	cfg := defaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Create terminal
	terminal, err := NewTerminal()
	if err != nil {
		return err
	}
	defer terminal.Close()

	// Configure terminal
	if cfg.alternateScreen {
		terminal.EnableAlternateScreen()
	}
	if cfg.hideCursor {
		terminal.HideCursor()
	}
	switch cfg.mouseMode {
	case mouseButton:
		terminal.EnableMouseButtons()
	case mouseDrag:
		terminal.EnableMouseDrag()
	case mouseHover:
		terminal.EnableMouseTracking()
	}
	if cfg.bracketedPaste {
		terminal.EnableBracketedPaste()
	}

	// Create and configure runtime
	runtime := NewRuntime(terminal, app, cfg.fps)
	runtime.SetPasteTabWidth(cfg.pasteTabWidth)
	runtime.SetBackslashEnter(cfg.backslashEnter)

	// Ensure these modes are disabled on cleanup (terminal.Close doesn't handle this)
	if cfg.mouseMode != mouseOff {
		defer terminal.DisableMouseTracking()
	}
	if cfg.bracketedPaste {
		defer terminal.DisableBracketedPaste()
	}

	// Run the application
	return runtime.Run()
}
