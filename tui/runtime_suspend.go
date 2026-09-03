package tui

import (
	"errors"
	"os"

	"golang.org/x/term"
)

// ErrSuspendReentrant is returned when Suspend is called while a suspend is
// already in progress.
var ErrSuspendReentrant = errors.New("tui: Suspend is already in progress")

// Suspend hands the terminal back to plain output for the duration of fn.
//
// It leaves the alternate screen, releases mouse reporting, shows the cursor,
// and leaves raw mode, so anything fn writes lands in the terminal's own
// scrollback, where the terminal's find, selection, and copy all work on it.
// When fn returns, the screen, mouse mode, cursor state, and raw mode are
// restored and the next frame repaints from scratch.
//
// This is how an application offers "show me this in my real terminal" — a
// transcript dump, a diff, a log — without reimplementing search and selection
// inside the alternate screen.
//
// Input while fn runs: the runtime's input reader owns stdin and cannot hand it
// back, so fn does not read os.Stdin. It reads decoded events from the keys
// channel instead, which receives everything the user types while fn runs
// (events are dropped rather than queued if fn is not reading). The application
// receives no events and no frames are rendered until fn returns.
//
// Call Suspend from HandleEvent — that is the event-loop goroutine, between
// frames. Calling it from another goroutine is safe but will race with a render
// in progress; calling it from inside fn returns ErrSuspendReentrant.
//
// The error reports a failure to restore, not a failure of fn: fn has already
// run and the rest of the terminal state has still been put back. A terminal
// that cannot be returned to raw mode will not deliver keys as the application
// expects, so it is worth surfacing rather than swallowing.
func (r *Runtime) Suspend(fn func(keys <-chan Event)) (err error) {
	if fn == nil {
		return nil
	}

	keys := make(chan Event, 16)
	r.suspendMu.Lock()
	if r.suspendKeys != nil {
		r.suspendMu.Unlock()
		return ErrSuspendReentrant
	}
	r.suspendKeys = keys
	r.suspendMu.Unlock()

	// Remember what to put back. The terminal's own guards make each of these
	// idempotent, so restoring something that was never on is harmless.
	hadAltScreen := r.terminal.IsAlternateScreen()
	mouseMode := r.terminal.MouseMode()
	hadHiddenCursor := r.terminal.IsCursorHidden()
	hadRawMode := term.IsTerminal(int(os.Stdin.Fd())) && r.terminal.IsRawMode()

	if mouseMode != MouseModeOff {
		r.terminal.DisableMouseTracking()
	}
	if hadHiddenCursor {
		r.terminal.ShowCursor()
	}
	if hadAltScreen {
		r.terminal.DisableAlternateScreen()
	}
	if hadRawMode {
		r.terminal.DisableRawMode()
	}

	defer func() {
		r.suspendMu.Lock()
		r.suspendKeys = nil
		r.suspendMu.Unlock()

		if hadRawMode {
			if rawErr := r.terminal.EnableRawMode(); rawErr != nil && err == nil {
				err = rawErr
			}
		}
		if hadAltScreen {
			r.terminal.EnableAlternateScreen()
		}
		if hadHiddenCursor {
			r.terminal.HideCursor()
		}
		// Back into the mode the application chose, not whichever one is
		// handiest: an app that asked for buttons only would come back with
		// drag reporting it never wanted, and one that asked for hover would
		// come back without it.
		switch mouseMode {
		case MouseModeButtons:
			r.terminal.EnableMouseButtons()
		case MouseModeDrag:
			r.terminal.EnableMouseDrag()
		case MouseModeTracking:
			r.terminal.EnableMouseTracking()
		}
		// The alternate screen came back blank but the front buffer still
		// describes what used to be on it, so a diffing flush would send
		// almost nothing. Force the next frame to redraw every cell.
		r.terminal.Invalidate()
		r.render()
	}()

	fn(keys)
	return nil
}
