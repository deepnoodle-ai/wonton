package color_test

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
)

func TestGradient_NonPositiveSteps(t *testing.T) {
	start := color.NewRGB(255, 0, 0)
	end := color.NewRGB(0, 255, 0)
	assert.Len(t, color.Gradient(start, end, 0), 0)
	assert.Len(t, color.Gradient(start, end, -3), 0)
}

func TestGradient_SingleStep(t *testing.T) {
	start := color.NewRGB(255, 0, 0)
	end := color.NewRGB(0, 255, 0)
	colors := color.Gradient(start, end, 1)
	assert.Len(t, colors, 1)
	assert.Equal(t, start, colors[0])
}

func TestGradient_TwoSteps(t *testing.T) {
	start := color.NewRGB(0, 0, 0)
	end := color.NewRGB(255, 255, 255)
	colors := color.Gradient(start, end, 2)
	assert.Len(t, colors, 2)
	assert.Equal(t, start, colors[0])
	assert.Equal(t, end, colors[1])
}

func TestGradient_MultipleSteps(t *testing.T) {
	start := color.NewRGB(255, 0, 0)
	end := color.NewRGB(0, 0, 255)
	colors := color.Gradient(start, end, 5)
	assert.Len(t, colors, 5)
	assert.Equal(t, start, colors[0])
	assert.Equal(t, end, colors[4])
	// Middle color should be a blend
	assert.NotEqual(t, start, colors[2])
	assert.NotEqual(t, end, colors[2])
}

func TestGradient_DescendingChannels(t *testing.T) {
	// Channels that decrease from start to end must not wrap around due to
	// unsigned arithmetic.
	start := color.NewRGB(255, 0, 0)
	end := color.NewRGB(0, 0, 255)
	colors := color.Gradient(start, end, 3)
	assert.Len(t, colors, 3)
	assert.Equal(t, start, colors[0])
	assert.Equal(t, color.NewRGB(128, 0, 128), colors[1])
	assert.Equal(t, end, colors[2])
}

func TestGradient_MidpointAccuracy(t *testing.T) {
	colors := color.Gradient(color.NewRGB(200, 100, 50), color.NewRGB(100, 200, 250), 3)
	assert.Equal(t, color.NewRGB(150, 150, 150), colors[1])
}

func TestMultiGradient_EmptyStops(t *testing.T) {
	assert.Len(t, color.MultiGradient([]color.RGB{}, 5), 0)
}

func TestMultiGradient_NonPositiveSteps(t *testing.T) {
	stops := []color.RGB{color.NewRGB(255, 0, 0), color.NewRGB(0, 0, 255)}
	assert.Len(t, color.MultiGradient(stops, 0), 0)
	assert.Len(t, color.MultiGradient(stops, -3), 0)
}

func TestMultiGradient_OneStep(t *testing.T) {
	stops := []color.RGB{color.NewRGB(255, 0, 0), color.NewRGB(0, 0, 255)}
	colors := color.MultiGradient(stops, 1)
	assert.Len(t, colors, 1)
	assert.Equal(t, stops[0], colors[0])
}

func TestMultiGradient_SingleStop(t *testing.T) {
	stop := color.NewRGB(128, 128, 128)
	colors := color.MultiGradient([]color.RGB{stop}, 5)
	assert.Len(t, colors, 5)
	for _, c := range colors {
		assert.Equal(t, stop, c)
	}
}

func TestMultiGradient_MultipleStops(t *testing.T) {
	stops := []color.RGB{
		color.NewRGB(255, 0, 0), // Red
		color.NewRGB(0, 255, 0), // Green
		color.NewRGB(0, 0, 255), // Blue
	}
	colors := color.MultiGradient(stops, 5)
	assert.Len(t, colors, 5)
	assert.Equal(t, stops[0], colors[0])
	assert.Equal(t, stops[1], colors[2])
	assert.Equal(t, stops[2], colors[4])
}

func TestRainbowGradient(t *testing.T) {
	assert.Len(t, color.RainbowGradient(0), 0)

	single := color.RainbowGradient(1)
	assert.Len(t, single, 1)
	assert.Equal(t, color.NewRGB(255, 0, 0), single[0])

	colors := color.RainbowGradient(10)
	assert.Len(t, colors, 10)
	// First should be red, last should be violet
	assert.Equal(t, color.NewRGB(255, 0, 0), colors[0])
	assert.Equal(t, color.NewRGB(148, 0, 211), colors[9])
	// Should have variation in colors
	assert.NotEqual(t, colors[0], colors[5])
}

func TestSmoothRainbow(t *testing.T) {
	assert.Len(t, color.SmoothRainbow(0), 0)

	single := color.SmoothRainbow(1)
	assert.Len(t, single, 1)
	assert.Equal(t, color.NewRGB(255, 0, 0), single[0])

	colors := color.SmoothRainbow(10)
	assert.Len(t, colors, 10)
	uniqueColors := make(map[color.RGB]bool)
	for _, c := range colors {
		uniqueColors[c] = true
	}
	assert.Greater(t, len(uniqueColors), 5, "Should have variety of colors")
}

func TestApplyGradient(t *testing.T) {
	forceColor(t)
	out := color.ApplyGradient("abc", color.NewRGB(255, 0, 0), color.NewRGB(0, 0, 255))
	expected := "\033[38;2;255;0;0ma" +
		"\033[38;2;128;0;128mb" +
		"\033[38;2;0;0;255mc" +
		"\033[0m"
	assert.Equal(t, expected, out)
}

func TestApplyGradient_SingleStop(t *testing.T) {
	forceColor(t)
	out := color.ApplyGradient("ab", color.NewRGB(255, 0, 0))
	assert.Equal(t, "\033[38;2;255;0;0ma\033[38;2;255;0;0mb\033[0m", out)
}

func TestApplyGradient_GraphemeClusters(t *testing.T) {
	forceColor(t)
	// The family emoji is one grapheme cluster built from multiple runes
	// joined by ZWJ; it must be colored as a single unit, not split.
	text := "a👨‍👩‍👧b"
	out := color.ApplyGradient(text, color.NewRGB(255, 0, 0), color.NewRGB(0, 0, 255))
	assert.Contains(t, out, "👨‍👩‍👧")
	// 3 clusters -> 3 color sequences
	assert.Equal(t, 3, strings.Count(out, "\033[38;2;"))
}

func TestApplyGradient_Edges(t *testing.T) {
	forceColor(t)
	// No stops or empty text: unchanged
	assert.Equal(t, "abc", color.ApplyGradient("abc"))
	assert.Equal(t, "", color.ApplyGradient("", color.NewRGB(255, 0, 0)))
}

func TestApplyGradient_RespectsEnabled(t *testing.T) {
	disableColor(t)
	assert.Equal(t, "abc", color.ApplyGradient("abc", color.NewRGB(255, 0, 0), color.NewRGB(0, 0, 255)))
}
