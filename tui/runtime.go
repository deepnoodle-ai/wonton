package tui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// Application is the main interface for declarative UI applications.
// Implementations provide a View that describes the current UI state.
//
// Thread Safety: View and HandleEvent (if implemented) are NEVER called concurrently.
// The Runtime ensures these methods run sequentially in a single goroutine,
// eliminating the need for locks in application code.
type Application interface {
	// View returns the declarative view tree representing the current UI.
	// This is called automatically after each event is processed.
	View() View
}

// EventHandler is an optional interface that applications can implement
// to handle events and trigger async operations.
type EventHandler interface {
	// HandleEvent processes an event and optionally returns commands for async execution.
	// This is called in a single-threaded event loop, so state can be mutated freely
	// without locks. Return commands for async operations (HTTP requests, timers, etc.).
	HandleEvent(event Event) []Cmd
}

// LegacyApplication is the deprecated interface for imperative rendering.
// New applications should implement Application with View() instead.
//
// Deprecated: Use Application with View() for declarative rendering.
type LegacyApplication interface {
	HandleEvent(event Event) []Cmd
	Render(frame RenderFrame)
}

// Initializable is an optional interface that applications can implement
// to perform initialization before the event loop starts.
type Initializable interface {
	Init() error
}

// Destroyable is an optional interface that applications can implement
// to perform cleanup when the runtime stops.
type Destroyable interface {
	Destroy()
}

// Runtime manages the event-driven execution of an Application.
// It provides a race-free, single-threaded event loop while supporting
// async operations through the command system.
//
// Architecture:
//   - Goroutine 1: Main event loop (processes events sequentially, calls HandleEvent/View)
//   - Goroutine 2: Input reader (blocks on stdin, sends KeyEvents)
//   - Goroutine 3: Command executor (runs async commands, sends results as events)
//
// This design eliminates race conditions in application code while maintaining
// responsive UI through non-blocking async operations.
type Runtime struct {
	terminal *Terminal
	app      any // Application or LegacyApplication
	events   chan Event
	cmds     chan Cmd
	done     chan struct{}
	ticker   *time.Ticker
	fps      int
	frame    uint64 // Frame counter for TickEvents

	mu          sync.Mutex
	running     bool
	resizeUnsub func() // Unsubscribe function for resize callback

	// Paste handling configuration
	pasteTabWidth int // 0 = preserve tabs, >0 = convert to this many spaces

	// Mouse click synthesis state
	mousePressX      int         // X position of last mouse press
	mousePressY      int         // Y position of last mouse press
	mousePressButton MouseButton // Button that was pressed
	mousePressed     bool        // Whether a mouse button is currently pressed
}

// NewRuntime creates a new Runtime for the given application.
//
// Parameters:
//   - terminal: The Terminal instance to use for rendering and input
//   - app: The Application (declarative) or LegacyApplication (imperative) to run
//   - fps: Frames per second for TickEvents (30 recommended, 60 for smooth animations)
//
// The runtime does not start automatically. Call Run() to start the event loop.
func NewRuntime(terminal *Terminal, app any, fps int) *Runtime {
	if fps <= 0 {
		fps = 30 // Default to 30 FPS
	}

	return &Runtime{
		terminal:      terminal,
		app:           app,
		events:        make(chan Event, 100), // Buffered to prevent blocking
		cmds:          make(chan Cmd, 100),
		done:          make(chan struct{}),
		fps:           fps,
		frame:         0,
		pasteTabWidth: 0, // Default: preserve tabs
	}
}

// SetPasteTabWidth configures how tabs in pasted content are handled.
// If width is 0 (default), tabs are preserved as-is.
// If width > 0, each tab is converted to that many spaces.
// Must be called before Run().
func (r *Runtime) SetPasteTabWidth(width int) {
	r.pasteTabWidth = width
}

// Run starts the runtime's event loop and blocks until the application quits.
// This method is the main entry point for message-driven applications.
//
// Execution flow:
//  1. Initialize application (if it implements Initializable)
//  2. Register resize handler with terminal
//  3. Start three goroutines: event loop, input reader, command executor
//  4. Block until QuitEvent is received
//  5. Clean up and call Destroy (if implemented)
//
// Returns error if initialization fails.
func (r *Runtime) Run() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("runtime is already running")
	}
	r.running = true
	r.mu.Unlock()

	// Initialize application if it implements Initializable
	if init, ok := r.app.(Initializable); ok {
		if err := init.Init(); err != nil {
			return fmt.Errorf("application initialization failed: %w", err)
		}
	}

	// Enable raw mode for character-by-character input
	// Only enable if stdin is actually a terminal (not piped or redirected)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		// Detect Kitty keyboard protocol support before enabling raw mode
		// This probes the terminal and enables the protocol if supported
		r.terminal.DetectKittyProtocol()

		if err := r.terminal.EnableRawMode(); err != nil {
			return fmt.Errorf("failed to enable raw mode: %w", err)
		}

		// Enable Kitty keyboard protocol if the terminal supports it
		// This allows detection of modifier keys (Shift+Enter, etc.)
		// For terminals that don't support it, backslash+Enter fallback is used
		if r.terminal.IsKittyProtocolSupported() {
			r.terminal.EnableEnhancedKeyboard()
		}
	}

	// Register resize handler
	r.resizeUnsub = r.terminal.OnResize(func(width, height int) {
		// Send resize event to event loop
		r.events <- ResizeEvent{
			Time:   time.Now(),
			Width:  width,
			Height: height,
		}
	})

	// Start watching for resize signals
	r.terminal.WatchResize()

	// Send initial resize event with current terminal size
	width, height := r.terminal.Size()
	r.events <- ResizeEvent{
		Time:   time.Now(),
		Width:  width,
		Height: height,
	}

	// Start ticker for animation frames
	r.ticker = time.NewTicker(time.Second / time.Duration(r.fps))

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
	r.ticker.Stop()
	r.terminal.StopWatchResize()
	if r.resizeUnsub != nil {
		r.resizeUnsub()
	}
	if r.terminal.IsKittyProtocolEnabled() {
		r.terminal.DisableEnhancedKeyboard()
	}
	r.terminal.DisableRawMode()

	// Call Destroy if implemented
	if destroy, ok := r.app.(Destroyable); ok {
		destroy.Destroy()
	}

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()

	return nil
}

// Stop gracefully stops the runtime by sending a QuitEvent.
// This can be called from application code or externally.
func (r *Runtime) Stop() {
	select {
	case r.events <- QuitEvent{Time: time.Now()}:
	case <-r.done:
		// Already stopped
	}
}

// eventLoop is the main event processing loop (Goroutine 1).
// It processes events sequentially, ensuring no race conditions.
// Events are batched: all pending events are processed before rendering once.
func (r *Runtime) eventLoop() {
	for {
		select {
		case event := <-r.events:
			// Process this event and drain any other pending events
			if r.processEventWithQuitCheck(event) {
				close(r.done)
				return
			}

			// Drain all pending events before rendering
			// This prevents slow rendering from causing event backlog
		drainLoop:
			for {
				select {
				case event := <-r.events:
					if r.processEventWithQuitCheck(event) {
						close(r.done)
						return
					}
				default:
					// No more pending events
					break drainLoop
				}
			}

			// Render once after processing all pending events
			r.render()

		case <-r.ticker.C:
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
func (r *Runtime) processEventWithQuitCheck(event Event) bool {
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
func (r *Runtime) processEvent(event Event) {
	// Route events to interactive elements via unified focus manager
	switch e := event.(type) {
	case MouseEvent:
		if e.Type == MouseClick {
			// Check if the click hit a focusable element
			focusManager.HandleClick(e.X, e.Y)
			// Check if the click hit a non-focusable interactive region
			interactiveRegistry.HandleClick(e.X, e.Y)
		}
	case KeyEvent:
		// Route key events to focused element (handles Tab/Shift+Tab navigation)
		if focusManager.HandleKey(e) {
			// Focused element consumed the event, but still pass to app
		}
	}

	// Call user's event handler
	// Try EventHandler first (new interface), then LegacyApplication (deprecated)
	var cmds []Cmd
	if handler, ok := r.app.(EventHandler); ok {
		cmds = handler.HandleEvent(event)
	} else if legacy, ok := r.app.(LegacyApplication); ok {
		cmds = legacy.HandleEvent(event)
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

// ViewProvider is an alias for Application for backward compatibility.
//
// Deprecated: Use Application directly instead.
type ViewProvider = Application

// render calls the application's View() or Render() method using BeginFrame/EndFrame.
// Application uses declarative View() rendering, LegacyApplication uses imperative Render().
func (r *Runtime) render() {
	frame, err := r.terminal.BeginFrame()
	if err != nil {
		// Terminal not ready, skip this frame
		return
	}

	// Check for LegacyApplication first (imperative rendering)
	if legacy, ok := r.app.(LegacyApplication); ok {
		legacy.Render(frame)
	} else if app, ok := r.app.(Application); ok {
		// Application interface - use declarative View() rendering
		// Clear registries before render (they get repopulated during render)
		focusManager.Clear()
		buttonRegistry.Clear()
		interactiveRegistry.Clear()
		inputRegistry.Clear()

		// Clear the frame before rendering. This ensures that when views shrink,
		// old content outside their new bounds is erased. The double-buffering
		// system ensures only actual changes are sent to the terminal.
		frame.Fill(' ', NewStyle())

		view := app.View()
		width, height := frame.Size()

		// Create render context with frame counter for animations
		ctx := NewRenderContext(frame, r.frame)

		// Measure phase (populates cached child sizes)
		view.size(width, height)
		// Render phase
		view.render(ctx)
	}

	// Flush to screen (diffs and sends only dirty regions)
	r.terminal.EndFrame(frame)
}

// inputReader reads keyboard and mouse events from stdin (Goroutine 2).
// This goroutine blocks on stdin reads and forwards events to the main loop.
// When mouse tracking is enabled (via terminal.EnableMouseTracking()),
// this reader will automatically decode and forward mouse events as well.
//
// IMPORTANT: This uses a nested goroutine pattern to allow clean shutdown.
// The problem: decoder.ReadEvent() is a blocking call on stdin. If we called it
// directly in a select statement's default case, we'd be stuck waiting for input
// even after r.done is closed, preventing the program from exiting cleanly.
// This manifested as the terminal not clearing until another key was pressed.
//
// The solution: A nested goroutine continuously reads from stdin and sends events
// to a channel. The main loop uses select to monitor both this channel and r.done,
// allowing immediate exit when r.done is closed. The nested goroutine may remain
// blocked on stdin, but that's okay - the parent goroutine exits, allowing
// wg.Wait() in Run() to proceed and the program to clean up and exit immediately.
func (r *Runtime) inputReader() {
	decoder := NewKeyDecoder(os.Stdin)
	decoder.SetPasteTabWidth(r.pasteTabWidth)

	// Channel to receive stdin reads from a separate goroutine
	inputChan := make(chan Event, 1)
	errChan := make(chan error, 1)

	// Start a nested goroutine that continuously reads from stdin
	// This goroutine may remain blocked on stdin after shutdown, which is acceptable
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

	// Timeout for backslash+Enter detection
	// Gemini CLI uses 5ms but Go's scheduler may need more time
	const backslashEnterTimeout = 100 * time.Millisecond

	// Main loop that can be interrupted by r.done
	// This pattern ensures we can exit immediately when quit is requested,
	// rather than waiting for the next stdin input
	for {
		select {
		case <-r.done:
			return
		case event := <-inputChan:
			// Check for backslash key - might be start of backslash+Enter sequence
			// This provides a terminal-agnostic way to input Shift+Enter
			if keyEvent, ok := event.(KeyEvent); ok && keyEvent.Rune == '\\' && keyEvent.Key == KeyUnknown {
				// Wait briefly for Enter to follow
				select {
				case <-r.done:
					return
				case nextEvent := <-inputChan:
					// Check if Enter followed the backslash
					if nextKeyEvent, ok := nextEvent.(KeyEvent); ok && nextKeyEvent.Key == KeyEnter {
						// Convert backslash+Enter to Shift+Enter
						event = KeyEvent{Key: KeyEnter, Shift: true}
					} else {
						// Not Enter - forward backslash first, then process nextEvent
						r.forwardEvent(keyEvent)
						event = nextEvent
					}
				case <-time.After(backslashEnterTimeout):
					// Timeout - just a regular backslash, forward it
					// (event is already the backslash, will be forwarded below)
				}
			}
			// Process mouse events to synthesize clicks from Press/Release pairs.
			// See processMouseEvent for detailed documentation on click synthesis.
			event, clickEvent := r.processMouseEvent(event)

			// IMPORTANT: Send Click BEFORE Release
			// This ordering ensures MouseHandler.HandleEvent sees the Click first,
			// sets clickSynthesized=true, and then skips synthesis in handleRelease.
			// Without this ordering, handleRelease would create a duplicate click.
			if clickEvent != nil {
				select {
				case r.events <- clickEvent:
				case <-r.done:
					return
				}
			}

			// Forward original event (Press, Release, Move, etc.)
			select {
			case r.events <- event:
			case <-r.done:
				return
			}
		case err := <-errChan:
			// EOF or error - send error event
			select {
			case r.events <- ErrorEvent{
				Time:  time.Now(),
				Err:   err,
				Cause: "input reader",
			}:
			case <-r.done:
				return
			}
			return
		}
	}
}

// forwardEvent sends an event to the main event loop.
// Used by inputReader when it needs to send multiple events (e.g., backslash followed by non-Enter).
func (r *Runtime) forwardEvent(event Event) {
	select {
	case r.events <- event:
	case <-r.done:
	}
}

// processMouseEvent tracks mouse state and returns any additional synthetic events.
//
// # Mouse Click Synthesis
//
// Terminal mouse input only provides raw Press and Release events. To provide a
// convenient "click" abstraction, the Runtime synthesizes MouseClick events when
// a press and release occur at the same location.
//
// Applications receive events in this order for a click:
//  1. MousePress - button went down
//  2. MouseClick - synthetic click (same location as press)
//  3. MouseRelease - button came up
//
// The Click is sent BEFORE Release so that MouseHandler (if used) can detect that
// a click was already synthesized and skip its own click synthesis in handleRelease.
//
// Applications can handle whichever events they need:
//   - Most apps just handle MouseClick for simple button behavior
//   - Apps needing drag or press feedback can also handle MousePress/MouseRelease
//
// Returns the original event and an optional synthetic click event.
func (r *Runtime) processMouseEvent(event Event) (Event, Event) {
	mouseEvent, ok := event.(MouseEvent)
	if !ok {
		return event, nil
	}

	switch mouseEvent.Type {
	case MousePress:
		// Track the press location and button
		r.mousePressX = mouseEvent.X
		r.mousePressY = mouseEvent.Y
		r.mousePressButton = mouseEvent.Button
		r.mousePressed = true
		return event, nil

	case MouseRelease:
		// Check if release is at the same location as press
		if r.mousePressed &&
			mouseEvent.X == r.mousePressX &&
			mouseEvent.Y == r.mousePressY {
			// Return both the release AND a synthetic click
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
// Each command runs in its own goroutine and sends its result back as an event.
func (r *Runtime) commandExecutor() {
	for {
		select {
		case cmd := <-r.cmds:
			// Execute command in a new goroutine
			go func(c Cmd) {
				// Execute the command (may take time)
				event := c()

				// Send result back to main event loop
				select {
				case r.events <- event:
				case <-r.done:
					// Runtime stopped, ignore result
				}
			}(cmd)

		case <-r.done:
			return
		}
	}
}

// SendEvent sends an event to the runtime's event loop.
// This is useful for custom event sources or testing.
// It's safe to call from any goroutine.
func (r *Runtime) SendEvent(event Event) {
	select {
	case r.events <- event:
	case <-r.done:
		// Runtime stopped, ignore event
	}
}
