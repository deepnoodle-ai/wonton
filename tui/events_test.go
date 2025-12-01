package tui

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/require"
)

var errTestError = errors.New("test error")

// TestEventTypes tests all event types can be created and used
func TestEventTypes(t *testing.T) {

	t.Run("ResizeEvent", func(t *testing.T) {
		event := ResizeEvent{
			Time:   time.Now(),
			Width:  80,
			Height: 24,
		}
		require.Equal(t, 80, event.Width)
		require.Equal(t, 24, event.Height)
		require.False(t, event.Time.IsZero())
	})

	t.Run("TickEvent", func(t *testing.T) {
		event := TickEvent{
			Time: time.Now(),
		}
		require.False(t, event.Time.IsZero())
	})

	t.Run("QuitEvent", func(t *testing.T) {
		event := QuitEvent{
			Time: time.Now(),
		}
		require.False(t, event.Time.IsZero())
	})

	t.Run("ErrorEvent", func(t *testing.T) {
		event := ErrorEvent{
			Time: time.Now(),
			Err:  errTestError,
		}
		require.Error(t, event.Err)
		require.Equal(t, errTestError, event.Err)
		require.False(t, event.Time.IsZero())
	})
}

// TestEventBuffering tests that events are properly buffered
func TestEventBuffering(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	const numEvents = 100
	receivedCount := 0

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd {
			if _, ok := event.(bufferEvent); ok {
				receivedCount++
				if receivedCount >= numEvents {
					return []Cmd{Quit()}
				}
			}
			return nil
		},
		renderFunc: func(frame RenderFrame) {},
	}

	runtime := NewRuntime(terminal, app, 30)

	// Send many events rapidly
	go func() {
		time.Sleep(50 * time.Millisecond)
		for i := 0; i < numEvents; i++ {
			runtime.SendEvent(bufferEvent{id: i})
		}
	}()

	err := runtime.Run()
	require.NoError(t, err)

	// Verify all events were received
	require.Equal(t, numEvents, receivedCount)
}

type bufferEvent struct {
	id int
}

func (e bufferEvent) Timestamp() time.Time {
	return time.Now()
}
