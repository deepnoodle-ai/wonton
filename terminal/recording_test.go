package terminal

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestDefaultRecordingOptions(t *testing.T) {
	opts := DefaultRecordingOptions()

	assert.True(t, opts.Compress)
	assert.True(t, opts.RedactSecrets)
	assert.Equal(t, float64(0), opts.IdleTimeLimit)
	assert.NotNil(t, opts.Env)
}

func TestRecordingEvent_MarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		event RecordingEvent
		want  string
	}{
		{
			name: "output event",
			event: RecordingEvent{
				Time: 1.5,
				Type: "o",
				Data: "hello",
			},
			want: `[1.5,"o","hello"]`,
		},
		{
			name: "input event",
			event: RecordingEvent{
				Time: 2.75,
				Type: "i",
				Data: "x",
			},
			want: `[2.75,"i","x"]`,
		},
		{
			name: "zero time event",
			event: RecordingEvent{
				Time: 0,
				Type: "o",
				Data: "start",
			},
			want: `[0,"o","start"]`,
		},
		{
			name: "event with special characters",
			event: RecordingEvent{
				Time: 1.0,
				Type: "o",
				Data: "hello\nworld\ttab",
			},
			want: `[1,"o","hello\nworld\ttab"]`,
		},
		{
			name: "event with unicode",
			event: RecordingEvent{
				Time: 1.0,
				Type: "o",
				Data: "你好世界",
			},
			want: `[1,"o","你好世界"]`,
		},
		{
			name: "event with escape sequences",
			event: RecordingEvent{
				Time: 1.0,
				Type: "o",
				Data: "\033[31mred\033[0m",
			},
			want: `[1,"o","\u001b[31mred\u001b[0m"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestRecorder_RecordOutput(t *testing.T) {
	// Create a recorder directly to test RecordOutput
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
	}

	// Create a temp file for the writer
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Test basic output recording
	r.RecordOutput("hello world")

	// Flush and verify
	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "hello world")
}

func TestRecorder_RecordOutput_Nil(t *testing.T) {
	// Test nil recorder doesn't panic
	var r *Recorder = nil
	r.RecordOutput("test") // Should not panic
}

func TestRecorder_RecordOutput_Paused(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
		paused:        true,
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Record while paused - should be ignored
	r.RecordOutput("should not appear")

	// Flush and verify nothing was written
	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, "", string(content))
}

func TestRecorder_RecordOutput_EmptyData(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Record empty data - should be ignored
	r.RecordOutput("")

	// Flush and verify nothing was written
	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, "", string(content))
}

func TestRecorder_RecordInput(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Test basic input recording
	r.RecordInput("user input")

	// Flush and verify
	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "user input")
	assert.Contains(t, string(content), `"i"`)
}

func TestRecorder_RecordInput_Nil(t *testing.T) {
	// Test nil recorder doesn't panic
	var r *Recorder = nil
	r.RecordInput("test") // Should not panic
}

func TestRecorder_RecordInput_Paused(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
		paused:        true,
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Record while paused
	r.RecordInput("should not appear")

	// Flush and verify nothing was written
	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, "", string(content))
}

func TestRecorder_RecordInput_EmptyData(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Record empty data
	r.RecordInput("")

	// Flush and verify nothing was written
	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, "", string(content))
}

func TestRecorder_IdleTimeLimit(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now().Add(-10 * time.Second), // Started 10 seconds ago
		lastEventTime: time.Now().Add(-5 * time.Second),  // Last event 5 seconds ago
		idleTimeLimit: 2.0,                               // Max 2 seconds idle
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Record - should clamp the idle time
	r.RecordOutput("after idle")

	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)

	// Parse the event to check the time was clamped
	var event []interface{}
	err = json.Unmarshal(content[:len(content)-1], &event) // Remove trailing newline
	assert.NoError(t, err)

	eventTime := event[0].(float64)
	// Time should be around 5 + 2 = 7 seconds (not 10+)
	assert.True(t, eventTime <= 8.0, "Event time should be clamped, got %f", eventTime)
}

func TestRecorder_SecretRedaction(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
		redactSecrets: true,
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Record output with a secret
	r.RecordOutput("password: supersecret123")

	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)

	// Secret should be redacted
	assert.NotContains(t, string(content), "supersecret123")
	assert.Contains(t, string(content), "[REDACTED]")
}

func TestRecorder_NoSecretRedaction(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
		redactSecrets: false,
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test_recording_*.cast")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	r.file = tmpFile
	r.writer = bufio.NewWriter(tmpFile)

	// Record output with a secret
	r.RecordOutput("password: supersecret123")

	r.writer.Flush()
	tmpFile.Seek(0, 0)
	content, err := io.ReadAll(tmpFile)
	assert.NoError(t, err)

	// Secret should NOT be redacted when disabled
	assert.Contains(t, string(content), "supersecret123")
}

func TestRecordingHeader_JSON(t *testing.T) {
	header := RecordingHeader{
		Version:   2,
		Width:     80,
		Height:    24,
		Timestamp: 1609459200, // 2021-01-01 00:00:00 UTC
		Title:     "Test Recording",
		Env: map[string]string{
			"SHELL": "/bin/bash",
			"TERM":  "xterm-256color",
		},
	}

	data, err := json.Marshal(header)
	assert.NoError(t, err)

	var decoded RecordingHeader
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, header.Version, decoded.Version)
	assert.Equal(t, header.Width, decoded.Width)
	assert.Equal(t, header.Height, decoded.Height)
	assert.Equal(t, header.Timestamp, decoded.Timestamp)
	assert.Equal(t, header.Title, decoded.Title)
	assert.Equal(t, header.Env["SHELL"], decoded.Env["SHELL"])
	assert.Equal(t, header.Env["TERM"], decoded.Env["TERM"])
}

func TestRecordingHeader_EmptyEnv(t *testing.T) {
	header := RecordingHeader{
		Version:   2,
		Width:     80,
		Height:    24,
		Timestamp: 1609459200,
	}

	data, err := json.Marshal(header)
	assert.NoError(t, err)

	// Empty env should be omitted from JSON
	assert.NotContains(t, string(data), "env")
}

func TestTerminal_StartStopRecording(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.cast")

	opts := RecordingOptions{
		Compress:      false,
		RedactSecrets: false,
		Title:         "Test",
	}

	// Start recording
	err := term.StartRecording(filename, opts)
	assert.NoError(t, err)

	// Should be recording now
	assert.True(t, term.IsRecording())

	// Stop recording
	err = term.StopRecording()
	assert.NoError(t, err)

	// Should not be recording now
	assert.False(t, term.IsRecording())

	// File should exist and have header
	content, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Contains(t, string(content), `"version":2`)
	assert.Contains(t, string(content), `"width":80`)
	assert.Contains(t, string(content), `"height":24`)
}

func TestTerminal_StartRecording_AlreadyRecording(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	tmpDir := t.TempDir()
	filename1 := filepath.Join(tmpDir, "test1.cast")
	filename2 := filepath.Join(tmpDir, "test2.cast")

	opts := RecordingOptions{Compress: false}

	// Start first recording
	err := term.StartRecording(filename1, opts)
	assert.NoError(t, err)
	defer term.StopRecording()

	// Try to start second recording
	err = term.StartRecording(filename2, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in progress")
}

func TestTerminal_StopRecording_NotRecording(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	// Stop recording when not recording - should not error
	err := term.StopRecording()
	assert.NoError(t, err)
}

func TestTerminal_PauseResumeRecording(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.cast")

	opts := RecordingOptions{Compress: false}

	err := term.StartRecording(filename, opts)
	assert.NoError(t, err)
	defer term.StopRecording()

	// Pause recording
	term.PauseRecording()

	// Resume recording
	term.ResumeRecording()
}

func TestTerminal_PauseRecording_NotRecording(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	// Should not panic when not recording
	term.PauseRecording()
	term.ResumeRecording()
}

func TestTerminal_IsRecording(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	// Initially not recording
	assert.False(t, term.IsRecording())

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.cast")

	opts := RecordingOptions{Compress: false}

	// Start recording
	err := term.StartRecording(filename, opts)
	assert.NoError(t, err)

	// Now recording
	assert.True(t, term.IsRecording())

	// Stop recording
	term.StopRecording()

	// Not recording again
	assert.False(t, term.IsRecording())
}

func TestTerminal_Recording_WithCompression(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.cast.gz")

	opts := RecordingOptions{
		Compress:      true,
		RedactSecrets: false,
		Title:         "Compressed Test",
	}

	err := term.StartRecording(filename, opts)
	assert.NoError(t, err)

	// Write some output (would need to access recorder directly)
	// For now just verify we can start/stop
	err = term.StopRecording()
	assert.NoError(t, err)

	// File should exist
	_, err = os.Stat(filename)
	assert.NoError(t, err)

	// Read and decompress
	file, err := os.Open(filename)
	assert.NoError(t, err)
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	assert.NoError(t, err)
	defer gzReader.Close()

	content, err := io.ReadAll(gzReader)
	assert.NoError(t, err)

	// Should have valid header
	assert.Contains(t, string(content), `"version":2`)
}

func TestTerminal_Recording_InvalidPath(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	// Try to record to an invalid path
	opts := RecordingOptions{Compress: false}
	err := term.StartRecording("/nonexistent/path/file.cast", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create")
}

func TestTerminal_Recording_WithTitle(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.cast")

	opts := RecordingOptions{
		Compress: false,
		Title:    "My Test Recording",
	}

	err := term.StartRecording(filename, opts)
	assert.NoError(t, err)

	err = term.StopRecording()
	assert.NoError(t, err)

	content, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Contains(t, string(content), `"title":"My Test Recording"`)
}

func TestTerminal_Recording_WithEnv(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.cast")

	opts := RecordingOptions{
		Compress: false,
		Env: map[string]string{
			"TERM":  "xterm-256color",
			"SHELL": "/bin/zsh",
		},
	}

	err := term.StartRecording(filename, opts)
	assert.NoError(t, err)

	err = term.StopRecording()
	assert.NoError(t, err)

	content, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Contains(t, string(content), `"TERM":"xterm-256color"`)
	assert.Contains(t, string(content), `"SHELL":"/bin/zsh"`)
}
