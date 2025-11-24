package gooey

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStyle(t *testing.T) {
	style := NewStyle()
	assert.Equal(t, ColorDefault, style.Foreground)
	assert.Equal(t, ColorDefault, style.Background)
	assert.Nil(t, style.FgRGB)
	assert.Nil(t, style.BgRGB)
	assert.False(t, style.Bold)
	assert.False(t, style.Italic)
}

func TestStyle_WithForeground(t *testing.T) {
	style := NewStyle().WithForeground(ColorRed)
	assert.Equal(t, ColorRed, style.Foreground)
	assert.Nil(t, style.FgRGB, "WithForeground should clear RGB override")
}

func TestStyle_WithBackground(t *testing.T) {
	style := NewStyle().WithBackground(ColorBlue)
	assert.Equal(t, ColorBlue, style.Background)
	assert.Nil(t, style.BgRGB, "WithBackground should clear RGB override")
}

func TestStyle_WithFgRGB(t *testing.T) {
	rgb := NewRGB(255, 128, 0)
	style := NewStyle().WithFgRGB(rgb)
	require.NotNil(t, style.FgRGB)
	assert.Equal(t, uint8(255), style.FgRGB.R)
	assert.Equal(t, uint8(128), style.FgRGB.G)
	assert.Equal(t, uint8(0), style.FgRGB.B)
}

func TestStyle_WithBgRGB(t *testing.T) {
	rgb := NewRGB(0, 128, 255)
	style := NewStyle().WithBgRGB(rgb)
	require.NotNil(t, style.BgRGB)
	assert.Equal(t, uint8(0), style.BgRGB.R)
	assert.Equal(t, uint8(128), style.BgRGB.G)
	assert.Equal(t, uint8(255), style.BgRGB.B)
}

func TestStyle_AttributeChaining(t *testing.T) {
	style := NewStyle().WithBold().WithItalic().WithUnderline()
	assert.True(t, style.Bold)
	assert.True(t, style.Italic)
	assert.True(t, style.Underline)
	assert.False(t, style.Strikethrough)
}

func TestStyle_AllAttributes(t *testing.T) {
	style := NewStyle().
		WithBold().
		WithItalic().
		WithUnderline().
		WithStrikethrough().
		WithBlink().
		WithReverse().
		WithDim()

	assert.True(t, style.Bold)
	assert.True(t, style.Italic)
	assert.True(t, style.Underline)
	assert.True(t, style.Strikethrough)
	assert.True(t, style.Blink)
	assert.True(t, style.Reverse)
	assert.True(t, style.Dim)
}

func TestStyle_String_DefaultStyle(t *testing.T) {
	style := NewStyle()
	output := style.String()
	assert.Contains(t, output, "\033[")
	assert.Contains(t, output, "0") // Reset code
}

func TestStyle_String_WithBold(t *testing.T) {
	style := NewStyle().WithBold()
	output := style.String()
	assert.Contains(t, output, "1") // Bold code
}

func TestStyle_String_WithForegroundColor(t *testing.T) {
	style := NewStyle().WithForeground(ColorRed)
	output := style.String()
	assert.Contains(t, output, "31") // Red foreground
}

func TestStyle_String_WithBackgroundColor(t *testing.T) {
	style := NewStyle().WithBackground(ColorGreen)
	output := style.String()
	assert.Contains(t, output, "42") // Green background
}

func TestStyle_String_WithRGBForeground(t *testing.T) {
	rgb := NewRGB(255, 128, 64)
	style := NewStyle().WithFgRGB(rgb)
	output := style.String()
	assert.Contains(t, output, "38;2;255;128;64")
}

func TestStyle_String_WithRGBBackground(t *testing.T) {
	rgb := NewRGB(64, 128, 255)
	style := NewStyle().WithBgRGB(rgb)
	output := style.String()
	assert.Contains(t, output, "48;2;64;128;255")
}

func TestStyle_Apply(t *testing.T) {
	style := NewStyle().WithForeground(ColorRed)
	text := style.Apply("Hello")
	assert.Contains(t, text, "Hello")
	assert.Contains(t, text, "\033[")
	assert.Contains(t, text, "\033[0m") // Reset at end
}

func TestStyle_Apply_EmptyStyle(t *testing.T) {
	style := NewStyle()
	text := style.Apply("Hello")
	assert.Equal(t, "Hello", text, "Empty style should not modify text")
}

func TestStyle_IsEmpty(t *testing.T) {
	assert.True(t, NewStyle().IsEmpty())
	assert.False(t, NewStyle().WithBold().IsEmpty())
	assert.False(t, NewStyle().WithForeground(ColorRed).IsEmpty())
}

func TestColor_ForegroundCode_AllColors(t *testing.T) {
	tests := []struct {
		color    Color
		expected string
	}{
		{ColorBlack, "30"},
		{ColorRed, "31"},
		{ColorGreen, "32"},
		{ColorYellow, "33"},
		{ColorBlue, "34"},
		{ColorMagenta, "35"},
		{ColorCyan, "36"},
		{ColorWhite, "37"},
		{ColorBrightBlack, "90"},
		{ColorBrightRed, "91"},
		{ColorBrightGreen, "92"},
		{ColorBrightYellow, "93"},
		{ColorBrightBlue, "94"},
		{ColorBrightMagenta, "95"},
		{ColorBrightCyan, "96"},
		{ColorBrightWhite, "97"},
		{ColorDefault, "39"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.color.foregroundCode())
		})
	}
}

func TestColor_BackgroundCode_AllColors(t *testing.T) {
	tests := []struct {
		color    Color
		expected string
	}{
		{ColorBlack, "40"},
		{ColorRed, "41"},
		{ColorGreen, "42"},
		{ColorYellow, "43"},
		{ColorBlue, "44"},
		{ColorMagenta, "45"},
		{ColorCyan, "46"},
		{ColorWhite, "47"},
		{ColorBrightBlack, "100"},
		{ColorBrightRed, "101"},
		{ColorBrightGreen, "102"},
		{ColorBrightYellow, "103"},
		{ColorBrightBlue, "104"},
		{ColorBrightMagenta, "105"},
		{ColorBrightCyan, "106"},
		{ColorBrightWhite, "107"},
		{ColorDefault, "49"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.color.backgroundCode())
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

func TestStyle_RGBOverridesColor(t *testing.T) {
	// When RGB is set, it should be used instead of Color
	rgb := NewRGB(100, 150, 200)
	style := NewStyle().WithForeground(ColorRed).WithFgRGB(rgb)
	output := style.String()
	// Should contain RGB code, not color code
	assert.Contains(t, output, "38;2;100;150;200")
	assert.NotContains(t, output, "31") // Red should not appear
}

func TestStyle_ColorOverridesRGB(t *testing.T) {
	// WithForeground should clear RGB
	rgb := NewRGB(100, 150, 200)
	style := NewStyle().WithFgRGB(rgb).WithForeground(ColorRed)
	assert.Nil(t, style.FgRGB)
	assert.Equal(t, ColorRed, style.Foreground)
}

func TestStyle_String_CombinedAttributes(t *testing.T) {
	style := NewStyle().
		WithForeground(ColorRed).
		WithBackground(ColorBlue).
		WithBold().
		WithUnderline()

	output := style.String()
	assert.Contains(t, output, "0")  // Reset
	assert.Contains(t, output, "1")  // Bold
	assert.Contains(t, output, "4")  // Underline
	assert.Contains(t, output, "31") // Red fg
	assert.Contains(t, output, "44") // Blue bg

	// Should be properly formatted
	assert.True(t, strings.HasPrefix(output, "\033["))
	assert.True(t, strings.HasSuffix(output, "m"))
}
