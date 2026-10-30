//go:build darwin || linux

package termsession

import (
	"strings"
	"sync"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// TestSession_ConcurrentStart verifies that concurrent Start calls do not
// double-start the command or panic on a double close of the done channel.
func TestSession_ConcurrentStart(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"true"},
		Input:   strings.NewReader(""),
		Output:  &strings.Builder{},
	})
	assert.NoError(t, err)
	defer s.Close()

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = s.Start()
		}(i)
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
		}
	}
	assert.Equal(t, 1, succeeded)

	assert.NoError(t, s.Wait())
}

// TestSession_WaitClosesPTY verifies that Wait() cleans up the PTY file
// descriptor even before Close() is called.
func TestSession_WaitClosesPTY(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"true"},
		Input:   strings.NewReader(""),
		Output:  &strings.Builder{},
	})
	assert.NoError(t, err)

	assert.NoError(t, s.Start())
	assert.NoError(t, s.Wait())

	s.mu.Lock()
	ptyAfterWait := s.pty
	s.mu.Unlock()
	assert.Nil(t, ptyAfterWait)

	assert.NoError(t, s.Close())
}

// TestSession_CloseWaitsForCopyOutput verifies that Close() waits for
// copyOutput to finish draining before returning.
func TestSession_CloseWaitsForCopyOutput(t *testing.T) {
	s, err := NewSession(SessionOptions{
		Command: []string{"sleep", "5"},
		Input:   strings.NewReader(""),
		Output:  &strings.Builder{},
	})
	assert.NoError(t, err)

	assert.NoError(t, s.Start())
	assert.NoError(t, s.Close())

	select {
	case <-s.done:
	default:
		t.Fatal("Close returned before copyOutput finished")
	}
}
