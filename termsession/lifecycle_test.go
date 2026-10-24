package termsession

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func writeTestCast(t *testing.T, content string) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "test.cast")
	assert.NoError(t, os.WriteFile(filename, []byte(content), 0644))
	return filename
}

// TestPlayer_SeekSkipsElapsedEvents verifies that Seek positions playback at
// the first event at or after the target time, not on an event that already
// elapsed.
func TestPlayer_SeekSkipsElapsedEvents(t *testing.T) {
	filename := writeTestCast(t, `{"version": 2, "width": 80, "height": 24}
[0.0, "o", "a"]
[1.0, "o", "b"]
[2.0, "o", "c"]
`)
	player, err := NewPlayer(filename, PlayerOptions{Output: &lockedBuffer{}})
	assert.NoError(t, err)

	player.Seek(1.5)
	assert.Equal(t, 2, player.currentIndex)
	player.Seek(1.0)
	assert.Equal(t, 1, player.currentIndex)
	player.Seek(0)
	assert.Equal(t, 0, player.currentIndex)
	player.Seek(5)
	assert.Equal(t, 3, player.currentIndex)
}

// blockingWriter blocks the first Write until released.
type blockingWriter struct {
	lockedBuffer
	entered chan struct{}
	release chan struct{}
	first   bool
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	if !w.first {
		w.first = true
		close(w.entered)
		<-w.release
	}
	return w.lockedBuffer.Write(p)
}

// TestPlayer_SeekDuringWriteIsKept verifies that a Seek made while Play is
// writing a batch is not overwritten when the batch finishes.
func TestPlayer_SeekDuringWriteIsKept(t *testing.T) {
	filename := writeTestCast(t, `{"version": 2, "width": 80, "height": 24}
[0.0, "o", "a"]
[0.3, "o", "b"]
`)
	out := &blockingWriter{entered: make(chan struct{}), release: make(chan struct{})}
	player, err := NewPlayer(filename, PlayerOptions{Output: out})
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		player.Play()
		close(done)
	}()

	<-out.entered
	player.Seek(0) // rewind while "a" is being written
	close(out.release)

	deadline := time.Now().Add(2 * time.Second)
	for len(out.String()) < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	player.Stop()
	<-done
	assert.Equal(t, "aa", out.String()[:2], "the rewind was lost")
}

// TestRecorder_PauseClampsIdleBeforePause verifies that idle time before a
// pause still counts against IdleTimeLimit.
func TestRecorder_PauseClampsIdleBeforePause(t *testing.T) {
	r, err := NewRecorder(filepath.Join(t.TempDir(), "idle.cast"), 80, 24, RecordingOptions{IdleTimeLimit: 0.05})
	assert.NoError(t, err)
	defer r.Close()

	r.mu.Lock()
	r.startTime = r.startTime.Add(-2 * time.Second)
	r.lastEventTime = r.startTime // last event 2s ago
	r.mu.Unlock()

	r.Pause()
	r.Resume()

	r.mu.Lock()
	got := r.eventTimeLocked(time.Now())
	r.mu.Unlock()
	if got > 0.2 {
		t.Errorf("event time = %.3fs, want about 0.05s (idle before pause must be clamped)", got)
	}
}
