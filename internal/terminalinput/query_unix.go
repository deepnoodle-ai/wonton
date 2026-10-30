//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix

package terminalinput

import (
	"errors"
	"io"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// maxReply bounds how much Query reads while it waits for the DA1 reply. It
// is a var so that a test can lower it below the pty input queue size.
var maxReply = 1024

// Query writes query to out and returns what the terminal on fd sends back.
// query must end with a primary device attributes request (ESC [ c). Every
// terminal answers it, and answers in order, so the DA1 reply marks the end
// of whatever the terminal will say. Query puts fd in raw mode while it
// waits, polls fd directly rather than through an *os.File, and reads one
// byte at a time up to the end of the DA1 reply, so input that follows the
// reply stays unread.
//
// If no DA1 reply arrives within timeout, a terminal that has started to
// answer gets one more timeout to finish. After that, or after maxReply bytes
// without a DA1 reply, Query discards pending input, so the rest of a slow
// reply is not left to be read as keys. Keys typed while Query waits are
// read or discarded with the reply. Bytes that arrive after the discard,
// such as a reply slower than two timeouts, are left for the caller's input
// reader.
func Query(fd int, out io.Writer, query string, timeout time.Duration) ([]byte, error) {
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer term.Restore(fd, state)

	if _, err := io.WriteString(out, query); err != nil {
		return nil, err
	}

	var buf []byte
	var one [1]byte
	deadline := time.Now().Add(timeout)
	extended := false
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		if HasReply(buf, 'c') {
			return buf, nil
		}
		if len(buf) >= maxReply {
			return buf, FlushInput(fd)
		}
		wait := time.Until(deadline)
		if wait <= 0 {
			if len(buf) > 0 && !extended {
				deadline = deadline.Add(timeout)
				extended = true
				continue
			}
			return buf, FlushInput(fd)
		}
		n, err := unix.Poll(fds, int((wait+time.Millisecond-1)/time.Millisecond))
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return buf, err
		}
		if n == 0 {
			continue
		}
		if fds[0].Revents&unix.POLLIN == 0 {
			return buf, nil // hung up or invalid
		}
		m, err := unix.Read(fd, one[:])
		if m == 1 {
			buf = append(buf, one[0])
			continue
		}
		if err == nil {
			return buf, nil // end of input
		}
		if !errors.Is(err, unix.EINTR) && !errors.Is(err, unix.EAGAIN) {
			return buf, err
		}
	}
}
