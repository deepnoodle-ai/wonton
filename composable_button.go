package gooey

import (
	"fmt"
	"image"
	"strings"
)

// ComposableButton is a bounds-based button that implements ComposableWidget.
// Unlike the original Button which uses absolute X, Y coordinates, this button
// works within a container-assigned bounds and supports the full composition system.
type ComposableButton struct {
	BaseWidget
	Label      string
	Style      Style
	HoverStyle Style
	Focused    bool
	Hovered    bool
	OnClick    func()
}

// NewComposableButton creates a new composable button
func NewComposableButton(label string, onClick func()) *ComposableButton {
	btn := &ComposableButton{
		BaseWidget: NewBaseWidget(),
		Label:      label,
		Style:      NewStyle().WithBackground(ColorBlue).WithForeground(ColorWhite),
		HoverStyle: NewStyle().WithBackground(ColorCyan).WithForeground(ColorBlack).WithBold(),
		OnClick:    onClick,
	}

	// Set minimum size based on label
	minWidth := len(label) + 4 // Add padding and brackets
	btn.SetMinSize(image.Point{X: minWidth, Y: 1})
	btn.SetPreferredSize(image.Point{X: minWidth, Y: 1})

	return btn
}

// Draw renders the button within its assigned bounds
func (cb *ComposableButton) Draw(frame RenderFrame) {
	if !cb.visible {
		return
	}

	bounds := cb.GetBounds()
	style := cb.Style
	if cb.Hovered {
		style = cb.HoverStyle
	}
	if cb.Focused {
		style = style.WithUnderline()
	}

	// Determine if we're drawing in a positioned SubFrame
	// If the frame's size matches our bounds size, we're in a SubFrame at our position
	frameWidth, frameHeight := frame.Size()
	inSubFrame := (frameWidth == bounds.Dx() && frameHeight == bounds.Dy())

	// Calculate button width from bounds
	width := bounds.Dx()

	// Handle very small widths
	if width < 2 {
		cb.ClearDirty()
		return
	}

	// Build button text with label
	availableWidth := width - 2 // -2 for brackets
	labelText := cb.Label

	// Truncate label if too long
	if len(labelText) > availableWidth-2 { // -2 for spaces around label
		if availableWidth > 2 {
			labelText = labelText[:availableWidth-2]
		} else {
			labelText = ""
		}
	}

	// Create button text with padding
	buttonText := fmt.Sprintf(" %s ", labelText)
	if len(buttonText) < availableWidth {
		padding := availableWidth - len(buttonText)
		buttonText += strings.Repeat(" ", padding)
	} else if len(buttonText) > availableWidth {
		// Truncate to fit
		buttonText = buttonText[:availableWidth]
	}

	// Draw the button - ensure it doesn't wrap by using character-by-character drawing
	fullText := "[" + buttonText + "]"
	if len(fullText) > width {
		fullText = fullText[:width]
	}

	// TEMP DEBUG: For the first button only, show what we're drawing
	if cb.Label == "Click Me!" {
		_ = width // Keep compiler happy
		// Log would go here but we can't easily log in Draw()
		// Instead we'll just ensure fullText is correct
	}

	// Draw each character to avoid wrapping issues
	// Use a regular for loop with index to handle byte positions correctly
	// If we're in a SubFrame, draw at (0,0). Otherwise use absolute bounds.
	var startX, startY int
	if inSubFrame {
		startX, startY = 0, 0
	} else {
		startX, startY = bounds.Min.X, bounds.Min.Y
	}

	for i := 0; i < len(fullText) && i < width; i++ {
		err := frame.SetCell(startX+i, startY, rune(fullText[i]), style)
		if err != nil {
			// SetCell failed - likely out of bounds
			// This shouldn't happen but let's be defensive
			break
		}
	}

	cb.ClearDirty()
}

// HandleKey handles keyboard events
func (cb *ComposableButton) HandleKey(event KeyEvent) bool {
	if !cb.visible || !cb.Focused {
		return false
	}

	if event.Key == KeyEnter || event.Rune == ' ' {
		if cb.OnClick != nil {
			cb.OnClick()
		}
		return true
	}

	return false
}

// HandleMouse handles mouse events
func (cb *ComposableButton) HandleMouse(event MouseEvent) bool {
	if !cb.visible {
		return false
	}

	bounds := cb.GetBounds()

	// Check if mouse is within button bounds
	if event.X >= bounds.Min.X && event.X < bounds.Max.X &&
		event.Y >= bounds.Min.Y && event.Y < bounds.Max.Y {

		// Update hover state
		wasHovered := cb.Hovered
		cb.Hovered = true
		if wasHovered != cb.Hovered {
			cb.MarkDirty()
		}

		// Handle click
		if event.Button == MouseLeft && event.Type == MouseClick {
			if cb.OnClick != nil {
				cb.OnClick()
			}
			return true
		}

		return true
	}

	// Mouse is outside button
	if cb.Hovered {
		cb.Hovered = false
		cb.MarkDirty()
	}

	return false
}

// SetFocused sets the focused state
func (cb *ComposableButton) SetFocused(focused bool) {
	if cb.Focused != focused {
		cb.Focused = focused
		cb.MarkDirty()
	}
}
