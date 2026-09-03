package tui

import (
	"image"

	"github.com/deepnoodle-ai/wonton/runewidth"
)

// BorderedView wraps a view with an optional border
type BorderedView struct {
	inner       View
	border      *BorderStyle
	title       string
	titleStyle  Style
	borderStyle Style

	// Focus-aware styling
	focusID         string // Watch this focus ID for styling changes
	focusBorderFg   Color  // Border color when focused
	hasFocusBorder  bool   // true if focusBorderFg was set
	focusTitleStyle *Style // Title style when focused
}

// Bordered wraps a view with a border and optional title.
// The border consumes 2 cells of width and height (1 on each side).
//
// The border defaults to SingleBorder. Pass Border(nil) for a wrapper that
// reserves no cells and draws nothing.
//
// Use the builder pattern to customize the border:
//
//	Bordered(content).
//	    Border(&RoundedBorder).
//	    Title("Box Title").
//	    BorderFg(ColorCyan)
//
// Focus-aware borders change color when a watched element is focused:
//
//	Bordered(InputField(&app.input)).
//	    FocusID("my-input").
//	    FocusBorderFg(ColorGreen)
func Bordered(inner View) *BorderedView {
	return &BorderedView{
		inner:       inner,
		border:      &SingleBorder,
		borderStyle: NewStyle(),
		titleStyle:  NewStyle(),
	}
}

// Border sets the border style for the frame. A nil style draws no border and
// reserves no cells for one.
func (f *BorderedView) Border(style *BorderStyle) *BorderedView {
	f.border = style
	return f
}

// Title sets the title shown in the border.
func (f *BorderedView) Title(title string) *BorderedView {
	f.title = title
	return f
}

// TitleStyle sets the style for the title text.
func (f *BorderedView) TitleStyle(s Style) *BorderedView {
	f.titleStyle = s
	return f
}

// BorderFg sets the border foreground color.
func (f *BorderedView) BorderFg(c Color) *BorderedView {
	f.borderStyle = f.borderStyle.WithForeground(c)
	return f
}

// FocusID sets the focus ID to watch for styling changes.
// When the element with this ID is focused, focus styles will be applied.
func (f *BorderedView) FocusID(id string) *BorderedView {
	f.focusID = id
	return f
}

// FocusBorderFg sets the border color when the watched element is focused.
func (f *BorderedView) FocusBorderFg(c Color) *BorderedView {
	f.focusBorderFg = c
	f.hasFocusBorder = true
	return f
}

// FocusTitleStyle sets the title style when the watched element is focused.
func (f *BorderedView) FocusTitleStyle(s Style) *BorderedView {
	f.focusTitleStyle = &s
	return f
}

// flex implements the Flexible interface by delegating to the inner view.
// This allows bordered views containing flexible content (like Fill) to
// participate in flex layout distribution.
func (f *BorderedView) flex() int {
	if flex, ok := f.inner.(Flexible); ok {
		return flex.flex()
	}
	return 0
}

func (f *BorderedView) size(maxWidth, maxHeight int) (int, int) {
	borderSize := 0
	if f.border != nil {
		borderSize = 2 // 1 char on each side
	}

	innerMaxW := maxWidth
	if maxWidth > 0 {
		innerMaxW = maxWidth - borderSize
		if innerMaxW < 0 {
			innerMaxW = 0
		}
	}
	innerMaxH := maxHeight
	if maxHeight > 0 {
		innerMaxH = maxHeight - borderSize
		if innerMaxH < 0 {
			innerMaxH = 0
		}
	}

	innerW, innerH := f.inner.size(innerMaxW, innerMaxH)
	return innerW + borderSize, innerH + borderSize
}

func (f *BorderedView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	if f.border == nil {
		// No border, just render inner
		f.inner.render(ctx)
		return
	}

	// Determine if the watched element is focused
	fm := ctx.FocusManager()
	isFocused := f.focusID != "" && fm != nil && fm.GetFocusedID() == f.focusID

	// Choose border style based on focus
	borderStyle := f.borderStyle
	if isFocused && f.hasFocusBorder {
		borderStyle = NewStyle().WithForeground(f.focusBorderFg)
	}

	// Choose title style based on focus
	titleStyle := f.titleStyle
	if isFocused && f.focusTitleStyle != nil {
		titleStyle = *f.focusTitleStyle
	}

	// Draw border
	// Top border
	ctx.PrintTruncated(0, 0, f.border.TopLeft, borderStyle)
	for x := 1; x < w-1; x++ {
		ctx.PrintTruncated(x, 0, f.border.Horizontal, borderStyle)
	}
	if w > 1 {
		ctx.PrintTruncated(w-1, 0, f.border.TopRight, borderStyle)
	}

	// Title in top border. Truncate by display width, not by bytes: slicing
	// bytes splits a multi-byte character into garbage.
	if f.title != "" && w > 4 {
		ctx.PrintTruncated(2, 0, runewidth.Truncate(f.title, w-4, "…"), titleStyle)
	}

	// Side borders
	for y := 1; y < h-1; y++ {
		ctx.PrintTruncated(0, y, f.border.Vertical, borderStyle)
		if w > 1 {
			ctx.PrintTruncated(w-1, y, f.border.Vertical, borderStyle)
		}
	}

	// Bottom border
	if h > 1 {
		ctx.PrintTruncated(0, h-1, f.border.BottomLeft, borderStyle)
		for x := 1; x < w-1; x++ {
			ctx.PrintTruncated(x, h-1, f.border.Horizontal, borderStyle)
		}
		if w > 1 {
			ctx.PrintTruncated(w-1, h-1, f.border.BottomRight, borderStyle)
		}
	}

	// Inner content (1 cell padding for border)
	innerBounds := image.Rect(1, 1, w-1, h-1)
	if innerBounds.Dx() > 0 && innerBounds.Dy() > 0 {
		innerCtx := ctx.SubContext(innerBounds)
		f.inner.render(innerCtx)
	}
}
