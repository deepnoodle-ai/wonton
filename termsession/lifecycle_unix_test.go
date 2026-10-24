//go:build darwin || linux

package termsession

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// TestSession_NaturalExitReleasesPTY verifies that a session whose command
// exits on its own closes the PTY master and ends the recording, without a
// call to Close.
func TestSession_NaturalExitReleasesPTY(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"true"},
		Input:   bytes.NewReader(nil),
		Output:  &bytes.Buffer{},
	})
	assert.NoError(t, err)

	assert.NoError(t, s.Record(filepath.Join(t.TempDir(), "a.cast"), RecordingOptions{}))
	assert.NoError(t, s.Wait())

	assert.False(t, s.IsRecording(), "IsRecording must be false after the command exits")
	s.mu.Lock()
	ptmx := s.pty
	s.mu.Unlock()
	assert.Nil(t, ptmx, "PTY master must be closed after the command exits")

	assert.NoError(t, s.Close())
}
