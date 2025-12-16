package tui

import (
	"fmt"
	"image"
	"strings"
)

// inputFieldView is a high-level component combining a label, input, and optional border
// with automatic focus-aware styling.
type inputFieldView struct {
	// Input configuration
	id               string
	binding          *string
	placeholder      string
	placeholderStyle *Style
	mask             rune
	onChange         func(string)
	onSubmit         func(string)
	width            int
	maxHeight        int
	pastePlaceholder bool
	cursorBlink      bool
	multiline        bool

	// Label configuration
	label           string
	labelStyle      Style
	focusLabelStyle *Style

	// Border configuration
	bordered       bool
	border         *BorderStyle
	borderFg       Color
	focusBorderFg  Color
	hasFocusBorder bool
}

// InputField creates a high-level input component with optional label and border.
// Focus styling is handled automatically - the label and border change appearance
// when the input is focused.
//
// Example:
//
//	InputField(&app.name).
//	    Label("Name:").
//	    Placeholder("Enter your name").
//	    Bordered().
//	    Width(40)
func InputField(binding *string) *inputFieldView {
	id := ""
	if binding != nil {
		id = generateInputID(binding)
	}
	return &inputFieldView{
		id:         id,
		binding:    binding,
		width:      20,
		labelStyle: NewStyle().WithForeground(ColorBrightBlack),
	}
}

// generateInputID creates a unique ID from a pointer address
func generateInputID(binding *string) string {
	return fmt.Sprintf("input_%p", binding)
}

// ID sets a specific ID for this input field.
func (f *inputFieldView) ID(id string) *inputFieldView {
	f.id = id
	return f
}

// Label sets the label text displayed before the input.
func (f *inputFieldView) Label(text string) *inputFieldView {
	f.label = text
	return f
}

// LabelStyle sets the style for the label when unfocused.
func (f *inputFieldView) LabelStyle(s Style) *inputFieldView {
	f.labelStyle = s
	return f
}

// FocusLabelStyle sets the style for the label when the input is focused.
// Defaults to cyan + bold if not set.
func (f *inputFieldView) FocusLabelStyle(s Style) *inputFieldView {
	f.focusLabelStyle = &s
	return f
}

// Placeholder sets the placeholder text shown when empty.
func (f *inputFieldView) Placeholder(text string) *inputFieldView {
	f.placeholder = text
	return f
}

// PlaceholderStyle sets the style for the placeholder text.
func (f *inputFieldView) PlaceholderStyle(s Style) *inputFieldView {
	f.placeholderStyle = &s
	return f
}

// Mask sets a mask character for password input.
func (f *inputFieldView) Mask(r rune) *inputFieldView {
	f.mask = r
	return f
}

// OnChange sets a callback invoked when the value changes.
func (f *inputFieldView) OnChange(fn func(string)) *inputFieldView {
	f.onChange = fn
	return f
}

// OnSubmit sets a callback invoked when Enter is pressed.
func (f *inputFieldView) OnSubmit(fn func(string)) *inputFieldView {
	f.onSubmit = fn
	return f
}

// Width sets the display width of the input (not including label or border).
func (f *inputFieldView) Width(w int) *inputFieldView {
	f.width = w
	return f
}

// MaxHeight sets the maximum height in lines for multiline input.
func (f *inputFieldView) MaxHeight(lines int) *inputFieldView {
	f.maxHeight = lines
	return f
}

// PastePlaceholder enables paste placeholder mode.
func (f *inputFieldView) PastePlaceholder(enabled bool) *inputFieldView {
	f.pastePlaceholder = enabled
	return f
}

// CursorBlink enables or disables cursor blinking.
func (f *inputFieldView) CursorBlink(enabled bool) *inputFieldView {
	f.cursorBlink = enabled
	return f
}

// Multiline enables multiline input where Shift+Enter inserts newlines.
func (f *inputFieldView) Multiline(enabled bool) *inputFieldView {
	f.multiline = enabled
	return f
}

// Bordered enables a border around the input.
func (f *inputFieldView) Bordered() *inputFieldView {
	f.bordered = true
	if f.border == nil {
		f.border = &RoundedBorder
	}
	return f
}

// Border sets the border style (implies Bordered).
func (f *inputFieldView) Border(style *BorderStyle) *inputFieldView {
	f.bordered = true
	f.border = style
	return f
}

// BorderFg sets the border foreground color.
func (f *inputFieldView) BorderFg(c Color) *inputFieldView {
	f.borderFg = c
	return f
}

// FocusBorderFg sets the border color when the input is focused.
// Defaults to cyan if not set.
func (f *inputFieldView) FocusBorderFg(c Color) *inputFieldView {
	f.focusBorderFg = c
	f.hasFocusBorder = true
	return f
}

func (f *inputFieldView) size(maxWidth, maxHeight int) (int, int) {
	// Account for border if present
	borderSize := 0
	if f.bordered && f.border != nil {
		borderSize = 2
	}

	// Calculate input width
	totalW := f.width

	// Inner width for content calculation (input area inside border)
	innerW := totalW
	if f.bordered {
		innerW = totalW - borderSize
		if innerW < 1 {
			innerW = 1
		}
	}

	// Calculate height based on content wrapping
	h := 1
	if f.binding != nil && *f.binding != "" && innerW > 0 {
		displayText := *f.binding
		if state, exists := inputRegistry.inputs[f.id]; exists {
			displayText = state.input.DisplayText()
		}
		h = calcWrappedHeight(displayText, innerW)
	}

	// Apply max height constraint
	if f.maxHeight > 0 && h > f.maxHeight {
		h = f.maxHeight
	}

	// If bordered, label is embedded in border, so no extra width needed
	// If not bordered, add label width to total
	if !f.bordered && f.label != "" {
		labelW, _ := MeasureText(f.label)
		totalW += labelW
	}

	totalH := h + borderSize

	// Apply constraints
	if maxWidth > 0 && totalW > maxWidth {
		totalW = maxWidth
	}
	if maxHeight > 0 && totalH > maxHeight {
		totalH = maxHeight
	}

	return totalW, totalH
}

func (f *inputFieldView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Determine if this input is focused
	isFocused := focusManager.GetFocusedID() == f.id

	if f.bordered && f.border != nil {
		// Draw bordered input with label embedded in top border
		f.renderBorderedInput(ctx, 0, w, h, isFocused)
	} else {
		// Draw label to the left of input (no border case)
		labelW := 0
		if f.label != "" {
			labelW, _ = MeasureText(f.label)
			labelStyle := f.labelStyle
			if isFocused {
				if f.focusLabelStyle != nil {
					labelStyle = *f.focusLabelStyle
				} else {
					labelStyle = NewStyle().WithForeground(ColorCyan).WithBold()
				}
			}
			ctx.PrintTruncated(0, 0, f.label, labelStyle)
		}

		// Draw input after label
		inputW := w - labelW
		if inputW <= 0 {
			return
		}
		inputBounds := image.Rect(labelW, 0, labelW+inputW, h)
		inputCtx := ctx.SubContext(inputBounds)
		f.renderInput(inputCtx, isFocused)
	}
}

func (f *inputFieldView) renderBorderedInput(ctx *RenderContext, x, w, h int, isFocused bool) {
	// Determine border color based on focus
	borderStyle := NewStyle()
	if isFocused {
		if f.hasFocusBorder {
			borderStyle = borderStyle.WithForeground(f.focusBorderFg)
		} else {
			// Default focus border: cyan
			borderStyle = borderStyle.WithForeground(ColorCyan)
		}
	} else if f.borderFg != 0 {
		borderStyle = borderStyle.WithForeground(f.borderFg)
	}

	// Determine label style based on focus
	labelStyle := f.labelStyle
	if isFocused {
		if f.focusLabelStyle != nil {
			labelStyle = *f.focusLabelStyle
		} else {
			// Default focus label style: cyan + bold
			labelStyle = NewStyle().WithForeground(ColorCyan).WithBold()
		}
	}

	// Draw top border with embedded label
	// Format: ╭─ Label ─────────────╮
	ctx.PrintTruncated(x, 0, f.border.TopLeft, borderStyle)
	bx := x + 1

	if f.label != "" && w > 4 {
		// Draw horizontal line before label
		ctx.PrintTruncated(bx, 0, f.border.Horizontal, borderStyle)
		bx++

		// Draw label with space padding, stripping trailing colon for cleaner look
		label := strings.TrimSuffix(strings.TrimSuffix(f.label, ": "), ":")
		labelText := " " + label + " "
		labelW, _ := MeasureText(labelText)
		maxLabelW := w - 4 // Leave room for corners and some border
		if labelW > maxLabelW {
			labelW = maxLabelW
		}
		ctx.PrintTruncated(bx, 0, labelText, labelStyle)
		bx += labelW
	}

	// Fill rest of top border
	for ; bx < x+w-1; bx++ {
		ctx.PrintTruncated(bx, 0, f.border.Horizontal, borderStyle)
	}
	if w > 1 {
		ctx.PrintTruncated(x+w-1, 0, f.border.TopRight, borderStyle)
	}

	// Side borders
	for y := 1; y < h-1; y++ {
		ctx.PrintTruncated(x, y, f.border.Vertical, borderStyle)
		if w > 1 {
			ctx.PrintTruncated(x+w-1, y, f.border.Vertical, borderStyle)
		}
	}

	// Bottom border
	if h > 1 {
		ctx.PrintTruncated(x, h-1, f.border.BottomLeft, borderStyle)
		for bx := x + 1; bx < x+w-1; bx++ {
			ctx.PrintTruncated(bx, h-1, f.border.Horizontal, borderStyle)
		}
		if w > 1 {
			ctx.PrintTruncated(x+w-1, h-1, f.border.BottomRight, borderStyle)
		}
	}

	// Inner content area
	innerBounds := image.Rect(x+1, 1, x+w-1, h-1)
	if innerBounds.Dx() > 0 && innerBounds.Dy() > 0 {
		innerCtx := ctx.SubContext(innerBounds)
		f.renderInput(innerCtx, isFocused)
	}
}

func (f *inputFieldView) renderInput(ctx *RenderContext, isFocused bool) {
	// Register this input - use absolute bounds for click registration
	inputBounds := ctx.AbsoluteBounds()
	state := inputRegistry.Register(f.id, f.binding, inputBounds, f.placeholder, f.placeholderStyle, f.mask, f.pastePlaceholder, f.cursorBlink, f.multiline, f.maxHeight, f.onChange, f.onSubmit)

	// Update TextInput bounds
	state.input.SetBounds(inputBounds)

	// Draw the TextInput
	state.input.Draw(ctx.frame)
}
