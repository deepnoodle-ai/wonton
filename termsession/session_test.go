package termsession

import (
	"fmt"
	"testing"

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
