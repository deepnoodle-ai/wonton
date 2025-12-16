package tui

import "image"

// animatedBorderedView wraps a view with an animated border.
type animatedBorderedView struct {
	inner            View
	border           *BorderStyle
	borderAnimation  BorderAnimation
	title            string
	titleStyle       Style
	staticBorderStyle Style
}

// AnimatedBordered wraps a view with an animated border.
func AnimatedBordered(inner View, animation BorderAnimation) *animatedBorderedView {
	return &animatedBorderedView{
		inner:             inner,
		border:            &BorderStyle{
			TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘",
			Horizontal: "─", Vertical: "│",
		},
		borderAnimation:   animation,
		staticBorderStyle: NewStyle(),
		titleStyle:        NewStyle(),
	}
}

// Border sets the border characters.
func (f *animatedBorderedView) Border(style *BorderStyle) *animatedBorderedView {
	f.border = style
	return f
}

// Title sets the title shown in the top border.
func (f *animatedBorderedView) Title(title string) *animatedBorderedView {
	f.title = title
	return f
}

// TitleStyle sets the style for the title text.
func (f *animatedBorderedView) TitleStyle(s Style) *animatedBorderedView {
	f.titleStyle = s
	return f
}

// StaticBorderStyle sets a fallback style when no animation is active.
func (f *animatedBorderedView) StaticBorderStyle(s Style) *animatedBorderedView {
	f.staticBorderStyle = s
	return f
}

func (f *animatedBorderedView) size(maxWidth, maxHeight int) (int, int) {
	borderSize := 0
	if f.border != nil {
		borderSize = 2
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

func (f *animatedBorderedView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	if f.border == nil {
		f.inner.render(ctx)
		return
	}

	frame := ctx.Frame()

	// Calculate total perimeter for border animation
	perimeter := (w-2)*2 + (h-2)*2 + 4 // Include corners

	// Draw top border
	position := 0
	for x := 1; x < w-1; x++ {
		style := f.staticBorderStyle
		if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, BorderPartTop, position, perimeter)
		}
		ctx.PrintTruncated(x, 0, f.border.Horizontal, style)
		position++
	}

	// Draw right border
	for y := 1; y < h-1; y++ {
		style := f.staticBorderStyle
		if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, BorderPartRight, position, perimeter)
		}
		if w > 1 {
			ctx.PrintTruncated(w-1, y, f.border.Vertical, style)
		}
		position++
	}

	// Draw bottom border
	for x := w - 2; x > 0; x-- {
		style := f.staticBorderStyle
		if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, BorderPartBottom, position, perimeter)
		}
		if h > 1 {
			ctx.PrintTruncated(x, h-1, f.border.Horizontal, style)
		}
		position++
	}

	// Draw left border
	for y := h - 2; y > 0; y-- {
		style := f.staticBorderStyle
		if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, BorderPartLeft, position, perimeter)
		}
		ctx.PrintTruncated(0, y, f.border.Vertical, style)
		position++
	}

	// Draw corners
	tlStyle := f.staticBorderStyle
	trStyle := f.staticBorderStyle
	blStyle := f.staticBorderStyle
	brStyle := f.staticBorderStyle

	if f.borderAnimation != nil {
		tlStyle = f.borderAnimation.GetBorderStyle(frame, BorderPartTopLeft, 0, perimeter)
		trStyle = f.borderAnimation.GetBorderStyle(frame, BorderPartTopRight, 0, perimeter)
		if h > 1 {
			blStyle = f.borderAnimation.GetBorderStyle(frame, BorderPartBottomLeft, 0, perimeter)
		}
		if w > 1 && h > 1 {
			brStyle = f.borderAnimation.GetBorderStyle(frame, BorderPartBottomRight, 0, perimeter)
		}
	}

	ctx.PrintTruncated(0, 0, f.border.TopLeft, tlStyle)
	if w > 1 {
		ctx.PrintTruncated(w-1, 0, f.border.TopRight, trStyle)
	}
	if h > 1 {
		ctx.PrintTruncated(0, h-1, f.border.BottomLeft, blStyle)
		if w > 1 {
			ctx.PrintTruncated(w-1, h-1, f.border.BottomRight, brStyle)
		}
	}

	// Title
	if f.title != "" && w > 4 {
		titleW, _ := MeasureText(f.title)
		maxTitleW := w - 4
		if titleW > maxTitleW {
			titleW = maxTitleW
		}
		titleX := 2
		ctx.PrintTruncated(titleX, 0, f.title[:min(len(f.title), maxTitleW)], f.titleStyle)
	}

	// Inner content
	innerBounds := image.Rect(1, 1, w-1, h-1)
	if innerBounds.Dx() > 0 && innerBounds.Dy() > 0 {
		innerCtx := ctx.SubContext(innerBounds)
		f.inner.render(innerCtx)
	}
}

// fadeView applies a fade/brightness animation to a view.
type fadeView struct {
	inner      View
	animation  *Animation
	minOpacity float64
	maxOpacity float64
}

// Fade wraps a view with a fade animation.
// The animation controls opacity from minOpacity to maxOpacity.
func Fade(inner View, animation *Animation, minOpacity, maxOpacity float64) *fadeView {
	return &fadeView{
		inner:      inner,
		animation:  animation,
		minOpacity: minOpacity,
		maxOpacity: maxOpacity,
	}
}

func (f *fadeView) size(maxWidth, maxHeight int) (int, int) {
	return f.inner.size(maxWidth, maxHeight)
}

func (f *fadeView) render(ctx *RenderContext) {
	// Update animation
	f.animation.Update(ctx.Frame())

	// Calculate current opacity
	opacity := f.minOpacity + (f.maxOpacity-f.minOpacity)*f.animation.Value()

	// Create a wrapper that applies opacity to all cells
	wrappedFrame := &opacityFrame{
		inner:   ctx.RenderFrame(),
		opacity: opacity,
	}

	wrappedCtx := ctx.WithFrame(wrappedFrame)
	f.inner.render(wrappedCtx)
}

// opacityFrame wraps a RenderFrame and applies opacity to all rendered cells.
type opacityFrame struct {
	inner   RenderFrame
	opacity float64
}

func (o *opacityFrame) SetCell(x, y int, char rune, style Style) error {
	// Adjust brightness based on opacity
	if style.FgRGB != nil && (style.FgRGB.R != 0 || style.FgRGB.G != 0 || style.FgRGB.B != 0) {
		style = style.WithFgRGB(RGB{
			R: uint8(float64(style.FgRGB.R) * o.opacity),
			G: uint8(float64(style.FgRGB.G) * o.opacity),
			B: uint8(float64(style.FgRGB.B) * o.opacity),
		})
	}
	if style.BgRGB != nil && (style.BgRGB.R != 0 || style.BgRGB.G != 0 || style.BgRGB.B != 0) {
		style = style.WithBgRGB(RGB{
			R: uint8(float64(style.BgRGB.R) * o.opacity),
			G: uint8(float64(style.BgRGB.G) * o.opacity),
			B: uint8(float64(style.BgRGB.B) * o.opacity),
		})
	}
	return o.inner.SetCell(x, y, char, style)
}

func (o *opacityFrame) Size() (int, int)                                             { return o.inner.Size() }
func (o *opacityFrame) PrintStyled(x, y int, text string, style Style) error        { return o.inner.PrintStyled(x, y, text, style) }
func (o *opacityFrame) PrintTruncated(x, y int, text string, style Style) error     { return o.inner.PrintTruncated(x, y, text, style) }
func (o *opacityFrame) FillStyled(x, y, w, h int, char rune, style Style) error     { return o.inner.FillStyled(x, y, w, h, char, style) }
func (o *opacityFrame) Fill(char rune, style Style) error                           { return o.inner.Fill(char, style) }
func (o *opacityFrame) PrintHyperlink(x, y int, link Hyperlink) error               { return o.inner.PrintHyperlink(x, y, link) }
func (o *opacityFrame) PrintHyperlinkFallback(x, y int, link Hyperlink) error       { return o.inner.PrintHyperlinkFallback(x, y, link) }
func (o *opacityFrame) GetBounds() image.Rectangle                                  { return o.inner.GetBounds() }
func (o *opacityFrame) SubFrame(bounds image.Rectangle) RenderFrame                 { return &opacityFrame{inner: o.inner.SubFrame(bounds), opacity: o.opacity} }

// brightnessView applies animated brightness changes to a view.
type brightnessView struct {
	inner      View
	animation  *Animation
	minBright  float64
	maxBright  float64
	colorTint  *RGB // Optional color to blend in
}

// Brightness wraps a view with a brightness animation.
func Brightness(inner View, animation *Animation, minBright, maxBright float64) *brightnessView {
	return &brightnessView{
		inner:     inner,
		animation: animation,
		minBright: minBright,
		maxBright: maxBright,
	}
}

// WithColorTint adds a color tint that becomes stronger as brightness increases.
func (b *brightnessView) WithColorTint(color RGB) *brightnessView {
	b.colorTint = &color
	return b
}

func (b *brightnessView) size(maxWidth, maxHeight int) (int, int) {
	return b.inner.size(maxWidth, maxHeight)
}

func (b *brightnessView) render(ctx *RenderContext) {
	b.animation.Update(ctx.Frame())
	brightness := b.minBright + (b.maxBright-b.minBright)*b.animation.Value()

	wrappedFrame := &brightnessFrame{
		inner:      ctx.RenderFrame(),
		brightness: brightness,
		colorTint:  b.colorTint,
	}

	wrappedCtx := ctx.WithFrame(wrappedFrame)
	b.inner.render(wrappedCtx)
}

type brightnessFrame struct {
	inner      RenderFrame
	brightness float64
	colorTint  *RGB
}

func (b *brightnessFrame) SetCell(x, y int, char rune, style Style) error {
	// Apply brightness
	if style.FgRGB != nil && (style.FgRGB.R != 0 || style.FgRGB.G != 0 || style.FgRGB.B != 0) {
		r := float64(style.FgRGB.R) * b.brightness
		g := float64(style.FgRGB.G) * b.brightness
		bl := float64(style.FgRGB.B) * b.brightness

		// Apply color tint if specified
		if b.colorTint != nil {
			tintStrength := (b.brightness - 0.5) * 0.3 // Tint gets stronger with brightness
			if tintStrength > 0 {
				r = r*(1-tintStrength) + float64(b.colorTint.R)*tintStrength
				g = g*(1-tintStrength) + float64(b.colorTint.G)*tintStrength
				bl = bl*(1-tintStrength) + float64(b.colorTint.B)*tintStrength
			}
		}

		style = style.WithFgRGB(RGB{
			R: uint8(min(255, r)),
			G: uint8(min(255, g)),
			B: uint8(min(255, bl)),
		})
	}
	return b.inner.SetCell(x, y, char, style)
}

func (b *brightnessFrame) Size() (int, int)                                             { return b.inner.Size() }
func (b *brightnessFrame) PrintStyled(x, y int, text string, style Style) error        { return b.inner.PrintStyled(x, y, text, style) }
func (b *brightnessFrame) PrintTruncated(x, y int, text string, style Style) error     { return b.inner.PrintTruncated(x, y, text, style) }
func (b *brightnessFrame) FillStyled(x, y, w, h int, char rune, style Style) error     { return b.inner.FillStyled(x, y, w, h, char, style) }
func (b *brightnessFrame) Fill(char rune, style Style) error                           { return b.inner.Fill(char, style) }
func (b *brightnessFrame) PrintHyperlink(x, y int, link Hyperlink) error               { return b.inner.PrintHyperlink(x, y, link) }
func (b *brightnessFrame) PrintHyperlinkFallback(x, y int, link Hyperlink) error       { return b.inner.PrintHyperlinkFallback(x, y, link) }
func (b *brightnessFrame) GetBounds() image.Rectangle                                  { return b.inner.GetBounds() }
func (b *brightnessFrame) SubFrame(bounds image.Rectangle) RenderFrame                 { return &brightnessFrame{inner: b.inner.SubFrame(bounds), brightness: b.brightness, colorTint: b.colorTint} }
