package color_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
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
		{color.NoColor, "39"},
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
		{color.NoColor, "49"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.c.BackgroundCode())
		})
	}
}

func TestColor_ExtendedPalette(t *testing.T) {
	// Test 256-color palette
	tests := []struct {
		n          uint8
		expectedFg string
		expectedBg string
	}{
		{16, "38;5;16", "48;5;16"},
		{196, "38;5;196", "48;5;196"}, // bright red in 256 palette
		{232, "38;5;232", "48;5;232"}, // grayscale start
		{255, "38;5;255", "48;5;255"}, // grayscale end
	}

	for _, tt := range tests {
		c := color.Palette(tt.n)
		assert.Equal(t, tt.expectedFg, c.ForegroundCode())
		assert.Equal(t, tt.expectedBg, c.BackgroundCode())
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

func TestRainbowGradient_SingleStep(t *testing.T) {
	colors := color.RainbowGradient(1)
	assert.Len(t, colors, 1)
	assert.Equal(t, color.NewRGB(255, 0, 0), colors[0])
}

func TestRainbowGradient_MultipleSteps(t *testing.T) {
	colors := color.RainbowGradient(10)
	assert.Len(t, colors, 10)
	// First should be red
	assert.Equal(t, color.NewRGB(255, 0, 0), colors[0])
	// Should have variation in colors
	assert.NotEqual(t, colors[0], colors[5])
}

func TestSmoothRainbow(t *testing.T) {
	colors := color.SmoothRainbow(10)
	assert.Len(t, colors, 10)
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

func TestColor_ForegroundSeq(t *testing.T) {
	assert.Equal(t, "\033[31m", color.Red.ForegroundSeq())
	assert.Equal(t, "\033[92m", color.BrightGreen.ForegroundSeq())
	assert.Equal(t, "", color.NoColor.ForegroundSeq())
}

func TestColor_BackgroundSeq(t *testing.T) {
	assert.Equal(t, "\033[41m", color.Red.BackgroundSeq())
	assert.Equal(t, "\033[102m", color.BrightGreen.BackgroundSeq())
	assert.Equal(t, "", color.NoColor.BackgroundSeq())
}

func TestColor_ForegroundSeqDim(t *testing.T) {
	assert.Equal(t, "\033[2;31m", color.Red.ForegroundSeqDim())
	assert.Equal(t, "\033[2m", color.NoColor.ForegroundSeqDim())
}

func TestShouldColorize_RespectsNO_COLOR(t *testing.T) {
	// Save original state
	originalValue, hadValue := os.LookupEnv("NO_COLOR")

	// Test with NO_COLOR set
	os.Setenv("NO_COLOR", "1")
	assert.False(t, color.ShouldColorize(os.Stdout))

	// Test with NO_COLOR set to empty string (still counts as set)
	os.Setenv("NO_COLOR", "")
	assert.False(t, color.ShouldColorize(os.Stdout))

	// Test with NO_COLOR unset (behavior depends on whether stdout is a TTY)
	os.Unsetenv("NO_COLOR")
	// Can't easily test the true case without a real TTY, but we can verify
	// the function doesn't panic and returns a boolean
	_ = color.ShouldColorize(os.Stdout)

	// Restore original state
	if hadValue {
		os.Setenv("NO_COLOR", originalValue)
	} else {
		os.Unsetenv("NO_COLOR")
	}
}

func TestColorize_RespectsEnabled(t *testing.T) {
	// Save original state
	originalEnabled := color.Enabled

	// Test with Enabled = true
	color.Enabled = true
	result := color.Colorize(color.Red, "test")
	assert.Contains(t, result, "\033[")
	assert.Contains(t, result, "test")

	// Test with Enabled = false
	color.Enabled = false
	result = color.Colorize(color.Red, "test")
	assert.Equal(t, "test", result)

	// Restore original state
	color.Enabled = originalEnabled
}

// Example_basicColors demonstrates using standard ANSI colors.
func Example_basicColors() {
	fmt.Println(color.Red.Apply("Error: something went wrong"))
	fmt.Println(color.Green.Apply("Success: operation completed"))
	fmt.Println(color.Yellow.Apply("Warning: proceed with caution"))
	fmt.Println(color.Blue.Apply("Info: processing data"))
}

// Example_rgbColors demonstrates using true color RGB values.
func Example_rgbColors() {
	orange := color.NewRGB(255, 128, 0)
	purple := color.NewRGB(128, 0, 255)

	fmt.Println(orange.Apply("Orange text", false))
	fmt.Println(purple.Apply("Purple background", true))
}

// Example_gradient demonstrates creating a color gradient.
func Example_gradient() {
	// Create a gradient from red to blue
	gradient := color.Gradient(
		color.NewRGB(255, 0, 0),
		color.NewRGB(0, 0, 255),
		10,
	)

	for _, c := range gradient {
		fmt.Print(c.Apply("█", false))
	}
	fmt.Println()
}

// Example_rainbowGradient demonstrates creating a rainbow gradient.
func Example_rainbowGradient() {
	rainbow := color.RainbowGradient(20)

	for _, c := range rainbow {
		fmt.Print(c.Apply("█", false))
	}
	fmt.Println()
}

// Example_hslColors demonstrates using HSL color space.
func Example_hslColors() {
	// Create colors by varying hue while keeping saturation and lightness constant
	for i := 0; i < 12; i++ {
		hue := float64(i) * 30.0 // Every 30 degrees
		c := color.HSLToRGB(hue, 1.0, 0.5)
		fmt.Print(c.Apply("█", false))
	}
	fmt.Println()

	// Create shades of red by varying lightness
	for i := 0; i < 5; i++ {
		lightness := 0.2 + float64(i)*0.15
		c := color.HSLToRGB(0, 1.0, lightness)
		fmt.Print(c.Apply("█", false))
	}
	fmt.Println()
}

// Example_conditionalColor demonstrates respecting terminal capabilities.
func Example_conditionalColor() {
	// Check if we should colorize output
	if color.ShouldColorize(os.Stdout) {
		fmt.Println(color.Green.Apply("Terminal supports colors"))
	} else {
		fmt.Println("Plain text output")
	}

	// Use the global Enabled variable
	originalEnabled := color.Enabled
	color.Enabled = true
	fmt.Println(color.Colorize(color.Blue, "This will be blue"))

	color.Enabled = false
	fmt.Println(color.Colorize(color.Blue, "This will be plain"))

	color.Enabled = originalEnabled
}

// ExampleColor_Apply demonstrates applying foreground colors.
func ExampleColor_Apply() {
	fmt.Println(color.Red.Apply("Error message"))
	fmt.Println(color.Green.Apply("Success message"))
	fmt.Println(color.BrightYellow.Apply("Bright warning"))
}

// ExampleColor_ApplyBg demonstrates applying background colors.
func ExampleColor_ApplyBg() {
	fmt.Println(color.Red.ApplyBg(" ERROR "))
	fmt.Println(color.Green.ApplyBg(" OK "))
}

// ExampleColor_ApplyDim demonstrates using dim colors for de-emphasis.
func ExampleColor_ApplyDim() {
	fmt.Println(color.White.Apply("Normal text"))
	fmt.Println(color.White.ApplyDim("Dimmed text (less important)"))
}

// ExampleColor_Sprintf demonstrates formatted color output.
func ExampleColor_Sprintf() {
	count := 5
	fmt.Println(color.Red.Sprintf("Found %d errors", count))
	fmt.Println(color.Green.Sprintf("Processed %d items successfully", count))
}

// ExampleNewRGB demonstrates creating custom RGB colors.
func ExampleNewRGB() {
	orange := color.NewRGB(255, 128, 0)
	teal := color.NewRGB(0, 128, 128)

	fmt.Println(orange.Apply("Orange text", false))
	fmt.Println(teal.Apply("Teal background", true))
}

// ExampleGradient demonstrates creating a linear gradient.
func ExampleGradient() {
	// Create a red-to-blue gradient
	gradient := color.Gradient(
		color.NewRGB(255, 0, 0),
		color.NewRGB(0, 0, 255),
		5,
	)

	for i, c := range gradient {
		fmt.Printf("Step %d: %s\n", i, c.Apply("█████", false))
	}
}

// ExampleMultiGradient demonstrates creating gradients with multiple stops.
func ExampleMultiGradient() {
	// Create a sunset gradient
	sunset := color.MultiGradient([]color.RGB{
		color.NewRGB(255, 0, 0),   // Red
		color.NewRGB(255, 128, 0), // Orange
		color.NewRGB(128, 0, 128), // Purple
	}, 10)

	for _, c := range sunset {
		fmt.Print(c.Apply("█", false))
	}
	fmt.Println()
}

// ExampleHSLToRGB demonstrates HSL to RGB conversion.
func ExampleHSLToRGB() {
	// Pure red
	red := color.HSLToRGB(0, 1.0, 0.5)
	fmt.Println(red.Apply("Red", false))

	// Dark red (low lightness)
	darkRed := color.HSLToRGB(0, 1.0, 0.3)
	fmt.Println(darkRed.Apply("Dark Red", false))

	// Desaturated red (brownish)
	brown := color.HSLToRGB(0, 0.5, 0.3)
	fmt.Println(brown.Apply("Brown", false))
}

// ExampleShouldColorize demonstrates checking for color support.
func ExampleShouldColorize() {
	if color.ShouldColorize(os.Stdout) {
		fmt.Println(color.Green.Apply("Colors are enabled"))
	} else {
		fmt.Println("Colors are disabled")
	}
}

// ExampleApplyBold demonstrates bold text formatting.
func ExampleApplyBold() {
	fmt.Println(color.ApplyBold("Important:"), "This is a message")
	fmt.Println("Normal text without bold")
}

// ExampleApplyDim demonstrates dim text formatting.
func ExampleApplyDim() {
	fmt.Println("Normal text")
	fmt.Println(color.ApplyDim("(This is less important)"))
}
