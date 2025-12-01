package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deepnoodle-ai/wonton/terminal"
	"golang.org/x/term"
)

// KeyEvent is a keyboard event from the terminal package
type KeyEvent = terminal.KeyEvent

// Key represents special keyboard keys
type Key = terminal.Key

// KeyDecoder handles key event decoding from a byte stream
type KeyDecoder = terminal.KeyDecoder

// NewKeyDecoder creates a new key decoder
var NewKeyDecoder = terminal.NewKeyDecoder

// Paste types
type (
	PasteHandlerDecision = terminal.PasteHandlerDecision
	PasteInfo            = terminal.PasteInfo
	PasteHandler         = terminal.PasteHandler
	PasteDisplayMode     = terminal.PasteDisplayMode
)

// Key constants
const (
	KeyUnknown    = terminal.KeyUnknown
	KeyEnter      = terminal.KeyEnter
	KeyTab        = terminal.KeyTab
	KeyBackspace  = terminal.KeyBackspace
	KeyEscape     = terminal.KeyEscape
	KeyArrowUp    = terminal.KeyArrowUp
	KeyArrowDown  = terminal.KeyArrowDown
	KeyArrowLeft  = terminal.KeyArrowLeft
	KeyArrowRight = terminal.KeyArrowRight
	KeyHome       = terminal.KeyHome
	KeyEnd        = terminal.KeyEnd
	KeyPageUp     = terminal.KeyPageUp
	KeyPageDown   = terminal.KeyPageDown
	KeyDelete     = terminal.KeyDelete
	KeyInsert     = terminal.KeyInsert
	KeyF1         = terminal.KeyF1
	KeyF2         = terminal.KeyF2
	KeyF3         = terminal.KeyF3
	KeyF4         = terminal.KeyF4
	KeyF5         = terminal.KeyF5
	KeyF6         = terminal.KeyF6
	KeyF7         = terminal.KeyF7
	KeyF8         = terminal.KeyF8
	KeyF9         = terminal.KeyF9
	KeyF10        = terminal.KeyF10
	KeyF11        = terminal.KeyF11
	KeyF12        = terminal.KeyF12
	KeyCtrlA      = terminal.KeyCtrlA
	KeyCtrlB      = terminal.KeyCtrlB
	KeyCtrlC      = terminal.KeyCtrlC
	KeyCtrlD      = terminal.KeyCtrlD
	KeyCtrlE      = terminal.KeyCtrlE
	KeyCtrlF      = terminal.KeyCtrlF
	KeyCtrlG      = terminal.KeyCtrlG
	KeyCtrlH      = terminal.KeyCtrlH
	KeyCtrlI      = terminal.KeyCtrlI
	KeyCtrlJ      = terminal.KeyCtrlJ
	KeyCtrlK      = terminal.KeyCtrlK
	KeyCtrlL      = terminal.KeyCtrlL
	KeyCtrlM      = terminal.KeyCtrlM
	KeyCtrlN      = terminal.KeyCtrlN
	KeyCtrlO      = terminal.KeyCtrlO
	KeyCtrlP      = terminal.KeyCtrlP
	KeyCtrlQ      = terminal.KeyCtrlQ
	KeyCtrlR      = terminal.KeyCtrlR
	KeyCtrlS      = terminal.KeyCtrlS
	KeyCtrlT      = terminal.KeyCtrlT
	KeyCtrlU      = terminal.KeyCtrlU
	KeyCtrlV      = terminal.KeyCtrlV
	KeyCtrlW      = terminal.KeyCtrlW
	KeyCtrlX      = terminal.KeyCtrlX
	KeyCtrlY      = terminal.KeyCtrlY
	KeyCtrlZ      = terminal.KeyCtrlZ
)

// Paste constants
const (
	PasteAccept   = terminal.PasteAccept
	PasteReject   = terminal.PasteReject
	PasteModified = terminal.PasteModified
)

const (
	PasteDisplayNormal      = terminal.PasteDisplayNormal
	PasteDisplayPlaceholder = terminal.PasteDisplayPlaceholder
	PasteDisplayHidden      = terminal.PasteDisplayHidden
)

// ReadPassword reads a password with no echo to the terminal.
func ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	fd := int(os.Stdin.Fd())
	bytePassword, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}

// ReadSimple reads a single line of input from stdin.
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
