package color_test

import (
	"math"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
)

func TestHSLToRGB_PrimaryColors(t *testing.T) {
	assert.Equal(t, color.NewRGB(255, 0, 0), color.HSLToRGB(0, 1.0, 0.5))
	assert.Equal(t, color.NewRGB(0, 255, 0), color.HSLToRGB(120, 1.0, 0.5))
	assert.Equal(t, color.NewRGB(0, 0, 255), color.HSLToRGB(240, 1.0, 0.5))
}

func TestHSLToRGB_Grayscale(t *testing.T) {
	// Saturation 0 should produce grayscale
	rgb := color.HSLToRGB(0, 0, 0.5)
	assert.Equal(t, rgb.R, rgb.G)
	assert.Equal(t, rgb.G, rgb.B)
}

func TestHSLToRGB_HueWrapping(t *testing.T) {
	// Hue values outside [0, 360) should wrap: 370 ≡ 10, 730 ≡ 10, -120 ≡ 240
	assert.Equal(t, color.HSLToRGB(10, 1.0, 0.5), color.HSLToRGB(370, 1.0, 0.5))
	assert.Equal(t, color.HSLToRGB(10, 1.0, 0.5), color.HSLToRGB(730, 1.0, 0.5))
	assert.Equal(t, color.HSLToRGB(240, 1.0, 0.5), color.HSLToRGB(-120, 1.0, 0.5))
}

func TestHSLToRGB_ClampsSaturationAndLightness(t *testing.T) {
	// Out-of-range saturation/lightness must not overflow the uint8
	// channel conversion.
	assert.Equal(t, color.HSLToRGB(0, 1.0, 1.0), color.HSLToRGB(0, 2.0, 1.5))
	assert.Equal(t, color.HSLToRGB(0, 0, 0), color.HSLToRGB(0, -1.0, -0.5))
}

func TestRGB_HSL_PrimaryColors(t *testing.T) {
	tests := []struct {
		rgb     color.RGB
		h, s, l float64
	}{
		{color.NewRGB(255, 0, 0), 0, 1, 0.5},
		{color.NewRGB(0, 255, 0), 120, 1, 0.5},
		{color.NewRGB(0, 0, 255), 240, 1, 0.5},
		{color.NewRGB(0, 0, 0), 0, 0, 0},
		{color.NewRGB(255, 255, 255), 0, 0, 1},
		{color.NewRGB(128, 128, 128), 0, 0, 128.0 / 255},
	}
	for _, tt := range tests {
		t.Run(tt.rgb.Hex(), func(t *testing.T) {
			h, s, l := tt.rgb.HSL()
			assert.True(t, math.Abs(h-tt.h) < 1e-9, "hue: got %v want %v", h, tt.h)
			assert.True(t, math.Abs(s-tt.s) < 1e-9, "saturation: got %v want %v", s, tt.s)
			assert.True(t, math.Abs(l-tt.l) < 1e-9, "lightness: got %v want %v", l, tt.l)
		})
	}
}

func TestHSL_RoundTrip(t *testing.T) {
	// RGB -> HSL -> RGB should be lossless (up to rounding) for arbitrary colors.
	colors := []color.RGB{
		color.NewRGB(255, 128, 0),
		color.NewRGB(12, 34, 56),
		color.NewRGB(200, 100, 50),
		color.NewRGB(1, 254, 127),
		color.NewRGB(95, 135, 255),
	}
	for _, original := range colors {
		t.Run(original.Hex(), func(t *testing.T) {
			h, s, l := original.HSL()
			back := color.HSLToRGB(h, s, l)
			assert.True(t, channelDelta(original, back) <= 1,
				"round trip %v -> (%v, %v, %v) -> %v", original, h, s, l, back)
		})
	}
}

// channelDelta returns the largest per-channel difference between two colors.
func channelDelta(a, b color.RGB) int {
	abs := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}
	d := abs(int(a.R) - int(b.R))
	if g := abs(int(a.G) - int(b.G)); g > d {
		d = g
	}
	if bd := abs(int(a.B) - int(b.B)); bd > d {
		d = bd
	}
	return d
}
