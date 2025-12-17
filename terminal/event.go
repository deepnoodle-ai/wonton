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
