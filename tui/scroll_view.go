package tui

import "image"

// ScrollAnchor determines which part of content to show when content exceeds viewport.
type ScrollAnchor int

const (
	// ScrollAnchorTop shows content from the top (default scroll behavior).
	ScrollAnchorTop ScrollAnchor = iota
	// ScrollAnchorBottom shows content from the bottom (chat-style, newest at bottom).
	ScrollAnchorBottom
)

// scrollView wraps content in a scrollable viewport.
type scrollView struct {
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
func Scroll(inner View, scrollY *int) *scrollView {
	return &scrollView{
		inner:   inner,
		scrollY: scrollY,
		anchor:  ScrollAnchorTop,
	}
}

// Anchor sets the scroll anchor behavior.
func (s *scrollView) Anchor(anchor ScrollAnchor) *scrollView {
	s.anchor = anchor
	return s
}

// Bottom is a shorthand for Anchor(ScrollAnchorBottom).
func (s *scrollView) Bottom() *scrollView {
	s.anchor = ScrollAnchorBottom
	return s
}

func (s *scrollView) flex() int {
	return 1 // Scroll views are flexible to fill available space
}

func (s *scrollView) size(maxWidth, maxHeight int) (int, int) {
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

func (s *scrollView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	viewportWidth := bounds.Dx()
	viewportHeight := bounds.Dy()

	// Measure inner content without height constraint to get full content height
	_, contentHeight := s.inner.size(viewportWidth, 0)

	// If content fits in viewport, just render directly
	if contentHeight <= viewportHeight {
		s.inner.render(frame, bounds)
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
	subFrame := frame.SubFrame(bounds)
	offsetFrame := &scrollRenderFrame{
		inner:   subFrame,
		offsetY: scrollY,
		clipH:   viewportHeight,
		clipW:   viewportWidth,
	}

	// Render inner content with full height bounds, offset frame handles clipping
	contentBounds := image.Rect(0, 0, viewportWidth, contentHeight)
	s.inner.render(offsetFrame, contentBounds)
}

// scrollRenderFrame wraps a RenderFrame and applies a vertical offset,
// only rendering cells that fall within the visible viewport.
type scrollRenderFrame struct {
	inner   RenderFrame
	offsetX int // X offset for subframes (from padding, etc.)
	offsetY int // scroll offset (how many rows to skip from top)
	clipH   int // viewport height
	clipW   int // viewport width
}

func (f *scrollRenderFrame) SetCell(x, y int, char rune, style Style) error {
	// Apply offsets and check if in viewport
	screenX := x + f.offsetX
	screenY := y - f.offsetY
	if screenY < 0 || screenY >= f.clipH || screenX < 0 || screenX >= f.clipW {
		return nil // Outside viewport, skip
	}
	return f.inner.SetCell(screenX, screenY, char, style)
}

func (f *scrollRenderFrame) PrintStyled(x, y int, text string, style Style) error {
	screenX := x + f.offsetX
	screenY := y - f.offsetY
	if screenY < 0 || screenY >= f.clipH {
		return nil // Entire line outside viewport
	}
	return f.inner.PrintStyled(screenX, screenY, text, style)
}

func (f *scrollRenderFrame) PrintTruncated(x, y int, text string, style Style) error {
	screenX := x + f.offsetX
	screenY := y - f.offsetY
	if screenY < 0 || screenY >= f.clipH {
		return nil
	}
	return f.inner.PrintTruncated(screenX, screenY, text, style)
}

func (f *scrollRenderFrame) FillStyled(x, y, width, height int, char rune, style Style) error {
	screenX := x + f.offsetX
	// For each row in the fill area, check if it's visible
	for row := 0; row < height; row++ {
		screenY := (y + row) - f.offsetY
		if screenY >= 0 && screenY < f.clipH {
			if err := f.inner.FillStyled(screenX, screenY, width, 1, char, style); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *scrollRenderFrame) Size() (width, height int) {
	return f.clipW, f.clipH
}

func (f *scrollRenderFrame) GetBounds() image.Rectangle {
	// Return the full content bounds, not the viewport bounds
	return image.Rect(0, 0, f.clipW, f.clipH+f.offsetY)
}

func (f *scrollRenderFrame) SubFrame(rect image.Rectangle) RenderFrame {
	// Create a subframe that accumulates X offset and adjusts Y offset
	return &scrollRenderFrame{
		inner:   f.inner,
		offsetX: f.offsetX + rect.Min.X, // Accumulate X offset
		offsetY: f.offsetY - rect.Min.Y, // Adjust Y offset for nested bounds
		clipH:   f.clipH,
		clipW:   f.clipW,
	}
}

func (f *scrollRenderFrame) PrintHyperlink(x, y int, link Hyperlink) error {
	screenX := x + f.offsetX
	screenY := y - f.offsetY
	if screenY < 0 || screenY >= f.clipH {
		return nil
	}
	return f.inner.PrintHyperlink(screenX, screenY, link)
}

func (f *scrollRenderFrame) PrintHyperlinkFallback(x, y int, link Hyperlink) error {
	screenX := x + f.offsetX
	screenY := y - f.offsetY
	if screenY < 0 || screenY >= f.clipH {
		return nil
	}
	return f.inner.PrintHyperlinkFallback(screenX, screenY, link)
}

func (f *scrollRenderFrame) Fill(char rune, style Style) error {
	return f.inner.Fill(char, style)
}
