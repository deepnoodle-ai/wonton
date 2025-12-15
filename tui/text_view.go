package tui

import (
	"fmt"
)

// textView displays styled text
type textView struct {
	content string
	style   Style
}

// Text creates a text view with optional Printf-style formatting.
func Text(format string, args ...any) *textView {
	content := format
	if len(args) > 0 {
		content = fmt.Sprintf(format, args...)
	}
	return &textView{
		content: content,
		style:   NewStyle(),
	}
}

// Fg sets the foreground color.
func (t *textView) Fg(c Color) *textView {
	t.style = t.style.WithForeground(c)
	return t
}

// FgRGB sets the foreground color using RGB values.
func (t *textView) FgRGB(r, g, b uint8) *textView {
	t.style = t.style.WithFgRGB(RGB{R: r, G: g, B: b})
	return t
}

// Bg sets the background color.
func (t *textView) Bg(c Color) *textView {
	t.style = t.style.WithBackground(c)
	return t
}

// BgRGB sets the background color using RGB values.
func (t *textView) BgRGB(r, g, b uint8) *textView {
	t.style = t.style.WithBgRGB(RGB{R: r, G: g, B: b})
	return t
}

// Bold enables bold text.
func (t *textView) Bold() *textView {
	t.style = t.style.WithBold()
	return t
}

// Italic enables italic text.
func (t *textView) Italic() *textView {
	t.style = t.style.WithItalic()
	return t
}

// Underline enables underlined text.
func (t *textView) Underline() *textView {
	t.style = t.style.WithUnderline()
	return t
}

// Strikethrough enables strikethrough text.
func (t *textView) Strikethrough() *textView {
	t.style = t.style.WithStrikethrough()
	return t
}

// Dim enables dim/faint text.
func (t *textView) Dim() *textView {
	t.style = t.style.WithDim()
	return t
}

// Reverse enables reverse video (swap fg/bg).
func (t *textView) Reverse() *textView {
	t.style = t.style.WithReverse()
	return t
}

// Blink enables blinking text.
func (t *textView) Blink() *textView {
	t.style = t.style.WithBlink()
	return t
}

// Style sets the complete style.
func (t *textView) Style(s Style) *textView {
	t.style = s
	return t
}

// Semantic styling methods for common text patterns

// Success returns text styled for success messages (green, bold).
func (t *textView) Success() *textView {
	t.style = t.style.WithForeground(ColorGreen).WithBold()
	return t
}

// Error returns text styled for error messages (red, bold).
func (t *textView) Error() *textView {
	t.style = t.style.WithForeground(ColorRed).WithBold()
	return t
}

// Warning returns text styled for warning messages (yellow, bold).
func (t *textView) Warning() *textView {
	t.style = t.style.WithForeground(ColorYellow).WithBold()
	return t
}

// Info returns text styled for informational messages (cyan).
func (t *textView) Info() *textView {
	t.style = t.style.WithForeground(ColorCyan)
	return t
}

// Muted returns text styled for secondary/muted content (dim, gray).
func (t *textView) Muted() *textView {
	t.style = t.style.WithForeground(ColorBrightBlack).WithDim()
	return t
}

// Hint returns text styled for hints and helper text (dim, italic).
func (t *textView) Hint() *textView {
	t.style = t.style.WithForeground(ColorBrightBlack).WithDim().WithItalic()
	return t
}

func (t *textView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}
	ctx.PrintTruncated(0, 0, t.content, t.style)
}

// Rainbow returns an animated text view with a rainbow color effect.
// Speed controls how fast the rainbow moves (lower = faster).
func (t *textView) Rainbow(speed int) *animatedTextView {
	return &animatedTextView{
		text: t.content,
		animation: &RainbowAnimation{
			Speed:  speed,
			Length: len([]rune(t.content)),
		},
		style: t.style,
	}
}

// RainbowReverse returns an animated text view with a reverse rainbow effect.
func (t *textView) RainbowReverse(speed int) *animatedTextView {
	return &animatedTextView{
		text: t.content,
		animation: &RainbowAnimation{
			Speed:    speed,
			Length:   len([]rune(t.content)),
			Reversed: true,
		},
		style: t.style,
	}
}

// Pulse returns an animated text view with a pulsing brightness effect.
// Color is the base color, speed controls the pulse rate (lower = faster).
func (t *textView) Pulse(color RGB, speed int) *animatedTextView {
	return &animatedTextView{
		text: t.content,
		animation: &PulseAnimation{
			Speed: speed,
			Color: color,
		},
		style: t.style,
	}
}

// Sparkle returns an animated text view with a twinkling star-like effect.
// Speed controls animation timing (lower = faster).
// BaseColor is the default color, sparkColor is the bright sparkle color.
func (t *textView) Sparkle(speed int, baseColor, sparkColor RGB) *animatedTextView {
	return &animatedTextView{
		text: t.content,
		animation: &SparkleAnimation{
			Speed:      speed,
			BaseColor:  baseColor,
			SparkColor: sparkColor,
			Density:    3,
		},
		style: t.style,
	}
}

// Typewriter returns an animated text view that reveals characters one by one.
// Speed controls how fast characters appear (lower = faster).
// TextColor is the revealed text color, cursorColor is the blinking cursor.
func (t *textView) Typewriter(speed int, textColor, cursorColor RGB) *animatedTextView {
	return &animatedTextView{
		text: t.content,
		animation: &TypewriterAnimation{
			Speed:       speed,
			TextColor:   textColor,
			CursorColor: cursorColor,
			Loop:        true,
			HoldFrames:  90,
		},
		style: t.style,
	}
}

// Glitch returns an animated text view with a cyberpunk-style glitch effect.
// Speed controls glitch timing (lower = faster).
// BaseColor is the normal color, glitchColor is the color during glitches.
func (t *textView) Glitch(speed int, baseColor, glitchColor RGB) *animatedTextView {
	return &animatedTextView{
		text: t.content,
		animation: &GlitchAnimation{
			Speed:       speed,
			BaseColor:   baseColor,
			GlitchColor: glitchColor,
			Intensity:   3,
		},
		style: t.style,
	}
}

// Slide returns an animated text view with a highlight sliding left to right.
// Speed controls how fast the highlight moves (lower = faster).
// BaseColor is the default text color, highlightColor is the sliding highlight.
func (t *textView) Slide(speed int, baseColor, highlightColor RGB) *animatedTextView {
	return &animatedTextView{
		text: t.content,
		animation: &SlideAnimation{
			Speed:          speed,
			BaseColor:      baseColor,
			HighlightColor: highlightColor,
		},
		style: t.style,
	}
}

// SlideReverse returns an animated text view with a highlight sliding right to left.
// Speed controls how fast the highlight moves (lower = faster).
// BaseColor is the default text color, highlightColor is the sliding highlight.
func (t *textView) SlideReverse(speed int, baseColor, highlightColor RGB) *animatedTextView {
	return &animatedTextView{
		text: t.content,
		animation: &SlideAnimation{
			Speed:          speed,
			BaseColor:      baseColor,
			HighlightColor: highlightColor,
			Reverse:        true,
		},
		style: t.style,
	}
}

// Wave returns an animated text view with a wave color effect.
// Speed controls how fast the wave moves (lower = faster).
// Colors are the colors to cycle through (defaults to magenta/green/purple).
func (t *textView) Wave(speed int, colors ...RGB) *animatedTextView {
	if len(colors) == 0 {
		colors = []RGB{
			NewRGB(255, 0, 100),
			NewRGB(0, 255, 100),
			NewRGB(100, 0, 255),
		}
	}
	return &animatedTextView{
		text: t.content,
		animation: &WaveAnimation{
			Speed:  speed,
			Colors: colors,
		},
		style: t.style,
	}
}

func (t *textView) size(maxWidth, maxHeight int) (int, int) {
	w, h := MeasureText(t.content)
	// Apply constraints
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}
