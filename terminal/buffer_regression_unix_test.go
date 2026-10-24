//go:build darwin || linux

package terminal

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func TestRefreshSizeIgnoresZeroSize(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	if err := pty.Setsize(slave, &pty.Winsize{Rows: 0, Cols: 0}); err != nil {
		t.Fatal(err)
	}

	terminal := NewTestTerminal(10, 5, &bytes.Buffer{})
	terminal.fd = int(slave.Fd())
	if err := terminal.RefreshSize(); err != nil {
		t.Fatal(err)
	}
	if w, h := terminal.Size(); w != 10 || h != 5 {
		t.Fatalf("size = %dx%d, want 10x5 kept", w, h)
	}
	terminal.Println("still drawable") // must not panic
}

func TestHandoffRestoreRecordsModesOnlyAfterWrite(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	original, err := term.MakeRaw(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(slave.Fd()), original)

	out := &restorationFaultWriter{}
	terminal := &Terminal{fd: int(slave.Fd()), out: out, oldState: original, rawMode: true, cursorHidden: true, bracketedPaste: true}
	transition, err := terminal.handoffSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := transition.Release(); err != nil {
		t.Fatal(err)
	}
	// The mode write is the first control write in Restore here (no alt
	// screen). Let it fail before any byte reaches the terminal.
	out.Reset()
	out.call, out.failAt, out.bytes = 0, 1, 0
	if err := transition.Restore(false); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("restore=%v", err)
	}
	if terminal.cursorHidden || terminal.bracketedPaste {
		t.Fatalf("modes recorded as active after a failed write: hidden=%v paste=%v", terminal.cursorHidden, terminal.bracketedPaste)
	}

	// A write that fails partway may have enabled some modes. They must be
	// recorded so that exit sends their disables.
	out.Reset()
	out.call, out.failAt, out.bytes = 0, 1, 3
	if err := transition.Restore(false); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("restore=%v", err)
	}
	if !terminal.cursorHidden || !terminal.bracketedPaste {
		t.Fatalf("modes not recorded after a partial write: hidden=%v paste=%v", terminal.cursorHidden, terminal.bracketedPaste)
	}
	out.failAt = 0
	if err := transition.Restore(true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[?2004l") {
		t.Fatalf("exit did not disable bracketed paste: %q", out.String())
	}
}
