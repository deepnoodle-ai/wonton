package terminal

import (
	"fmt"
	"strings"

	"github.com/deepnoodle-ai/gooey/color"
)

// Re-export color types for backward compatibility
type Color = color.Color
type RGB = color.RGB

// Re-export color constants for backward compatibility
const (
	ColorDefault       = color.Default
	ColorBlack         = color.Black
	ColorRed           = color.Red
	ColorGreen         = color.Green
	ColorYellow        = color.Yellow
	ColorBlue          = color.Blue
	ColorMagenta       = color.Magenta
	ColorCyan          = color.Cyan
	ColorWhite         = color.White
	ColorBrightBlack   = color.BrightBlack
	ColorBrightRed     = color.BrightRed
	ColorBrightGreen   = color.BrightGreen
	ColorBrightYellow  = color.BrightYellow
	ColorBrightBlue    = color.BrightBlue
	ColorBrightMagenta = color.BrightMagenta
	ColorBrightCyan    = color.BrightCyan
	ColorBrightWhite   = color.BrightWhite
)

// Re-export color functions for backward compatibility
var (
	NewRGB          = color.NewRGB
	Gradient        = color.Gradient
	RainbowGradient = color.RainbowGradient
	SmoothRainbow   = color.SmoothRainbow
	MultiGradient   = color.MultiGradient
)

// Style represents text styling attributes
type Style struct {
	Foreground    Color
	Background    Color
	FgRGB         *RGB // RGB override for foreground
	BgRGB         *RGB // RGB override for background
	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	Blink         bool
	Reverse       bool
	Hidden        bool
	Dim           bool
	URL           string // OSC 8 hyperlink URL (empty string = no hyperlink)
}

// NewStyle creates a new style with default values
func NewStyle() Style {
	return Style{
		Foreground: ColorDefault,
		Background: ColorDefault,
	}
}

// WithForeground sets the foreground color
func (s Style) WithForeground(c Color) Style {
	s.Foreground = c
	s.FgRGB = nil // Clear RGB override
	return s
}

// WithBackground sets the background color
func (s Style) WithBackground(c Color) Style {
	s.Background = c
	s.BgRGB = nil // Clear RGB override
	return s
}

// WithFgRGB sets the foreground RGB color
func (s Style) WithFgRGB(rgb RGB) Style {
	s.FgRGB = &rgb
	return s
}

// WithBgRGB sets the background RGB color
func (s Style) WithBgRGB(rgb RGB) Style {
	s.BgRGB = &rgb
	return s
}

// WithBold enables bold text
func (s Style) WithBold() Style {
	s.Bold = true
	return s
}

// WithItalic enables italic text
func (s Style) WithItalic() Style {
	s.Italic = true
	return s
}

// WithUnderline enables underlined text
func (s Style) WithUnderline() Style {
	s.Underline = true
	return s
}

// WithStrikethrough enables strikethrough text
func (s Style) WithStrikethrough() Style {
	s.Strikethrough = true
	return s
}

// WithBlink enables blinking text
func (s Style) WithBlink() Style {
	s.Blink = true
	return s
}

// WithReverse enables reverse video
func (s Style) WithReverse() Style {
	s.Reverse = true
	return s
}

// WithDim enables dim/faint text
func (s Style) WithDim() Style {
	s.Dim = true
	return s
}

// WithURL sets a hyperlink URL for this style (OSC 8)
func (s Style) WithURL(url string) Style {
	s.URL = url
	return s
}

// String returns the ANSI escape sequence for this style
func (s Style) String() string {
	var codes []string

	// Reset
	codes = append(codes, "0")

	// Attributes
	if s.Bold {
		codes = append(codes, "1")
	}
	if s.Dim {
		codes = append(codes, "2")
	}
	if s.Italic {
		codes = append(codes, "3")
	}
	if s.Underline {
		codes = append(codes, "4")
	}
	if s.Blink {
		codes = append(codes, "5")
	}
	if s.Reverse {
		codes = append(codes, "7")
	}
	if s.Hidden {
		codes = append(codes, "8")
	}
	if s.Strikethrough {
		codes = append(codes, "9")
	}

	// Foreground color
	if s.FgRGB != nil {
		codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", s.FgRGB.R, s.FgRGB.G, s.FgRGB.B))
	} else if s.Foreground != ColorDefault {
		codes = append(codes, s.Foreground.ForegroundCode())
	}

	// Background color
	if s.BgRGB != nil {
		codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", s.BgRGB.R, s.BgRGB.G, s.BgRGB.B))
	} else if s.Background != ColorDefault {
		codes = append(codes, s.Background.BackgroundCode())
	}

	return fmt.Sprintf("\033[%sm", strings.Join(codes, ";"))
}

// Apply applies the style to the given text
func (s Style) Apply(text string) string {
	if s.IsEmpty() {
		return text
	}
	return s.String() + text + "\033[0m"
}

// IsEmpty checks if the style has no attributes set
func (s Style) IsEmpty() bool {
	return s == NewStyle()
}
