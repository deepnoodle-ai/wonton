package gooey

// RunOption is a functional option for configuring Run.
type RunOption func(*runConfig)

type runConfig struct {
	fps             int
	alternateScreen bool
	hideCursor      bool
	mouseTracking   bool
	pasteTabWidth   int
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

// WithPasteTabWidth configures how tabs in pasted content are handled.
// If width is 0 (default), tabs are preserved as-is.
// If width > 0, each tab is converted to that many spaces.
func WithPasteTabWidth(width int) RunOption {
	return func(c *runConfig) {
		c.pasteTabWidth = width
	}
}

// Run is the simplest way to start a Gooey application.
// It creates a terminal, configures it, runs the application, and cleans up.
//
// Example:
//
//	type MyApp struct {
//	    // state
//	}
//
//	func (app *MyApp) HandleEvent(event gooey.Event) []gooey.Cmd {
//	    // handle events
//	    return nil
//	}
//
//	func (app *MyApp) Render(frame gooey.RenderFrame) {
//	    // render state
//	}
//
//	func main() {
//	    if err := gooey.Run(&MyApp{}); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// Options can be passed to customize behavior:
//
//	gooey.Run(&MyApp{},
//	    gooey.WithFPS(60),
//	    gooey.WithMouseTracking(true),
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
	if cfg.mouseTracking {
		terminal.EnableMouseTracking()
	}

	// Create and configure runtime
	runtime := NewRuntime(terminal, app, cfg.fps)
	runtime.SetPasteTabWidth(cfg.pasteTabWidth)

	// Ensure mouse tracking is disabled on cleanup (terminal.Close doesn't handle this)
	if cfg.mouseTracking {
		defer terminal.DisableMouseTracking()
	}

	// Run the application
	return runtime.Run()
}
