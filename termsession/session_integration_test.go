//go:build darwin || linux

package termsession

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Integration tests - these actually spawn processes via PTY.
// Gated to darwin/linux because creack/pty has no Windows support.

func TestSession_Integration_SimpleCommand(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"echo", "hello world"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Start()
	assert.NoError(t, err)

	err = s.Wait()
	assert.NoError(t, err)
	assert.Equal(t, 0, s.ExitCode())
}

func TestSession_Integration_ExitCode(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"sh", "-c", "exit 42"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Start()
	assert.NoError(t, err)

	err = s.Wait()
	assert.Error(t, err) // Non-zero exit is an error
	assert.Equal(t, 42, s.ExitCode())
}

func TestSession_Integration_ExitCodeZero(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"sh", "-c", "exit 0"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Start()
	assert.NoError(t, err)

	err = s.Wait()
	assert.NoError(t, err)
	assert.Equal(t, 0, s.ExitCode())
}

func TestSession_Integration_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()

	s, err := NewSession(SessionOptions{
		Command: []string{"pwd"},
		Dir:     dir,
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Start()
	assert.NoError(t, err)

	err = s.Wait()
	assert.NoError(t, err)
	assert.Equal(t, 0, s.ExitCode())
}

func TestSession_Integration_Environment(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"sh", "-c", "test \"$TEST_VAR\" = \"test_value\""},
		Env:     []string{"TEST_VAR=test_value"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Start()
	assert.NoError(t, err)

	err = s.Wait()
	assert.NoError(t, err)
	assert.Equal(t, 0, s.ExitCode())
}

func TestSession_Integration_Resize(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"sleep", "0.1"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Start()
	assert.NoError(t, err)

	// Resize after start should succeed
	err = s.Resize(120, 40)
	assert.NoError(t, err)

	err = s.Wait()
	assert.NoError(t, err)
}

func TestSession_Integration_Record(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	s, err := NewSession(SessionOptions{
		// The trailing sleep keeps the PTY slave open long enough for
		// copyOutput to drain the master; if the child exits immediately
		// after writing, the kernel (macOS in particular) may discard
		// unread PTY output when the slave side closes.
		Command: []string{"sh", "-c", "echo recorded output; sleep 0.2"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Record(filename, RecordingOptions{
		Compress: false,
		Title:    "Test Recording",
	})
	assert.NoError(t, err)
	assert.True(t, s.IsRecording())

	err = s.Wait()
	assert.NoError(t, err)

	// Give a moment for file to be flushed
	time.Sleep(50 * time.Millisecond)

	// Verify the recording file exists and has content
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Greater(t, len(data), 0)

	// Parse header
	lines := bytes.Split(data, []byte("\n"))
	assert.GreaterOrEqual(t, len(lines), 2)

	var header RecordingHeader
	err = json.Unmarshal(lines[0], &header)
	assert.NoError(t, err)
	assert.Equal(t, 2, header.Version)
	assert.Equal(t, "Test Recording", header.Title)

	// Verify output was recorded
	content := string(data)
	assert.Contains(t, content, "recorded output")
}

func TestSession_Integration_RecordCompressed(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast.gz")

	s, err := NewSession(SessionOptions{
		// Trailing sleep: see TestSession_Integration_Record.
		Command: []string{"sh", "-c", "echo compressed recording; sleep 0.2"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Record(filename, RecordingOptions{
		Compress: true,
	})
	assert.NoError(t, err)

	err = s.Wait()
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Verify gzip compressed
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

	assert.Contains(t, buf.String(), "compressed recording")
}

func TestSession_Integration_RecordPauseResume(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	s, err := NewSession(SessionOptions{
		Command: []string{"sh", "-c", "echo before; sleep 0.05; echo after"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Record(filename, RecordingOptions{
		Compress: false,
	})
	assert.NoError(t, err)

	// Pause and resume during execution
	time.Sleep(10 * time.Millisecond)
	s.PauseRecording()
	time.Sleep(10 * time.Millisecond)
	s.ResumeRecording()

	err = s.Wait()
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Verify file exists
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Greater(t, len(data), 0)
}

func TestSession_Integration_RecordWithResize(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	s, err := NewSession(SessionOptions{
		Command: []string{"sleep", "0.1"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Record(filename, RecordingOptions{
		Compress: false,
	})
	assert.NoError(t, err)

	// Resize during recording
	err = s.Resize(100, 30)
	assert.NoError(t, err)

	err = s.Wait()
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Verify recording file exists
	_, err = os.Stat(filename)
	assert.NoError(t, err)
}

func TestSession_Integration_InvalidCommand(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"/nonexistent/command/that/does/not/exist"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Start()
	assert.Error(t, err)
}

func TestSession_Integration_CloseBeforeWait(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"sleep", "10"},
	})
	assert.NoError(t, err)

	err = s.Start()
	assert.NoError(t, err)

	// Close immediately without waiting
	err = s.Close()
	assert.NoError(t, err)
}

func TestSession_Integration_MultipleCommands(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"sh", "-c", "echo first && echo second && echo third"},
	})
	assert.NoError(t, err)
	defer s.Close()

	err = s.Start()
	assert.NoError(t, err)

	err = s.Wait()
	assert.NoError(t, err)
	assert.Equal(t, 0, s.ExitCode())
}
