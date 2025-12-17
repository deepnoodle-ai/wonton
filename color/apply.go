package color

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/tty"
)

// ANSI escape code constants
const (
	Escape = "\033["
	Reset  = Escape + "0m"
	Bold   = Escape + "1m"
	Dim    = Escape + "2m"
)

func init() {
	// Respect NO_COLOR environment variable (https://no-color.org/)
	// If NO_COLOR is set (to any value), disable colors by default
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		Enabled = false
	}
}

// Apply applies the ANSI color to text as a foreground color.
func (c Color) Apply(text string) string {
	if c < 0 {
		return text
	}
	return c.ForegroundSeq() + text + Reset
}

// ApplyDim applies the ANSI color to text with dim attribute.
func (c Color) ApplyDim(text string) string {
	if c < 0 {
		return Dim + text + Reset
	}
	return c.ForegroundSeqDim() + text + Reset
}

// ApplyBg applies the ANSI color to text as a background color.
func (c Color) ApplyBg(text string) string {
	if c < 0 {
		return text
	}
	return c.BackgroundSeq() + text + Reset
}

// ApplyBold applies bold formatting to text.
func ApplyBold(text string) string {
	return Bold + text + Reset
}

// ApplyDim applies dim formatting to text.
func ApplyDim(text string) string {
	return Dim + text + Reset
}

// Sprintf formats a string with the color applied.
func (c Color) Sprintf(format string, args ...any) string {
	return c.Apply(fmt.Sprintf(format, args...))
}

// Sprint applies the color to all arguments concatenated.
func (c Color) Sprint(args ...any) string {
	return c.Apply(fmt.Sprint(args...))
}

// IsTerminal reports whether f is a terminal.
// This is a convenience wrapper around tty.IsTerminal.
func IsTerminal(f *os.File) bool {
	return tty.IsTerminal(f)
}

// ShouldColorize returns true if colors should be used for the given output.
// It checks both that the output is a terminal AND that NO_COLOR is not set.
// This is the recommended way to determine if colors should be enabled.
func ShouldColorize(f *os.File) bool {
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return false
	}
	return tty.IsTerminal(f)
}

// Enabled controls whether Colorize and ColorizeRGB produce colored output.
// This is automatically set to false if the NO_COLOR environment variable is set.
// Can be manually set to override the default behavior.
var Enabled = true

// Colorize returns the colored text if colors are enabled, otherwise plain text.
func Colorize(c Color, text string) string {
	if !Enabled {
		return text
	}
	return c.Apply(text)
}

// ColorizeIf returns colored text if the condition is true.
func ColorizeIf(enabled bool, c Color, text string) string {
	if !enabled {
		return text
	}
	return c.Apply(text)
}

// ColorizeRGB applies RGB color to text.
func ColorizeRGB(rgb RGB, text string) string {
	if !Enabled {
		return text
	}
	return rgb.ForegroundSeq() + text + Reset
}
