package color

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/tty"
)

// ANSI escape code constants for text styling and color reset.
const (
	Escape = "\033["       // ANSI escape sequence prefix
	Reset  = Escape + "0m" // Resets all attributes (color, bold, dim, etc.)
	Bold   = Escape + "1m" // Bold/increased intensity
	Dim    = Escape + "2m" // Dim/faint/decreased intensity
)

func init() {
	// Seed the global Enabled flag from the environment. Command-line flags
	// take precedence and are handled by callers (e.g., the cli package).
	//
	// Precedence (highest first):
	//   1. FORCE_COLOR / CLICOLOR_FORCE (force on)
	//   2. NO_COLOR / CLICOLOR=0        (force off)
	//   3. Default (on; TTY detection is applied by ShouldColorize)
	if envForceColor() {
		Enabled = true
		return
	}
	if envNoColor() {
		Enabled = false
	}
}

// envNoColor reports whether NO_COLOR or CLICOLOR=0 is set in a way that
// disables colored output.
//
// NO_COLOR spec (https://no-color.org/): any non-empty value disables color.
// CLICOLOR spec (https://bixense.com/clicolors/): CLICOLOR=0 disables color.
func envNoColor() bool {
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return true
	}
	if v, ok := os.LookupEnv("CLICOLOR"); ok && v == "0" {
		return true
	}
	return false
}

// envForceColor reports whether FORCE_COLOR or CLICOLOR_FORCE is set in a
// way that forces colored output, per common cross-tool conventions.
// An empty value or "0" does not force.
func envForceColor() bool {
	if v, ok := os.LookupEnv("FORCE_COLOR"); ok && v != "" && v != "0" {
		return true
	}
	if v, ok := os.LookupEnv("CLICOLOR_FORCE"); ok && v != "" && v != "0" {
		return true
	}
	return false
}

// Apply applies the ANSI color to text as a foreground color and automatically
// appends a reset sequence. This is the recommended way to colorize text.
//
// Example:
//
//	fmt.Println(color.Red.Apply("Error:"), "Something went wrong")
//	fmt.Println(color.Green.Apply("Success:"), "Operation completed")
func (c Color) Apply(text string) string {
	if c < 0 {
		return text
	}
	return c.ForegroundSeq() + text + Reset
}

// ApplyDim applies the ANSI color to text with dim (faint) attribute.
// The dim attribute reduces the intensity of both the color and the text.
// This is useful for de-emphasizing secondary information.
//
// Example:
//
//	fmt.Println(color.White.ApplyDim("(optional)"))
func (c Color) ApplyDim(text string) string {
	if c < 0 {
		return Dim + text + Reset
	}
	return c.ForegroundSeqDim() + text + Reset
}

// ApplyBg applies the ANSI color to text as a background color and automatically
// appends a reset sequence.
//
// Example:
//
//	fmt.Println(color.Red.ApplyBg(" ERROR "))
func (c Color) ApplyBg(text string) string {
	if c < 0 {
		return text
	}
	return c.BackgroundSeq() + text + Reset
}

// ApplyBold applies bold formatting to text without color.
//
// Example:
//
//	fmt.Println(color.ApplyBold("Important:"), "Read this carefully")
func ApplyBold(text string) string {
	return Bold + text + Reset
}

// ApplyDim applies dim (faint) formatting to text without color.
// This reduces text intensity, making it appear lighter/grayed out.
//
// Example:
//
//	fmt.Println(color.ApplyDim("// This is a comment"))
func ApplyDim(text string) string {
	return Dim + text + Reset
}

// Sprintf formats a string with the color applied, combining fmt.Sprintf and Apply.
//
// Example:
//
//	fmt.Println(color.Red.Sprintf("Found %d errors", count))
func (c Color) Sprintf(format string, args ...any) string {
	return c.Apply(fmt.Sprintf(format, args...))
}

// Sprint applies the color to all arguments concatenated, combining fmt.Sprint and Apply.
//
// Example:
//
//	fmt.Println(color.Green.Sprint("Success: ", count, " items processed"))
func (c Color) Sprint(args ...any) string {
	return c.Apply(fmt.Sprint(args...))
}

// IsTerminal reports whether f is a terminal (TTY).
// This is a convenience wrapper around tty.IsTerminal.
//
// Returns true if f is connected to a terminal, false if it's redirected to a file
// or pipe.
//
// Example:
//
//	if color.IsTerminal(os.Stdout) {
//	    fmt.Println("Output is going to a terminal")
//	}
func IsTerminal(f *os.File) bool {
	return tty.IsTerminal(f)
}

// ShouldColorize reports whether colors should be used for the given output
// file, following the common precedence:
//
//  1. FORCE_COLOR / CLICOLOR_FORCE (non-empty, non-"0"): force on
//  2. NO_COLOR (non-empty) or CLICOLOR=0: force off
//  3. Auto-detect: true iff f is a terminal
//
// Command-line flags should take precedence over this function; callers
// typically set their own flag and only consult ShouldColorize when the user
// didn't pass an explicit override.
//
// Example:
//
//	if color.ShouldColorize(os.Stdout) {
//	    fmt.Println(color.Green.Apply("Colorized output"))
//	} else {
//	    fmt.Println("Plain output")
//	}
func ShouldColorize(f *os.File) bool {
	if envForceColor() {
		return true
	}
	if envNoColor() {
		return false
	}
	return tty.IsTerminal(f)
}

// Enabled controls whether Colorize and ColorizeRGB produce colored output.
// This is automatically set to false during package initialization if the NO_COLOR
// environment variable is set (per https://no-color.org/).
//
// You can manually set this variable to override the default behavior for your application.
//
// Example:
//
//	// Disable colors globally
//	color.Enabled = false
//
//	// Now Colorize returns plain text
//	text := color.Colorize(color.Red, "Error") // Returns "Error" without color codes
var Enabled = true

// Colorize returns the colored text if the Enabled variable is true, otherwise
// returns plain text. This provides a convenient way to conditionally apply colors
// based on a global setting.
//
// Example:
//
//	// Respects the Enabled variable
//	fmt.Println(color.Colorize(color.Red, "Error message"))
func Colorize(c Color, text string) string {
	if !Enabled {
		return text
	}
	return c.Apply(text)
}

// ColorizeIf returns colored text if the enabled parameter is true, otherwise
// returns plain text. This provides fine-grained control over when colors are applied.
//
// Example:
//
//	useColor := color.ShouldColorize(os.Stdout)
//	fmt.Println(color.ColorizeIf(useColor, color.Green, "Success"))
func ColorizeIf(enabled bool, c Color, text string) string {
	if !enabled {
		return text
	}
	return c.Apply(text)
}

// ColorizeRGB applies RGB color to text if the Enabled variable is true,
// otherwise returns plain text.
//
// Example:
//
//	orange := color.NewRGB(255, 128, 0)
//	fmt.Println(color.ColorizeRGB(orange, "Warning"))
func ColorizeRGB(rgb RGB, text string) string {
	if !Enabled {
		return text
	}
	return rgb.ForegroundSeq() + text + Reset
}
