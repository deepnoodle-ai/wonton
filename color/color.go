// Package color provides ANSI color types and utilities for terminal rendering.
//
// The package supports three color systems:
//   - Standard and bright ANSI colors (0-15)
//   - Extended 256-color palette (16-255)
//   - True color RGB values, including hex parsing and HSL conversion
//
// # Automatic color handling
//
// The high-level helpers (Apply, ApplyBg, ApplyDim, ApplyBold, Sprint,
// Sprintf, and ApplyGradient) automatically produce plain text when colors
// are not appropriate: when NO_COLOR is set, when CLICOLOR=0 is set, or when
// stdout is not a terminal. FORCE_COLOR and CLICOLOR_FORCE override in the
// other direction. This means the obvious code is also the correct code:
//
//	fmt.Println(color.Red.Apply("Error message"))
//	fmt.Println(color.Green.Apply("Success message"))
//
// The decision is captured in the Enabled variable at startup and can be
// overridden, for example from a command-line flag:
//
//	color.Enabled = true // force color on
//
// The low-level sequence functions (ForegroundSeq, BackgroundSeq, and the
// SGR code functions) are not affected by Enabled; they always produce
// escape sequences. Use them when you are managing terminal state yourself,
// as the terminal and tui packages do.
//
// # RGB colors and gradients
//
//	orange := color.MustHex("#ff8800")
//	fmt.Println(orange.Apply("Orange text"))
//
//	// Color a string with a gradient across its characters
//	fmt.Println(color.ApplyGradient("Hello, gradient!",
//	    color.NewRGB(255, 0, 0),
//	    color.NewRGB(0, 0, 255),
//	))
package color

import (
	"fmt"
	"strconv"
)

// ANSI escape sequences for text styling and reset.
const (
	Reset = "\033[0m" // Resets all attributes (color, bold, dim, etc.)
	Bold  = "\033[1m" // Bold/increased intensity
	Dim   = "\033[2m" // Dim/faint/decreased intensity
)

// Color represents an ANSI color. Values 0-7 are standard colors, 8-15 are
// bright colors, and 16-255 are extended 256-color palette colors. Use
// Default (-1) to represent the terminal's default color. Values outside
// [-1, 255] are not meaningful and produce undefined escape sequences.
type Color int

// Default represents the terminal's default color: the absence of an
// explicit color choice. Its escape sequences (ForegroundSeq, BackgroundSeq)
// are empty strings and Apply returns text unchanged, while its SGR codes
// (ForegroundCode, BackgroundCode) are the "default color" parameters 39 and
// 49, which actively reset a previously set color.
const Default Color = -1

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

// Palette returns the Color for an entry in the 256-color terminal palette.
// Entries 0-15 are the standard and bright ANSI colors, 16-231 form a 6x6x6
// color cube, and 232-255 are a grayscale ramp.
//
// Note that entries 0-15 render using the classic SGR codes (30-37, 90-97)
// rather than the extended "38;5;N" form; terminals display these
// identically.
func Palette(n uint8) Color {
	return Color(n)
}

// ForegroundCode returns the ANSI SGR parameter for foreground color.
// For basic colors (0-7) returns "30"-"37".
// For bright colors (8-15) returns "90"-"97".
// For extended colors (16-255) returns "38;5;N".
// For Default returns "39", which resets to the terminal's default
// foreground (unlike ForegroundSeq, which returns an empty string).
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
	return "38;5;" + strconv.Itoa(int(c))
}

// BackgroundCode returns the ANSI SGR parameter for background color.
// For basic colors (0-7) returns "40"-"47".
// For bright colors (8-15) returns "100"-"107".
// For extended colors (16-255) returns "48;5;N".
// For Default returns "49", which resets to the terminal's default
// background (unlike BackgroundSeq, which returns an empty string).
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
	return "48;5;" + strconv.Itoa(int(c))
}

// ForegroundSeq returns the full ANSI escape sequence for foreground color,
// or an empty string for Default. It is not affected by Enabled.
func (c Color) ForegroundSeq() string {
	if c < 0 {
		return ""
	}
	return "\033[" + c.ForegroundCode() + "m"
}

// BackgroundSeq returns the full ANSI escape sequence for background color,
// or an empty string for Default. It is not affected by Enabled.
func (c Color) BackgroundSeq() string {
	if c < 0 {
		return ""
	}
	return "\033[" + c.BackgroundCode() + "m"
}

// ForegroundSeqDim returns the ANSI escape sequence for foreground color
// with the dim (faint) attribute. It is not affected by Enabled.
func (c Color) ForegroundSeqDim() string {
	if c < 0 {
		return Dim
	}
	return "\033[2;" + c.ForegroundCode() + "m"
}

// ForegroundSeqBold returns the ANSI escape sequence for foreground color
// with the bold attribute. It is not affected by Enabled.
func (c Color) ForegroundSeqBold() string {
	if c < 0 {
		return Bold
	}
	return "\033[1;" + c.ForegroundCode() + "m"
}

// Apply applies the ANSI color to text as a foreground color and appends a
// reset sequence. This is the recommended way to colorize text: it returns
// plain text when Enabled is false (NO_COLOR set, output piped, etc.).
//
// Example:
//
//	fmt.Println(color.Red.Apply("Error:"), "Something went wrong")
//	fmt.Println(color.Green.Apply("Success:"), "Operation completed")
func (c Color) Apply(text string) string {
	if !Enabled || c < 0 {
		return text
	}
	return c.ForegroundSeq() + text + Reset
}

// ApplyBg applies the ANSI color to text as a background color and appends
// a reset sequence. Returns plain text when Enabled is false.
//
// Example:
//
//	fmt.Println(color.Red.ApplyBg(" ERROR "))
func (c Color) ApplyBg(text string) string {
	if !Enabled || c < 0 {
		return text
	}
	return c.BackgroundSeq() + text + Reset
}

// ApplyDim applies the ANSI color to text with the dim (faint) attribute,
// useful for de-emphasizing secondary information. Returns plain text when
// Enabled is false.
//
// Example:
//
//	fmt.Println(color.White.ApplyDim("(optional)"))
func (c Color) ApplyDim(text string) string {
	if !Enabled {
		return text
	}
	return c.ForegroundSeqDim() + text + Reset
}

// ApplyBold applies the ANSI color to text with the bold attribute. Returns
// plain text when Enabled is false.
//
// Example:
//
//	fmt.Println(color.Red.ApplyBold("FAILED"))
func (c Color) ApplyBold(text string) string {
	if !Enabled {
		return text
	}
	return c.ForegroundSeqBold() + text + Reset
}

// Sprintf formats a string with the color applied, combining fmt.Sprintf
// and Apply.
//
// Example:
//
//	fmt.Println(color.Red.Sprintf("Found %d errors", count))
func (c Color) Sprintf(format string, args ...any) string {
	return c.Apply(fmt.Sprintf(format, args...))
}

// Sprint applies the color to all arguments concatenated, combining
// fmt.Sprint and Apply.
//
// Example:
//
//	fmt.Println(color.Green.Sprint("Success: ", count, " items processed"))
func (c Color) Sprint(args ...any) string {
	return c.Apply(fmt.Sprint(args...))
}

// ApplyBold applies bold formatting to text without color. Returns plain
// text when Enabled is false.
//
// Example:
//
//	fmt.Println(color.ApplyBold("Important:"), "Read this carefully")
func ApplyBold(text string) string {
	if !Enabled {
		return text
	}
	return Bold + text + Reset
}

// ApplyDim applies dim (faint) formatting to text without color. Returns
// plain text when Enabled is false.
//
// Example:
//
//	fmt.Println(color.ApplyDim("// This is a comment"))
func ApplyDim(text string) string {
	if !Enabled {
		return text
	}
	return Dim + text + Reset
}
