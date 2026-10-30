package tui

import (
	"image"
	"math"
)

// SplitAxis is the direction a split arranges its children in.
type SplitAxis int

const (
	// SplitColumns places children side by side, left to right.
	SplitColumns SplitAxis = iota
	// SplitRows stacks children top to bottom.
	SplitRows
)

// SplitNode is one node of a split layout: a leaf naming a pane, or a
// branch dividing its space among its children along an axis. Branches nest,
// so a column of rows of columns is a tree three levels deep.
type SplitNode struct {
	// ID names the pane a leaf shows. Branches have no ID.
	ID string

	// Axis is how a branch arranges its children.
	Axis SplitAxis

	// Children are a branch's nodes, in order along its axis.
	Children []*SplitNode

	// Weight is the node's share of its parent's space, relative to its
	// siblings' weights. Zero or less counts as 1.
	Weight float64

	bounds image.Rectangle // where the node was drawn, in screen cells
}

// IsLeaf reports whether the node is a pane rather than a branch.
func (n *SplitNode) IsLeaf() bool { return len(n.Children) == 0 }

// Bounds returns where the node was drawn at the last render, in screen
// coordinates.
func (n *SplitNode) Bounds() image.Rectangle { return n.bounds }

func (n *SplitNode) weight() float64 {
	if n.Weight <= 0 || math.IsNaN(n.Weight) || math.IsInf(n.Weight, 0) {
		return 1
	}
	return n.Weight
}

// SplitState is a resizable split layout: a tree of panes the application
// owns and passes to Split every frame, in the same way as ViewportState.
// Panes are named by string IDs the application chooses.
//
// The layout leaves a one-cell gutter between neighboring panes, or none
// when Tight is set. Dragging a gutter with the left button moves it; route
// mouse events to HandleMouse to make that work.
//
//	split := &tui.SplitState{}
//	split.Append("editor", tui.SplitColumns)
//	split.Split("editor", "shell", tui.SplitRows)   // shell below editor
//	split.Append("logs", tui.SplitColumns)          // a new column on the right
//
//	func (a *app) View() tui.View {
//	    return tui.Split(&a.split, func(id string) tui.View { return a.pane(id) })
//	}
//
//	func (a *app) HandleEvent(ev tui.Event) []tui.Cmd {
//	    if e, ok := ev.(tui.MouseEvent); ok && a.split.HandleMouse(e) {
//	        return nil
//	    }
//	    ...
//	}
//
// Positions reported by the state (Bounds, PaneAt, DividerAt) are screen
// coordinates from the last render, which is what mouse events carry.
type SplitState struct {
	// Root is the layout tree. Nil is an empty layout. The application may
	// edit the tree directly — change weights, reorder children — as long
	// as every leaf has an ID and every branch has children.
	Root *SplitNode

	// MinSize is the smallest width or height, in cells, that dragging a
	// gutter leaves a pane. Zero means 3.
	MinSize int

	// Tight leaves no gutter: neighboring panes touch. It suits panes that
	// draw their own borders, which already show where one ends. The two
	// cells either side of the edge where panes meet act as the gutter for
	// dragging, and are highlighted under the pointer.
	Tight bool

	dividers []splitDivider
	drag     *splitDivider // the gutter being dragged
	grab     int           // where in the gutter the drag took hold
	hover    *splitDivider // the gutter under the pointer
}

// splitDivider is a gutter between two neighboring children of a branch.
type splitDivider struct {
	node   *SplitNode
	index  int // between node.Children[index] and node.Children[index+1]
	bounds image.Rectangle
}

func (d *splitDivider) same(o *splitDivider) bool {
	return d != nil && o != nil && d.node == o.node && d.index == o.index
}

const splitGutter = 1

// gap is the number of cells between neighboring panes.
func (s *SplitState) gap() int {
	if s.Tight {
		return 0
	}
	return splitGutter
}

func (s *SplitState) minSize() int {
	if s.MinSize <= 0 {
		return 3
	}
	return s.MinSize
}

// Leaves returns the IDs of every pane, in reading order: depth first, left
// to right and top to bottom.
func (s *SplitState) Leaves() []string {
	var ids []string
	var walk func(n *SplitNode)
	walk = func(n *SplitNode) {
		if n == nil {
			return
		}
		if n.IsLeaf() {
			ids = append(ids, n.ID)
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(s.Root)
	return ids
}

// Contains reports whether the layout has a pane with this ID.
func (s *SplitState) Contains(id string) bool {
	_, leaf := s.find(id)
	return leaf != nil
}

// Node returns the leaf for a pane, or nil.
func (s *SplitState) Node(id string) *SplitNode {
	_, leaf := s.find(id)
	return leaf
}

// find returns a pane's leaf and its parent (nil for the root).
func (s *SplitState) find(id string) (parent, leaf *SplitNode) {
	var walk func(p, n *SplitNode) bool
	walk = func(p, n *SplitNode) bool {
		if n == nil {
			return false
		}
		if n.IsLeaf() {
			if n.ID == id {
				parent, leaf = p, n
				return true
			}
			return false
		}
		for _, c := range n.Children {
			if walk(n, c) {
				return true
			}
		}
		return false
	}
	walk(nil, s.Root)
	return parent, leaf
}

// Append adds a pane at the far end of the layout along axis: a new column
// on the right for SplitColumns, a new row at the bottom for SplitRows. The
// new pane gets an equal share. On an empty layout it becomes the only pane.
func (s *SplitState) Append(id string, axis SplitAxis) {
	leaf := &SplitNode{ID: id}
	// The gutters from the last render describe the old tree.
	s.drag, s.hover, s.dividers = nil, nil, nil
	switch {
	case s.Root == nil:
		s.Root = leaf
	case !s.Root.IsLeaf() && s.Root.Axis == axis:
		leaf.Weight = averageWeight(s.Root.Children)
		s.Root.Children = append(s.Root.Children, leaf)
	default:
		old := s.Root
		s.Root = &SplitNode{Axis: axis, Children: []*SplitNode{old, leaf}}
		old.Weight, leaf.Weight = 1, 1
	}
}

func averageWeight(nodes []*SplitNode) float64 {
	if len(nodes) == 0 {
		return 1
	}
	var sum float64
	for _, n := range nodes {
		sum += n.weight()
	}
	return sum / float64(len(nodes))
}

// Split divides the pane target in two along axis and puts the new pane id
// after it: to its right for SplitColumns, below it for SplitRows. The two
// share the space target had. It reports false if target is not in the
// layout.
func (s *SplitState) Split(target, id string, axis SplitAxis) bool {
	parent, leaf := s.find(target)
	if leaf == nil {
		return false
	}
	// The gutters from the last render describe the old tree.
	s.drag, s.hover, s.dividers = nil, nil, nil
	half := leaf.weight() / 2
	added := &SplitNode{ID: id, Weight: half}
	if parent != nil && parent.Axis == axis {
		// Join the existing row or column rather than nesting another.
		leaf.Weight = half
		i := indexOf(parent.Children, leaf)
		parent.Children = append(parent.Children[:i+1], append([]*SplitNode{added}, parent.Children[i+1:]...)...)
		return true
	}
	// Turn the leaf into a branch holding the old pane and the new one.
	moved := &SplitNode{ID: leaf.ID, Weight: 1}
	added.Weight = 1
	leaf.ID = ""
	leaf.Axis = axis
	leaf.Children = []*SplitNode{moved, added}
	return true
}

func indexOf(nodes []*SplitNode, n *SplitNode) int {
	for i, c := range nodes {
		if c == n {
			return i
		}
	}
	return -1
}

// Remove takes a pane out of the layout. Its neighbors take over its space,
// and a branch left with one child is replaced by that child. It reports
// false if the pane is not in the layout.
func (s *SplitState) Remove(id string) bool {
	parent, leaf := s.find(id)
	if leaf == nil {
		return false
	}
	// The gutters from the last render describe the old tree.
	s.drag, s.hover, s.dividers = nil, nil, nil
	if parent == nil {
		s.Root = nil
		return true
	}
	i := indexOf(parent.Children, leaf)
	parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
	if len(parent.Children) == 1 {
		only := parent.Children[0]
		w := parent.Weight
		*parent = *only
		parent.Weight = w
		// The surviving child may share the grandparent's axis: merge it in
		// so the tree stays as flat as the layout looks.
		s.flatten()
	}
	return true
}

// flatten merges any branch whose parent has the same axis into its parent,
// scaling its children's weights so nothing moves on screen.
func (s *SplitState) flatten() {
	var walk func(n *SplitNode)
	walk = func(n *SplitNode) {
		if n == nil || n.IsLeaf() {
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
		var merged []*SplitNode
		for _, c := range n.Children {
			if c.IsLeaf() || c.Axis != n.Axis {
				merged = append(merged, c)
				continue
			}
			var total float64
			for _, g := range c.Children {
				total += g.weight()
			}
			for _, g := range c.Children {
				g.Weight = c.weight() * g.weight() / total
				merged = append(merged, g)
			}
		}
		n.Children = merged
	}
	walk(s.Root)
}

// Equalize gives every pane in every branch an equal share.
func (s *SplitState) Equalize() {
	var walk func(n *SplitNode)
	walk = func(n *SplitNode) {
		if n == nil {
			return
		}
		n.Weight = 1
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(s.Root)
}

// Bounds returns where a pane was drawn at the last render, in screen
// coordinates.
func (s *SplitState) Bounds(id string) (image.Rectangle, bool) {
	leaf := s.Node(id)
	if leaf == nil || leaf.bounds.Empty() {
		return image.Rectangle{}, false
	}
	return leaf.bounds, true
}

// PaneAt returns the pane drawn at a screen position.
func (s *SplitState) PaneAt(x, y int) (string, bool) {
	pt := image.Pt(x, y)
	for _, id := range s.Leaves() {
		if b, ok := s.Bounds(id); ok && pt.In(b) {
			return id, true
		}
	}
	return "", false
}

// DividerAt returns the gutter at a screen position: the branch it belongs
// to and the index of the child before it. The gutter separates
// branch.Children[index] and branch.Children[index+1].
func (s *SplitState) DividerAt(x, y int) (branch *SplitNode, index int, ok bool) {
	if d := s.dividerAt(x, y); d != nil {
		return d.node, d.index, true
	}
	return nil, 0, false
}

func (s *SplitState) dividerAt(x, y int) *splitDivider {
	pt := image.Pt(x, y)
	for i := range s.dividers {
		if pt.In(s.dividers[i].bounds) {
			return &s.dividers[i]
		}
	}
	return nil
}

// Dragging reports whether a gutter is being dragged.
func (s *SplitState) Dragging() bool { return s.drag != nil }

// Neighbor returns the pane next to id in a direction: dx of -1 or +1 looks
// left or right, dy of -1 or +1 up or down. Of the panes on that side, it
// picks the one that overlaps id's span the most. It uses the bounds from the
// last render.
func (s *SplitState) Neighbor(id string, dx, dy int) (string, bool) {
	from, ok := s.Bounds(id)
	if !ok || (dx == 0 && dy == 0) {
		return "", false
	}
	best, bestOverlap, bestDist := "", 0, math.MaxInt
	for _, other := range s.Leaves() {
		if other == id {
			continue
		}
		b, ok := s.Bounds(other)
		if !ok {
			continue
		}
		var dist, overlap int
		switch {
		case dx > 0 && b.Min.X >= from.Max.X:
			dist, overlap = b.Min.X-from.Max.X, span(from.Min.Y, from.Max.Y, b.Min.Y, b.Max.Y)
		case dx < 0 && b.Max.X <= from.Min.X:
			dist, overlap = from.Min.X-b.Max.X, span(from.Min.Y, from.Max.Y, b.Min.Y, b.Max.Y)
		case dy > 0 && b.Min.Y >= from.Max.Y:
			dist, overlap = b.Min.Y-from.Max.Y, span(from.Min.X, from.Max.X, b.Min.X, b.Max.X)
		case dy < 0 && b.Max.Y <= from.Min.Y:
			dist, overlap = from.Min.Y-b.Max.Y, span(from.Min.X, from.Max.X, b.Min.X, b.Max.X)
		default:
			continue
		}
		if overlap <= 0 {
			continue
		}
		if dist < bestDist || (dist == bestDist && overlap > bestOverlap) {
			best, bestOverlap, bestDist = other, overlap, dist
		}
	}
	return best, best != ""
}

func span(a0, a1, b0, b1 int) int {
	return min(a1, b1) - max(a0, b0)
}

// HandleMouse moves gutters. A left press on a gutter starts a drag, which
// takes every event until the button is released. Pointer motion over a
// gutter highlights it but is not consumed. It reports whether the event was
// consumed.
func (s *SplitState) HandleMouse(e MouseEvent) bool {
	if s.drag != nil {
		switch e.Type {
		case MouseDrag, MouseMove:
			s.dragTo(e.X, e.Y)
		case MouseRelease, MouseDragEnd, MouseDragCancel:
			s.drag = nil
		}
		return true
	}
	switch e.Type {
	case MouseMove:
		s.hover = nil
		if d := s.dividerAt(e.X, e.Y); d != nil {
			hover := *d
			s.hover = &hover
		}
		return false
	case MousePress:
		if e.Button != MouseButtonLeft {
			return false
		}
		d := s.dividerAt(e.X, e.Y)
		if d == nil {
			return false
		}
		drag := *d
		s.drag = &drag
		// Measure from where the edge is, not the pointer, so the gutter
		// does not jump when a tight one is taken by its first cell.
		a := d.node.Children[d.index].bounds
		if d.node.Axis == SplitColumns {
			s.grab = a.Max.X - e.X
		} else {
			s.grab = a.Max.Y - e.Y
		}
		return true
	}
	return false
}

// dragTo moves the dragged gutter so it sits at the pointer, as far as the
// minimum size of the panes on either side allows.
func (s *SplitState) dragTo(x, y int) {
	d := s.drag
	if d.index+1 >= len(d.node.Children) {
		s.drag = nil
		return
	}
	a, b := d.node.Children[d.index], d.node.Children[d.index+1]
	var start, end, pos int
	if d.node.Axis == SplitColumns {
		start, end, pos = a.bounds.Min.X, b.bounds.Max.X, x+s.grab
	} else {
		start, end, pos = a.bounds.Min.Y, b.bounds.Max.Y, y+s.grab
	}
	gap := s.gap()
	total := end - start - gap
	if total <= 0 {
		return // no room on either side to move the gutter into
	}
	minSize := min(s.minSize(), total/2)
	sizeA := max(minSize, min(pos-start, total-minSize))
	pair := a.weight() + b.weight()
	a.Weight = pair * float64(sizeA) / float64(total)
	b.Weight = pair - a.Weight
	// Keep the bounds current so a second drag event before the next render
	// measures from where the gutter now is.
	if d.node.Axis == SplitColumns {
		a.bounds.Max.X = start + sizeA
		b.bounds.Min.X = a.bounds.Max.X + gap
	} else {
		a.bounds.Max.Y = start + sizeA
		b.bounds.Min.Y = a.bounds.Max.Y + gap
	}
}

// layout assigns screen bounds to every node and records the gutters.
func (s *SplitState) layout(n *SplitNode, r image.Rectangle) {
	n.bounds = r
	if n.IsLeaf() {
		return
	}
	length := r.Dx()
	if n.Axis == SplitRows {
		length = r.Dy()
	}
	count := len(n.Children)
	gap := s.gap()
	avail := max(0, length-gap*(count-1))

	var total float64
	for _, c := range n.Children {
		total += c.weight()
	}
	// Place each boundary by rounding the running total rather than each
	// size on its own, so rounding never adds up to a gap or an overflow,
	// and moving one gutter never shifts another.
	var prefix float64
	offset := 0
	for i, c := range n.Children {
		prefix += c.weight()
		end := int(math.Round(float64(avail) * prefix / total))
		if i == count-1 {
			end = avail
		}
		size := max(0, end-offset)
		pos := offset + i*gap
		var child image.Rectangle
		if n.Axis == SplitColumns {
			child = image.Rect(r.Min.X+pos, r.Min.Y, r.Min.X+pos+size, r.Max.Y)
		} else {
			child = image.Rect(r.Min.X, r.Min.Y+pos, r.Max.X, r.Min.Y+pos+size)
		}
		s.layout(c, child)
		if i < count-1 {
			// A tight gutter is the last cell of one child and the first
			// of the next.
			lead := 0
			if gap == 0 {
				lead = 1
			}
			var gutter image.Rectangle
			if n.Axis == SplitColumns {
				gutter = image.Rect(child.Max.X-lead, r.Min.Y, child.Max.X+max(gap, 1), r.Max.Y)
			} else {
				gutter = image.Rect(r.Min.X, child.Max.Y-lead, r.Max.X, child.Max.Y+max(gap, 1))
			}
			s.dividers = append(s.dividers, splitDivider{node: n, index: i, bounds: gutter})
		}
		offset = end
	}
}

// SplitView draws a SplitState.
type SplitView struct {
	state       *SplitState
	pane        func(id string) View
	gutterStyle Style
	activeStyle Style
	showGutter  bool
}

// Split draws the layout in state, calling pane for the view of each pane.
// It fills the space it is given.
//
// Gutters are blank until the pointer is over one or it is being dragged,
// when a line is drawn in it (see ActiveGutterStyle). GutterStyle draws a
// line in every gutter all the time. In a Tight layout there is no gutter to
// draw in: the line-drawing characters of the panes' borders either side of
// the edge take the active style instead.
func Split(state *SplitState, pane func(id string) View) *SplitView {
	return &SplitView{
		state:       state,
		pane:        pane,
		activeStyle: NewStyle().WithForeground(ColorCyan),
	}
}

// GutterStyle draws a line in every gutter in this style.
func (v *SplitView) GutterStyle(s Style) *SplitView {
	v.gutterStyle = s
	v.showGutter = true
	return v
}

// ActiveGutterStyle sets the style of the line drawn in the gutter under the
// pointer or being dragged. The default is cyan.
func (v *SplitView) ActiveGutterStyle(s Style) *SplitView {
	v.activeStyle = s
	return v
}

func (v *SplitView) flex() int { return 1 }

func (v *SplitView) size(maxWidth, maxHeight int) (int, int) {
	return maxWidth, maxHeight
}

func (v *SplitView) render(ctx *RenderContext) {
	s := v.state
	if s == nil {
		return
	}
	s.dividers = s.dividers[:0]
	if s.Root == nil {
		return
	}
	abs := ctx.AbsoluteBounds()
	s.layout(s.Root, abs)
	// The drag and hover gutters are copies made from an earlier frame's
	// list; bring their geometry up to date, or drop them if they are gone.
	for _, d := range []**splitDivider{&s.drag, &s.hover} {
		if *d == nil {
			continue
		}
		if cur := s.findDivider(*d); cur != nil {
			(*d).bounds = cur.bounds
		} else {
			*d = nil
		}
	}

	for _, id := range s.Leaves() {
		leaf := s.Node(id)
		r := leaf.bounds
		if r.Empty() || v.pane == nil {
			continue
		}
		child := v.pane(id)
		if child == nil {
			continue
		}
		local := r.Sub(abs.Min)
		child.size(local.Dx(), local.Dy())
		child.render(ctx.SubContext(local))
	}

	for i := range s.dividers {
		d := &s.dividers[i]
		active := d.same(s.drag) || (s.drag == nil && d.same(s.hover))
		if !active && !v.showGutter {
			continue
		}
		if s.Tight {
			if active {
				restyleBorders(ctx, d.bounds.Sub(abs.Min), v.activeStyle)
			}
			continue
		}
		style := v.gutterStyle
		if active {
			style = v.activeStyle
		}
		v.drawGutter(ctx, d.bounds.Sub(abs.Min), d.node.Axis, style)
	}
}

func (s *SplitState) findDivider(d *splitDivider) *splitDivider {
	for i := range s.dividers {
		if s.dividers[i].same(d) {
			return &s.dividers[i]
		}
	}
	return nil
}

// restyleBorders gives the box-drawing characters in r the style, leaving
// everything else, such as a title in a border, as it is.
func restyleBorders(ctx *RenderContext, r image.Rectangle, style Style) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if ch := ctx.Cell(x, y).Char; ch >= 0x2500 && ch <= 0x257f {
				ctx.RestyleCell(x, y, style)
			}
		}
	}
}

func (v *SplitView) drawGutter(ctx *RenderContext, r image.Rectangle, axis SplitAxis, style Style) {
	if axis == SplitColumns {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			ctx.SetCell(r.Min.X, y, '│', style)
		}
		return
	}
	for x := r.Min.X; x < r.Max.X; x++ {
		ctx.SetCell(x, r.Min.Y, '─', style)
	}
}
