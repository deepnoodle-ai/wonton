//go:build darwin || linux

package terminal

import (
	"os"
	"strings"
	"testing"
	"time"

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
