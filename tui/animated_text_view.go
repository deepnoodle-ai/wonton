package tui

import "github.com/deepnoodle-ai/wonton/runewidth"

// forEachAnimatedCluster calls fn for each grapheme cluster of text with the
// cluster's index, the total cluster count, the cluster's column offset from
// the start of text, and its display width. Animated widgets style text one
// cluster at a time and must advance by display width, not by rune, so that
// wide and combined characters stay intact. Iteration stops when fn returns
// false.
func forEachAnimatedCluster(text string, fn func(index, total, col int, cluster string, width int) bool) {
	total := 0
	for range runewidth.Graphemes(text) {
		total++
	}
	index, col := 0, 0
	for cluster, w := range runewidth.Graphemes(text) {
		if !fn(index, total, col, cluster, w) {
			return
		}
		index++
		col += w
	}
}

// AnimatedTextView displays text with per-character animation (declarative view)
type AnimatedTextView struct {
	text      string
	animation TextAnimation
	style     Style // fallback style if no animation
	width     int
}

// Width sets a fixed width for the animated text.
func (a *AnimatedTextView) Width(w int) *AnimatedTextView {
	a.width = w
	return a
}

// Style sets the fallback style (used when animation is nil).
func (a *AnimatedTextView) Style(s Style) *AnimatedTextView {
	a.style = s
	return a
}

func (a *AnimatedTextView) size(maxWidth, maxHeight int) (int, int) {
	w, _ := MeasureText(a.text)
	if a.width > 0 {
		w = a.width
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (a *AnimatedTextView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	frameCount := ctx.Frame()

	forEachAnimatedCluster(a.text, func(i, total, col int, cluster string, cw int) bool {
		if col+cw > w {
			return false
		}
		style := a.style
		if a.animation != nil {
			style = a.animation.GetStyle(frameCount, i, total)
		}
		ctx.PrintTruncated(col, 0, cluster, style)
		return true
	})
}
