package tui

import "time"

// Event represents any event that can occur in the application.
// All events must provide a timestamp indicating when they occurred.
type Event interface {
	// Timestamp returns when the event occurred
	Timestamp() time.Time
}

// TickEvent is emitted periodically based on the runtime's FPS setting.
// Use this for animations and periodic updates.
type TickEvent struct {
	Time  time.Time
	Frame uint64 // Frame counter, increments with each tick
}

func (e TickEvent) Timestamp() time.Time {
	return e.Time
}

// ResizeEvent is emitted when the terminal window is resized.
type ResizeEvent struct {
	Time   time.Time
	Width  int
	Height int
}

func (e ResizeEvent) Timestamp() time.Time {
	return e.Time
}

// QuitEvent signals that the application should shut down.
type QuitEvent struct {
	Time time.Time
}

func (e QuitEvent) Timestamp() time.Time {
	return e.Time
}

// ErrorEvent represents an error that occurred during async command execution.
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
type BatchEvent struct {
	Time   time.Time
	Events []Event
}

func (e BatchEvent) Timestamp() time.Time {
	return e.Time
}

// Note: KeyEvent.Timestamp and MouseEvent.Timestamp are defined in input.go and mouse.go respectively
