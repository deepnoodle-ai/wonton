package terminal

import (
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestKeyEventTimestampUsesProvidedTime(t *testing.T) {
	expected := time.Date(2024, 1, 2, 3, 4, 5, 6, time.UTC)
	event := KeyEvent{Time: expected}

	assert.Equal(t, event.Timestamp(), expected)
}

func TestKeyEventTimestampDefaultsToNow(t *testing.T) {
	start := time.Now()
	event := KeyEvent{}
	ts := event.Timestamp()
	end := time.Now()

	assert.False(t, ts.Before(start))
	assert.False(t, ts.After(end))
}
