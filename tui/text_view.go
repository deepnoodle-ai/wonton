package tui

import (
	"fmt"
	"image"
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

func (t *textView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}
	// Create a subframe for the text bounds
	subFrame := frame.SubFrame(bounds)
	// Print using truncation (no wrap) to stay within bounds
	subFrame.PrintTruncated(0, 0, t.content, t.style)
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
