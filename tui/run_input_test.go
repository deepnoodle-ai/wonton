package tui

import (
	"bytes"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// WithInput must reach the Runtime that Run builds.
func TestRun_WithInputFeedsRuntime(t *testing.T) {
	source := NewMockInputSource()
	cfg := defaultRunConfig()
	WithInput(source)(&cfg)

	var buf bytes.Buffer
	app := &testRuntimeModel{}
	runtime := newRunRuntime(NewTestTerminal(80, 24, &buf), app, cfg)
	go func() {
		time.Sleep(50 * time.Millisecond)
		source.Send(KeyEvent{Rune: 'q'})
	}()
	assert.NoError(t, runWithTimeout(t, runtime, 2*time.Second))
}
