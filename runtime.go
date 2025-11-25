package gooey

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// Application is the main interface for message-driven applications.
// Implementations of this interface handle events and render the current state.
//
// Thread Safety: HandleEvent and Render are NEVER called concurrently.
// The Runtime ensures these methods run sequentially in a single goroutine,
// eliminating the need for locks in application code.
type Application interface {
	// HandleEvent processes an event and optionally returns commands for async execution.
	// This is called in a single-threaded event loop, so state can be mutated freely
	// without locks. Return commands for async operations (HTTP requests, timers, etc.).
	HandleEvent(event Event) []Cmd

	// Render draws the current application state using the provided frame.
	// This is called automatically after each HandleEvent.
	// Uses Gooey's existing BeginFrame/EndFrame system for flicker-free rendering.
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
//   - Goroutine 1: Main event loop (processes events sequentially, calls HandleEvent/Render)
//   - Goroutine 2: Input reader (blocks on stdin, sends KeyEvents)
//   - Goroutine 3: Command executor (runs async commands, sends results as events)
//
// This design eliminates race conditions in application code while maintaining
// responsive UI through non-blocking async operations.
type Runtime struct {
	terminal *Terminal
	app      Application
	events   chan Event
	cmds     chan Cmd
	done     chan struct{}
	ticker   *time.Ticker
	fps      int
	frame    uint64 // Frame counter for TickEvents

	mu          sync.Mutex
	running     bool
	resizeUnsub func() // Unsubscribe function for resize callback
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

	return &Runtime{
		terminal: terminal,
		app:      app,
		events:   make(chan Event, 100), // Buffered to prevent blocking
		cmds:     make(chan Cmd, 100),
		done:     make(chan struct{}),
		fps:      fps,
		frame:    0,
	}
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
		if err := r.terminal.EnableRawMode(); err != nil {
			return fmt.Errorf("failed to enable raw mode: %w", err)
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
func (r *Runtime) eventLoop() {
	for {
		select {
		case event := <-r.events:
			// Check for quit event
			if _, isQuit := event.(QuitEvent); isQuit {
				close(r.done)
				return
			}

			// Handle batch events by unpacking them
			if batch, isBatch := event.(BatchEvent); isBatch {
				for _, e := range batch.Events {
					r.processEvent(e)
					if _, isQuit := e.(QuitEvent); isQuit {
						close(r.done)
						return
					}
				}
			} else {
				r.processEvent(event)
			}

			// Render after processing event
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

// processEvent calls the application's HandleEvent and queues any returned commands.
func (r *Runtime) processEvent(event Event) {
	// Call user's event handler
	cmds := r.app.HandleEvent(event)

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

// render calls the application's Render method using BeginFrame/EndFrame.
func (r *Runtime) render() {
	frame, err := r.terminal.BeginFrame()
	if err != nil {
		// Terminal not ready, skip this frame
		return
	}

	// Application renders to back buffer
	r.app.Render(frame)

	// Flush to screen (diffs and sends only dirty regions)
	r.terminal.EndFrame(frame)
}

// inputReader reads keyboard events from stdin (Goroutine 2).
// This goroutine blocks on stdin reads and forwards events to the main loop.
func (r *Runtime) inputReader() {
	decoder := NewKeyDecoder(os.Stdin)

	for {
		select {
		case <-r.done:
			return
		default:
			// This blocks until a key is pressed
			event, err := decoder.ReadKeyEvent()
			if err != nil {
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

			// Forward to main event loop
			select {
			case r.events <- event:
			case <-r.done:
				return
			}
		}
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
