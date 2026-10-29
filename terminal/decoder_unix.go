//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix

package terminal

import (
	"errors"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
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

// probeKittyProtocol queries out for Kitty progressive enhancement support
// (\x1b[?u) followed by primary device attributes (\x1b[c), and reads the
// response from in one byte at a time after polling for readability.
//
// Polling and reading a byte at a time avoids two failure modes of
// os.File.SetReadDeadline + os.File.Read on terminal descriptors:
//   - Standard os.Stdin (os.NewFile) is not registered with Go's netpoller,
//     so SetReadDeadline fails with os.ErrNoDeadline on real terminals.
//   - On an *os.File opened via os.OpenFile (such as a PTY slave), calling
//     Fd() clears O_NONBLOCK while leaving the netpoller context attached:
//     SetReadDeadline then succeeds, but Read enters a blocking syscall.Read
//     before consulting the netpoller and parks a leaked goroutine that
//     swallows the user's first keystroke after a probe timeout.
//
// Stopping at the end of the DA1 reply ('c') leaves any keystrokes queued
// after the reply unread in the kernel buffer.
func probeKittyProtocol(in *os.File, out io.Writer, timeout time.Duration) bool {
	if in == nil || out == nil {
		return false
	}
	fd := int(in.Fd())
	if !term.IsTerminal(fd) {
		return false
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return false
	}
	defer term.Restore(fd, oldState)

	if _, err := io.WriteString(out, "\x1b[?u\x1b[c"); err != nil {
		return false
	}

	var buf []byte
	var one [1]byte
	deadline := time.Now().Add(timeout)
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for len(buf) < 256 && !hasCSIQuestionReply(buf, 'c') {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		n, err := unix.Poll(fds, int((remaining+time.Millisecond-1)/time.Millisecond))
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			break
		}
		if n == 0 || fds[0].Revents&unix.POLLIN == 0 {
			break
		}
		m, err := unix.Read(fd, one[:])
		if m == 1 {
			buf = append(buf, one[0])
			continue
		}
		if err == nil || (!errors.Is(err, unix.EINTR) && !errors.Is(err, unix.EAGAIN)) {
			break
		}
	}
	return hasCSIQuestionReply(buf, 'u') && hasCSIQuestionReply(buf, 'c')
}
