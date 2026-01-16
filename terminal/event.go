package terminal

import "time"

// Event represents any input event from the terminal.
// Events can be keyboard input (KeyEvent) or mouse input (MouseEvent).
// All events provide a timestamp indicating when they occurred.
//
// Use type assertion to determine the specific event type:
//
//	event, err := decoder.ReadEvent()
//	switch e := event.(type) {
//	case terminal.KeyEvent:
//	    // Handle keyboard input
//	    if e.Key == terminal.KeyEnter {
//	        fmt.Println("Enter pressed")
//	    }
//	case terminal.MouseEvent:
//	    // Handle mouse input
//	    if e.Type == terminal.MouseClick {
//	        fmt.Printf("Clicked at %d,%d\n", e.X, e.Y)
//	    }
//	}
type Event interface {
	// Timestamp returns when the event occurred.
	Timestamp() time.Time
}

// CursorPositionEvent represents a cursor position response from the terminal.
// This is sent by the terminal in response to a cursor position query (ESC[6n).
// The response format is ESC [ row ; col R.
type CursorPositionEvent struct {
	Row  int       // 1-based row number
	Col  int       // 1-based column number
	Time time.Time // When the event was received
}

// Timestamp implements Event.
func (e CursorPositionEvent) Timestamp() time.Time {
	return e.Time
}
