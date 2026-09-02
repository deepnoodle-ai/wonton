package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestMeasureReportsTheSizeAViewWouldRenderAt(t *testing.T) {
	w, h := Measure(Text("hello"), 0, 0)
	assert.Equal(t, 5, w)
	assert.Equal(t, 1, h)

	// Wrapping is a function of the width you ask about.
	wrapped := Text("one two three four five six").Wrap()
	_, h10 := Measure(wrapped, 10, 0)
	_, h30 := Measure(wrapped, 30, 0)
	assert.True(t, h10 > h30, "narrower means taller: %d vs %d", h10, h30)
	assert.Equal(t, 1, h30)
}

func TestMeasureMatchesWhatIsActuallyRendered(t *testing.T) {
	view := Stack(
		Text("a longish line that will need to wrap somewhere").Wrap(),
		Text("second"),
	).Gap(1)

	_, height := Measure(view, 20, 0)

	screen := SprintScreen(view, WithWidth(20))
	rows := 0
	for y := 0; y < height+5; y++ {
		if screen.Row(y) != "" {
			rows = y + 1
		}
	}
	assert.Equal(t, height, rows)
}

func TestMeasureOfNilIsZero(t *testing.T) {
	w, h := Measure(nil, 40, 0)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}
