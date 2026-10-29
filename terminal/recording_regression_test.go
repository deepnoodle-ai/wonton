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

func TestRecordingRecordsScreenModes(t *testing.T) {
	var screen bytes.Buffer
	term := NewTestTerminal(20, 3, &screen)
	term.buffered = false
	filename := filepath.Join(t.TempDir(), "out.cast")
	assert.NoError(t, term.StartRecording(filename, RecordingOptions{}))

	term.EnableAlternateScreen()
	term.HideCursor()
	term.Clear()
	term.MoveCursor(2, 1)
	term.EnableMouseTracking()
	term.EnableBracketedPaste()
	term.ShowCursor()
	term.DisableAlternateScreen()
	assert.NoError(t, term.StopRecording())

	recorded := strings.Join(readCastEvents(t, filename), "")
	assert.Equal(t, recorded, "\033[?1049h\033[?25l\033[2J\033[2;3H\033[?25h\033[?1049l")
	// Input modes reach the terminal but stay out of the recording.
	assert.Contains(t, screen.String(), "\033[?1000h")
	assert.Contains(t, screen.String(), "\033[?2004h")
}

func TestRecordingStartsInCurrentScreenMode(t *testing.T) {
	term := NewTestTerminal(20, 3, &bytes.Buffer{})
	term.EnableAlternateScreen()
	term.HideCursor()
	filename := filepath.Join(t.TempDir(), "out.cast")
	assert.NoError(t, term.StartRecording(filename, RecordingOptions{}))
	assert.NoError(t, term.StopRecording())

	assert.Equal(t, readCastEvents(t, filename), []string{"\033[?1049h", "\033[?25l"})
}

func TestRecordingRedactsValueWrittenAfterLabelNonBuffered(t *testing.T) {
	term := NewTestTerminal(30, 2, &bytes.Buffer{})
	term.buffered = false
	filename := filepath.Join(t.TempDir(), "out.cast")
	assert.NoError(t, term.StartRecording(filename, RecordingOptions{RedactSecrets: true}))

	term.PrintAtStyled(0, 0, "Token: ", NewStyle().WithForeground(ColorCyan))
	term.PrintAtStyled(7, 0, "s3cr3tvalue", NewStyle())
	term.PrintAtStyled(0, 1, "next line", NewStyle())
	assert.NoError(t, term.StopRecording())

	recorded := strings.Join(readCastEvents(t, filename), "")
	assert.NotContains(t, recorded, "s3cr3tvalue")
	assert.Contains(t, recorded, "Token: \033[0m")
	assert.Contains(t, recorded, "H[REDACTED]")
	// Text drawn after the value is not joined to it.
	assert.Contains(t, recorded, "Hnext line")
}

func TestRedactorKeepsEscapeSequences(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"style reset after the secret is kept",
			"\033[31mpassword: \033[1mhunter2\033[0m done",
			"\033[31mpassword: \033[1m[REDACTED]\033[0m done",
		},
		{
			"secret split by a style change is replaced once",
			"\033[1;1Hapi_key=abc\033[32mdef\033[0m",
			"\033[1;1Hapi_key=[REDACTED]\033[32m\033[0m",
		},
		{
			"value after a cursor move is redacted, text after the next move is not",
			"password:\033[1;10Hhunter2\033[2;1Hnext",
			"password:\033[1;10H[REDACTED]\033[2;1Hnext",
		},
		{
			"hyperlink URL is redacted inside the sequence",
			"\033]8;;https://x.test/?token=abc123\033\\link\033]8;;\033\\",
			"\033]8;;https://x.test/?token=[REDACTED]\033\\link\033]8;;\033\\",
		},
	}
	for _, tt := range tests {
		var r redactor
		assert.Equal(t, r.redact(tt.in), tt.want, tt.name)
	}
}

func TestRedactorSecretSplitAcrossEvents(t *testing.T) {
	var r redactor
	assert.Equal(t, r.redact("Authorization: Bearer "), "Authorization: Bearer ")
	assert.Equal(t, r.redact("dGhpcyBpcy"), "[REDACTED]")
	assert.Equal(t, r.redact("BhIHNlY3JldA== ok"), "[REDACTED] ok")
	assert.Equal(t, r.redact("plain text"), "plain text")
}

func TestRecordingPauseDropsRedactionContext(t *testing.T) {
	term := NewTestTerminal(30, 2, &bytes.Buffer{})
	term.buffered = false
	filename := filepath.Join(t.TempDir(), "out.cast")
	assert.NoError(t, term.StartRecording(filename, RecordingOptions{RedactSecrets: true}))

	term.PrintStyled("Token: ", NewStyle())
	term.PauseRecording()
	term.ResumeRecording()
	term.PrintStyled("hello", NewStyle())
	assert.NoError(t, term.StopRecording())

	recorded := strings.Join(readCastEvents(t, filename), "")
	assert.Contains(t, recorded, "hello")
}
