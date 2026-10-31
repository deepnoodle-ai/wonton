package tui

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/internal/terminalstate"
)

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
	terminal        *Terminal
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

// WithInput sets a custom input source for the runtime, in place of the
// keyboard. This is primarily used for testing. When the source ends, Run
// ends: on io.EOF it draws the last frame and returns nil, and on any other
// error it returns the error (see Runtime.SetInputSource).
func WithInput(source InputSource) RunOption {
	return func(c *runConfig) {
		c.inputSource = source
	}
}

// WithTerminal makes Run draw on term instead of opening a terminal on
// standard output. A test uses it to run the real Run path into an in-memory
// screen: the same setup, frames, resize handling and cleanup that users get.
// Make term with NewTestTerminal and a termtest.Screen, and send keys with a
// TestInput and WithInput. Run returns when the test closes the TestInput.
// Without WithInput, Run reads keys from standard input.
//
//	screen := termtest.NewScreen(80, 24)
//	term := tui.NewTestTerminal(80, 24, screen)
//	keys := tui.NewTestInput()
//	keys.Type("q")
//	keys.Close()
//	err := tui.Run(app, tui.WithTerminal(term), tui.WithInput(keys))
//
// Run owns term from the start of the call: it closes term when it returns,
// as it closes the terminal it opens itself. Close leaves the alternate
// screen, shows the cursor and turns off the modes Run turned on, so the
// screen shows what a user sees after the app quits. Do not use term after
// Run returns: a later Run with the same term returns an error that wraps
// ErrClosed. Make a new terminal for each Run.
//
// A terminal from NewTestTerminal has no tty. With one, Run leaves the
// process's own terminal alone: no raw mode and no keyboard protocol. So the
// bytes a test sees are the same in a terminal, in tmux and in CI.
//
// To resize while the app runs, resize the screen and then call
// term.SetSize with the same size, as Terminal.SetSize describes.
//
// A nil term keeps the default.
func WithTerminal(term *Terminal) RunOption {
	return func(c *runConfig) {
		c.terminal = term
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
// To test an application with the same path, see WithTerminal.
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

	// Create the terminal, unless WithTerminal gave one. Run owns it either
	// way and closes it on return.
	terminal := cfg.terminal
	if terminal == nil {
		var err error
		if terminal, err = NewTerminal(); err != nil {
			return err
		}
	} else if terminalstate.Closed(terminal) {
		return fmt.Errorf("tui: Run: the WithTerminal terminal is closed: %w", ErrClosed)
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

	runtime := newRunRuntime(terminal, app, cfg)

	// Reset requested modes before the deferred terminal.Close cleanup.
	if cfg.mouseMode != mouseOff {
		defer terminal.DisableMouseTracking()
	}
	if cfg.bracketedPaste {
		defer terminal.DisableBracketedPaste()
	}

	// Run the application
	return runtime.Run()
}

// newRunRuntime creates the Runtime for Run and applies the options that
// configure it.
func newRunRuntime(terminal *Terminal, app Application, cfg runConfig) *Runtime {
	runtime := NewRuntime(terminal, app, cfg.fps)
	if cfg.terminal != nil {
		// A terminal from NewTestTerminal has no tty. Then raw mode and the
		// keyboard protocol would only follow whether the test itself runs
		// in a terminal or in tmux, and change the bytes the test sees.
		_, hasTTY := terminalstate.Inspect(cfg.terminal)
		runtime.noTTY = !hasTTY
	}
	runtime.SetPasteTabWidth(cfg.pasteTabWidth)
	runtime.SetBackslashEnter(cfg.backslashEnter)
	if cfg.inputSource != nil {
		runtime.SetInputSource(cfg.inputSource)
	}
	return runtime
}
