package tui

// ViewportItems is the application's list of items. It is read every frame, so
// items may be appended, replaced, or removed between frames.
//
// Item(i) is called at most once per index until Invalidate(i) is called or the
// render width changes. The returned view is retained by the ViewportState, so
// views that cache per instance — a parsed Markdown tree, a syntax-highlighted
// block — keep hitting their caches frame after frame.
type ViewportItems interface {
	// Len returns the number of items.
	Len() int

	// Item returns the view for item i, or nil for an item that renders
	// nothing. A nil item occupies no rows and gets no gap.
	Item(i int) View
}

// ViewportState is the scroll state of a Viewport. The application owns it and
// passes it to Viewport every frame, following the same convention as
// Scroll(content, &offset).
//
// Scrolling is virtualized: only the items that intersect the viewport are
// built and measured, so the per-frame cost is proportional to what is on
// screen rather than to the length of the list.
//
// The scroll position is an anchor — an item index plus a line within it —
// rather than an absolute line offset. That is what keeps the view still while
// content changes: appending to the list, or an item above the anchor growing
// as it streams, moves no pixels.
type ViewportState struct {
	// Follow pins the viewport to the end of the list. Set it for chat-style
	// behaviour, where new content should scroll into view. ScrollToBottom sets
	// it; scrolling up clears it; scrolling down far enough to reach the end
	// sets it again.
	Follow bool

	// Written by the view on every render; read them from HandleEvent.
	Width, Height int  // the viewport's size at the last render
	AtBottom      bool // nothing below the viewport
	LinesBelow    int  // content lines below the viewport, 0 when AtBottom

	anchorItem int // item at the top of the viewport
	anchorLine int // line within that item shown on the viewport's first row

	items ViewportItems
	gap   int
	cache []viewportEntry
	width int // the width cache entries were measured at; 0 before first render
}

// viewportEntry caches one item's view and its height at the cached width.
type viewportEntry struct {
	view   View
	height int
	valid  bool
}

// Invalidate drops the cached view and height for item i. Call it whenever an
// item's content changes in place; appending needs no call.
func (s *ViewportState) Invalidate(i int) {
	if i >= 0 && i < len(s.cache) {
		s.cache[i] = viewportEntry{}
	}
}

// InvalidateAll drops every cached view and height. Call it when the list is
// replaced wholesale, so indices no longer mean what they did.
func (s *ViewportState) InvalidateAll() {
	s.cache = s.cache[:0]
}

// ScrollBy moves the viewport by lines: positive scrolls down (toward newer
// content), negative up. The result is clamped to the content, so scrolling
// past either end is a no-op rather than an error.
//
// Any upward movement clears Follow; reaching the bottom sets it.
func (s *ViewportState) ScrollBy(lines int) {
	if !s.ready() || lines == 0 {
		return
	}
	if lines < 0 {
		s.Follow = false
		s.anchorItem, s.anchorLine = s.moveUp(s.anchorItem, s.anchorLine, -lines)
	} else {
		s.anchorItem, s.anchorLine = s.moveDown(s.anchorItem, s.anchorLine, lines)
	}
	s.clampAnchor()
	s.Follow = s.atBottom()
	s.updatePosition()
}

// PageUp scrolls up by one viewport, less a line of overlap so the reader keeps
// their place.
func (s *ViewportState) PageUp() { s.ScrollBy(-s.pageLines()) }

// PageDown scrolls down by one viewport, less a line of overlap.
func (s *ViewportState) PageDown() { s.ScrollBy(s.pageLines()) }

// ScrollToTop jumps to the start of the list.
func (s *ViewportState) ScrollToTop() {
	s.Follow = false
	if !s.ready() {
		return
	}
	s.anchorItem, s.anchorLine = s.firstVisible(), 0
	s.clampAnchor()
	s.Follow = s.atBottom()
	s.updatePosition()
}

// ScrollToBottom jumps to the end of the list and follows it from then on.
func (s *ViewportState) ScrollToBottom() {
	s.Follow = true
	if s.ready() {
		s.anchorItem, s.anchorLine = s.maxAnchor()
		s.AtBottom, s.LinesBelow = true, 0
	}
}

// ScrollToItem puts the top of item i on the viewport's first row, or as close
// as the end of the list allows.
func (s *ViewportState) ScrollToItem(i int) {
	s.Follow = false
	if !s.ready() {
		return
	}
	s.anchorItem, s.anchorLine = i, 0
	s.clampAnchor()
	s.Follow = s.atBottom()
	s.updatePosition()
}

// Anchor returns the item index and line currently at the top of the viewport.
// Useful for tests and for restoring a position across a rebuild.
func (s *ViewportState) Anchor() (item, line int) {
	return s.anchorItem, s.anchorLine
}

// ready reports whether the state has been through a render and so knows the
// width to measure at and the height to clamp against. Before that, the scroll
// methods have nothing to compute against and do nothing.
func (s *ViewportState) ready() bool {
	return s.items != nil && s.width > 0 && s.Height > 0
}

func (s *ViewportState) pageLines() int {
	if s.Height > 1 {
		return s.Height - 1
	}
	return 1
}

func (s *ViewportState) len() int {
	if s.items == nil {
		return 0
	}
	return s.items.Len()
}

// entry returns the cache slot for item i, growing the cache as needed.
func (s *ViewportState) entry(i int) *viewportEntry {
	if n := s.len(); len(s.cache) < n {
		s.cache = append(s.cache, make([]viewportEntry, n-len(s.cache))...)
	} else if len(s.cache) > n {
		s.cache = s.cache[:n]
	}
	if i < 0 || i >= len(s.cache) {
		return &viewportEntry{}
	}
	return &s.cache[i]
}

// viewOf returns item i's view, building it on first use.
func (s *ViewportState) viewOf(i int) View {
	e := s.entry(i)
	if !e.valid {
		e.view = s.items.Item(i)
		_, e.height = Measure(e.view, s.width, 0)
		if e.height < 0 {
			e.height = 0
		}
		e.valid = true
	}
	return e.view
}

// heightOf returns item i's height at the cached width, measuring on demand.
func (s *ViewportState) heightOf(i int) int {
	s.viewOf(i)
	return s.entry(i).height
}

// spanOf returns the rows item i occupies including the gap that follows it.
// An item that renders nothing occupies no rows and brings no gap, so a nil
// item does not leave a blank line behind.
func (s *ViewportState) spanOf(i int) int {
	h := s.heightOf(i)
	if h == 0 {
		return 0
	}
	return h + s.gap
}

func (s *ViewportState) firstVisible() int {
	for i, n := 0, s.len(); i < n; i++ {
		if s.heightOf(i) > 0 {
			return i
		}
	}
	return 0
}

func (s *ViewportState) nextVisible(i int) int {
	for j, n := i+1, s.len(); j < n; j++ {
		if s.heightOf(j) > 0 {
			return j
		}
	}
	return -1
}

func (s *ViewportState) prevVisible(i int) int {
	for j := i - 1; j >= 0; j-- {
		if s.heightOf(j) > 0 {
			return j
		}
	}
	return -1
}

// lastVisible returns the index of the last item that renders anything, or -1.
func (s *ViewportState) lastVisible() int {
	return s.prevVisible(s.len())
}

// moveDown walks the anchor forward by delta lines, stopping at the end of the
// list. Clamping to the bottom of the content is clampAnchor's job.
func (s *ViewportState) moveDown(item, line, delta int) (int, int) {
	for delta > 0 {
		span := s.spanOf(item)
		if line+delta < span {
			return item, line + delta
		}
		next := s.nextVisible(item)
		if next < 0 {
			return item, line
		}
		delta -= span - line
		item, line = next, 0
	}
	return item, line
}

// moveUp walks the anchor backward by delta lines, stopping at the start.
func (s *ViewportState) moveUp(item, line, delta int) (int, int) {
	for delta > 0 {
		if line >= delta {
			return item, line - delta
		}
		prev := s.prevVisible(item)
		if prev < 0 {
			return item, 0
		}
		delta -= line + 1
		item = prev
		line = s.spanOf(prev) - 1
	}
	return item, line
}

// maxAnchor returns the furthest the viewport can scroll: the anchor that puts
// the end of the content on the viewport's last row. When the content is
// shorter than the viewport this is the top of the list, so a short transcript
// grows downward from the top rather than clinging to the bottom.
//
// It walks backward from the end and stops as soon as it has covered Height
// rows, so its cost is proportional to the viewport, not to the list.
func (s *ViewportState) maxAnchor() (int, int) {
	last := s.lastVisible()
	if last < 0 || s.Height <= 0 {
		return s.firstVisible(), 0
	}

	remaining := s.Height
	for i := last; i >= 0; {
		h := s.heightOf(i)
		if h >= remaining {
			return i, h - remaining
		}
		remaining -= h

		prev := s.prevVisible(i)
		if prev < 0 {
			return i, 0 // everything fits; show it from the top
		}
		if s.gap >= remaining {
			// The viewport's first row falls inside the gap above item i,
			// which belongs to the preceding item's span.
			return prev, s.heightOf(prev) + (s.gap - remaining)
		}
		remaining -= s.gap
		i = prev
	}
	return s.firstVisible(), 0
}

// clampAnchor pulls the anchor back into range: not before the first visible
// item, not past the point where the content's end reaches the viewport's
// bottom, and never resting on an item that renders nothing.
func (s *ViewportState) clampAnchor() {
	n := s.len()
	if n == 0 {
		s.anchorItem, s.anchorLine = 0, 0
		return
	}
	if s.anchorItem >= n {
		s.anchorItem, s.anchorLine = n-1, 0
	}
	if s.anchorItem < 0 {
		s.anchorItem, s.anchorLine = 0, 0
	}
	// An item that renders nothing cannot hold an anchor; slide to the next
	// one that can, then to the previous if there is none after.
	if s.heightOf(s.anchorItem) == 0 {
		if next := s.nextVisible(s.anchorItem); next >= 0 {
			s.anchorItem, s.anchorLine = next, 0
		} else if prev := s.prevVisible(s.anchorItem); prev >= 0 {
			s.anchorItem, s.anchorLine = prev, 0
		} else {
			s.anchorItem, s.anchorLine = 0, 0
			return
		}
	}
	if s.anchorLine < 0 {
		s.anchorLine = 0
	}
	if span := s.spanOf(s.anchorItem); s.anchorLine >= span && span > 0 {
		s.anchorLine = span - 1
	}

	first := s.firstVisible()
	if anchorLess(s.anchorItem, s.anchorLine, first, 0) {
		s.anchorItem, s.anchorLine = first, 0
	}
	maxI, maxL := s.maxAnchor()
	if anchorLess(maxI, maxL, s.anchorItem, s.anchorLine) {
		s.anchorItem, s.anchorLine = maxI, maxL
	}
}

func (s *ViewportState) atBottom() bool {
	maxI, maxL := s.maxAnchor()
	return !anchorLess(s.anchorItem, s.anchorLine, maxI, maxL)
}

// linesBetween counts the content rows from anchor a to anchor b, where a is
// at or before b.
func (s *ViewportState) linesBetween(ai, al, bi, bl int) int {
	if ai > bi || (ai == bi && al >= bl) {
		return 0
	}
	total, i, l := 0, ai, al
	for i < bi {
		total += s.spanOf(i) - l
		l = 0
		next := s.nextVisible(i)
		if next < 0 || next > bi {
			return total
		}
		i = next
	}
	return total + (bl - l)
}

// anchorLess reports whether anchor (ai, al) is above (bi, bl).
func anchorLess(ai, al, bi, bl int) bool {
	if ai != bi {
		return ai < bi
	}
	return al < bl
}

// ViewportView is a scrollable, virtualized list of item views.
type ViewportView struct {
	state *ViewportState
	items ViewportItems
	gap   int
}

// Viewport returns a scrollable, virtualized list rendering items, with its
// scroll position in state.
//
// It is flexible (flex 1) and reports whatever size it is offered, so in a
// Stack it takes exactly the rows the fixed children leave — a chat transcript
// above a footer whose height changes as the input grows needs no arithmetic:
//
//	tui.Stack(
//	    tui.Viewport(&app.viewport, app).Gap(1),
//	    app.footerView(),
//	)
//
// Each frame it lays out from the scroll anchor (or, when state.Follow is set,
// backward from the end), building and measuring only the items that intersect
// the viewport and caching each one it touches. A list of ten thousand items
// costs the same per frame as a list of ten.
func Viewport(state *ViewportState, items ViewportItems) *ViewportView {
	return &ViewportView{state: state, items: items}
}

// Gap sets the number of blank rows between items. Items that render nothing
// bring no gap with them.
func (v *ViewportView) Gap(rows int) *ViewportView {
	if rows < 0 {
		rows = 0
	}
	v.gap = rows
	return v
}

func (v *ViewportView) flex() int { return 1 }

// size reports whatever it is offered rather than measuring its content: a
// viewport is defined by the space it is given, and measuring the content would
// be the O(list) pass virtualization exists to avoid.
func (v *ViewportView) size(maxWidth, maxHeight int) (int, int) {
	return maxWidth, maxHeight
}

func (v *ViewportView) render(ctx *RenderContext) {
	s := v.state
	if s == nil || v.items == nil {
		return
	}

	width, height := ctx.Size()
	if width <= 0 || height <= 0 {
		return
	}

	s.items = v.items
	s.gap = v.gap
	if width != s.width {
		// Every cached height was measured at the old width and is now wrong.
		s.InvalidateAll()
		s.width = width
	}
	s.Width, s.Height = width, height

	if s.Follow {
		s.anchorItem, s.anchorLine = s.maxAnchor()
	} else {
		s.clampAnchor()
	}

	// Draw from the anchor down until the viewport is full. The first item is
	// usually cut off at the top, which is what the offset frame is for.
	y := -s.anchorLine
	for i := s.anchorItem; i >= 0 && i < s.len() && y < height; i = s.nextVisible(i) {
		h := s.heightOf(i)
		if h > 0 && y+h > 0 {
			frame := &scrollRenderFrame{
				inner:         ctx.RenderFrame(),
				offsetY:       -y,
				clipH:         height,
				clipW:         width,
				contentHeight: h,
			}
			s.viewOf(i).render(ctx.WithFrame(frame))
		}
		y += h + s.gap
	}

	s.updatePosition()
}

// updatePosition refreshes the AtBottom and LinesBelow an application reads to
// draw a scroll indicator. Every scroll operation ends with it, so a footer
// built in the same frame as the scroll already reflects it — otherwise the
// indicator would trail the scroll by a frame.
func (s *ViewportState) updatePosition() {
	if !s.ready() {
		return
	}
	maxI, maxL := s.maxAnchor()
	s.AtBottom = !anchorLess(s.anchorItem, s.anchorLine, maxI, maxL)
	if s.AtBottom {
		s.LinesBelow = 0
	} else {
		s.LinesBelow = s.linesBetween(s.anchorItem, s.anchorLine, maxI, maxL)
	}
}
