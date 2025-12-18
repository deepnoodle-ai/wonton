package tui

import "fmt"

// RunOption is a functional option for configuring Run.
type RunOption func(*runConfig)

type runConfig struct {
	fps             int
	alternateScreen bool
	hideCursor      bool
	mouseTracking   bool
	bracketedPaste  bool
	pasteTabWidth   int
	inputSource     InputSource
}

func defaultRunConfig() runConfig {
	return runConfig{
		fps:             30,
		alternateScreen: true,
		hideCursor:      true,
		mouseTracking:   false,
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

// WithMouseTracking enables mouse event tracking.
// When enabled, the application will receive MouseEvent events.
func WithMouseTracking(enabled bool) RunOption {
	return func(c *runConfig) {
		c.mouseTracking = enabled
	}
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
func Run(app any, opts ...RunOption) error {
	// Validate app implements required interface
	_, isApp := app.(Application)
	if !isApp {
		return fmt.Errorf("app must implement Application (View())")
	}

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
	if cfg.mouseTracking {
		terminal.EnableMouseTracking()
	}
	if cfg.bracketedPaste {
		terminal.EnableBracketedPaste()
	}

	// Create and configure runtime
	runtime := NewRuntime(terminal, app, cfg.fps)
	runtime.SetPasteTabWidth(cfg.pasteTabWidth)

	// Ensure these modes are disabled on cleanup (terminal.Close doesn't handle this)
	if cfg.mouseTracking {
		defer terminal.DisableMouseTracking()
	}
	if cfg.bracketedPaste {
		defer terminal.DisableBracketedPaste()
	}

	// Run the application
	return runtime.Run()
}
