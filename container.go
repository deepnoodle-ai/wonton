package gooey

import (
	"image"
)

// Container is a general-purpose composable widget that can contain child widgets.
// It uses a LayoutManager to position and size its children, enabling complex
// nested layouts similar to web design patterns.
//
// Example usage:
//
//	container := NewContainer(NewVBoxLayout(2)) // vertical layout with 2px spacing
//	container.AddChild(button1)
//	container.AddChild(button2)
//	container.AddChild(inputField)
type Container struct {
	BaseWidget
	children         []ComposableWidget
	layout           LayoutManager
	style            Style
	borderStyle      *BorderStyle
	drawBorder       bool
	contentBounds    image.Rectangle  // Bounds available for content (inside border/padding)
	hoveredChild     ComposableWidget // Track which child is currently hovered
	lastConstraints  SizeConstraints  // Constraints from the last Measure pass
	unregisterResize func()           // Cleanup function for resize callback (only for root containers)
}

// Measure calculates the size of the container given the constraints.
// This implements the Measurable interface.
func (c *Container) Measure(constraints SizeConstraints) image.Point {
	c.lastConstraints = constraints

	// Calculate padding and border size
	params := c.GetLayoutParams()
	paddingX := params.PaddingLeft + params.PaddingRight
	paddingY := params.PaddingTop + params.PaddingBottom
	if c.drawBorder {
		paddingX += 2
		paddingY += 2
	}

	// Reduce constraints by padding/border for content measurement
	contentConstraints := constraints
	if contentConstraints.HasMaxWidth() {
		contentConstraints.MaxWidth -= paddingX
		if contentConstraints.MaxWidth < 0 {
			contentConstraints.MaxWidth = 0
		}
	}
	if contentConstraints.HasMaxHeight() {
		contentConstraints.MaxHeight -= paddingY
		if contentConstraints.MaxHeight < 0 {
			contentConstraints.MaxHeight = 0
		}
	}
	// Adjust min constraints as well? Yes, usually.
	contentConstraints.MinWidth -= paddingX
	if contentConstraints.MinWidth < 0 {
		contentConstraints.MinWidth = 0
	}
	contentConstraints.MinHeight -= paddingY
	if contentConstraints.MinHeight < 0 {
		contentConstraints.MinHeight = 0
	}

	// Filter to visible children
	visibleChildren := make([]ComposableWidget, 0, len(c.children))
	for _, child := range c.children {
		if child.IsVisible() {
			visibleChildren = append(visibleChildren, child)
		}
	}

	var size image.Point
	if clm, ok := c.layout.(ConstraintLayoutManager); ok {
		size = clm.Measure(visibleChildren, contentConstraints)
	} else {
		// Legacy fallback: use preferred size and clamp
		if c.layout != nil {
			size = c.layout.CalculatePreferredSize(visibleChildren)
		}
		size = ApplyConstraints(size, contentConstraints)
	}

	// Add padding/border back to result
	size.X += paddingX
	size.Y += paddingY

	// Ensure result respects original min constraints (ApplyConstraints does max too)
	return ApplyConstraints(size, constraints)
}

// NewContainer creates a new container with the specified layout manager
func NewContainer(layout LayoutManager) *Container {
	return &Container{
		BaseWidget: NewBaseWidget(),
		children:   make([]ComposableWidget, 0),
		layout:     layout,
		style:      NewStyle(),
		drawBorder: false,
	}
}

// NewContainerWithBorder creates a new container with a border
func NewContainerWithBorder(layout LayoutManager, borderStyle *BorderStyle) *Container {
	c := NewContainer(layout)
	c.SetBorder(borderStyle)
	return c
}

// SetBorder sets the border style and enables border drawing
func (c *Container) SetBorder(borderStyle *BorderStyle) {
	c.borderStyle = borderStyle
	c.drawBorder = borderStyle != nil
	c.MarkDirty()
	c.updateContentBounds()
}

// SetStyle sets the container's style (background color, etc.)
func (c *Container) SetStyle(style Style) {
	c.style = style
	c.MarkDirty()
}

// AddChild adds a child widget to the container
func (c *Container) AddChild(child ComposableWidget) {
	child.SetParent(c)
	c.children = append(c.children, child)
	child.Init()
	c.MarkDirty()
	c.relayout()
}

// RemoveChild removes a child widget from the container
func (c *Container) RemoveChild(child ComposableWidget) {
	for i, w := range c.children {
		if w == child {
			// Call Destroy lifecycle method
			w.Destroy()
			w.SetParent(nil)
			// Remove from slice
			c.children = append(c.children[:i], c.children[i+1:]...)
			c.MarkDirty()
			c.relayout()
			return
		}
	}
}

// RemoveChildAt removes the child at the specified index
func (c *Container) RemoveChildAt(index int) {
	if index >= 0 && index < len(c.children) {
		child := c.children[index]
		child.Destroy()
		child.SetParent(nil)
		c.children = append(c.children[:index], c.children[index+1:]...)
		c.MarkDirty()
		c.relayout()
	}
}

// Clear removes all children from the container
func (c *Container) Clear() {
	for _, child := range c.children {
		child.Destroy()
		child.SetParent(nil)
	}
	c.children = make([]ComposableWidget, 0)
	c.MarkDirty()
}

// GetChildren returns a copy of the children slice
func (c *Container) GetChildren() []ComposableWidget {
	result := make([]ComposableWidget, len(c.children))
	copy(result, c.children)
	return result
}

// GetChildCount returns the number of children
func (c *Container) GetChildCount() int {
	return len(c.children)
}

// SetLayout changes the layout manager and triggers re-layout
func (c *Container) SetLayout(layout LayoutManager) {
	c.layout = layout
	c.MarkDirty()
	c.relayout()
}

// SetBounds overrides BaseWidget.SetBounds to trigger re-layout
func (c *Container) SetBounds(bounds image.Rectangle) {
	if c.bounds != bounds {
		c.bounds = bounds
		c.updateContentBounds()
		c.MarkDirty()
		c.relayout()
	}
}

// updateContentBounds calculates the content bounds (inside border/padding)
func (c *Container) updateContentBounds() {
	c.contentBounds = c.bounds

	// Account for border
	if c.drawBorder && c.borderStyle != nil {
		c.contentBounds = image.Rect(
			c.bounds.Min.X+1,
			c.bounds.Min.Y+1,
			c.bounds.Max.X-1,
			c.bounds.Max.Y-1,
		)
	}

	// Account for padding from layout params
	params := c.GetLayoutParams()
	c.contentBounds = image.Rect(
		c.contentBounds.Min.X+params.PaddingLeft,
		c.contentBounds.Min.Y+params.PaddingTop,
		c.contentBounds.Max.X-params.PaddingRight,
		c.contentBounds.Max.Y-params.PaddingBottom,
	)
}

// relayout triggers the layout manager to reposition children
func (c *Container) relayout() {
	if c.layout != nil && len(c.children) > 0 {
		// Filter to visible children only
		visibleChildren := make([]ComposableWidget, 0, len(c.children))
		for _, child := range c.children {
			if child.IsVisible() {
				visibleChildren = append(visibleChildren, child)
			}
		}

		if clm, ok := c.layout.(ConstraintLayoutManager); ok {
			// Recalculate content constraints from lastConstraints or synthesize them
			contentConstraints := c.lastConstraints

			// If we have no prior constraints (e.g. legacy SetBounds called directly),
			// we must synthesize constraints from the current content bounds.
			// This ensures that constraint-based layouts still work even if Measure() wasn't called.
			if !contentConstraints.HasMaxWidth() && !contentConstraints.HasMaxHeight() && contentConstraints.MinWidth == 0 && contentConstraints.MinHeight == 0 {
				width := c.contentBounds.Dx()
				height := c.contentBounds.Dy()
				contentConstraints = SizeConstraints{
					MinWidth:  width,
					MaxWidth:  width,
					MinHeight: height,
					MaxHeight: height,
				}
			} else {
				// If we have stored constraints, we need to reduce them by padding/border again
				// because Measure() was called on the Container, but LayoutWithConstraints()
				// expects constraints for the CONTENT area.
				params := c.GetLayoutParams()
				paddingX := params.PaddingLeft + params.PaddingRight
				paddingY := params.PaddingTop + params.PaddingBottom
				if c.drawBorder {
					paddingX += 2
					paddingY += 2
				}

				if contentConstraints.HasMaxWidth() {
					contentConstraints.MaxWidth -= paddingX
					if contentConstraints.MaxWidth < 0 {
						contentConstraints.MaxWidth = 0
					}
				}
				if contentConstraints.HasMaxHeight() {
					contentConstraints.MaxHeight -= paddingY
					if contentConstraints.MaxHeight < 0 {
						contentConstraints.MaxHeight = 0
					}
				}
				contentConstraints.MinWidth -= paddingX
				if contentConstraints.MinWidth < 0 {
					contentConstraints.MinWidth = 0
				}
				contentConstraints.MinHeight -= paddingY
				if contentConstraints.MinHeight < 0 {
					contentConstraints.MinHeight = 0
				}
			}

			clm.LayoutWithConstraints(visibleChildren, contentConstraints, c.contentBounds)
		} else {
			c.layout.Layout(c.contentBounds, visibleChildren)
		}
	}
}

// GetMinSize returns the minimum size needed for this container
func (c *Container) GetMinSize() image.Point {
	minSize := image.Point{X: 0, Y: 0}

	if c.layout != nil && len(c.children) > 0 {
		// Filter to visible children
		visibleChildren := make([]ComposableWidget, 0, len(c.children))
		for _, child := range c.children {
			if child.IsVisible() {
				visibleChildren = append(visibleChildren, child)
			}
		}
		minSize = c.layout.CalculateMinSize(visibleChildren)
	}

	// Add border size
	if c.drawBorder {
		minSize.X += 2
		minSize.Y += 2
	}

	// Add padding
	params := c.GetLayoutParams()
	minSize.X += params.PaddingLeft + params.PaddingRight
	minSize.Y += params.PaddingTop + params.PaddingBottom

	return minSize
}

// GetPreferredSize returns the preferred size for this container
func (c *Container) GetPreferredSize() image.Point {
	preferredSize := image.Point{X: 0, Y: 0}

	if c.layout != nil && len(c.children) > 0 {
		// Filter to visible children
		visibleChildren := make([]ComposableWidget, 0, len(c.children))
		for _, child := range c.children {
			if child.IsVisible() {
				visibleChildren = append(visibleChildren, child)
			}
		}
		preferredSize = c.layout.CalculatePreferredSize(visibleChildren)
	}

	// Add border size
	if c.drawBorder {
		preferredSize.X += 2
		preferredSize.Y += 2
	}

	// Add padding
	params := c.GetLayoutParams()
	preferredSize.X += params.PaddingLeft + params.PaddingRight
	preferredSize.Y += params.PaddingTop + params.PaddingBottom

	return preferredSize
}

// Draw renders the container and all its children
func (c *Container) Draw(frame RenderFrame) {
	if !c.visible {
		return
	}

	// Create a SubFrame for this container's bounds to establish our coordinate space.
	// This ensures the container draws correctly even when positioned away from (0,0).
	// We convert our absolute bounds to be relative to the frame we received.
	frameBounds := frame.GetBounds()
	relativeContainerBounds := c.bounds.Sub(frameBounds.Min)
	containerFrame := frame.SubFrame(relativeContainerBounds)

	// From here on, all drawing uses coordinates relative to the container (0,0 origin)
	width := c.bounds.Dx()
	height := c.bounds.Dy()

	// Draw background if style has background color
	if c.style.Background != ColorDefault || c.style.BgRGB != nil {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				containerFrame.SetCell(x, y, ' ', c.style)
			}
		}
	}

	// Draw border if enabled
	if c.drawBorder && c.borderStyle != nil {
		c.drawBorderLines(containerFrame, width, height)
	}

	// Draw all visible children
	// Each child gets its own SubFrame positioned at its bounds for proper clipping
	for _, child := range c.children {
		if child.IsVisible() {
			childBounds := child.GetBounds()
			// Convert child bounds to be relative to this container's bounds
			// SubFrame expects coordinates relative to the current frame
			relativeBounds := childBounds.Sub(c.bounds.Min)
			// Clip to content bounds (also made relative)
			relativeContent := c.contentBounds.Sub(c.bounds.Min)
			childFrame := containerFrame.SubFrame(relativeBounds.Intersect(relativeContent))
			child.Draw(childFrame)
			child.ClearDirty()
		}
	}

	c.ClearDirty()
}

// drawBorderLines draws the border around the container using relative coordinates
func (c *Container) drawBorderLines(frame RenderFrame, width, height int) {
	style := c.style

	// Convert border style strings to runes
	topLeft := []rune(c.borderStyle.TopLeft)[0]
	topRight := []rune(c.borderStyle.TopRight)[0]
	bottomLeft := []rune(c.borderStyle.BottomLeft)[0]
	bottomRight := []rune(c.borderStyle.BottomRight)[0]
	horizontal := []rune(c.borderStyle.Horizontal)[0]
	vertical := []rune(c.borderStyle.Vertical)[0]

	// Top border
	frame.SetCell(0, 0, topLeft, style)
	for x := 1; x < width-1; x++ {
		frame.SetCell(x, 0, horizontal, style)
	}
	frame.SetCell(width-1, 0, topRight, style)

	// Sides
	for y := 1; y < height-1; y++ {
		frame.SetCell(0, y, vertical, style)
		frame.SetCell(width-1, y, vertical, style)
	}

	// Bottom border
	frame.SetCell(0, height-1, bottomLeft, style)
	for x := 1; x < width-1; x++ {
		frame.SetCell(x, height-1, horizontal, style)
	}
	frame.SetCell(width-1, height-1, bottomRight, style)
}

// HandleKey handles keyboard events and delegates to focused child
func (c *Container) HandleKey(event KeyEvent) bool {
	if !c.visible {
		return false
	}

	// Iterate through children in reverse order (top to bottom in z-order)
	// to give topmost widgets first chance to handle events
	for i := len(c.children) - 1; i >= 0; i-- {
		child := c.children[i]
		if child.IsVisible() && child.HandleKey(event) {
			return true
		}
	}

	return false
}

// HandleMouse handles mouse events and delegates to children.
// Note: MouseClick events are synthesized by the Runtime from Press/Release pairs,
// so this method receives ready-to-use click events.
func (c *Container) HandleMouse(event MouseEvent) bool {
	if !c.visible {
		return false
	}

	// Check if event is within container bounds
	bounds := c.GetBounds()
	if event.X < bounds.Min.X || event.X >= bounds.Max.X ||
		event.Y < bounds.Min.Y || event.Y >= bounds.Max.Y {
		// Clear hovered child if mouse leaves container
		if c.hoveredChild != nil {
			if mouseAware, ok := c.hoveredChild.(MouseAware); ok {
				mouseAware.HandleMouse(event)
			}
			c.hoveredChild = nil
		}
		return false
	}

	var targetChild ComposableWidget

	// Find the child under the mouse
	for i := len(c.children) - 1; i >= 0; i-- {
		child := c.children[i]
		if !child.IsVisible() {
			continue
		}

		childBounds := child.GetBounds()
		if event.X >= childBounds.Min.X && event.X < childBounds.Max.X &&
			event.Y >= childBounds.Min.Y && event.Y < childBounds.Max.Y {
			targetChild = child
			break
		}
	}

	// Handle hover state changes
	if targetChild != c.hoveredChild {
		if c.hoveredChild != nil {
			if mouseAware, ok := c.hoveredChild.(MouseAware); ok {
				// Send event to clear hover state
				mouseAware.HandleMouse(event)
			}
		}
		c.hoveredChild = targetChild
	}

	// Forward event to the target child
	if targetChild != nil {
		if mouseAware, ok := targetChild.(MouseAware); ok {
			return mouseAware.HandleMouse(event)
		}
	}

	return targetChild != nil
}

// Init initializes the container and all its children
func (c *Container) Init() {
	c.BaseWidget.Init()
	for _, child := range c.children {
		child.Init()
	}
}

// Destroy cleans up the container and all its children
func (c *Container) Destroy() {
	// Unregister resize callback if registered
	if c.unregisterResize != nil {
		c.unregisterResize()
		c.unregisterResize = nil
	}

	for _, child := range c.children {
		child.Destroy()
	}
	c.children = nil
	c.BaseWidget.Destroy()
}

// WatchResize enables automatic resize handling for this container.
// This should only be called on root-level containers that are directly
// rendered to the terminal. When the terminal resizes, the container's
// bounds will be automatically updated and children will be re-laid out.
//
// Example:
//
//	container := NewContainer(layout)
//	container.SetBounds(image.Rect(0, 0, width, height))
//	container.WatchResize(terminal)
//	defer container.Destroy() // Automatically unregisters resize handler
func (c *Container) WatchResize(terminal *Terminal) {
	// Unregister any existing handler
	if c.unregisterResize != nil {
		c.unregisterResize()
	}

	c.unregisterResize = terminal.OnResize(func(width, height int) {
		// Update container bounds to fill the terminal
		c.SetBounds(image.Rect(0, 0, width, height))
		c.MarkDirty()
	})
}
