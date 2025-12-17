package color

// HSLToRGB converts HSL (Hue, Saturation, Lightness) color space to RGB.
//
// Parameters:
//   - h: Hue in degrees (0-360). Values wrap around (e.g., 370 is the same as 10).
//   - s: Saturation as a fraction (0.0-1.0). 0 is grayscale, 1 is fully saturated.
//   - l: Lightness as a fraction (0.0-1.0). 0 is black, 0.5 is pure color, 1 is white.
//
// HSL is often more intuitive than RGB for generating color variations, as you can
// easily adjust brightness (lightness) or color intensity (saturation) while keeping
// the same hue.
//
// Example:
//
//	// Create a pure red
//	red := color.HSLToRGB(0, 1.0, 0.5)
//
//	// Create a darker, less saturated red
//	burgundy := color.HSLToRGB(0, 0.7, 0.3)
//
//	// Create colors by rotating hue
//	for i := 0; i < 12; i++ {
//	    hue := float64(i) * 30.0 // Every 30 degrees
//	    c := color.HSLToRGB(hue, 1.0, 0.5)
//	    fmt.Print(c.Apply("█", false))
//	}
func HSLToRGB(h, s, l float64) RGB {
	// Normalize hue to 0-1 range
	h = h / 360.0

	var r, g, b float64

	if s == 0 {
		// Grayscale
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}

	return RGB{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
	}
}

// hueToRGB is a helper function for HSL to RGB conversion.
// It calculates the RGB component value for a given hue position.
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}
