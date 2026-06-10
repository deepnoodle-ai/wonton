package color

import "math"

// HSLToRGB converts HSL (Hue, Saturation, Lightness) color space to RGB.
//
// Parameters:
//   - h: Hue in degrees (0-360). Values wrap around (e.g., 370 is the same as 10).
//   - s: Saturation as a fraction (0.0-1.0). 0 is grayscale, 1 is fully saturated.
//   - l: Lightness as a fraction (0.0-1.0). 0 is black, 0.5 is pure color, 1 is white.
//
// HSL is often more intuitive than RGB for generating color variations, as
// you can easily adjust brightness (lightness) or color intensity
// (saturation) while keeping the same hue. Use RGB.HSL for the inverse
// conversion.
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
//	    fmt.Print(c.Apply("█"))
//	}
func HSLToRGB(h, s, l float64) RGB {
	// Wrap hue into [0, 360) and normalize to 0-1 range
	h = math.Mod(h, 360.0)
	if h < 0 {
		h += 360.0
	}
	h = h / 360.0

	// Clamp saturation and lightness to [0, 1]
	s = math.Min(math.Max(s, 0), 1)
	l = math.Min(math.Max(l, 0), 1)

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
		R: uint8(math.Round(r * 255)),
		G: uint8(math.Round(g * 255)),
		B: uint8(math.Round(b * 255)),
	}
}

// HSL converts the RGB color to HSL color space. It returns hue in degrees
// [0, 360), and saturation and lightness as fractions [0, 1]. This is the
// inverse of HSLToRGB.
//
// Example:
//
//	h, s, l := color.NewRGB(255, 128, 0).HSL()
//	darker := color.HSLToRGB(h, s, l*0.5)
func (rgb RGB) HSL() (h, s, l float64) {
	r := float64(rgb.R) / 255
	g := float64(rgb.G) / 255
	b := float64(rgb.B) / 255

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l = (max + min) / 2

	if max == min {
		return 0, 0, l // achromatic
	}

	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s, l
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
