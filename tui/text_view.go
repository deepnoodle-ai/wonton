package tui

import (
	"fmt"
)

// TextView displays styled text
type TextView struct {
	content    string
	style      Style
	wrap       bool
	align      Alignment
	fillBg     bool
	flexFactor int
}

// Text creates a text view with optional Printf-style formatting.
// This is the fundamental component for displaying text in TUI applications.
//
// The text view supports:
//   - Styling methods (Fg, Bg, Bold, Italic, etc.)
//   - Semantic styling (Success, Error, Warning, Info, Muted, Hint)
//   - Animation effects via Animate (Rainbow, Pulse, Typewriter, Glitch, etc.)
//
// Example:
//
//	Text("Hello, %s!", userName).Fg(ColorGreen).Bold()
//	Text("Error: %s", err).Error()
//	Text("Loading...").Animate(Pulse(tui.NewRGB(0, 255, 0), 10))
func Text(format string, args ...any) *TextView {
	content := format
	if len(args) > 0 {
		content = fmt.Sprintf(format, args...)
	}
	return &TextView{
		content: content,
		style:   NewStyle(),
	}
}

// Fg sets the foreground color.
func (t *TextView) Fg(c Color) *TextView {
	t.style = t.style.WithForeground(c)
	return t
}

// FgRGB sets the foreground color using RGB values.
func (t *TextView) FgRGB(r, g, b uint8) *TextView {
	t.style = t.style.WithFgRGB(RGB{R: r, G: g, B: b})
	return t
}

// Bg sets the background color.
func (t *TextView) Bg(c Color) *TextView {
	t.style = t.style.WithBackground(c)
	return t
}

// BgRGB sets the background color using RGB values.
func (t *TextView) BgRGB(r, g, b uint8) *TextView {
	t.style = t.style.WithBgRGB(RGB{R: r, G: g, B: b})
	return t
}

// Bold enables bold text.
func (t *TextView) Bold() *TextView {
	t.style = t.style.WithBold()
	return t
}

// Italic enables italic text.
func (t *TextView) Italic() *TextView {
	t.style = t.style.WithItalic()
	return t
}

// Underline enables underlined text.
func (t *TextView) Underline() *TextView {
	t.style = t.style.WithUnderline()
	return t
}

// Strikethrough enables strikethrough text.
func (t *TextView) Strikethrough() *TextView {
	t.style = t.style.WithStrikethrough()
	return t
}

// Dim enables dim/faint text.
func (t *TextView) Dim() *TextView {
	t.style = t.style.WithDim()
	return t
}

// Reverse enables reverse video (swap fg/bg).
func (t *TextView) Reverse() *TextView {
	t.style = t.style.WithReverse()
	return t
}

// Blink enables blinking text.
func (t *TextView) Blink() *TextView {
	t.style = t.style.WithBlink()
	return t
}

// Style sets the complete style.
func (t *TextView) Style(s Style) *TextView {
	t.style = s
	return t
}

// Semantic styling methods for common text patterns

// Success returns text styled for success messages (green, bold).
func (t *TextView) Success() *TextView {
	t.style = t.style.WithForeground(ColorGreen).WithBold()
	return t
}

// Error returns text styled for error messages (red, bold).
func (t *TextView) Error() *TextView {
	t.style = t.style.WithForeground(ColorRed).WithBold()
	return t
}

// Warning returns text styled for warning messages (yellow, bold).
func (t *TextView) Warning() *TextView {
	t.style = t.style.WithForeground(ColorYellow).WithBold()
	return t
}

// Info returns text styled for informational messages (cyan).
func (t *TextView) Info() *TextView {
	t.style = t.style.WithForeground(ColorCyan)
	return t
}

// Muted returns text styled for secondary/muted content (dim, gray).
func (t *TextView) Muted() *TextView {
	t.style = t.style.WithForeground(ColorBrightBlack).WithDim()
	return t
}

// Hint returns text styled for hints and helper text (dim, italic).
func (t *TextView) Hint() *TextView {
	t.style = t.style.WithForeground(ColorBrightBlack).WithDim().WithItalic()
	return t
}

// Wrap enables text wrapping to fit within the available width.
// By default, text is truncated instead of wrapped.
func (t *TextView) Wrap() *TextView {
	t.wrap = true
	return t
}

// Truncate disables text wrapping, causing text to be truncated at the edge.
// This is the default behavior; use Wrap() to enable wrapping instead.
func (t *TextView) Truncate() *TextView {
	t.wrap = false
	return t
}

// Align sets the text alignment (left, center, or right).
func (t *TextView) Align(a Alignment) *TextView {
	t.align = a
	return t
}

// Center is a shorthand for Align(AlignCenter).
func (t *TextView) Center() *TextView {
	t.align = AlignCenter
	return t
}

// Right is a shorthand for Align(AlignRight).
func (t *TextView) Right() *TextView {
	t.align = AlignRight
	return t
}

// FillBg fills the entire background with the background color.
func (t *TextView) FillBg() *TextView {
	t.fillBg = true
	return t
}

// Flex sets the flex factor for this view in flex layouts.
// A higher value means this view gets more of the available space.
// Set to 0 to make the view non-flexible (fixed size).
func (t *TextView) Flex(factor int) *TextView {
	t.flexFactor = factor
	return t
}

// flex implements the Flexible interface.
func (t *TextView) flex() int {
	return t.flexFactor
}

// Animate applies a TextAnimation to the text, returning an animated text view.
// This is the preferred way to add animations to text as it allows any TextAnimation
// implementation to be used, making it fully extensible.
//
// Example:
//
//	Text("Rainbow text").Animate(Rainbow(3))
//	Text("Pulsing").Animate(Pulse(tui.NewRGB(0, 255, 0), 10))
//	Text("Custom").Animate(&MyCustomAnimation{})
//
// Animation constructors provide chainable configuration:
//
//	Text("Reversed rainbow").Animate(Rainbow(3).Reverse())
//	Text("Bright pulse").Animate(Pulse(green, 10).Brightness(0.5, 1.0))
func (t *TextView) Animate(animation TextAnimation) *AnimatedTextView {
	return &AnimatedTextView{
		text:      t.content,
		animation: animation,
		style:     t.style,
	}
}

func (t *TextView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Fill background if requested
	if t.fillBg {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				ctx.SetCell(x, y, ' ', t.style)
			}
		}
	}

	// Process text
	displayText := t.content
	if t.wrap && width > 0 {
		displayText = WrapText(displayText, width)
	}

	// Align text if alignment is set
	if t.align != AlignLeft && width > 0 {
		displayText = AlignText(displayText, width, t.align)
	}

	// Render
	if t.wrap {
		lines := splitLinesSimple(displayText)
		for y, line := range lines {
			if y >= height {
				break
			}
			ctx.PrintStyled(0, y, line, t.style)
		}
	} else {
		ctx.PrintTruncated(0, 0, displayText, t.style)
	}
}

func (t *TextView) size(maxWidth, maxHeight int) (int, int) {
	w, h := MeasureText(t.content)

	// For wrapped text, expand to fill available width
	if t.wrap && maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}

	// Calculate height based on wrapped lines
	if t.wrap && maxWidth > 0 {
		wrapped := WrapText(t.content, maxWidth)
		lines := splitLinesSimple(wrapped)
		h = len(lines)
	}

	// Apply constraints
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

// splitLinesSimple splits text on newlines (used by TextView)
func splitLinesSimple(s string) []string {
	if s == "" {
		return []string{}
	}

	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	} else if start == len(s) && len(s) > 0 && s[len(s)-1] == '\n' {
		// Trailing newline creates empty last line
		lines = append(lines, "")
	}
	return lines
}
