//go:build darwin || linux

package terminal

import (
	"bytes"
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

func TestDetectKittyProtocol_NoReplyDoesNotSwallowNextKeystroke(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	oldStdin, oldStdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = slave, slave
	defer func() { os.Stdin, os.Stdout = oldStdin, oldStdout }()

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")

	// Drain the probe query from master without answering it.
	go func() {
		buf := make([]byte, 256)
		for {
			if _, err := master.Read(buf); err != nil {
				return
			}
		}
	}()

	term, err := NewTerminal()
	assert.NoError(t, err)
	assert.False(t, term.DetectKittyProtocol())
	assert.NoError(t, term.EnableRawMode())
	defer term.DisableRawMode()

	// The probe timed out with no reply. The first keystroke typed afterward
	// must reach the decoder rather than be consumed by a leaked probe read.
	_, err = master.WriteString("ab")
	assert.NoError(t, err)

	decoder := NewKeyDecoder(os.Stdin)
	done := make(chan [2]KeyEvent, 1)
	go func() {
		ev1, _ := decoder.ReadKeyEvent()
		ev2, _ := decoder.ReadKeyEvent()
		done <- [2]KeyEvent{ev1, ev2}
	}()

	select {
	case events := <-done:
		assert.Equal(t, events[0], KeyEvent{Rune: 'a'})
		assert.Equal(t, events[1], KeyEvent{Rune: 'b'})
	case <-time.After(time.Second):
		master.WriteString("zz") // unblock any stuck reader
		t.Fatal("first keystroke after timed-out DetectKittyProtocol was lost")
	}
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
			master, slave, err := pty.Open()
			if err != nil {
				t.Skipf("pty unavailable: %v", err)
			}
			defer master.Close()
			defer slave.Close()

			oldStdin, oldStdout := os.Stdin, os.Stdout
			os.Stdin, os.Stdout = slave, slave
			defer func() { os.Stdin, os.Stdout = oldStdin, oldStdout }()

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
				_, _ = master.WriteString(tc.reply)
			}()

			term, err := NewTerminal()
			assert.NoError(t, err)
			assert.Equal(t, term.DetectKittyProtocol(), tc.want)
			assert.Equal(t, term.IsKittyProtocolSupported(), tc.want)
			assert.NoError(t, term.EnableRawMode())
			defer term.DisableRawMode()

			decoder := NewKeyDecoder(os.Stdin)
			ev1, err := decoder.ReadKeyEvent()
			assert.NoError(t, err)
			assert.Equal(t, ev1, KeyEvent{Rune: 'x'})
			ev2, err := decoder.ReadKeyEvent()
			assert.NoError(t, err)
			assert.Equal(t, ev2, KeyEvent{Rune: 'y'})
		})
	}
}
