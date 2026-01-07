package tty_test

import (
	"os"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/tty"
)

func TestIsTerminal(t *testing.T) {
	// 1. Test with a temporary file (should not be a terminal)
	tmpFile, err := os.CreateTemp("", "not-a-tty")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	assert.False(t, tty.IsTerminal(tmpFile), "Temporary file should not be a terminal")

	// 2. Test with a pipe (should not be a terminal)
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	defer r.Close()
	defer w.Close()

	assert.False(t, tty.IsTerminal(r), "Pipe reader should not be a terminal")
	assert.False(t, tty.IsTerminal(w), "Pipe writer should not be a terminal")

	// 3. Test with /dev/tty if possible (should be a terminal)
	f, err := os.Open("/dev/tty")
	if err == nil {
		defer f.Close()
		assert.True(t, tty.IsTerminal(f), "/dev/tty should be a terminal")
	} else {
		t.Log("Could not open /dev/tty, skipping positive test (this is expected in some non-interactive environments)")
	}
}
