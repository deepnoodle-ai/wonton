package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/terminal"
)

// MouseEvent is a mouse event from the terminal package
type MouseEvent = terminal.MouseEvent

// MouseRegion is a clickable region from the terminal package
type MouseRegion = terminal.MouseRegion

// Mouse types from terminal
type (
	MouseButton    = terminal.MouseButton
	MouseEventType = terminal.MouseEventType
	MouseModifiers = terminal.MouseModifiers
	CursorStyle    = terminal.CursorStyle
)

// MouseButton constants
const (
	MouseButtonLeft       = terminal.MouseButtonLeft
	MouseButtonMiddle     = terminal.MouseButtonMiddle
	MouseButtonRight      = terminal.MouseButtonRight
	MouseButtonNone       = terminal.MouseButtonNone
	MouseButtonWheelUp    = terminal.MouseButtonWheelUp
	MouseButtonWheelDown  = terminal.MouseButtonWheelDown
	MouseButtonWheelLeft  = terminal.MouseButtonWheelLeft
	MouseButtonWheelRight = terminal.MouseButtonWheelRight
)

// MouseEventType constants
const (
	MousePress       = terminal.MousePress
	MouseRelease     = terminal.MouseRelease
	MouseClick       = terminal.MouseClick
	MouseDoubleClick = terminal.MouseDoubleClick
	MouseTripleClick = terminal.MouseTripleClick
	MouseDrag        = terminal.MouseDrag
	MouseDragStart   = terminal.MouseDragStart
	MouseDragEnd     = terminal.MouseDragEnd
	MouseDragCancel  = terminal.MouseDragCancel
	MouseMove        = terminal.MouseMove
	MouseEnter       = terminal.MouseEnter
	MouseLeave       = terminal.MouseLeave
	MouseScroll      = terminal.MouseScroll
)

// MouseModifiers constants
const (
	ModShift = terminal.ModShift
	ModAlt   = terminal.ModAlt
	ModCtrl  = terminal.ModCtrl
	ModMeta  = terminal.ModMeta
)

// CursorStyle constants
const (
	CursorDefault    = terminal.CursorDefault
	CursorPointer    = terminal.CursorPointer
	CursorText       = terminal.CursorText
	CursorResizeEW   = terminal.CursorResizeEW
	CursorResizeNS   = terminal.CursorResizeNS
	CursorResizeNESW = terminal.CursorResizeNESW
	CursorResizeNWSE = terminal.CursorResizeNWSE
	CursorMove       = terminal.CursorMove
	CursorNotAllowed = terminal.CursorNotAllowed
)

// ParseMouseEvent parses a mouse event from terminal input
var ParseMouseEvent = terminal.ParseMouseEvent

// MouseHandler manages mouse regions and events with advanced features.
// This is a higher-level UI component that tracks regions, handles
// click detection, drag state, and dispatches events to handlers.
type MouseHandler struct {
	regions          []*MouseRegion
	hoveredRegion    *MouseRegion
	capturedRegion   *MouseRegion
	dragRegion       *MouseRegion
	lastClickRegion  *MouseRegion
	lastClickTime    time.Time
	lastClickButton  MouseButton
	capturedButton   MouseButton
	clickCount       int
	isDragging       bool
	dragStartX       int
	dragStartY       int
	debugMode        bool
	clickSynthesized bool

	// Configuration
	DoubleClickThreshold time.Duration
	TripleClickThreshold time.Duration
	ClickMoveThreshold   int
	DragStartThreshold   int
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

func (h *MouseHandler) sortRegionsByZIndex() {
	for i := 0; i < len(h.regions); i++ {
		for j := i + 1; j < len(h.regions); j++ {
			if h.regions[i].ZIndex < h.regions[j].ZIndex {
				h.regions[i], h.regions[j] = h.regions[j], h.regions[i]
			}
		}
	}
}

func (h *MouseHandler) findRegionAt(x, y int) *MouseRegion {
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

	if event.Type == MouseClick || event.Type == MouseDoubleClick || event.Type == MouseTripleClick {
		h.clickSynthesized = true
		targetRegion := h.findRegionAt(event.X, event.Y)
		if targetRegion != nil {
			h.dispatchToRegion(targetRegion, event)
		}
		return
	}

	if h.capturedRegion != nil {
		h.handleCapturedEvent(event)
		return
	}

	targetRegion := h.findRegionAt(event.X, event.Y)
	h.handleEnterLeave(event, targetRegion)

	if targetRegion != nil {
		h.dispatchToRegion(targetRegion, event)
	}

	switch event.Type {
	case MousePress:
		h.handlePress(event, targetRegion)
	case MouseRelease:
		h.handleRelease(event, targetRegion)
	case MouseDrag:
		h.handleDrag(event, targetRegion)
	}
}

func (h *MouseHandler) handleEnterLeave(event *MouseEvent, targetRegion *MouseRegion) {
	if targetRegion != h.hoveredRegion {
		if h.hoveredRegion != nil {
			leaveEvent := *event
			leaveEvent.Type = MouseLeave
			h.dispatchToRegion(h.hoveredRegion, &leaveEvent)
		}
		if targetRegion != nil {
			enterEvent := *event
			enterEvent.Type = MouseEnter
			h.dispatchToRegion(targetRegion, &enterEvent)
		}
		h.hoveredRegion = targetRegion
	}
}

func (h *MouseHandler) handlePress(event *MouseEvent, region *MouseRegion) {
	h.clickSynthesized = false
	if region != nil {
		h.capturedRegion = region
		h.capturedButton = event.Button
		h.dragStartX = event.X
		h.dragStartY = event.Y
		h.isDragging = false
	}
}

func (h *MouseHandler) handleRelease(event *MouseEvent, _ *MouseRegion) {
	if h.capturedRegion != nil && !h.clickSynthesized {
		dx := event.X - h.dragStartX
		dy := event.Y - h.dragStartY
		moved := dx*dx + dy*dy

		if moved <= h.ClickMoveThreshold*h.ClickMoveThreshold {
			h.detectAndDispatchClick(event, h.capturedRegion)
		}

		if h.isDragging {
			h.endDrag(event, h.capturedRegion)
		}
	}
	h.capturedRegion = nil
}

func (h *MouseHandler) handleDrag(event *MouseEvent, _ *MouseRegion) {
	if h.capturedRegion != nil && !h.isDragging {
		dx := event.X - h.dragStartX
		dy := event.Y - h.dragStartY
		moved := dx*dx + dy*dy

		if moved > h.DragStartThreshold*h.DragStartThreshold {
			h.isDragging = true
			h.dragRegion = h.capturedRegion
			dragStartEvent := *event
			dragStartEvent.Type = MouseDragStart
			h.dispatchToRegion(h.capturedRegion, &dragStartEvent)
		}
	}
}

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

func (h *MouseHandler) detectAndDispatchClick(event *MouseEvent, region *MouseRegion) {
	now := event.Time
	timeSinceLastClick := now.Sub(h.lastClickTime)

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

	clickEvent := *event
	clickEvent.Button = h.capturedButton
	clickEvent.ClickCount = h.clickCount

	clickEvent.Type = MouseClick
	h.dispatchToRegion(region, &clickEvent)

	switch h.clickCount {
	case 2:
		doubleClickEvent := clickEvent
		doubleClickEvent.Type = MouseDoubleClick
		h.dispatchToRegion(region, &doubleClickEvent)
	case 3:
		tripleClickEvent := clickEvent
		tripleClickEvent.Type = MouseTripleClick
		h.dispatchToRegion(region, &tripleClickEvent)
		h.clickCount = 0
	}
}

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
		// No specific handler
	case MouseScroll:
		if region.OnScroll != nil {
			region.OnScroll(event)
		}
	}
}
