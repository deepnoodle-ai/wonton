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
	cursorShape      InputCursorStyle
	cursorColor      *Color
	multiline        bool

	// Label configuration
	label           string
	labelStyle      Style
	focusLabelStyle *Style

	// Prompt configuration (left-side caret/prompt character)
	prompt      string // Optional prompt character (e.g., ">", "❯", etc.)
	promptStyle Style
	hasPrompt   bool

	// Border configuration
	bordered          bool
	border            *BorderStyle
	borderFg          Color
	focusBorderFg     Color
	hasFocusBorder    bool
	horizontalBarOnly bool // If true, only draw top and bottom borders
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
		width:      0, // 0 means fill available width
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

// HorizontalBorderOnly enables horizontal bar border style (top and bottom only).
// This creates a cleaner look with just horizontal lines above and below the input.
func (f *inputFieldView) HorizontalBorderOnly() *inputFieldView {
	f.bordered = true
	f.horizontalBarOnly = true
	return f
}

// Prompt sets a prompt character displayed on the left side of the input.
// Common examples: ">", "❯", "$", etc.
func (f *inputFieldView) Prompt(text string) *inputFieldView {
	f.prompt = text
	f.hasPrompt = true
	return f
}

// PromptStyle sets the style for the prompt character.
func (f *inputFieldView) PromptStyle(s Style) *inputFieldView {
	f.promptStyle = s
	return f
}

// CursorShape sets the cursor shape/style for the input.
// Options are: InputCursorBlock (default), InputCursorUnderline, InputCursorBar.
func (f *inputFieldView) CursorShape(shape InputCursorStyle) *inputFieldView {
	f.cursorShape = shape
	return f
}

// CursorColor sets a custom cursor color.
func (f *inputFieldView) CursorColor(color Color) *inputFieldView {
	f.cursorColor = &color
	return f
}

func (f *inputFieldView) size(maxWidth, maxHeight int) (int, int) {
	// Account for border if present
	borderSize := 0
	if f.bordered && f.border != nil && !f.horizontalBarOnly {
		borderSize = 2
	}

	// Calculate input width (0 means fill available width)
	totalW := f.width
	if totalW <= 0 && maxWidth > 0 {
		totalW = maxWidth
	} else if totalW <= 0 {
		totalW = 80 // Fallback if no maxWidth specified
	}

	// Account for prompt width
	promptW := 0
	if f.hasPrompt && f.prompt != "" {
		promptW, _ = MeasureText(f.prompt)
		promptW++ // Add space after prompt
	}

	// Inner width for content calculation (input area inside border and prompt)
	innerW := totalW
	if f.bordered && !f.horizontalBarOnly {
		innerW = totalW - borderSize
		if innerW < 1 {
			innerW = 1
		}
	}
	if f.hasPrompt {
		innerW -= promptW
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

	// If bordered (full border), label is embedded in border, so no extra width needed
	// If not bordered or horizontal bar only, add label width to total
	if (!f.bordered || f.horizontalBarOnly) && f.label != "" {
		labelW, _ := MeasureText(f.label)
		totalW += labelW
	}

	// Add prompt width to total
	if f.hasPrompt {
		totalW += promptW
	}

	totalH := h
	if f.bordered && !f.horizontalBarOnly {
		totalH += borderSize
	} else if f.horizontalBarOnly {
		totalH += 2 // Top and bottom bars
	}

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
	fm := ctx.FocusManager()
	isFocused := fm != nil && fm.GetFocusedID() == f.id

	if f.bordered && f.border != nil && !f.horizontalBarOnly {
		// Draw full bordered input with label embedded in top border
		f.renderBorderedInput(ctx, 0, w, h, isFocused)
	} else if f.horizontalBarOnly {
		// Draw horizontal bar style (top and bottom only)
		f.renderHorizontalBarInput(ctx, 0, w, h, isFocused)
	} else {
		// Draw label and/or prompt to the left of input (no border case)
		x := 0

		// Draw label if present
		if f.label != "" {
			labelStyle := f.labelStyle
			if isFocused {
				if f.focusLabelStyle != nil {
					labelStyle = *f.focusLabelStyle
				} else {
					labelStyle = NewStyle().WithForeground(ColorCyan).WithBold()
				}
			}
			// Add space after label if it doesn't already have one
			labelText := f.label
			if !strings.HasSuffix(labelText, " ") {
				labelText += " "
			}
			labelW, _ := MeasureText(labelText)
			ctx.PrintTruncated(x, 0, labelText, labelStyle)
			x += labelW
		}

		// Draw prompt if present
		if f.hasPrompt && f.prompt != "" {
			promptStyle := f.promptStyle
			if promptStyle.IsEmpty() {
				promptStyle = NewStyle().WithForeground(ColorBrightBlack)
			}
			ctx.PrintTruncated(x, 0, f.prompt+" ", promptStyle)
			promptW, _ := MeasureText(f.prompt)
			x += promptW + 1
		}

		// Draw input after label and prompt
		inputW := w - x
		if inputW <= 0 {
			return
		}
		inputBounds := image.Rect(x, 0, x+inputW, h)
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

func (f *inputFieldView) renderHorizontalBarInput(ctx *RenderContext, x, w, h int, isFocused bool) {
	// Determine border color based on focus
	borderStyle := NewStyle()
	if isFocused {
		if f.hasFocusBorder {
			borderStyle = borderStyle.WithForeground(f.focusBorderFg)
		} else {
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
			labelStyle = NewStyle().WithForeground(ColorCyan).WithBold()
		}
	}

	// Get border character (use horizontal from border style, or default to simple line)
	borderChar := "─"
	if f.border != nil {
		borderChar = f.border.Horizontal
	}

	// Draw top border
	topY := 0
	if f.label != "" && w > 4 {
		// Draw a few border characters before the label
		bx := x
		prefixChars := 2 // Number of border chars before label
		for i := 0; i < prefixChars && bx < x+w; i++ {
			ctx.PrintTruncated(bx, topY, borderChar, borderStyle)
			bx++
		}

		// Draw label
		label := strings.TrimSuffix(strings.TrimSuffix(f.label, ": "), ":")
		labelText := " " + label + " "
		labelW, _ := MeasureText(labelText)
		maxLabelW := w - prefixChars - 2
		if labelW > maxLabelW {
			labelW = maxLabelW
		}
		ctx.PrintTruncated(bx, topY, labelText, labelStyle)
		bx += labelW

		// Fill rest of line with border
		for ; bx < x+w; bx++ {
			ctx.PrintTruncated(bx, topY, borderChar, borderStyle)
		}
	} else {
		// No label, just draw full border
		for bx := x; bx < x+w; bx++ {
			ctx.PrintTruncated(bx, topY, borderChar, borderStyle)
		}
	}

	// Draw prompt if present (on the input line)
	inputY := 1
	inputX := x
	if f.hasPrompt && f.prompt != "" {
		promptStyle := f.promptStyle
		if promptStyle.IsEmpty() {
			promptStyle = NewStyle().WithForeground(ColorBrightBlack)
		}
		ctx.PrintTruncated(inputX, inputY, f.prompt+" ", promptStyle)
		promptW, _ := MeasureText(f.prompt)
		inputX += promptW + 1
	}

	// Draw input content (between borders)
	contentHeight := h - 2
	if contentHeight > 0 {
		inputW := w - (inputX - x)
		if inputW > 0 {
			inputBounds := image.Rect(inputX, inputY, inputX+inputW, inputY+contentHeight)
			inputCtx := ctx.SubContext(inputBounds)
			f.renderInput(inputCtx, isFocused)
		}
	}

	// Draw bottom border
	bottomY := h - 1
	for bx := x; bx < x+w; bx++ {
		ctx.PrintTruncated(bx, bottomY, borderChar, borderStyle)
	}
}

func (f *inputFieldView) renderInput(ctx *RenderContext, isFocused bool) {
	// Register this input - use absolute bounds for click registration
	inputBounds := ctx.AbsoluteBounds()
	state := inputRegistry.Register(f.id, f.binding, inputBounds, f.placeholder, f.placeholderStyle, f.mask, f.pastePlaceholder, f.cursorBlink, f.multiline, f.maxHeight, f.onChange, f.onSubmit, ctx.FocusManager())

	// Apply cursor customizations if set
	if f.cursorShape != InputCursorBlock {
		state.input.CursorShape = f.cursorShape
	}
	if f.cursorColor != nil {
		state.input.CursorColor = f.cursorColor
	}

	// Update TextInput bounds
	state.input.SetBounds(inputBounds)

	// Draw the TextInput
	state.input.Draw(ctx.frame)
}
