package tui

import "image"

// View is the core interface for declarative UI.
// Views form a tree structure where containers measure and position children.
//
// Methods are unexported - users compose views using builder functions like
// Text(), VStack(), HStack(), etc. rather than implementing View directly.
type View interface {
	// render draws this view to the given frame within the specified bounds.
	// The bounds are in the frame's coordinate system.
	render(frame RenderFrame, bounds image.Rectangle)

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

func (e emptyView) render(frame RenderFrame, bounds image.Rectangle) {}
func (e emptyView) size(maxWidth, maxHeight int) (int, int)          { return 0, 0 }

// spacerView expands to fill available space
type spacerView struct {
	flexFactor int
	minWidth   int
	minHeight  int
}

// Spacer returns a flexible view that expands to fill available space.
// In a VStack, it fills vertical space. In an HStack, it fills horizontal space.
func Spacer() *spacerView {
	return &spacerView{flexFactor: 1}
}

// Flex sets the flex factor for this spacer.
// A spacer with flex(2) will get twice as much space as one with flex(1).
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

func (s *spacerView) render(frame RenderFrame, bounds image.Rectangle) {
	// Spacer renders nothing - it just takes up space
}

func (s *spacerView) size(maxWidth, maxHeight int) (int, int) {
	// Return minimum size - the layout will expand this
	return s.minWidth, s.minHeight
}
