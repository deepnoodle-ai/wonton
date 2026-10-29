//go:build darwin || linux

package terminalinput

import (
	"bytes"
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
	for {
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		if n, _ := unix.Poll(fds, 100); n == 0 {
			return string(out)
		}
		n, err := unix.Read(fd, buf)
		if n <= 0 || err != nil {
			return string(out)
		}
		out = append(out, buf[:n]...)
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
	tty := answerQuery(t, func(master *os.File) {
		master.WriteString(strings.Repeat("x", maxReply+100))
	})
	reply, err := Query(int(tty.Fd()), tty, testQuery, time.Second)
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
