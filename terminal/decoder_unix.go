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
// Accessing the raw descriptor via SyscallConn, polling with unix.Poll, and
// reading a byte at a time avoids two failure modes of os.File.SetReadDeadline
// + os.File.Read on terminal descriptors:
//   - Standard os.Stdin (os.NewFile) is not registered with Go's netpoller,
//     so SetReadDeadline fails with os.ErrNoDeadline on real terminals.
//   - On an *os.File opened via os.OpenFile (such as a PTY slave), calling
//     Fd() clears O_NONBLOCK while leaving the netpoller context attached:
//     SetReadDeadline then succeeds, but Read enters a blocking syscall.Read
//     before consulting the netpoller and parks a leaked goroutine that
//     swallows the user's first keystroke after a probe timeout. Using
//     SyscallConn avoids calling Fd() and preserves O_NONBLOCK on in.
//
// Stopping at the end of the DA1 reply ('c') leaves any keystrokes queued
// after the reply unread in the kernel buffer. On a timeout, a partial reply
// gets one additional timeout window to finish; if the reply still does not
// complete, pending input is flushed so a slow or partial DA1 response is not
// left in the input buffer to be decoded as key events.
func probeKittyProtocol(in *os.File, out io.Writer, timeout time.Duration) bool {
	if in == nil || out == nil {
		return false
	}
	conn, err := in.SyscallConn()
	if err != nil {
		return false
	}
	var supported bool
	_ = conn.Control(func(rawFD uintptr) {
		fd := int(rawFD)
		if !term.IsTerminal(fd) {
			return
		}
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return
		}
		defer term.Restore(fd, oldState)

		if _, err := io.WriteString(out, "\x1b[?u\x1b[c"); err != nil {
			return
		}

		var buf []byte
		var one [1]byte
		deadline := time.Now().Add(timeout)
		extended := false
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		for {
			if hasCSIQuestionReply(buf, 'c') {
				supported = hasCSIQuestionReply(buf, 'u')
				return
			}
			if len(buf) >= 256 {
				_ = flushInput(fd)
				return
			}
			remaining := time.Until(deadline)
			if remaining <= 0 {
				// A terminal that has started to answer gets one more timeout
				// window to finish, so a slow reply (e.g. over SSH) is less
				// likely to arrive later and be read as keys.
				if len(buf) > 0 && !extended {
					deadline = deadline.Add(timeout)
					extended = true
					continue
				}
				// No complete DA1 reply yet: discard whatever input has arrived
				// so a partial or late reply is not read as keys. This also
				// drops keys typed during the probe window.
				_ = flushInput(fd)
				return
			}
			n, err := unix.Poll(fds, int((remaining+time.Millisecond-1)/time.Millisecond))
			if err != nil {
				if errors.Is(err, unix.EINTR) {
					continue
				}
				return
			}
			if n == 0 {
				continue
			}
			if fds[0].Revents&unix.POLLIN == 0 {
				return
			}
			m, err := unix.Read(fd, one[:])
			if m == 1 {
				buf = append(buf, one[0])
				continue
			}
			if err == nil || (!errors.Is(err, unix.EINTR) && !errors.Is(err, unix.EAGAIN)) {
				return
			}
		}
	})
	return supported
}
