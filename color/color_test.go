package color_test

import (
	"testing"

	"github.com/deepnoodle-ai/gooey/assert"
	"github.com/deepnoodle-ai/gooey/color"
	"github.com/deepnoodle-ai/gooey/require"
)

func TestColor_ForegroundCode_AllColors(t *testing.T) {
	tests := []struct {
		c        color.Color
		expected string
	}{
		{color.Black, "30"},
		{color.Red, "31"},
		{color.Green, "32"},
		{color.Yellow, "33"},
		{color.Blue, "34"},
		{color.Magenta, "35"},
		{color.Cyan, "36"},
		{color.White, "37"},
		{color.BrightBlack, "90"},
		{color.BrightRed, "91"},
		{color.BrightGreen, "92"},
		{color.BrightYellow, "93"},
		{color.BrightBlue, "94"},
		{color.BrightMagenta, "95"},
		{color.BrightCyan, "96"},
		{color.BrightWhite, "97"},
		{color.Default, "39"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.c.ForegroundCode())
		})
	}
}

func TestColor_BackgroundCode_AllColors(t *testing.T) {
	tests := []struct {
		c        color.Color
		expected string
	}{
		{color.Black, "40"},
		{color.Red, "41"},
		{color.Green, "42"},
		{color.Yellow, "43"},
		{color.Blue, "44"},
		{color.Magenta, "45"},
		{color.Cyan, "46"},
		{color.White, "47"},
		{color.BrightBlack, "100"},
		{color.BrightRed, "101"},
		{color.BrightGreen, "102"},
		{color.BrightYellow, "103"},
		{color.BrightBlue, "104"},
		{color.BrightMagenta, "105"},
		{color.BrightCyan, "106"},
		{color.BrightWhite, "107"},
		{color.Default, "49"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.c.BackgroundCode())
		})
	}
}

func TestRGB_Foreground(t *testing.T) {
	rgb := color.NewRGB(255, 0, 127)
	output := rgb.Foreground()
	assert.Equal(t, "\033[38;2;255;0;127m", output)
}

func TestRGB_Background(t *testing.T) {
	rgb := color.NewRGB(127, 0, 255)
	output := rgb.Background()
	assert.Equal(t, "\033[48;2;127;0;255m", output)
}

func TestRGB_Apply_Foreground(t *testing.T) {
	rgb := color.NewRGB(255, 128, 0)
	text := rgb.Apply("Test", false)
	assert.Contains(t, text, "Test")
	assert.Contains(t, text, "38;2;255;128;0")
	assert.Contains(t, text, "\033[0m")
}

func TestRGB_Apply_Background(t *testing.T) {
	rgb := color.NewRGB(0, 128, 255)
	text := rgb.Apply("Test", true)
	assert.Contains(t, text, "Test")
	assert.Contains(t, text, "48;2;0;128;255")
	assert.Contains(t, text, "\033[0m")
}

func TestGradient_SingleStep(t *testing.T) {
	start := color.NewRGB(255, 0, 0)
	end := color.NewRGB(0, 255, 0)
	colors := color.Gradient(start, end, 1)
	require.Len(t, colors, 1)
	assert.Equal(t, start, colors[0])
}

func TestGradient_TwoSteps(t *testing.T) {
	start := color.NewRGB(0, 0, 0)
	end := color.NewRGB(255, 255, 255)
	colors := color.Gradient(start, end, 2)
	require.Len(t, colors, 2)
	assert.Equal(t, start, colors[0])
	assert.Equal(t, end, colors[1])
}

func TestGradient_MultipleSteps(t *testing.T) {
	start := color.NewRGB(255, 0, 0)
	end := color.NewRGB(0, 0, 255)
	colors := color.Gradient(start, end, 5)
	require.Len(t, colors, 5)
	assert.Equal(t, start, colors[0])
	assert.Equal(t, end, colors[4])
	// Middle color should be a blend
	assert.NotEqual(t, start, colors[2])
	assert.NotEqual(t, end, colors[2])
}

func TestRainbowGradient_SingleStep(t *testing.T) {
	colors := color.RainbowGradient(1)
	require.Len(t, colors, 1)
	assert.Equal(t, color.NewRGB(255, 0, 0), colors[0])
}

func TestRainbowGradient_MultipleSteps(t *testing.T) {
	colors := color.RainbowGradient(10)
	require.Len(t, colors, 10)
	// First should be red
	assert.Equal(t, color.NewRGB(255, 0, 0), colors[0])
	// Should have variation in colors
	assert.NotEqual(t, colors[0], colors[5])
}

func TestSmoothRainbow(t *testing.T) {
	colors := color.SmoothRainbow(10)
	require.Len(t, colors, 10)
	// Should have distinct colors
	uniqueColors := make(map[color.RGB]bool)
	for _, c := range colors {
		uniqueColors[c] = true
	}
	assert.Greater(t, len(uniqueColors), 5, "Should have variety of colors")
}

func TestMultiGradient_EmptyStops(t *testing.T) {
	colors := color.MultiGradient([]color.RGB{}, 5)
	assert.Len(t, colors, 0)
}

func TestMultiGradient_SingleStop(t *testing.T) {
	stop := color.NewRGB(128, 128, 128)
	colors := color.MultiGradient([]color.RGB{stop}, 5)
	require.Len(t, colors, 5)
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
	require.Len(t, colors, 5)
	assert.Equal(t, stops[0], colors[0])
	assert.Equal(t, stops[2], colors[4])
}

func TestHSLToRGB_Red(t *testing.T) {
	rgb := color.HSLToRGB(0, 1.0, 0.5)
	assert.Equal(t, uint8(255), rgb.R)
	assert.Equal(t, uint8(0), rgb.G)
	assert.Equal(t, uint8(0), rgb.B)
}

func TestHSLToRGB_Green(t *testing.T) {
	rgb := color.HSLToRGB(120, 1.0, 0.5)
	assert.Equal(t, uint8(0), rgb.R)
	assert.Equal(t, uint8(255), rgb.G)
	assert.Equal(t, uint8(0), rgb.B)
}

func TestHSLToRGB_Blue(t *testing.T) {
	rgb := color.HSLToRGB(240, 1.0, 0.5)
	assert.Equal(t, uint8(0), rgb.R)
	assert.Equal(t, uint8(0), rgb.G)
	assert.Equal(t, uint8(255), rgb.B)
}

func TestHSLToRGB_Grayscale(t *testing.T) {
	// Saturation 0 should produce grayscale
	rgb := color.HSLToRGB(0, 0, 0.5)
	assert.Equal(t, rgb.R, rgb.G)
	assert.Equal(t, rgb.G, rgb.B)
}
