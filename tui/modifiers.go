package tui

import "image"

// paddingView wraps a view with padding
type paddingView struct {
	inner                    View
	top, right, bottom, left int
}

// Padding wraps a view with equal padding on all sides.
// Padding adds empty space around content, measured in character cells.
//
// Example:
//
//	Padding(2, Text("Content"))  // 2 cells of padding on all sides
func Padding(n int, inner View) View {
	return &paddingView{
		inner:  inner,
		top:    n,
		right:  n,
		bottom: n,
		left:   n,
	}
}

// PaddingHV wraps a view with horizontal and vertical padding.
// The first parameter is horizontal (left and right), the second is vertical (top and bottom).
//
// Example:
//
//	PaddingHV(4, 1, Text("Content"))  // 4 cells horizontal, 1 cell vertical
func PaddingHV(h, v int, inner View) View {
	return &paddingView{
		inner:  inner,
		top:    v,
		right:  h,
		bottom: v,
		left:   h,
	}
}

// PaddingLTRB wraps a view with specific padding on each side.
// Parameters are in CSS order: left, top, right, bottom.
//
// Example:
//
//	PaddingLTRB(1, 2, 3, 4, Text("Content"))  // Different padding on each side
func PaddingLTRB(left, top, right, bottom int, inner View) View {
	return &paddingView{
		inner:  inner,
		top:    top,
		right:  right,
		bottom: bottom,
		left:   left,
	}
}

func (p *paddingView) size(maxWidth, maxHeight int) (int, int) {
	paddingW := p.left + p.right
	paddingH := p.top + p.bottom

	// Reduce constraints by padding
	innerMaxW := maxWidth
	if maxWidth > 0 {
		innerMaxW = maxWidth - paddingW
		if innerMaxW < 0 {
			innerMaxW = 0
		}
	}
	innerMaxH := maxHeight
	if maxHeight > 0 {
		innerMaxH = maxHeight - paddingH
		if innerMaxH < 0 {
			innerMaxH = 0
		}
	}

	innerW, innerH := p.inner.size(innerMaxW, innerMaxH)
	return innerW + paddingW, innerH + paddingH
}

func (p *paddingView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Calculate inner bounds
	innerBounds := image.Rect(
		p.left,
		p.top,
		width-p.right,
		height-p.bottom,
	)

	if innerBounds.Dx() > 0 && innerBounds.Dy() > 0 {
		innerCtx := ctx.SubContext(innerBounds)
		p.inner.render(innerCtx)
	}
}

// Padding modifier methods for stack types

// Padding adds equal padding on all sides of a Stack.
func (v *stack) Padding(n int) View {
	return Padding(n, v)
}

// PaddingHV adds horizontal and vertical padding to a Stack.
func (v *stack) PaddingHV(h, vpad int) View {
	return PaddingHV(h, vpad, v)
}

// PaddingLTRB adds specific padding to each side of a Stack.
func (v *stack) PaddingLTRB(left, top, right, bottom int) View {
	return PaddingLTRB(left, top, right, bottom, v)
}

// Padding adds equal padding on all sides of an HStack.
func (h *group) Padding(n int) View {
	return Padding(n, h)
}

// PaddingHV adds horizontal and vertical padding to an HStack.
func (h *group) PaddingHV(hpad, v int) View {
	return PaddingHV(hpad, v, h)
}

// PaddingLTRB adds specific padding to each side of an HStack.
func (h *group) PaddingLTRB(left, top, right, bottom int) View {
	return PaddingLTRB(left, top, right, bottom, h)
}

// Padding adds equal padding on all sides of a ZStack.
func (z *zStack) Padding(n int) View {
	return Padding(n, z)
}

// sizeView wraps a view with fixed or maximum size constraints
type sizeView struct {
	inner     View
	width     int // 0 = use inner's width
	height    int // 0 = use inner's height
	maxWidth  int // 0 = no max constraint
	maxHeight int // 0 = no max constraint
}

// Width wraps a view with a fixed width in character cells.
// The view will be exactly this width, clipping or padding as needed.
//
// Example:
//
//	Width(40, Text("This text will be exactly 40 cells wide"))
func Width(w int, inner View) View {
	return &sizeView{inner: inner, width: w}
}

// Height wraps a view with a fixed height in rows.
// The view will be exactly this height, clipping or padding as needed.
//
// Example:
//
//	Height(10, content)  // Exactly 10 rows tall
func Height(h int, inner View) View {
	return &sizeView{inner: inner, height: h}
}

// Size wraps a view with fixed width and height.
// Combines Width and Height into a single modifier.
//
// Example:
//
//	Size(80, 24, content)  // Exactly 80x24 cells
func Size(w, h int, inner View) View {
	return &sizeView{inner: inner, width: w, height: h}
}

// MaxWidth wraps a view with a maximum width constraint.
// The view can be smaller but will not exceed this width.
//
// Example:
//
//	MaxWidth(80, Text("Long text..."))  // Won't exceed 80 cells
func MaxWidth(w int, inner View) View {
	return &sizeView{inner: inner, maxWidth: w}
}

// MaxHeight wraps a view with a maximum height constraint.
// The view can be smaller but will not exceed this height.
//
// Example:
//
//	MaxHeight(20, content)  // Won't exceed 20 rows
func MaxHeight(h int, inner View) View {
	return &sizeView{inner: inner, maxHeight: h}
}

func (s *sizeView) size(maxWidth, maxHeight int) (int, int) {
	// Apply max constraints
	constrainedMaxW := maxWidth
	if s.maxWidth > 0 && (constrainedMaxW == 0 || s.maxWidth < constrainedMaxW) {
		constrainedMaxW = s.maxWidth
	}
	constrainedMaxH := maxHeight
	if s.maxHeight > 0 && (constrainedMaxH == 0 || s.maxHeight < constrainedMaxH) {
		constrainedMaxH = s.maxHeight
	}

	// Get inner size
	innerW, innerH := s.inner.size(constrainedMaxW, constrainedMaxH)

	// Apply fixed sizes
	w := innerW
	if s.width > 0 {
		w = s.width
	}
	h := innerH
	if s.height > 0 {
		h = s.height
	}

	// Clamp to constraints
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}

	return w, h
}

func (s *sizeView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}
	s.inner.render(ctx)
}

// Width modifier methods for view types

// Width sets a fixed width for a textView.
func (t *textView) Width(w int) View {
	return Width(w, t)
}

// Height sets a fixed height for a textView.
func (t *textView) Height(h int) View {
	return Height(h, t)
}

// MaxWidth sets a maximum width for a textView.
func (t *textView) MaxWidth(w int) View {
	return MaxWidth(w, t)
}

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
	isFocused := f.focusID != "" && focusManager.GetFocusedID() == f.focusID

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

// Bordered modifier methods for stack types

// Bordered wraps a Stack with a border.
func (v *stack) Bordered() *borderedView {
	return Bordered(v)
}

// Bordered wraps an HStack with a border.
func (h *group) Bordered() *borderedView {
	return Bordered(h)
}

// Bordered wraps a ZStack with a border.
func (z *zStack) Bordered() *borderedView {
	return Bordered(z)
}

// Background wraps a view with a background fill.
func Background(char rune, style Style, inner View) View {
	return &zStack{
		children: []View{
			&fillView{char: char, style: style},
			inner,
		},
		alignment: AlignLeft,
	}
}

// Bg adds a background color to a Stack.
func (v *stack) Bg(c Color) View {
	return Background(' ', NewStyle().WithBackground(c), v)
}

// Bg adds a background color to an HStack.
func (h *group) Bg(c Color) View {
	return Background(' ', NewStyle().WithBackground(c), h)
}
