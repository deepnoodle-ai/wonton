package tui

import (
	"image"
	"strings"
)

// ScrollAnchor determines which part of content to show when content exceeds viewport.
type ScrollAnchor int

const (
	// ScrollAnchorTop shows content from the top (default scroll behavior).
	ScrollAnchorTop ScrollAnchor = iota
	// ScrollAnchorBottom shows content from the bottom (chat-style, newest at bottom).
	ScrollAnchorBottom
)

// ScrollView wraps content in a scrollable viewport.
type ScrollView struct {
	inner   View
	scrollY *int         // external scroll position (optional)
	anchor  ScrollAnchor // where to anchor when content exceeds viewport
}

// Scroll creates a scrollable viewport around the inner view.
// If scrollY is provided, it controls the scroll offset externally.
//
// Example:
//
//	Scroll(content, &app.scrollY).Anchor(ScrollAnchorBottom)
//	Scroll(content, nil).Bottom() // auto-scroll to bottom
func Scroll(inner View, scrollY *int) *ScrollView {
	return &ScrollView{
		inner:   inner,
		scrollY: scrollY,
		anchor:  ScrollAnchorTop,
	}
}

// Anchor sets the scroll anchor behavior.
func (s *ScrollView) Anchor(anchor ScrollAnchor) *ScrollView {
	s.anchor = anchor
	return s
}

// Bottom is a shorthand for Anchor(ScrollAnchorBottom).
func (s *ScrollView) Bottom() *ScrollView {
	s.anchor = ScrollAnchorBottom
	return s
}

func (s *ScrollView) flex() int {
	return 1 // Scroll views are flexible to fill available space
}

func (s *ScrollView) size(maxWidth, maxHeight int) (int, int) {
	// The scroll view takes all available space
	w, _ := s.inner.size(maxWidth, 0) // Get content width, ignore height constraint
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	h := maxHeight
	if h == 0 {
		// If no height constraint, use inner height
		_, h = s.inner.size(maxWidth, 0)
	}
	return w, h
}

func (s *ScrollView) render(ctx *RenderContext) {
	viewportWidth, viewportHeight := ctx.Size()
	if viewportWidth == 0 || viewportHeight == 0 {
		return
	}

	// Measure inner content without height constraint to get full content height
	_, contentHeight := s.inner.size(viewportWidth, 0)

	// If content fits in viewport, just render directly
	if contentHeight <= viewportHeight {
		s.inner.render(ctx)
		return
	}

	// Calculate scroll offset
	scrollY := 0
	if s.scrollY != nil {
		scrollY = *s.scrollY
	}

	// Calculate max scroll
	maxScroll := contentHeight - viewportHeight

	// Apply anchor behavior when no external scroll control
	if s.anchor == ScrollAnchorBottom && s.scrollY == nil {
		scrollY = maxScroll
	}

	// Clamp scroll position
	if scrollY < 0 {
		scrollY = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}

	// Update external scroll if provided
	if s.scrollY != nil && *s.scrollY != scrollY {
		*s.scrollY = scrollY
	}

	// Create an offset render frame that translates coordinates
	offsetFrame := newScrollFrame(ctx.RenderFrame(), scrollY, viewportWidth, viewportHeight, contentHeight)

	// Create a new context with the scroll frame, preserving the frame counter
	scrollCtx := ctx.WithFrame(offsetFrame)

	// Render inner content
	s.inner.render(scrollCtx)
}

// scrollRenderFrame wraps a RenderFrame and applies a vertical offset,
// only rendering cells that fall within the visible viewport.
type scrollRenderFrame struct {
	inner         RenderFrame
	offsetX       int             // X offset for subframes (from padding, etc.)
	offsetY       int             // scroll offset (how many rows to skip from top)
	clip          image.Rectangle // drawable area, in inner coordinates
	width         int             // frame width (for Size() reporting)
	contentHeight int             // full content height (for Size() reporting)
}

// newScrollFrame returns a frame over inner that shows a width x height
// window of content scrolled down by offsetY rows.
func newScrollFrame(inner RenderFrame, offsetY, width, height, contentHeight int) *scrollRenderFrame {
	return &scrollRenderFrame{
		inner:         inner,
		offsetY:       offsetY,
		clip:          image.Rect(0, 0, width, height),
		width:         width,
		contentHeight: contentHeight,
	}
}

// screen converts a point in this frame to inner coordinates and reports
// whether it is in the drawable area.
func (f *scrollRenderFrame) screen(x, y int) (int, int, bool) {
	screenX, screenY := x+f.offsetX, y-f.offsetY
	return screenX, screenY, image.Pt(screenX, screenY).In(f.clip)
}

// shows reports whether a point in this frame reaches the screen, through
// every scroll frame it is nested in.
func (f *scrollRenderFrame) shows(x, y int) bool {
	screenX, screenY, ok := f.screen(x, y)
	if !ok {
		return false
	}
	if inner, nested := f.inner.(*scrollRenderFrame); nested {
		return inner.shows(screenX, screenY)
	}
	return true
}

// rowVisible reports whether inner row screenY is in the drawable area.
func (f *scrollRenderFrame) rowVisible(screenY int) bool {
	return screenY >= f.clip.Min.Y && screenY < f.clip.Max.Y
}

// rows returns a frame over the drawable columns of inner, from row screenY
// to the bottom of the drawable area, so text clips at this frame's edges
// and wraps inside them.
func (f *scrollRenderFrame) rows(screenY int) RenderFrame {
	return f.inner.SubFrame(image.Rect(f.clip.Min.X, screenY, f.clip.Max.X, f.clip.Max.Y))
}

func (f *scrollRenderFrame) SetCell(x, y int, char rune, style Style) error {
	screenX, screenY, ok := f.screen(x, y)
	if !ok {
		return nil // Outside viewport, skip
	}
	return f.inner.SetCell(screenX, screenY, char, style)
}

func (f *scrollRenderFrame) PrintStyled(x, y int, text string, style Style) error {
	return f.printPerLine(x, y, text, style, RenderFrame.PrintStyled)
}

func (f *scrollRenderFrame) PrintTruncated(x, y int, text string, style Style) error {
	return f.printPerLine(x, y, text, style, RenderFrame.PrintTruncated)
}

// printPerLine clips a print to the viewport one line at a time.
//
// Text with embedded newlines occupies as many rows as it has lines, and a
// scrolled frame usually shows only some of them. Testing the starting row
// alone would drop a whole multi-line block the moment its first line scrolled
// off — which is exactly what an unwrapped Text renders as: one call, every
// line in it.
func (f *scrollRenderFrame) printPerLine(
	x, y int,
	text string,
	style Style,
	print func(RenderFrame, int, int, string, Style) error,
) error {
	rowX := x + f.offsetX - f.clip.Min.X
	for i, line := range strings.Split(text, "\n") {
		screenY := y + i - f.offsetY
		if !f.rowVisible(screenY) {
			continue
		}
		if err := print(f.rows(screenY), rowX, 0, line, style); err != nil {
			return err
		}
	}
	return nil
}

func (f *scrollRenderFrame) FillStyled(x, y, width, height int, char rune, style Style) error {
	area := image.Rect(x, y, x+width, y+height).
		Add(image.Pt(f.offsetX, -f.offsetY)).
		Intersect(f.clip)
	if area.Empty() {
		return nil
	}
	return f.inner.FillStyled(area.Min.X, area.Min.Y, area.Dx(), area.Dy(), char, style)
}

// Cell reads through the same offset SetCell writes through, so a caller works
// in content coordinates and gets a blank for anything scrolled out of view.
func (f *scrollRenderFrame) Cell(x, y int) Cell {
	screenX, screenY, ok := f.screen(x, y)
	if !ok {
		return Cell{Char: ' '}
	}
	return f.inner.Cell(screenX, screenY)
}

// RestyleCell restyles through the same offset Cell reads through, so a cell
// scrolled out of view is left alone rather than restyling whatever the
// viewport happens to be showing at that row.
func (f *scrollRenderFrame) RestyleCell(x, y int, style Style) error {
	screenX, screenY, ok := f.screen(x, y)
	if !ok {
		return nil
	}
	return f.inner.RestyleCell(screenX, screenY, style)
}

func (f *scrollRenderFrame) Size() (width, height int) {
	// Report full content height so inner views render all content,
	// even parts that are scrolled out of view (they get clipped by SetCell/PrintStyled)
	return f.width, f.contentHeight
}

// GetBounds returns where the frame's content sits on screen, scrolled-off
// rows included, so a view can add its own coordinates to bounds.Min to place
// a click region.
func (f *scrollRenderFrame) GetBounds() image.Rectangle {
	origin, _ := screenArea(f)
	return image.Rect(0, 0, f.width, f.contentHeight).Add(origin)
}

func (f *scrollRenderFrame) SubFrame(rect image.Rectangle) RenderFrame {
	// Create a subframe that accumulates X offset and adjusts Y offset, and
	// clips to the part of rect this frame can draw on.
	return &scrollRenderFrame{
		inner:         f.inner,
		offsetX:       f.offsetX + rect.Min.X, // Accumulate X offset
		offsetY:       f.offsetY - rect.Min.Y, // Adjust Y offset for nested bounds
		clip:          rect.Add(image.Pt(f.offsetX, -f.offsetY)).Intersect(f.clip),
		width:         rect.Dx(),
		contentHeight: rect.Dy(), // Use subframe's height for content
	}
}

func (f *scrollRenderFrame) PrintHyperlink(x, y int, link Hyperlink) error {
	screenY := y - f.offsetY
	if !f.rowVisible(screenY) {
		return nil
	}
	return f.rows(screenY).PrintHyperlink(x+f.offsetX-f.clip.Min.X, 0, link)
}

func (f *scrollRenderFrame) PrintHyperlinkFallback(x, y int, link Hyperlink) error {
	screenY := y - f.offsetY
	if !f.rowVisible(screenY) {
		return nil
	}
	return f.rows(screenY).PrintHyperlinkFallback(x+f.offsetX-f.clip.Min.X, 0, link)
}

func (f *scrollRenderFrame) Fill(char rune, style Style) error {
	return f.FillStyled(0, 0, f.width, f.contentHeight, char, style)
}
