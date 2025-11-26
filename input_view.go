package gooey

import (
	"fmt"
	"image"
	"sync"
)

// InputRegistry manages text inputs for focus and key routing.
var inputRegistry = &inputRegistryImpl{
	inputs:  make(map[string]*inputState),
	focused: "",
}

type inputRegistryImpl struct {
	mu      sync.Mutex
	inputs  map[string]*inputState
	focused string
	order   []string // track registration order for tab navigation
}

type inputState struct {
	id          string
	input       *TextInput
	binding     *string
	bounds      image.Rectangle
	onChange    func(string)
	onSubmit    func(string)
	placeholder string
}

// Clear clears all registered inputs (called before each render).
func (r *inputRegistryImpl) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = r.order[:0]
}

// Register adds or updates an input.
func (r *inputRegistryImpl) Register(id string, binding *string, bounds image.Rectangle, placeholder string, mask rune, onChange, onSubmit func(string)) *inputState {
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
		// Sync initial value from binding
		if binding != nil && *binding != "" {
			ti.SetValue(*binding)
		}
		state = &inputState{
			id:          id,
			input:       ti,
			binding:     binding,
			placeholder: placeholder,
		}
		r.inputs[id] = state
	}

	// Update state
	state.bounds = bounds
	state.onChange = onChange
	state.onSubmit = onSubmit
	state.binding = binding

	// Sync value from binding
	if binding != nil {
		currentValue := state.input.Value()
		if *binding != currentValue {
			state.input.SetValue(*binding)
		}
	}

	// Track registration order
	r.order = append(r.order, id)

	// Auto-focus first input if none focused
	if r.focused == "" {
		r.focused = id
		state.input.SetFocused(true)
	} else {
		state.input.SetFocused(r.focused == id)
	}

	return state
}

// GetFocused returns the currently focused input.
func (r *inputRegistryImpl) GetFocused() *inputState {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.focused == "" {
		return nil
	}
	return r.inputs[r.focused]
}

// SetFocus sets focus to a specific input.
func (r *inputRegistryImpl) SetFocus(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Unfocus previous
	if prev, exists := r.inputs[r.focused]; exists {
		prev.input.SetFocused(false)
	}

	r.focused = id

	// Focus new
	if next, exists := r.inputs[id]; exists {
		next.input.SetFocused(true)
	}
}

// FocusNext moves focus to the next input.
func (r *inputRegistryImpl) FocusNext() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.order) == 0 {
		return
	}

	currentIdx := -1
	for i, id := range r.order {
		if id == r.focused {
			currentIdx = i
			break
		}
	}

	// Unfocus current
	if prev, exists := r.inputs[r.focused]; exists {
		prev.input.SetFocused(false)
	}

	// Focus next (wrap around)
	nextIdx := (currentIdx + 1) % len(r.order)
	r.focused = r.order[nextIdx]

	if next, exists := r.inputs[r.focused]; exists {
		next.input.SetFocused(true)
	}
}

// HandleKey routes a key event to the focused input.
// Returns true if the event was handled.
func (r *inputRegistryImpl) HandleKey(event KeyEvent) bool {
	r.mu.Lock()
	state := r.inputs[r.focused]
	r.mu.Unlock()

	if state == nil {
		return false
	}

	// Handle Tab for focus navigation
	if event.Key == KeyTab && !event.Shift {
		r.FocusNext()
		return true
	}

	// Handle Enter for submit
	if event.Key == KeyEnter && !event.Shift {
		if state.onSubmit != nil {
			state.onSubmit(state.input.Value())
		}
		return true
	}

	// Route to TextInput
	handled := state.input.HandleKey(event)

	// Sync value back to binding
	if handled && state.binding != nil {
		*state.binding = state.input.Value()
		if state.onChange != nil {
			state.onChange(*state.binding)
		}
	}

	return handled
}

// HandleClick checks if a click hit any input and focuses it.
func (r *inputRegistryImpl) HandleClick(x, y int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	pt := image.Pt(x, y)
	for id, state := range r.inputs {
		if pt.In(state.bounds) {
			// Unfocus previous
			if prev, exists := r.inputs[r.focused]; exists {
				prev.input.SetFocused(false)
			}
			r.focused = id
			state.input.SetFocused(true)
			return true
		}
	}
	return false
}

// inputView wraps a TextInput for declarative use
type inputView struct {
	id          string
	binding     *string
	placeholder string
	mask        rune
	onChange    func(string)
	onSubmit    func(string)
	width       int
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

func (i *inputView) size(maxWidth, maxHeight int) (int, int) {
	w := i.width
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (i *inputView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	// Register this input
	state := inputRegistry.Register(i.id, i.binding, bounds, i.placeholder, i.mask, i.onChange, i.onSubmit)

	// Update TextInput bounds
	state.input.SetBounds(bounds)

	// Draw the TextInput
	subFrame := frame.SubFrame(bounds)
	state.input.Draw(subFrame)
}
