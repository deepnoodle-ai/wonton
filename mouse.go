package gooey

import (
	"fmt"
	"os"
	"time"
)

// MouseEvent represents a mouse interaction
type MouseEvent struct {
	X, Y       int
	Button     MouseButton
	Type       MouseEventType
	Modifiers  MouseModifiers
	DeltaX     int // For wheel events
	DeltaY     int // For wheel events
	Time       time.Time
	ClickCount int // 1 for single, 2 for double, 3 for triple
}

// MouseButton represents which mouse button was pressed
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonMiddle
	MouseButtonRight
	MouseButtonNone // For move/release events with no button pressed
	MouseButtonWheelUp
	MouseButtonWheelDown
	MouseButtonWheelLeft
	MouseButtonWheelRight
)

// Deprecated: Use MouseButtonLeft
const MouseLeft = MouseButtonLeft

// Deprecated: Use MouseButtonMiddle
const MouseMiddle = MouseButtonMiddle

// Deprecated: Use MouseButtonRight
const MouseRight = MouseButtonRight

// Deprecated: Use MouseButtonWheelUp
const MouseWheelUp = MouseButtonWheelUp

// Deprecated: Use MouseButtonWheelDown
const MouseWheelDown = MouseButtonWheelDown

// MouseEventType represents the type of mouse event
type MouseEventType int

const (
	MousePress MouseEventType = iota
	MouseRelease
	MouseClick
	MouseDoubleClick
	MouseTripleClick
	MouseDrag
	MouseDragStart
	MouseDragEnd
	MouseDragCancel
	MouseMove
	MouseEnter
	MouseLeave
	MouseScroll
)

// MouseModifiers represents keyboard modifiers held during mouse event
type MouseModifiers int

const (
	ModShift MouseModifiers = 1 << iota
	ModAlt
	ModCtrl
	ModMeta
)

// EnableMouseTracking enables mouse event reporting in the terminal
func (t *Terminal) EnableMouseTracking() {
	// Enable SGR extended mouse mode (supports coordinates beyond 223)
	fmt.Print("\033[?1006h")
	// Enable mouse tracking - report button press and release
	fmt.Print("\033[?1000h")
	// Enable all mouse motion tracking (including when no button is pressed)
	// This is needed for proper hover state detection
	fmt.Print("\033[?1003h")
}

// DisableMouseTracking disables mouse event reporting
func (t *Terminal) DisableMouseTracking() {
	fmt.Print("\033[?1000l")
	fmt.Print("\033[?1003l")
	fmt.Print("\033[?1006l")
}

// ParseMouseEvent parses a mouse event from terminal input
func ParseMouseEvent(seq []byte) (*MouseEvent, error) {
	// SGR format: \033[<button>;x;yM (press) or m (release)
	// Example: \033[<0;10;5M for left click at (10,5)

	if len(seq) < 6 {
		return nil, fmt.Errorf("invalid mouse sequence")
	}

	// Parse the sequence
	var button, x, y int
	var action byte

	// Look for SGR format
	if seq[0] == '<' {
		// Parse SGR format: <button>;x;y[Mm]
		end := 1
		for end < len(seq) && seq[end] != 'M' && seq[end] != 'm' {
			end++
		}
		if end >= len(seq) {
			return nil, fmt.Errorf("incomplete mouse sequence")
		}

		action = seq[end]
		coords := string(seq[1:end])

		if _, err := fmt.Sscanf(coords, "%d;%d;%d", &button, &x, &y); err != nil {
			return nil, err
		}
	} else {
		// Legacy format - limited to coordinates < 223
		if len(seq) != 3 {
			return nil, fmt.Errorf("invalid legacy mouse sequence")
		}
		button = int(seq[0]) - 32
		x = int(seq[1]) - 32
		y = int(seq[2]) - 32
		action = 'M' // Legacy is always press
	}

	// Convert to 0-based coordinates
	x--
	y--

	event := &MouseEvent{
		X:    x,
		Y:    y,
		Time: time.Now(),
	}

	// Parse modifiers (bits 2-4 of button byte)
	if button&4 != 0 {
		event.Modifiers |= ModShift
	}
	if button&8 != 0 {
		event.Modifiers |= ModAlt
	}
	if button&16 != 0 {
		event.Modifiers |= ModCtrl
	}

	// Determine button and type
	if action == 'm' {
		event.Button = MouseButtonNone
		event.Type = MouseRelease
	} else {
		baseButton := button & 3

		// Check for wheel events first
		if button&64 != 0 {
			event.Type = MouseScroll
			// Bit 0 determines up/down, bit 1 determines vertical/horizontal
			if button&2 != 0 {
				// Horizontal wheel
				if button&1 != 0 {
					event.Button = MouseButtonWheelRight
					event.DeltaX = 1
				} else {
					event.Button = MouseButtonWheelLeft
					event.DeltaX = -1
				}
			} else {
				// Vertical wheel
				if button&1 != 0 {
					event.Button = MouseButtonWheelDown
					event.DeltaY = 1
				} else {
					event.Button = MouseButtonWheelUp
					event.DeltaY = -1
				}
			}
		} else if button&32 != 0 {
			// Motion flag is set
			if baseButton == 3 {
				// No button pressed - this is just mouse movement
				event.Type = MouseMove
				event.Button = MouseButtonNone
			} else {
				// Button pressed while moving - this is a drag
				event.Type = MouseDrag
				switch baseButton {
				case 0:
					event.Button = MouseButtonLeft
				case 1:
					event.Button = MouseButtonMiddle
				case 2:
					event.Button = MouseButtonRight
				}
			}
		} else {
			// Regular press event
			event.Type = MousePress
			switch baseButton {
			case 0:
				event.Button = MouseButtonLeft
			case 1:
				event.Button = MouseButtonMiddle
			case 2:
				event.Button = MouseButtonRight
			case 3:
				event.Button = MouseButtonNone
			}
		}
	}

	return event, nil
}

// CursorStyle represents a cursor style hint
type CursorStyle int

const (
	CursorDefault CursorStyle = iota
	CursorPointer
	CursorText
	CursorResizeEW // East-West resize
	CursorResizeNS // North-South resize
	CursorResizeNESW
	CursorResizeNWSE
	CursorMove
	CursorNotAllowed
)

// MouseRegion represents a clickable area on screen
type MouseRegion struct {
	X, Y          int
	Width, Height int
	ZIndex        int
	Label         string
	CursorStyle   CursorStyle

	// Event handlers
	OnPress       func(event *MouseEvent)
	OnRelease     func(event *MouseEvent)
	OnClick       func(event *MouseEvent)
	OnDoubleClick func(event *MouseEvent)
	OnTripleClick func(event *MouseEvent)
	OnEnter       func(event *MouseEvent)
	OnLeave       func(event *MouseEvent)
	OnMove        func(event *MouseEvent)
	OnDragStart   func(event *MouseEvent)
	OnDrag        func(event *MouseEvent)
	OnDragEnd     func(event *MouseEvent)
	OnScroll      func(event *MouseEvent)

	// Legacy handlers for backwards compatibility
	Handler      func(event *MouseEvent) // Maps to OnClick
	HoverHandler func(hovering bool)     // Maps to OnEnter/OnLeave
}

// Contains checks if a point is within the region
func (r *MouseRegion) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width &&
		y >= r.Y && y < r.Y+r.Height
}

// MouseHandler manages mouse regions and events with advanced features
type MouseHandler struct {
	regions         []*MouseRegion
	hoveredRegion   *MouseRegion
	capturedRegion  *MouseRegion
	dragRegion      *MouseRegion
	lastClickRegion *MouseRegion
	lastClickTime   time.Time
	lastClickButton MouseButton
	lastClickX      int
	lastClickY      int
	capturedButton  MouseButton // Button that was pressed (for click detection)
	clickCount      int
	isDragging      bool
	dragStartX      int
	dragStartY      int
	debugMode       bool

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
func (h *MouseHandler) handleRelease(event *MouseEvent, region *MouseRegion) {
	// Check if this is a click (release on same region as press, minimal movement)
	if h.capturedRegion != nil {
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
func (h *MouseHandler) handleDrag(event *MouseEvent, region *MouseRegion) {
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

	switch h.clickCount {
	case 1:
		clickEvent.Type = MouseClick
	case 2:
		clickEvent.Type = MouseDoubleClick
	case 3:
		clickEvent.Type = MouseTripleClick
		h.clickCount = 0 // Reset after triple
	default:
		clickEvent.Type = MouseClick
		h.clickCount = 1
	}

	h.dispatchToRegion(region, &clickEvent)
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
		} else if region.Handler != nil {
			// Legacy compatibility
			region.Handler(event)
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
		if region.HoverHandler != nil {
			// Legacy compatibility
			region.HoverHandler(true)
		}
	case MouseLeave:
		if region.OnLeave != nil {
			region.OnLeave(event)
		}
		if region.HoverHandler != nil {
			// Legacy compatibility
			region.HoverHandler(false)
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
