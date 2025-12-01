package color

import (
	"testing"

	"github.com/deepnoodle-ai/gooey/assert"
	"github.com/deepnoodle-ai/gooey/require"
)

func TestColor_ForegroundCode_AllColors(t *testing.T) {
	tests := []struct {
		color    Color
		expected string
	}{
		{Black, "30"},
		{Red, "31"},
		{Green, "32"},
		{Yellow, "33"},
		{Blue, "34"},
		{Magenta, "35"},
		{Cyan, "36"},
		{White, "37"},
		{BrightBlack, "90"},
		{BrightRed, "91"},
		{BrightGreen, "92"},
		{BrightYellow, "93"},
		{BrightBlue, "94"},
		{BrightMagenta, "95"},
		{BrightCyan, "96"},
		{BrightWhite, "97"},
		{Default, "39"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.color.ForegroundCode())
		})
	}
}

func TestColor_BackgroundCode_AllColors(t *testing.T) {
	tests := []struct {
		color    Color
		expected string
	}{
		{Black, "40"},
		{Red, "41"},
		{Green, "42"},
		{Yellow, "43"},
		{Blue, "44"},
		{Magenta, "45"},
		{Cyan, "46"},
		{White, "47"},
		{BrightBlack, "100"},
		{BrightRed, "101"},
		{BrightGreen, "102"},
		{BrightYellow, "103"},
		{BrightBlue, "104"},
		{BrightMagenta, "105"},
		{BrightCyan, "106"},
		{BrightWhite, "107"},
		{Default, "49"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.color.BackgroundCode())
		})
	}
}

func TestRGB_Foreground(t *testing.T) {
	rgb := NewRGB(255, 0, 127)
	output := rgb.Foreground()
	assert.Equal(t, "\033[38;2;255;0;127m", output)
}

func TestRGB_Background(t *testing.T) {
	rgb := NewRGB(127, 0, 255)
	output := rgb.Background()
	assert.Equal(t, "\033[48;2;127;0;255m", output)
}

func TestRGB_Apply_Foreground(t *testing.T) {
	rgb := NewRGB(255, 128, 0)
	text := rgb.Apply("Test", false)
	assert.Contains(t, text, "Test")
	assert.Contains(t, text, "38;2;255;128;0")
	assert.Contains(t, text, "\033[0m")
}

func TestRGB_Apply_Background(t *testing.T) {
	rgb := NewRGB(0, 128, 255)
	text := rgb.Apply("Test", true)
	assert.Contains(t, text, "Test")
	assert.Contains(t, text, "48;2;0;128;255")
	assert.Contains(t, text, "\033[0m")
}

func TestGradient_SingleStep(t *testing.T) {
	start := NewRGB(255, 0, 0)
	end := NewRGB(0, 255, 0)
	colors := Gradient(start, end, 1)
	require.Len(t, colors, 1)
	assert.Equal(t, start, colors[0])
}

func TestGradient_TwoSteps(t *testing.T) {
	start := NewRGB(0, 0, 0)
	end := NewRGB(255, 255, 255)
	colors := Gradient(start, end, 2)
	require.Len(t, colors, 2)
	assert.Equal(t, start, colors[0])
	assert.Equal(t, end, colors[1])
}

func TestGradient_MultipleSteps(t *testing.T) {
	start := NewRGB(255, 0, 0)
	end := NewRGB(0, 0, 255)
	colors := Gradient(start, end, 5)
	require.Len(t, colors, 5)
	assert.Equal(t, start, colors[0])
	assert.Equal(t, end, colors[4])
	// Middle color should be a blend
	assert.NotEqual(t, start, colors[2])
	assert.NotEqual(t, end, colors[2])
}

func TestRainbowGradient_SingleStep(t *testing.T) {
	colors := RainbowGradient(1)
	require.Len(t, colors, 1)
	assert.Equal(t, NewRGB(255, 0, 0), colors[0])
}

func TestRainbowGradient_MultipleSteps(t *testing.T) {
	colors := RainbowGradient(10)
	require.Len(t, colors, 10)
	// First should be red
	assert.Equal(t, NewRGB(255, 0, 0), colors[0])
	// Should have variation in colors
	assert.NotEqual(t, colors[0], colors[5])
}

func TestSmoothRainbow(t *testing.T) {
	colors := SmoothRainbow(10)
	require.Len(t, colors, 10)
	// Should have distinct colors
	uniqueColors := make(map[RGB]bool)
	for _, c := range colors {
		uniqueColors[c] = true
	}
	assert.Greater(t, len(uniqueColors), 5, "Should have variety of colors")
}

func TestMultiGradient_EmptyStops(t *testing.T) {
	colors := MultiGradient([]RGB{}, 5)
	assert.Len(t, colors, 0)
}

func TestMultiGradient_SingleStop(t *testing.T) {
	stop := NewRGB(128, 128, 128)
	colors := MultiGradient([]RGB{stop}, 5)
	require.Len(t, colors, 5)
	for _, c := range colors {
		assert.Equal(t, stop, c)
	}
}

func TestMultiGradient_MultipleStops(t *testing.T) {
	stops := []RGB{
		NewRGB(255, 0, 0), // Red
		NewRGB(0, 255, 0), // Green
		NewRGB(0, 0, 255), // Blue
	}
	colors := MultiGradient(stops, 5)
	require.Len(t, colors, 5)
	assert.Equal(t, stops[0], colors[0])
	assert.Equal(t, stops[2], colors[4])
}

func TestHSLToRGB_Red(t *testing.T) {
	rgb := HSLToRGB(0, 1.0, 0.5)
	assert.Equal(t, uint8(255), rgb.R)
	assert.Equal(t, uint8(0), rgb.G)
	assert.Equal(t, uint8(0), rgb.B)
}

func TestHSLToRGB_Green(t *testing.T) {
	rgb := HSLToRGB(120, 1.0, 0.5)
	assert.Equal(t, uint8(0), rgb.R)
	assert.Equal(t, uint8(255), rgb.G)
	assert.Equal(t, uint8(0), rgb.B)
}

func TestHSLToRGB_Blue(t *testing.T) {
	rgb := HSLToRGB(240, 1.0, 0.5)
	assert.Equal(t, uint8(0), rgb.R)
	assert.Equal(t, uint8(0), rgb.G)
	assert.Equal(t, uint8(255), rgb.B)
}

func TestHSLToRGB_Grayscale(t *testing.T) {
	// Saturation 0 should produce grayscale
	rgb := HSLToRGB(0, 0, 0.5)
	assert.Equal(t, rgb.R, rgb.G)
	assert.Equal(t, rgb.G, rgb.B)
}
