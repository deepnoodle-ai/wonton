package tui

import "image"

// Focus returns a command that sets focus to the specified input ID.
// Use this in HandleEvent to programmatically focus an input.
//
// Example:
//
//	func (app *App) HandleEvent(e Event) []Cmd {
//	    if key, ok := e.(KeyEvent); ok && key.Key == KeyTab {
//	        return []Cmd{Focus("input-name")}
//	    }
//	    return nil
//	}
func Focus(id string) Cmd {
	return func() Event {
		inputRegistry.SetFocus(id)
		return nil
	}
}

// FocusNext returns a command that moves focus to the next input.
func FocusNext() Cmd {
	return func() Event {
		inputRegistry.FocusNext()
		return nil
	}
}

// focusableView wraps a view to make it participate in focus management
type focusableView struct {
	id         string
	inner      View
	focusStyle Style
}

// Focusable wraps a view with a focus ID.
// When focused, the view can receive an optional focus ring style.
//
// Example:
//
//	Focusable("submit-btn",
//	    Button("Submit", app.submit),
//	).FocusStyle(NewStyle().WithReverse())
func Focusable(id string, inner View) *focusableView {
	return &focusableView{
		id:         id,
		inner:      inner,
		focusStyle: NewStyle(),
	}
}

// FocusStyle sets the style applied when this view is focused.
func (f *focusableView) FocusStyle(s Style) *focusableView {
	f.focusStyle = s
	return f
}

func (f *focusableView) size(maxWidth, maxHeight int) (int, int) {
	return f.inner.size(maxWidth, maxHeight)
}

func (f *focusableView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	// Check if this view is focused (by checking input registry)
	// This is a simplified implementation - full focus management would
	// need its own registry for non-input focusable items
	isFocused := inputRegistry.focused == f.id

	if isFocused && !f.focusStyle.IsEmpty() {
		// Apply focus style as background/border
		subFrame := frame.SubFrame(bounds)
		width, height := subFrame.Size()
		subFrame.FillStyled(0, 0, width, height, ' ', f.focusStyle)
	}

	f.inner.render(frame, bounds)
}
