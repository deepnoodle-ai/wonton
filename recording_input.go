package gooey

// keyEventToString converts a KeyEvent to a string representation for recording
// This represents the actual key pressed in a format that could be replayed
func keyEventToString(event KeyEvent) string {
	// If it's a printable rune, just return it
	if event.Rune != 0 {
		return string(event.Rune)
	}

	// Handle special keys by converting them to their terminal escape sequences
	switch event.Key {
	case KeyEnter:
		return "\r"
	case KeyTab:
		return "\t"
	case KeyBackspace:
		return "\x7f"
	case KeyEscape:
		return "\x1b"

	// Arrow keys
	case KeyArrowUp:
		return "\x1b[A"
	case KeyArrowDown:
		return "\x1b[B"
	case KeyArrowRight:
		return "\x1b[C"
	case KeyArrowLeft:
		return "\x1b[D"

	// Home/End
	case KeyHome:
		return "\x1b[H"
	case KeyEnd:
		return "\x1b[F"

	// Page Up/Down
	case KeyPageUp:
		return "\x1b[5~"
	case KeyPageDown:
		return "\x1b[6~"

	// Insert/Delete
	case KeyInsert:
		return "\x1b[2~"
	case KeyDelete:
		return "\x1b[3~"

	// Function keys
	case KeyF1:
		return "\x1bOP"
	case KeyF2:
		return "\x1bOQ"
	case KeyF3:
		return "\x1bOR"
	case KeyF4:
		return "\x1bOS"
	case KeyF5:
		return "\x1b[15~"
	case KeyF6:
		return "\x1b[17~"
	case KeyF7:
		return "\x1b[18~"
	case KeyF8:
		return "\x1b[19~"
	case KeyF9:
		return "\x1b[20~"
	case KeyF10:
		return "\x1b[21~"
	case KeyF11:
		return "\x1b[23~"
	case KeyF12:
		return "\x1b[24~"

	// Ctrl combinations
	case KeyCtrlA:
		return "\x01"
	case KeyCtrlB:
		return "\x02"
	case KeyCtrlC:
		return "\x03"
	case KeyCtrlD:
		return "\x04"
	case KeyCtrlE:
		return "\x05"
	case KeyCtrlF:
		return "\x06"
	case KeyCtrlG:
		return "\x07"
	case KeyCtrlK:
		return "\x0b"
	case KeyCtrlL:
		return "\x0c"
	case KeyCtrlN:
		return "\x0e"
	case KeyCtrlP:
		return "\x10"
	case KeyCtrlR:
		return "\x12"
	case KeyCtrlT:
		return "\x14"
	case KeyCtrlU:
		return "\x15"
	case KeyCtrlW:
		return "\x17"
	case KeyCtrlY:
		return "\x19"
	case KeyCtrlZ:
		return "\x1a"

	default:
		// Unknown or unsupported key - return empty string
		return ""
	}
}
