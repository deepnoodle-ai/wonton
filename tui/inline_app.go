package tui

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deepnoodle-ai/wonton/internal/terminalstate"
	"github.com/deepnoodle-ai/wonton/terminal"
	"golang.org/x/term"
)

// reentrantLock is a goroutine-reentrant mutex used to serialize HandleEvent
// and LiveView calls across goroutines while allowing HandleEvent to call
// Print, PrintRaw, or ClearScrollback synchronously on the event-loop
// goroutine without deadlocking.
type reentrantLock struct {
	mu        sync.Mutex
	owner     int64
	recursion int
}

// curGoroutineID extracts the current goroutine ID from runtime.Stack so
// reentrantLock can detect when Print, PrintRaw, or ClearScrollback is called
// reentrantly from inside HandleEvent on the event-loop goroutine, while still
// blocking concurrent Print calls from background Cmd goroutines so LiveView
// and HandleEvent never execute concurrently.
func curGoroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	field := bytes.TrimPrefix(buf[:n], []byte("goroutine "))
	if idx := bytes.IndexByte(field, ' '); idx >= 0 {
		field = field[:idx]
	}
	id, err := strconv.ParseInt(string(field), 10, 64)
	if err != nil {
		return -1
	}
	return id
}

func (l *reentrantLock) Lock() {
	gid := curGoroutineID()
	if gid >= 0 && atomic.LoadInt64(&l.owner) == gid {
		l.recursion++
		return
	}
	l.mu.Lock()
	atomic.StoreInt64(&l.owner, gid)
	l.recursion = 1
}

func (l *reentrantLock) Unlock() {
	l.recursion--
	if l.recursion == 0 {
		atomic.StoreInt64(&l.owner, -1)
		l.mu.Unlock()
	}
}

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

// inlineConfig configures an InlineApp.
// All fields are optional with sensible zero-value defaults.
type inlineConfig struct {
	Width          int       // 0 = auto (terminal width or 80). Rendering width.
	Output         io.Writer // nil = os.Stdout. Where to write output.
	Input          io.Reader // nil = os.Stdin. Where to read input.
	FPS            int       // 0 = no ticks. Frames per second for TickEvents.
	MouseTracking  bool      // Enable mouse event tracking.
	BracketedPaste bool      // Enable bracketed paste mode.
	PasteTabWidth  int       // 0 = preserve tabs. Convert tabs to N spaces in pastes.
	KittyKeyboard  bool      // Enable Kitty keyboard protocol.
	BackslashEnter bool      // Synthesize Shift+Enter from backslash+Enter (delays typed backslashes; see WithInlineBackslashEnter).
}

// InlineOption is a functional option for configuring NewInlineApp and
// RunInline.
type InlineOption func(*inlineConfig)

// WithInlineWidth sets the rendering width in columns.
// Default is the terminal width, or 80 when not attached to a terminal.
func WithInlineWidth(width int) InlineOption {
	return func(c *inlineConfig) {
		c.Width = width
	}
}

// WithInlineOutput sets where output is written. Default is os.Stdout.
func WithInlineOutput(w io.Writer) InlineOption {
	return func(c *inlineConfig) {
		c.Output = w
	}
}

// WithInlineInput sets where input events are read from. Default is os.Stdin.
func WithInlineInput(r io.Reader) InlineOption {
	return func(c *inlineConfig) {
		c.Input = r
	}
}

// WithInlineFPS sets the frames per second for TickEvents.
// Default is 0, meaning no ticks are delivered.
func WithInlineFPS(fps int) InlineOption {
	return func(c *inlineConfig) {
		c.FPS = fps
	}
}

// WithInlineMouseTracking enables mouse event tracking.
// When enabled, the application will receive MouseEvent events.
func WithInlineMouseTracking(enabled bool) InlineOption {
	return func(c *inlineConfig) {
		c.MouseTracking = enabled
	}
}

// WithInlineBracketedPaste enables bracketed paste mode.
// When enabled, the terminal can distinguish pasted text from typed text,
// allowing proper handling of multi-line pastes.
func WithInlineBracketedPaste(enabled bool) InlineOption {
	return func(c *inlineConfig) {
		c.BracketedPaste = enabled
	}
}

// WithInlinePasteTabWidth configures how tabs in pasted content are handled.
// If width is 0 (default), tabs are preserved as-is.
// If width > 0, each tab is converted to that many spaces.
func WithInlinePasteTabWidth(width int) InlineOption {
	return func(c *inlineConfig) {
		c.PasteTabWidth = width
	}
}

// WithInlineKittyKeyboard enables the Kitty keyboard protocol, which lets
// supporting terminals report keys (e.g. Shift+Enter) unambiguously. Inside
// tmux, which ignores that protocol, it asks for tmux's extended keys instead;
// see terminal.Terminal.EnableEnhancedKeyboard.
func WithInlineKittyKeyboard(enabled bool) InlineOption {
	return func(c *inlineConfig) {
		c.KittyKeyboard = enabled
	}
}

// WithInlineBackslashEnter enables synthesizing Shift+Enter from a backslash
// immediately followed by Enter — a fallback for terminals without the
// Kitty keyboard protocol, useful for chat-style apps where Shift+Enter
// inserts a newline.
//
// Disabled by default: when enabled, every typed backslash is delayed
// briefly while the runtime waits to see if Enter follows, and a backslash
// followed quickly by Enter is rewritten to Shift+Enter (losing the
// backslash). Only enable this for applications that need the Shift+Enter
// affordance.
func WithInlineBackslashEnter(enabled bool) InlineOption {
	return func(c *inlineConfig) {
		c.BackslashEnter = enabled
	}
}

func (c inlineConfig) withDefaults() inlineConfig {
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
	config inlineConfig

	// Runtime state
	app           any
	events        chan Event
	cmds          chan Cmd
	done          chan struct{}
	doneOnce      sync.Once
	stopOnce      sync.Once
	stopRequested chan struct{}
	ticker        *time.Ticker
	frame         uint64

	// Panic capture: a panic in any runtime-managed goroutine is recorded here
	// so Run can restore the terminal before re-panicking with the original
	// stack trace.
	panicMu    sync.Mutex
	panicVal   any
	panicStack []byte
	runExited  bool

	// Focus management
	focusMgr *FocusManager

	// Per-app view state stores (buttons, inputs, clickable regions, ...)
	reg *registries

	// Rendering
	live   *LivePrinter
	output io.Writer

	// Synchronization
	appMu         reentrantLock // Serializes HandleEvent and LiveView calls
	mu            sync.Mutex    // Protects Print operations and running state
	running       bool
	managedInput  *managedInput
	outputMu      sync.Mutex
	handoffActive bool
	handoffErr    error

	// Terminal state (for cleanup)
	oldState                                 *term.State
	stdinFd                                  int
	pasteEnabled, kittyEnabled, mouseEnabled bool
	kittyFraming                             bool

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
//	runner := tui.NewInlineApp(
//	    tui.WithInlineWidth(80),
//	    tui.WithInlineBracketedPaste(true),
//	)
//	if err := runner.Run(&MyApp{}); err != nil {
//	    log.Fatal(err)
//	}
func NewInlineApp(opts ...InlineOption) *InlineApp {
	var cfg inlineConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg = cfg.withDefaults()
	live := NewLivePrinter(WithWidth(cfg.Width), WithOutput(cfg.Output))

	// Set max height to terminal height so the live region never exceeds it.
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if _, h, err := term.GetSize(fd); err == nil && h > 0 {
			live.SetMaxHeight(h)
		}
	}

	reg := newRegistries()
	live.reg = reg

	return &InlineApp{
		config:        cfg,
		events:        make(chan Event, 100),
		cmds:          make(chan Cmd, 100),
		done:          make(chan struct{}),
		stopRequested: make(chan struct{}),
		output:        cfg.Output,
		live:          live,
		focusMgr:      NewFocusManager(),
		reg:           reg,
		appMu:         reentrantLock{owner: -1},
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
func (r *InlineApp) Run(app InlineApplication) error {
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

	// Track attempted enables so failed startup also terminates partial sequences.
	features := []struct {
		wanted bool
		enable func() error
	}{
		{r.config.BracketedPaste, func() error {
			r.pasteEnabled = true
			return checkedHandoffWrite(r.output, "\x1b[?2004h")
		}},
		{r.config.KittyKeyboard, func() error {
			return errors.Join(r.enableKeyboard(), terminalstate.Flush(r.output))
		}},
		{r.config.MouseTracking, func() error {
			r.mouseEnabled = true
			return checkedHandoffWrite(r.output, "\x1b[?1000h\x1b[?1006h")
		}},
	}
	for _, feature := range features {
		if !feature.wanted {
			continue
		}
		if err := feature.enable(); err != nil {
			cleanupErr := r.cleanup()
			r.mu.Lock()
			r.running = false
			r.mu.Unlock()
			return errors.Join(fmt.Errorf("enable terminal features: %w", err), cleanupErr)
		}
	}

	if err := r.initializeManagedInput(); err != nil {
		cleanupErr := r.cleanup()
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
		return errors.Join(fmt.Errorf("initialize terminal input: %w", err), cleanupErr)
	}

	// Start ticker if FPS > 0
	if r.config.FPS > 0 {
		r.ticker = time.NewTicker(time.Second / time.Duration(r.config.FPS))
	}

	// Initialize resize watcher (platform-specific)
	r.setupResizeWatcher()

	// Render initial view. A panic here is captured like one in the event
	// loop, so the terminal is restored below before Run re-panics.
	func() {
		defer r.capturePanic()
		r.render()
	}()
	r.panicMu.Lock()
	renderPanicked := r.panicVal != nil
	r.panicMu.Unlock()

	// Start the four goroutines, unless the first render already failed
	var wg sync.WaitGroup
	if renderPanicked && r.managedInput != nil {
		// The input reader never ran, so release the input it would own.
		r.managedInput.owner.Stop()
		_ = r.managedInput.owner.Close()
	}
	if !renderPanicked {
		wg.Add(4)

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

		// Goroutine 4: Resize listener
		go func() {
			defer wg.Done()
			defer r.capturePanic()
			r.resizeListener()
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Cleanup
	cleanupErr := r.cleanup()

	// Call Destroy if implemented
	if destroy, ok := app.(Destroyable); ok {
		destroy.Destroy()
	}

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()

	// If a goroutine panicked, re-panic now that the terminal is restored.
	// The original stack is embedded in the message since the panic crossed
	// goroutines.
	r.panicMu.Lock()
	pv, ps := r.panicVal, r.panicStack
	r.runExited = true
	r.panicMu.Unlock()
	if pv != nil {
		panic(fmt.Sprintf("tui: application panic: %v\n\noriginal stack:\n%s", pv, ps))
	}

	r.mu.Lock()
	handoffErr := r.handoffErr
	r.mu.Unlock()
	return errors.Join(handoffErr, cleanupErr)
}

// closeDone signals shutdown to all goroutines. Safe to call multiple times
// (quit path and panic path can both reach it).
func (r *InlineApp) closeDone() {
	r.doneOnce.Do(func() { close(r.done) })
}

// capturePanic recovers a panic from a runtime-managed goroutine, records it,
// and triggers shutdown. Run re-panics with the original stack after the
// terminal has been restored, so the trace is readable instead of being
// printed to a raw-mode terminal.
//
// Must be invoked via defer.
func (r *InlineApp) capturePanic() {
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

// cleanup restores terminal state
func (r *InlineApp) cleanup() error {
	if r.ticker != nil {
		r.ticker.Stop()
	}
	r.cleanupResizeWatcher()
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	// Cleanup must attempt all protocols, including an incompletely written enable.
	sequence := "\x1b[?25h"
	if r.live.started && r.live.lastHeight > 0 {
		sequence += "\r\n"
	}
	if r.mouseEnabled {
		sequence += "\x1b[?1006l\x1b[?1000l"
	}
	keyboardErr := r.releaseKeyboard()
	if r.pasteEnabled {
		sequence += "\x1b[?2004l"
	}
	complete, writeErr := terminalstate.WriteControl(r.output, sequence)
	if complete {
		r.live.hiddenCursor = false
		r.pasteEnabled, r.mouseEnabled = false, false
	}
	outputErr := errors.Join(keyboardErr, writeErr, terminalstate.Flush(r.output))
	var attributeErr error
	if r.oldState != nil {
		attributeErr = term.Restore(r.stdinFd, r.oldState)
	}
	return errors.Join(outputErr, attributeErr)
}

// eventLoop is the main event processing loop (Goroutine 1).
// It processes events sequentially, ensuring no race conditions.
func (r *InlineApp) eventLoop() {
	for {
		if r.stopping() {
			r.closeDone()
			return
		}
		select {
		case <-r.stopRequested:
			r.closeDone()
			return
		case event := <-r.events:
			// Process this event and drain any other pending events
			if r.processEventWithQuitCheck(event) {
				r.closeDone()
				return
			}

			// Drain all pending events before rendering
		drainLoop:
			for {
				select {
				case event := <-r.events:
					if r.processEventWithQuitCheck(event) {
						r.closeDone()
						return
					}
				default:
					break drainLoop
				}
			}

			// Render once after processing all pending events
			if r.stopping() {
				r.closeDone()
				return
			}
			r.render()

		case <-func() <-chan time.Time {
			if r.ticker != nil {
				return r.ticker.C
			}
			return nil
		}():
			// Send tick event for animations
			if r.stopping() {
				r.closeDone()
				return
			}
			r.frame++
			tickEvent := TickEvent{
				Time:  time.Now(),
				Frame: r.frame,
			}
			r.processEvent(tickEvent)
			if r.stopping() {
				r.closeDone()
				return
			}
			r.render()

		case <-r.done:
			return
		}
	}
}

// processEventWithQuitCheck processes an event and returns true if it's a quit event
func (r *InlineApp) processEventWithQuitCheck(event Event) bool {
	if r.stopping() {
		return true
	}
	// Check for quit event
	if _, isQuit := event.(QuitEvent); isQuit {
		return true
	}

	// Handle batch events by unpacking them. Recurse so a nested batch (from
	// a nested Sequence) is unpacked and its Quit is honored.
	if batch, isBatch := event.(BatchEvent); isBatch {
		for _, e := range batch.Events {
			if r.processEventWithQuitCheck(e) {
				return true
			}
		}
	} else {
		r.processEvent(event)
	}

	return false
}

// processEvent calls the application's HandleEvent (if implemented) and queues any returned commands.
func (r *InlineApp) processEvent(event Event) {
	cmds := r.handleEventLocked(event)

	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		// Quit and the focus commands take effect now, so an event already
		// queued (the next key press, say) sees their result.
		if isRuntimeCmd(cmd) {
			event := cmd()
			if _, isQuit := event.(QuitEvent); isQuit {
				r.Stop()
				return
			}
			r.processEvent(event)
			continue
		}
		// Queue other commands for async execution
		select {
		case r.cmds <- cmd:
		case <-r.stopRequested:
			return
		case <-r.done:
			return
		}
	}
}

func (r *InlineApp) handleEventLocked(event Event) []Cmd {
	r.appMu.Lock()
	defer r.appMu.Unlock()

	// Handle resize events
	if resize, ok := event.(ResizeEvent); ok {
		r.mu.Lock()
		r.config.Width = resize.Width
		r.live.SetWidth(resize.Width)
		if resize.Height > 0 {
			r.live.SetMaxHeight(resize.Height)
		}
		r.mu.Unlock()
	}

	// Handle focus events from commands
	switch e := event.(type) {
	case FocusSetEvent:
		r.focusMgr.SetFocus(e.ID)
		return nil
	case FocusNextEvent:
		r.focusMgr.FocusNext()
		return nil
	case FocusPrevEvent:
		r.focusMgr.FocusPrev()
		return nil
	}

	// Route events to interactive elements via focus manager
	switch e := event.(type) {
	case MouseEvent:
		if e.Type == MouseClick {
			r.focusMgr.HandleClick(e.X, e.Y)
			r.reg.interactive.HandleClick(e.X, e.Y)
		}
	case KeyEvent:
		// A key consumed by the focused element is NOT delivered to the app,
		// so typing into a focused input can't trigger app shortcuts.
		// Ctrl+C is always delivered so apps can implement a reliable quit
		// shortcut regardless of focus.
		if r.focusMgr.HandleKey(e) && !isInterruptKey(e) {
			return nil
		}
	}

	// Call user's event handler
	if handler, ok := r.app.(EventHandler); ok {
		return handler.HandleEvent(event)
	}
	return nil
}

// render calls the application's LiveView() and updates the live region.
func (r *InlineApp) render() {
	r.appMu.Lock()
	defer r.appMu.Unlock()
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	if r.stopping() {
		return
	}
	_ = r.renderChecked()
}

func (r *InlineApp) renderChecked() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.renderLiveViewLocked(true)
}

// renderLiveViewLocked calls LiveView and draws the live region. The caller
// holds r.appMu, r.outputMu, and r.mu.
func (r *InlineApp) renderLiveViewLocked(useSync bool) error {
	if app, ok := r.app.(InlineApplication); ok {
		// Clear registries before render
		r.focusMgr.Clear()
		r.reg.clearForRender()

		// Build the view tree with this app's registries as the capture
		// target for stateful view constructors (InputField, TextArea, ...).
		view := buildViews(r.reg, app.LiveView)

		// Render the view using LivePrinter with focus manager
		if err := r.live.update(view, useSync, r.focusMgr); err != nil {
			return err
		}

		// Prune TextArea state for IDs that weren't rendered
		r.reg.textAreas.Prune()
	}
	return nil
}

// inputReader reads keyboard and mouse events from stdin (Goroutine 2).
func (r *InlineApp) inputReader() {
	if r.managedInput != nil {
		r.managedInput.run()
		return
	}
	decoder := terminal.NewKeyDecoder(r.config.Input)
	decoder.SetPasteTabWidth(r.config.PasteTabWidth)

	// Channel to receive stdin reads from a separate goroutine
	inputChan := make(chan Event, 1)
	errChan := make(chan error, 1)

	// Start a nested goroutine that continuously reads from stdin
	go func() {
		defer r.capturePanic()
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
			// Check for backslash key - might be start of backslash+Enter
			// sequence (opt-in via WithInlineBackslashEnter)
			if keyEvent, ok := event.(KeyEvent); ok && r.config.BackslashEnter && keyEvent.Rune == '\\' && keyEvent.Key == KeyUnknown {
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

	case MouseDrag:
		// A drag that leaves the press cell is not a click, even if it
		// comes back before the release.
		if mouseEvent.X != r.mousePressX || mouseEvent.Y != r.mousePressY {
			r.mousePressed = false
		}
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
				defer r.capturePanic()
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
// Print does not call LiveView. Below the new scrollback content it redraws
// the live region exactly as the event loop last drew it, so focus styling
// stays and LiveView never runs at the same time as HandleEvent. The event
// loop draws a fresh live region after the current event, as usual.
//
// Thread-safe: Can be called from HandleEvent (recommended) or from
// a Cmd goroutine via the app reference.
func (r *InlineApp) Print(view View) {
	r.appMu.Lock()
	defer r.appMu.Unlock()
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopping() || r.handoffErr != nil || r.finished() {
		return
	}

	// Wrap entire operation in synchronized output mode to prevent flicker.
	// This ensures clear + print + re-render appears as one atomic update.
	fmt.Fprint(r.output, "\033[?2026h") // Begin sync

	// Clear live region (moves cursor back)
	r.live.Clear()

	// Print to scrollback with raw mode line endings
	// Always styled: the scrollback belongs to the same interactive screen as
	// the live region, which LivePrinter always draws in color.
	Fprint(r.output, view, WithWidth(r.config.Width), WithRawMode(true), WithColor(true))
	fmt.Fprint(r.output, "\r\n") // Add newline after printed content

	// Re-render live region (skip its internal sync since we're already in one)
	_ = r.renderLiveViewLocked(false)

	fmt.Fprint(r.output, "\033[?2026l") // End sync
}

// Printf is a convenience method that prints formatted text to scrollback.
// Equivalent to: r.Print(tui.Text(format, args...))
func (r *InlineApp) Printf(format string, args ...any) {
	r.Print(Text(format, args...))
}

// PrintRaw writes raw bytes directly to the terminal scrollback, bypassing the
// cell-based view renderer. This is needed for escape sequences that cannot go
// through the view system (e.g., Kitty graphics protocol image data).
//
// The method clears the live region, writes the raw data, then restores the live
// region — the same lifecycle as Print, but without cell rendering.
//
// Thread-safe: Can be called from HandleEvent or from a Cmd goroutine.
func (r *InlineApp) PrintRaw(data []byte) {
	r.appMu.Lock()
	defer r.appMu.Unlock()
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopping() || r.handoffErr != nil || r.finished() {
		return
	}

	fmt.Fprint(r.output, "\033[?2026h")       // Begin sync
	defer fmt.Fprint(r.output, "\033[?2026l") // End sync (even on panic)

	r.live.Clear()
	r.output.Write(data)

	_ = r.renderLiveViewLocked(false)
}

// Stop requests shutdown without waiting for event dispatch or cleanup.
// Safe to call repeatedly from any goroutine. The current handler settles
// before the event loop stops and Run restores the terminal.
func (r *InlineApp) Stop() {
	r.stopOnce.Do(func() { close(r.stopRequested) })
}

func (r *InlineApp) stopping() bool {
	select {
	case <-r.stopRequested:
		return true
	default:
		return false
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
	r.appMu.Lock()
	defer r.appMu.Unlock()
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopping() || r.handoffErr != nil || r.finished() {
		return
	}
	// Clear live region first
	r.live.Clear()

	// Clear screen and scrollback
	// \033[3J clears scrollback, \033[2J clears screen, \033[H moves to top
	fmt.Fprint(r.output, "\033[3J\033[2J\033[H")

	// Re-render live region
	_ = r.renderLiveViewLocked(true)
}

// RunInline is a convenience function that creates and runs an InlineApp.
//
//	err := tui.RunInline(&MyApp{}, tui.WithInlineBracketedPaste(true))
//
// If the app needs a reference to the runner (to call Print/Printf from
// HandleEvent), create it explicitly instead:
//
//	app := &MyApp{}
//	runner := tui.NewInlineApp()
//	app.runner = runner
//	err := runner.Run(app)
func RunInline(app InlineApplication, opts ...InlineOption) error {
	return NewInlineApp(opts...).Run(app)
}

func (r *InlineApp) finished() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}
