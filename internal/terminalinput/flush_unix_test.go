//go:build darwin || linux

package terminalinput

import (
	"testing"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func TestFlushInputDropsPendingBytes(t *testing.T) {
	master, tty, err := pty.Open()
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	defer master.Close()
	defer tty.Close()
	fd := int(tty.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(fd, state)

	master.WriteString("\x1b]11;rgb:ffff/ffff/ffff\x07")
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	unix.Poll(fds, 1000) // wait for the bytes to arrive
	if err := FlushInput(fd); err != nil {
		t.Fatal(err)
	}
	if got := pending(t, tty); got != "" {
		t.Fatalf("pending = %q after FlushInput", got)
	}
}
