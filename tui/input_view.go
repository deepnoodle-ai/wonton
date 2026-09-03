package tui

import (
	"fmt"
	"image"
	"sync"

	"github.com/deepnoodle-ai/wonton/runewidth"
)

// inputRegistryImpl manages text input state (bindings, callbacks, etc.)
// Owned per-runtime via the registries struct. Focus management is delegated
// to the FocusManager passed via RenderContext.

type inputRegistryImpl struct {
	mu     sync.Mutex
	inputs map[string]*inputState
}

type inputState struct {
	id               string
	input            *textInput
	binding          *string
	bounds           image.Rectangle
	onChange         func(string)
	onSubmit         func(string)
	onKey            func(KeyEvent) bool
	onComplete       func(string) []string
	placeholder      string
	placeholderStyle *Style
	pastePlaceholder bool
	cursorBlink      bool
	multiline        bool
	maxHeight        int
	focused          bool

	// History recall state. historyIdx is -1 while editing the live draft;
	// otherwise it indexes into history. historyDraft preserves the live
	// text while navigating history so ArrowDown past the newest entry
	// restores it.
	history      []string
	historyIdx   int
	historyDraft string

	// Completion state. completions is non-nil while cycling candidates;
	// completionOrig is the text before completion started, restored on Esc.
	completions    []string
	completionIdx  int
	completionOrig string
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

// HandleKeyEvent processes a key event for a focused input. Dispatch order:
//
//  1. OnKey hook — the application sees every key first and may consume it.
//  2. Completion — Tab starts/cycles candidates; while cycling, arrows also
//     cycle, Esc restores the original text, and any other key accepts the
//     current candidate and is then processed normally.
//  3. Enter — submits (Shift+Enter falls through for multiline newlines).
//  4. History — Up/Down recall entries when the cursor can't move within
//     the text (always, in single-line mode; at the first/last visual line
//     in multiline mode).
//  5. Core editing.
//
// Returns false for keys the input doesn't use, so they propagate to the
// application's HandleEvent.
func (s *inputState) HandleKeyEvent(event KeyEvent) bool {
	// Paste events go straight to the input; OnKey is for keystrokes.
	if event.Paste != "" {
		s.completions = nil // a paste accepts any in-progress completion
		handled := s.input.HandlePaste(event.Paste)
		if handled {
			s.syncBinding()
		}
		return handled
	}

	// 1. Application hook.
	if s.onKey != nil && s.onKey(event) {
		return true
	}

	// 2. Completion.
	if s.onComplete != nil && s.handleCompletionKey(event) {
		return true
	}

	// 3. Enter for submit (unless Shift is pressed for multiline newlines).
	if event.Key == KeyEnter && !event.Shift {
		s.completions = nil
		s.historyIdx = -1
		s.historyDraft = ""
		if s.onSubmit != nil {
			s.onSubmit(s.input.Value())
		}
		return true
	}

	// 4. History recall.
	if s.handleHistoryKey(event) {
		return true
	}

	// 5. Core editing.
	handled := s.input.HandleKey(event)
	if handled {
		s.completions = nil // an edit accepts any in-progress completion
		s.syncBinding()
	}
	return handled
}

// syncBinding copies the input value back to the bound string and fires
// OnChange.
func (s *inputState) syncBinding() {
	if s.binding != nil {
		*s.binding = s.input.Value()
		if s.onChange != nil {
			s.onChange(*s.binding)
		}
	}
}

// setText replaces the input text (cursor moves to the end) and syncs the
// binding. Used by history recall and completion.
func (s *inputState) setText(value string) {
	s.input.SetValue(value)
	s.syncBinding()
}

// handleCompletionKey implements Tab completion with in-buffer cycling.
// Tab asks OnComplete for candidates: a single candidate is accepted
// immediately; multiple candidates are cycled in place with Tab/Shift+Tab
// or Down/Up. Esc restores the text from before completion started.
// Returns true if the key was consumed.
func (s *inputState) handleCompletionKey(event KeyEvent) bool {
	if event.Key == KeyTab && !event.Shift && s.completions == nil {
		candidates := s.onComplete(s.input.Value())
		switch len(candidates) {
		case 0:
			// Nothing to offer, but the field still claims Tab so behavior
			// doesn't flip between consuming and focus-cycling based on
			// the callback's result.
			return true
		case 1:
			s.setText(candidates[0])
			return true
		default:
			s.completionOrig = s.input.Value()
			s.completions = candidates
			s.completionIdx = 0
			s.setText(candidates[0])
			return true
		}
	}

	if s.completions == nil {
		return false
	}

	// Cycling candidates: Tab/Down advance, Shift+Tab/Up go back.
	forward := (event.Key == KeyTab && !event.Shift) || event.Key == KeyArrowDown
	backward := (event.Key == KeyTab && event.Shift) || event.Key == KeyArrowUp
	switch {
	case forward:
		s.completionIdx = (s.completionIdx + 1) % len(s.completions)
		s.setText(s.completions[s.completionIdx])
		return true
	case backward:
		s.completionIdx = (s.completionIdx - 1 + len(s.completions)) % len(s.completions)
		s.setText(s.completions[s.completionIdx])
		return true
	case event.Key == KeyEscape:
		s.setText(s.completionOrig)
		s.completions = nil
		return true
	}
	return false
}

// handleHistoryKey implements Up/Down history recall. Recall only engages
// when the cursor can't move within the text, so in multiline mode arrows
// first navigate the buffer (like zsh/fish). The live draft is preserved
// while navigating and restored when moving past the newest entry. Edits to
// a recalled entry persist only until navigating away.
func (s *inputState) handleHistoryKey(event KeyEvent) bool {
	if len(s.history) == 0 {
		return false
	}
	// The history slice is refreshed each render and may have changed.
	if s.historyIdx >= len(s.history) {
		s.historyIdx = -1
	}

	switch event.Key {
	case KeyArrowUp:
		if !s.input.atFirstVisualLine() {
			return false // cursor can still move up within the text
		}
		switch {
		case s.historyIdx == -1:
			s.historyDraft = s.input.Value()
			s.historyIdx = len(s.history) - 1
		case s.historyIdx > 0:
			s.historyIdx--
		default:
			return true // already at the oldest entry
		}
		s.setText(s.history[s.historyIdx])
		return true
	case KeyArrowDown:
		if s.historyIdx == -1 {
			return false // editing the live draft: nothing newer
		}
		if !s.input.atLastVisualLine() {
			return false // cursor can still move down within the text
		}
		if s.historyIdx < len(s.history)-1 {
			s.historyIdx++
			s.setText(s.history[s.historyIdx])
		} else {
			s.historyIdx = -1
			s.setText(s.historyDraft)
			s.historyDraft = ""
		}
		return true
	}
	return false
}

// Clear clears input tracking (called before each render).
// lookup returns the persistent state for id, if any.
func (r *inputRegistryImpl) lookup(id string) (*inputState, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.inputs[id]
	return state, ok
}

func (r *inputRegistryImpl) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Note: we don't clear the inputs map, just let focus manager handle order
}

// inputConfig carries the per-render configuration for an input, gathered
// from the view builders (Input, InputField).
type inputConfig struct {
	binding          *string
	bounds           image.Rectangle
	placeholder      string
	placeholderStyle *Style
	textStyle        *Style
	mask             rune
	pastePlaceholder bool
	cursorBlink      bool
	multiline        bool
	maxHeight        int
	onChange         func(string)
	onSubmit         func(string)
	onKey            func(KeyEvent) bool
	onComplete       func(string) []string
	history          []string
}

// Register adds or updates an input.
func (r *inputRegistryImpl) Register(id string, cfg inputConfig, fm *FocusManager) *inputState {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.inputs[id]
	if !exists {
		// Create new textInput widget
		ti := newTextInput()
		if cfg.mask != 0 {
			ti.WithMask(cfg.mask)
		}
		if cfg.placeholder != "" {
			ti.WithPlaceholder(cfg.placeholder)
		}
		if cfg.placeholderStyle != nil {
			ti.PlaceholderStyle = *cfg.placeholderStyle
		}
		if cfg.textStyle != nil {
			ti.Style = *cfg.textStyle
		}
		if cfg.pastePlaceholder {
			ti.WithPastePlaceholderMode(true)
		}
		if cfg.cursorBlink {
			ti.WithCursorBlink(true)
		}
		if cfg.multiline {
			ti.WithMultilineMode(true)
		}
		if cfg.maxHeight > 0 {
			ti.WithMaxHeight(cfg.maxHeight)
		}
		// Sync initial value from binding
		if cfg.binding != nil && *cfg.binding != "" {
			ti.SetValue(*cfg.binding)
		}
		state = &inputState{
			id:               id,
			input:            ti,
			binding:          cfg.binding,
			placeholder:      cfg.placeholder,
			placeholderStyle: cfg.placeholderStyle,
			pastePlaceholder: cfg.pastePlaceholder,
			cursorBlink:      cfg.cursorBlink,
			multiline:        cfg.multiline,
			maxHeight:        cfg.maxHeight,
			historyIdx:       -1,
		}
		r.inputs[id] = state
	}

	// Update state
	state.bounds = cfg.bounds
	state.onChange = cfg.onChange
	state.onSubmit = cfg.onSubmit
	state.onKey = cfg.onKey
	state.onComplete = cfg.onComplete
	state.history = cfg.history
	state.binding = cfg.binding

	// Sync multiline mode (in case it changed)
	state.input.MultilineMode = cfg.multiline
	state.multiline = cfg.multiline

	// Sync max height (in case it changed)
	state.input.MaxHeight = cfg.maxHeight
	state.maxHeight = cfg.maxHeight

	// Sync value from binding. A text change from outside the input (the
	// app mutated the bound string, e.g. clearing it on submit or from an
	// OnKey hook) invalidates any in-progress completion cycling.
	if cfg.binding != nil {
		currentValue := state.input.Value()
		if *cfg.binding != currentValue {
			state.input.SetValue(*cfg.binding)
			state.completions = nil
		}
	}

	// Register with the focus manager (if available)
	if fm != nil {
		fm.Register(state)
	}

	return state
}

// GetFocused returns the currently focused input state (if any input is focused).
func (r *inputRegistryImpl) GetFocused(fm *FocusManager) *inputState {
	if fm == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	focused := fm.GetFocused()
	if focused == nil {
		return nil
	}

	// Check if the focused element is an input
	if state, ok := focused.(*inputState); ok {
		return state
	}
	return nil
}

// InputView wraps a textInput for declarative use
type InputView struct {
	reg              *registries // captured at construction; refreshed from ctx during render
	id               string
	binding          *string
	placeholder      string
	placeholderStyle *Style
	mask             rune
	onChange         func(string)
	onSubmit         func(string)
	onKey            func(KeyEvent) bool
	onComplete       func(string) []string
	history          []string
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
func Input(binding *string) *InputView {
	// Generate a unique ID based on the pointer address
	id := ""
	if binding != nil {
		id = fmt.Sprintf("input_%p", binding)
	}
	return &InputView{
		reg:     capturedRegistries(),
		id:      id,
		binding: binding,
		width:   20,
	}
}

// ID sets a specific ID for this input (useful for focus management).
func (i *InputView) ID(id string) *InputView {
	i.id = id
	return i
}

// Placeholder sets the placeholder text shown when empty.
func (i *InputView) Placeholder(text string) *InputView {
	i.placeholder = text
	return i
}

// PlaceholderStyle sets the style for the placeholder text.
func (i *InputView) PlaceholderStyle(style Style) *InputView {
	i.placeholderStyle = &style
	return i
}

// Mask sets a mask character for password input.
func (i *InputView) Mask(r rune) *InputView {
	i.mask = r
	return i
}

// OnChange sets a callback invoked when the value changes.
func (i *InputView) OnChange(fn func(string)) *InputView {
	i.onChange = fn
	return i
}

// OnSubmit sets a callback invoked when Enter is pressed.
func (i *InputView) OnSubmit(fn func(string)) *InputView {
	i.onSubmit = fn
	return i
}

// OnKey sets a hook that sees every key event before the input's own
// handling (completion, history, submit, editing). Return true to consume
// the event; return false to let the input process it normally. Use this to
// claim specific keys for application shortcuts while the input is focused.
func (i *InputView) OnKey(fn func(KeyEvent) bool) *InputView {
	i.onKey = fn
	return i
}

// History enables Up/Down recall of previous entries, oldest first.
// Recall engages when the cursor can't move within the text: always in
// single-line mode, and at the first/last visual line in multiline mode
// (arrows navigate the text first, like zsh/fish). The in-progress draft is
// preserved while navigating and restored when moving past the newest entry.
// Pass the current history slice on each render; appending submitted values
// to it is the application's responsibility.
func (i *InputView) History(items []string) *InputView {
	i.history = items
	return i
}

// OnComplete enables Tab completion. The callback receives the current
// input text and returns candidate replacements. A single candidate is
// accepted immediately; multiple candidates are cycled in place with
// Tab/Shift+Tab or Down/Up, and Esc restores the original text. Any other
// key accepts the shown candidate and is processed normally.
//
// While OnComplete is set the input claims Tab, so Tab no longer cycles
// focus while this input is focused.
func (i *InputView) OnComplete(fn func(value string) []string) *InputView {
	i.onComplete = fn
	return i
}

// Width sets the display width of the input.
func (i *InputView) Width(w int) *InputView {
	i.width = w
	return i
}

// PastePlaceholder enables paste placeholder mode.
// When enabled, multi-line pastes are collapsed into "[pasted N lines]"
// placeholders that can be deleted atomically with backspace.
func (i *InputView) PastePlaceholder(enabled bool) *InputView {
	i.pastePlaceholder = enabled
	return i
}

// CursorBlink enables or disables cursor blinking.
func (i *InputView) CursorBlink(enabled bool) *InputView {
	i.cursorBlink = enabled
	return i
}

// Multiline enables multiline input where Shift+Enter inserts newlines.
func (i *InputView) Multiline(enabled bool) *InputView {
	i.multiline = enabled
	return i
}

// MaxHeight sets the maximum height in lines for a multiline input.
// When content exceeds this height, the input becomes scrollable.
// Overflow indicators (▲/▼) show when content exists above/below.
// A value of 0 means unlimited height (default).
func (i *InputView) MaxHeight(lines int) *InputView {
	i.maxHeight = lines
	return i
}

// Style sets the style for the input text.
func (i *InputView) Style(s Style) *InputView {
	i.style = &s
	return i
}

// FocusStyle sets the style applied when this input is focused.
// If not set, the normal style is used.
func (i *InputView) FocusStyle(s Style) *InputView {
	i.focusStyle = &s
	return i
}

// calcWrappedHeight calculates how many lines text will take when wrapped at width.
// Iterates grapheme clusters (not runes) so multi-rune clusters like emoji ZWJ
// sequences, keycaps, and base+combining marks don't get miscounted.
func calcWrappedHeight(text string, width int) int {
	if width <= 0 || text == "" {
		return 1
	}

	lines := 1
	x := 0
	for cluster, charWidth := range runewidth.Graphemes(text) {
		if cluster == "\n" {
			lines++
			x = 0
			continue
		}
		if x > 0 && x+charWidth > width {
			lines++
			x = charWidth
		} else {
			x += charWidth
		}
	}
	return lines
}

func (i *InputView) size(maxWidth, maxHeight int) (int, int) {
	w := i.width
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}

	// Calculate height based on content wrapping
	h := 1
	if i.binding != nil && *i.binding != "" && w > 0 {
		// Get the display text (need to check registry for paste placeholders)
		displayText := *i.binding
		if state, exists := i.reg.inputs.lookup(i.id); exists {
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

func (i *InputView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Determine if this input is focused
	fm := ctx.FocusManager()
	isFocused := fm != nil && fm.GetFocusedID() == i.id

	// Register this input - use absolute bounds for click registration
	inputBounds := ctx.AbsoluteBounds()
	i.reg = ctx.registries()
	state := i.reg.inputs.Register(i.id, inputConfig{
		binding:          i.binding,
		bounds:           inputBounds,
		placeholder:      i.placeholder,
		placeholderStyle: i.placeholderStyle,
		mask:             i.mask,
		pastePlaceholder: i.pastePlaceholder,
		cursorBlink:      i.cursorBlink,
		multiline:        i.multiline,
		maxHeight:        i.maxHeight,
		onChange:         i.onChange,
		onSubmit:         i.onSubmit,
		onKey:            i.onKey,
		onComplete:       i.onComplete,
		history:          i.history,
	}, fm)

	// Apply focus-aware styling to the textInput
	if isFocused && i.focusStyle != nil {
		state.input.Style = *i.focusStyle
	} else if i.style != nil {
		state.input.Style = *i.style
	}

	// Update textInput bounds
	state.input.SetBounds(inputBounds)

	// Draw the textInput - pass the underlying frame
	state.input.Draw(ctx.frame)
}
