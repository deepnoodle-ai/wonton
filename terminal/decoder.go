package terminal

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// KeyDecoder handles low-level decoding of terminal input into structured events.
//
// The decoder reads raw bytes from an input stream (typically os.Stdin) and
// decodes them into KeyEvent and MouseEvent structs. It handles the complexity
// of multi-byte sequences, escape codes, and various terminal protocols.
//
// # Supported Input
//
// The decoder supports:
//   - Single-byte printable ASCII characters
//   - Multi-byte UTF-8 characters (Unicode)
//   - ANSI escape sequences (arrows, function keys, Home/End, etc.)
//   - Ctrl combinations (Ctrl+A through Ctrl+Z)
//   - Alt/Meta modifiers (Alt+key)
//   - Shift modifiers (when supported by the terminal)
//   - Mouse events in SGR extended format
//   - Bracketed paste mode (large clipboard pastes)
//   - Kitty keyboard protocol for enhanced key detection
//
// # Usage
//
// Create a decoder and call ReadEvent in a loop:
//
//	decoder := terminal.NewKeyDecoder(os.Stdin)
//	for {
//	    event, err := decoder.ReadEvent()
//	    if err != nil {
//	        if err == io.EOF {
//	            break
//	        }
//	        log.Printf("Read error: %v", err)
//	        continue
//	    }
//
//	    switch e := event.(type) {
//	    case terminal.KeyEvent:
//	        // Handle keyboard input
//	    case terminal.MouseEvent:
//	        // Handle mouse input
//	    }
//	}
//
// # Buffering
//
// The decoder uses internal buffering to avoid consuming more bytes than necessary.
// This ensures that each ReadEvent call reads exactly one complete event, making it
// safe to interleave with other input operations if needed.
//
// # Thread Safety
//
// KeyDecoder is NOT thread-safe. Only one goroutine should call ReadEvent at a time.
type KeyDecoder struct {
	reader        *bufio.Reader
	pasteTabWidth int // 0 = preserve tabs, >0 = convert to this many spaces
}

// NewKeyDecoder creates a new KeyDecoder that reads from the given input stream.
//
// For production use, pass os.Stdin:
//
//	decoder := terminal.NewKeyDecoder(os.Stdin)
//
// For testing, pass a bytes.Buffer or strings.Reader:
//
//	input := strings.NewReader("\x1b[A") // Up arrow
//	decoder := terminal.NewKeyDecoder(input)
//
// The decoder must be paired with a terminal in raw mode to receive
// input character-by-character rather than line-by-line.
func NewKeyDecoder(reader io.Reader) *KeyDecoder {
	return &KeyDecoder{
		reader:        bufio.NewReader(reader),
		pasteTabWidth: 0, // Default: preserve tabs
	}
}

// SetPasteTabWidth configures how tabs in pasted content are handled.
// If width is 0 (default), tabs are preserved as-is.
// If width > 0, each tab is converted to that many spaces.
func (kd *KeyDecoder) SetPasteTabWidth(width int) {
	kd.pasteTabWidth = width
}

// ReadEvent reads a single input event from the input stream.
// It returns either a KeyEvent or MouseEvent, both implementing the Event interface.
// This is the recommended method for reading input when mouse support is needed.
func (kd *KeyDecoder) ReadEvent() (Event, error) {
	// Read first byte
	firstByte, err := kd.reader.ReadByte()
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// Check the first byte to see what we're dealing with
	switch firstByte {
	// Control characters
	case 0x0D: // CR - normal Enter
		return KeyEvent{Key: KeyEnter}, nil
	case 0x0A: // LF - Shift+Enter (via terminal key binding like iTerm2)
		return KeyEvent{Key: KeyEnter, Shift: true}, nil
	case 0x09: // Tab
		return KeyEvent{Key: KeyTab}, nil
	case 0x7F, 0x08: // Backspace (DEL or BS)
		return KeyEvent{Key: KeyBackspace}, nil
	case 0x1B: // Escape (might be start of sequence or mouse event)
		return kd.handleEscapeEvent()

	// Ctrl combinations (0x01-0x1A map to Ctrl+A through Ctrl+Z)
	case 0x01:
		return KeyEvent{Key: KeyCtrlA, Ctrl: true}, nil
	case 0x02:
		return KeyEvent{Key: KeyCtrlB, Ctrl: true}, nil
	case 0x03:
		return KeyEvent{Key: KeyCtrlC, Ctrl: true}, nil
	case 0x04:
		return KeyEvent{Key: KeyCtrlD, Ctrl: true}, nil
	case 0x05:
		return KeyEvent{Key: KeyCtrlE, Ctrl: true}, nil
	case 0x06:
		return KeyEvent{Key: KeyCtrlF, Ctrl: true}, nil
	case 0x07:
		return KeyEvent{Key: KeyCtrlG, Ctrl: true}, nil
	case 0x0B:
		return KeyEvent{Key: KeyCtrlK, Ctrl: true}, nil
	case 0x0C:
		return KeyEvent{Key: KeyCtrlL, Ctrl: true}, nil
	case 0x0E:
		return KeyEvent{Key: KeyCtrlN, Ctrl: true}, nil
	case 0x0F:
		return KeyEvent{Key: KeyCtrlO, Ctrl: true}, nil
	case 0x10:
		return KeyEvent{Key: KeyCtrlP, Ctrl: true}, nil
	case 0x11:
		return KeyEvent{Key: KeyCtrlQ, Ctrl: true}, nil
	case 0x12:
		return KeyEvent{Key: KeyCtrlR, Ctrl: true}, nil
	case 0x13:
		return KeyEvent{Key: KeyCtrlS, Ctrl: true}, nil
	case 0x14:
		return KeyEvent{Key: KeyCtrlT, Ctrl: true}, nil
	case 0x15:
		return KeyEvent{Key: KeyCtrlU, Ctrl: true}, nil
	case 0x16:
		return KeyEvent{Key: KeyCtrlV, Ctrl: true}, nil
	case 0x17:
		return KeyEvent{Key: KeyCtrlW, Ctrl: true}, nil
	case 0x18:
		return KeyEvent{Key: KeyCtrlX, Ctrl: true}, nil
	case 0x19:
		return KeyEvent{Key: KeyCtrlY, Ctrl: true}, nil
	case 0x1A:
		return KeyEvent{Key: KeyCtrlZ, Ctrl: true}, nil

	// Printable ASCII
	default:
		if firstByte >= 0x20 && firstByte < 0x7F {
			return KeyEvent{Rune: rune(firstByte)}, nil
		}
		// Might be start of UTF-8 multi-byte character
		keyEvent, err := kd.decodeUTF8(firstByte)
		return keyEvent, err
	}
}

// handleEscapeEvent processes an escape key or escape sequence, including mouse events
// We've already consumed the ESC byte (0x1B)
func (kd *KeyDecoder) handleEscapeEvent() (Event, error) {
	// Check if there are more bytes already buffered.
	// Escape sequences arrive as a rapid burst, so if the terminal sent an escape
	// sequence, the following bytes should already be in the buffer.
	// If nothing is buffered, this is a standalone Escape key press.
	if kd.reader.Buffered() == 0 {
		return KeyEvent{Key: KeyEscape}, nil
	}

	// Peek at the next byte to see if this is an escape sequence
	nextByte, err := kd.reader.ReadByte()
	if err != nil {
		// No more bytes available, it's just the Escape key
		return KeyEvent{Key: KeyEscape}, nil
	}

	// Check what follows the ESC
	switch nextByte {
	case '[':
		// ANSI CSI sequence: ESC [ - could be keyboard or mouse
		return kd.decodeCSIEvent()
	case 'O':
		// ANSI SS3 sequence: ESC O
		keyEvent, err := kd.decodeSS3()
		return keyEvent, err
	case 0x0D, 0x0A:
		// ESC + Enter (CR or LF) = Shift+Enter (iTerm2 and similar terminals)
		return KeyEvent{Key: KeyEnter, Shift: true}, nil
	default:
		// Alt+key combination or unknown
		if nextByte >= 0x20 && nextByte < 0x7F {
			// Printable character with Alt
			return KeyEvent{Rune: rune(nextByte), Alt: true}, nil
		}
		// Unknown sequence, return escape
		kd.reader.UnreadByte()
		return KeyEvent{Key: KeyEscape}, nil
	}
}

// decodeCSIEvent decodes ANSI CSI sequences (ESC [ ...), including mouse events
// We've already consumed ESC and '['
func (kd *KeyDecoder) decodeCSIEvent() (Event, error) {
	first, err := kd.reader.ReadByte()
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// Legacy mouse format: ESC [ M followed by 3 raw bytes (the payload
	// bytes are not CSI parameter bytes, so handle this before the
	// generic sequence reader).
	if first == 'M' {
		return kd.decodeLegacyMouseEvent()
	}

	params, final, err := kd.readCSISequence(first)
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// SGR mouse event: ESC [ < button ; x ; y [Mm]
	if len(params) > 0 && params[0] == '<' && (final == 'M' || final == 'm') {
		event, err := ParseMouseEvent(append([]byte(params), final))
		if err != nil {
			return KeyEvent{Key: KeyUnknown}, err
		}
		return *event, nil
	}

	return kd.decodeCSIKey(params, final)
}

// readCSISequence reads the remainder of a CSI sequence through its final
// byte (0x40-0x7E), starting from the already-consumed byte first. It
// returns the parameter/intermediate bytes and the final byte. Per ECMA-48
// every byte in 0x20-0x3F before the final byte belongs to the sequence, so
// multi-digit and multi-part parameters are consumed without leaving stray
// bytes in the stream to be misread as typed keys.
//
// A final byte of 0 indicates a malformed sequence (embedded control byte
// or runaway length).
func (kd *KeyDecoder) readCSISequence(first byte) (string, byte, error) {
	const maxCSILength = 64 // generous bound; real sequences are much shorter
	var params []byte
	b := first
	for {
		if b >= 0x40 && b <= 0x7E {
			return string(params), b, nil
		}
		if b < 0x20 {
			// Control byte inside a CSI sequence: malformed. Put it back so
			// it can be processed as the start of fresh input.
			kd.reader.UnreadByte()
			return string(params), 0, nil
		}
		if len(params) >= maxCSILength {
			return string(params), 0, nil
		}
		params = append(params, b)
		var err error
		b, err = kd.reader.ReadByte()
		if err != nil {
			return string(params), 0, err
		}
	}
}

// decodeCSIKey interprets a fully-read CSI sequence as a key event.
func (kd *KeyDecoder) decodeCSIKey(params string, final byte) (KeyEvent, error) {
	parts := strings.Split(params, ";")
	num := csiPrimary(parts[0])
	modifier := 1
	if len(parts) >= 2 {
		if m, err := strconv.Atoi(csiPrimary(parts[1])); err == nil && m > 0 {
			modifier = m
		}
	}

	switch final {
	case 'A':
		return applyKeyModifiers(KeyEvent{Key: KeyArrowUp}, modifier), nil
	case 'B':
		return applyKeyModifiers(KeyEvent{Key: KeyArrowDown}, modifier), nil
	case 'C':
		return applyKeyModifiers(KeyEvent{Key: KeyArrowRight}, modifier), nil
	case 'D':
		return applyKeyModifiers(KeyEvent{Key: KeyArrowLeft}, modifier), nil
	case 'H':
		return applyKeyModifiers(KeyEvent{Key: KeyHome}, modifier), nil
	case 'F':
		return applyKeyModifiers(KeyEvent{Key: KeyEnd}, modifier), nil
	case 'Z':
		// Shift+Tab (also known as BackTab), optionally with modifiers
		if params == "" || num == "" || num == "1" {
			event := applyKeyModifiers(KeyEvent{Key: KeyTab}, modifier)
			event.Shift = true
			return event, nil
		}
		return KeyEvent{Key: KeyUnknown}, nil
	case 'u':
		// Kitty keyboard protocol: ESC [ codepoint ; modifier u
		return kd.decodeCSIu(num, modifier)
	case '~':
		if num == "200" {
			// Bracketed paste start
			return kd.decodeBracketedPaste()
		}
		if num == "27" && len(parts) >= 3 {
			// xterm modifyOtherKeys: ESC [ 27 ; modifier ; codepoint ~
			return kd.decodeCSIu(csiPrimary(parts[2]), modifier)
		}
		event, err := kd.decodeCSINumber(num)
		if err != nil || event.Key == KeyUnknown {
			return event, err
		}
		return applyKeyModifiers(event, modifier), nil
	}

	// Anything else (cursor position reports, device attributes, ...) is not
	// a key; the sequence has been fully consumed so the stream stays in sync.
	return KeyEvent{Key: KeyUnknown}, nil
}

// csiPrimary returns the primary value of a CSI parameter, stripping any
// colon-separated sub-parameters (e.g. Kitty's "shifted:base" key codes or
// "modifier:event_type" fields).
func csiPrimary(part string) string {
	if idx := strings.IndexByte(part, ':'); idx >= 0 {
		return part[:idx]
	}
	return part
}

// applyKeyModifiers applies an xterm/kitty modifier parameter to a key event.
// The encoding is modifier = 1 + bitfield (1=Shift, 2=Alt, 4=Ctrl; higher
// bits such as Super or CapsLock are ignored).
func applyKeyModifiers(event KeyEvent, modifier int) KeyEvent {
	if modifier <= 1 {
		return event
	}
	bits := modifier - 1
	event.Shift = event.Shift || bits&1 != 0
	event.Alt = event.Alt || bits&2 != 0
	event.Ctrl = event.Ctrl || bits&4 != 0
	// Normalize Ctrl+Enter and Alt+Enter to also set Shift, so apps can just
	// check Shift for "soft newline".
	if event.Key == KeyEnter && (event.Ctrl || event.Alt) {
		event.Shift = true
	}
	return event
}

// decodeLegacyMouseEvent decodes legacy mouse format: ESC [ M followed by 3 bytes
// We've already consumed ESC [ M
func (kd *KeyDecoder) decodeLegacyMouseEvent() (Event, error) {
	// Read the 3 bytes
	buf := make([]byte, 3)
	for i := 0; i < 3; i++ {
		b, err := kd.reader.ReadByte()
		if err != nil {
			return KeyEvent{Key: KeyUnknown}, err
		}
		buf[i] = b
	}

	event, err := ParseMouseEvent(buf)
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}
	return *event, nil
}

// ReadKeyEvent reads a single key event from the input stream.
// It blocks until a complete key sequence is available.
//
// Returns:
//   - KeyEvent with either a special Key or a Rune set
//   - error if read fails (io.EOF, closed pipe, etc.)
//
// The function handles:
//   - Single-byte special keys (Enter, Tab, Backspace, Ctrl+Letter)
//   - Multi-byte escape sequences (arrows, function keys, Home/End, etc.)
//   - UTF-8 multi-byte characters
//   - Alt modifier (ESC followed by character)
func (kd *KeyDecoder) ReadKeyEvent() (KeyEvent, error) {
	// Read first byte
	firstByte, err := kd.reader.ReadByte()
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// Check the first byte to see what we're dealing with
	switch firstByte {
	// Control characters
	case 0x0D: // CR - normal Enter
		return KeyEvent{Key: KeyEnter}, nil
	case 0x0A: // LF - Shift+Enter (via terminal key binding like iTerm2)
		return KeyEvent{Key: KeyEnter, Shift: true}, nil
	case 0x09: // Tab
		return KeyEvent{Key: KeyTab}, nil
	case 0x7F, 0x08: // Backspace (DEL or BS)
		return KeyEvent{Key: KeyBackspace}, nil
	case 0x1B: // Escape (might be start of sequence)
		return kd.handleEscape()

	// Ctrl combinations (0x01-0x1A map to Ctrl+A through Ctrl+Z)
	case 0x01:
		return KeyEvent{Key: KeyCtrlA, Ctrl: true}, nil
	case 0x02:
		return KeyEvent{Key: KeyCtrlB, Ctrl: true}, nil
	case 0x03:
		return KeyEvent{Key: KeyCtrlC, Ctrl: true}, nil
	case 0x04:
		return KeyEvent{Key: KeyCtrlD, Ctrl: true}, nil
	case 0x05:
		return KeyEvent{Key: KeyCtrlE, Ctrl: true}, nil
	case 0x06:
		return KeyEvent{Key: KeyCtrlF, Ctrl: true}, nil
	case 0x07:
		return KeyEvent{Key: KeyCtrlG, Ctrl: true}, nil
	case 0x0B:
		return KeyEvent{Key: KeyCtrlK, Ctrl: true}, nil
	case 0x0C:
		return KeyEvent{Key: KeyCtrlL, Ctrl: true}, nil
	case 0x0E:
		return KeyEvent{Key: KeyCtrlN, Ctrl: true}, nil
	case 0x0F:
		return KeyEvent{Key: KeyCtrlO, Ctrl: true}, nil
	case 0x10:
		return KeyEvent{Key: KeyCtrlP, Ctrl: true}, nil
	case 0x11:
		return KeyEvent{Key: KeyCtrlQ, Ctrl: true}, nil
	case 0x12:
		return KeyEvent{Key: KeyCtrlR, Ctrl: true}, nil
	case 0x13:
		return KeyEvent{Key: KeyCtrlS, Ctrl: true}, nil
	case 0x14:
		return KeyEvent{Key: KeyCtrlT, Ctrl: true}, nil
	case 0x15:
		return KeyEvent{Key: KeyCtrlU, Ctrl: true}, nil
	case 0x16:
		return KeyEvent{Key: KeyCtrlV, Ctrl: true}, nil
	case 0x17:
		return KeyEvent{Key: KeyCtrlW, Ctrl: true}, nil
	case 0x18:
		return KeyEvent{Key: KeyCtrlX, Ctrl: true}, nil
	case 0x19:
		return KeyEvent{Key: KeyCtrlY, Ctrl: true}, nil
	case 0x1A:
		return KeyEvent{Key: KeyCtrlZ, Ctrl: true}, nil

	// Printable ASCII
	default:
		if firstByte >= 0x20 && firstByte < 0x7F {
			return KeyEvent{Rune: rune(firstByte)}, nil
		}
		// Might be start of UTF-8 multi-byte character
		return kd.decodeUTF8(firstByte)
	}
}

// handleEscape processes an escape key or escape sequence
// We've already consumed the ESC byte (0x1B)
func (kd *KeyDecoder) handleEscape() (KeyEvent, error) {
	// Check if there are more bytes already buffered.
	// Escape sequences arrive as a rapid burst, so if the terminal sent an escape
	// sequence, the following bytes should already be in the buffer.
	// If nothing is buffered, this is a standalone Escape key press.
	if kd.reader.Buffered() == 0 {
		return KeyEvent{Key: KeyEscape}, nil
	}

	// Peek at the next byte to see if this is an escape sequence
	nextByte, err := kd.reader.ReadByte()
	if err != nil {
		// No more bytes available, it's just the Escape key
		return KeyEvent{Key: KeyEscape}, nil
	}

	// Check what follows the ESC
	switch nextByte {
	case '[':
		// ANSI CSI sequence: ESC [
		return kd.decodeCSI()
	case 'O':
		// ANSI SS3 sequence: ESC O
		return kd.decodeSS3()
	case 0x0D, 0x0A:
		// ESC + Enter (CR or LF) = Shift+Enter (iTerm2 and similar terminals)
		return KeyEvent{Key: KeyEnter, Shift: true}, nil
	default:
		// Alt+key combination or unknown
		if nextByte >= 0x20 && nextByte < 0x7F {
			// Printable character with Alt
			return KeyEvent{Rune: rune(nextByte), Alt: true}, nil
		}
		// Unknown sequence, return escape
		kd.reader.UnreadByte()
		return KeyEvent{Key: KeyEscape}, nil
	}
}

// decodeCSI decodes ANSI CSI sequences (ESC [ ...)
// We've already consumed ESC and '['
func (kd *KeyDecoder) decodeCSI() (KeyEvent, error) {
	first, err := kd.reader.ReadByte()
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// Legacy mouse report on the key-only API: consume the 3 raw payload
	// bytes to keep the stream in sync, then report unknown.
	if first == 'M' {
		for i := 0; i < 3; i++ {
			if _, err := kd.reader.ReadByte(); err != nil {
				return KeyEvent{Key: KeyUnknown}, err
			}
		}
		return KeyEvent{Key: KeyUnknown}, nil
	}

	params, final, err := kd.readCSISequence(first)
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// SGR mouse report on the key-only API: fully consumed, report unknown.
	if len(params) > 0 && params[0] == '<' && (final == 'M' || final == 'm') {
		return KeyEvent{Key: KeyUnknown}, nil
	}

	return kd.decodeCSIKey(params, final)
}

// decodeCSINumber decodes CSI sequences ending with ~ (e.g., ESC [ 3 ~)
func (kd *KeyDecoder) decodeCSINumber(num string) (KeyEvent, error) {
	switch num {
	case "1":
		return KeyEvent{Key: KeyHome}, nil
	case "2":
		return KeyEvent{Key: KeyInsert}, nil
	case "3":
		return KeyEvent{Key: KeyDelete}, nil
	case "4":
		return KeyEvent{Key: KeyEnd}, nil
	case "5":
		return KeyEvent{Key: KeyPageUp}, nil
	case "6":
		return KeyEvent{Key: KeyPageDown}, nil
	case "11":
		return KeyEvent{Key: KeyF1}, nil
	case "12":
		return KeyEvent{Key: KeyF2}, nil
	case "13":
		return KeyEvent{Key: KeyF3}, nil
	case "14":
		return KeyEvent{Key: KeyF4}, nil
	case "15":
		return KeyEvent{Key: KeyF5}, nil
	case "17":
		return KeyEvent{Key: KeyF6}, nil
	case "18":
		return KeyEvent{Key: KeyF7}, nil
	case "19":
		return KeyEvent{Key: KeyF8}, nil
	case "20":
		return KeyEvent{Key: KeyF9}, nil
	case "21":
		return KeyEvent{Key: KeyF10}, nil
	case "23":
		return KeyEvent{Key: KeyF11}, nil
	case "24":
		return KeyEvent{Key: KeyF12}, nil
	default:
		return KeyEvent{Key: KeyUnknown}, nil
	}
}

// decodeCSIu decodes CSI u sequences (Kitty keyboard protocol) and
// xterm modifyOtherKeys codepoints.
// Format: ESC [ codepoint [; modifier] u
// Codepoint is the Unicode code point of the key.
// Modifier is 1 + bitfield: 1=Shift, 2=Alt, 4=Ctrl (higher bits ignored).
func (kd *KeyDecoder) decodeCSIu(codepoint string, modifier int) (KeyEvent, error) {
	// Decode modifiers
	var shift, alt, ctrl bool
	if modifier > 1 {
		bits := modifier - 1
		shift = bits&1 != 0
		alt = bits&2 != 0
		ctrl = bits&4 != 0
	}

	// Handle special keys by codepoint
	switch codepoint {
	case "9":
		return KeyEvent{Key: KeyTab, Shift: shift, Alt: alt, Ctrl: ctrl}, nil
	case "13":
		event := KeyEvent{Key: KeyEnter, Shift: shift, Alt: alt, Ctrl: ctrl}
		// Normalize Ctrl+Enter and Alt+Enter to also set Shift
		if ctrl || alt {
			event.Shift = true
		}
		return event, nil
	case "27":
		return KeyEvent{Key: KeyEscape, Shift: shift, Alt: alt, Ctrl: ctrl}, nil
	case "127":
		return KeyEvent{Key: KeyBackspace, Shift: shift, Alt: alt, Ctrl: ctrl}, nil

	// Ctrl+letter combinations (codepoints 1-26 map to Ctrl+A through Ctrl+Z)
	case "1":
		return KeyEvent{Key: KeyCtrlA, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "2":
		return KeyEvent{Key: KeyCtrlB, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "3":
		return KeyEvent{Key: KeyCtrlC, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "4":
		return KeyEvent{Key: KeyCtrlD, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "5":
		return KeyEvent{Key: KeyCtrlE, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "6":
		return KeyEvent{Key: KeyCtrlF, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "7":
		return KeyEvent{Key: KeyCtrlG, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "8":
		return KeyEvent{Key: KeyBackspace, Ctrl: true, Shift: shift, Alt: alt}, nil // Ctrl+H = Backspace
	case "10":
		// LF - treat as Shift+Enter
		return KeyEvent{Key: KeyEnter, Shift: true, Alt: alt, Ctrl: ctrl}, nil
	case "11":
		return KeyEvent{Key: KeyCtrlK, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "12":
		return KeyEvent{Key: KeyCtrlL, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "14":
		return KeyEvent{Key: KeyCtrlN, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "15":
		return KeyEvent{Key: KeyCtrlO, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "16":
		return KeyEvent{Key: KeyCtrlP, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "17":
		return KeyEvent{Key: KeyCtrlQ, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "18":
		return KeyEvent{Key: KeyCtrlR, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "19":
		return KeyEvent{Key: KeyCtrlS, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "20":
		return KeyEvent{Key: KeyCtrlT, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "21":
		return KeyEvent{Key: KeyCtrlU, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "22":
		return KeyEvent{Key: KeyCtrlV, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "23":
		return KeyEvent{Key: KeyCtrlW, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "24":
		return KeyEvent{Key: KeyCtrlX, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "25":
		return KeyEvent{Key: KeyCtrlY, Ctrl: true, Shift: shift, Alt: alt}, nil
	case "26":
		return KeyEvent{Key: KeyCtrlZ, Ctrl: true, Shift: shift, Alt: alt}, nil
	}

	// For other codepoints, try to parse as a number and return as a rune
	if cp, err := strconv.Atoi(codepoint); err == nil && cp >= 32 && cp < 0x10FFFF {
		// If Ctrl is pressed with a letter, return the appropriate KeyCtrl* constant
		// This handles Kitty protocol sending Ctrl+C as codepoint 99 ('c') with Ctrl modifier
		if ctrl && cp >= 'a' && cp <= 'z' {
			return kd.ctrlLetterEvent(rune(cp), shift, alt)
		}
		if ctrl && cp >= 'A' && cp <= 'Z' {
			return kd.ctrlLetterEvent(rune(cp-'A'+'a'), shift, alt) // normalize to lowercase
		}
		return KeyEvent{Rune: rune(cp), Shift: shift, Alt: alt, Ctrl: ctrl}, nil
	}

	return KeyEvent{Key: KeyUnknown}, nil
}

// ctrlLetterEvent returns a KeyEvent for Ctrl+letter combinations
func (kd *KeyDecoder) ctrlLetterEvent(letter rune, shift, alt bool) (KeyEvent, error) {
	var key Key
	switch letter {
	case 'a':
		key = KeyCtrlA
	case 'b':
		key = KeyCtrlB
	case 'c':
		key = KeyCtrlC
	case 'd':
		key = KeyCtrlD
	case 'e':
		key = KeyCtrlE
	case 'f':
		key = KeyCtrlF
	case 'g':
		key = KeyCtrlG
	case 'h':
		key = KeyCtrlH
	case 'i':
		key = KeyCtrlI
	case 'j':
		key = KeyCtrlJ
	case 'k':
		key = KeyCtrlK
	case 'l':
		key = KeyCtrlL
	case 'm':
		key = KeyCtrlM
	case 'n':
		key = KeyCtrlN
	case 'o':
		key = KeyCtrlO
	case 'p':
		key = KeyCtrlP
	case 'q':
		key = KeyCtrlQ
	case 'r':
		key = KeyCtrlR
	case 's':
		key = KeyCtrlS
	case 't':
		key = KeyCtrlT
	case 'u':
		key = KeyCtrlU
	case 'v':
		key = KeyCtrlV
	case 'w':
		key = KeyCtrlW
	case 'x':
		key = KeyCtrlX
	case 'y':
		key = KeyCtrlY
	case 'z':
		key = KeyCtrlZ
	default:
		return KeyEvent{Rune: letter, Ctrl: true, Shift: shift, Alt: alt}, nil
	}
	return KeyEvent{Key: key, Ctrl: true, Shift: shift, Alt: alt}, nil
}

// decodeSS3 decodes ANSI SS3 sequences (ESC O ...)
// We've already consumed ESC and 'O'
func (kd *KeyDecoder) decodeSS3() (KeyEvent, error) {
	ch, err := kd.reader.ReadByte()
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	switch ch {
	case 'P':
		return KeyEvent{Key: KeyF1}, nil
	case 'Q':
		return KeyEvent{Key: KeyF2}, nil
	case 'R':
		return KeyEvent{Key: KeyF3}, nil
	case 'S':
		return KeyEvent{Key: KeyF4}, nil
	case 'H':
		return KeyEvent{Key: KeyHome}, nil
	case 'F':
		return KeyEvent{Key: KeyEnd}, nil
	default:
		return KeyEvent{Key: KeyUnknown}, nil
	}
}

// decodeUTF8 decodes a multi-byte UTF-8 character
// We've already read the first byte
func (kd *KeyDecoder) decodeUTF8(firstByte byte) (KeyEvent, error) {
	// Determine how many bytes we need
	var numBytes int
	if firstByte&0x80 == 0 {
		// Single-byte ASCII (should have been handled earlier)
		return KeyEvent{Rune: rune(firstByte)}, nil
	} else if firstByte&0xE0 == 0xC0 {
		numBytes = 2
	} else if firstByte&0xF0 == 0xE0 {
		numBytes = 3
	} else if firstByte&0xF8 == 0xF0 {
		numBytes = 4
	} else {
		// Invalid UTF-8
		return KeyEvent{Key: KeyUnknown}, nil
	}

	// Read the remaining bytes
	buf := make([]byte, numBytes)
	buf[0] = firstByte
	for i := 1; i < numBytes; i++ {
		b, err := kd.reader.ReadByte()
		if err != nil {
			return KeyEvent{Key: KeyUnknown}, err
		}
		buf[i] = b
	}

	// Decode the UTF-8 sequence
	r, _ := utf8.DecodeRune(buf)
	if r == utf8.RuneError {
		return KeyEvent{Key: KeyUnknown}, nil
	}

	return KeyEvent{Rune: r}, nil
}

// normalizePasteContent normalizes pasted content:
// - Converts \r\n (Windows) and \r (old Mac) line endings to \n
// - Optionally converts tabs to spaces based on tabWidth (0 = preserve tabs)
func normalizePasteContent(s string, tabWidth int) string {
	// First replace \r\n (Windows) with \n
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Then replace any remaining \r (old Mac) with \n
	s = strings.ReplaceAll(s, "\r", "\n")
	// Convert tabs to spaces if configured
	if tabWidth > 0 {
		spaces := strings.Repeat(" ", tabWidth)
		s = strings.ReplaceAll(s, "\t", spaces)
	}
	return s
}

// decodeBracketedPaste decodes a bracketed paste sequence
// We've already consumed ESC [ 200 ~, now read until ESC [ 201 ~
func (kd *KeyDecoder) decodeBracketedPaste() (KeyEvent, error) {
	var content []byte

	// Read bytes until we find the end sequence: ESC [ 201 ~
	for {
		b, err := kd.reader.ReadByte()
		if err != nil {
			// EOF or error while reading paste content
			// Return what we have so far (with normalized line endings)
			return KeyEvent{Paste: normalizePasteContent(string(content), kd.pasteTabWidth)}, err
		}

		// Check if this might be the start of the end sequence
		if b == 0x1B { // ESC
			// Peek ahead to check for [ 2 0 1 ~
			next1, err1 := kd.reader.ReadByte()
			if err1 != nil {
				content = append(content, b)
				break
			}

			if next1 == '[' {
				next2, err2 := kd.reader.ReadByte()
				if err2 != nil {
					content = append(content, b, next1)
					break
				}

				if next2 == '2' {
					next3, err3 := kd.reader.ReadByte()
					if err3 != nil {
						content = append(content, b, next1, next2)
						break
					}

					if next3 == '0' {
						next4, err4 := kd.reader.ReadByte()
						if err4 != nil {
							content = append(content, b, next1, next2, next3)
							break
						}

						if next4 == '1' {
							next5, err5 := kd.reader.ReadByte()
							if err5 != nil {
								content = append(content, b, next1, next2, next3, next4)
								break
							}

							if next5 == '~' {
								// Found the end sequence!
								return KeyEvent{Paste: normalizePasteContent(string(content), kd.pasteTabWidth)}, nil
							}
							// Not the end sequence, add all bytes to content
							content = append(content, b, next1, next2, next3, next4, next5)
						} else {
							content = append(content, b, next1, next2, next3, next4)
						}
					} else {
						content = append(content, b, next1, next2, next3)
					}
				} else {
					content = append(content, b, next1, next2)
				}
			} else {
				content = append(content, b, next1)
			}
		} else {
			// Regular character, add to content
			content = append(content, b)
		}
	}

	// If we reach here, we hit EOF or error
	return KeyEvent{Paste: normalizePasteContent(string(content), kd.pasteTabWidth)}, nil
}
