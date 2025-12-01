package terminal

import (
	"fmt"
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

// Timestamp implements the Event interface
func (e MouseEvent) Timestamp() time.Time {
	return e.Time
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

// ParseMouseEvent parses a mouse event from terminal input.
// It supports both SGR format (<button>;x;y[Mm]) and legacy format (3 bytes).
func ParseMouseEvent(seq []byte) (*MouseEvent, error) {
	// SGR format: \033[<button>;x;yM (press) or m (release)
	// Example: \033[<0;10;5M for left click at (10,5)

	if len(seq) < 3 {
		return nil, fmt.Errorf("invalid mouse sequence: too short")
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
}

// Contains checks if a point is within the region
func (r *MouseRegion) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width &&
		y >= r.Y && y < r.Y+r.Height
}
