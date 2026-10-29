package tmux

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/deepnoodle-ai/wonton/terminal"
)

// KeyName returns the tmux key name that types ev with send-keys ("Enter",
// "C-a", "M-Left", "BTab"), or, when literal is true, text to send with
// send-keys -l. ok is false for an event that types nothing.
//
// Sending a key by name rather than by its bytes lets tmux encode it for the
// pane's current modes: the cursor keys in application mode, and keys with
// modifiers in the extended form a program has asked for.
func KeyName(ev terminal.KeyEvent) (name string, literal bool, ok bool) {
	if ev.Paste != "" {
		return ev.Paste, true, true
	}
	var mods strings.Builder
	if ev.Ctrl {
		mods.WriteString("C-")
	}
	if ev.Alt {
		mods.WriteString("M-")
	}
	if ev.Shift {
		mods.WriteString("S-")
	}
	prefix := mods.String()

	var base string
	switch ev.Key {
	case terminal.KeyUnknown:
		if ev.Rune == 0 {
			return "", false, false
		}
		if !ev.Ctrl && !ev.Alt {
			return string(ev.Rune), true, true
		}
		// The rune's case already says whether Shift was held.
		prefix = strings.Replace(prefix, "S-", "", 1)
		r := ev.Rune
		if r == ' ' {
			return prefix + "Space", false, true
		}
		if ev.Ctrl {
			r = unicode.ToLower(r)
		}
		return prefix + string(r), false, true
	case terminal.KeyEnter:
		base = "Enter"
	case terminal.KeyTab:
		if ev.Shift {
			return strings.Replace(prefix, "S-", "", 1) + "BTab", false, true
		}
		base = "Tab"
	case terminal.KeyBackspace:
		base = "BSpace"
	case terminal.KeyEscape:
		base = "Escape"
	case terminal.KeyArrowUp:
		base = "Up"
	case terminal.KeyArrowDown:
		base = "Down"
	case terminal.KeyArrowLeft:
		base = "Left"
	case terminal.KeyArrowRight:
		base = "Right"
	case terminal.KeyHome:
		base = "Home"
	case terminal.KeyEnd:
		base = "End"
	case terminal.KeyPageUp:
		base = "PPage"
	case terminal.KeyPageDown:
		base = "NPage"
	case terminal.KeyDelete:
		base = "DC"
	case terminal.KeyInsert:
		base = "IC"
	default:
		switch {
		case ev.Key >= terminal.KeyF1 && ev.Key <= terminal.KeyF12:
			base = fmt.Sprintf("F%d", int(ev.Key-terminal.KeyF1)+1)
		case ev.Key >= terminal.KeyCtrlA && ev.Key <= terminal.KeyCtrlZ:
			// The key itself carries Ctrl; only Alt can be added to it.
			alt := ""
			if ev.Alt {
				alt = "M-"
			}
			return alt + "C-" + string(rune('a'+int(ev.Key-terminal.KeyCtrlA))), false, true
		default:
			return "", false, false
		}
	}
	return prefix + base, false, true
}

// MouseSequence encodes ev as the bytes a terminal sends a program that has
// turned on mouse reporting, at the pane-local cell (x, y). sgr selects SGR
// (1006) encoding over the legacy X10 form. ok is false for an event a
// program would not be sent, such as a click synthesized from a press and
// release that were already sent.
func MouseSequence(ev terminal.MouseEvent, x, y int, sgr bool) (seq string, ok bool) {
	var code int
	release := false
	switch ev.Button {
	case terminal.MouseButtonLeft:
		code = 0
	case terminal.MouseButtonMiddle:
		code = 1
	case terminal.MouseButtonRight:
		code = 2
	case terminal.MouseButtonWheelUp:
		code = 64
	case terminal.MouseButtonWheelDown:
		code = 65
	case terminal.MouseButtonWheelLeft:
		code = 66
	case terminal.MouseButtonWheelRight:
		code = 67
	case terminal.MouseButtonNone:
		code = 3 // motion with no button held
	default:
		return "", false
	}
	switch ev.Type {
	case terminal.MousePress, terminal.MouseScroll:
	case terminal.MouseRelease:
		release = true
	case terminal.MouseDrag, terminal.MouseMove:
		code += 32
	default:
		return "", false
	}
	if ev.Modifiers&terminal.ModShift != 0 {
		code += 4
	}
	if ev.Modifiers&terminal.ModAlt != 0 || ev.Modifiers&terminal.ModMeta != 0 {
		code += 8
	}
	if ev.Modifiers&terminal.ModCtrl != 0 {
		code += 16
	}
	if sgr {
		final := 'M'
		if release {
			final = 'm'
		}
		return fmt.Sprintf("\x1b[<%d;%d;%d%c", code, x+1, y+1, final), true
	}
	if release {
		code = 3 | (code &^ 3)
	}
	// X10 coordinates are single bytes offset by 32, so they top out at 223.
	if x+1 > 223 || y+1 > 223 {
		return "", false
	}
	return string([]byte{0x1b, '[', 'M', byte(32 + code), byte(32 + x + 1), byte(32 + y + 1)}), true
}
