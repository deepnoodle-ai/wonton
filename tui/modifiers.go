package tui

import "image"

// paddingView wraps a view with padding
type paddingView struct {
	inner                    View
	top, right, bottom, left int
}

// Padding wraps a view with equal padding on all sides.
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

func (p *paddingView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	// Calculate inner bounds
	innerBounds := image.Rect(
		bounds.Min.X+p.left,
		bounds.Min.Y+p.top,
		bounds.Max.X-p.right,
		bounds.Max.Y-p.bottom,
	)

	if !innerBounds.Empty() {
		p.inner.render(frame, innerBounds)
	}
}

// Padding modifier methods for stack types

// Padding adds equal padding on all sides of a VStack.
func (v *vStack) Padding(n int) View {
	return Padding(n, v)
}

// PaddingHV adds horizontal and vertical padding to a VStack.
func (v *vStack) PaddingHV(h, vpad int) View {
	return PaddingHV(h, vpad, v)
}

// PaddingLTRB adds specific padding to each side of a VStack.
func (v *vStack) PaddingLTRB(left, top, right, bottom int) View {
	return PaddingLTRB(left, top, right, bottom, v)
}

// Padding adds equal padding on all sides of an HStack.
func (h *hStack) Padding(n int) View {
	return Padding(n, h)
}

// PaddingHV adds horizontal and vertical padding to an HStack.
func (h *hStack) PaddingHV(hpad, v int) View {
	return PaddingHV(hpad, v, h)
}

// PaddingLTRB adds specific padding to each side of an HStack.
func (h *hStack) PaddingLTRB(left, top, right, bottom int) View {
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

// Width wraps a view with a fixed width.
func Width(w int, inner View) View {
	return &sizeView{inner: inner, width: w}
}

// Height wraps a view with a fixed height.
func Height(h int, inner View) View {
	return &sizeView{inner: inner, height: h}
}

// Size wraps a view with fixed width and height.
func Size(w, h int, inner View) View {
	return &sizeView{inner: inner, width: w, height: h}
}

// MaxWidth wraps a view with a maximum width constraint.
func MaxWidth(w int, inner View) View {
	return &sizeView{inner: inner, maxWidth: w}
}

// MaxHeight wraps a view with a maximum height constraint.
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

func (s *sizeView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}
	s.inner.render(frame, bounds)
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
}

// Bordered wraps a view with a border (optional title).
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

func (f *borderedView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	if f.border == nil {
		// No border, just render inner
		f.inner.render(frame, bounds)
		return
	}

	// Draw border
	w, h := bounds.Dx(), bounds.Dy()
	subFrame := frame.SubFrame(bounds)

	// Top border
	subFrame.PrintTruncated(0, 0, f.border.TopLeft, f.borderStyle)
	for x := 1; x < w-1; x++ {
		subFrame.PrintTruncated(x, 0, f.border.Horizontal, f.borderStyle)
	}
	if w > 1 {
		subFrame.PrintTruncated(w-1, 0, f.border.TopRight, f.borderStyle)
	}

	// Title in top border
	if f.title != "" && w > 4 {
		titleW, _ := MeasureText(f.title)
		maxTitleW := w - 4
		if titleW > maxTitleW {
			titleW = maxTitleW
		}
		titleX := 2
		subFrame.PrintTruncated(titleX, 0, f.title[:min(len(f.title), maxTitleW)], f.titleStyle)
	}

	// Side borders
	for y := 1; y < h-1; y++ {
		subFrame.PrintTruncated(0, y, f.border.Vertical, f.borderStyle)
		if w > 1 {
			subFrame.PrintTruncated(w-1, y, f.border.Vertical, f.borderStyle)
		}
	}

	// Bottom border
	if h > 1 {
		subFrame.PrintTruncated(0, h-1, f.border.BottomLeft, f.borderStyle)
		for x := 1; x < w-1; x++ {
			subFrame.PrintTruncated(x, h-1, f.border.Horizontal, f.borderStyle)
		}
		if w > 1 {
			subFrame.PrintTruncated(w-1, h-1, f.border.BottomRight, f.borderStyle)
		}
	}

	// Inner content (1 cell padding for border)
	innerBounds := image.Rect(
		bounds.Min.X+1,
		bounds.Min.Y+1,
		bounds.Max.X-1,
		bounds.Max.Y-1,
	)
	if !innerBounds.Empty() {
		f.inner.render(frame, innerBounds)
	}
}

// Bordered modifier methods for stack types

// Bordered wraps a VStack with a border.
func (v *vStack) Bordered() *borderedView {
	return Bordered(v)
}

// Bordered wraps an HStack with a border.
func (h *hStack) Bordered() *borderedView {
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

// Bg adds a background color to a VStack.
func (v *vStack) Bg(c Color) View {
	return Background(' ', NewStyle().WithBackground(c), v)
}

// Bg adds a background color to an HStack.
func (h *hStack) Bg(c Color) View {
	return Background(' ', NewStyle().WithBackground(c), h)
}
