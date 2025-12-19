package terminal

import "time"

// Key represents special keyboard keys that don't correspond to printable characters.
// These are commonly used control keys, function keys, and navigation keys.
//
// When a KeyEvent has a non-zero Key value, it represents a special key press.
// When Key is KeyUnknown (zero), check the Rune field for printable characters.
type Key int

const (
	// KeyUnknown is the zero value, used when no special key is pressed.
	// When Key is KeyUnknown, check the KeyEvent.Rune field for the actual character.
	//
	// IMPORTANT: Must be 0 so regular characters (with Key field unset) don't match special keys.
	KeyUnknown Key = 0

	// Special keys start at 1 to avoid conflicting with zero value
	KeyEnter Key = iota
	KeyTab
	KeyBackspace
	KeyEscape
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyDelete
	KeyInsert
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyCtrlA
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlG
	KeyCtrlH
	KeyCtrlI
	KeyCtrlJ
	KeyCtrlK
	KeyCtrlL
	KeyCtrlM
	KeyCtrlN
	KeyCtrlO
	KeyCtrlP
	KeyCtrlQ
	KeyCtrlR
	KeyCtrlS
	KeyCtrlT
	KeyCtrlU
	KeyCtrlV
	KeyCtrlW
	KeyCtrlX
	KeyCtrlY
	KeyCtrlZ
)

// KeyEvent represents a keyboard input event from the terminal.
//
// A KeyEvent can represent either:
//  1. A special key press (arrows, function keys, etc.) - check the Key field
//  2. A printable character - check the Rune field
//  3. A paste operation - check the Paste field
//
// # Special Keys
//
// When Key is not KeyUnknown, the event represents a special key like Enter, Escape, or function keys.
// The Alt, Ctrl, and Shift fields indicate which modifiers were held.
//
//	if event.Key == terminal.KeyEnter && event.Ctrl {
//	    // Ctrl+Enter pressed
//	}
//
// # Printable Characters
//
// When Key is KeyUnknown, check the Rune field for the actual character:
//
//	if event.Key == terminal.KeyUnknown && event.Rune == 'a' {
//	    // User typed 'a'
//	}
//
// # Paste Events
//
// When bracketed paste mode is enabled (via Terminal.EnableBracketedPaste),
// pasted content is delivered as a single KeyEvent with the Paste field set:
//
//	if event.Paste != "" {
//	    // User pasted text
//	    fmt.Println("Pasted:", event.Paste)
//	}
type KeyEvent struct {
	Key   Key       // Special key (KeyEnter, KeyArrowUp, etc.), or KeyUnknown for regular chars
	Rune  rune      // The character for printable keys (only valid when Key is KeyUnknown)
	Alt   bool      // True if Alt/Option modifier was held
	Ctrl  bool      // True if Ctrl/Command modifier was held
	Shift bool      // True if Shift modifier was held
	Paste string    // If non-empty, this event represents a paste operation (bracketed paste mode)
	Time  time.Time // When the event occurred
}

// Timestamp implements the Event interface
func (e KeyEvent) Timestamp() time.Time {
	if e.Time.IsZero() {
		return time.Now()
	}
	return e.Time
}

// PasteHandlerDecision represents the decision made by a PasteHandler
// about how to handle pasted content.
type PasteHandlerDecision int

const (
	// PasteAccept indicates the paste should be accepted and inserted normally.
	PasteAccept PasteHandlerDecision = iota

	// PasteReject indicates the paste should be rejected completely.
	// Use this for security checks or when the paste contains invalid content.
	PasteReject

	// PasteModified indicates the paste content has been modified by the handler.
	// Return this along with the modified content string.
	PasteModified
)

// PasteInfo contains information about a paste event that can be used
// by a PasteHandler to make decisions about the pasted content.
type PasteInfo struct {
	Content   string // The pasted content (may contain multiple lines)
	LineCount int    // Number of lines in the paste (lines separated by \n)
	ByteCount int    // Number of bytes in the paste
}

// PasteHandler is called when paste content is received in bracketed paste mode.
// It allows the application to inspect, modify, or reject pasted content before insertion.
//
// The handler receives a PasteInfo with details about the paste and should return:
//   - (PasteAccept, "") to accept the paste as-is
//   - (PasteReject, "") to reject the paste completely
//   - (PasteModified, newContent) to replace the paste with modified content
//
// Example use cases:
//   - Limit paste size to prevent resource exhaustion
//   - Strip dangerous content or escape sequences
//   - Normalize line endings or whitespace
//   - Show a confirmation dialog for large pastes
//
// Example:
//
//	handler := func(info terminal.PasteInfo) (terminal.PasteHandlerDecision, string) {
//	    if info.LineCount > 100 {
//	        // Reject pastes larger than 100 lines
//	        return terminal.PasteReject, ""
//	    }
//	    if strings.Contains(info.Content, "\x1b") {
//	        // Strip ANSI escape sequences
//	        cleaned := stripANSI(info.Content)
//	        return terminal.PasteModified, cleaned
//	    }
//	    return terminal.PasteAccept, ""
//	}
type PasteHandler func(info PasteInfo) (PasteHandlerDecision, string)

// PasteDisplayMode controls how pasted content is displayed in the terminal.
// This is separate from whether the paste is accepted or rejected.
type PasteDisplayMode int

const (
	// PasteDisplayNormal shows the pasted content normally (default behavior).
	// The content is echoed to the terminal as it would be if typed.
	PasteDisplayNormal PasteDisplayMode = iota

	// PasteDisplayPlaceholder shows a placeholder like "[pasted 27 lines]" instead of the content.
	// Use this to avoid cluttering the screen with large pastes.
	PasteDisplayPlaceholder

	// PasteDisplayHidden doesn't show anything (content is added silently).
	// Use this when you're handling the display yourself or for password fields.
	PasteDisplayHidden
)
