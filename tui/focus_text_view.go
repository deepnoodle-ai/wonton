package tui

// focusTextView displays text with different styles based on focus state
type focusTextView struct {
	content    string
	focusID    string // The focus ID to watch
	style      Style  // Normal style
	focusStyle *Style // Style when watched element is focused
}

// FocusText creates a text view that changes style based on a watched focus ID.
// This is useful for labels that should highlight when their associated input is focused.
//
// Example:
//
//	FocusText("Name: ", "name-input").
//	    Style(dimStyle).
//	    FocusStyle(brightStyle)
func FocusText(content string, focusID string) *focusTextView {
	return &focusTextView{
		content: content,
		focusID: focusID,
		style:   NewStyle(),
	}
}

// Style sets the normal (unfocused) style.
func (f *focusTextView) Style(s Style) *focusTextView {
	f.style = s
	return f
}

// FocusStyle sets the style applied when the watched element is focused.
func (f *focusTextView) FocusStyle(s Style) *focusTextView {
	f.focusStyle = &s
	return f
}

// Fg sets the normal foreground color.
func (f *focusTextView) Fg(c Color) *focusTextView {
	f.style = f.style.WithForeground(c)
	return f
}

// Bg sets the normal background color.
func (f *focusTextView) Bg(c Color) *focusTextView {
	f.style = f.style.WithBackground(c)
	return f
}

// Bold enables bold text in the normal style.
func (f *focusTextView) Bold() *focusTextView {
	f.style = f.style.WithBold()
	return f
}

// Dim enables dim text in the normal style.
func (f *focusTextView) Dim() *focusTextView {
	f.style = f.style.WithDim()
	return f
}

func (f *focusTextView) size(maxWidth, maxHeight int) (int, int) {
	w, h := MeasureText(f.content)
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (f *focusTextView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Determine if the watched element is focused
	isFocused := f.focusID != "" && focusManager.GetFocusedID() == f.focusID

	// Choose style based on focus state
	style := f.style
	if isFocused && f.focusStyle != nil {
		style = *f.focusStyle
	}

	ctx.PrintTruncated(0, 0, f.content, style)
}
