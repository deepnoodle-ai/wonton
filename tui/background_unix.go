//go:build darwin || linux

package tui

import (
	"errors"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// queryBackground writes backgroundQuery to out and reads the replies from
// in, a terminal, for up to timeout. It reads one byte at a time and stops
// at the end of the DA1 reply, so it never consumes input that follows. On a
// timeout it flushes pending input instead.
func queryBackground(in *os.File, out io.Writer, timeout time.Duration) (RGB, bool, error) {
	fd := int(in.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return RGB{}, false, err
	}
	defer term.Restore(fd, state)

	if _, err := io.WriteString(out, backgroundQuery); err != nil {
		return RGB{}, false, err
	}

	var buf []byte
	var one [1]byte
	deadline := time.Now().Add(timeout)
	extended := false
	for {
		replies := scanBackgroundReplies(buf)
		if replies.done || len(buf) > 1024 {
			return replies.color, replies.ok, nil
		}
		wait := time.Until(deadline)
		if wait <= 0 {
			// A terminal that has started to answer gets one more timeout
			// to finish, so a slow reply is less likely to arrive later
			// and be read as keys.
			if len(buf) > 0 && !extended {
				deadline = deadline.Add(timeout)
				extended = true
				continue
			}
			// No DA1 reply yet: drop whatever input has arrived, so a reply
			// already on its way is not read as keys. This also drops keys
			// typed during the query.
			return replies.color, replies.ok, flushInput(fd)
		}
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, int(wait/time.Millisecond)+1)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return replies.color, replies.ok, err
		}
		if n == 0 {
			continue
		}
		if fds[0].Revents&unix.POLLIN == 0 {
			return replies.color, replies.ok, nil // hung up or invalid
		}
		m, err := unix.Read(fd, one[:])
		if m == 1 {
			buf = append(buf, one[0])
			continue
		}
		if err == nil {
			return replies.color, replies.ok, nil // end of input
		}
		if !errors.Is(err, unix.EINTR) && !errors.Is(err, unix.EAGAIN) {
			return replies.color, replies.ok, err
		}
	}
}
