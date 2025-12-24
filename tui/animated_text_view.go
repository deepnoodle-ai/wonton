package tui

// animatedTextView displays text with per-character animation (declarative view)
type animatedTextView struct {
	text      string
	animation TextAnimation
	style     Style // fallback style if no animation
	width     int
}

// Width sets a fixed width for the animated text.
func (a *animatedTextView) Width(w int) *animatedTextView {
	a.width = w
	return a
}

// Style sets the fallback style (used when animation is nil).
func (a *animatedTextView) Style(s Style) *animatedTextView {
	a.style = s
	return a
}

func (a *animatedTextView) size(maxWidth, maxHeight int) (int, int) {
	w, _ := MeasureText(a.text)
	if a.width > 0 {
		w = a.width
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (a *animatedTextView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	runes := []rune(a.text)
	totalChars := len(runes)
	frameCount := ctx.Frame()

	for i, r := range runes {
		if i >= w {
			break
		}
		var style Style
		if a.animation != nil {
			style = a.animation.GetStyle(frameCount, i, totalChars)
		} else {
			style = a.style
		}
		ctx.SetCell(i, 0, r, style)
	}
}
