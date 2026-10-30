//go:build darwin || linux

package terminalinput

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const testQuery = "\x1b[?u\x1b[c"

// answerQuery opens a pty and calls answer on its master once the pty has
// received testQuery. It returns the pty's terminal side.
func answerQuery(t *testing.T, answer func(master *os.File)) *os.File {
	t.Helper()
	master, tty, err := pty.Open()
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	t.Cleanup(func() { master.Close(); tty.Close() })
	go func() {
		var seen []byte
		buf := make([]byte, 64)
		for !bytes.Contains(seen, []byte(testQuery)) {
			n, err := master.Read(buf)
			if err != nil {
				return
			}
			seen = append(seen, buf[:n]...)
		}
		answer(master)
		io.Copy(io.Discard, master)
	}()
	return tty
}

// pending returns the input bytes left unread on tty.
func pending(t *testing.T, tty *os.File) string {
	t.Helper()
	fd := int(tty.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(fd, state)
	var out []byte
	buf := make([]byte, 64)
	for waitReadable(fd, 100*time.Millisecond) {
		n, err := unix.Read(fd, buf)
		if n <= 0 || err != nil {
			return string(out)
		}
		out = append(out, buf[:n]...)
	}
	return string(out)
}

// waitReadable reports whether fd has input within timeout. Unlike a bare
// unix.Poll, it retries when a signal interrupts the poll.
func waitReadable(fd int, timeout time.Duration) bool {
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		n, err := unix.Poll(fds, int(timeout/time.Millisecond))
		if !errors.Is(err, unix.EINTR) {
			return n > 0
		}
	}
}

func TestQueryFinishesReplyDuringGraceWindow(t *testing.T) {
	const timeout = 150 * time.Millisecond
	start := time.Now()
	tty := answerQuery(t, func(master *os.File) {
		// Start the reply in the first timeout and finish it halfway through
		// the second, measured from when Query started.
		master.WriteString("\x1b[?0u\x1b[?62;")
		time.Sleep(time.Until(start.Add(timeout * 3 / 2)))
		master.WriteString("22ca")
	})
	reply, err := Query(int(tty.Fd()), tty, testQuery, timeout)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(reply); got != "\x1b[?0u\x1b[?62;22c" {
		t.Fatalf("reply = %q", got)
	}
	if got := pending(t, tty); got != "a" {
		t.Fatalf("pending = %q, want the key typed after the reply", got)
	}
}

func TestQueryFlushesOverlongReply(t *testing.T) {
	master, tty, err := pty.Open()
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	t.Cleanup(func() { master.Close(); tty.Close() })
	// Queue the whole reply before Query runs. Written while Query reads,
	// the tail can reach the tty after the flush, which Query leaves unread
	// by design. A macOS pty queues less than the default maxReply, so lower
	// it to fit the reply in the queue.
	defer func(old int) { maxReply = old }(maxReply)
	maxReply = 64
	fd := int(tty.Fd())
	// Raw mode, so the tty neither echoes the bytes nor holds them for a
	// newline.
	state, err := term.MakeRaw(fd)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := master.WriteString(strings.Repeat("x", maxReply+100)); err != nil {
		t.Fatal(err)
	}
	term.Restore(fd, state)
	go io.Copy(io.Discard, master)

	reply, err := Query(fd, tty, testQuery, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(reply) != maxReply {
		t.Fatalf("read %d bytes, want %d", len(reply), maxReply)
	}
	if got := pending(t, tty); got != "" {
		t.Fatalf("pending = %q, want the rest flushed", got)
	}
}
