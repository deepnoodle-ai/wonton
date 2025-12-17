package terminal

import (
	"fmt"
	"os"
	"time"
)

// MouseHandler manages mouse regions and dispatches events to them.
//
// MouseHandler provides a higher-level abstraction over raw mouse events,
// implementing:
//   - Region-based event routing (only regions under the cursor receive events)
//   - Click detection (press + release without movement)
//   - Double-click and triple-click detection with configurable thresholds
//   - Drag detection with configurable movement threshold
//   - Enter/leave events when mouse moves between regions
//   - Z-index based layering for overlapping regions
//   - Event capture during drag operations
//
// # Basic Usage
//
//	handler := terminal.NewMouseHandler()
//
//	// Add a clickable region
//	region := &terminal.MouseRegion{
//	    X: 10, Y: 5,
//	    Width: 15, Height: 1,
//	    OnClick: func(e *terminal.MouseEvent) {
//	        fmt.Println("Button clicked!")
//	    },
//	}
//	handler.AddRegion(region)
//
//	// Route mouse events to regions
//	event := &terminal.MouseEvent{...}
//	handler.HandleEvent(event)
//
// # Event Flow
//
// When a mouse event is received:
//  1. The handler finds the topmost region (highest ZIndex) under the cursor
//  2. If the region changed, OnLeave is called on the old region and OnEnter on the new
//  3. The event is dispatched to the appropriate handler (OnClick, OnDrag, etc.)
//  4. Click detection tracks press/release to synthesize click events
//  5. Drag detection starts when movement exceeds DragStartThreshold
//
// # Configuration
//
// The handler has configurable thresholds:
//   - DoubleClickThreshold: max time between clicks for double-click (default 500ms)
//   - TripleClickThreshold: max time between clicks for triple-click (default 500ms)
//   - ClickMoveThreshold: max pixel movement to count as click (default 2)
//   - DragStartThreshold: min pixel movement to start drag (default 5)
type MouseHandler struct {
	regions         []*MouseRegion
	hoveredRegion   *MouseRegion
	capturedRegion  *MouseRegion
	dragRegion      *MouseRegion
	lastClickRegion *MouseRegion
	lastClickTime   time.Time
	lastClickButton MouseButton
	capturedButton  MouseButton // Button that was pressed (for click detection)
	clickCount      int
	isDragging      bool
	dragStartX      int
	dragStartY      int
	debugMode       bool
	// clickSynthesized tracks whether a MouseClick event was received for the current
	// press cycle. When using Runtime, clicks are pre-synthesized and sent BEFORE the
	// Release event. This flag prevents handleRelease from creating a duplicate click.
	// Reset to false on each MousePress, set to true when MouseClick is received.
	clickSynthesized bool

	// Configuration
	DoubleClickThreshold time.Duration // Maximum time between clicks for double-click
	TripleClickThreshold time.Duration // Maximum time between clicks for triple-click
	ClickMoveThreshold   int           // Maximum pixel movement to still count as click
	DragStartThreshold   int           // Minimum pixel movement to start drag
}

// NewMouseHandler creates a new mouse handler
func NewMouseHandler() *MouseHandler {
	return &MouseHandler{
		regions:              make([]*MouseRegion, 0),
		DoubleClickThreshold: 500 * time.Millisecond,
		TripleClickThreshold: 500 * time.Millisecond,
		ClickMoveThreshold:   2,
		DragStartThreshold:   5,
	}
}

// EnableDebug enables debug logging for mouse events
func (h *MouseHandler) EnableDebug() {
	h.debugMode = true
}

// DisableDebug disables debug logging
func (h *MouseHandler) DisableDebug() {
	h.debugMode = false
}

// AddRegion adds a clickable region
func (h *MouseHandler) AddRegion(region *MouseRegion) {
	h.regions = append(h.regions, region)
	h.sortRegionsByZIndex()
}

// RemoveRegion removes a specific region
func (h *MouseHandler) RemoveRegion(region *MouseRegion) {
	for i, r := range h.regions {
		if r == region {
			h.regions = append(h.regions[:i], h.regions[i+1:]...)
			if h.hoveredRegion == region {
				h.hoveredRegion = nil
			}
			if h.capturedRegion == region {
				h.capturedRegion = nil
			}
			if h.dragRegion == region {
				h.dragRegion = nil
			}
			break
		}
	}
}

// ClearRegions removes all regions
func (h *MouseHandler) ClearRegions() {
	h.regions = h.regions[:0]
	h.hoveredRegion = nil
	h.capturedRegion = nil
	h.dragRegion = nil
	h.isDragging = false
}

// sortRegionsByZIndex sorts regions by z-index (highest first for hit-testing)
func (h *MouseHandler) sortRegionsByZIndex() {
	// Simple bubble sort - regions list is typically small
	for i := 0; i < len(h.regions); i++ {
		for j := i + 1; j < len(h.regions); j++ {
			if h.regions[i].ZIndex < h.regions[j].ZIndex {
				h.regions[i], h.regions[j] = h.regions[j], h.regions[i]
			}
		}
	}
}

// findRegionAt finds the topmost region at given coordinates
func (h *MouseHandler) findRegionAt(x, y int) *MouseRegion {
	// Regions are sorted by z-index (highest first)
	for _, region := range h.regions {
		if region.Contains(x, y) {
			return region
		}
	}
	return nil
}

// HandleEvent processes a mouse event with full state machine
func (h *MouseHandler) HandleEvent(event *MouseEvent) {
	if h.debugMode {
		fmt.Fprintf(os.Stderr, "[Mouse] Event: Type=%v Button=%v X=%d Y=%d Mods=%v\n",
			event.Type, event.Button, event.X, event.Y, event.Modifiers)
	}

	// Handle pre-synthesized click events (from Runtime or other sources).
	// These arrive BEFORE MouseRelease in the event stream. We dispatch them directly
	// and set clickSynthesized=true so that handleRelease knows to skip its own
	// click synthesis (which would cause duplicate clicks).
	if event.Type == MouseClick || event.Type == MouseDoubleClick || event.Type == MouseTripleClick {
		h.clickSynthesized = true
		targetRegion := h.findRegionAt(event.X, event.Y)
		if targetRegion != nil {
			h.dispatchToRegion(targetRegion, event)
		}
		return
	}

	// If we have a captured region, route all events to it
	if h.capturedRegion != nil {
		h.handleCapturedEvent(event)
		return
	}

	// Find target region
	targetRegion := h.findRegionAt(event.X, event.Y)

	// Handle enter/leave
	h.handleEnterLeave(event, targetRegion)

	// Route event to target region
	if targetRegion != nil {
		h.dispatchToRegion(targetRegion, event)
	}

	// Update state based on event type
	switch event.Type {
	case MousePress:
		h.handlePress(event, targetRegion)
	case MouseRelease:
		h.handleRelease(event, targetRegion)
	case MouseDrag:
		h.handleDrag(event, targetRegion)
	}
}

// handleEnterLeave manages enter/leave events
func (h *MouseHandler) handleEnterLeave(event *MouseEvent, targetRegion *MouseRegion) {
	if targetRegion != h.hoveredRegion {
		// Leave previous region
		if h.hoveredRegion != nil {
			leaveEvent := *event
			leaveEvent.Type = MouseLeave
			h.dispatchToRegion(h.hoveredRegion, &leaveEvent)
		}

		// Enter new region
		if targetRegion != nil {
			enterEvent := *event
			enterEvent.Type = MouseEnter
			h.dispatchToRegion(targetRegion, &enterEvent)
		}

		h.hoveredRegion = targetRegion
	}
}

// handlePress handles press events and initiates capture if needed
func (h *MouseHandler) handlePress(event *MouseEvent, region *MouseRegion) {
	// Reset clickSynthesized for the new press/release cycle.
	// If Runtime sends a pre-synthesized Click, it will arrive before Release
	// and set this back to true.
	h.clickSynthesized = false

	if region != nil {
		// Capture the region for drag/release tracking
		h.capturedRegion = region
		h.capturedButton = event.Button
		h.dragStartX = event.X
		h.dragStartY = event.Y
		h.isDragging = false
	}
}

// handleRelease handles release events and generates click events
func (h *MouseHandler) handleRelease(event *MouseEvent, _ *MouseRegion) {
	// Synthesize a click if:
	// 1. We have a captured region (from the press)
	// 2. No pre-synthesized click was received (clickSynthesized is false)
	// 3. Mouse didn't move much (within ClickMoveThreshold)
	//
	// When using Runtime, clickSynthesized will be true because Runtime sends
	// MouseClick BEFORE MouseRelease, and HandleEvent sets the flag.
	// This prevents duplicate click dispatch.
	if h.capturedRegion != nil && !h.clickSynthesized {
		dx := event.X - h.dragStartX
		dy := event.Y - h.dragStartY
		moved := dx*dx + dy*dy

		if moved <= h.ClickMoveThreshold*h.ClickMoveThreshold {
			// This is a click!
			h.detectAndDispatchClick(event, h.capturedRegion)
		}

		// End drag if active
		if h.isDragging {
			h.endDrag(event, h.capturedRegion)
		}
	}

	// Release capture
	h.capturedRegion = nil
}

// handleDrag manages drag state machine
func (h *MouseHandler) handleDrag(event *MouseEvent, _ *MouseRegion) {
	if h.capturedRegion != nil && !h.isDragging {
		dx := event.X - h.dragStartX
		dy := event.Y - h.dragStartY
		moved := dx*dx + dy*dy

		if moved > h.DragStartThreshold*h.DragStartThreshold {
			// Start drag
			h.isDragging = true
			h.dragRegion = h.capturedRegion
			dragStartEvent := *event
			dragStartEvent.Type = MouseDragStart
			h.dispatchToRegion(h.capturedRegion, &dragStartEvent)
		}
	}
}

// handleCapturedEvent routes events to captured region
func (h *MouseHandler) handleCapturedEvent(event *MouseEvent) {
	switch event.Type {
	case MouseRelease:
		h.handleRelease(event, h.capturedRegion)
	case MouseDrag:
		if h.isDragging {
			h.dispatchToRegion(h.capturedRegion, event)
		} else {
			h.handleDrag(event, h.capturedRegion)
		}
	default:
		h.dispatchToRegion(h.capturedRegion, event)
	}
}

// endDrag ends an active drag
func (h *MouseHandler) endDrag(event *MouseEvent, region *MouseRegion) {
	dragEndEvent := *event
	dragEndEvent.Type = MouseDragEnd
	h.dispatchToRegion(region, &dragEndEvent)
	h.isDragging = false
	h.dragRegion = nil
}

// CancelDrag cancels an active drag (e.g., on Escape key)
func (h *MouseHandler) CancelDrag() {
	if h.isDragging && h.dragRegion != nil {
		cancelEvent := &MouseEvent{
			Type: MouseDragCancel,
			X:    h.dragStartX,
			Y:    h.dragStartY,
			Time: time.Now(),
		}
		h.dispatchToRegion(h.dragRegion, cancelEvent)
		h.isDragging = false
		h.dragRegion = nil
		h.capturedRegion = nil
	}
}

// detectAndDispatchClick detects double/triple clicks and dispatches appropriate event
func (h *MouseHandler) detectAndDispatchClick(event *MouseEvent, region *MouseRegion) {
	now := event.Time
	timeSinceLastClick := now.Sub(h.lastClickTime)

	// Check if this continues a multi-click sequence
	// Use capturedButton (the button that was pressed) rather than event.Button (which is None on release)
	if region == h.lastClickRegion &&
		h.capturedButton == h.lastClickButton &&
		timeSinceLastClick <= h.DoubleClickThreshold {
		h.clickCount++
	} else {
		h.clickCount = 1
	}

	h.lastClickTime = now
	h.lastClickRegion = region
	h.lastClickButton = h.capturedButton

	// Create click event with the button that was originally pressed
	clickEvent := *event
	clickEvent.Button = h.capturedButton
	clickEvent.ClickCount = h.clickCount

	// Always dispatch a regular click first - this ensures OnClick handlers
	// are called for every click, even during double/triple click sequences
	clickEvent.Type = MouseClick
	h.dispatchToRegion(region, &clickEvent)

	// Then dispatch double/triple click events if applicable
	switch h.clickCount {
	case 2:
		doubleClickEvent := clickEvent
		doubleClickEvent.Type = MouseDoubleClick
		h.dispatchToRegion(region, &doubleClickEvent)
	case 3:
		tripleClickEvent := clickEvent
		tripleClickEvent.Type = MouseTripleClick
		h.dispatchToRegion(region, &tripleClickEvent)
		h.clickCount = 0 // Reset after triple
	}
}

// dispatchToRegion dispatches event to region's appropriate handler
func (h *MouseHandler) dispatchToRegion(region *MouseRegion, event *MouseEvent) {
	if region == nil {
		return
	}

	switch event.Type {
	case MousePress:
		if region.OnPress != nil {
			region.OnPress(event)
		}
	case MouseRelease:
		if region.OnRelease != nil {
			region.OnRelease(event)
		}
	case MouseClick:
		if region.OnClick != nil {
			region.OnClick(event)
		}
	case MouseDoubleClick:
		if region.OnDoubleClick != nil {
			region.OnDoubleClick(event)
		}
	case MouseTripleClick:
		if region.OnTripleClick != nil {
			region.OnTripleClick(event)
		}
	case MouseEnter:
		if region.OnEnter != nil {
			region.OnEnter(event)
		}
	case MouseLeave:
		if region.OnLeave != nil {
			region.OnLeave(event)
		}
	case MouseMove:
		if region.OnMove != nil {
			region.OnMove(event)
		}
	case MouseDragStart:
		if region.OnDragStart != nil {
			region.OnDragStart(event)
		}
	case MouseDrag:
		if region.OnDrag != nil {
			region.OnDrag(event)
		}
	case MouseDragEnd:
		if region.OnDragEnd != nil {
			region.OnDragEnd(event)
		}
	case MouseDragCancel:
		// No specific handler, but could be added
	case MouseScroll:
		if region.OnScroll != nil {
			region.OnScroll(event)
		}
	}
}
