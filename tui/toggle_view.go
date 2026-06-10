package tui

import (
	"fmt"
	"image"
)

// ToggleView displays an on/off toggle switch
type ToggleView struct {
	id         string
	value      *bool
	onLabel    string
	offLabel   string
	onStyle    Style
	offStyle   Style
	onChange   func(bool)
	showLabels bool
	bounds     image.Rectangle
	focused    bool
}

// Toggle creates an on/off toggle switch.
// value should be a pointer to a bool controlling the toggle state.
//
// The component handles keyboard input (Space/Enter to toggle) automatically when focused.
// Use Tab to focus the toggle.
//
// Example:
//
//	Toggle(&app.darkMode).OnChange(func(v bool) { app.updateTheme() })
func Toggle(value *bool) *ToggleView {
	// Generate ID from value pointer address
	id := fmt.Sprintf("toggle_%p", value)
	return &ToggleView{
		id:         id,
		value:      value,
		onLabel:    "ON",
		offLabel:   "OFF",
		onStyle:    NewStyle().WithForeground(ColorGreen).WithBold(),
		offStyle:   NewStyle().WithForeground(ColorBrightBlack),
		showLabels: true,
	}
}

// OnLabel sets the label for the ON state.
func (t *ToggleView) OnLabel(label string) *ToggleView {
	t.onLabel = label
	return t
}

// OffLabel sets the label for the OFF state.
func (t *ToggleView) OffLabel(label string) *ToggleView {
	t.offLabel = label
	return t
}

// OnStyle sets the style for the ON state.
func (t *ToggleView) OnStyle(s Style) *ToggleView {
	t.onStyle = s
	return t
}

// OffStyle sets the style for the OFF state.
func (t *ToggleView) OffStyle(s Style) *ToggleView {
	t.offStyle = s
	return t
}

// OnChange sets a callback when the toggle is clicked.
func (t *ToggleView) OnChange(fn func(bool)) *ToggleView {
	t.onChange = fn
	return t
}

// ShowLabels enables/disables showing ON/OFF labels.
func (t *ToggleView) ShowLabels(show bool) *ToggleView {
	t.showLabels = show
	return t
}

// ID sets a custom ID for this toggle (for focus management).
func (t *ToggleView) ID(id string) *ToggleView {
	t.id = id
	return t
}

// Focusable interface implementation
func (t *ToggleView) FocusID() string {
	return t.id
}

func (t *ToggleView) IsFocused() bool {
	return t.focused
}

func (t *ToggleView) SetFocused(focused bool) {
	t.focused = focused
}

func (t *ToggleView) FocusBounds() image.Rectangle {
	return t.bounds
}

func (t *ToggleView) HandleKeyEvent(event KeyEvent) bool {
	if t.value == nil {
		return false
	}

	// Handle Space or Enter to toggle
	if event.Rune == ' ' || event.Key == KeyEnter {
		*t.value = !*t.value
		if t.onChange != nil {
			t.onChange(*t.value)
		}
		return true
	}

	return false
}

func (t *ToggleView) size(maxWidth, maxHeight int) (int, int) {
	// [●] ON  or  [○] OFF
	w := 3 // switch chars
	if t.showLabels {
		onW, _ := MeasureText(t.onLabel)
		offW, _ := MeasureText(t.offLabel)
		if onW > offW {
			w += 1 + onW
		} else {
			w += 1 + offW
		}
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (t *ToggleView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Register with focus manager for keyboard input (if available)
	t.bounds = ctx.AbsoluteBounds()
	if fm := ctx.FocusManager(); fm != nil {
		fm.Register(t)
	}

	isOn := t.value != nil && *t.value

	var switchChar string
	var style Style
	var label string

	if isOn {
		switchChar = "●"
		style = t.onStyle
		label = t.onLabel
	} else {
		switchChar = "○"
		style = t.offStyle
		label = t.offLabel
	}

	// Draw switch
	text := "[" + switchChar + "]"
	if t.showLabels {
		text += " " + label
	}
	ctx.PrintStyled(0, 0, text, style)

	// Register click region
	if t.value != nil {
		ctx.registries().interactive.RegisterButton(ctx.AbsoluteBounds(), func() {
			*t.value = !*t.value
			if t.onChange != nil {
				t.onChange(*t.value)
			}
		})
	}
}
