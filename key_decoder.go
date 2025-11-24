package gooey

import (
	"bufio"
	"io"
	"unicode/utf8"
)

// KeyDecoder handles low-level key event decoding from a byte stream.
// It supports:
// - Multi-byte UTF-8 characters
// - ANSI escape sequences (arrows, function keys, etc.)
// - Alt/Meta modifiers
// - Ctrl combinations
// - Proper error handling and EOF detection
//
// The decoder uses internal buffering to ensure we only consume bytes
// for one key event at a time, making it suitable for sequential reads.
type KeyDecoder struct {
	reader *bufio.Reader
}

// NewKeyDecoder creates a new key decoder that reads from the given reader.
// For production use, pass os.Stdin. For testing, pass a bytes.Buffer or other io.Reader.
func NewKeyDecoder(reader io.Reader) *KeyDecoder {
	return &KeyDecoder{
		reader: bufio.NewReader(reader),
	}
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
	case 0x0D, 0x0A: // Enter (CR or LF)
		return KeyEvent{Key: KeyEnter}, nil
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
	// Read the next character
	ch, err := kd.reader.ReadByte()
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// Simple sequences: ESC [ A/B/C/D/H/F
	switch ch {
	case 'A':
		return KeyEvent{Key: KeyArrowUp}, nil
	case 'B':
		return KeyEvent{Key: KeyArrowDown}, nil
	case 'C':
		return KeyEvent{Key: KeyArrowRight}, nil
	case 'D':
		return KeyEvent{Key: KeyArrowLeft}, nil
	case 'H':
		return KeyEvent{Key: KeyHome}, nil
	case 'F':
		return KeyEvent{Key: KeyEnd}, nil
	}

	// Numeric sequences: ESC [ <number> ~  or  ESC [ <number> ; <modifier> <key>
	if ch >= '0' && ch <= '9' {
		// Read the full numeric sequence
		num := []byte{ch}
		for {
			b, err := kd.reader.ReadByte()
			if err != nil {
				return KeyEvent{Key: KeyUnknown}, err
			}
			if b == '~' {
				// Sequence ends with ~
				// Check for bracketed paste
				numStr := string(num)
				if numStr == "200" {
					// Bracketed paste start
					return kd.decodeBracketedPaste()
				}
				return kd.decodeCSINumber(numStr)
			}
			if b == ';' {
				// Modified key sequence
				return kd.decodeCSIModified(string(num))
			}
			if b >= '0' && b <= '9' {
				num = append(num, b)
			} else {
				// Unexpected character
				return KeyEvent{Key: KeyUnknown}, nil
			}
		}
	}

	return KeyEvent{Key: KeyUnknown}, nil
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

// decodeCSIModified decodes CSI sequences with modifiers (e.g., ESC [ 1 ; 5 A for Ctrl+Up)
func (kd *KeyDecoder) decodeCSIModified(num string) (KeyEvent, error) {
	// Read modifier number
	modByte, err := kd.reader.ReadByte()
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// Read the key code
	keyByte, err := kd.reader.ReadByte()
	if err != nil {
		return KeyEvent{Key: KeyUnknown}, err
	}

	// Decode the key
	var key Key
	switch keyByte {
	case 'A':
		key = KeyArrowUp
	case 'B':
		key = KeyArrowDown
	case 'C':
		key = KeyArrowRight
	case 'D':
		key = KeyArrowLeft
	case 'u':
		// CSI u sequence (e.g., ESC[13;2u for Shift+Enter in iTerm2)
		// num contains the key code
		switch num {
		case "13":
			key = KeyEnter
		case "9":
			key = KeyTab
		case "27":
			key = KeyEscape
		default:
			return KeyEvent{Key: KeyUnknown}, nil
		}
	case '~':
		// Modified key ending with ~ (e.g., ESC[3;5~ for Ctrl+Delete)
		switch num {
		case "3":
			key = KeyDelete
		case "5":
			key = KeyPageUp
		case "6":
			key = KeyPageDown
		default:
			return KeyEvent{Key: KeyUnknown}, nil
		}
	default:
		return KeyEvent{Key: KeyUnknown}, nil
	}

	event := KeyEvent{Key: key}

	// Decode modifiers (2=Shift, 3=Alt, 4=Shift+Alt, 5=Ctrl, 6=Shift+Ctrl, 7=Alt+Ctrl, 8=Shift+Alt+Ctrl)
	if modByte == '2' || modByte == '4' || modByte == '6' || modByte == '8' {
		event.Shift = true
	}
	if modByte == '3' || modByte == '4' || modByte == '7' || modByte == '8' {
		event.Alt = true
	}
	if modByte == '5' || modByte == '6' || modByte == '7' || modByte == '8' {
		event.Ctrl = true
	}

	return event, nil
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

// decodeBracketedPaste decodes a bracketed paste sequence
// We've already consumed ESC [ 200 ~, now read until ESC [ 201 ~
func (kd *KeyDecoder) decodeBracketedPaste() (KeyEvent, error) {
	var content []byte

	// Read bytes until we find the end sequence: ESC [ 201 ~
	for {
		b, err := kd.reader.ReadByte()
		if err != nil {
			// EOF or error while reading paste content
			// Return what we have so far
			return KeyEvent{Paste: string(content)}, err
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
								return KeyEvent{Paste: string(content)}, nil
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
	return KeyEvent{Paste: string(content)}, nil
}
