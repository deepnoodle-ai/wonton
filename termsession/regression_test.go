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

// TestRecorder_EventTimeNeverGoesBackwards verifies that a pause can never
// order two event timestamps backwards, however coarse the host clock is.
//
// eventTimeLocked used to convert the elapsed time and the accumulated pause
// adjustment to float64 seconds separately and subtract them. Each conversion
// rounds on its own, so when the true gap between two events is exactly zero
// -- which happens on hosts whose clock granularity is coarser than the
// interval being measured, such as Windows -- the two roundings could disagree
// by an ulp and put the later event a few hundred attoseconds before the
// earlier one. That surfaced as a rare "gap = -0.000s" failure in
// TestRecorder_PauseRemovesGap.
//
// The nanosecond values below are real triggers for the old arithmetic. They
// drive eventTimeLocked directly so the check does not depend on the host
// clock being coarse enough to reproduce the condition naturally.
func TestRecorder_EventTimeNeverGoesBackwards(t *testing.T) {
	cases := []struct {
		name          string
		firstEventAt  time.Duration // after startTime
		pauseDuration time.Duration
	}{
		{"case1", 1292884469, 629424483},
		{"case2", 3532116411, 1856234380},
		{"case3", 3719384267, 289458594},
		{"case4", 4330296842, 1257011184},
		{"case5", 931165873, 1159095051},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Unix(1700000000, 0)
			r := &Recorder{startTime: start, lastEventTime: start}

			// An event, then a pause that begins the instant the event was
			// recorded and ends the instant the next one is, so the pause
			// removes the entire interval between them and the true gap is
			// exactly zero.
			first := r.eventTimeLocked(start.Add(tc.firstEventAt))
			r.timeAdjust += tc.pauseDuration
			second := r.eventTimeLocked(start.Add(tc.firstEventAt + tc.pauseDuration))

			if second < first {
				t.Errorf("event times went backwards: first = %v, second = %v, delta = %v",
					first, second, second-first)
			}
		})
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

// recordChunks records each chunk as an output ("o") or input ("i") event
// with redaction on, then closes the recorder and returns the events. The
// recorder runs fn, if any, after the chunks and before Close.
func recordChunks(t *testing.T, typ string, chunks []string, fn func(*Recorder)) []RecordingEvent {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "redact.cast")
	r, err := NewRecorder(filename, 80, 24, RecordingOptions{RedactSecrets: true})
	assert.NoError(t, err)
	for _, c := range chunks {
		if typ == "i" {
			r.RecordInput(c)
		} else {
			r.RecordOutput(c)
		}
	}
	if fn != nil {
		fn(r)
	}
	assert.NoError(t, r.Close())
	_, events, err := LoadCastFile(filename)
	assert.NoError(t, err)
	return events
}

// eventData joins the data of events of type typ.
func eventData(events []RecordingEvent, typ string) string {
	var b strings.Builder
	for _, e := range events {
		if e.Type == typ {
			b.WriteString(e.Data)
		}
	}
	return b.String()
}

// TestRecorder_RedactsSecretSplitAcrossReads verifies that a secret split
// across PTY reads is joined and redacted whole. PTY reads split output at
// any byte, with no cursor move between the pieces.
func TestRecorder_RedactsSecretSplitAcrossReads(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
		want   string
	}{
		{
			"value written after its label",
			[]string{"Token: ", "s3cr3tvalue\r\n"},
			"Token: [REDACTED]\r\n",
		},
		{
			"read ends in the middle of the value",
			[]string{"export API_KEY=sk-1234", "567890abcdefghij\r\n"},
			"export API_KEY=[REDACTED]\r\n",
		},
		{
			"unlabeled AWS key in two reads",
			[]string{"key AKIAIOSFODNN", "7EXAMPLE done\r\n"},
			"key [REDACTED] done\r\n",
		},
		{
			"style escape split between label and value",
			[]string{"password: \033[1", "mhunter2\033[0m\r\n"},
			"password: \033[1m[REDACTED]\033[0m\r\n",
		},
		{
			"secret at the end of the recording",
			[]string{"password: ", "hunt", "er2"},
			"password: [REDACTED]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := recordChunks(t, "o", tt.chunks, nil)
			assert.Equal(t, eventData(events, "o"), tt.want)
		})
	}
}

// TestRecorder_RedactsInputTypedKeyByKey verifies that input keystrokes are
// matched together, so a secret typed after its key is redacted.
func TestRecorder_RedactsInputTypedKeyByKey(t *testing.T) {
	var keys []string
	for _, c := range "export TOKEN=abc123\r" {
		keys = append(keys, string(c))
	}
	events := recordChunks(t, "i", keys, nil)
	assert.Equal(t, eventData(events, "i"), "export TOKEN=[REDACTED]\r")
}

// TestRecorder_PauseFlushesHeldText verifies that Pause records the text the
// redactor holds back, and that the text before a pause is not joined to the
// text after it.
func TestRecorder_PauseFlushesHeldText(t *testing.T) {
	events := recordChunks(t, "o", []string{"Token: ", "abc"}, func(r *Recorder) {
		r.RecordInput("q")
		r.Pause()
		r.Resume()
		r.RecordOutput("hello")
	})
	assert.Equal(t, eventData(events, "o"), "Token: [REDACTED]hello")
	assert.Equal(t, eventData(events, "i"), "q")
	// The held "abc" and "q" are recorded at the pause, before "hello".
	assert.Equal(t, events[len(events)-1].Data, "hello")
}

// TestRecorder_CloseFlushesHeldText verifies that no output is lost when the
// recording ends in the middle of a run of text.
func TestRecorder_CloseFlushesHeldText(t *testing.T) {
	events := recordChunks(t, "o", []string{"hello ", "wor", "ld"}, nil)
	assert.Equal(t, eventData(events, "o"), "hello world")
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

	var out lockedBuffer
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

// lockedBuffer is a goroutine-safe string buffer for player output.
// (session_e2e_test.go has a similar syncBuffer, but that file is gated to
// darwin/linux; this one must build on every platform.)
type lockedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func TestPlayer_SeekBeforePlay(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "first "},
		{Time: 1.0, Type: "o", Data: "second"},
	})

	var out lockedBuffer
	player, err := NewPlayer(filename, PlayerOptions{Output: &out, Speed: 50.0})
	assert.NoError(t, err)

	player.Seek(1.0)
	assert.NoError(t, player.Play())
	assert.Equal(t, "second", out.String())
}

func TestPlayer_SeekPastEndDuringPlay(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "first"},
		{Time: 5.0, Type: "o", Data: "second"},
	})

	var out lockedBuffer
	player, err := NewPlayer(filename, PlayerOptions{Output: &out, Loop: true})
	assert.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- player.Play() }()

	// With Loop, Play keeps restarting at the first event while several
	// goroutines seek past the end. Play used to release its lock between
	// checking currentIndex and reading events[currentIndex], so a Seek in
	// that gap made it index past the end and panic.
	deadline := time.Now().Add(200 * time.Millisecond)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				player.Seek(100.0)
			}
		}()
	}
	wg.Wait()
	player.Stop()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Play did not return after Stop")
	}
}

func TestPlayer_MaxIdleCapsInitialDelay(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 10.0, Type: "o", Data: "hello"},
	})

	var out lockedBuffer
	player, err := NewPlayer(filename, PlayerOptions{
		Output:  &out,
		MaxIdle: 0.2,
		Speed:   10.0,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0.2, player.GetDuration())

	start := time.Now()
	assert.NoError(t, player.Play())
	assert.Less(t, time.Since(start), 500*time.Millisecond)
	assert.Equal(t, "hello", out.String())
}

func TestRecorder_UpdateSizeWhilePausedRecordedOnResume(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "paused_resize.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{})
	assert.NoError(t, err)

	r.Pause()
	r.UpdateSize(120, 40)
	r.Resume()
	r.RecordOutput("after")
	assert.NoError(t, r.Close())

	_, events, err := LoadCastFile(filename)
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "r", events[0].Type)
	assert.Equal(t, "120x40", events[0].Data)
	assert.Equal(t, "o", events[1].Type)
	assert.Equal(t, "after", events[1].Data)
}
