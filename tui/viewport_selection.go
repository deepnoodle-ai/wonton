package tui

import "strings"

// SelectionPoint is one end of a selection, addressed the same way the scroll
// anchor is: an item, a line within that item, and a column.
//
// Screen coordinates would be wrong here. A selection made while a reply is
// streaming has to stay on the words it was made on, and those words move down
// the screen every time a line is appended above them.
type SelectionPoint struct {
	Item int
	Line int
	Col  int
}

// before reports whether p comes earlier in the content than q.
func (p SelectionPoint) before(q SelectionPoint) bool {
	if p.Item != q.Item {
		return p.Item < q.Item
	}
	if p.Line != q.Line {
		return p.Line < q.Line
	}
	return p.Col < q.Col
}

// viewportSpan records where one item landed on screen during a render, so a
// mouse position can be turned back into a point in the content. top is the
// screen row of the item's first line and may be negative when the item is cut
// off at the top of the viewport.
type viewportSpan struct {
	item   int
	top    int
	height int
}

// SelectionActive reports whether a drag is in progress.
func (s *ViewportState) SelectionActive() bool { return s.selecting }

// HasSelection reports whether there is a selection to copy or draw.
func (s *ViewportState) HasSelection() bool { return s.hasSelection }

// Selection returns the selection in content order, earliest point first. The
// second return is false when there is no selection.
func (s *ViewportState) Selection() (start, end SelectionPoint, ok bool) {
	if !s.hasSelection {
		return SelectionPoint{}, SelectionPoint{}, false
	}
	if s.selCursor.before(s.selAnchor) {
		return s.selCursor, s.selAnchor, true
	}
	return s.selAnchor, s.selCursor, true
}

// ClearSelection drops the selection and restores whatever Follow was before
// it was made, so a selection taken during a stream does not silently leave the
// viewport unpinned once it is dismissed.
func (s *ViewportState) ClearSelection() {
	if s.followSuspended {
		s.Follow = s.followBeforeSelect
		s.followSuspended = false
	}
	s.hasSelection = false
	s.selecting = false
	s.dragEdge = 0
}

// suspendFollow pins the viewport for the life of a selection, remembering what
// to put back. Called by every gesture that starts one.
func (s *ViewportState) suspendFollow() {
	if !s.followSuspended {
		s.followBeforeSelect = s.Follow
		s.followSuspended = true
	}
	s.Follow = false
}

// BeginSelection starts a selection at a screen position, as a mouse press
// does. Exposed for applications that select from something other than a drag.
func (s *ViewportState) BeginSelection(x, y int) {
	p, ok := s.pointAt(x, y)
	if !ok {
		return
	}
	// Follow would drag the content out from under the pointer as the reply
	// streams. Remember what it was so dismissing the selection can put it back.
	s.suspendFollow()
	s.selAnchor, s.selCursor = p, p
	s.selecting = true
	// One press is not yet a selection: it becomes one when the drag moves off
	// the starting cell. Otherwise every click would leave a zero-width
	// highlight behind.
	s.hasSelection = false
}

// ExtendSelection moves the free end of the selection to a screen position.
func (s *ViewportState) ExtendSelection(x, y int) {
	if !s.selecting {
		return
	}
	p, ok := s.pointAt(x, y)
	if !ok {
		return
	}
	s.selCursor = p
	s.hasSelection = p != s.selAnchor
	s.dragEdge = s.edgeOf(y)
}

// EndSelection finishes a drag, leaving the selection in place to be copied.
//
// A drag that never became a selection puts Follow back on the way out. It has
// to: there is no selection left for the user to dismiss, so nothing else would
// ever restore it, and the viewport would stop following for good.
func (s *ViewportState) EndSelection() {
	s.selecting = false
	s.dragEdge = 0
	if !s.hasSelection {
		s.ClearSelection()
	}
}

// SelectWord selects the word under a screen position, as a double-click does.
func (s *ViewportState) SelectWord(x, y int) { s.selectRun(x, y, isWordRune) }

// SelectLine selects the whole line under a screen position, as a triple-click
// does.
func (s *ViewportState) SelectLine(x, y int) {
	p, ok := s.pointAt(x, y)
	if !ok {
		return
	}
	line := s.itemLine(p.Item, p.Line)
	if line.end == 0 {
		// A blank line selects nothing. Returning before suspendFollow matters:
		// suspending here would pin the viewport with no selection to show for
		// it and nothing to dismiss, and a chat would just stop following.
		return
	}
	s.suspendFollow()
	s.selAnchor = SelectionPoint{Item: p.Item, Line: p.Line, Col: 0}
	s.selCursor = SelectionPoint{Item: p.Item, Line: p.Line, Col: line.end}
	s.selecting = false
	s.hasSelection = true
}

// selectRun grows a selection out from a position while keep holds, working in
// graphemes and reporting the result in columns.
func (s *ViewportState) selectRun(x, y int, keep func(string) bool) {
	p, ok := s.pointAt(x, y)
	if !ok {
		return
	}
	line := s.itemLine(p.Item, p.Line)
	i := line.indexAt(p.Col)
	if i >= len(line.text) || !keep(line.text[i]) {
		return
	}
	first, last := i, i+1
	for first > 0 && keep(line.text[first-1]) {
		first--
	}
	for last < len(line.text) && keep(line.text[last]) {
		last++
	}
	s.suspendFollow()
	s.selAnchor = SelectionPoint{Item: p.Item, Line: p.Line, Col: line.col[first]}
	s.selCursor = SelectionPoint{Item: p.Item, Line: p.Line, Col: line.col[last]}
	s.selecting = false
	s.hasSelection = true
}

// isWordRune is what a double-click treats as part of the same word. Paths,
// identifiers and flags are what a user double-clicks in a terminal, so the
// separators inside them count as word characters.
func isWordRune(g string) bool {
	if g == "" {
		return false
	}
	r := []rune(g)[0]
	switch r {
	case '_', '-', '.', '/', ':', '@', '~', '+':
		return true
	}
	return r > ' ' && r != '"' && r != '\'' && r != '`' &&
		r != '(' && r != ')' && r != '[' && r != ']' && r != '{' && r != '}' &&
		r != ',' && r != ';' && r != '<' && r != '>' && r != '|' && r != '='
}

// HandleMouse drives selection from mouse events. It returns true when it
// consumed the event, so an application can fall through to its own handling —
// wheel scrolling, click targets — when it returns false.
//
// The gestures are the ones a terminal itself would provide: press to anchor,
// drag to extend, release to finish, double-click for a word, triple-click for
// a line. A press with no drag clears the selection, which is how a user
// dismisses one.
//
// A plain left click that neither dismissed a selection nor made one returns
// false. A press has to be taken to anchor a drag that may yet happen, but the
// click that follows it is the application's: without the fall-through no
// clickable element could live inside a Viewport at all.
func (s *ViewportState) HandleMouse(e MouseEvent) bool {
	if e.Button != MouseButtonLeft {
		return false
	}
	switch e.Type {
	case MousePress:
		s.BeginSelection(e.X, e.Y)
		return true
	case MouseDrag, MouseMove:
		if !s.selecting {
			return false
		}
		s.ExtendSelection(e.X, e.Y)
		return true
	case MouseRelease:
		if !s.selecting {
			return false
		}
		s.EndSelection()
		return true
	case MouseClick:
		switch {
		case e.ClickCount >= 3:
			s.SelectLine(e.X, e.Y)
		case e.ClickCount == 2:
			s.SelectWord(e.X, e.Y)
		default:
			// A single click that never became a drag dismisses the selection,
			// and that dismissal is the whole of the event. A click with
			// nothing to dismiss belongs to whatever the user clicked on.
			if s.hasSelection {
				s.ClearSelection()
				return true
			}
			s.ClearSelection()
			return false
		}
		return true
	}
	return false
}

// edgeOf reports which way a drag at screen row y is pulling: -1 above the
// viewport, +1 below, 0 inside it.
func (s *ViewportState) edgeOf(y int) int {
	switch {
	case y < 0:
		return -1
	case y >= s.Height:
		return 1
	default:
		return 0
	}
}

// DragAutoScroll scrolls the viewport when a drag is being held past its top or
// bottom edge, and reports whether it moved anything.
//
// Call it once per frame while SelectionActive: a pointer held still outside
// the viewport sends no further mouse events, so without a per-frame nudge the
// selection would stop growing the moment the user stopped moving.
func (s *ViewportState) DragAutoScroll() bool {
	if !s.selecting || s.dragEdge == 0 {
		return false
	}
	item, line := s.anchorItem, s.anchorLine
	s.ScrollBy(s.dragEdge)
	// ScrollBy sets Follow when it lands on the end of the content, which is
	// right for a user scrolling and wrong here: reaching the bottom mid-drag
	// would re-pin the viewport and let streaming content move the very words
	// being selected. The selection owns Follow until it is dismissed.
	if s.followSuspended {
		s.Follow = false
	}
	if s.anchorItem == item && s.anchorLine == line {
		return false
	}
	// The drag is still pinned at the edge, so the selection now reaches one
	// row further into the newly revealed content.
	//
	// The edge is resolved from the anchor rather than through pointAt, because
	// pointAt reads the layout recorded by the last render and the scroll above
	// just invalidated it. Going through it left the endpoint one row behind the
	// edge on every step — and since the scroll stops before the endpoint
	// catches up, the last line could never be reached by dragging at all.
	item, line = s.anchorItem, s.anchorLine
	if s.dragEdge > 0 {
		item, line = s.moveDown(item, line, s.Height-1)
	}
	p := SelectionPoint{Item: item, Line: line, Col: s.selCursor.Col}
	s.selCursor = p
	s.hasSelection = p != s.selAnchor
	return true
}

// pointAt turns a screen position into a point in the content, using the layout
// recorded by the last render. It fails only when nothing has been rendered.
func (s *ViewportState) pointAt(x, y int) (SelectionPoint, bool) {
	if len(s.layout) == 0 {
		return SelectionPoint{}, false
	}
	if x < 0 {
		x = 0
	}
	// Above the first row anchors at the very start of what is on screen.
	first := s.layout[0]
	if y < first.top {
		return SelectionPoint{Item: first.item, Line: 0, Col: 0}, true
	}
	prev := s.layout[0]
	for _, span := range s.layout {
		if y < span.top {
			// The row fell in the gap above this item. A drag through a gap
			// reads as continuing from the end of the item above it, not as
			// jumping to the one below.
			return s.endOf(prev), true
		}
		if y < span.top+span.height {
			return SelectionPoint{Item: span.item, Line: y - span.top, Col: x}, true
		}
		prev = span
	}
	// Past the last item on screen.
	return s.endOf(s.layout[len(s.layout)-1]), true
}

// endOf is the point just past the last character of a span's last line.
func (s *ViewportState) endOf(span viewportSpan) SelectionPoint {
	line := max(span.height-1, 0)
	return SelectionPoint{Item: span.item, Line: line, Col: s.itemLine(span.item, line).end}
}

// spanFor returns the layout span for item i, if it is on screen.
func (s *ViewportState) spanFor(item int) (viewportSpan, bool) {
	for _, span := range s.layout {
		if span.item == item {
			return span, true
		}
	}
	return viewportSpan{}, false
}

// SelectedText returns the selected text, one line per line of content, with
// trailing spaces removed.
//
// The text comes from re-rendering the selected items rather than from reading
// the screen, so a selection that runs off the top or bottom of the viewport —
// which auto-scrolling during a drag makes easy — copies in full.
func (s *ViewportState) SelectedText() string {
	start, end, ok := s.Selection()
	if !ok || s.width <= 0 {
		return ""
	}

	var out []string
	for item := start.Item; item <= end.Item; item++ {
		if item < 0 || item >= s.len() {
			continue
		}
		lines := s.itemLines(item)
		from, to := 0, len(lines)-1
		if item == start.Item {
			from = start.Line
		}
		if item == end.Item {
			to = end.Line
		}
		for line := max(from, 0); line <= to && line < len(lines); line++ {
			lo, hi := 0, lines[line].end
			if item == start.Item && line == start.Line {
				lo = start.Col
			}
			if item == end.Item && line == end.Line {
				hi = end.Col
			}
			out = append(out, strings.TrimRight(lines[line].slice(lo, hi), " "))
		}
	}
	return strings.Join(out, "\n")
}

// itemLine is one rendered row of an item: the graphemes on it, and the screen
// column each one starts at.
//
// Columns and graphemes are not interchangeable. A wide character occupies two
// columns, and a selection is made in columns — indexing the text by column
// would cut 世界 in half and hand back a broken rune.
type itemLine struct {
	text []string // one entry per grapheme
	col  []int    // col[i] is where text[i] starts; col has one extra entry, the end
	end  int      // one past the last column occupied
}

// slice returns the graphemes whose starting column falls in [lo, hi). A wide
// character is kept whole: it belongs to the slice that contains its first
// column, and to no other.
func (l itemLine) slice(lo, hi int) string {
	var out strings.Builder
	for i, g := range l.text {
		if l.col[i] >= lo && l.col[i] < hi {
			out.WriteString(g)
		}
	}
	return out.String()
}

// indexAt is the grapheme covering a column, or len(text) past the end.
func (l itemLine) indexAt(col int) int {
	for i := range l.text {
		if col < l.col[i+1] {
			return i
		}
	}
	return len(l.text)
}

// itemLine returns one line of one item, or an empty line if there is no such
// line.
func (s *ViewportState) itemLine(item, line int) itemLine {
	lines := s.itemLines(item)
	if line < 0 || line >= len(lines) {
		return itemLine{col: []int{0}}
	}
	return lines[line]
}

// itemLines returns the rendered rows of one item, rendering it off-screen at
// the viewport's width the first time it is asked. Styling is dropped; what
// comes back is what the user sees.
//
// The result is cached on the item's entry, so it is dropped by the same
// Invalidate, InvalidateAll and width change that drop the item's height. That
// caching is what keeps a selection cheap: the highlight asks for the end of
// every selected row on every frame, and without it each of those questions
// would re-render the whole item.
func (s *ViewportState) itemLines(item int) []itemLine {
	if e := s.entry(item); e.linesValid {
		return e.lines
	}
	lines := s.renderItemLines(item)
	// entry is re-read: rendering the item can grow the cache and invalidate
	// any pointer taken before it.
	e := s.entry(item)
	e.lines, e.linesValid = lines, true
	return lines
}

// renderItemLines does the off-screen render behind itemLines.
func (s *ViewportState) renderItemLines(item int) []itemLine {
	view := s.viewOf(item)
	if view == nil || s.width <= 0 {
		return nil
	}
	height := s.heightOf(item)
	if height <= 0 {
		return nil
	}

	var sink strings.Builder
	term := NewTestTerminal(s.width, height, &sink)
	frame, err := term.BeginFrame()
	if err != nil {
		return nil
	}
	frame.Fill(' ', NewStyle())
	view.render(NewRenderContext(frame, 0))
	term.EndFrame(frame)

	lines := make([]itemLine, height)
	for y := range height {
		row := itemLine{}
		for x := 0; x < s.width; x++ {
			cell := term.GetCell(x, y)
			if cell.Continuation {
				// The second half of a wide character: already accounted for.
				continue
			}
			char := cell.Char
			if char == 0 {
				char = ' '
			}
			row.col = append(row.col, x)
			row.text = append(row.text, string(char)+cell.Trailing)
		}
		// Trim the blanks off the end so a line's end is where its text stops.
		for len(row.text) > 0 && row.text[len(row.text)-1] == " " {
			row.text = row.text[:len(row.text)-1]
			row.col = row.col[:len(row.col)-1]
		}
		if n := len(row.text); n > 0 {
			row.end = row.col[n-1] + max(term.GetCell(row.col[n-1], y).Width, 1)
		} else {
			row.end = 0
		}
		row.col = append(row.col, row.end)
		lines[y] = row
	}
	return lines
}

// paintSelection repaints the selected cells in the selection style, keeping
// whatever character is already there. It runs after the items have drawn, so
// it is painting over their output rather than asking them to style themselves.
func (s *ViewportState) paintSelection(ctx *RenderContext) {
	start, end, ok := s.Selection()
	if !ok {
		return
	}
	style := s.SelectionStyle
	if style == (Style{}) {
		style = NewStyle().WithReverse()
	}

	for _, span := range s.layout {
		if span.item < start.Item || span.item > end.Item {
			continue
		}
		for line := range span.height {
			y := span.top + line
			if y < 0 || y >= s.Height {
				continue
			}
			lo, hi := s.rowRange(span.item, line, start, end)
			for x := lo; x < hi; x++ {
				// A continuation cell is restyled along with the wide
				// character that owns it, so it needs no visit of its own.
				if ctx.Cell(x, y).Continuation {
					continue
				}
				ctx.RestyleCell(x, y, style)
			}
		}
	}
}

// rowRange is the column span selected on one line of one item. A line in the
// middle of the selection runs to the end of its text rather than to the edge
// of the viewport, so the highlight does not trail blanks across the screen.
func (s *ViewportState) rowRange(item, line int, start, end SelectionPoint) (int, int) {
	lo, hi := 0, s.itemLine(item, line).end
	if item == start.Item {
		if line < start.Line {
			return 0, 0
		}
		if line == start.Line {
			lo = max(start.Col, 0)
		}
	}
	if item == end.Item {
		if line > end.Line {
			return 0, 0
		}
		if line == end.Line {
			hi = min(hi, end.Col)
		}
	}
	if hi > s.width {
		hi = s.width
	}
	return lo, max(lo, hi)
}

// ItemAt reports which item is drawn at a position in the viewport's own
// coordinates, and which of that item's lines the position falls on.
//
// It is how an application puts a click target inside a Viewport. HandleMouse
// returns false for a click it does not want — one that neither made a
// selection nor dismissed one — and this says what that click landed on.
//
// The last return is false outside the viewport's width, above the first item,
// below the last, and in the gaps between items.
func (s *ViewportState) ItemAt(x, y int) (item, line int, ok bool) {
	if x < 0 || (s.Width > 0 && x >= s.Width) {
		return 0, 0, false
	}
	for _, span := range s.layout {
		if y >= span.top && y < span.top+span.height {
			return span.item, y - span.top, true
		}
	}
	return 0, 0, false
}
