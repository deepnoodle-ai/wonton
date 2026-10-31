//go:build windows

package clipboard

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// holdClipboard opens the clipboard on its own OS thread, the way another
// process would, and returns a func that releases it.
func holdClipboard(t *testing.T) (release func()) {
	t.Helper()
	opened := make(chan error)
	done := make(chan struct{})
	released := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := openClipboard(ctx)
		opened <- err
		if err != nil {
			return
		}
		<-done
		procCloseClipboard.Call()
		close(released)
	}()
	assert.NoError(t, <-opened)
	return func() {
		close(done)
		<-released
	}
}

func TestWriteWaitsForClipboardHolder(t *testing.T) {
	release := holdClipboard(t)
	time.AfterFunc(300*time.Millisecond, release)

	text := "wonton held " + time.Now().String()
	assert.NoError(t, WriteWithTimeout(text, 5*time.Second))

	got, err := Read()
	assert.NoError(t, err)
	assert.Equal(t, text, got)
}

func TestWriteTimesOutWhileClipboardHeld(t *testing.T) {
	release := holdClipboard(t)
	defer release()

	err := WriteWithTimeout("never written", 200*time.Millisecond)
	assert.ErrorIs(t, err, ErrTimeout)

	_, err = ReadWithTimeout(200 * time.Millisecond)
	assert.ErrorIs(t, err, ErrTimeout)
}

func TestReadWriteRoundTripsUnicodeAndLineEndings(t *testing.T) {
	for _, text := range []string{
		"héllo wörld ✓ 日本語",
		"line one\r\nline two\r\n",
		"trailing newline\n",
		"",
	} {
		assert.NoError(t, Write(text))
		got, err := Read()
		assert.NoError(t, err)
		assert.Equal(t, text, got, "text %q", text)
	}
}
