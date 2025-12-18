package terminal

import (
	"fmt"
	"strings"

	"github.com/deepnoodle-ai/wonton/color"
)

// Re-export color types for backward compatibility
type Color = color.Color
type RGB = color.RGB

// Re-export color constants for backward compatibility
const (
	ColorDefault       = color.NoColor
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

// Style represents text styling attributes including colors, text attributes, and hyperlinks.
//
// Styles are immutable - all With* methods return a new Style with the requested changes.
// This makes it safe to use a base style and derive variations without affecting the original.
//
// # Colors
//
// Colors can be specified as basic ANSI colors or full RGB:
//
//	// Basic ANSI colors (16 colors)
//	style := terminal.NewStyle().WithForeground(terminal.ColorRed)
//
//	// RGB colors (24-bit true color)
//	rgb := terminal.NewRGB(255, 100, 50)
//	style = style.WithFgRGB(rgb)
//
// # Text Attributes
//
// Multiple attributes can be combined:
//
//	style := terminal.NewStyle().
//	    WithBold().
//	    WithItalic().
//	    WithUnderline()
//
// # Hyperlinks
//
// Styles can include OSC 8 hyperlinks for terminals that support them:
//
//	style := terminal.NewStyle().
//	    WithForeground(terminal.ColorBlue).
//	    WithUnderline().
//	    WithURL("https://example.com")
//
// # Applying Styles
//
// Styles can be applied to text or used with rendering methods:
//
//	// Apply to a string (wraps with ANSI codes)
//	styledText := style.Apply("Hello")
//	fmt.Print(styledText)
//
//	// Use with frame rendering
//	frame.PrintStyled(x, y, "Hello", style)
type Style struct {
	Foreground    Color  // Basic ANSI foreground color (or ColorDefault)
	Background    Color  // Basic ANSI background color (or ColorDefault)
	FgRGB         *RGB   // RGB override for foreground (nil = use Foreground)
	BgRGB         *RGB   // RGB override for background (nil = use Background)
	Bold          bool   // Bold or increased intensity
	Italic        bool   // Italic text
	Underline     bool   // Underlined text
	Strikethrough bool   // Strikethrough text
	Blink         bool   // Blinking text (rarely supported)
	Reverse       bool   // Reverse video (swap foreground and background)
	Hidden        bool   // Hidden text (rarely used, for passwords)
	Dim           bool   // Dim or decreased intensity
	URL           string // OSC 8 hyperlink URL (empty = no hyperlink)
}

// NewStyle creates a new Style with default values (no colors, no attributes).
// This is the recommended starting point for building custom styles.
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

// String returns the ANSI escape sequence representation of this style.
// The sequence includes a reset (ESC[0m) followed by all active attributes and colors.
// This can be printed to the terminal to activate the style.
//
// Note: This does not include hyperlink (OSC 8) escape codes. For hyperlinks,
// use the Hyperlink type or Style.WithURL with frame rendering.
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

// Apply applies the style to the given text by wrapping it with ANSI escape codes.
// The text is prefixed with the style's ANSI sequence and suffixed with a reset.
//
// Example:
//
//	style := terminal.NewStyle().WithBold().WithForeground(terminal.ColorRed)
//	styled := style.Apply("Important")
//	fmt.Println(styled) // Prints "Important" in bold red
//
// If the style is empty (no attributes set), the text is returned unchanged.
func (s Style) Apply(text string) string {
	if s.IsEmpty() {
		return text
	}
	return s.String() + text + "\033[0m"
}

// IsEmpty returns true if the style has no attributes, colors, or URL set.
// An empty style produces no visual changes when applied.
func (s Style) IsEmpty() bool {
	return s == NewStyle()
}

// Merge combines two styles, with the other style's non-default values taking precedence.
// This is useful for applying a base style and then overriding specific attributes.
//
// Example:
//
//	base := terminal.NewStyle().WithBold().WithForeground(terminal.ColorBlue)
//	highlight := terminal.NewStyle().WithBackground(terminal.ColorYellow)
//	combined := base.Merge(highlight) // Bold blue text on yellow background
//
// Attributes from 'other' override attributes from 's', but only if they are non-default.
// This means Merge never removes attributes, only adds or replaces them.
func (s Style) Merge(other Style) Style {
	result := s
	if other.Foreground != ColorDefault {
		result.Foreground = other.Foreground
	}
	if other.Background != ColorDefault {
		result.Background = other.Background
	}
	if other.FgRGB != nil {
		rgb := *other.FgRGB
		result.FgRGB = &rgb
	}
	if other.BgRGB != nil {
		rgb := *other.BgRGB
		result.BgRGB = &rgb
	}
	if other.Bold {
		result.Bold = true
	}
	if other.Italic {
		result.Italic = true
	}
	if other.Underline {
		result.Underline = true
	}
	if other.Strikethrough {
		result.Strikethrough = true
	}
	if other.Blink {
		result.Blink = true
	}
	if other.Reverse {
		result.Reverse = true
	}
	if other.Hidden {
		result.Hidden = true
	}
	if other.Dim {
		result.Dim = true
	}
	if other.URL != "" {
		result.URL = other.URL
	}
	return result
}
