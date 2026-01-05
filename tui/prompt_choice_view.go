package tui

import (
	"fmt"
	"image"
)

// promptChoiceRegistry manages text input state for prompt choices.
// This registry persists TextInput widgets across renders, maintaining
// cursor position and input state.
var promptChoiceRegistry = &promptChoiceRegistryImpl{
	inputs: make(map[string]*TextInput),
}

type promptChoiceRegistryImpl struct {
	inputs map[string]*TextInput
}

func (r *promptChoiceRegistryImpl) Get(id string) *TextInput {
	if ti, ok := r.inputs[id]; ok {
		return ti
	}
	ti := NewTextInput()
	ti.SubmitOnEnter = false // We handle Enter ourselves
	r.inputs[id] = ti
	return ti
}

// promptChoiceView displays a selection with numbered options and optional inline input.
//
// This component provides a Claude Code-style selection interface where users can:
//   - Navigate between options using arrow keys
//   - Jump directly to an option using number keys (1-9)
//   - Type custom text when an input option is selected
//   - Confirm selection with Enter
//   - Cancel with Escape
//
// The component is ideal for confirmation dialogs, action menus, and any UI
// that needs both preset choices and a custom input option.
//
// # Visual Layout
//
// When rendered, the component displays:
//
//	❯ 1. First option
//	  2. Second option
//	  3. Type custom response here
//
//	Esc to cancel
//
// The cursor indicator (❯) shows the currently selected option.
// When option 3 is selected, the user can type directly.
//
// # Keyboard Controls
//
//   - Arrow Up/Down: Navigate between options
//   - 1-9: Jump directly to numbered option
//   - Enter: Confirm selection (triggers OnSelect callback)
//   - Escape: Cancel (triggers OnCancel callback)
//   - Any character: When on input option, types into the input field
//   - Backspace: When on input option, deletes characters
//
// # Focus Integration
//
// The component implements the Focusable interface and integrates with
// the FocusManager for Tab navigation between multiple focusable elements.
type promptChoiceView struct {
	id          string
	selected    *int      // Currently selected option index
	inputText   *string   // Text for the input option (if any)
	options     []string  // Fixed options
	inputLabel  string    // Label for the input option (empty = no input option)
	onSelect    func(idx int, inputText string)
	onCancel    func()
	style       Style
	inputStyle  Style
	cursorStyle Style
	cursorChar  string
	showNumbers bool
	hintText    string
	width       int
	bounds      image.Rectangle
	focused     bool
}

// PromptChoice creates a prompt choice view with selectable options.
//
// This component creates a Claude Code-style selection interface with numbered
// options and optional inline text input. It's ideal for confirmation dialogs,
// action menus, and prompts that need both preset choices and custom input.
//
// # Parameters
//
//   - selected: Pointer to track the currently selected index (0-based).
//     This value updates as the user navigates.
//   - inputText: Pointer to store custom input text. Can be nil if no input
//     option is needed. When an InputOption is added, typed text is stored here.
//
// # Basic Example
//
//	var selected int
//	var customText string
//
//	view := PromptChoice(&selected, &customText).
//	    Option("Yes").
//	    Option("No").
//	    OnSelect(func(idx int, text string) {
//	        if idx == 0 {
//	            // User selected "Yes"
//	        }
//	    })
//
// # With Input Option
//
//	view := PromptChoice(&selected, &customText).
//	    Option("Approve").
//	    Option("Reject").
//	    InputOption("Provide feedback...").
//	    OnSelect(func(idx int, text string) {
//	        if idx == 2 {
//	            // User typed custom feedback in 'text'
//	            fmt.Println("Feedback:", text)
//	        }
//	    }).
//	    OnCancel(func() {
//	        fmt.Println("Cancelled")
//	    })
//
// # Customization Example
//
//	view := PromptChoice(&selected, nil).
//	    Option("Option A").
//	    Option("Option B").
//	    CursorChar(">").
//	    ShowNumbers(false).
//	    HintText("Press Enter to confirm").
//	    Style(tui.NewStyle().WithForeground(tui.ColorCyan))
//
// # InlineApp Integration
//
// In an InlineApp, include PromptChoice in your LiveView:
//
//	func (app *App) LiveView() tui.View {
//	    return tui.Stack(
//	        tui.Text("Do you want to proceed?").Bold(),
//	        tui.Text(""),
//	        tui.PromptChoice(&app.selected, &app.input).
//	            Option("Yes").
//	            Option("No").
//	            InputOption("Other...").
//	            OnSelect(app.handleSelection).
//	            OnCancel(app.handleCancel),
//	    )
//	}
func PromptChoice(selected *int, inputText *string) *promptChoiceView {
	id := fmt.Sprintf("prompt_choice_%p", selected)
	return &promptChoiceView{
		id:          id,
		selected:    selected,
		inputText:   inputText,
		options:     []string{},
		style:       NewStyle(),
		inputStyle:  NewStyle().WithForeground(ColorBrightBlack),
		cursorStyle: NewStyle(),
		cursorChar:  "❯",
		showNumbers: true,
		hintText:    "Esc to cancel",
	}
}

// ID sets a custom ID for this view.
//
// The ID is used for focus management when multiple focusable elements
// exist in the same view. By default, an ID is generated from the selected
// pointer address.
//
// Example:
//
//	PromptChoice(&selected, nil).ID("main-menu")
func (p *promptChoiceView) ID(id string) *promptChoiceView {
	p.id = id
	return p
}

// Option adds a fixed option to the list.
//
// Options are displayed in the order they are added, with numbers starting at 1.
// Users can select an option by pressing its number key or navigating with arrows.
//
// Example:
//
//	PromptChoice(&selected, nil).
//	    Option("Save and exit").
//	    Option("Exit without saving").
//	    Option("Cancel")
func (p *promptChoiceView) Option(label string) *promptChoiceView {
	p.options = append(p.options, label)
	return p
}

// Options adds multiple fixed options at once.
//
// This is a convenience method for adding several options in one call.
//
// Example:
//
//	PromptChoice(&selected, nil).Options("Yes", "No", "Maybe")
func (p *promptChoiceView) Options(labels ...string) *promptChoiceView {
	p.options = append(p.options, labels...)
	return p
}

// InputOption adds an option that allows inline text input.
//
// The input option is always positioned as the last option. When selected,
// the user can type directly and the text is stored in the inputText pointer
// provided to PromptChoice.
//
// The label parameter is shown as placeholder text when no input has been typed.
// As the user types, the placeholder is replaced with their input.
//
// Example:
//
//	PromptChoice(&selected, &customText).
//	    Option("Use default").
//	    InputOption("Enter custom value...")
func (p *promptChoiceView) InputOption(label string) *promptChoiceView {
	p.inputLabel = label
	return p
}

// OnSelect sets the callback invoked when an option is selected.
//
// The callback is triggered when the user presses Enter. It receives:
//   - idx: The zero-based index of the selected option
//   - inputText: The typed text if the input option was selected, otherwise empty
//
// Example:
//
//	PromptChoice(&selected, &text).
//	    Option("Approve").
//	    Option("Reject").
//	    InputOption("Custom...").
//	    OnSelect(func(idx int, inputText string) {
//	        switch idx {
//	        case 0:
//	            approve()
//	        case 1:
//	            reject()
//	        case 2:
//	            handleCustom(inputText)
//	        }
//	    })
func (p *promptChoiceView) OnSelect(fn func(idx int, inputText string)) *promptChoiceView {
	p.onSelect = fn
	return p
}

// OnCancel sets the callback invoked when Escape is pressed.
//
// Use this to handle cancellation, such as reverting to a previous state
// or closing the dialog.
//
// Example:
//
//	PromptChoice(&selected, nil).
//	    Option("Confirm").
//	    OnCancel(func() {
//	        app.showMenu = false
//	    })
func (p *promptChoiceView) OnCancel(fn func()) *promptChoiceView {
	p.onCancel = fn
	return p
}

// Style sets the style for option text.
//
// This style applies to the option labels (not the cursor or numbers).
//
// Example:
//
//	PromptChoice(&selected, nil).
//	    Option("Delete").
//	    Style(tui.NewStyle().WithForeground(tui.ColorRed))
func (p *promptChoiceView) Style(s Style) *promptChoiceView {
	p.style = s
	return p
}

// InputStyle sets the style for the input option placeholder text.
//
// This style is used when the input option is empty and showing the placeholder.
// Once the user starts typing, the regular Style is used.
//
// Example:
//
//	PromptChoice(&selected, &text).
//	    InputOption("Type here...").
//	    InputStyle(tui.NewStyle().WithForeground(tui.ColorBrightBlack).WithItalic())
func (p *promptChoiceView) InputStyle(s Style) *promptChoiceView {
	p.inputStyle = s
	return p
}

// CursorStyle sets the style for the cursor indicator.
//
// Example:
//
//	PromptChoice(&selected, nil).
//	    CursorStyle(tui.NewStyle().WithForeground(tui.ColorCyan).WithBold())
func (p *promptChoiceView) CursorStyle(s Style) *promptChoiceView {
	p.cursorStyle = s
	return p
}

// CursorChar sets the cursor indicator character.
//
// The default is "❯". Common alternatives include ">", "→", "•", or "*".
//
// Example:
//
//	PromptChoice(&selected, nil).CursorChar(">")
func (p *promptChoiceView) CursorChar(c string) *promptChoiceView {
	p.cursorChar = c
	return p
}

// ShowNumbers enables or disables numbered options.
//
// When enabled (default), options are prefixed with "1. ", "2. ", etc.
// Users can press number keys to jump directly to an option.
// When disabled, options show without numbers but number key navigation
// still works.
//
// Example:
//
//	// Without numbers:
//	// ❯ Yes
//	//   No
//	PromptChoice(&selected, nil).
//	    Option("Yes").
//	    Option("No").
//	    ShowNumbers(false)
func (p *promptChoiceView) ShowNumbers(show bool) *promptChoiceView {
	p.showNumbers = show
	return p
}

// HintText sets the hint text shown at the bottom.
//
// The default is "Esc to cancel". Set to an empty string to hide the hint.
//
// Example:
//
//	PromptChoice(&selected, nil).HintText("Enter to confirm, Esc to go back")
func (p *promptChoiceView) HintText(text string) *promptChoiceView {
	p.hintText = text
	return p
}

// Width sets a fixed width for the view.
//
// By default, the width is calculated from the longest option.
// Setting a fixed width can help with layout consistency.
//
// Example:
//
//	PromptChoice(&selected, nil).Width(40)
func (p *promptChoiceView) Width(w int) *promptChoiceView {
	p.width = w
	return p
}

// totalOptions returns the total number of options including input option if present
func (p *promptChoiceView) totalOptions() int {
	n := len(p.options)
	if p.inputLabel != "" {
		n++
	}
	return n
}

// inputOptionIndex returns the index of the input option, or -1 if none
func (p *promptChoiceView) inputOptionIndex() int {
	if p.inputLabel == "" {
		return -1
	}
	return len(p.options) // Input option is always last
}

// isInputSelected returns true if the input option is currently selected
func (p *promptChoiceView) isInputSelected() bool {
	if p.selected == nil || p.inputLabel == "" {
		return false
	}
	return *p.selected == p.inputOptionIndex()
}

// Focusable interface implementation

func (p *promptChoiceView) FocusID() string {
	return p.id
}

func (p *promptChoiceView) IsFocused() bool {
	return p.focused
}

func (p *promptChoiceView) SetFocused(focused bool) {
	p.focused = focused
	// Update TextInput focus state
	if ti := promptChoiceRegistry.Get(p.id); ti != nil {
		ti.SetFocused(focused && p.isInputSelected())
	}
}

func (p *promptChoiceView) FocusBounds() image.Rectangle {
	return p.bounds
}

func (p *promptChoiceView) HandleKeyEvent(event KeyEvent) bool {
	total := p.totalOptions()
	if total == 0 {
		return false
	}

	selected := 0
	if p.selected != nil {
		selected = *p.selected
	}

	// Handle Escape first
	if event.Key == KeyEscape {
		if p.onCancel != nil {
			p.onCancel()
		}
		return true
	}

	// Handle navigation with arrow keys (always works, even when on input option)
	switch event.Key {
	case KeyArrowUp:
		if selected > 0 {
			*p.selected = selected - 1
			p.updateInputFocus()
		}
		return true
	case KeyArrowDown:
		if selected < total-1 {
			*p.selected = selected + 1
			p.updateInputFocus()
		}
		return true
	case KeyEnter:
		if p.onSelect != nil {
			inputText := ""
			if p.inputText != nil {
				inputText = *p.inputText
			}
			p.onSelect(selected, inputText)
		}
		return true
	}

	// Handle number keys (1-9) for quick selection
	if event.Rune >= '1' && event.Rune <= '9' {
		idx := int(event.Rune - '1')
		if idx < total {
			*p.selected = idx
			p.updateInputFocus()
			return true
		}
	}

	// If input option is selected, route character input to TextInput
	if p.isInputSelected() {
		ti := promptChoiceRegistry.Get(p.id)
		if ti != nil {
			// Handle paste events
			if event.Paste != "" {
				ti.HandlePaste(event.Paste)
				if p.inputText != nil {
					*p.inputText = ti.Value()
				}
				return true
			}
			// Route to TextInput for character input and editing
			if ti.HandleKey(event) {
				if p.inputText != nil {
					*p.inputText = ti.Value()
				}
				return true
			}
		}
	}

	return false
}

// updateInputFocus updates the TextInput focus based on current selection
func (p *promptChoiceView) updateInputFocus() {
	if ti := promptChoiceRegistry.Get(p.id); ti != nil {
		ti.SetFocused(p.focused && p.isInputSelected())
	}
}

func (p *promptChoiceView) size(maxWidth, maxHeight int) (int, int) {
	// Calculate width from options
	w := p.width
	if w == 0 {
		// cursor + space + number + dot + space + label
		prefixW := 2 // cursor space
		if p.showNumbers {
			prefixW += 3 // "1. "
		}
		for _, opt := range p.options {
			optW, _ := MeasureText(opt)
			if optW+prefixW > w {
				w = optW + prefixW
			}
		}
		if p.inputLabel != "" {
			optW, _ := MeasureText(p.inputLabel)
			if optW+prefixW > w {
				w = optW + prefixW
			}
		}
	}

	// Height = options + blank line + hint (if present)
	h := p.totalOptions()
	if p.hintText != "" {
		h += 2 // blank line + hint
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (p *promptChoiceView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Register with focus manager
	p.bounds = ctx.AbsoluteBounds()
	if fm := ctx.FocusManager(); fm != nil {
		fm.Register(p)
	}

	selected := 0
	if p.selected != nil {
		selected = *p.selected
	}

	// Sync input text to TextInput
	ti := promptChoiceRegistry.Get(p.id)
	if p.inputText != nil && ti.Value() != *p.inputText {
		ti.SetValue(*p.inputText)
	}

	y := 0

	// Render fixed options
	for i, opt := range p.options {
		if y >= height {
			break
		}
		p.renderOption(ctx, y, i, opt, i == selected, false, width)
		y++
	}

	// Render input option if present
	if p.inputLabel != "" && y < height {
		inputIdx := p.inputOptionIndex()
		isSelected := inputIdx == selected
		p.renderInputOption(ctx, y, inputIdx, isSelected, width, ti)
		y++
	}

	// Render hint text
	if p.hintText != "" && y+1 < height {
		y++ // blank line
		hintStyle := NewStyle().WithForeground(ColorBrightBlack)
		ctx.PrintStyled(0, y, p.hintText, hintStyle)
	}
}

func (p *promptChoiceView) renderOption(ctx *RenderContext, y, idx int, label string, isSelected bool, isInput bool, width int) {
	x := 0
	style := p.style

	// Draw cursor
	if isSelected {
		ctx.PrintStyled(x, y, p.cursorChar, p.cursorStyle)
	}
	x += 2 // cursor + space

	// Draw number
	if p.showNumbers {
		numStr := fmt.Sprintf("%d. ", idx+1)
		ctx.PrintStyled(x, y, numStr, style)
		x += len(numStr)
	}

	// Draw label
	ctx.PrintTruncated(x, y, label, style)
}

func (p *promptChoiceView) renderInputOption(ctx *RenderContext, y, idx int, isSelected bool, width int, ti *TextInput) {
	x := 0

	// Draw cursor
	if isSelected {
		ctx.PrintStyled(x, y, p.cursorChar, p.cursorStyle)
	}
	x += 2 // cursor + space

	// Draw number
	if p.showNumbers {
		numStr := fmt.Sprintf("%d. ", idx+1)
		ctx.PrintStyled(x, y, numStr, p.style)
		x += len(numStr)
	}

	// Calculate available width for input
	inputWidth := width - x
	if inputWidth <= 0 {
		inputWidth = 20
	}

	// Draw input content or placeholder
	inputValue := ""
	if p.inputText != nil {
		inputValue = *p.inputText
	}

	if inputValue == "" {
		// Show placeholder
		ctx.PrintTruncated(x, y, p.inputLabel, p.inputStyle)
	} else {
		// Show input value
		ctx.PrintTruncated(x, y, inputValue, p.style)
	}

	// Draw cursor if this option is selected and focused
	if isSelected && p.focused {
		cursorX := x + ti.CursorPos
		if inputValue == "" {
			cursorX = x // At start when empty
		}
		if cursorX < width {
			// Draw block cursor
			charUnderCursor := " "
			if inputValue == "" && len(p.inputLabel) > 0 {
				charUnderCursor = string([]rune(p.inputLabel)[0])
			} else if ti.CursorPos < len(inputValue) {
				charUnderCursor = string([]rune(inputValue)[ti.CursorPos])
			}
			cursorStyle := NewStyle().WithBackground(ColorWhite).WithForeground(ColorBlack)
			ctx.PrintStyled(cursorX, y, charUnderCursor, cursorStyle)
		}
	}
}
