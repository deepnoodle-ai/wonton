package tui

import "time"

// Event represents any event that can occur in the application.
// All events must provide a timestamp indicating when they occurred.
//
// The Runtime processes events sequentially in a single-threaded loop,
// calling HandleEvent (if implemented) and then View() to render updates.
//
// Built-in event types include:
//   - KeyEvent: Keyboard input
//   - MouseEvent: Mouse clicks, movement, scrolling
//   - TickEvent: Periodic timer for animations
//   - ResizeEvent: Terminal size changed
//   - QuitEvent: Application shutdown request
//   - ErrorEvent: Error from async command
//   - BatchEvent: Multiple events to process together
//
// Applications can define custom event types for command results:
//
//	type DataReceivedEvent struct {
//	    Time time.Time
//	    Data []byte
//	}
//
//	func (e DataReceivedEvent) Timestamp() time.Time { return e.Time }
type Event interface {
	// Timestamp returns when the event occurred
	Timestamp() time.Time
}

// TickEvent is emitted periodically based on the runtime's FPS setting.
// Use this for animations and periodic updates.
//
// The Frame counter increments with each tick, starting from 1. This is
// useful for frame-based animations that need consistent timing.
//
// Example:
//
//	func (a *App) HandleEvent(event Event) []Cmd {
//	    if tick, ok := event.(TickEvent); ok {
//	        a.rotation = (a.rotation + 1) % 360
//	    }
//	    return nil
//	}
type TickEvent struct {
	Time  time.Time
	Frame uint64 // Frame counter, increments with each tick
}

func (e TickEvent) Timestamp() time.Time {
	return e.Time
}

// ResizeEvent is emitted when the terminal window is resized.
// Applications receive an initial ResizeEvent on startup with the
// current terminal dimensions.
//
// Example:
//
//	func (a *App) HandleEvent(event Event) []Cmd {
//	    if resize, ok := event.(ResizeEvent); ok {
//	        a.width = resize.Width
//	        a.height = resize.Height
//	    }
//	    return nil
//	}
type ResizeEvent struct {
	Time   time.Time
	Width  int // Terminal width in columns
	Height int // Terminal height in rows
}

func (e ResizeEvent) Timestamp() time.Time {
	return e.Time
}

// QuitEvent signals that the application should shut down.
// The Runtime will exit the event loop, call Destroy() if implemented,
// and return from Run().
type QuitEvent struct {
	Time time.Time
}

func (e QuitEvent) Timestamp() time.Time {
	return e.Time
}

// ErrorEvent represents an error that occurred during async command execution.
// Applications should handle these events to display errors to users.
//
// Example:
//
//	func (a *App) HandleEvent(event Event) []Cmd {
//	    if err, ok := event.(ErrorEvent); ok {
//	        a.errorMsg = err.Error()
//	    }
//	    return nil
//	}
type ErrorEvent struct {
	Time  time.Time
	Err   error
	Cause string // Optional description of what caused the error
}

func (e ErrorEvent) Timestamp() time.Time {
	return e.Time
}

func (e ErrorEvent) Error() string {
	if e.Cause != "" {
		return e.Cause + ": " + e.Err.Error()
	}
	return e.Err.Error()
}

// BatchEvent contains multiple events that should be processed together.
// The runtime will unpack this and process each event individually in order.
//
// This is typically returned by Sequence() to group related command results.
type BatchEvent struct {
	Time   time.Time
	Events []Event
}

func (e BatchEvent) Timestamp() time.Time {
	return e.Time
}

// Note: KeyEvent.Timestamp and MouseEvent.Timestamp are defined in input.go and mouse.go respectively
