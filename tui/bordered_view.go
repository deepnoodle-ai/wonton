package tui

import "image"

// borderedView wraps a view with an optional border
type borderedView struct {
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
func Bordered(inner View) *borderedView {
	return &borderedView{
		inner:       inner,
		borderStyle: NewStyle(),
		titleStyle:  NewStyle(),
	}
}

// Border sets the border style for the frame.
func (f *borderedView) Border(style *BorderStyle) *borderedView {
	f.border = style
	return f
}

// Title sets the title shown in the border.
func (f *borderedView) Title(title string) *borderedView {
	f.title = title
	return f
}

// TitleStyle sets the style for the title text.
func (f *borderedView) TitleStyle(s Style) *borderedView {
	f.titleStyle = s
	return f
}

// BorderFg sets the border foreground color.
func (f *borderedView) BorderFg(c Color) *borderedView {
	f.borderStyle = f.borderStyle.WithForeground(c)
	return f
}

// FocusID sets the focus ID to watch for styling changes.
// When the element with this ID is focused, focus styles will be applied.
func (f *borderedView) FocusID(id string) *borderedView {
	f.focusID = id
	return f
}

// FocusBorderFg sets the border color when the watched element is focused.
func (f *borderedView) FocusBorderFg(c Color) *borderedView {
	f.focusBorderFg = c
	f.hasFocusBorder = true
	return f
}

// FocusTitleStyle sets the title style when the watched element is focused.
func (f *borderedView) FocusTitleStyle(s Style) *borderedView {
	f.focusTitleStyle = &s
	return f
}

// flex implements the Flexible interface by delegating to the inner view.
// This allows bordered views containing flexible content (like Fill) to
// participate in flex layout distribution.
func (f *borderedView) flex() int {
	if flex, ok := f.inner.(Flexible); ok {
		return flex.flex()
	}
	return 0
}

func (f *borderedView) size(maxWidth, maxHeight int) (int, int) {
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

func (f *borderedView) render(ctx *RenderContext) {
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

	// Title in top border
	if f.title != "" && w > 4 {
		titleW, _ := MeasureText(f.title)
		maxTitleW := w - 4
		if titleW > maxTitleW {
			titleW = maxTitleW
		}
		titleX := 2
		ctx.PrintTruncated(titleX, 0, f.title[:min(len(f.title), maxTitleW)], titleStyle)
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
