package color

// Gradient creates a gradient between two RGB colors
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

// RainbowGradient creates a rainbow gradient with the full spectrum
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

// SmoothRainbow creates a smooth rainbow using proper HSL conversion
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

// MultiGradient creates a gradient through multiple color stops
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
