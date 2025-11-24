package gooey

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

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

// Input handles user input with borders and hotkeys
type Input struct {
	terminal        *Terminal
	reader          io.Reader // Injected reader for testability (defaults to os.Stdin)
	decoder         *KeyDecoder
	prompt          string
	promptStyle     Style
	inputStyle      Style
	borderStyle     BorderStyle
	borderColor     Style
	showBorder      bool
	history         []string
	historyIndex    int
	hotkeys         map[Key]func()
	beforeLine      bool
	afterLine       bool
	lineStyle       Style
	placeholder     string
	maxLength       int
	mask            rune
	multiline       bool
	suggestions     []string
	showSuggestions bool
	mu              sync.RWMutex

	// Paste handling
	pasteHandler     PasteHandler
	pasteDisplayMode PasteDisplayMode
	placeholderStyle Style
}

// NewInput creates a new input handler
func NewInput(terminal *Terminal) *Input {
	reader := io.Reader(os.Stdin)
	return &Input{
		terminal:         terminal,
		reader:           reader,
		decoder:          NewKeyDecoder(reader),
		prompt:           "> ",
		promptStyle:      NewStyle().WithForeground(ColorCyan),
		inputStyle:       NewStyle(),
		borderStyle:      SingleBorder,
		borderColor:      NewStyle().WithForeground(ColorBlue),
		showBorder:       false,
		history:          make([]string, 0),
		historyIndex:     -1,
		hotkeys:          make(map[Key]func()),
		beforeLine:       false,
		afterLine:        false,
		lineStyle:        NewStyle().WithForeground(ColorBrightBlack),
		maxLength:        0,
		mask:             0,
		multiline:        false,
		suggestions:      make([]string, 0),
		showSuggestions:  false,
		pasteDisplayMode: PasteDisplayNormal,
		placeholderStyle: NewStyle().WithForeground(ColorBrightBlack).WithItalic(),
	}
}

// WithPrompt sets the input prompt
func (i *Input) WithPrompt(prompt string, style Style) *Input {
	i.prompt = prompt
	i.promptStyle = style
	return i
}

// WithBorder enables bordered input
func (i *Input) WithBorder(border BorderStyle, color Style) *Input {
	i.showBorder = true
	i.borderStyle = border
	i.borderColor = color
	return i
}

// WithLines adds horizontal lines before and after input
func (i *Input) WithLines(before, after bool, style Style) *Input {
	i.beforeLine = before
	i.afterLine = after
	i.lineStyle = style
	return i
}

// WithPlaceholder sets placeholder text
func (i *Input) WithPlaceholder(placeholder string) *Input {
	i.placeholder = placeholder
	return i
}

// WithMaxLength sets maximum input length
func (i *Input) WithMaxLength(length int) *Input {
	i.maxLength = length
	return i
}

// WithMask sets a mask character for password input
func (i *Input) WithMask(mask rune) *Input {
	i.mask = mask
	return i
}

// SetReader sets the input reader for testability.
// By default, Input reads from os.Stdin, but this allows injecting
// a custom reader (e.g., bytes.Buffer) for testing.
func (i *Input) SetReader(reader io.Reader) *Input {
	i.reader = reader
	i.decoder = NewKeyDecoder(reader)
	return i
}

// EnableMultiline enables multiline input
func (i *Input) EnableMultiline() *Input {
	i.multiline = true
	return i
}

// SetHotkey registers a hotkey handler
func (i *Input) SetHotkey(key Key, handler func()) *Input {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.hotkeys[key] = handler
	return i
}

// AddToHistory adds input to history
func (i *Input) AddToHistory(input string) {
	if input != "" && (len(i.history) == 0 || i.history[len(i.history)-1] != input) {
		i.history = append(i.history, input)
	}
	i.historyIndex = len(i.history)
}

// SetSuggestions sets autocomplete suggestions
func (i *Input) SetSuggestions(suggestions []string) *Input {
	i.suggestions = suggestions
	i.showSuggestions = len(suggestions) > 0
	return i
}

// WithPasteHandler sets a custom paste handler callback.
// The handler is called when paste content is received and can inspect,
// modify, or reject the paste.
//
// Example - Reject pastes over 1000 characters:
//
//	input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
//	    if info.ByteCount > 1000 {
//	        return gooey.PasteReject, ""
//	    }
//	    return gooey.PasteAccept, ""
//	})
//
// Example - Strip ANSI codes from paste:
//
//	input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
//	    cleaned := stripANSI(info.Content)
//	    return gooey.PasteModified, cleaned
//	})
func (i *Input) WithPasteHandler(handler PasteHandler) *Input {
	i.pasteHandler = handler
	return i
}

// WithPasteDisplayMode sets how pasted content is displayed.
//
// - PasteDisplayNormal: Shows the pasted content normally (default)
// - PasteDisplayPlaceholder: Shows a placeholder like "[pasted 27 lines]"
// - PasteDisplayHidden: Content is added silently without visual feedback
//
// Example - Show placeholder for multi-line pastes:
//
//	input.WithPasteDisplayMode(gooey.PasteDisplayPlaceholder)
func (i *Input) WithPasteDisplayMode(mode PasteDisplayMode) *Input {
	i.pasteDisplayMode = mode
	return i
}

// WithPlaceholderStyle sets the style for paste placeholders.
// Only applies when PasteDisplayMode is PasteDisplayPlaceholder.
func (i *Input) WithPlaceholderStyle(style Style) *Input {
	i.placeholderStyle = style
	return i
}

// Read reads user input with all configured features
func (i *Input) Read() (string, error) {
	// Draw input area
	i.drawInputArea()
	i.terminal.Flush()

	// Enable raw mode for key detection
	if err := i.terminal.EnableRawMode(); err != nil {
		return "", err
	}
	defer i.terminal.DisableRawMode()

	// Drain any leftover input from the buffer to prevent stray newlines
	// This can happen after terminal mode transitions (e.g., after password input)
	for i.decoder.reader.Buffered() > 0 {
		b, err := i.decoder.reader.ReadByte()
		if err != nil {
			break
		}
		// Only drain newlines and carriage returns, preserve other characters
		if b != '\n' && b != '\r' {
			// Put the byte back if it's not a newline
			i.decoder.reader.UnreadByte()
			break
		}
	}

	var buffer []rune
	cursorPos := 0

	for {
		// Read key event
		event := i.readKeyEvent()

		// Handle paste events
		if event.Paste != "" {
			// Create paste info
			pasteInfo := PasteInfo{
				Content:   event.Paste,
				LineCount: strings.Count(event.Paste, "\n") + 1,
				ByteCount: len(event.Paste),
			}

			// Call paste handler if configured
			finalContent := event.Paste
			if i.pasteHandler != nil {
				decision, modifiedContent := i.pasteHandler(pasteInfo)
				switch decision {
				case PasteReject:
					// Reject the paste - don't insert anything
					continue
				case PasteModified:
					// Use the modified content
					finalContent = modifiedContent
				case PasteAccept:
					// Use original content (already set)
				}
			}

			// Handle display mode
			switch i.pasteDisplayMode {
			case PasteDisplayPlaceholder:
				// Show placeholder instead of content
				var placeholder string
				if pasteInfo.LineCount == 1 {
					placeholder = fmt.Sprintf("[pasted %d chars]", pasteInfo.ByteCount)
				} else {
					placeholder = fmt.Sprintf("[pasted %d lines]", pasteInfo.LineCount)
				}

				// Insert actual content into buffer (hidden from display)
				for _, r := range finalContent {
					if i.maxLength == 0 || len(buffer) < i.maxLength {
						buffer = i.insertRune(buffer, cursorPos, r)
						cursorPos++
					}
				}

				// Update display with placeholder temporarily
				i.updateDisplayWithPlaceholder(buffer, cursorPos, placeholder)
				i.terminal.Flush()

			case PasteDisplayHidden:
				// Insert content silently without visual feedback
				for _, r := range finalContent {
					if i.maxLength == 0 || len(buffer) < i.maxLength {
						buffer = i.insertRune(buffer, cursorPos, r)
						cursorPos++
					}
				}
				// Update display normally (buffer includes paste but no visual indication)
				i.updateDisplay(buffer, cursorPos)
				i.terminal.Flush()

			case PasteDisplayNormal:
				fallthrough
			default:
				// Insert entire paste content at cursor position
				for _, r := range finalContent {
					if i.maxLength == 0 || len(buffer) < i.maxLength {
						buffer = i.insertRune(buffer, cursorPos, r)
						cursorPos++
					}
				}
				// Update display and continue
				i.updateDisplay(buffer, cursorPos)
				i.terminal.Flush()
			}
			continue
		}

		// Check hotkeys
		i.mu.RLock()
		handler, ok := i.hotkeys[event.Key]
		i.mu.RUnlock()

		if ok {
			handler()
			continue
		}

		switch event.Key {
		case KeyEnter:
			if !i.multiline || (i.multiline && event.Ctrl) {
				i.clearInputArea()
				result := string(buffer)
				i.AddToHistory(result)
				i.terminal.Println("")
				i.terminal.Flush()
				return result, nil
			}
			// In multiline mode, add newline
			buffer = i.insertRune(buffer, cursorPos, '\n')
			cursorPos++

		case KeyBackspace:
			if cursorPos > 0 && len(buffer) > 0 {
				buffer = i.deleteRune(buffer, cursorPos-1)
				cursorPos--
			}

		case KeyDelete:
			if cursorPos < len(buffer) {
				buffer = i.deleteRune(buffer, cursorPos)
			}

		case KeyArrowLeft:
			if cursorPos > 0 {
				cursorPos--
			}

		case KeyArrowRight:
			if cursorPos < len(buffer) {
				cursorPos++
			}

		case KeyArrowUp:
			if len(i.history) > 0 && i.historyIndex > 0 {
				i.historyIndex--
				buffer = []rune(i.history[i.historyIndex])
				cursorPos = len(buffer)
			}

		case KeyArrowDown:
			if i.historyIndex < len(i.history)-1 {
				i.historyIndex++
				buffer = []rune(i.history[i.historyIndex])
				cursorPos = len(buffer)
			} else if i.historyIndex == len(i.history)-1 {
				i.historyIndex = len(i.history)
				buffer = []rune{}
				cursorPos = 0
			}

		case KeyHome:
			cursorPos = 0

		case KeyEnd:
			cursorPos = len(buffer)

		case KeyEscape, KeyCtrlC:
			i.clearInputArea()
			return "", fmt.Errorf("input cancelled")

		case KeyTab:
			// Handle autocomplete
			if i.showSuggestions && len(i.suggestions) > 0 {
				current := string(buffer)
				matches := FuzzySearch(current, i.suggestions)
				if len(matches) > 0 {
					buffer = []rune(matches[0])
					cursorPos = len(buffer)
				}
			}

		default:
			// Regular character input
			if event.Rune != 0 {
				if i.maxLength == 0 || len(buffer) < i.maxLength {
					buffer = i.insertRune(buffer, cursorPos, event.Rune)
					cursorPos++
				}
			}
		}

		// Update display
		i.updateDisplay(buffer, cursorPos)

		// Show suggestions if applicable
		if i.showSuggestions {
			i.drawSuggestions(string(buffer))
		}

		i.terminal.Flush()
	}
}

func (i *Input) drawInputArea() {
	width, _ := i.terminal.Size()

	if i.beforeLine {
		i.drawHorizontalLine()
	}

	if i.showBorder {
		// Draw bordered input area
		frame := NewFrame(0, 0, width, 3).
			WithBorderStyle(i.borderStyle).
			WithColor(i.borderColor)

		rf, _ := i.terminal.BeginFrame()
		frame.Draw(rf)
		i.terminal.EndFrame(rf)

		i.terminal.MoveCursor(2, 1)
	}

	// Show prompt
	i.terminal.SetStyle(i.promptStyle)
	i.terminal.Print(i.prompt)

	// Show placeholder if needed
	if i.placeholder != "" {
		placeholderStyle := NewStyle().WithForeground(ColorBrightBlack)
		i.terminal.SetStyle(placeholderStyle)
		i.terminal.Print(i.placeholder)
		i.terminal.MoveCursorLeft(utf8.RuneCountInString(i.placeholder))
	}

	i.terminal.Reset()
}

func (i *Input) clearInputArea() {
	if i.showBorder {
		// Clear bordered area
		i.terminal.MoveCursor(0, 0)
		for j := 0; j < 3; j++ {
			i.terminal.ClearLine()
			i.terminal.MoveCursorDown(1)
		}
		i.terminal.MoveCursor(0, 0)
	} else {
		i.terminal.ClearLine()
	}
}

func (i *Input) updateDisplay(buffer []rune, cursorPos int) {
	// Clear current line
	i.terminal.MoveCursorLeft(1000)
	i.terminal.ClearToEndOfLine()

	// Redraw prompt
	i.terminal.SetStyle(i.promptStyle)
	i.terminal.Print(i.prompt)

	// Draw input text
	text := string(buffer)
	if i.mask != 0 {
		text = strings.Repeat(string(i.mask), len(buffer))
	}
	i.terminal.SetStyle(i.inputStyle)
	i.terminal.Print(text)
	i.terminal.Reset()

	// Position cursor
	if cursorPos < len(buffer) {
		i.terminal.MoveCursorLeft(len(buffer) - cursorPos)
	}
}

// updateDisplayWithPlaceholder updates the display showing a placeholder instead of the actual buffer content.
// This is used for PasteDisplayPlaceholder mode to show something like "[pasted 27 lines]" instead of the full content.
func (i *Input) updateDisplayWithPlaceholder(buffer []rune, cursorPos int, placeholder string) {
	// Clear current line
	i.terminal.MoveCursorLeft(1000)
	i.terminal.ClearToEndOfLine()

	// Redraw prompt
	i.terminal.SetStyle(i.promptStyle)
	i.terminal.Print(i.prompt)

	// Show the placeholder with special styling
	i.terminal.SetStyle(i.placeholderStyle)
	i.terminal.Print(placeholder)
	i.terminal.Reset()

	// Position cursor at end of placeholder
	// (The actual buffer has the full content, but we're only showing the placeholder)
}

func (i *Input) drawHorizontalLine() {
	width, _ := i.terminal.Size()
	line := strings.Repeat("─", width)
	i.terminal.SetStyle(i.lineStyle)
	i.terminal.Println(line)
	i.terminal.Reset()
}

func (i *Input) drawSuggestions(current string) {
	// Clear suggestion area
	i.terminal.MoveCursorDown(1)
	i.terminal.ClearLine()

	matches := FuzzySearch(current, i.suggestions)
	filtered := []string{}
	for _, m := range matches {
		if m != current {
			filtered = append(filtered, m)
		}
	}
	matches = filtered

	if len(matches) > 0 {
		suggestionStyle := NewStyle().WithForeground(ColorBrightBlack)
		i.terminal.SetStyle(suggestionStyle)
		i.terminal.Print("  Suggestions: ")
		for idx, match := range matches {
			if idx > 2 { // Show max 3 suggestions
				break
			}
			if idx > 0 {
				i.terminal.Print(", ")
			}
			i.terminal.Print(match)
		}
	}

	// Move back to input line
	i.terminal.MoveCursorUp(1)
	i.terminal.MoveCursorRight(len(i.prompt) + len(current))
}

func (i *Input) insertRune(buffer []rune, pos int, r rune) []rune {
	if pos < 0 || pos > len(buffer) {
		return buffer
	}

	result := make([]rune, 0, len(buffer)+1)
	result = append(result, buffer[:pos]...)
	result = append(result, r)
	result = append(result, buffer[pos:]...)
	return result
}

func (i *Input) deleteRune(buffer []rune, pos int) []rune {
	if pos < 0 || pos >= len(buffer) {
		return buffer
	}

	result := make([]rune, 0, len(buffer)-1)
	result = append(result, buffer[:pos]...)
	result = append(result, buffer[pos+1:]...)
	return result
}

// readKeyEvent reads a single key event using the unified KeyDecoder.
// This method delegates to the decoder for consistent key handling across all input methods.
func (i *Input) readKeyEvent() KeyEvent {
	event, err := i.decoder.ReadKeyEvent()
	if err != nil {
		// On error (EOF, closed pipe, etc.), return unknown key
		// This maintains backward compatibility with the old behavior
		return KeyEvent{Key: KeyUnknown}
	}

	// Record input if recording is active
	if i.terminal.recorder != nil {
		inputStr := keyEventToString(event)
		if inputStr != "" {
			i.terminal.recorder.RecordInput(inputStr)
		}
	}

	return event
}

// ReadKeyEvent reads a single key event from the input stream.
// This is a public method for character-by-character input handling.
// It blocks until a key is pressed.
//
// Example:
//
//	input := gooey.NewInput(terminal)
//	event := input.ReadKeyEvent()
//	if event.Rune != 0 {
//	    fmt.Printf("You pressed: %c\n", event.Rune)
//	}
func (i *Input) ReadKeyEvent() KeyEvent {
	return i.readKeyEvent()
}

// ReadSimple reads a single line of input using a simple buffered scanner.
// This is the recommended method for basic line reading without advanced features.
// For advanced features like history, suggestions, or masking, use Read() instead.
//
// Example:
//
//	input := gooey.NewInput(terminal)
//	input.WithPrompt("Name: ", gooey.NewStyle())
//	name, err := input.ReadSimple()
func (i *Input) ReadSimple() (string, error) {
	if i.beforeLine {
		i.drawHorizontalLine()
	}

	// Show prompt
	i.terminal.SetStyle(i.promptStyle)
	i.terminal.Print(i.prompt)
	i.terminal.Flush()

	// Read input using decoder's underlying reader to avoid creating multiple buffers on stdin
	// This prevents buffer conflicts when used with other input methods
	var result strings.Builder
	for {
		b, err := i.decoder.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		if b == '\n' {
			break
		}
		if b == '\r' {
			// Skip carriage return, wait for newline
			continue
		}
		result.WriteByte(b)
	}

	input := result.String()

	// Move to next line after input
	i.terminal.Println("")
	i.terminal.Flush()

	if i.afterLine {
		i.drawHorizontalLine()
	}

	i.AddToHistory(input)
	return input, nil
}

// ReadPassword reads a password with no echo to the terminal.
// This is the recommended method for secure password input.
// The input is not displayed on screen.
//
// Example:
//
//	input := gooey.NewInput(terminal)
//	input.WithPrompt("Password: ", gooey.NewStyle())
//	password, err := input.ReadPassword()
func (i *Input) ReadPassword() (string, error) {
	if i.beforeLine {
		i.drawHorizontalLine()
	}

	i.terminal.SetStyle(i.promptStyle)
	i.terminal.Print(i.prompt)
	i.terminal.Reset()
	i.terminal.Flush()

	// Use terminal package for secure password input
	fd := int(os.Stdin.Fd())
	bytePassword, err := term.ReadPassword(fd)
	if err != nil {
		return "", err
	}

	result := string(bytePassword)
	i.terminal.Println("") // term.ReadPassword doesn't add newline

	if i.afterLine {
		i.drawHorizontalLine()
	}

	// Note: We don't add passwords to history
	return result, nil
}
