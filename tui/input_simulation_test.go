package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// MockInputSource implements InputSource for testing
type MockInputSource struct {
	events chan Event
}

func NewMockInputSource() *MockInputSource {
	return &MockInputSource{
		events: make(chan Event, 100),
	}
}

func (m *MockInputSource) ReadEvent() (Event, error) {
	event, ok := <-m.events
	if !ok {
		return nil, errors.New("EOF")
	}
	return event, nil
}

func (m *MockInputSource) SetPasteTabWidth(w int) {}

func (m *MockInputSource) Send(event Event) {
	m.events <- event
}

func (m *MockInputSource) Close() {
	close(m.events)
}

// TestRuntime_BackslashEnterFallback tests the backslash+enter -> shift+enter conversion
func TestRuntime_BackslashEnterFallback(t *testing.T) {
	// Create a test app that records keys
	app := &testKeyboardApp{
		events: make([]Event, 0),
	}

	// Setup runtime with mock input
	// We pass a nil terminal because we aren't testing rendering here, just input loop logic.
	// However, Runtime requires a terminal. We can use NewTestTerminal.
	terminal := NewTestTerminal(80, 24, nil)
	runtime := NewRuntime(terminal, app, 30)
	
	inputSource := NewMockInputSource()
	runtime.SetInputSource(inputSource)

	// Start runtime in background
	go func() {
		runtime.Run()
	}()
	defer runtime.Stop()

	// 1. Send Backslash
	inputSource.Send(KeyEvent{
		Rune: 0x5C, // Backslash
		Key:  KeyUnknown,
		Time: time.Now(),
	})

	// 2. Send Enter immediately after
	inputSource.Send(KeyEvent{
		Key:  KeyEnter,
		Time: time.Now(),
	})

	// 3. Send Quit to stop processing
	inputSource.Send(QuitEvent{Time: time.Now()})

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify events
	// We expect ONE KeyEvent: Enter with Shift=true
	// We might also see ResizeEvents etc., so filter for KeyEvents
	var keyEvents []KeyEvent
	for _, e := range app.events {
		if ke, ok := e.(KeyEvent); ok {
			keyEvents = append(keyEvents, ke)
		}
	}

	assert.True(t, len(keyEvents) > 0, "Should have received key events")
	if len(keyEvents) > 0 {
		lastEvent := keyEvents[len(keyEvents)-1]
		assert.Equal(t, KeyEnter, lastEvent.Key)
		assert.True(t, lastEvent.Shift, "Should have Shift modifier set")
	}
}

// TestRuntime_RegularBackslash tests that backslash alone is handled correctly
func TestRuntime_RegularBackslash(t *testing.T) {
	app := &testKeyboardApp{
		events: make([]Event, 0),
	}

	terminal := NewTestTerminal(80, 24, nil)
	runtime := NewRuntime(terminal, app, 30)
	
	inputSource := NewMockInputSource()
	runtime.SetInputSource(inputSource)

	go func() {
		runtime.Run()
	}()
	defer runtime.Stop()

	// Send Backslash
	inputSource.Send(KeyEvent{
		Rune: '\\',
		Key:  KeyUnknown,
		Time: time.Now(),
	})

	// Wait for timeout (100ms in runtime)
	time.Sleep(200 * time.Millisecond)

	// Send 'a' to prove we moved on
	inputSource.Send(KeyEvent{
		Rune: 'a',
		Time: time.Now(),
	})
	
	inputSource.Send(QuitEvent{Time: time.Now()})
	time.Sleep(50 * time.Millisecond)

	var keyEvents []KeyEvent
	for _, e := range app.events {
		if ke, ok := e.(KeyEvent); ok {
			keyEvents = append(keyEvents, ke)
		}
	}

	// Should have received '\' then 'a'
	foundBackslash := false
	for _, e := range keyEvents {
		if e.Rune == 0x5C { // Backslash
			foundBackslash = true
			break
		}
	}
	assert.True(t, foundBackslash, "Should have received backslash event")
}
