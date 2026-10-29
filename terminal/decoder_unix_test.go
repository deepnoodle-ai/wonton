//go:build darwin || linux

package terminal

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/deepnoodle-ai/wonton/assert"
)

func TestKeyDecoder_FileEscapeSequenceSplitAcrossWrites(t *testing.T) {
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	defer r.Close()
	defer w.Close()

	_, err = w.WriteString("x\x1b")
	assert.NoError(t, err)
	go func() {
		time.Sleep(10 * time.Millisecond)
		w.WriteString("[A")
	}()

	// The wait ends as soon as "[A" arrives. A long timeout keeps a slow
	// writer goroutine on a busy runner from turning ESC into KeyEscape.
	decoder := NewKeyDecoder(r)
	decoder.SetEscapeTimeout(time.Second)
	event, err := decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent), KeyEvent{Rune: 'x'})
	event, err = decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent), KeyEvent{Key: KeyArrowUp})
}

func TestKeyDecoder_FileLoneEscapeWaitsForTimeout(t *testing.T) {
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	defer r.Close()
	defer w.Close()

	_, err = w.WriteString("\x1b")
	assert.NoError(t, err)

	decoder := NewKeyDecoder(r)
	decoder.SetEscapeTimeout(20 * time.Millisecond)
	start := time.Now()
	event, err := decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent), KeyEvent{Key: KeyEscape})
	assert.True(t, time.Since(start) >= 20*time.Millisecond)
}

func TestKeyDecoder_FileLoneEscapeAtFullBuffer(t *testing.T) {
	// ESC is the last byte of a read that fills the 4096-byte buffer, and
	// nothing follows. The decoder must report KeyEscape after the timeout
	// rather than block until the next key press.
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	defer r.Close()
	defer w.Close()

	_, err = w.WriteString(strings.Repeat("a", 4095) + "\x1b")
	assert.NoError(t, err)

	decoder := NewKeyDecoder(r)
	decoder.SetEscapeTimeout(20 * time.Millisecond)
	done := make(chan KeyEvent, 1)
	go func() {
		for range 4095 {
			if _, err := decoder.ReadEvent(); err != nil {
				return
			}
		}
		event, _ := decoder.ReadEvent()
		done <- event.(KeyEvent)
	}()
	select {
	case event := <-done:
		assert.Equal(t, event, KeyEvent{Key: KeyEscape})
	case <-time.After(5 * time.Second):
		w.WriteString("q") // unblock the reader
		t.Fatal("lone ESC at a full buffer did not time out")
	}
}

// kittyTerminal opens a pty, makes it stdin and stdout, and answers the
// Kitty keyboard probe on its master with reply. An empty reply leaves the
// probe unanswered. It returns the master, where the test types keys.
func kittyTerminal(t *testing.T, reply string) *os.File {
	t.Helper()
	master, slave, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	t.Cleanup(func() { master.Close(); slave.Close() })

	oldStdin, oldStdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = slave, slave
	t.Cleanup(func() { os.Stdin, os.Stdout = oldStdin, oldStdout })
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")

	go func() {
		var seen []byte
		buf := make([]byte, 64)
		for !bytes.Contains(seen, []byte("\x1b[?u\x1b[c")) {
			n, err := master.Read(buf)
			if err != nil {
				return
			}
			seen = append(seen, buf[:n]...)
		}
		master.WriteString(reply)
		io.Copy(io.Discard, master)
	}()
	return master
}

// readKeys reads n key events from stdin. It fails instead of hanging when
// the keys never arrive, as when the probe consumed them.
func readKeys(t *testing.T, master *os.File, n int) []KeyEvent {
	t.Helper()
	decoder := NewKeyDecoder(os.Stdin)
	done := make(chan []KeyEvent, 1)
	go func() {
		var events []KeyEvent
		for range n {
			ev, _ := decoder.ReadKeyEvent()
			events = append(events, ev)
		}
		done <- events
	}()
	select {
	case events := <-done:
		return events
	case <-time.After(2 * time.Second):
		master.WriteString(strings.Repeat("z", n)) // unblock the reader
		t.Fatal("keys typed after DetectKittyProtocol never reached the decoder")
		return nil
	}
}

func TestDetectKittyProtocol_NoReplyDoesNotSwallowNextKeystroke(t *testing.T) {
	master := kittyTerminal(t, "")
	term, err := NewTerminal()
	assert.NoError(t, err)
	assert.False(t, term.DetectKittyProtocol())
	assert.NoError(t, term.EnableRawMode())
	defer term.DisableRawMode()

	// The probe timed out with no reply. The first keystroke typed afterward
	// must reach the decoder rather than be consumed by a leaked probe read.
	_, err = master.WriteString("ab")
	assert.NoError(t, err)
	assert.Equal(t, readKeys(t, master, 2), []KeyEvent{{Rune: 'a'}, {Rune: 'b'}})
}

func TestDetectKittyProtocol_ReplyAndLeavesSubsequentKeys(t *testing.T) {
	for _, tc := range []struct {
		name  string
		reply string
		want  bool
	}{
		{"supported", "\x1b[?0u\x1b[?62;22cxy", true},
		{"unsupported DA1 only", "\x1b[?62;22cxy", false},
		{"unsupported DA1 with typed u", "u\x1b[?62;22cxy", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master := kittyTerminal(t, tc.reply)
			term, err := NewTerminal()
			assert.NoError(t, err)
			assert.Equal(t, term.DetectKittyProtocol(), tc.want)
			assert.Equal(t, term.IsKittyProtocolSupported(), tc.want)
			assert.NoError(t, term.EnableRawMode())
			defer term.DisableRawMode()

			// The probe stops at the end of the DA1 reply, so the keys
			// typed after it are still there.
			assert.Equal(t, readKeys(t, master, 2), []KeyEvent{{Rune: 'x'}, {Rune: 'y'}})
		})
	}
}
