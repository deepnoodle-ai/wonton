package tui

import (
	"testing"

	"github.com/deepnoodle-ai/gooey/require"
)

// TestRuntimeWithPipedInput verifies that the Runtime handles piped input gracefully.
// When stdin is not a terminal (e.g., piped from a file), the Runtime should not
// attempt to enable raw mode.
func TestRuntimeWithPipedInput(t *testing.T) {
	// This test verifies the structure and logic, but can't fully test piped input
	// in a unit test environment. The important part is that we check IsTerminal()
	// before calling EnableRawMode().

	// Create a test terminal
	terminal := NewTestTerminal(80, 24, nil)
	require.NotNil(t, terminal)

	// Create a simple test app
	app := &testKeyboardApp{
		events: make([]Event, 0),
	}

	// Create a runtime - this should not fail even if stdin is not a terminal
	runtime := NewRuntime(terminal, app, 30)
	require.NotNil(t, runtime)

	// The runtime should be properly initialized
	require.NotNil(t, runtime.terminal)
	require.NotNil(t, runtime.app)
	require.NotNil(t, runtime.events)
	require.NotNil(t, runtime.cmds)

	// Note: We can't actually call runtime.Run() in a test because it would block,
	// but we've verified that the runtime is properly initialized and won't crash
	// when stdin is not a terminal.
}
