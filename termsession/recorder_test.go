package termsession

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestRecorder_BasicRecording(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	// Create recorder without compression for easier testing
	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress: false,
		Title:    "Test Recording",
	})
	assert.NoError(t, err)

	// Record some output
	r.RecordOutput("Hello, World!\n")
	r.RecordOutput("Second line\n")

	// Close recorder
	err = r.Close()
	assert.NoError(t, err)

	// Read and verify the file
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	lines := bytes.Split(data, []byte("\n"))
	assert.GreaterOrEqual(t, len(lines), 3) // header + 2 events + empty

	// Parse header
	var header RecordingHeader
	err = json.Unmarshal(lines[0], &header)
	assert.NoError(t, err)
	assert.Equal(t, 2, header.Version)
	assert.Equal(t, 80, header.Width)
	assert.Equal(t, 24, header.Height)
	assert.Equal(t, "Test Recording", header.Title)

	// Parse first event
	var event1 []interface{}
	err = json.Unmarshal(lines[1], &event1)
	assert.NoError(t, err)
	assert.Len(t, event1, 3)
	assert.Equal(t, "o", event1[1])
	assert.Equal(t, "Hello, World!\n", event1[2])
}

func TestRecorder_GzipCompression(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress: true,
	})
	assert.NoError(t, err)

	r.RecordOutput("Compressed output\n")
	err = r.Close()
	assert.NoError(t, err)

	// Verify it's gzip compressed
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, byte(0x1f), data[0], "should have gzip magic byte 1")
	assert.Equal(t, byte(0x8b), data[1], "should have gzip magic byte 2")

	// Decompress and verify content
	gr, err := gzip.NewReader(bytes.NewReader(data))
	assert.NoError(t, err)
	defer gr.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(gr)
	assert.NoError(t, err)

	assert.Contains(t, buf.String(), "Compressed output")
}

func TestRecorder_Pause(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress: false,
	})
	assert.NoError(t, err)

	r.RecordOutput("Before pause\n")
	r.Pause()
	assert.True(t, r.IsPaused())

	r.RecordOutput("During pause\n") // Should be ignored
	r.Resume()
	assert.False(t, r.IsPaused())

	r.RecordOutput("After pause\n")
	err = r.Close()
	assert.NoError(t, err)

	// Verify content
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Before pause")
	assert.NotContains(t, content, "During pause")
	assert.Contains(t, content, "After pause")
}

func TestRecorder_InputRecording(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress: false,
	})
	assert.NoError(t, err)

	r.RecordInput("user input")
	err = r.Close()
	assert.NoError(t, err)

	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	lines := bytes.Split(data, []byte("\n"))
	assert.GreaterOrEqual(t, len(lines), 2)

	// Parse input event
	var event []interface{}
	err = json.Unmarshal(lines[1], &event)
	assert.NoError(t, err)
	assert.Equal(t, "i", event[1]) // Input type
	assert.Equal(t, "user input", event[2])
}

func TestRecorder_SecretRedaction(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress:      false,
		RedactSecrets: true,
	})
	assert.NoError(t, err)

	r.RecordOutput("password: mysecret123\n")
	r.RecordOutput("api_key=abc123def456ghi789jkl012mno345pqr678\n")
	err = r.Close()
	assert.NoError(t, err)

	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	content := string(data)
	assert.NotContains(t, content, "mysecret123")
	assert.NotContains(t, content, "abc123def456ghi789jkl012mno345pqr678")
	assert.Contains(t, content, "[REDACTED]")
}

func TestRecorder_NilSafe(t *testing.T) {
	var r *Recorder

	// All these should not panic
	r.RecordOutput("test")
	r.RecordInput("test")
	r.Pause()
	r.Resume()
	assert.False(t, r.IsPaused())
	assert.NoError(t, r.Flush())
	assert.NoError(t, r.Close())
}

func TestDefaultRecordingOptions(t *testing.T) {
	opts := DefaultRecordingOptions()
	assert.True(t, opts.Compress)
	assert.True(t, opts.RedactSecrets)
	assert.Equal(t, float64(0), opts.IdleTimeLimit)
	assert.NotNil(t, opts.Env)
}

// ExampleNewRecorder demonstrates basic recording usage.
func ExampleNewRecorder() {
	// Create a temporary file for the recording
	tmpfile := filepath.Join(os.TempDir(), "example.cast")
	defer os.Remove(tmpfile)

	// Create a recorder
	recorder, err := NewRecorder(tmpfile, 80, 24, RecordingOptions{
		Compress: false,
		Title:    "Example Recording",
	})
	if err != nil {
		panic(err)
	}
	defer recorder.Close()

	// Record some output
	recorder.RecordOutput("Hello, ")
	recorder.RecordOutput("World!\n")

	// Output:
}

// ExampleRecorder_Pause demonstrates pausing and resuming recording.
func ExampleRecorder_Pause() {
	tmpfile := filepath.Join(os.TempDir(), "pause-example.cast")
	defer os.Remove(tmpfile)

	recorder, err := NewRecorder(tmpfile, 80, 24, RecordingOptions{
		Compress: false,
	})
	if err != nil {
		panic(err)
	}
	defer recorder.Close()

	recorder.RecordOutput("Before pause\n")
	recorder.Pause()
	recorder.RecordOutput("During pause (not recorded)\n")
	recorder.Resume()
	recorder.RecordOutput("After resume\n")

	// Output:
}
