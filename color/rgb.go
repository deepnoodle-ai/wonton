package color

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RGB represents a true color RGB value with 8-bit channels (0-255).
// RGB colors can be used to create ANSI escape sequences for terminals
// that support 24-bit color.
type RGB struct {
	R, G, B uint8
}

// NewRGB creates a new RGB color with the specified red, green, and blue
// values. Each channel accepts values from 0-255.
//
// Example:
//
//	orange := color.NewRGB(255, 128, 0)
//	purple := color.NewRGB(128, 0, 255)
func NewRGB(r, g, b uint8) RGB {
	return RGB{R: r, G: g, B: b}
}

// Hex parses a hex color string into an RGB value. It accepts the forms
// "#rrggbb", "rrggbb", "#rgb", and "rgb" (case-insensitive).
//
// Example:
//
//	orange, err := color.Hex("#ff8800")
//	teal, err := color.Hex("088")
func Hex(s string) (RGB, error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return RGB{}, fmt.Errorf("color: invalid hex color %q", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("color: invalid hex color %q", s)
	}
	return RGB{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}, nil
}

// MustHex is like Hex but panics on invalid input. Use it for color
// constants known at compile time.
//
// Example:
//
//	var brand = color.MustHex("#5f87ff")
func MustHex(s string) RGB {
	rgb, err := Hex(s)
	if err != nil {
		panic(err)
	}
	return rgb
}

// Hex returns the color as a lowercase hex string in the form "#rrggbb".
func (rgb RGB) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", rgb.R, rgb.G, rgb.B)
}

// ForegroundSeq returns the ANSI escape sequence for RGB foreground color.
// It is not affected by Enabled; remember to append color.Reset after your
// text to return to default colors.
func (rgb RGB) ForegroundSeq() string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", rgb.R, rgb.G, rgb.B)
}

// BackgroundSeq returns the ANSI escape sequence for RGB background color.
// It is not affected by Enabled; remember to append color.Reset after your
// text to return to default colors.
func (rgb RGB) BackgroundSeq() string {
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", rgb.R, rgb.G, rgb.B)
}

// Apply applies the RGB color to text as a foreground color and appends a
// reset sequence. Returns plain text when Enabled is false.
//
// Example:
//
//	orange := color.NewRGB(255, 128, 0)
//	fmt.Println(orange.Apply("Orange text"))
func (rgb RGB) Apply(text string) string {
	if !Enabled {
		return text
	}
	return rgb.ForegroundSeq() + text + Reset
}

// ApplyBg applies the RGB color to text as a background color and appends a
// reset sequence. Returns plain text when Enabled is false.
//
// Example:
//
//	navy := color.NewRGB(0, 0, 128)
//	fmt.Println(navy.ApplyBg(" DEPLOY "))
func (rgb RGB) ApplyBg(text string) string {
	if !Enabled {
		return text
	}
	return rgb.BackgroundSeq() + text + Reset
}

// Sprintf formats a string with the color applied, combining fmt.Sprintf
// and Apply.
func (rgb RGB) Sprintf(format string, args ...any) string {
	return rgb.Apply(fmt.Sprintf(format, args...))
}

// Sprint applies the color to all arguments concatenated, combining
// fmt.Sprint and Apply.
func (rgb RGB) Sprint(args ...any) string {
	return rgb.Apply(fmt.Sprint(args...))
}

// Lerp linearly interpolates between this color and other in RGB space.
// t is clamped to [0, 1], where 0 returns the receiver and 1 returns other.
//
// Example:
//
//	mid := color.NewRGB(255, 0, 0).Lerp(color.NewRGB(0, 0, 255), 0.5)
func (rgb RGB) Lerp(other RGB, t float64) RGB {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return RGB{
		R: uint8(math.Round(float64(rgb.R)*(1-t) + float64(other.R)*t)),
		G: uint8(math.Round(float64(rgb.G)*(1-t) + float64(other.G)*t)),
		B: uint8(math.Round(float64(rgb.B)*(1-t) + float64(other.B)*t)),
	}
}
