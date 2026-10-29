//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix

package terminal

import (
	"errors"
	"os"
	"time"

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
