package tui

import "github.com/deepnoodle-ai/wonton/runewidth"

// AnimatedBorderedView wraps a view with an animated border.
type AnimatedBorderedView struct {
	inner             View
	border            *BorderStyle
	borderAnimation   BorderAnimation
	title             string
	titleStyle        Style
	staticBorderStyle Style
}

// AnimatedBordered wraps a view with an animated border.
func AnimatedBordered(inner View, animation BorderAnimation) *AnimatedBorderedView {
	return &AnimatedBorderedView{
		inner: inner,
		border: &BorderStyle{
			TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘",
			Horizontal: "─", Vertical: "│",
		},
		borderAnimation:   animation,
		staticBorderStyle: NewStyle(),
		titleStyle:        NewStyle(),
	}
}

// Border sets the border characters.
func (f *AnimatedBorderedView) Border(style *BorderStyle) *AnimatedBorderedView {
	f.border = style
	return f
}

// Title sets the title shown in the top border.
func (f *AnimatedBorderedView) Title(title string) *AnimatedBorderedView {
	f.title = title
	return f
}

// TitleStyle sets the style for the title text.
func (f *AnimatedBorderedView) TitleStyle(s Style) *AnimatedBorderedView {
	f.titleStyle = s
	return f
}

// StaticBorderStyle sets a fallback style when no animation is active.
func (f *AnimatedBorderedView) StaticBorderStyle(s Style) *AnimatedBorderedView {
	f.staticBorderStyle = s
	return f
}

func (f *AnimatedBorderedView) size(maxWidth, maxHeight int) (int, int) {
	borderSize := 0
	if f.border != nil {
		borderSize = 2
	}

	return insetSize(f.inner, maxWidth, maxHeight, borderSize, borderSize)
}

func (f *AnimatedBorderedView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	if f.border == nil {
		f.inner.render(ctx)
		return
	}

	frame := ctx.Frame()
	cells := f.borderCells(w, h)
	for i, c := range cells {
		style := f.staticBorderStyle
		if ea, ok := f.borderAnimation.(edgeBorderAnimation); ok {
			style = ea.edgeBorderStyle(frame, c.part, c.edgeIndex, c.edgeLen)
		} else if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, c.part, i, len(cells))
		}
		ctx.PrintTruncated(c.x, c.y, c.glyph, style)
	}

	// Title
	if f.title != "" && w > 4 {
		titleX := 2
		ctx.PrintTruncated(titleX, 0, runewidth.Truncate(f.title, w-4, ""), f.titleStyle)
	}

	// Inner content
	if innerBounds, ok := insetBounds(w, h, 1, 1, 1, 1); ok {
		innerCtx := ctx.SubContext(innerBounds)
		f.inner.render(innerCtx)
	}
}

// borderCell is one cell of the border: where it goes, which part of the
// border it belongs to, what it draws, and for an edge cell, its index along
// that edge in clockwise order and the edge's length.
type borderCell struct {
	x, y      int
	part      BorderPart
	glyph     string
	edgeIndex int
	edgeLen   int
}

// borderCells lists the border's cells clockwise from the top-left corner,
// corners included. A cell's index is its position along the perimeter, so
// an animation flows through the corners instead of jumping at them.
func (f *AnimatedBorderedView) borderCells(w, h int) []borderCell {
	b := f.border
	cells := []borderCell{{x: 0, y: 0, part: BorderPartTopLeft, glyph: b.TopLeft}}
	edge := func(part BorderPart, glyph string, n int, at func(i int) (int, int)) {
		for i := range n {
			x, y := at(i)
			cells = append(cells, borderCell{x, y, part, glyph, i, n})
		}
	}
	edge(BorderPartTop, b.Horizontal, w-2, func(i int) (int, int) { return 1 + i, 0 })
	if w > 1 {
		cells = append(cells, borderCell{x: w - 1, y: 0, part: BorderPartTopRight, glyph: b.TopRight})
		edge(BorderPartRight, b.Vertical, h-2, func(i int) (int, int) { return w - 1, 1 + i })
		if h > 1 {
			cells = append(cells, borderCell{x: w - 1, y: h - 1, part: BorderPartBottomRight, glyph: b.BottomRight})
		}
	}
	if h > 1 {
		edge(BorderPartBottom, b.Horizontal, w-2, func(i int) (int, int) { return w - 2 - i, h - 1 })
		cells = append(cells, borderCell{x: 0, y: h - 1, part: BorderPartBottomLeft, glyph: b.BottomLeft})
	}
	edge(BorderPartLeft, b.Vertical, h-2, func(i int) (int, int) { return 0, h - 2 - i })
	return cells
}
