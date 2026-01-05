package tui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/terminal"
	"golang.org/x/term"
)

// =============================================================================
// INLINE APPLICATION
// =============================================================================
//
// InlineApp provides a way to build terminal applications that coexist with
// normal terminal output, rather than taking over the entire screen like Run().
//
// # Concept
//
// An inline application has two output areas:
//
//   - Scrollback: Normal terminal output that scrolls up as new content is added.
//     Use Print() or Printf() to add content here.
//
//   - Live Region: A fixed area at the bottom that updates in place without
//     scrolling. This is rendered by LiveView() and typically contains input
//     prompts, status indicators, or interactive controls.
//
// This pattern is ideal for:
//   - Chat interfaces (messages scroll, input stays at bottom)
//   - Build tools (logs scroll, progress bar stays visible)
//   - REPLs (history scrolls, prompt stays at bottom)
//   - Any app that needs both streaming output and interactive controls
//
// # Architecture
//
// InlineApp uses the same proven three-goroutine architecture as Run():
//
//   - Goroutine 1: Main event loop (processes events sequentially)
//   - Goroutine 2: Input reader (blocks on stdin, sends KeyEvents/MouseEvents)
//   - Goroutine 3: Command executor (runs async Cmd functions)
//
// This design eliminates race conditions in application code. HandleEvent and
// LiveView are NEVER called concurrently - you don't need locks.
//
// # Rendering Optimization
//
// InlineApp uses several techniques to minimize flicker:
//
//   - Line-level diffing: Only lines that changed are redrawn
//   - Synchronized output mode: Terminal buffers changes and renders atomically
//   - Atomic Print: Scrollback output and live region re-render happen together
//
// # Example
//
//	type CounterApp struct {
//	    runner *tui.InlineApp
//	    count  int
//	}
//
//	func (app *CounterApp) LiveView() tui.View {
//	    return tui.Stack(
//	        tui.Divider(),
//	        tui.Text(" Count: %d", app.count).Bold(),
//	        tui.Text(" Press +/- to change, q to quit").Dim(),
//	        tui.Divider(),
//	    )
//	}
//
//	func (app *CounterApp) HandleEvent(event tui.Event) []tui.Cmd {
//	    if key, ok := event.(tui.KeyEvent); ok {
//	        switch key.Rune {
//	        case '+':
//	            app.count++
//	            app.runner.Printf("Incremented to %d", app.count)
//	        case '-':
//	            app.count--
//	            app.runner.Printf("Decremented to %d", app.count)
//	        case 'q':
//	            return []tui.Cmd{tui.Quit()}
//	        }
//	    }
//	    return nil
//	}
//
//	func main() {
//	    app := &CounterApp{}
//	    app.runner = tui.NewInlineApp()
//	    if err := app.runner.Run(app); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// =============================================================================

// InlineApplication is the interface for inline applications.
// Applications implementing this interface render a live region that updates in place,
// while supporting scrollback output via the runner's Print() method.
//
// Thread Safety: LiveView and HandleEvent (if implemented) are NEVER called concurrently.
// The InlineApp ensures these methods run sequentially in a single goroutine,
// eliminating the need for locks in application code.
//
// Optional Interfaces:
//   - EventHandler: Implement HandleEvent(Event) []Cmd for event handling
//   - Initializable: Implement Init() error for setup before the event loop
//   - Destroyable: Implement Destroy() for cleanup after the event loop
type InlineApplication interface {
	// LiveView returns the view for the live region.
	// Called after each event is processed to re-render the live region.
	// The view should represent the current state of the interactive area.
	LiveView() View
}

// InlineOption configures an InlineApp.
type InlineOption func(*inlineConfig)

// inlineConfig holds configuration for InlineApp.
type inlineConfig struct {
	width          int
	output         io.Writer
	input          io.Reader
	fps            int
	mouseTracking  bool
	bracketedPaste bool
	pasteTabWidth  int
	kittyKeyboard  bool
}

func defaultInlineConfig() inlineConfig {
	// Try to get terminal width, fall back to 80
	width := 80
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}
	return inlineConfig{
		width:          width,
		output:         os.Stdout,
		input:          os.Stdin,
		fps:            0, // No ticks by default (different from Run which defaults to 30)
		mouseTracking:  false,
		bracketedPaste: false,
		pasteTabWidth:  0,
		kittyKeyboard:  false,
	}
}

// WithInlineWidth sets the rendering width. Default is terminal width or 80.
func WithInlineWidth(width int) InlineOption {
	return func(c *inlineConfig) {
		if width > 0 {
			c.width = width
		}
	}
}

// WithInlineOutput sets the output writer. Default is os.Stdout.
func WithInlineOutput(w io.Writer) InlineOption {
	return func(c *inlineConfig) {
		c.output = w
	}
}

// WithInlineInput sets the input reader. Default is os.Stdin.
// This is primarily used for testing.
func WithInlineInput(r io.Reader) InlineOption {
	return func(c *inlineConfig) {
		c.input = r
	}
}

// WithInlineFPS sets the frames per second for TickEvents.
// Default is 0 (no ticks). Set to > 0 to enable periodic TickEvents
// for animations.
func WithInlineFPS(fps int) InlineOption {
	return func(c *inlineConfig) {
		if fps >= 0 {
			c.fps = fps
		}
	}
}

// WithInlineMouseTracking enables mouse event tracking in the live region.
// When enabled, the application will receive MouseEvent events.
func WithInlineMouseTracking(enabled bool) InlineOption {
	return func(c *inlineConfig) {
		c.mouseTracking = enabled
	}
}

// WithInlineBracketedPaste enables bracketed paste mode.
// When enabled, pasted text is wrapped in escape sequences,
// allowing proper handling of multi-line pastes.
func WithInlineBracketedPaste(enabled bool) InlineOption {
	return func(c *inlineConfig) {
		c.bracketedPaste = enabled
	}
}

// WithInlinePasteTabWidth configures how tabs in pasted content are handled.
// If width is 0 (default), tabs are preserved as-is.
// If width > 0, each tab is converted to that many spaces.
func WithInlinePasteTabWidth(width int) InlineOption {
	return func(c *inlineConfig) {
		c.pasteTabWidth = width
	}
}

// WithInlineKittyKeyboard enables the Kitty keyboard protocol.
// This allows detection of modifier keys like Shift+Enter.
// For terminals that don't support it, a backslash+Enter fallback is used.
func WithInlineKittyKeyboard(enabled bool) InlineOption {
	return func(c *inlineConfig) {
		c.kittyKeyboard = enabled
	}
}

// InlineApp manages the runtime for inline terminal applications.
// It provides the same event-driven architecture as tui.Run() but renders
// inline rather than taking over the screen.
//
// Architecture:
//   - Goroutine 1: Main event loop (processes events sequentially, calls HandleEvent/LiveView)
//   - Goroutine 2: Input reader (blocks on stdin, sends KeyEvents)
//   - Goroutine 3: Command executor (runs async commands, sends results as events)
//
// This design eliminates race conditions in application code while maintaining
// responsive UI through non-blocking async operations.
type InlineApp struct {
	// Configuration
	config inlineConfig

	// Runtime state
	app    any
	events chan Event
	cmds   chan Cmd
	done   chan struct{}
	ticker *time.Ticker
	frame  uint64

	// Rendering
	live   *LivePrinter
	output io.Writer

	// Synchronization
	mu      sync.Mutex // Protects Print operations and running state
	running bool

	// Terminal state (for cleanup)
	oldState *term.State
	stdinFd  int

	// Mouse click synthesis state (same as Runtime)
	mousePressX      int
	mousePressY      int
	mousePressButton MouseButton
	mousePressed     bool
}

// NewInlineApp creates a new inline application runner.
//
// Example:
//
//	runner := tui.NewInlineApp(
//	    tui.WithInlineWidth(80),
//	    tui.WithInlineBracketedPaste(true),
//	)
//	if err := runner.Run(&MyApp{}); err != nil {
//	    log.Fatal(err)
//	}
func NewInlineApp(opts ...InlineOption) *InlineApp {
	cfg := defaultInlineConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &InlineApp{
		config: cfg,
		events: make(chan Event, 100),
		cmds:   make(chan Cmd, 100),
		done:   make(chan struct{}),
		output: cfg.output,
		live:   NewLivePrinter(WithWidth(cfg.width), WithOutput(cfg.output)),
	}
}

// Run starts the inline application and blocks until it exits.
// This is the main entry point, similar to tui.Run().
//
// The app parameter must implement InlineApplication (LiveView method).
// Optionally implement EventHandler, Initializable, Destroyable.
//
// Lifecycle:
//  1. Call Init() if implemented
//  2. Enter raw mode, enable configured terminal features
//  3. Start event loop, input reader, and command executor goroutines
//  4. Render initial LiveView()
//  5. Process events until Quit command
//  6. Restore terminal state
//  7. Call Destroy() if implemented
func (r *InlineApp) Run(app any) error {
	// Validate app implements required interface
	if _, ok := app.(InlineApplication); !ok {
		return fmt.Errorf("app must implement InlineApplication (LiveView())")
	}

	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("InlineApp is already running")
	}
	r.running = true
	r.app = app
	r.mu.Unlock()

	// Initialize application if it implements Initializable
	if init, ok := app.(Initializable); ok {
		if err := init.Init(); err != nil {
			r.mu.Lock()
			r.running = false
			r.mu.Unlock()
			return fmt.Errorf("application initialization failed: %w", err)
		}
	}

	// Check if stdin is a terminal
	r.stdinFd = int(os.Stdin.Fd())
	if !term.IsTerminal(r.stdinFd) {
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
		return fmt.Errorf("stdin is not a terminal")
	}

	// Enable raw mode
	var err error
	r.oldState, err = term.MakeRaw(r.stdinFd)
	if err != nil {
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
		return fmt.Errorf("failed to enable raw mode: %w", err)
	}

	// Enable terminal features
	if r.config.bracketedPaste {
		fmt.Fprint(r.output, "\033[?2004h")
	}
	if r.config.kittyKeyboard {
		fmt.Fprint(r.output, "\033[>1u")
	}
	if r.config.mouseTracking {
		// Enable mouse tracking (SGR mode for better coordinates)
		fmt.Fprint(r.output, "\033[?1000h\033[?1006h")
	}

	// Start ticker if FPS > 0
	if r.config.fps > 0 {
		r.ticker = time.NewTicker(time.Second / time.Duration(r.config.fps))
	}

	// Render initial view
	r.render()

	// Start the three goroutines
	var wg sync.WaitGroup
	wg.Add(3)

	// Goroutine 1: Main event loop
	go func() {
		defer wg.Done()
		r.eventLoop()
	}()

	// Goroutine 2: Input reader
	go func() {
		defer wg.Done()
		r.inputReader()
	}()

	// Goroutine 3: Command executor
	go func() {
		defer wg.Done()
		r.commandExecutor()
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// Cleanup
	r.cleanup()

	// Call Destroy if implemented
	if destroy, ok := app.(Destroyable); ok {
		destroy.Destroy()
	}

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()

	return nil
}

// cleanup restores terminal state
func (r *InlineApp) cleanup() {
	if r.ticker != nil {
		r.ticker.Stop()
	}

	// Stop live printer
	r.live.Stop()

	// Disable terminal features (reverse order)
	if r.config.mouseTracking {
		fmt.Fprint(r.output, "\033[?1006l\033[?1000l")
	}
	if r.config.kittyKeyboard {
		fmt.Fprint(r.output, "\033[<u")
	}
	if r.config.bracketedPaste {
		fmt.Fprint(r.output, "\033[?2004l")
	}

	// Restore terminal state
	if r.oldState != nil {
		term.Restore(r.stdinFd, r.oldState)
	}
}

// eventLoop is the main event processing loop (Goroutine 1).
// It processes events sequentially, ensuring no race conditions.
func (r *InlineApp) eventLoop() {
	for {
		select {
		case event := <-r.events:
			// Process this event and drain any other pending events
			if r.processEventWithQuitCheck(event) {
				close(r.done)
				return
			}

			// Drain all pending events before rendering
		drainLoop:
			for {
				select {
				case event := <-r.events:
					if r.processEventWithQuitCheck(event) {
						close(r.done)
						return
					}
				default:
					break drainLoop
				}
			}

			// Render once after processing all pending events
			r.render()

		case <-func() <-chan time.Time {
			if r.ticker != nil {
				return r.ticker.C
			}
			return nil
		}():
			// Send tick event for animations
			r.frame++
			tickEvent := TickEvent{
				Time:  time.Now(),
				Frame: r.frame,
			}
			r.processEvent(tickEvent)
			r.render()

		case <-r.done:
			return
		}
	}
}

// processEventWithQuitCheck processes an event and returns true if it's a quit event
func (r *InlineApp) processEventWithQuitCheck(event Event) bool {
	// Check for quit event
	if _, isQuit := event.(QuitEvent); isQuit {
		return true
	}

	// Handle batch events by unpacking them
	if batch, isBatch := event.(BatchEvent); isBatch {
		for _, e := range batch.Events {
			if _, isQuit := e.(QuitEvent); isQuit {
				return true
			}
			r.processEvent(e)
		}
	} else {
		r.processEvent(event)
	}

	return false
}

// processEvent calls the application's HandleEvent (if implemented) and queues any returned commands.
func (r *InlineApp) processEvent(event Event) {
	// Route events to interactive elements via unified focus manager
	switch e := event.(type) {
	case MouseEvent:
		if e.Type == MouseClick {
			focusManager.HandleClick(e.X, e.Y)
			interactiveRegistry.HandleClick(e.X, e.Y)
		}
	case KeyEvent:
		focusManager.HandleKey(e)
	}

	// Call user's event handler
	var cmds []Cmd
	if handler, ok := r.app.(EventHandler); ok {
		cmds = handler.HandleEvent(event)
	}

	// Queue commands for async execution
	if len(cmds) > 0 {
		for _, cmd := range cmds {
			select {
			case r.cmds <- cmd:
			case <-r.done:
				return
			}
		}
	}
}

// render calls the application's LiveView() and updates the live region.
func (r *InlineApp) render() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if app, ok := r.app.(InlineApplication); ok {
		// Clear registries before render
		focusManager.Clear()
		buttonRegistry.Clear()
		interactiveRegistry.Clear()
		inputRegistry.Clear()
		textAreaRegistry.Clear()

		view := app.LiveView()

		// Render the view using LivePrinter
		r.live.Update(view)

		// Prune TextArea state for IDs that weren't rendered
		textAreaRegistry.Prune()
	}
}

// inputReader reads keyboard and mouse events from stdin (Goroutine 2).
func (r *InlineApp) inputReader() {
	decoder := terminal.NewKeyDecoder(r.config.input)
	decoder.SetPasteTabWidth(r.config.pasteTabWidth)

	// Channel to receive stdin reads from a separate goroutine
	inputChan := make(chan Event, 1)
	errChan := make(chan error, 1)

	// Start a nested goroutine that continuously reads from stdin
	go func() {
		for {
			event, err := decoder.ReadEvent()
			if err != nil {
				select {
				case errChan <- err:
				case <-r.done:
					return
				}
				return
			}
			select {
			case inputChan <- event:
			case <-r.done:
				return
			}
		}
	}()

	const backslashEnterTimeout = 100 * time.Millisecond

	for {
		select {
		case <-r.done:
			return
		case event := <-inputChan:
			// Check for backslash key - might be start of backslash+Enter sequence
			if keyEvent, ok := event.(KeyEvent); ok && keyEvent.Rune == '\\' && keyEvent.Key == KeyUnknown {
				select {
				case <-r.done:
					return
				case nextEvent := <-inputChan:
					if nextKeyEvent, ok := nextEvent.(KeyEvent); ok && nextKeyEvent.Key == KeyEnter {
						event = KeyEvent{Key: KeyEnter, Shift: true}
					} else {
						r.forwardEvent(keyEvent)
						event = nextEvent
					}
				case <-time.After(backslashEnterTimeout):
					// Timeout - just a regular backslash
				}
			}

			// Process mouse events for click synthesis
			event, clickEvent := r.processMouseEvent(event)

			// Send Click BEFORE Release
			if clickEvent != nil {
				select {
				case r.events <- clickEvent:
				case <-r.done:
					return
				}
			}

			// Forward original event
			select {
			case r.events <- event:
			case <-r.done:
				return
			}

		case err := <-errChan:
			select {
			case r.events <- ErrorEvent{Time: time.Now(), Err: err, Cause: "input reader"}:
			case <-r.done:
				return
			}
			return
		}
	}
}

// forwardEvent sends an event to the main event loop.
func (r *InlineApp) forwardEvent(event Event) {
	select {
	case r.events <- event:
	case <-r.done:
	}
}

// processMouseEvent tracks mouse state and returns any additional synthetic events.
func (r *InlineApp) processMouseEvent(event Event) (Event, Event) {
	mouseEvent, ok := event.(MouseEvent)
	if !ok {
		return event, nil
	}

	switch mouseEvent.Type {
	case MousePress:
		r.mousePressX = mouseEvent.X
		r.mousePressY = mouseEvent.Y
		r.mousePressButton = mouseEvent.Button
		r.mousePressed = true
		return event, nil

	case MouseRelease:
		if r.mousePressed &&
			mouseEvent.X == r.mousePressX &&
			mouseEvent.Y == r.mousePressY {
			r.mousePressed = false
			clickEvent := MouseEvent{
				X:         mouseEvent.X,
				Y:         mouseEvent.Y,
				Button:    r.mousePressButton,
				Type:      MouseClick,
				Modifiers: mouseEvent.Modifiers,
				Time:      mouseEvent.Time,
			}
			return event, clickEvent
		}
		r.mousePressed = false
		return event, nil

	default:
		return event, nil
	}
}

// commandExecutor runs async commands (Goroutine 3).
func (r *InlineApp) commandExecutor() {
	for {
		select {
		case cmd := <-r.cmds:
			go func(c Cmd) {
				event := c()
				select {
				case r.events <- event:
				case <-r.done:
				}
			}(cmd)

		case <-r.done:
			return
		}
	}
}

// Print renders a View to the scrollback history.
// The view can be any tui.View: Text, Stack, Group, Bordered, etc.
//
// This method temporarily clears the live region, prints the view to scrollback,
// and restores the live region below.
//
// Thread-safe: Can be called from HandleEvent (recommended) or from
// a Cmd goroutine via the app reference.
func (r *InlineApp) Print(view View) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Wrap entire operation in synchronized output mode to prevent flicker.
	// This ensures clear + print + re-render appears as one atomic update.
	fmt.Fprint(r.output, "\033[?2026h") // Begin sync

	// Clear live region (moves cursor back)
	r.live.Clear()

	// Print to scrollback with raw mode line endings
	Fprint(r.output, view, WithWidth(r.config.width), WithRawMode())
	fmt.Fprint(r.output, "\r\n") // Add newline after printed content

	// Re-render live region (skip its internal sync since we're already in one)
	if app, ok := r.app.(InlineApplication); ok {
		r.live.UpdateNoSync(app.LiveView())
	}

	fmt.Fprint(r.output, "\033[?2026l") // End sync
}

// Printf is a convenience method that prints formatted text to scrollback.
// Equivalent to: r.Print(tui.Text(format, args...))
func (r *InlineApp) Printf(format string, args ...any) {
	r.Print(Text(format, args...))
}

// Stop gracefully stops the inline application by sending a QuitEvent.
// This can be called from any goroutine.
func (r *InlineApp) Stop() {
	select {
	case r.events <- QuitEvent{Time: time.Now()}:
	case <-r.done:
		// Already stopped
	}
}

// SendEvent sends an event to the application's event loop.
// This is useful for custom event sources or testing.
// It's safe to call from any goroutine.
func (r *InlineApp) SendEvent(event Event) {
	select {
	case r.events <- event:
	case <-r.done:
		// Runtime stopped, ignore event
	}
}

// ClearScrollback clears the terminal scrollback history.
// This sends the ANSI escape sequence to clear the scrollback buffer.
// Safe to call from HandleEvent.
func (r *InlineApp) ClearScrollback() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear live region first
	r.live.Clear()

	// Clear screen and scrollback
	// \033[3J clears scrollback, \033[2J clears screen, \033[H moves to top
	fmt.Fprint(r.output, "\033[3J\033[2J\033[H")

	// Re-render live region
	if app, ok := r.app.(InlineApplication); ok {
		r.live.Update(app.LiveView())
	}
}

// RunInline is a convenience function that creates and runs an InlineApp.
// Equivalent to: NewInlineApp(opts...).Run(app)
func RunInline(app any, opts ...InlineOption) error {
	return NewInlineApp(opts...).Run(app)
}
