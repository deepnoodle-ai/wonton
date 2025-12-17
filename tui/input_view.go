package tui

import (
	"fmt"
	"image"
	"sync"

	"github.com/mattn/go-runewidth"
)

// inputRegistry manages text input state (bindings, callbacks, etc.)
// Focus management is delegated to the global focusManager.
var inputRegistry = &inputRegistryImpl{
	inputs: make(map[string]*inputState),
}

type inputRegistryImpl struct {
	mu     sync.Mutex
	inputs map[string]*inputState
}

type inputState struct {
	id               string
	input            *TextInput
	binding          *string
	bounds           image.Rectangle
	onChange         func(string)
	onSubmit         func(string)
	placeholder      string
	placeholderStyle *Style
	pastePlaceholder bool
	cursorBlink      bool
	multiline        bool
	maxHeight        int
	focused          bool
}

// Focusable interface implementation for inputState

func (s *inputState) FocusID() string {
	return s.id
}

func (s *inputState) IsFocused() bool {
	return s.focused
}

func (s *inputState) SetFocused(focused bool) {
	s.focused = focused
	s.input.SetFocused(focused)
}

func (s *inputState) FocusBounds() image.Rectangle {
	return s.bounds
}

func (s *inputState) HandleKeyEvent(event KeyEvent) bool {
	// Handle paste events
	if event.Paste != "" {
		handled := s.input.HandlePaste(event.Paste)
		if handled && s.binding != nil {
			*s.binding = s.input.Value()
			if s.onChange != nil {
				s.onChange(*s.binding)
			}
		}
		return handled
	}

	// Handle Enter for submit (but not in multiline mode with Shift)
	if event.Key == KeyEnter && !event.Shift && !s.multiline {
		if s.onSubmit != nil {
			s.onSubmit(s.input.Value())
		}
		return true
	}

	// Route to TextInput
	handled := s.input.HandleKey(event)

	// Sync value back to binding
	if handled && s.binding != nil {
		*s.binding = s.input.Value()
		if s.onChange != nil {
			s.onChange(*s.binding)
		}
	}

	return handled
}

// Clear clears input tracking (called before each render).
func (r *inputRegistryImpl) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Note: we don't clear the inputs map, just let focusManager handle order
}

// Register adds or updates an input.
func (r *inputRegistryImpl) Register(id string, binding *string, bounds image.Rectangle, placeholder string, placeholderStyle *Style, mask rune, pastePlaceholder bool, cursorBlink bool, multiline bool, maxHeight int, onChange, onSubmit func(string)) *inputState {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.inputs[id]
	if !exists {
		// Create new TextInput widget
		ti := NewTextInput()
		if mask != 0 {
			ti.WithMask(mask)
		}
		if placeholder != "" {
			ti.WithPlaceholder(placeholder)
		}
		if placeholderStyle != nil {
			ti.PlaceholderStyle = *placeholderStyle
		}
		if pastePlaceholder {
			ti.WithPastePlaceholderMode(true)
		}
		if cursorBlink {
			ti.WithCursorBlink(true)
		}
		if multiline {
			ti.WithMultilineMode(true)
		}
		if maxHeight > 0 {
			ti.WithMaxHeight(maxHeight)
		}
		// Sync initial value from binding
		if binding != nil && *binding != "" {
			ti.SetValue(*binding)
		}
		state = &inputState{
			id:               id,
			input:            ti,
			binding:          binding,
			placeholder:      placeholder,
			placeholderStyle: placeholderStyle,
			pastePlaceholder: pastePlaceholder,
			cursorBlink:      cursorBlink,
			multiline:        multiline,
			maxHeight:        maxHeight,
		}
		r.inputs[id] = state
	}

	// Update state
	state.bounds = bounds
	state.onChange = onChange
	state.onSubmit = onSubmit
	state.binding = binding

	// Sync multiline mode (in case it changed)
	state.input.MultilineMode = multiline
	state.multiline = multiline

	// Sync max height (in case it changed)
	state.input.MaxHeight = maxHeight
	state.maxHeight = maxHeight

	// Sync value from binding
	if binding != nil {
		currentValue := state.input.Value()
		if *binding != currentValue {
			state.input.SetValue(*binding)
		}
	}

	// Register with the global focus manager
	focusManager.Register(state)

	return state
}

// GetFocused returns the currently focused input state (if any input is focused).
func (r *inputRegistryImpl) GetFocused() *inputState {
	r.mu.Lock()
	defer r.mu.Unlock()

	focused := focusManager.GetFocused()
	if focused == nil {
		return nil
	}

	// Check if the focused element is an input
	if state, ok := focused.(*inputState); ok {
		return state
	}
	return nil
}

// inputView wraps a TextInput for declarative use
type inputView struct {
	id               string
	binding          *string
	placeholder      string
	placeholderStyle *Style
	mask             rune
	onChange         func(string)
	onSubmit         func(string)
	width            int
	maxHeight        int // Maximum height in lines (0 = unlimited)
	pastePlaceholder bool
	cursorBlink      bool
	multiline        bool

	// Focus styling
	style      *Style // Normal style (optional)
	focusStyle *Style // Style when focused (optional)
}

// Input creates a text input view bound to a string pointer.
// Changes to the input will update the bound string.
//
// Example:
//
//	Input(&app.name).Placeholder("Enter name...")
func Input(binding *string) *inputView {
	// Generate a unique ID based on the pointer address
	id := ""
	if binding != nil {
		id = fmt.Sprintf("input_%p", binding)
	}
	return &inputView{
		id:      id,
		binding: binding,
		width:   20,
	}
}

// ID sets a specific ID for this input (useful for focus management).
func (i *inputView) ID(id string) *inputView {
	i.id = id
	return i
}

// Placeholder sets the placeholder text shown when empty.
func (i *inputView) Placeholder(text string) *inputView {
	i.placeholder = text
	return i
}

// PlaceholderStyle sets the style for the placeholder text.
func (i *inputView) PlaceholderStyle(style Style) *inputView {
	i.placeholderStyle = &style
	return i
}

// Mask sets a mask character for password input.
func (i *inputView) Mask(r rune) *inputView {
	i.mask = r
	return i
}

// OnChange sets a callback invoked when the value changes.
func (i *inputView) OnChange(fn func(string)) *inputView {
	i.onChange = fn
	return i
}

// OnSubmit sets a callback invoked when Enter is pressed.
func (i *inputView) OnSubmit(fn func(string)) *inputView {
	i.onSubmit = fn
	return i
}

// Width sets the display width of the input.
func (i *inputView) Width(w int) *inputView {
	i.width = w
	return i
}

// PastePlaceholder enables paste placeholder mode.
// When enabled, multi-line pastes are collapsed into "[pasted N lines]"
// placeholders that can be deleted atomically with backspace.
func (i *inputView) PastePlaceholder(enabled bool) *inputView {
	i.pastePlaceholder = enabled
	return i
}

// CursorBlink enables or disables cursor blinking.
func (i *inputView) CursorBlink(enabled bool) *inputView {
	i.cursorBlink = enabled
	return i
}

// Multiline enables multiline input where Shift+Enter inserts newlines.
func (i *inputView) Multiline(enabled bool) *inputView {
	i.multiline = enabled
	return i
}

// MaxHeight sets the maximum height in lines for a multiline input.
// When content exceeds this height, the input becomes scrollable.
// Overflow indicators (▲/▼) show when content exists above/below.
// A value of 0 means unlimited height (default).
func (i *inputView) MaxHeight(lines int) *inputView {
	i.maxHeight = lines
	return i
}

// Style sets the style for the input text.
func (i *inputView) Style(s Style) *inputView {
	i.style = &s
	return i
}

// FocusStyle sets the style applied when this input is focused.
// If not set, the normal style is used.
func (i *inputView) FocusStyle(s Style) *inputView {
	i.focusStyle = &s
	return i
}

// calcWrappedHeight calculates how many lines text will take when wrapped at width
func calcWrappedHeight(text string, width int) int {
	if width <= 0 || text == "" {
		return 1
	}

	lines := 1
	x := 0
	for _, r := range text {
		if r == '\n' {
			lines++
			x = 0
			continue
		}
		charWidth := runewidth.StringWidth(string(r))
		if x+charWidth > width {
			lines++
			x = charWidth
		} else {
			x += charWidth
		}
	}
	return lines
}

func (i *inputView) size(maxWidth, maxHeight int) (int, int) {
	w := i.width
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}

	// Calculate height based on content wrapping
	h := 1
	if i.binding != nil && *i.binding != "" && w > 0 {
		// Get the display text (need to check registry for paste placeholders)
		displayText := *i.binding
		if state, exists := inputRegistry.inputs[i.id]; exists {
			displayText = state.input.DisplayText()
		}
		h = calcWrappedHeight(displayText, w)
	}

	// Apply max height constraint
	if i.maxHeight > 0 && h > i.maxHeight {
		h = i.maxHeight
	}

	return w, h
}

func (i *inputView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Determine if this input is focused
	isFocused := focusManager.GetFocusedID() == i.id

	// Register this input - use absolute bounds for click registration
	inputBounds := ctx.AbsoluteBounds()
	state := inputRegistry.Register(i.id, i.binding, inputBounds, i.placeholder, i.placeholderStyle, i.mask, i.pastePlaceholder, i.cursorBlink, i.multiline, i.maxHeight, i.onChange, i.onSubmit)

	// Apply focus-aware styling to the TextInput
	if isFocused && i.focusStyle != nil {
		state.input.Style = *i.focusStyle
	} else if i.style != nil {
		state.input.Style = *i.style
	}

	// Update TextInput bounds
	state.input.SetBounds(inputBounds)

	// Draw the TextInput - pass the underlying frame
	state.input.Draw(ctx.frame)
}
