//go:build darwin || linux

package tui

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/deepnoodle-ai/wonton/assert"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// fakeTerminal answers background queries on the pty master with reply.
func fakeTerminal(t *testing.T, reply string) *os.File {
	t.Helper()
	master, tty, err := pty.Open()
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	t.Cleanup(func() { master.Close(); tty.Close() })
	go func() {
		var seen []byte
		buf := make([]byte, 64)
		for !bytes.Contains(seen, []byte(backgroundQuery)) {
			n, err := master.Read(buf)
			if err != nil {
				return
			}
			seen = append(seen, buf[:n]...)
		}
		if reply != "" {
			master.WriteString(reply)
		}
	}()
	return tty
}

// pending returns the input bytes the query left unread.
func pending(t *testing.T, tty *os.File) string {
	t.Helper()
	state, err := term.MakeRaw(int(tty.Fd()))
	assert.NoError(t, err)
	defer term.Restore(int(tty.Fd()), state)
	var out []byte
	buf := make([]byte, 64)
	for {
		fds := []unix.PollFd{{Fd: int32(tty.Fd()), Events: unix.POLLIN}}
		if n, _ := unix.Poll(fds, 100); n == 0 {
			return string(out)
		}
		n, err := unix.Read(int(tty.Fd()), buf)
		if n <= 0 || err != nil {
			return string(out)
		}
		out = append(out, buf[:n]...)
	}
}

func TestQueryBackgroundReadsReplyAndLeavesKeys(t *testing.T) {
	tty := fakeTerminal(t, "\x1b]11;rgb:fdfd/f6f6/e3e3\x1b\\\x1b[?62;22ckeys")
	start := time.Now()
	color, ok, err := queryBackground(tty, tty, time.Second)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, color, RGB{R: 253, G: 246, B: 227})
	assert.True(t, time.Since(start) < 500*time.Millisecond)
	assert.Equal(t, pending(t, tty), "keys")
}

func TestQueryBackgroundWithoutOSC11Support(t *testing.T) {
	tty := fakeTerminal(t, "\x1b[?1;2c")
	start := time.Now()
	_, ok, err := queryBackground(tty, tty, time.Second)
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.True(t, time.Since(start) < 500*time.Millisecond, "should stop at the DA1 reply")
}

func TestQueryBackgroundTimesOut(t *testing.T) {
	tty := fakeTerminal(t, "")
	start := time.Now()
	_, ok, err := queryBackground(tty, tty, 50*time.Millisecond)
	assert.NoError(t, err)
	assert.False(t, ok)
	elapsed := time.Since(start)
	assert.True(t, elapsed >= 50*time.Millisecond && elapsed < time.Second, "elapsed %v", elapsed)
	assert.Equal(t, pending(t, tty), "")
}

func TestQueryBackgroundTimeoutLeavesNoPartialReply(t *testing.T) {
	// The terminal starts a reply but never finishes it or answers DA1.
	tty := fakeTerminal(t, "\x1b]11;rgb:0000/00")
	start := time.Now()
	_, ok, err := queryBackground(tty, tty, 50*time.Millisecond)
	assert.NoError(t, err)
	assert.False(t, ok)
	// A started reply gets one extra timeout.
	assert.True(t, time.Since(start) >= 100*time.Millisecond)
	assert.Equal(t, pending(t, tty), "")
}
