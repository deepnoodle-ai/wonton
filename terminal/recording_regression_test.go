package terminal

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestRedactCredentials_SecretContainsDelimiters(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Authorization: Bearer dGhpcyBpcyBhIHNlY3JldA==", "Authorization: Bearer [REDACTED]"},
		{"password=my_secret_prefix:rest", "password=[REDACTED]"},
		{"TOKEN=https://user:pass@example.com", "TOKEN=[REDACTED]"},
		{"password: hunter2", "password: [REDACTED]"},
		{"password hunter2 next", "password [REDACTED] next"},
		{"api_key=abc=def", "api_key=[REDACTED]"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, RedactCredentials(tt.in), "input %q", tt.in)
	}
}

// readCastEvents returns the output events of an uncompressed .cast file.
func readCastEvents(t *testing.T, filename string) []string {
	t.Helper()
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var out []string
	for _, line := range lines[1:] {
		var ev []any
		assert.NoError(t, json.Unmarshal([]byte(line), &ev))
		if ev[1] == "o" {
			out = append(out, ev[2].(string))
		}
	}
	return out
}

func TestRecordingBufferedModeRecordsWhatIsDrawn(t *testing.T) {
	var screen bytes.Buffer
	term := NewTestTerminal(20, 3, &screen)
	filename := filepath.Join(t.TempDir(), "out.cast")
	assert.NoError(t, term.StartRecording(filename, RecordingOptions{}))

	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	frame.SubFrame(image.Rect(0, 0, 5, 1)).PrintStyled(0, 0, "Hello World", NewStyle())
	frame.SetCell(0, 1, 'Z', NewStyle())
	frame.FillStyled(0, 2, 3, 1, 'Q', NewStyle())
	assert.NoError(t, term.EndFrame(frame))
	assert.NoError(t, term.StopRecording())

	recorded := strings.Join(readCastEvents(t, filename), "")
	assert.Equal(t, 1, strings.Count(recorded, "Hello"), "recorded: %q", recorded)
	assert.False(t, strings.Contains(recorded, "World"), "clipped text was recorded: %q", recorded)
	assert.True(t, strings.Contains(recorded, "Z"), "SetCell missing: %q", recorded)
	assert.True(t, strings.Contains(recorded, "QQQ"), "FillStyled missing: %q", recorded)
	// The recording carries the same bytes the terminal received.
	assert.Equal(t, screen.String(), recorded)
}

func TestRecorderPauseClampsIdleBeforePause(t *testing.T) {
	now := time.Now()
	r := &Recorder{
		startTime:     now.Add(-2 * time.Second),
		lastEventTime: now.Add(-2 * time.Second), // last event 2s ago
		idleTimeLimit: 0.05,
	}
	term := NewTestTerminal(10, 2, &bytes.Buffer{})
	term.recorder = r
	term.PauseRecording()
	term.ResumeRecording()

	r.mu.Lock()
	got := r.eventTimeLocked(time.Now())
	r.mu.Unlock()
	if got > 0.2 {
		t.Errorf("event time = %.3fs, want about 0.05s (idle before pause must be clamped)", got)
	}
}

func TestRecordingRedactsValueRedrawnWithoutLabel(t *testing.T) {
	term := NewTestTerminal(30, 2, &bytes.Buffer{})
	filename := filepath.Join(t.TempDir(), "out.cast")
	assert.NoError(t, term.StartRecording(filename, RecordingOptions{RedactSecrets: true}))

	drawFrame := func(draw func(f RenderFrame)) {
		frame, err := term.BeginFrame()
		assert.NoError(t, err)
		draw(frame)
		assert.NoError(t, term.EndFrame(frame))
	}
	drawFrame(func(f RenderFrame) { f.PrintStyled(0, 0, "Token: abc", NewStyle()) })
	// Only the value cells change in the second frame.
	drawFrame(func(f RenderFrame) { f.PrintStyled(7, 0, "s3cr3tvalue", NewStyle()) })
	assert.NoError(t, term.StopRecording())

	recorded := strings.Join(readCastEvents(t, filename), "")
	assert.False(t, strings.Contains(recorded, "s3cr3tvalue"), "secret recorded in plain text: %q", recorded)
	assert.False(t, strings.Contains(recorded, "abc"), "secret recorded in plain text: %q", recorded)
	assert.True(t, strings.Contains(recorded, "Token: [REDACTED]"), "recorded: %q", recorded)
}
