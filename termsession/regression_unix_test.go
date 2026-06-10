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
