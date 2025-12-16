package termsession

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func createTestRecording(t *testing.T, events []RecordingEvent) string {
	t.Helper()

	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress: false,
		Title:    "Test",
	})
	assert.NoError(t, err)

	// Manually write events with specific timing
	r.mu.Lock()
	for _, e := range events {
		r.writeEvent(e)
	}
	r.mu.Unlock()

	err = r.Close()
	assert.NoError(t, err)

	return filename
}

func TestPlayer_LoadRecording(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "Hello\n"},
		{Time: 0.5, Type: "o", Data: "World\n"},
	})

	player, err := NewPlayer(filename, PlayerOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, player)

	header := player.GetHeader()
	assert.Equal(t, 80, header.Width)
	assert.Equal(t, 24, header.Height)
	assert.Equal(t, "Test", header.Title)
	assert.Equal(t, 2, player.EventCount())
}

func TestPlayer_LoadGzipRecording(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress: true,
	})
	assert.NoError(t, err)
	r.RecordOutput("Gzip test\n")
	err = r.Close()
	assert.NoError(t, err)

	player, err := NewPlayer(filename, PlayerOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, player.EventCount())
}

func TestPlayer_Play(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "Line 1\n"},
		{Time: 0.05, Type: "o", Data: "Line 2\n"},
	})

	var buf bytes.Buffer
	player, err := NewPlayer(filename, PlayerOptions{
		Output: &buf,
		Speed:  10.0, // Fast playback for testing
	})
	assert.NoError(t, err)

	err = player.Play()
	assert.NoError(t, err)

	assert.Equal(t, "Line 1\nLine 2\n", buf.String())
}

func TestPlayer_Speed(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "Start\n"},
		{Time: 0.1, Type: "o", Data: "End\n"},
	})

	var buf bytes.Buffer
	player, err := NewPlayer(filename, PlayerOptions{
		Output: &buf,
		Speed:  10.0, // 10x speed
	})
	assert.NoError(t, err)

	start := time.Now()
	err = player.Play()
	assert.NoError(t, err)
	elapsed := time.Since(start)

	// At 10x speed, 0.1s should take ~0.01s
	// Allow generous tolerance for CI environments
	assert.Less(t, elapsed, 100*time.Millisecond)
	assert.Equal(t, "Start\nEnd\n", buf.String())
}

func TestPlayer_Stop(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "Line 1\n"},
		{Time: 1.0, Type: "o", Data: "Line 2\n"}, // 1 second delay
		{Time: 2.0, Type: "o", Data: "Line 3\n"},
	})

	var buf bytes.Buffer
	player, err := NewPlayer(filename, PlayerOptions{
		Output: &buf,
	})
	assert.NoError(t, err)

	// Stop after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		player.Stop()
	}()

	err = player.Play()
	assert.NoError(t, err)

	// Should have stopped before all events played
	assert.Contains(t, buf.String(), "Line 1")
}

func TestPlayer_PauseResume(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "A"},
		{Time: 0.05, Type: "o", Data: "B"},
	})

	var buf bytes.Buffer
	player, err := NewPlayer(filename, PlayerOptions{
		Output: &buf,
		Speed:  10.0,
	})
	assert.NoError(t, err)

	assert.False(t, player.IsPaused())
	player.Pause()
	assert.True(t, player.IsPaused())
	player.Resume()
	assert.False(t, player.IsPaused())
}

func TestPlayer_GetDuration(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "Start"},
		{Time: 5.0, Type: "o", Data: "End"},
	})

	player, err := NewPlayer(filename, PlayerOptions{})
	assert.NoError(t, err)

	assert.Equal(t, 5.0, player.GetDuration())
}

func TestPlayer_MaxIdle(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "A"},
		{Time: 10.0, Type: "o", Data: "B"}, // 10 second gap
		{Time: 10.1, Type: "o", Data: "C"},
	})

	var buf bytes.Buffer
	player, err := NewPlayer(filename, PlayerOptions{
		Output:  &buf,
		Speed:   100.0, // Fast
		MaxIdle: 0.5,   // Cap idle at 0.5s
	})
	assert.NoError(t, err)

	start := time.Now()
	err = player.Play()
	assert.NoError(t, err)
	elapsed := time.Since(start)

	// With maxIdle=0.5 and speed=100, the 10s gap becomes 0.5s/100 = 5ms
	assert.Less(t, elapsed, 100*time.Millisecond)
	assert.Equal(t, "ABC", buf.String())
}

func TestPlayer_InputEventsIgnored(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "Output"},
		{Time: 0.1, Type: "i", Data: "Input"},
	})

	var buf bytes.Buffer
	player, err := NewPlayer(filename, PlayerOptions{
		Output: &buf,
		Speed:  10.0,
	})
	assert.NoError(t, err)

	err = player.Play()
	assert.NoError(t, err)

	assert.Equal(t, "Output", buf.String())
}

func TestPlayer_EmptyRecording(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{})

	player, err := NewPlayer(filename, PlayerOptions{})
	assert.NoError(t, err)

	assert.Equal(t, 0.0, player.GetDuration())
	assert.Equal(t, 0, player.EventCount())

	err = player.Play()
	assert.NoError(t, err)
}

func TestPlayer_InvalidFile(t *testing.T) {
	_, err := NewPlayer("/nonexistent/file.cast", PlayerOptions{})
	assert.Error(t, err)
}

func TestPlayer_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "invalid.cast")

	err := os.WriteFile(filename, []byte("not json"), 0644)
	assert.NoError(t, err)

	_, err = NewPlayer(filename, PlayerOptions{})
	assert.Error(t, err)
}

func TestPlayer_SetSpeed(t *testing.T) {
	filename := createTestRecording(t, []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "Test"},
	})

	player, err := NewPlayer(filename, PlayerOptions{Speed: 1.0})
	assert.NoError(t, err)

	assert.Equal(t, 1.0, player.Speed())
	player.SetSpeed(2.0)
	assert.Equal(t, 2.0, player.Speed())

	// Invalid speed should be ignored
	player.SetSpeed(0)
	assert.Equal(t, 2.0, player.Speed())
	player.SetSpeed(-1)
	assert.Equal(t, 2.0, player.Speed())
}

func TestDefaultPlayerOptions(t *testing.T) {
	opts := DefaultPlayerOptions()
	assert.Equal(t, 1.0, opts.Speed)
	assert.False(t, opts.Loop)
	assert.Equal(t, 0.0, opts.MaxIdle)
	assert.Equal(t, os.Stdout, opts.Output)
}
