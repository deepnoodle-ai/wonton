package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
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

// InlineAppConfig configures an InlineApp.
// All fields are optional with sensible zero-value defaults.
type InlineAppConfig struct {
	Width          int       // 0 = auto (terminal width or 80). Rendering width.
	Output         io.Writer // nil = os.Stdout. Where to write output.
	Input          io.Reader // nil = os.Stdin. Where to read input.
	FPS            int       // 0 = no ticks. Frames per second for TickEvents.
	MouseTracking  bool      // Enable mouse event tracking.
	BracketedPaste bool      // Enable bracketed paste mode.
	PasteTabWidth  int       // 0 = preserve tabs. Convert tabs to N spaces in pastes.
	KittyKeyboard  bool      // Enable Kitty keyboard protocol.
}

func (c InlineAppConfig) withDefaults() InlineAppConfig {
	if c.Width == 0 {
		c.Width = 80
		if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
			if w, _, err := term.GetSize(fd); err == nil && w > 0 {
				c.Width = w
			}
		}
	}
	if c.Output == nil {
		c.Output = os.Stdout
	}
	if c.Input == nil {
		c.Input = os.Stdin
	}
	return c
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
	config InlineAppConfig

	// Runtime state
	app    any
	events chan Event
	cmds   chan Cmd
	done   chan struct{}
	ticker *time.Ticker
	frame  uint64

	// Focus management
	focusMgr *FocusManager

	// Rendering
	live   *LivePrinter
	output io.Writer

	// Synchronization
	mu      sync.Mutex // Protects Print operations and running state
	running bool

	// Live region initialization (one-time spacing for pinned mode)
	liveInitialized bool

	// Terminal state (for cleanup)
	oldState *term.State
	stdinFd  int
	termRows int

	// Resize handling
	resizeChan chan os.Signal

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
//	runner := tui.NewInlineApp(tui.InlineAppConfig{
//	    Width:          80,
//	    BracketedPaste: true,
//	})
//	if err := runner.Run(&MyApp{}); err != nil {
//	    log.Fatal(err)
//	}
func NewInlineApp(cfgs ...InlineAppConfig) *InlineApp {
	cfg := InlineAppConfig{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	cfg = cfg.withDefaults()
	return &InlineApp{
		config:   cfg,
		events:   make(chan Event, 100),
		cmds:     make(chan Cmd, 100),
		done:     make(chan struct{}),
		output:   cfg.Output,
		live:     NewLivePrinter(PrintConfig{Width: cfg.Width, Output: cfg.Output}),
		focusMgr: NewFocusManager(),
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

	if width, height, err := term.GetSize(r.stdinFd); err == nil {
		r.mu.Lock()
		r.config.Width = width
		r.live.SetWidth(width)
		r.termRows = height
		r.mu.Unlock()
	}

	// Enable terminal features
	if r.config.BracketedPaste {
		fmt.Fprint(r.output, "\033[?2004h")
	}
	if r.config.KittyKeyboard {
		fmt.Fprint(r.output, "\033[>1u")
	}
	if r.config.MouseTracking {
		// Enable mouse tracking (SGR mode for better coordinates)
		fmt.Fprint(r.output, "\033[?1000h\033[?1006h")
	}

	// Start ticker if FPS > 0
	if r.config.FPS > 0 {
		r.ticker = time.NewTicker(time.Second / time.Duration(r.config.FPS))
	}

	// Initialize resize watcher (platform-specific)
	r.setupResizeWatcher()

	// Render initial view
	r.render()

	// Start the four goroutines
	var wg sync.WaitGroup
	wg.Add(4)

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

	// Goroutine 4: Resize listener
	go func() {
		defer wg.Done()
		r.resizeListener()
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

	// Stop resize listener (platform-specific)
	r.cleanupResizeWatcher()

	// Stop live printer
	r.live.Stop()

	// Disable terminal features (reverse order)
	if r.config.MouseTracking {
		fmt.Fprint(r.output, "\033[?1006l\033[?1000l")
	}
	if r.config.KittyKeyboard {
		fmt.Fprint(r.output, "\033[<u")
	}
	if r.config.BracketedPaste {
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
	// Handle resize events
	if resize, ok := event.(ResizeEvent); ok {
		r.mu.Lock()
		r.config.Width = resize.Width
		r.live.SetWidth(resize.Width)
		r.termRows = resize.Height
		r.mu.Unlock()
	}

	// Handle focus events from commands
	switch e := event.(type) {
	case FocusSetEvent:
		r.focusMgr.SetFocus(e.ID)
		return
	case FocusNextEvent:
		r.focusMgr.FocusNext()
		return
	case FocusPrevEvent:
		r.focusMgr.FocusPrev()
		return
	}

	// Route events to interactive elements via focus manager
	switch e := event.(type) {
	case MouseEvent:
		if e.Type == MouseClick {
			r.focusMgr.HandleClick(e.X, e.Y)
			interactiveRegistry.HandleClick(e.X, e.Y)
		}
	case KeyEvent:
		r.focusMgr.HandleKey(e)
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
	r.renderNoLock()
}

// renderNoLock is the internal version of render that assumes r.mu is already held.
func (r *InlineApp) renderNoLock() {
	if app, ok := r.app.(InlineApplication); ok {
		// Clear registries before render
		r.focusMgr.Clear()
		buttonRegistry.Clear()
		interactiveRegistry.Clear()
		inputRegistry.Clear()
		textAreaRegistry.Clear()

		view := app.LiveView()
		_, viewHeight := view.size(r.config.Width, 0)
		if viewHeight == 0 {
			viewHeight = 1
		}
		if r.termRows > 0 && viewHeight > r.termRows {
			viewHeight = r.termRows
		}
		if !r.liveInitialized && r.termRows > 0 {
			fmt.Fprint(r.output, strings.Repeat("\r\n", viewHeight))
			r.liveInitialized = true
		}

		// Render the view using LivePrinter with focus manager
		r.live.UpdatePinnedWithFocus(view, r.focusMgr, r.termRows)

		// Prune TextArea state for IDs that weren't rendered
		textAreaRegistry.Prune()
	}
}

// inputReader reads keyboard and mouse events from stdin (Goroutine 2).
func (r *InlineApp) inputReader() {
	decoder := terminal.NewKeyDecoder(r.config.Input)
	decoder.SetPasteTabWidth(r.config.PasteTabWidth)

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

// resizeListener watches for terminal resize signals (Goroutine 4).
func (r *InlineApp) resizeListener() {
	for {
		select {
		case <-r.resizeChan:
			// Get new size
			width, height, err := term.GetSize(r.stdinFd)
			if err == nil {
				select {
				case r.events <- ResizeEvent{Time: time.Now(), Width: width, Height: height}:
				case <-r.done:
					return
				}
			}
		case <-r.done:
			return
		}
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
	fmt.Fprint(r.output, "\033[?2026h") // Begin sync

	liveHeight := r.live.LastHeight()

	// 1. Clear the current live region to prevent ghosting
	if r.termRows > 0 && liveHeight > 0 {
		// Pinned mode: Clear using absolute positioning
		startRow := r.termRows - liveHeight + 1
		if startRow < 1 {
			startRow = 1
		}
		for i := 0; i < liveHeight; i++ {
			fmt.Fprintf(r.output, "\033[%d;1H\033[2K", startRow+i)
		}
		// Move cursor to the start of where the live region was
		fmt.Fprintf(r.output, "\033[%d;1H", startRow)
	} else {
		// Standard mode: Clear using LivePrinter's logic (moves up and clears)
		r.live.Clear()
	}

	// 2. Print the new content
	// This uses natural terminal behavior. If the content hits the bottom
	// of the screen, the terminal will scroll the history up automatically.
	// We use RawMode=true for proper line endings (\r\n).
	Fprint(r.output, view, PrintConfig{Width: r.config.Width, RawMode: true})
	fmt.Fprint(r.output, "\r\n") // Ensure we end on a new line

	// 3. Re-render the live region at the bottom
	// If scrolling happened, the live region needs to be drawn at the new bottom.
	// renderNoLock handles UpdatePinned vs Update automatically based on config.
	r.renderNoLock()

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
	r.liveInitialized = false
	r.renderNoLock()
}

// RunInline is a convenience function that creates and runs an InlineApp.
// You can configure it by passing a configuration function that receives the app.
//
// Example:
//
//	err := tui.RunInline(&MyApp{}, func(app *tui.InlineApp) {
//	    app.Width(80).BracketedPaste(true)
//	})
func RunInline(app any, cfg func(*InlineApp)) error {
	runner := NewInlineApp()
	if cfg != nil {
		cfg(runner)
	}
	return runner.Run(app)
}
