// Package terminal provides terminal keyboard and mouse input handling.
//
// This package decodes terminal input sequences including:
//   - Multi-byte UTF-8 characters
//   - ANSI escape sequences (arrows, function keys, etc.)
//   - Alt/Meta and Ctrl modifiers
//   - Mouse events (SGR extended mode)
//   - Bracketed paste detection
//
// Example usage:
//
//	decoder := terminal.NewKeyDecoder(os.Stdin)
//	for {
//	    event, err := decoder.ReadEvent()
//	    if err != nil {
//	        break
//	    }
//	    switch e := event.(type) {
//	    case terminal.KeyEvent:
//	        fmt.Printf("Key: %v\n", e)
//	    case terminal.MouseEvent:
//	        fmt.Printf("Mouse: %v\n", e)
//	    }
//	}
package terminal

import "time"

// Event represents any input event from the terminal.
// All events provide a timestamp indicating when they occurred.
type Event interface {
	// Timestamp returns when the event occurred
	Timestamp() time.Time
}
