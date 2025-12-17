package slog

import (
	"log/slog"

	"github.com/deepnoodle-ai/wonton/color"
)

// coloredValue wraps a slog.Value with a color.
type coloredValue struct {
	slog.Value
	Color color.Color
}

// LogValue implements slog.LogValuer.
func (v coloredValue) LogValue() slog.Value {
	return v.Value
}

// Err returns a red-colored attribute for errors.
// The attribute key is "err" and the value is the error.
//
// When used with a non-colorized handler, it behaves as slog.Any("err", err).
//
// Example:
//
//	logger.Error("failed to connect", wontonslog.Err(err))
func Err(err error) slog.Attr {
	return Colored(color.BrightRed, slog.Any("err", err))
}

// Colored returns an attribute that will be displayed in the specified color.
// When used with a non-colorized handler, it behaves as a plain slog.Attr.
//
// Color values:
//   - 0-7: standard ANSI colors (black, red, green, yellow, blue, magenta, cyan, white)
//   - 8-15: bright ANSI colors
//   - 16-231: 216-color cube (6×6×6) via color.Palette()
//   - 232-255: grayscale (24 steps) via color.Palette()
//
// Example:
//
//	logger.Info("user created",
//	    wontonslog.Colored(color.Green, slog.String("name", user.Name)),
//	    wontonslog.Colored(color.Cyan, slog.Int("id", user.ID)),
//	    wontonslog.Colored(color.Palette(208), slog.String("status", "active")), // orange
//	)
func Colored(c color.Color, attr slog.Attr) slog.Attr {
	attr.Value = slog.AnyValue(coloredValue{attr.Value, c})
	return attr
}

// Red returns an attribute colored red.
func Red(attr slog.Attr) slog.Attr {
	return Colored(color.BrightRed, attr)
}

// Green returns an attribute colored green.
func Green(attr slog.Attr) slog.Attr {
	return Colored(color.BrightGreen, attr)
}

// Yellow returns an attribute colored yellow.
func Yellow(attr slog.Attr) slog.Attr {
	return Colored(color.BrightYellow, attr)
}

// Blue returns an attribute colored blue.
func Blue(attr slog.Attr) slog.Attr {
	return Colored(color.BrightBlue, attr)
}

// Cyan returns an attribute colored cyan.
func Cyan(attr slog.Attr) slog.Attr {
	return Colored(color.BrightCyan, attr)
}

// Magenta returns an attribute colored magenta.
func Magenta(attr slog.Attr) slog.Attr {
	return Colored(color.BrightMagenta, attr)
}
