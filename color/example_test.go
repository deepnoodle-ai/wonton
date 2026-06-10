package color_test

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/color"
)

// Example_basicColors demonstrates using standard ANSI colors. The Apply
// helpers automatically emit plain text when colors are disabled (NO_COLOR,
// piped output, etc.), so no conditional logic is needed.
func Example_basicColors() {
	fmt.Println(color.Red.Apply("Error: something went wrong"))
	fmt.Println(color.Green.Apply("Success: operation completed"))
	fmt.Println(color.Yellow.Apply("Warning: proceed with caution"))
	fmt.Println(color.Red.ApplyBold("FAILED"))
	fmt.Println(color.White.ApplyDim("(details below)"))
}

// Example_rgbColors demonstrates true color RGB values and hex parsing.
func Example_rgbColors() {
	orange := color.NewRGB(255, 128, 0)
	purple := color.MustHex("#8000ff")

	fmt.Println(orange.Apply("Orange text"))
	fmt.Println(purple.ApplyBg("Purple background"))
	fmt.Println(orange.Hex()) // "#ff8000"
}

// Example_gradient demonstrates creating and applying color gradients.
func Example_gradient() {
	// Color each cell of a bar with a red-to-blue gradient
	for _, c := range color.Gradient(color.NewRGB(255, 0, 0), color.NewRGB(0, 0, 255), 10) {
		fmt.Print(c.Apply("█"))
	}
	fmt.Println()

	// Or color a string directly, one grapheme cluster per gradient step
	fmt.Println(color.ApplyGradient("Hello, gradient!",
		color.MustHex("#ff5f5f"),
		color.MustHex("#5f87ff"),
	))
}

// Example_rainbowGradient demonstrates the rainbow gradient helpers.
func Example_rainbowGradient() {
	for _, c := range color.RainbowGradient(20) {
		fmt.Print(c.Apply("█"))
	}
	fmt.Println()

	for _, c := range color.SmoothRainbow(20) {
		fmt.Print(c.Apply("█"))
	}
	fmt.Println()
}

// Example_hslColors demonstrates converting between HSL and RGB.
func Example_hslColors() {
	// Create colors by rotating hue while keeping saturation and lightness fixed
	for i := 0; i < 12; i++ {
		c := color.HSLToRGB(float64(i)*30.0, 1.0, 0.5)
		fmt.Print(c.Apply("█"))
	}
	fmt.Println()

	// Derive a darker variant of an existing color via HSL
	h, s, l := color.MustHex("#ff8000").HSL()
	darker := color.HSLToRGB(h, s, l*0.6)
	fmt.Println(darker.Apply("Darker orange"))
}

// Example_conditionalColor demonstrates overriding automatic color detection.
func Example_conditionalColor() {
	// Colors are enabled automatically based on FORCE_COLOR, NO_COLOR, and
	// whether stdout is a terminal. Override for special cases, e.g. when
	// writing colored output to stderr while stdout is piped:
	color.Enabled = color.ShouldColorize(os.Stderr)
	fmt.Fprintln(os.Stderr, color.Red.Apply("error: details"))
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

// ExampleColor_Sprintf demonstrates formatted color output.
func ExampleColor_Sprintf() {
	count := 5
	fmt.Println(color.Red.Sprintf("Found %d errors", count))
	fmt.Println(color.Green.Sprintf("Processed %d items successfully", count))
}

// ExampleHex demonstrates parsing hex color strings.
func ExampleHex() {
	c, err := color.Hex("#5f87ff")
	if err != nil {
		panic(err)
	}
	fmt.Println(c.Apply("Cornflower-ish"))
}

// ExampleRGB_Lerp demonstrates interpolating between colors.
func ExampleRGB_Lerp() {
	red := color.NewRGB(255, 0, 0)
	blue := color.NewRGB(0, 0, 255)
	mid := red.Lerp(blue, 0.5)
	fmt.Println(mid.Hex())
	// Output: #800080
}

// ExampleMultiGradient demonstrates creating gradients with multiple stops.
func ExampleMultiGradient() {
	sunset := color.MultiGradient([]color.RGB{
		color.NewRGB(255, 0, 0),   // Red
		color.NewRGB(255, 128, 0), // Orange
		color.NewRGB(128, 0, 128), // Purple
	}, 10)

	for _, c := range sunset {
		fmt.Print(c.Apply("█"))
	}
	fmt.Println()
}

// ExampleShouldColorize demonstrates per-stream color decisions.
func ExampleShouldColorize() {
	if color.ShouldColorize(os.Stdout) {
		fmt.Println("stdout gets colors")
	} else {
		fmt.Println("stdout is piped or colors are disabled")
	}
}
