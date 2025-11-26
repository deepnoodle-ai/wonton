package gooey

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Key represents special keyboard keys
type Key int

const (
	// KeyUnknown is the zero value, used when no special key is pressed
	// IMPORTANT: Must be 0 so regular characters (with Key field unset) don't match special keys
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

// KeyEvent represents a keyboard event
type KeyEvent struct {
	Key   Key
	Rune  rune
	Alt   bool
	Ctrl  bool
	Shift bool
	Paste string // If non-empty, this event represents a paste operation
}

// PasteHandlerDecision represents the decision made by a paste handler
type PasteHandlerDecision int

const (
	// PasteAccept indicates the paste should be accepted and inserted normally
	PasteAccept PasteHandlerDecision = iota
	// PasteReject indicates the paste should be rejected completely
	PasteReject
	// PasteModified indicates the paste content has been modified by the handler
	PasteModified
)

// PasteInfo contains information about a paste event
type PasteInfo struct {
	Content   string // The pasted content
	LineCount int    // Number of lines in the paste
	ByteCount int    // Number of bytes in the paste
}

// PasteHandler is called when paste content is received.
// It can inspect, modify, or reject the paste.
// Return (decision, modifiedContent):
//   - (PasteAccept, "") to accept the paste as-is
//   - (PasteReject, "") to reject the paste
//   - (PasteModified, newContent) to replace the paste with modified content
type PasteHandler func(info PasteInfo) (PasteHandlerDecision, string)

// PasteDisplayMode controls how pasted content is displayed
type PasteDisplayMode int

const (
	// PasteDisplayNormal shows the pasted content normally (default behavior)
	PasteDisplayNormal PasteDisplayMode = iota
	// PasteDisplayPlaceholder shows a placeholder like "[pasted 27 lines]" instead of the content
	PasteDisplayPlaceholder
	// PasteDisplayHidden doesn't show anything (content is added silently)
	PasteDisplayHidden
)

// Legacy input helpers removed. Use Runtime.HandleEvent with KeyEvent instead.
// For secure password input, use term.ReadPassword directly or standard library.

// ReadPassword reads a password with no echo to the terminal.
// This is a helper wrapper around term.ReadPassword.
func ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	fd := int(os.Stdin.Fd())
	bytePassword, err := term.ReadPassword(fd)
	fmt.Println() // Add newline
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}

// ReadSimple reads a single line of input from stdin.
// This is a simple blocking helper for basic scripts.
// For interactive TUI applications, use the Runtime.
func ReadSimple(prompt string) (string, error) {
	fmt.Print(prompt)
	var result strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if n == 0 {
			continue
		}
		b := buf[0]
		if b == '\n' {
			break
		}
		if b == '\r' {
			continue
		}
		result.WriteByte(b)
	}
	return result.String(), nil
}
