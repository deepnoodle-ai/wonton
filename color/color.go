// Package color provides color types and utilities for terminal rendering.
package color

import "fmt"

// Color represents an ANSI color
type Color int

// ANSI color constants
const (
	Default Color = iota
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	BrightBlack
	BrightRed
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	BrightWhite
)

// ForegroundCode returns the ANSI code for foreground color
func (c Color) ForegroundCode() string {
	switch c {
	case Black:
		return "30"
	case Red:
		return "31"
	case Green:
		return "32"
	case Yellow:
		return "33"
	case Blue:
		return "34"
	case Magenta:
		return "35"
	case Cyan:
		return "36"
	case White:
		return "37"
	case BrightBlack:
		return "90"
	case BrightRed:
		return "91"
	case BrightGreen:
		return "92"
	case BrightYellow:
		return "93"
	case BrightBlue:
		return "94"
	case BrightMagenta:
		return "95"
	case BrightCyan:
		return "96"
	case BrightWhite:
		return "97"
	default:
		return "39"
	}
}

// BackgroundCode returns the ANSI code for background color
func (c Color) BackgroundCode() string {
	switch c {
	case Black:
		return "40"
	case Red:
		return "41"
	case Green:
		return "42"
	case Yellow:
		return "43"
	case Blue:
		return "44"
	case Magenta:
		return "45"
	case Cyan:
		return "46"
	case White:
		return "47"
	case BrightBlack:
		return "100"
	case BrightRed:
		return "101"
	case BrightGreen:
		return "102"
	case BrightYellow:
		return "103"
	case BrightBlue:
		return "104"
	case BrightMagenta:
		return "105"
	case BrightCyan:
		return "106"
	case BrightWhite:
		return "107"
	default:
		return "49"
	}
}

// RGB represents an RGB color
type RGB struct {
	R, G, B uint8
}

// NewRGB creates a new RGB color
func NewRGB(r, g, b uint8) RGB {
	return RGB{R: r, G: g, B: b}
}

// Foreground returns the ANSI escape sequence for RGB foreground
func (rgb RGB) Foreground() string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", rgb.R, rgb.G, rgb.B)
}

// Background returns the ANSI escape sequence for RGB background
func (rgb RGB) Background() string {
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", rgb.R, rgb.G, rgb.B)
}

// Apply applies RGB color to text
func (rgb RGB) Apply(text string, background bool) string {
	if background {
		return rgb.Background() + text + "\033[0m"
	}
	return rgb.Foreground() + text + "\033[0m"
}
