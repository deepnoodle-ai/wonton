package gooey

import (
	"fmt"
)

// CheckboxGroup represents a group of checkboxes
type CheckboxGroup struct {
	X, Y          int
	Options       []string
	Selected      map[int]bool
	Cursor        int
	Style         Style
	SelectedStyle Style
	CursorStyle   Style
	OnChange      func(selected []string)
	Focused       bool
}

// NewCheckboxGroup creates a new checkbox group
func NewCheckboxGroup(x, y int, options []string) *CheckboxGroup {
	return &CheckboxGroup{
		X:             x,
		Y:             y,
		Options:       options,
		Selected:      make(map[int]bool),
		Cursor:        0,
		Style:         NewStyle(),
		SelectedStyle: NewStyle().WithForeground(ColorGreen).WithBold(),
		CursorStyle:   NewStyle().WithBackground(ColorBrightBlack),
		Focused:       true,
	}
}

// Toggle toggles the selection of an item
func (cg *CheckboxGroup) Toggle(index int) {
	if index >= 0 && index < len(cg.Options) {
		if cg.Selected[index] {
			delete(cg.Selected, index)
		} else {
			cg.Selected[index] = true
		}
		if cg.OnChange != nil {
			cg.OnChange(cg.GetSelectedItems())
		}
	}
}

// GetSelectedItems returns the list of currently selected options
func (cg *CheckboxGroup) GetSelectedItems() []string {
	var items []string
	for i, opt := range cg.Options {
		if cg.Selected[i] {
			items = append(items, opt)
		}
	}
	return items
}

// Draw renders the checkbox group
func (cg *CheckboxGroup) Draw(frame RenderFrame) {
	// Coordinate handling for compatibility with composition system
	drawX, drawY := cg.X, cg.Y
	_, frameHeight := frame.Size()
	// If frame height matches our content height exactly, assume we are in a SubFrame
	// and should draw relative to it (at 0,0) instead of absolute coordinates.
	if frameHeight == len(cg.Options) {
		drawX, drawY = 0, 0
	}

	for i, option := range cg.Options {
		style := cg.Style
		indicator := "[ ]"
		if cg.Selected[i] {
			style = cg.SelectedStyle
			indicator = "[x]"
		}

		line := fmt.Sprintf("%s %s", indicator, option)

		// Highlight cursor line if focused
		if cg.Focused && i == cg.Cursor {
			// Apply cursor background to the style
			lineStyle := style
			if cg.CursorStyle.Background != ColorDefault {
				lineStyle = lineStyle.WithBackground(cg.CursorStyle.Background)
			} else if cg.CursorStyle.BgRGB != nil {
				lineStyle.BgRGB = cg.CursorStyle.BgRGB
			} else {
				lineStyle = lineStyle.WithBackground(ColorBrightBlack)
			}

			frame.PrintStyled(drawX, drawY+i, line, lineStyle)
		} else {
			frame.PrintStyled(drawX, drawY+i, line, style)
		}
	}
}

// HandleKey handles key events
func (cg *CheckboxGroup) HandleKey(event KeyEvent) bool {
	if !cg.Focused {
		return false
	}

	switch event.Key {
	case KeyArrowUp:
		if cg.Cursor > 0 {
			cg.Cursor--
			return true
		}
	case KeyArrowDown:
		if cg.Cursor < len(cg.Options)-1 {
			cg.Cursor++
			return true
		}
	default:
		if event.Rune == ' ' {
			cg.Toggle(cg.Cursor)
			return true
		}
	}
	return false
}

// SetFocus sets the focus state
func (cg *CheckboxGroup) SetFocus(focus bool) {
	cg.Focused = focus
}
