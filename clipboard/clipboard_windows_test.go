//go:build windows

package clipboard

import (
	"fmt"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/deepnoodle-ai/wonton/assert"
	"golang.org/x/sys/windows"
)

var (
	procCreateWindowEx = user32.NewProc("CreateWindowExW")
	procDestroyWindow  = user32.NewProc("DestroyWindow")
)

// holdClipboard opens the clipboard and returns a func that releases it.
// It opens the clipboard with its own message-only window, the way another
// application would. An open with no window would not block this process,
// because the clipboard treats all such opens in a process as one owner.
func holdClipboard(t *testing.T) (release func()) {
	t.Helper()
	opened := make(chan error)
	done := make(chan struct{})
	released := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(released)

		class, _ := windows.UTF16PtrFromString("STATIC")
		const hwndMessage = ^uintptr(2) // HWND_MESSAGE is (HWND)-3
		hwnd, _, err := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(class)),
			0, 0, 0, 0, 0, 0, hwndMessage, 0, 0, 0)
		if hwnd == 0 {
			opened <- fmt.Errorf("CreateWindowExW: %w", err)
			return
		}
		defer procDestroyWindow.Call(hwnd)

		if r, _, err := procOpenClipboard.Call(hwnd); r == 0 {
			opened <- fmt.Errorf("OpenClipboard: %w", err)
			return
		}
		opened <- nil
		<-done
		procCloseClipboard.Call()
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
