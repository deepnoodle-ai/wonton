//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix

package terminal

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/internal/terminalinput"
	"golang.org/x/sys/unix"
)

// fileWaiter returns a function that waits up to a timeout for f to have
// input, without reading it. It returns nil if f has no file descriptor.
func fileWaiter(f *os.File) func(time.Duration) bool {
	conn, err := f.SyscallConn()
	if err != nil {
		return nil
	}
	return func(timeout time.Duration) bool {
		ready := false
		conn.Control(func(fd uintptr) {
			ready = pollReadable(int(fd), timeout)
		})
		return ready
	}
}

// pollReadable reports whether fd becomes readable within timeout. A hangup
// counts as readable, so the next read reports it.
func pollReadable(fd int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		remaining := time.Until(deadline)
		if remaining < 0 {
			remaining = 0
		}
		n, err := unix.Poll(fds, int((remaining+time.Millisecond-1)/time.Millisecond))
		if errors.Is(err, unix.EINTR) && remaining > 0 {
			continue
		}
		return err == nil && n > 0
	}
}

// probeKittyProtocol asks the terminal on in whether it supports the Kitty
// keyboard protocol (ESC [ ? u), followed by a primary device attributes
// request (ESC [ c) that every terminal answers. A terminal that supports the
// protocol answers the first with ESC [ ? <flags> u before its DA1 reply.
//
// It reaches the descriptor through SyscallConn and lets terminalinput.Query
// poll and read it directly. Reading through the *os.File instead fails two
// ways: os.Stdin from os.NewFile does not support read deadlines, and on a
// file whose Fd has been called, Read blocks in the kernel past any deadline,
// so a reader left behind after a timeout steals the next keystroke.
func probeKittyProtocol(in *os.File, out io.Writer, timeout time.Duration) bool {
	conn, err := in.SyscallConn()
	if err != nil {
		return false
	}
	var reply []byte
	conn.Control(func(fd uintptr) {
		reply, _ = terminalinput.Query(int(fd), out, "\x1b[?u\x1b[c", timeout)
	})
	return terminalinput.HasReply(reply, 'c') && terminalinput.HasReply(reply, 'u')
}
