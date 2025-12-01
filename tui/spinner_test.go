package tui

import (
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestSpinnerStyle_Dots(t *testing.T) {
	assert.NotNil(t, SpinnerDots.Frames)
	assert.Greater(t, len(SpinnerDots.Frames), 0)
	assert.Greater(t, SpinnerDots.Interval, time.Duration(0))
}

func TestSpinnerStyle_Line(t *testing.T) {
	assert.Equal(t, []string{"-", "\\", "|", "/"}, SpinnerLine.Frames)
	assert.Equal(t, 100*time.Millisecond, SpinnerLine.Interval)
}

func TestSpinnerStyle_Arrows(t *testing.T) {
	assert.NotNil(t, SpinnerArrows.Frames)
	assert.Equal(t, 8, len(SpinnerArrows.Frames))
}

func TestSpinnerStyle_AllStylesHaveFrames(t *testing.T) {
	styles := []SpinnerStyle{
		SpinnerDots,
		SpinnerLine,
		SpinnerArrows,
		SpinnerCircle,
		SpinnerSquare,
		SpinnerBounce,
		SpinnerBar,
		SpinnerStars,
		SpinnerStarField,
		SpinnerAsterisk,
		SpinnerSparkle,
	}

	for _, style := range styles {
		assert.Greater(t, len(style.Frames), 0, "Spinner style should have frames")
		assert.Greater(t, style.Interval, time.Duration(0), "Spinner style should have positive interval")
	}
}
