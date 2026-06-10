package color_test

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
)

func TestGradient_DescendingChannels(t *testing.T) {
	// Channels that decrease from start to end previously wrapped around due
	// to unsigned arithmetic, producing wildly wrong intermediate colors.
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

func TestHSLToRGB_HueWrapping(t *testing.T) {
	// Hue values outside [0, 360) should wrap: 370 ≡ 10, 730 ≡ 10, -120 ≡ 240
	assert.Equal(t, color.HSLToRGB(10, 1.0, 0.5), color.HSLToRGB(370, 1.0, 0.5))
	assert.Equal(t, color.HSLToRGB(10, 1.0, 0.5), color.HSLToRGB(730, 1.0, 0.5))
	assert.Equal(t, color.HSLToRGB(240, 1.0, 0.5), color.HSLToRGB(-120, 1.0, 0.5))
}

func TestHSLToRGB_ClampsSaturationAndLightness(t *testing.T) {
	// Out-of-range saturation/lightness previously overflowed the uint8
	// channel conversion.
	assert.Equal(t, color.HSLToRGB(0, 1.0, 1.0), color.HSLToRGB(0, 2.0, 1.5))
	assert.Equal(t, color.HSLToRGB(0, 0, 0), color.HSLToRGB(0, -1.0, -0.5))
}
