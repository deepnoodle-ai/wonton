package tui

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// syncBuffer is a bytes.Buffer that is safe for concurrent use.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newTestPlayback(events ...RecordingEvent) *PlaybackController {
	return &PlaybackController{
		events:   events,
		speed:    1.0,
		stopChan: make(chan struct{}),
	}
}

// playAsync runs Play in a goroutine and returns a channel closed when it returns.
func playAsync(p *PlaybackController, term *Terminal) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Play(term)
	}()
	return done
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Play did not return")
	}
}

func TestPlayback_LoadRecordingLongLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "long.cast")
	big := strings.Repeat("x", 200*1024)
	ev, err := json.Marshal([]any{0.5, "o", big})
	assert.NoError(t, err)
	content := `{"version":2,"width":80,"height":24}` + "\n" + string(ev) + "\n"
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	p, err := LoadRecording(path)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(p.events))
	assert.Equal(t, big, p.events[0].Data)
}

func TestPlayback_StopPlayStopDoesNotPanic(t *testing.T) {
	p := newTestPlayback(RecordingEvent{Time: 0, Type: "o", Data: "a"})
	out := &syncBuffer{}
	term := NewTestTerminal(10, 2, out)

	p.Stop()
	// Play after Stop must play again, not exit immediately.
	waitDone(t, playAsync(p, term))
	assert.Equal(t, "a", out.String())

	p.Stop()
	p.Stop()
}

func TestPlayback_StopInterruptsReplayedPlay(t *testing.T) {
	p := newTestPlayback(RecordingEvent{Time: 10, Type: "o", Data: "late"})
	term := NewTestTerminal(10, 2, &syncBuffer{})

	p.Stop()
	done := playAsync(p, term)
	time.Sleep(20 * time.Millisecond)
	p.Stop()
	waitDone(t, done)
}

func TestPlayback_EmptyLoopedRecordingReturns(t *testing.T) {
	p := newTestPlayback()
	p.SetLoop(true)
	term := NewTestTerminal(10, 2, &syncBuffer{})
	waitDone(t, playAsync(p, term))
}

func TestPlayback_NoOutputWhilePausedAfterSleep(t *testing.T) {
	p := newTestPlayback(RecordingEvent{Time: 0.1, Type: "o", Data: "x"})
	out := &syncBuffer{}
	term := NewTestTerminal(10, 2, out)
	done := playAsync(p, term)

	time.Sleep(30 * time.Millisecond)
	p.Pause()
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, "", out.String())

	p.Stop()
	waitDone(t, done)
}

func TestPlayback_SeekDuringSleepIsNotClobbered(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 0.1, Type: "o", Data: "a"},
		RecordingEvent{Time: 5, Type: "o", Data: "b"},
		RecordingEvent{Time: 5.05, Type: "o", Data: "c"},
	)
	out := &syncBuffer{}
	term := NewTestTerminal(10, 2, out)
	done := playAsync(p, term)

	// Play is sleeping until event "a" at 0.1s. Seek to event "b" at 5s.
	time.Sleep(30 * time.Millisecond)
	p.Seek(5)
	time.Sleep(150 * time.Millisecond)
	p.Stop()
	waitDone(t, done)

	assert.Equal(t, "bc", out.String())
}

func TestPlayback_SetSpeedWhilePausedKeepsPosition(t *testing.T) {
	p := newTestPlayback(RecordingEvent{Time: 1, Type: "o", Data: "x"})
	p.startTime = time.Now().Add(-500 * time.Millisecond)
	p.Pause()
	p.SetSpeed(2)
	p.Resume()

	// 0.5s of recording time elapsed at 1x. At 2x the same recording
	// position is 0.25s of playback time.
	p.mu.RLock()
	elapsed := p.elapsedLocked()
	p.mu.RUnlock()
	assert.True(t, elapsed > 0.24 && elapsed < 0.26, "elapsed = %v, want ~0.25", elapsed)
}

func TestPlayback_SeekWhilePausedDoesNotDoubleCountPause(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 1, Type: "o", Data: "a"},
		RecordingEvent{Time: 2, Type: "o", Data: "b"},
	)
	p.startTime = time.Now()
	p.Pause()
	time.Sleep(100 * time.Millisecond)
	p.Seek(1)
	p.Resume()

	p.mu.RLock()
	elapsed := p.elapsedLocked()
	p.mu.RUnlock()
	assert.True(t, elapsed > 0.95 && elapsed < 1.05, "elapsed after seek = %v, want ~1", elapsed)
}

func TestPlayback_ConcurrentControlIsRaceFree(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 0.02, Type: "o", Data: "a"},
		RecordingEvent{Time: 0.04, Type: "o", Data: "b"},
		RecordingEvent{Time: 0.06, Type: "o", Data: "c"},
	)
	p.SetLoop(true)
	term := NewTestTerminal(10, 2, &syncBuffer{})
	done := playAsync(p, term)

	for i := 0; i < 20; i++ {
		p.Seek(0.03)
		p.SetSpeed(float64(i%3 + 1))
		p.TogglePause()
		time.Sleep(2 * time.Millisecond)
	}
	p.Stop()
	waitDone(t, done)
}

func TestPlayback_SeekDuringLongGapTakesEffectPromptly(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 0, Type: "o", Data: "a"},
		RecordingEvent{Time: 2, Type: "o", Data: "b"},
		RecordingEvent{Time: 4, Type: "o", Data: "c"},
	)
	out := &syncBuffer{}
	term := NewTestTerminal(10, 2, out)
	done := playAsync(p, term)

	// Play is waiting 2s for "b". Seek to "c" partway through the wait.
	time.Sleep(50 * time.Millisecond)
	p.Seek(4)
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, "ac", out.String())

	p.Stop()
	waitDone(t, done)
}

func TestPlayback_SetSpeedDuringLongGapTakesEffectPromptly(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 0, Type: "o", Data: "a"},
		RecordingEvent{Time: 2, Type: "o", Data: "b"},
	)
	out := &syncBuffer{}
	term := NewTestTerminal(10, 2, out)
	done := playAsync(p, term)

	// About 0.1s of 2s has played. At 100x the rest takes about 19ms.
	time.Sleep(100 * time.Millisecond)
	p.SetSpeed(100)
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, "ab", out.String())

	waitDone(t, done)
}

func TestPlayback_PauseStopPlayRunsOneLoop(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 0, Type: "o", Data: "a"},
		RecordingEvent{Time: 0.05, Type: "o", Data: "b"},
		RecordingEvent{Time: 1, Type: "o", Data: "c"},
	)
	out := &syncBuffer{}
	term := NewTestTerminal(10, 2, out)
	first := playAsync(p, term)

	// Pause before "b" is due. When the wait for "b" ends, the first loop
	// sees the pause and enters its paused sleep.
	time.Sleep(20 * time.Millisecond)
	p.Pause()
	time.Sleep(60 * time.Millisecond)
	p.Stop()
	second := playAsync(p, term)

	// The first loop must exit on its own Stop, not keep running beside
	// the second loop until the recording ends.
	select {
	case <-first:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first Play loop kept running after Stop and a new Play")
	}

	waitDone(t, second)
	// "a" from the first run, then "abc" from the second run only.
	assert.Equal(t, "aabc", out.String())
}

func TestPlayback_SeekPicksNextEvent(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 1, Type: "o", Data: "a"},
		RecordingEvent{Time: 2, Type: "o", Data: "b"},
	)
	p.startTime = time.Now()
	p.Seek(1.5)
	assert.Equal(t, p.currentIndex, 1)
	// Past the end, no event is left to replay.
	p.Seek(10)
	assert.Equal(t, p.currentIndex, 2)
}

func TestPlayback_PositionFollowsClock(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 1, Type: "o", Data: "a"},
		RecordingEvent{Time: 10, Type: "o", Data: "b"},
	)
	assert.Equal(t, p.GetPosition(), 0.0)

	// 2s into playback, the next event is still the one at 10s.
	p.startTime = time.Now().Add(-2 * time.Second)
	p.currentIndex = 1
	pos := p.GetPosition()
	assert.True(t, pos > 1.9 && pos < 2.1, "position = %v, want ~2", pos)
}

func TestPlayback_SeekAndPauseBeforePlay(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 1, Type: "o", Data: "a"},
		RecordingEvent{Time: 5, Type: "o", Data: "b"},
		RecordingEvent{Time: 10, Type: "o", Data: "c"},
	)
	p.SetSpeed(2)
	p.Seek(5)
	p.Pause()
	assert.Equal(t, p.GetPosition(), 5.0)
	assert.Equal(t, p.GetProgress(), 0.5)

	// Play starts paused at the seek position.
	out := &syncBuffer{}
	done := playAsync(p, NewTestTerminal(10, 2, out))
	time.Sleep(100 * time.Millisecond)
	assert.True(t, p.IsPaused())
	pos := p.GetPosition()
	assert.True(t, pos > 4.9 && pos < 5.1, "position = %v, want ~5", pos)
	assert.Equal(t, out.String(), "")

	// On resume, the event at 5s plays and the one at 10s does not yet.
	p.Resume()
	time.Sleep(150 * time.Millisecond)
	p.Stop()
	waitDone(t, done)
	assert.Equal(t, out.String(), "b")
}

func TestPlayback_PositionHoldsAfterStop(t *testing.T) {
	p := newTestPlayback(
		RecordingEvent{Time: 0, Type: "o", Data: "a"},
		RecordingEvent{Time: 10, Type: "o", Data: "b"},
	)
	done := playAsync(p, NewTestTerminal(10, 2, &syncBuffer{}))
	time.Sleep(50 * time.Millisecond)
	p.Stop()
	waitDone(t, done)

	pos := p.GetPosition()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, p.GetPosition(), pos)

	// A Pause and Resume after Stop do not move the stopped clock.
	p.Pause()
	time.Sleep(20 * time.Millisecond)
	p.Resume()
	assert.Equal(t, p.GetPosition(), pos)
}
