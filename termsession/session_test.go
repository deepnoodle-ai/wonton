package termsession

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNewSession_Defaults(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Len(t, s.command, 0)
	assert.Equal(t, "", s.dir)
	assert.Len(t, s.env, 0)
	assert.NotNil(t, s.done)
	assert.False(t, s.started)
}

func TestNewSession_WithCommand(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"echo", "hello"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, []string{"echo", "hello"}, s.command)
}

func TestNewSession_WithDir(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Dir: "/tmp",
	})
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, "/tmp", s.dir)
}

func TestNewSession_WithEnv(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Env: []string{"FOO=bar", "BAZ=qux"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, []string{"FOO=bar", "BAZ=qux"}, s.env)
}

func TestNewSession_AllOptions(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"bash", "-c", "echo test"},
		Dir:     "/tmp",
		Env:     []string{"TERM=xterm-256color"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, []string{"bash", "-c", "echo test"}, s.command)
	assert.Equal(t, "/tmp", s.dir)
	assert.Equal(t, []string{"TERM=xterm-256color"}, s.env)
}

func TestSession_ExitCode_Default(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, s.ExitCode())
}

func TestSession_IsRecording_NoRecorder(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)
	assert.False(t, s.IsRecording())
}

func TestSession_PauseResume_NoRecorder(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)

	// Should not panic when no recorder is set
	s.PauseRecording()
	s.ResumeRecording()
}

func TestSession_Resize_NotStarted(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)

	err = s.Resize(100, 50)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not started")
}

func TestSession_Close_NotStarted(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)

	// Close on unstarted session should not error
	err = s.Close()
	assert.NoError(t, err)
}

func TestSession_Close_Idempotent(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)

	// Multiple closes should be safe
	err = s.Close()
	assert.NoError(t, err)
	err = s.Close()
	assert.NoError(t, err)
}

func TestSession_Start_AlreadyStarted(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)

	// Manually set started flag to simulate already-started session
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	err = s.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session already started")
}

func TestSession_Record_AlreadyStarted(t *testing.T) {
	s, err := NewSession(SessionOptions{})
	assert.NoError(t, err)

	// Manually set started flag to simulate already-started session
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	err = s.Record("/tmp/test.cast", RecordingOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session already started")
}

// Integration tests - these actually spawn processes via PTY

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
		Command: []string{"echo", "recorded output"},
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
		Command: []string{"echo", "compressed recording"},
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

// ExampleNewSession demonstrates creating a PTY session.
func ExampleNewSession() {
	// Create a session that will run a command
	session, err := NewSession(SessionOptions{
		Command: []string{"bash", "-c", "exit 0"},
		Dir:     "/tmp",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Session created with command: %v\n", session.command)
	fmt.Printf("Working directory: %s\n", session.dir)

	// Output:
	// Session created with command: [bash -c exit 0]
	// Working directory: /tmp
}
