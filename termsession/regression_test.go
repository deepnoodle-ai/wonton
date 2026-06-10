package termsession

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// TestLoadCast_TruncatedFile verifies that a recording with a truncated final
// line (e.g. from a recorder killed mid-write) loads the valid events instead
// of hanging. Previously this caused an infinite loop because the json.Decoder
// sticks on syntax errors.
func TestLoadCast_TruncatedFile(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "truncated.cast")

	content := `{"version": 2, "width": 80, "height": 24}
[0.1, "o", "hello"]
[0.5, "o", "trunca`
	assert.NoError(t, os.WriteFile(filename, []byte(content), 0644))

	done := make(chan struct{})
	var header *RecordingHeader
	var events []RecordingEvent
	var err error
	go func() {
		header, events, err = LoadCastFile(filename)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("LoadCastFile hung on truncated file")
	}

	assert.NoError(t, err)
	assert.Equal(t, 80, header.Width)
	assert.Len(t, events, 1)
	assert.Equal(t, "hello", events[0].Data)
}

// TestLoadCast_MalformedMiddleLine verifies that a corrupt line in the middle
// of a recording is skipped and later valid events are still loaded.
func TestLoadCast_MalformedMiddleLine(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "corrupt.cast")

	content := `{"version": 2, "width": 80, "height": 24}
[0.1, "o", "first"]
this line is garbage
[0.5, "o", "second"]
`
	assert.NoError(t, os.WriteFile(filename, []byte(content), 0644))

	_, events, err := LoadCastFile(filename)
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "first", events[0].Data)
	assert.Equal(t, "second", events[1].Data)
}

// TestRecorder_PauseRemovesGap verifies that the time spent paused does not
// appear as a gap in the recorded timeline.
func TestRecorder_PauseRemovesGap(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "pause.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{})
	assert.NoError(t, err)

	r.RecordOutput("before")
	r.Pause()
	time.Sleep(150 * time.Millisecond)
	r.Resume()
	r.RecordOutput("after")
	assert.NoError(t, r.Close())

	_, events, err := LoadCastFile(filename)
	assert.NoError(t, err)
	assert.Len(t, events, 2)

	gap := events[1].Time - events[0].Time
	if gap > 0.1 {
		t.Errorf("pause gap not removed from timeline: gap = %.3fs", gap)
	}
	if gap < 0 {
		t.Errorf("event times went backwards: gap = %.3fs", gap)
	}
}

// TestRecorder_UpdateSizeEmitsResizeEvent verifies that resizes are recorded
// as asciicast v2 "r" events.
func TestRecorder_UpdateSizeEmitsResizeEvent(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "resize.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{})
	assert.NoError(t, err)

	r.UpdateSize(80, 24) // unchanged: should not emit an event
	r.UpdateSize(120, 40)
	r.RecordOutput("hello")
	assert.NoError(t, r.Close())

	_, events, err := LoadCastFile(filename)
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "r", events[0].Type)
	assert.Equal(t, "120x40", events[0].Data)
	assert.Equal(t, "o", events[1].Type)
}

// TestRecorder_RedactsInput verifies that RedactSecrets applies to input
// events, not just output.
func TestRecorder_RedactsInput(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "redact.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{RedactSecrets: true})
	assert.NoError(t, err)

	secret := "export API_KEY=sk-1234567890abcdefghij\n"
	r.RecordInput(secret)
	assert.NoError(t, r.Close())

	raw, err := os.ReadFile(filename)
	assert.NoError(t, err)
	if strings.Contains(string(raw), "sk-1234567890abcdefghij") {
		t.Errorf("input event was not redacted: %s", raw)
	}
}

// TestRecordingEvent_ResizeRoundTrip verifies resize events survive a
// save/load cycle through the JSON array encoding.
func TestRecordingEvent_ResizeRoundTrip(t *testing.T) {
	e := RecordingEvent{Time: 1.5, Type: "r", Data: "100x30"}
	data, err := json.Marshal(e)
	assert.NoError(t, err)
	assert.Equal(t, `[1.5,"r","100x30"]`, string(data))
}

// TestSession_WaitBeforeStart verifies Wait returns an error instead of
// deadlocking when the session was never started.
func TestSession_WaitBeforeStart(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- s.Wait() }()

	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Wait deadlocked on unstarted session")
	}
}

// TestSession_ConcurrentStart verifies that concurrent Start calls do not
// double-start the command or panic on a double close of the done channel.
func TestSession_ConcurrentStart(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"true"},
		Input:   strings.NewReader(""),
		Output:  &strings.Builder{},
	})
	assert.NoError(t, err)
	defer s.Close()

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = s.Start()
		}(i)
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
		}
	}
	assert.Equal(t, 1, succeeded)

	assert.NoError(t, s.Wait())
}

// TestPlayer_PauseInterruptsIdleGap verifies that Pause takes effect promptly
// even when the player is sleeping through a long idle gap, and that no
// events are emitted while paused.
func TestPlayer_PauseInterruptsIdleGap(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "gap.cast")

	content := `{"version": 2, "width": 80, "height": 24}
[0.05, "o", "first"]
[10.0, "o", "second"]
`
	assert.NoError(t, os.WriteFile(filename, []byte(content), 0644))

	var out syncBuffer
	player, err := NewPlayer(filename, PlayerOptions{Output: &out})
	assert.NoError(t, err)

	playDone := make(chan struct{})
	go func() {
		player.Play()
		close(playDone)
	}()

	// Wait for the first event, then pause mid-gap.
	deadline := time.Now().Add(2 * time.Second)
	for out.String() == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	assert.Equal(t, "first", out.String())

	player.Pause()
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, "first", out.String()) // nothing emitted while paused

	player.Stop()
	select {
	case <-playDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Play did not return after Stop")
	}
}

// TestPlayer_LoopEmptyRecording verifies that looping playback of an empty
// recording returns instead of busy-spinning forever.
func TestPlayer_LoopEmptyRecording(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "empty.cast")

	content := `{"version": 2, "width": 80, "height": 24}
`
	assert.NoError(t, os.WriteFile(filename, []byte(content), 0644))

	player, err := NewPlayer(filename, PlayerOptions{Loop: true, Output: &strings.Builder{}})
	assert.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- player.Play() }()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		player.Stop()
		t.Fatal("Play busy-spun on empty looping recording")
	}
}
