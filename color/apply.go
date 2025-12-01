package color

import (
	"fmt"
	"os"
)

// ANSI escape code constants
const (
	escape = "\033["
	reset  = escape + "0m"
	bold   = escape + "1m"
	dim    = escape + "2m"
)

// Apply applies the ANSI color to text as a foreground color.
func (c Color) Apply(text string) string {
	if c == Default {
		return text
	}
	return fmt.Sprintf("%s%sm%s%s", escape, c.ForegroundCode(), text, reset)
}

// ApplyBg applies the ANSI color to text as a background color.
func (c Color) ApplyBg(text string) string {
	if c == Default {
		return text
	}
	return fmt.Sprintf("%s%sm%s%s", escape, c.BackgroundCode(), text, reset)
}

// Bold applies bold formatting to text.
func Bold(text string) string {
	return bold + text + reset
}

// Dim applies dim formatting to text.
func Dim(text string) string {
	return dim + text + reset
}

// Sprintf formats a string with the color applied.
func (c Color) Sprintf(format string, args ...any) string {
	return c.Apply(fmt.Sprintf(format, args...))
}

// Sprint applies the color to all arguments concatenated.
func (c Color) Sprint(args ...any) string {
	return c.Apply(fmt.Sprint(args...))
}

// IsTerminal returns true if the given file is a terminal.
func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// NoColor can be set to true to disable all color output.
var NoColor = false

// Colorize returns the colored text if colors are enabled, otherwise plain text.
func Colorize(c Color, text string) string {
	if NoColor {
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
	if NoColor {
		return text
	}
	return rgb.Foreground() + text + reset
}
