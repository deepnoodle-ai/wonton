package terminal

import (
	"errors"
	"strings"

	"github.com/deepnoodle-ai/wonton/internal/terminalstate"
	"golang.org/x/term"
)

func init() {
	terminalstate.Inspect = func(value any) (terminalstate.Access, bool) {
		t, ok := value.(*Terminal)
		if !ok || t == nil || t.fd < 0 || t.closed {
			return terminalstate.Access{}, false
		}
		return terminalstate.Access{FD: t.fd, Output: t.out, Snapshot: t.handoffSnapshot, CleanupRuntime: t.cleanupRuntime}, true
	}
}

// handoffSnapshot preserves actual attributes and every enabled protocol.
// The runner calls it between frames while holding its output exclusion gate.
func (t *Terminal) handoffSnapshot() (terminalstate.Transition, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return terminalstate.Transition{}, ErrClosed
	}
	attributes, err := term.GetState(t.fd)
	if err != nil {
		return terminalstate.Transition{}, err
	}
	oldState := t.oldState
	raw, alt, hidden, paste, kitty, mouse := t.rawMode, t.altScreen, t.cursorHidden, t.bracketedPaste, t.kittyEnabled, t.mouseMode
	plain := func() error {
		t.mu.Lock()
		defer t.mu.Unlock()
		// Attempt all plain-terminal cleanup even if a protocol write fails.
		var keyboardErr error
		if t.kittyEnabled {
			complete, err := terminalstate.WriteControl(t.out, "\x1b[<u")
			keyboardErr = err
			if complete {
				t.kittyEnabled = false
			}
		}
		// Keep the active screen until its keyboard push has been removed.
		leaveAlt := t.altScreen && !t.kittyEnabled
		complete, writeErr := terminalstate.WriteControl(t.out, plainModes(leaveAlt, t.bracketedPaste, false, t.mouseMode))
		if complete {
			if leaveAlt {
				t.altScreen = false
			}
			t.cursorHidden, t.bracketedPaste, t.mouseMode = false, false, MouseModeOff
		}
		outputErr := errors.Join(keyboardErr, writeErr, terminalstate.Flush(t.out))
		var attributeErr error
		if raw && oldState != nil {
			attributeErr = term.Restore(t.fd, oldState)
		} else {
			attributeErr = term.Restore(t.fd, attributes)
		}
		if attributeErr == nil {
			t.rawMode = false
			t.oldState = nil
		}
		return errors.Join(outputErr, attributeErr)
	}
	return terminalstate.Transition{
		Release: plain,
		Restore: func(stopping bool) error {
			if stopping {
				return plain()
			}
			t.mu.Lock()
			defer t.mu.Unlock()
			attributeErr := term.Restore(t.fd, attributes)
			complete, writeErr := terminalstate.WriteControl(t.out, activeModes(alt, hidden, paste, false, mouse))
			if complete {
				t.altScreen, t.cursorHidden, t.bracketedPaste, t.mouseMode = alt, hidden, paste, mouse
			}
			var keyboardErr error
			if complete && kitty && !t.kittyEnabled {
				// Track an attempted push too: cleanup must terminate a partial enable.
				t.kittyEnabled = true
				_, keyboardErr = terminalstate.WriteControl(t.out, "\x1b[>1u")
			}
			outputErr := errors.Join(writeErr, keyboardErr, terminalstate.Flush(t.out))
			if attributeErr == nil {
				t.rawMode = raw
				t.oldState = oldState
			}
			return errors.Join(attributeErr, outputErr)
		},
	}, nil
}

func plainModes(alt, paste, kitty bool, mouse MouseMode) string {
	var b strings.Builder
	if mouse != MouseModeOff {
		b.WriteString("\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l")
	}
	if paste {
		b.WriteString("\x1b[?2004l")
	}
	if kitty {
		b.WriteString("\x1b[<u")
	}
	b.WriteString("\x1b[?25h")
	if alt {
		b.WriteString("\x1b[?1049l")
	}
	return b.String()
}
func activeModes(alt, hidden, paste, kitty bool, mouse MouseMode) string {
	var b strings.Builder
	if alt {
		b.WriteString("\x1b[?1049h")
	}
	if paste {
		b.WriteString("\x1b[?2004h")
	}
	if kitty {
		b.WriteString("\x1b[>1u")
	}
	if mouse != MouseModeOff {
		b.WriteString("\x1b[?1000h\x1b[?1006h")
		switch mouse {
		case MouseModeDrag:
			b.WriteString("\x1b[?1002h")
		case MouseModeTracking:
			b.WriteString("\x1b[?1002h\x1b[?1003h")
		}
	}
	if hidden {
		b.WriteString("\x1b[?25l")
	} else {
		b.WriteString("\x1b[?25h")
	}
	return b.String()
}

// cleanupRuntime restores only modes enabled by a failed runner startup.
func (t *Terminal) cleanupRuntime(raw, kitty bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	var outputErr, rawErr error
	if kitty && t.kittyEnabled {
		complete, writeErr := terminalstate.WriteControl(t.out, "\x1b[<u")
		if complete {
			t.kittyEnabled = false
		}
		outputErr = errors.Join(writeErr, terminalstate.Flush(t.out))
	}
	if raw && t.rawMode && t.oldState != nil {
		rawErr = term.Restore(t.fd, t.oldState)
		if rawErr == nil {
			t.rawMode = false
			t.oldState = nil
		}
	}
	return errors.Join(outputErr, rawErr)
}
