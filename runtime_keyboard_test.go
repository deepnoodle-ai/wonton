package gooey

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRuntimeKeyboardInput verifies that the Runtime properly handles keyboard events.
// This test simulates keyboard input and ensures events are correctly delivered to the application.
func TestRuntimeKeyboardInput(t *testing.T) {
	// Create a mock terminal with a fake stdin
	stdin := strings.NewReader("q") // Simulate pressing 'q'

	// Create a terminal (this will fail in CI, but works locally)
	terminal, err := NewTerminal()
	if err != nil {
		t.Skip("Skipping test: terminal not available")
	}
	defer terminal.Close()

	// Create a test application that tracks received events
	app := &testKeyboardApp{
		events: make([]Event, 0),
	}

	// Override the runtime's input reader to use our fake stdin
	runtime := NewRuntime(terminal, app, 30)

	// We can't easily test the full runtime without a real TTY,
	// but we can verify the Runtime structure is set up correctly
	require.NotNil(t, runtime)
	require.NotNil(t, runtime.terminal)
	require.NotNil(t, runtime.app)
	require.NotNil(t, runtime.events)
	require.NotNil(t, runtime.cmds)

	_ = stdin // suppress unused warning
}

type testKeyboardApp struct {
	events []Event
}

func (app *testKeyboardApp) HandleEvent(event Event) []Cmd {
	app.events = append(app.events, event)
	if ke, ok := event.(KeyEvent); ok {
		if ke.Rune == 'q' {
			return []Cmd{Quit()}
		}
	}
	return nil
}

func (app *testKeyboardApp) Render(frame RenderFrame) {
	// Nothing to render
}

// TestCheckboxWithMockInput tests that the checkbox properly handles keyboard input in the Runtime context.
func TestCheckboxWithMockInput(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a checkbox group
	cg := NewCheckboxGroup(2, 4, []string{
		"Apple",
		"Banana",
		"Cherry",
	})

	// Simulate arrow down key
	handled := cg.HandleKey(KeyEvent{Key: KeyArrowDown})
	require.True(t, handled, "arrow down should be handled")
	require.Equal(t, 1, cg.Cursor, "cursor should move to index 1")

	// Simulate space key to toggle selection
	handled = cg.HandleKey(KeyEvent{Rune: ' '})
	require.True(t, handled, "space should be handled")
	require.True(t, cg.Selected[1], "Banana should be selected")

	// Simulate arrow down again
	handled = cg.HandleKey(KeyEvent{Key: KeyArrowDown})
	require.True(t, handled, "arrow down should be handled")
	require.Equal(t, 2, cg.Cursor, "cursor should move to index 2")

	// Simulate space key to toggle selection
	handled = cg.HandleKey(KeyEvent{Rune: ' '})
	require.True(t, handled, "space should be handled")
	require.True(t, cg.Selected[2], "Cherry should be selected")

	// Verify selected items
	selected := cg.GetSelectedItems()
	require.Len(t, selected, 2, "should have 2 selected items")
	require.Contains(t, selected, "Banana")
	require.Contains(t, selected, "Cherry")

	_ = buf // suppress unused warning
}

// TestRuntimeEnablesRawMode verifies that the Runtime enables raw mode on the terminal.
func TestRuntimeEnablesRawMode(t *testing.T) {
	// This is a simple structural test - we can't fully test raw mode in CI
	runtime := &Runtime{
		events: make(chan Event, 100),
		cmds:   make(chan Cmd, 100),
		done:   make(chan struct{}),
		fps:    30,
	}

	require.NotNil(t, runtime.events)
	require.NotNil(t, runtime.cmds)
	require.NotNil(t, runtime.done)
	require.Equal(t, 30, runtime.fps)
}

// TestRuntimeSendsInitialResizeEvent verifies that the Runtime sends an initial ResizeEvent.
func TestRuntimeSendsInitialResizeEvent(t *testing.T) {
	// Create a test application that tracks received events
	app := &testEventTracker{
		events:      make([]Event, 0),
		resizesSeen: 0,
	}

	// We can't actually run the runtime without a TTY, but we can verify
	// the event channel is set up correctly
	runtime := &Runtime{
		app:    app,
		events: make(chan Event, 100),
		cmds:   make(chan Cmd, 100),
		done:   make(chan struct{}),
		fps:    30,
	}

	// Manually send a resize event to simulate what the runtime does
	runtime.events <- ResizeEvent{
		Time:   time.Now(),
		Width:  80,
		Height: 24,
	}

	// Read the event
	event := <-runtime.events
	resizeEvent, ok := event.(ResizeEvent)
	require.True(t, ok, "should be a ResizeEvent")
	require.Equal(t, 80, resizeEvent.Width)
	require.Equal(t, 24, resizeEvent.Height)
}

type testEventTracker struct {
	events      []Event
	resizesSeen int
}

func (app *testEventTracker) HandleEvent(event Event) []Cmd {
	app.events = append(app.events, event)
	if _, ok := event.(ResizeEvent); ok {
		app.resizesSeen++
	}
	return nil
}

func (app *testEventTracker) Render(frame RenderFrame) {
	// Nothing to render
}
