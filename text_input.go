package gooey

import (
	"image"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// TextInput is a simple single-line text input widget
type TextInput struct {
	BaseWidget
	Value       string
	Placeholder string
	CursorPos   int

	// Styles
	Style            Style
	PlaceholderStyle Style
	CursorStyle      Style

	// Callbacks
	OnChange func(value string)
	OnSubmit func(value string)

	// Internal
	focused bool
}

// NewTextInput creates a new text input widget
func NewTextInput() *TextInput {
	t := &TextInput{
		BaseWidget:       NewBaseWidget(),
		Value:            "",
		CursorPos:        0,
		Style:            NewStyle().WithForeground(ColorWhite),
		PlaceholderStyle: NewStyle().WithForeground(ColorBrightBlack),
		CursorStyle:      NewStyle().WithBackground(ColorWhite).WithForeground(ColorBlack),
	}
	t.SetMinSize(image.Point{X: 10, Y: 1})
	return t
}

// Draw renders the input
func (t *TextInput) Draw(frame RenderFrame) {
	bounds := t.GetBounds()
	width := bounds.Dx()

	if bounds.Dy() <= 0 {
		return
	}

	// Determine if we are in a SubFrame
	frameW, frameH := frame.Size()
	drawX, drawY := bounds.Min.X, bounds.Min.Y
	if frameW == width && frameH == bounds.Dy() {
		drawX, drawY = 0, 0
	}

	// Clear background
	frame.FillStyled(drawX, drawY, width, 1, ' ', t.Style)

	text := t.Value
	cursorIndex := t.CursorPos

	// Display text
	displayText := text
	displayStyle := t.Style

	if text == "" && t.Placeholder != "" {
		displayText = t.Placeholder
		displayStyle = t.PlaceholderStyle
	}

	// Simple truncation for now (TODO: scrolling)
	if runewidth.StringWidth(displayText) > width {
		displayText = runewidth.Truncate(displayText, width, "…")
	}

	frame.PrintStyled(drawX, drawY, displayText, displayStyle)

	// Draw cursor if focused
	if t.focused {
		// Calculate cursor visual position
		cursorX := drawX + runewidth.StringWidth(text[:cursorIndex])

		if cursorX < drawX+width {
			charUnderCursor := " "
			if cursorIndex < len(text) {
				_, w := utf8.DecodeRuneInString(text[cursorIndex:])
				charUnderCursor = text[cursorIndex : cursorIndex+w]
			}
			frame.PrintStyled(cursorX, drawY, charUnderCursor, t.CursorStyle)
		}
	}
}

// HandleKey handles key events
func (t *TextInput) HandleKey(event KeyEvent) bool {
	if !t.focused {
		return false
	}

	switch event.Key {
	case KeyArrowLeft:
		if t.CursorPos > 0 {
			_, w := utf8.DecodeLastRuneInString(t.Value[:t.CursorPos])
			t.CursorPos -= w
			t.MarkDirty()
		}
		return true
	case KeyArrowRight:
		if t.CursorPos < len(t.Value) {
			_, w := utf8.DecodeRuneInString(t.Value[t.CursorPos:])
			t.CursorPos += w
			t.MarkDirty()
		}
		return true
	case KeyBackspace:
		if t.CursorPos > 0 {
			// Find previous rune boundary
			_, w := utf8.DecodeLastRuneInString(t.Value[:t.CursorPos])
			t.Value = t.Value[:t.CursorPos-w] + t.Value[t.CursorPos:]
			t.CursorPos -= w
			if t.OnChange != nil {
				t.OnChange(t.Value)
			}
			t.MarkDirty()
		}
		return true
	case KeyDelete:
		if t.CursorPos < len(t.Value) {
			_, w := utf8.DecodeRuneInString(t.Value[t.CursorPos:])
			t.Value = t.Value[:t.CursorPos] + t.Value[t.CursorPos+w:]
			if t.OnChange != nil {
				t.OnChange(t.Value)
			}
			t.MarkDirty()
		}
		return true
	case KeyHome:
		t.CursorPos = 0
		t.MarkDirty()
		return true
	case KeyEnd:
		t.CursorPos = len(t.Value)
		t.MarkDirty()
		return true
	case KeyEnter:
		if t.OnSubmit != nil {
			t.OnSubmit(t.Value)
		}
		return true
	}

	if event.Rune != 0 && event.Rune >= 32 { // Printable characters
		// Insert rune
		newRune := string(event.Rune)
		t.Value = t.Value[:t.CursorPos] + newRune + t.Value[t.CursorPos:]
		t.CursorPos += len(newRune)
		if t.OnChange != nil {
			t.OnChange(t.Value)
		}
		t.MarkDirty()
		return true
	}

	return false
}

// SetFocused sets focus state
func (t *TextInput) SetFocused(focused bool) {
	t.focused = focused
	t.MarkDirty()
}

// HandleMouse handles mouse events
func (t *TextInput) HandleMouse(event MouseEvent) bool {
	bounds := t.GetBounds()
	if event.X < bounds.Min.X || event.X >= bounds.Max.X ||
		event.Y < bounds.Min.Y || event.Y >= bounds.Max.Y {
		return false
	}

	if event.Type == MousePress {
		t.SetFocused(true)
		// Simple cursor placement: end of text or try to calculate?
		// For now, just focus.
		return true
	}
	return false
}
