package color

// Gradient creates a linear gradient between two RGB colors with the specified
// number of steps. Each step is an RGB color interpolated between start and end.
//
// If steps is 1 or less, returns a single-element slice containing only the start color.
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
//	    fmt.Print(c.Apply("█", false))
//	}
func Gradient(start, end RGB, steps int) []RGB {
	if steps <= 1 {
		return []RGB{start}
	}

	colors := make([]RGB, steps)
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		colors[i] = RGB{
			R: uint8(float64(start.R) + t*float64(end.R-start.R)),
			G: uint8(float64(start.G) + t*float64(end.G-start.G)),
			B: uint8(float64(start.B) + t*float64(end.B-start.B)),
		}
	}
	return colors
}

// RainbowGradient creates a rainbow gradient with the full spectrum using
// classic rainbow color stops: red, orange, yellow, green, blue, indigo, and violet.
//
// The gradient interpolates linearly between these color stops in RGB space.
// For a smoother, more perceptually uniform rainbow, consider using SmoothRainbow instead.
//
// If steps is 1 or less, returns a single-element slice containing red.
//
// Example:
//
//	rainbow := color.RainbowGradient(20)
//	for _, c := range rainbow {
//	    fmt.Print(c.Apply("█", false))
//	}
func RainbowGradient(steps int) []RGB {
	if steps <= 1 {
		return []RGB{NewRGB(255, 0, 0)}
	}

	// Use the classic rainbow color stops for a proper spectrum
	rainbowStops := []RGB{
		NewRGB(255, 0, 0),   // Red
		NewRGB(255, 127, 0), // Orange
		NewRGB(255, 255, 0), // Yellow
		NewRGB(0, 255, 0),   // Green
		NewRGB(0, 0, 255),   // Blue
		NewRGB(75, 0, 130),  // Indigo
		NewRGB(148, 0, 211), // Violet
	}

	colors := make([]RGB, steps)

	for i := 0; i < steps; i++ {
		// Calculate position in the rainbow (0.0 to 1.0)
		t := float64(i) / float64(steps-1)
		// Scale to the number of segments (6 transitions between 7 colors)
		position := t * 6.0
		segment := int(position)
		localT := position - float64(segment)

		if segment >= 6 {
			colors[i] = rainbowStops[6]
		} else {
			start := rainbowStops[segment]
			end := rainbowStops[segment+1]
			colors[i] = RGB{
				R: uint8(float64(start.R)*(1-localT) + float64(end.R)*localT),
				G: uint8(float64(start.G)*(1-localT) + float64(end.G)*localT),
				B: uint8(float64(start.B)*(1-localT) + float64(end.B)*localT),
			}
		}
	}

	return colors
}

// SmoothRainbow creates a smooth rainbow gradient using HSL color space conversion.
// This produces a perceptually uniform rainbow by varying the hue from 0 to 360 degrees
// while keeping saturation at 100% and lightness at 50%.
//
// This method produces smoother color transitions than RainbowGradient, which uses
// RGB interpolation between fixed color stops.
//
// If steps is 1 or less, returns a single-element slice containing red.
//
// Example:
//
//	rainbow := color.SmoothRainbow(100)
//	for _, c := range rainbow {
//	    fmt.Print(c.Apply("█", false))
//	}
func SmoothRainbow(steps int) []RGB {
	if steps <= 1 {
		return []RGB{NewRGB(255, 0, 0)}
	}

	colors := make([]RGB, steps)
	for i := 0; i < steps; i++ {
		hue := float64(i) / float64(steps) * 360.0
		colors[i] = HSLToRGB(hue, 1.0, 0.5)
	}
	return colors
}

// MultiGradient creates a gradient that transitions through multiple color stops.
// The gradient is divided evenly across all color stops, interpolating linearly
// between each adjacent pair.
//
// If stops is empty, returns an empty slice. If stops contains one color, returns
// a slice of length steps where all elements are that color.
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
	if len(stops) == 0 {
		return []RGB{}
	}
	if len(stops) == 1 {
		result := make([]RGB, steps)
		for i := range result {
			result[i] = stops[0]
		}
		return result
	}

	colors := make([]RGB, steps)

	for i := 0; i < steps; i++ {
		position := float64(i) / float64(steps-1) * float64(len(stops)-1)
		segment := int(position)
		if segment >= len(stops)-1 {
			colors[i] = stops[len(stops)-1]
		} else {
			localT := position - float64(segment)
			start := stops[segment]
			end := stops[segment+1]
			colors[i] = RGB{
				R: uint8(float64(start.R)*(1-localT) + float64(end.R)*localT),
				G: uint8(float64(start.G)*(1-localT) + float64(end.G)*localT),
				B: uint8(float64(start.B)*(1-localT) + float64(end.B)*localT),
			}
		}
	}
	return colors
}
