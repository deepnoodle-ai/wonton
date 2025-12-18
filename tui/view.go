package tui

// View is the core interface for declarative UI.
// Views form a tree structure where containers measure and position children.
//
// Methods are unexported - users compose views using builder functions like
// Text(), Stack(), Group(), etc. rather than implementing View directly.
//
// For custom focusable components, use FocusableView() to wrap any view
// with focus handling capabilities.
type View interface {
	// render draws this view using the provided render context.
	// The context provides both drawing capabilities and contextual information
	// like the animation frame counter.
	render(ctx *RenderContext)

	// size returns the preferred size of this view given maximum constraints.
	// maxWidth/maxHeight of 0 means unconstrained.
	size(maxWidth, maxHeight int) (width, height int)
}

// Flexible is implemented by views that can expand to fill available space.
// Views implementing Flexible will share remaining space proportionally
// based on their flex factor.
type Flexible interface {
	flex() int
}

// emptyView renders nothing
type emptyView struct{}

// Empty returns a view that renders nothing.
// Useful for conditional rendering.
func Empty() View {
	return emptyView{}
}

func (e emptyView) render(ctx *RenderContext)               {}
func (e emptyView) size(maxWidth, maxHeight int) (int, int) { return 0, 0 }

// spacerView expands to fill available space
type spacerView struct {
	flexFactor int
	minWidth   int
	minHeight  int
}

// Spacer returns a flexible view that expands to fill available space.
// In a Stack, it fills vertical space. In a Group, it fills horizontal space.
//
// Spacers are useful for pushing content to edges or distributing space:
//
//	Stack(
//	    Text("Top"),
//	    Spacer(),        // Pushes footer to bottom
//	    Text("Bottom"),
//	)
//
//	Group(
//	    Text("Left"),
//	    Spacer(),        // Pushes right content to edge
//	    Text("Right"),
//	)
func Spacer() *spacerView {
	return &spacerView{flexFactor: 1}
}

// Flex sets the flex factor for this spacer, controlling space distribution.
// A spacer with Flex(2) will get twice as much space as one with Flex(1).
//
// Example:
//
//	Stack(
//	    Text("Header"),
//	    Spacer().Flex(1),  // Gets 1/3 of remaining space
//	    Text("Middle"),
//	    Spacer().Flex(2),  // Gets 2/3 of remaining space
//	    Text("Footer"),
//	)
func (s *spacerView) Flex(factor int) *spacerView {
	s.flexFactor = factor
	return s
}

// MinWidth sets the minimum width for the spacer.
func (s *spacerView) MinWidth(w int) *spacerView {
	s.minWidth = w
	return s
}

// MinHeight sets the minimum height for the spacer.
func (s *spacerView) MinHeight(h int) *spacerView {
	s.minHeight = h
	return s
}

func (s *spacerView) flex() int {
	return s.flexFactor
}

func (s *spacerView) render(ctx *RenderContext) {
	// Spacer renders nothing - it just takes up space
}

func (s *spacerView) size(maxWidth, maxHeight int) (int, int) {
	// Return minimum size - the layout will expand this
	return s.minWidth, s.minHeight
}
