package color

import (
	"strings"

	"github.com/deepnoodle-ai/wonton/runewidth"
)

// Gradient creates a linear gradient between two RGB colors with the
// specified number of steps. Each step is an RGB color interpolated between
// start and end.
//
// If steps <= 0, returns nil. If steps is 1, returns only the start color.
//
// Example:
//
//	// Create a red-to-blue gradient with 5 steps
//	gradient := color.Gradient(
//	    color.NewRGB(255, 0, 0),
//	    color.NewRGB(0, 0, 255),
//	    5,
//	)
//	for _, c := range gradient {
//	    fmt.Print(c.Apply("█"))
//	}
func Gradient(start, end RGB, steps int) []RGB {
	return MultiGradient([]RGB{start, end}, steps)
}

// MultiGradient creates a gradient that transitions through multiple color
// stops. The gradient is divided evenly across all color stops,
// interpolating linearly between each adjacent pair.
//
// If steps <= 0 or stops is empty, returns nil. If steps is 1, returns only
// the first stop. If stops contains one color, every element is that color.
//
// Example:
//
//	// Create a sunset gradient: red -> orange -> purple
//	sunset := color.MultiGradient([]color.RGB{
//	    color.NewRGB(255, 0, 0),     // Red
//	    color.NewRGB(255, 128, 0),   // Orange
//	    color.NewRGB(128, 0, 128),   // Purple
//	}, 20)
func MultiGradient(stops []RGB, steps int) []RGB {
	if len(stops) == 0 || steps <= 0 {
		return nil
	}
	if steps == 1 {
		return []RGB{stops[0]}
	}
	colors := make([]RGB, steps)
	if len(stops) == 1 {
		for i := range colors {
			colors[i] = stops[0]
		}
		return colors
	}
	for i := range colors {
		position := float64(i) / float64(steps-1) * float64(len(stops)-1)
		segment := int(position)
		if segment >= len(stops)-1 {
			colors[i] = stops[len(stops)-1]
		} else {
			localT := position - float64(segment)
			colors[i] = stops[segment].Lerp(stops[segment+1], localT)
		}
	}
	return colors
}

// rainbowStops are the classic rainbow color stops used by RainbowGradient.
var rainbowStops = []RGB{
	{255, 0, 0},   // Red
	{255, 127, 0}, // Orange
	{255, 255, 0}, // Yellow
	{0, 255, 0},   // Green
	{0, 0, 255},   // Blue
	{75, 0, 130},  // Indigo
	{148, 0, 211}, // Violet
}

// RainbowGradient creates a rainbow gradient using the classic rainbow
// color stops: red, orange, yellow, green, blue, indigo, and violet. The
// gradient interpolates linearly between these stops in RGB space. For a
// smoother, more perceptually uniform rainbow, consider SmoothRainbow.
//
// If steps <= 0, returns nil. If steps is 1, returns only red.
//
// Example:
//
//	rainbow := color.RainbowGradient(20)
//	for _, c := range rainbow {
//	    fmt.Print(c.Apply("█"))
//	}
func RainbowGradient(steps int) []RGB {
	return MultiGradient(rainbowStops, steps)
}

// SmoothRainbow creates a smooth rainbow gradient by rotating hue through
// HSL color space from 0 to 360 degrees with full saturation and 50%
// lightness. This produces smoother transitions than RainbowGradient, which
// interpolates in RGB space between fixed stops.
//
// If steps <= 0, returns nil.
//
// Example:
//
//	rainbow := color.SmoothRainbow(100)
//	for _, c := range rainbow {
//	    fmt.Print(c.Apply("█"))
//	}
func SmoothRainbow(steps int) []RGB {
	if steps <= 0 {
		return nil
	}
	colors := make([]RGB, steps)
	for i := range colors {
		hue := float64(i) / float64(steps) * 360.0
		colors[i] = HSLToRGB(hue, 1.0, 0.5)
	}
	return colors
}

// ApplyGradient colors text with a gradient across its characters, spread
// evenly over the given color stops, and appends a reset sequence. A single
// stop colors the whole text in that color. Characters are grapheme
// clusters, so emoji and combining marks are never split mid-sequence.
// Returns plain text when Enabled is false or no stops are given.
//
// Example:
//
//	banner := color.ApplyGradient("WONTON",
//	    color.MustHex("#ff5f5f"),
//	    color.MustHex("#5f87ff"),
//	)
func ApplyGradient(text string, stops ...RGB) string {
	if !Enabled || len(stops) == 0 || text == "" {
		return text
	}
	var clusters []string
	for cluster := range runewidth.Graphemes(text) {
		clusters = append(clusters, cluster)
	}
	colors := MultiGradient(stops, len(clusters))
	var b strings.Builder
	for i, cluster := range clusters {
		b.WriteString(colors[i].ForegroundSeq())
		b.WriteString(cluster)
	}
	b.WriteString(Reset)
	return b.String()
}
