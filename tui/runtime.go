package tui

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"golang.org/x/term"
)

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
	terminal   *Terminal
	app        any // Application
	events     chan Event
	cmds       chan Cmd
	done       chan struct{}
	doneOnce   sync.Once
	ticker     *time.Ticker
	fps        int
	lastRender time.Time // when render() last ran, for throttling resize repaints
	frame      uint64    // Frame counter for TickEvents

	// Panic capture: a panic in any runtime-managed goroutine (event loop,
	// input reader, command goroutines) is recorded here so Run can restore
	// the terminal before re-panicking with the original stack trace.
	panicMu    sync.Mutex
	panicVal   any
	panicStack []byte
	runExited  bool

	// Focus management
	focusMgr *FocusManager

	// Per-runtime view state stores (buttons, inputs, clickable regions, ...)
	reg *registries

	mu          sync.Mutex
	running     bool
	resizeUnsub func() // Unsubscribe function for resize callback

	// Input configuration
	inputSource    InputSource // Source of input events (defaults to stdin decoder)
	pasteTabWidth  int         // 0 = preserve tabs, >0 = convert to this many spaces
	backslashEnter bool        // synthesize Shift+Enter from backslash+Enter (opt-in)
	kittyKeyboard  bool        // enable the Kitty keyboard protocol without probing

	// Mouse click synthesis state
	mousePressX      int         // X position of last mouse press
	mousePressY      int         // Y position of last mouse press
	mousePressButton MouseButton // Button that was pressed
	mousePressed     bool        // Whether a mouse button is currently pressed

	// Click-count state: repeated clicks on the same cell with the same button
	// inside doubleClickThreshold count up, so views get double- and
	// triple-click for free rather than each keeping its own timer.
	lastClickTime   time.Time
	lastClickX      int
	lastClickY      int
	lastClickButton MouseButton
	clickCount      int

	// Suspend state. suspendKeys is non-nil exactly while Suspend is running
	// fn; the input reader diverts events into it instead of the event loop.
	suspendMu   sync.Mutex
	suspendKeys chan Event
}

// NewRuntime creates a new Runtime for the given application.
//
// Parameters:
//   - terminal: The Terminal instance to use for rendering and input
//   - app: The Application to run
//   - fps: Frames per second for TickEvents (30 recommended, 60 for smooth animations)
//
// The runtime does not start automatically. Call Run() to start the event loop.
func NewRuntime(terminal *Terminal, app Application, fps int) *Runtime {
	if fps <= 0 {
		fps = 30 // Default to 30 FPS
	}

	r := &Runtime{
		terminal:      terminal,
		app:           app,
		events:        make(chan Event, 100), // Buffered to prevent blocking
		cmds:          make(chan Cmd, 100),
		done:          make(chan struct{}),
		fps:           fps,
		frame:         0,
		pasteTabWidth: 0, // Default: preserve tabs
		focusMgr:      NewFocusManager(),
		reg:           newRegistries(),
	}
	// Before Init, so an application can reach Suspend and the terminal from
	// the first thing it does.
	if aware, ok := app.(RuntimeAware); ok {
		aware.SetRuntime(r)
	}
	return r
}

// Terminal returns the terminal the runtime is driving. An application reaches
// it through RuntimeAware; Run builds it and would otherwise keep it private.
func (r *Runtime) Terminal() *Terminal { return r.terminal }

// SetPasteTabWidth configures how tabs in pasted content are handled.
// If width is 0 (default), tabs are preserved as-is.
// If width > 0, each tab is converted to that many spaces.
// Must be called before Run().
func (r *Runtime) SetPasteTabWidth(width int) {
	r.pasteTabWidth = width
}

// SetKittyKeyboard asks for the Kitty keyboard protocol outright, instead of
// probing the terminal for it. Terminals that do not implement the protocol
// ignore the escape sequence, and the runtime turns it off again on the way
// out either way.
//
// Use it when the probe's answer would be wrong or too expensive: it is skipped
// under tmux and screen (TERM=screen*/tmux*) and in Apple Terminal, so a
// multiplexed session loses Shift+Enter even when the outer terminal supports
// it, and where it does run it costs up to 200 ms of startup. This is what
// InlineApp's WithInlineKittyKeyboard does. Must be called before Run().
func (r *Runtime) SetKittyKeyboard(enabled bool) {
	r.kittyKeyboard = enabled
}

// SetBackslashEnter enables synthesizing Shift+Enter from a backslash
// immediately followed by Enter. This is a fallback for terminals without
// the Kitty keyboard protocol, useful for chat-style apps where Shift+Enter
// inserts a newline.
//
// Disabled by default: when enabled, every typed backslash is delayed
// briefly while the runtime waits to see if Enter follows, and a backslash
// followed quickly by Enter is rewritten to Shift+Enter (losing the
// backslash). Only enable this for applications that need the Shift+Enter
// affordance. Must be called before Run().
func (r *Runtime) SetBackslashEnter(enabled bool) {
	r.backslashEnter = enabled
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
		// Detect Kitty keyboard protocol support before enabling raw mode.
		// SetKittyKeyboard skips the probe: the app has already decided, and
		// the probe costs up to 200 ms of startup.
		if !r.kittyKeyboard {
			r.terminal.DetectKittyProtocol()
		}

		if err := r.terminal.EnableRawMode(); err != nil {
			return fmt.Errorf("failed to enable raw mode: %w", err)
		}

		// Enable Kitty keyboard protocol if the terminal supports it
		// This allows detection of modifier keys (Shift+Enter, etc.)
		// For terminals that don't support it, backslash+Enter fallback is used
		if r.kittyKeyboard || r.terminal.IsKittyProtocolSupported() {
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
		defer r.capturePanic()
		r.eventLoop()
	}()

	// Goroutine 2: Input reader
	go func() {
		defer wg.Done()
		defer r.capturePanic()
		r.inputReader()
	}()

	// Goroutine 3: Command executor
	go func() {
		defer wg.Done()
		defer r.capturePanic()
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

	// If a runtime goroutine panicked, re-panic now that the terminal has been
	// restored (and run.go's deferred cleanup will complete during unwinding).
	// The original stack is embedded in the message since the panic crossed
	// goroutines.
	r.panicMu.Lock()
	pv, ps := r.panicVal, r.panicStack
	r.runExited = true
	r.panicMu.Unlock()
	if pv != nil {
		panic(fmt.Sprintf("tui: application panic: %v\n\noriginal stack:\n%s", pv, ps))
	}

	return nil
}

// closeDone signals shutdown to all runtime goroutines. Safe to call multiple
// times (quit path and panic path can both reach it).
func (r *Runtime) closeDone() {
	r.doneOnce.Do(func() { close(r.done) })
}

// capturePanic recovers a panic from a runtime-managed goroutine, records it,
// and triggers shutdown. Run re-panics with the original stack after the
// terminal has been restored, so the trace is readable instead of being
// printed to a raw-mode alternate screen.
//
// Must be invoked via defer.
func (r *Runtime) capturePanic() {
	rec := recover()
	if rec == nil {
		return
	}
	stack := debug.Stack()
	r.panicMu.Lock()
	if r.runExited {
		// Run has already returned; nobody will re-deliver this panic.
		// Crash here to preserve fail-fast semantics (terminal is restored).
		r.panicMu.Unlock()
		panic(rec)
	}
	if r.panicVal == nil {
		r.panicVal = rec
		r.panicStack = stack
	}
	r.panicMu.Unlock()
	r.closeDone()
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
			_, resizeOnly := event.(ResizeEvent)
			if r.processEventWithQuitCheck(event) {
				r.closeDone()
				return
			}

			// Drain all pending events before rendering
			// This prevents slow rendering from causing event backlog
		drainLoop:
			for {
				select {
				case event := <-r.events:
					if _, isResize := event.(ResizeEvent); !isResize {
						resizeOnly = false
					}
					if r.processEventWithQuitCheck(event) {
						r.closeDone()
						return
					}
				default:
					// No more pending events
					break drainLoop
				}
			}

			// Dragging a window edge delivers SIGWINCH far faster than the
			// frame rate, and every resize repaints each cell on the screen
			// because the size change invalidates the whole front buffer.
			// Repainting per signal floods the terminal and the drag visibly
			// stutters, so let the ticker pick these up instead. Only resize
			// is throttled: a keystroke has to feel immediate.
			if r.shouldThrottleResize(resizeOnly, time.Now()) {
				continue
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
			// Check if the click hit a focusable element
			r.focusMgr.HandleClick(e.X, e.Y)
			// Check if the click hit a non-focusable interactive region
			r.reg.interactive.HandleClick(e.X, e.Y)
		}
	case KeyEvent:
		// Route key events to the focused element first (this also handles
		// Tab/Shift+Tab navigation). A consumed key is NOT delivered to the
		// app, so typing into a focused input can't trigger app shortcuts
		// like 'q' to quit. Ctrl+C is always delivered so apps can implement
		// a reliable quit shortcut regardless of focus.
		if r.focusMgr.HandleKey(e) && !isInterruptKey(e) {
			return
		}
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

// render calls the application's View() method using BeginFrame/EndFrame.
// shouldThrottleResize reports whether a batch of nothing but resize events
// should wait for the ticker rather than repaint now. Resizes arrive faster
// than the frame rate during a drag and each one repaints the whole screen;
// the ticker renders within a frame either way.
func (r *Runtime) shouldThrottleResize(resizeOnly bool, now time.Time) bool {
	return resizeOnly && now.Sub(r.lastRender) < r.frameInterval()
}

// frameInterval is how long one frame lasts at the runtime's frame rate.
func (r *Runtime) frameInterval() time.Duration {
	if r.fps <= 0 {
		return 0
	}
	return time.Second / time.Duration(r.fps)
}

func (r *Runtime) render() {
	r.lastRender = time.Now()
	frame, err := r.terminal.BeginFrame()
	if err != nil {
		// Terminal not ready, skip this frame
		return
	}
	// Deferred so a panic in View()/render releases the terminal's frame lock
	// (BeginFrame holds it until EndFrame); otherwise cleanup would deadlock.
	// Flush to screen (diffs and sends only dirty regions).
	defer r.terminal.EndFrame(frame)

	if app, ok := r.app.(Application); ok {
		// Application interface - use declarative View() rendering
		// Clear registries before render (they get repopulated during render)
		r.focusMgr.Clear()
		r.reg.clearForRender()

		// Clear the frame before rendering. This ensures that when views shrink,
		// old content outside their new bounds is erased. The double-buffering
		// system ensures only actual changes are sent to the terminal.
		frame.Fill(' ', NewStyle())

		// Build the view tree with this runtime's registries as the capture
		// target for stateful view constructors (InputField, TextArea, ...).
		view := buildViews(r.reg, app.View)
		width, height := frame.Size()

		// Create render context with frame counter and focus manager for animations
		ctx := NewRenderContext(frame, r.frame).WithFocusManager(r.focusMgr).withRegistries(r.reg)

		// Measure phase (populates cached child sizes)
		view.size(width, height)
		// Render phase
		view.render(ctx)

		// Prune TextArea state for IDs that weren't rendered this frame
		r.reg.textAreas.Prune()
	}
}

// SetInputSource sets the input source for the runtime.
func (r *Runtime) SetInputSource(source InputSource) {
	r.inputSource = source
}

// defaultInputSource wraps KeyDecoder to satisfy InputSource
type defaultInputSource struct {
	decoder *KeyDecoder
}

func (d *defaultInputSource) ReadEvent() (Event, error) {
	return d.decoder.ReadEvent()
}

func (d *defaultInputSource) SetPasteTabWidth(w int) {
	d.decoder.SetPasteTabWidth(w)
}

// inputReader reads keyboard and mouse events from stdin (Goroutine 2).
// This goroutine blocks on stdin reads and forwards events to the main loop.
//
// Goroutine Lifecycle Note: The nested goroutine that calls ReadEvent() blocks
// on stdin and cannot be interrupted when the runtime stops. This is an inherent
// limitation of blocking I/O in Go. For typical TUI applications (single runtime
// for process lifetime), this is not an issue since all goroutines are cleaned
// up on process exit. However, if creating/destroying runtimes repeatedly (tests,
// embedded use), be aware that stopped runtimes may leave a blocked goroutine
// until the next stdin input or process exit.
func (r *Runtime) inputReader() {
	var source InputSource
	if r.inputSource != nil {
		source = r.inputSource
	} else {
		// Default to stdin decoder
		decoder := NewKeyDecoder(os.Stdin)
		decoder.SetPasteTabWidth(r.pasteTabWidth)
		source = &defaultInputSource{decoder: decoder}
	}

	// Channel to receive stdin reads from a separate goroutine
	inputChan := make(chan Event, 1)
	errChan := make(chan error, 1)

	// Start a nested goroutine that continuously reads from source
	go func() {
		defer r.capturePanic()
		for {
			event, err := source.ReadEvent()
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
			// This provides a terminal-agnostic way to input Shift+Enter.
			// Opt-in via WithBackslashEnter: intercepting backslashes delays
			// them and can rewrite legitimate input (backslash then Enter).
			if keyEvent, ok := event.(KeyEvent); ok && r.backslashEnter && keyEvent.Rune == '\\' && keyEvent.Key == KeyUnknown {
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
						_ = r.forwardEvent(keyEvent)
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
				if !r.forwardEvent(clickEvent) {
					return
				}
			}

			// Forward original event (Press, Release, Move, etc.)
			if !r.forwardEvent(event) {
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
// forwardEvent hands an input event to the event loop, or to a Suspend in
// progress. Returns false once the runtime is shutting down, at which point the
// caller should stop reading input.
func (r *Runtime) forwardEvent(event Event) bool {
	if keys := r.suspendChannel(); keys != nil {
		// The application is not running: fn owns the screen and any keys the
		// user presses belong to it. A full channel means fn is not reading, so
		// drop rather than block the decoder.
		select {
		case keys <- event:
		default:
		}
		return true
	}
	select {
	case r.events <- event:
		return true
	case <-r.done:
		return false
	}
}

// suspendChannel returns the channel a Suspend is waiting on, or nil when the
// runtime is not suspended.
func (r *Runtime) suspendChannel() chan Event {
	r.suspendMu.Lock()
	defer r.suspendMu.Unlock()
	return r.suspendKeys
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
// The synthetic click carries a ClickCount: repeated clicks on the same cell
// with the same button, each within doubleClickThreshold of the last, count up
// (1 = single, 2 = double, 3 = triple, and on). Any other click restarts at 1.
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
				X:          mouseEvent.X,
				Y:          mouseEvent.Y,
				Button:     r.mousePressButton,
				Type:       MouseClick,
				Modifiers:  mouseEvent.Modifiers,
				Time:       mouseEvent.Time,
				ClickCount: r.countClick(mouseEvent),
			}
			return event, clickEvent
		}
		r.mousePressed = false
		return event, nil

	default:
		return event, nil
	}
}

// doubleClickThreshold is how long a second click may follow the first and
// still count as part of the same multi-click. Matches MouseHandler.
const doubleClickThreshold = 500 * time.Millisecond

// countClick advances the multi-click counter for a click at ev's position and
// returns the resulting count.
func (r *Runtime) countClick(ev MouseEvent) int {
	sameSpot := ev.X == r.lastClickX && ev.Y == r.lastClickY &&
		r.mousePressButton == r.lastClickButton
	if sameSpot && !r.lastClickTime.IsZero() && ev.Time.Sub(r.lastClickTime) <= doubleClickThreshold {
		r.clickCount++
		// A terminal cycles word, line, word: the click after a triple starts
		// a new run rather than counting on to four. Without this, every click
		// past the third stays a line select for as long as the user keeps
		// clicking.
		if r.clickCount > 3 {
			r.clickCount = 1
		}
	} else {
		r.clickCount = 1
	}
	r.lastClickTime = ev.Time
	r.lastClickX, r.lastClickY = ev.X, ev.Y
	r.lastClickButton = r.mousePressButton
	return r.clickCount
}

// commandExecutor runs async commands (Goroutine 3).
// Each command runs in its own goroutine and sends its result back as an event.
//
// Design Note: Commands spawn unbounded goroutines without throttling. This is
// appropriate for typical TUI applications with low command volume. For applications
// that may generate many commands rapidly (e.g., batching operations), consider
// implementing application-level throttling or batching commands into fewer Cmd
// invocations.
func (r *Runtime) commandExecutor() {
	for {
		select {
		case cmd := <-r.cmds:
			// Execute command in a new goroutine
			go func(c Cmd) {
				defer r.capturePanic()
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
