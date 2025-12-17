// Package color provides color types and utilities for terminal rendering.
package color

import (
	"fmt"
	"strconv"
)

// Color represents an ANSI color. Values 0-7 are standard colors, 8-15 are
// bright colors, and 16-255 are extended 256-color palette colors.
// Use NoColor (-1) to represent the absence of color.
type Color int

// NoColor represents the absence of color.
const NoColor Color = -1

// Standard ANSI colors (0-7)
const (
	Black   Color = iota // 0
	Red                  // 1
	Green                // 2
	Yellow               // 3
	Blue                 // 4
	Magenta              // 5
	Cyan                 // 6
	White                // 7
)

// Bright ANSI colors (8-15)
const (
	BrightBlack   Color = iota + 8 // 8
	BrightRed                      // 9
	BrightGreen                    // 10
	BrightYellow                   // 11
	BrightBlue                     // 12
	BrightMagenta                  // 13
	BrightCyan                     // 14
	BrightWhite                    // 15
)

// Palette returns a Color for extended 256-color palette values (16-255).
// Values 16-231 are a 6x6x6 color cube, 232-255 are grayscale.
func Palette(n uint8) Color {
	return Color(n)
}

// ForegroundCode returns the ANSI SGR parameter for foreground color.
// For basic colors (0-7) returns "30"-"37".
// For bright colors (8-15) returns "90"-"97".
// For extended colors (16-255) returns "38;5;N".
func (c Color) ForegroundCode() string {
	if c < 0 {
		return "39" // default
	}
	if c < 8 {
		return strconv.Itoa(30 + int(c))
	}
	if c < 16 {
		return strconv.Itoa(90 + int(c) - 8)
	}
	return fmt.Sprintf("38;5;%d", c)
}

// BackgroundCode returns the ANSI SGR parameter for background color.
// For basic colors (0-7) returns "40"-"47".
// For bright colors (8-15) returns "100"-"107".
// For extended colors (16-255) returns "48;5;N".
func (c Color) BackgroundCode() string {
	if c < 0 {
		return "49" // default
	}
	if c < 8 {
		return strconv.Itoa(40 + int(c))
	}
	if c < 16 {
		return strconv.Itoa(100 + int(c) - 8)
	}
	return fmt.Sprintf("48;5;%d", c)
}

// ForegroundSeq returns the full ANSI escape sequence for foreground color.
func (c Color) ForegroundSeq() string {
	if c < 0 {
		return ""
	}
	return "\033[" + c.ForegroundCode() + "m"
}

// ForegroundSeqDim returns the ANSI escape sequence for foreground color
// with dim (faint) attribute.
func (c Color) ForegroundSeqDim() string {
	if c < 0 {
		return "\033[2m"
	}
	return "\033[2;" + c.ForegroundCode() + "m"
}

// BackgroundSeq returns the full ANSI escape sequence for background color.
func (c Color) BackgroundSeq() string {
	if c < 0 {
		return ""
	}
	return "\033[" + c.BackgroundCode() + "m"
}

// RGB represents an RGB color.
type RGB struct {
	R, G, B uint8
}

// NewRGB creates a new RGB color.
func NewRGB(r, g, b uint8) RGB {
	return RGB{R: r, G: g, B: b}
}

// ForegroundSeq returns the ANSI escape sequence for RGB foreground.
func (rgb RGB) ForegroundSeq() string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", rgb.R, rgb.G, rgb.B)
}

// BackgroundSeq returns the ANSI escape sequence for RGB background.
func (rgb RGB) BackgroundSeq() string {
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", rgb.R, rgb.G, rgb.B)
}

// Foreground returns the ANSI escape sequence for RGB foreground.
// Deprecated: Use ForegroundSeq instead.
func (rgb RGB) Foreground() string {
	return rgb.ForegroundSeq()
}

// Background returns the ANSI escape sequence for RGB background.
// Deprecated: Use BackgroundSeq instead.
func (rgb RGB) Background() string {
	return rgb.BackgroundSeq()
}

// Apply applies RGB color to text.
func (rgb RGB) Apply(text string, background bool) string {
	if background {
		return rgb.BackgroundSeq() + text + "\033[0m"
	}
	return rgb.ForegroundSeq() + text + "\033[0m"
}
