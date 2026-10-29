//go:build darwin || linux

package terminal

import (
	"os"
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

	decoder := NewKeyDecoder(r)
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
