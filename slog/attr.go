package slog

import "log/slog"

// ANSI color codes for common use
const (
	ColorBlack         uint8 = 0
	ColorRed           uint8 = 1
	ColorGreen         uint8 = 2
	ColorYellow        uint8 = 3
	ColorBlue          uint8 = 4
	ColorMagenta       uint8 = 5
	ColorCyan          uint8 = 6
	ColorWhite         uint8 = 7
	ColorBrightBlack   uint8 = 8
	ColorBrightRed     uint8 = 9
	ColorBrightGreen   uint8 = 10
	ColorBrightYellow  uint8 = 11
	ColorBrightBlue    uint8 = 12
	ColorBrightMagenta uint8 = 13
	ColorBrightCyan    uint8 = 14
	ColorBrightWhite   uint8 = 15
)

// coloredValue wraps a slog.Value with a color.
type coloredValue struct {
	slog.Value
	Color uint8
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
//	logger.Error("failed to connect", gooeyslog.Err(err))
func Err(err error) slog.Attr {
	return Colored(ColorBrightRed, slog.Any("err", err))
}

// Colored returns an attribute that will be displayed in the specified color.
// When used with a non-colorized handler, it behaves as a plain slog.Attr.
//
// Color values:
//   - 0-7: standard ANSI colors (black, red, green, yellow, blue, magenta, cyan, white)
//   - 8-15: bright ANSI colors
//   - 16-231: 216-color cube (6×6×6)
//   - 232-255: grayscale (24 steps)
//
// Example:
//
//	logger.Info("user created",
//	    gooeyslog.Colored(gooeyslog.ColorGreen, slog.String("name", user.Name)),
//	    gooeyslog.Colored(gooeyslog.ColorCyan, slog.Int("id", user.ID)),
//	)
func Colored(color uint8, attr slog.Attr) slog.Attr {
	attr.Value = slog.AnyValue(coloredValue{attr.Value, color})
	return attr
}

// Red returns an attribute colored red.
func Red(attr slog.Attr) slog.Attr {
	return Colored(ColorBrightRed, attr)
}

// Green returns an attribute colored green.
func Green(attr slog.Attr) slog.Attr {
	return Colored(ColorBrightGreen, attr)
}

// Yellow returns an attribute colored yellow.
func Yellow(attr slog.Attr) slog.Attr {
	return Colored(ColorBrightYellow, attr)
}

// Blue returns an attribute colored blue.
func Blue(attr slog.Attr) slog.Attr {
	return Colored(ColorBrightBlue, attr)
}

// Cyan returns an attribute colored cyan.
func Cyan(attr slog.Attr) slog.Attr {
	return Colored(ColorBrightCyan, attr)
}

// Magenta returns an attribute colored magenta.
func Magenta(attr slog.Attr) slog.Attr {
	return Colored(ColorBrightMagenta, attr)
}
