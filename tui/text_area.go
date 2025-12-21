package tui

import (
	"fmt"
	"image"
	"strings"
	"sync"
)

// textAreaRegistry manages transient state for TextAreas.
var textAreaRegistry = &textAreaRegistryImpl{
	states: make(map[string]*textAreaState),
	active: make(map[string]bool),
}

type textAreaRegistryImpl struct {
	mu     sync.Mutex
	states map[string]*textAreaState
	active map[string]bool // tracks which IDs were accessed this frame
}

type textAreaState struct {
	scrollY    int
	cursorLine int
	selection  TextSelection

	// Mouse selection state
	isDragging        bool
	contentBounds     image.Rectangle  // content area bounds (excludes border)
	lineNumberWidth   int
	cachedLines       []string         // cached for mouse handlers
	mouseHandler      *MouseHandler
	selectionEnabled  bool
	externalSelection *TextSelection   // pointer to externally bound selection (if any)
}

// getActiveSelection returns the selection to use (external if bound, internal otherwise).
func (s *textAreaState) getActiveSelection() *TextSelection {
	if s.externalSelection != nil {
		return s.externalSelection
	}
	return &s.selection
}

// setSelectionStart sets the start of selection on the active selection.
func (s *textAreaState) setSelectionStart(pos TextPosition) {
	sel := s.getActiveSelection()
	sel.SetStart(pos)
}

// setSelectionEnd sets the end of selection on the active selection.
func (s *textAreaState) setSelectionEnd(pos TextPosition) {
	sel := s.getActiveSelection()
	sel.SetEnd(pos)
}

// setFullSelection sets the complete selection on the active selection.
func (s *textAreaState) setFullSelection(sel TextSelection) {
	if s.externalSelection != nil {
		*s.externalSelection = sel
	} else {
		s.selection = sel
	}
}

// clearActiveSelection clears the active selection.
func (s *textAreaState) clearActiveSelection() {
	sel := s.getActiveSelection()
	sel.Clear()
}

// Clear marks all entries as inactive. Called at the start of each frame.
// Entries accessed during the frame via Get() are marked active.
// Call Prune() after the frame to remove entries that weren't accessed.
func (r *textAreaRegistryImpl) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Reset active tracking for new frame
	r.active = make(map[string]bool)
}

// Prune removes entries that weren't accessed since the last Clear().
// This prevents unbounded growth from dynamic TextArea IDs.
func (r *textAreaRegistryImpl) Prune() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id := range r.states {
		if !r.active[id] {
			delete(r.states, id)
		}
	}
}

func (r *textAreaRegistryImpl) Get(id string) *textAreaState {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.active[id] = true // mark as accessed this frame

	if state, exists := r.states[id]; exists {
		return state
	}

	// Create new default state
	newState := &textAreaState{}
	r.states[id] = newState
	return newState
}

// textAreaView is a high-level component for displaying scrollable text content
// with automatic focus-aware styling and keyboard scroll handling.
type textAreaView struct {
	// Content configuration
	id       string
	binding  *string // pointer to external string (optional)
	content  string  // static content (used if binding is nil)
	scrollY  *int    // external scroll position (optional, managed internally if nil)
	internal int     // internal scroll position if scrollY is nil

	// Dimensions
	width  int
	height int

	// Border configuration
	bordered        bool
	border          *BorderStyle
	borderFg        Color
	focusBorderFg   Color
	hasFocusBorder  bool
	title           string
	titleStyle      Style
	focusTitleStyle *Style
	leftBorderOnly  bool

	// Text styling
	textStyle        Style
	emptyPlaceholder string
	emptyStyle       Style

	// Line numbers
	showLineNumbers bool
	lineNumberStyle Style
	lineNumberFg    Color
	hasLineNumberFg bool

	// Current line highlighting
	highlightCurrentLine bool
	currentLineStyle     Style
	hasCurrentLineStyle  bool
	cursorLine           *int // pointer to external cursor line position
	internalCursorLine   int  // internal cursor line if cursorLine is nil

	// Text selection
	selectionEnabled bool
	selectionStyle   Style
	selection        *TextSelection // external selection (optional)
}

// TextArea creates a scrollable text display component.
// It's focusable, supports keyboard scrolling, and has focus-aware border styling.
//
// Example:
//
//	TextArea(&app.output).
//	    ID("output-view").
//	    Title("Output").
//	    Bordered().
//	    Size(60, 10)
func TextArea(binding *string) *textAreaView {
	id := ""
	if binding != nil {
		id = fmt.Sprintf("textarea_%p", binding)
	}
	return &textAreaView{
		id:               id,
		binding:          binding,
		width:            40,
		height:           10,
		textStyle:        NewStyle().WithForeground(ColorWhite),
		emptyPlaceholder: "(empty)",
		emptyStyle:       NewStyle().WithForeground(ColorBrightBlack),
		titleStyle:       NewStyle().WithForeground(ColorYellow),
		lineNumberStyle:  NewStyle().WithForeground(ColorBrightBlack),
	}
}

// ID sets a specific ID for this text area.
func (t *textAreaView) ID(id string) *textAreaView {
	t.id = id
	return t
}

// Content sets static content (ignored if binding is provided).
func (t *textAreaView) Content(content string) *textAreaView {
	t.content = content
	return t
}

// ScrollY binds the scroll position to an external variable.
func (t *textAreaView) ScrollY(scrollY *int) *textAreaView {
	t.scrollY = scrollY
	return t
}

// Width sets the display width.
func (t *textAreaView) Width(w int) *textAreaView {
	t.width = w
	return t
}

// Height sets the display height.
func (t *textAreaView) Height(h int) *textAreaView {
	t.height = h
	return t
}

// Size sets both width and height.
func (t *textAreaView) Size(w, h int) *textAreaView {
	t.width = w
	t.height = h
	return t
}

// Title sets the title shown in the border.
func (t *textAreaView) Title(title string) *textAreaView {
	t.title = title
	return t
}

// TitleStyle sets the style for the title text when unfocused.
func (t *textAreaView) TitleStyle(s Style) *textAreaView {
	t.titleStyle = s
	return t
}

// FocusTitleStyle sets the style for the title when focused.
func (t *textAreaView) FocusTitleStyle(s Style) *textAreaView {
	t.focusTitleStyle = &s
	return t
}

// TextStyle sets the style for the content text.
func (t *textAreaView) TextStyle(s Style) *textAreaView {
	t.textStyle = s
	return t
}

// EmptyPlaceholder sets the text shown when content is empty.
func (t *textAreaView) EmptyPlaceholder(text string) *textAreaView {
	t.emptyPlaceholder = text
	return t
}

// EmptyStyle sets the style for the empty placeholder.
func (t *textAreaView) EmptyStyle(s Style) *textAreaView {
	t.emptyStyle = s
	return t
}

// Bordered enables a border around the text area.
func (t *textAreaView) Bordered() *textAreaView {
	t.bordered = true
	if t.border == nil {
		t.border = &RoundedBorder
	}
	return t
}

// Border sets the border style (implies Bordered).
func (t *textAreaView) Border(style *BorderStyle) *textAreaView {
	t.bordered = true
	t.border = style
	return t
}

// BorderFg sets the border foreground color.
func (t *textAreaView) BorderFg(c Color) *textAreaView {
	t.borderFg = c
	return t
}

// FocusBorderFg sets the border color when the text area is focused.
func (t *textAreaView) FocusBorderFg(c Color) *textAreaView {
	t.focusBorderFg = c
	t.hasFocusBorder = true
	return t
}

// LeftBorderOnly shows only the left border (no top, right, or bottom borders).
// This creates a minimal left-side accent line for the text area.
func (t *textAreaView) LeftBorderOnly() *textAreaView {
	t.leftBorderOnly = true
	t.bordered = true
	if t.border == nil {
		t.border = &RoundedBorder
	}
	return t
}

// LineNumbers enables line numbers on the left side of the text area.
func (t *textAreaView) LineNumbers(show bool) *textAreaView {
	t.showLineNumbers = show
	return t
}

// LineNumberStyle sets the style for line numbers.
func (t *textAreaView) LineNumberStyle(s Style) *textAreaView {
	t.lineNumberStyle = s
	return t
}

// LineNumberFg sets the foreground color for line numbers.
func (t *textAreaView) LineNumberFg(c Color) *textAreaView {
	t.lineNumberFg = c
	t.hasLineNumberFg = true
	return t
}

// HighlightCurrentLine enables highlighting of the current line where the cursor is.
func (t *textAreaView) HighlightCurrentLine(highlight bool) *textAreaView {
	t.highlightCurrentLine = highlight
	return t
}

// CurrentLineStyle sets the style for the highlighted current line.
func (t *textAreaView) CurrentLineStyle(s Style) *textAreaView {
	t.currentLineStyle = s
	t.hasCurrentLineStyle = true
	return t
}

// CursorLine binds the cursor line position to an external variable.
func (t *textAreaView) CursorLine(line *int) *textAreaView {
	t.cursorLine = line
	return t
}

// EnableSelection enables text selection with mouse drag.
func (t *textAreaView) EnableSelection() *textAreaView {
	t.selectionEnabled = true
	if t.selectionStyle == (Style{}) {
		t.selectionStyle = SelectionStyle
	}
	return t
}

// SelectionStyle sets the style for selected text.
func (t *textAreaView) SelectionStyleOpt(s Style) *textAreaView {
	t.selectionStyle = s
	return t
}

// Selection binds the selection to an external variable.
func (t *textAreaView) Selection(sel *TextSelection) *textAreaView {
	t.selection = sel
	return t
}

func (t *textAreaView) getContent() string {
	if t.binding != nil {
		return *t.binding
	}
	return t.content
}

func (t *textAreaView) getScrollY() int {
	if t.scrollY != nil {
		return *t.scrollY
	}
	if t.id != "" {
		return textAreaRegistry.Get(t.id).scrollY
	}
	return t.internal
}

func (t *textAreaView) setScrollY(y int) {
	if t.scrollY != nil {
		*t.scrollY = y
	} else if t.id != "" {
		textAreaRegistry.Get(t.id).scrollY = y
	} else {
		t.internal = y
	}
}

func (t *textAreaView) getCursorLine() int {
	if t.cursorLine != nil {
		return *t.cursorLine
	}
	if t.id != "" {
		return textAreaRegistry.Get(t.id).cursorLine
	}
	return t.internalCursorLine
}

func (t *textAreaView) setCursorLine(line int) {
	if t.cursorLine != nil {
		*t.cursorLine = line
	} else if t.id != "" {
		textAreaRegistry.Get(t.id).cursorLine = line
	} else {
		t.internalCursorLine = line
	}
}

func (t *textAreaView) getSelection() *TextSelection {
	if t.selection != nil {
		return t.selection
	}
	if t.id != "" {
		return &textAreaRegistry.Get(t.id).selection
	}
	return nil
}

func (t *textAreaView) setSelection(sel TextSelection) {
	if t.selection != nil {
		*t.selection = sel
	} else if t.id != "" {
		textAreaRegistry.Get(t.id).selection = sel
	}
}

func (t *textAreaView) clearSelection() {
	if t.selection != nil {
		t.selection.Clear()
	} else if t.id != "" {
		textAreaRegistry.Get(t.id).selection.Clear()
	}
}

// buildLineView creates a view for a single line, handling selection highlighting.
func (t *textAreaView) buildLineView(line string, lineNum int, sel *TextSelection, baseStyle Style) View {
	if line == "" {
		// For empty lines, still show selection if the entire line is selected
		if sel != nil && sel.Active && sel.ContainsLine(lineNum) {
			return Text(" ").Style(t.selectionStyle)
		}
		return Text(" ") // preserve empty lines
	}

	// Check if any part of this line is selected
	if sel == nil || !sel.Active || !sel.ContainsLine(lineNum) {
		// No selection on this line
		return Text("%s", line).Style(baseStyle)
	}

	// Get the selected range within this line
	startCol, endCol := sel.LineRange(lineNum, len(line))

	// Build segmented view
	var segments []View

	// Text before selection
	if startCol > 0 {
		segments = append(segments, Text("%s", line[:startCol]).Style(baseStyle))
	}

	// Selected text
	if startCol < endCol {
		selectedText := line[startCol:endCol]
		segments = append(segments, Text("%s", selectedText).Style(t.selectionStyle))
	}

	// Text after selection
	if endCol < len(line) {
		segments = append(segments, Text("%s", line[endCol:]).Style(baseStyle))
	}

	if len(segments) == 1 {
		return segments[0]
	}
	return Group(segments...)
}

func (t *textAreaView) size(maxWidth, maxHeight int) (int, int) {
	w := t.width
	h := t.height
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

// lineNumberWidth calculates the width needed for line numbers.
func (t *textAreaView) lineNumberWidth() int {
	if !t.showLineNumbers {
		return 0
	}
	content := t.getContent()
	lines := strings.Split(content, "\n")
	maxLine := len(lines)

	// Calculate width needed for the largest line number
	width := 1
	for maxLine >= 10 {
		maxLine /= 10
		width++
	}
	return width + 1 // number + space
}

func (t *textAreaView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Determine if focused
	isFocused := t.id != "" && focusManager.GetFocusedID() == t.id

	// Build content view
	content := t.getContent()
	var contentView View
	if content == "" {
		contentView = Text("%s", t.emptyPlaceholder).Style(t.emptyStyle)
	} else {
		lines := strings.Split(content, "\n")
		lineViews := make([]View, len(lines))
		lnWidth := t.lineNumberWidth()
		cursorLine := t.getCursorLine()

		// Get selection state if enabled
		var sel *TextSelection
		if t.selectionEnabled {
			sel = t.getSelection()
		}

		// Determine line number style
		lnStyle := t.lineNumberStyle
		if t.hasLineNumberFg {
			lnStyle = lnStyle.WithForeground(t.lineNumberFg)
		}

		// Determine current line highlight style
		currentLineStyle := t.currentLineStyle
		if !t.hasCurrentLineStyle {
			// Default to a subtle background highlight
			currentLineStyle = NewStyle().WithBackground(ColorBrightBlack)
		}

		for i, line := range lines {
			var lineView View

			// Determine base text style
			lineStyle := t.textStyle
			if t.highlightCurrentLine && i == cursorLine {
				lineStyle = lineStyle.Merge(currentLineStyle)
			}

			// Build the line content with optional line number
			if t.showLineNumbers {
				lineNum := i + 1
				lineNumText := fmt.Sprintf("%*d ", lnWidth-1, lineNum)
				lineNumStyle := lnStyle
				if t.highlightCurrentLine && i == cursorLine {
					lineNumStyle = lineNumStyle.Merge(currentLineStyle)
				}
				lineNumView := Text("%s", lineNumText).Style(lineNumStyle)

				// Build text view with selection support
				textView := t.buildLineView(line, i, sel, lineStyle)
				lineView = Group(lineNumView, textView)
			} else {
				// No line numbers, build text view with selection support
				lineView = t.buildLineView(line, i, sel, lineStyle)
			}

			lineViews[i] = lineView
		}
		contentView = Stack(lineViews...)
	}

	// Build the scrollable content
	scrollY := t.getScrollY()
	scrollContent := Scroll(contentView, &scrollY)

	if t.bordered && t.border != nil {
		// Determine border style
		borderStyle := NewStyle()
		if isFocused {
			if t.hasFocusBorder {
				borderStyle = borderStyle.WithForeground(t.focusBorderFg)
			} else {
				borderStyle = borderStyle.WithForeground(ColorCyan)
			}
		} else if t.borderFg != 0 {
			borderStyle = borderStyle.WithForeground(t.borderFg)
		}

		// Determine title style
		titleStyle := t.titleStyle
		if isFocused {
			if t.focusTitleStyle != nil {
				titleStyle = *t.focusTitleStyle
			} else {
				titleStyle = NewStyle().WithForeground(ColorCyan).WithBold()
			}
		}

		// Render bordered view manually for focus-aware styling
		t.renderBordered(ctx, w, h, scrollContent, &scrollY, borderStyle, titleStyle)
	} else {
		// No border, just render the scroll content
		scrollContent.render(ctx)
	}

	// Update scroll position
	t.setScrollY(scrollY)

	// Register as focusable for Tab navigation
	bounds := ctx.AbsoluteBounds()
	handler := &textAreaFocusHandler{
		area:   t,
		bounds: bounds,
	}
	focusManager.Register(handler)

	// Register mouse region for selection if enabled
	if t.selectionEnabled && t.id != "" {
		state := textAreaRegistry.Get(t.id)
		state.selectionEnabled = true
		state.cachedLines = strings.Split(content, "\n")
		state.lineNumberWidth = t.lineNumberWidth()
		state.externalSelection = t.selection // sync external binding pointer

		// Calculate content bounds (inside border if present)
		contentBounds := bounds
		if t.bordered && !t.leftBorderOnly {
			contentBounds = image.Rect(
				bounds.Min.X+1,
				bounds.Min.Y+1,
				bounds.Max.X-1,
				bounds.Max.Y-1,
			)
		} else if t.leftBorderOnly {
			contentBounds = image.Rect(
				bounds.Min.X+1,
				bounds.Min.Y,
				bounds.Max.X,
				bounds.Max.Y,
			)
		}
		state.contentBounds = contentBounds

		t.registerMouseRegion(state, contentBounds)
	}
}

// registerMouseRegion sets up mouse handling for text selection.
func (t *textAreaView) registerMouseRegion(state *textAreaState, bounds image.Rectangle) {
	// Create mouse handler if needed
	if state.mouseHandler == nil {
		state.mouseHandler = NewMouseHandler()
	}

	// Clear and re-register the region each frame
	state.mouseHandler.ClearRegions()

	region := &MouseRegion{
		X:           bounds.Min.X,
		Y:           bounds.Min.Y,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		CursorStyle: CursorText,
		Label:       "textarea-selection",

		OnPress: func(e *MouseEvent) {
			// Start selection on press
			pos := t.screenToTextPos(state, e.X, e.Y)
			state.setSelectionStart(pos)
			state.isDragging = true
		},

		OnDragStart: func(e *MouseEvent) {
			// Selection drag started
			state.isDragging = true
		},

		OnDrag: func(e *MouseEvent) {
			if state.isDragging {
				// Update selection end during drag
				pos := t.screenToTextPos(state, e.X, e.Y)
				state.setSelectionEnd(pos)
			}
		},

		OnDragEnd: func(e *MouseEvent) {
			// Finalize selection
			pos := t.screenToTextPos(state, e.X, e.Y)
			state.setSelectionEnd(pos)
			state.isDragging = false

			// Copy-on-select: automatically copy to clipboard when selection completes
			sel := state.getActiveSelection()
			if sel != nil && sel.Active && !sel.IsEmpty() {
				CopySelectionToClipboard(state.cachedLines, *sel)
			}
		},

		OnDoubleClick: func(e *MouseEvent) {
			// Select word
			pos := t.screenToTextPos(state, e.X, e.Y)
			sel := SelectWord(state.cachedLines, pos)
			state.setFullSelection(sel)
			// Copy-on-select
			if sel.Active && !sel.IsEmpty() {
				CopySelectionToClipboard(state.cachedLines, sel)
			}
		},

		OnTripleClick: func(e *MouseEvent) {
			// Select line
			pos := t.screenToTextPos(state, e.X, e.Y)
			sel := SelectLine(state.cachedLines, pos.Line)
			state.setFullSelection(sel)
			// Copy-on-select
			if sel.Active && !sel.IsEmpty() {
				CopySelectionToClipboard(state.cachedLines, sel)
			}
		},

		OnClick: func(e *MouseEvent) {
			// Single click clears selection and sets cursor
			if e.ClickCount == 1 {
				state.clearActiveSelection()
			}
		},
	}

	state.mouseHandler.AddRegion(region)
}

// screenToTextPos converts screen coordinates to text position.
func (t *textAreaView) screenToTextPos(state *textAreaState, screenX, screenY int) TextPosition {
	// Convert to content-relative coordinates
	relX := screenX - state.contentBounds.Min.X
	relY := screenY - state.contentBounds.Min.Y

	return ScreenToTextPosition(
		relX,
		relY,
		state.scrollY,
		state.lineNumberWidth,
		state.cachedLines,
	)
}

// HandleMouseEvent processes mouse events for a TextArea with the given ID.
// This should be called from the application's event handler.
func TextAreaHandleMouseEvent(id string, event *MouseEvent) bool {
	state := textAreaRegistry.Get(id)
	if state == nil || !state.selectionEnabled || state.mouseHandler == nil {
		return false
	}

	// Check if event is within our bounds
	if event.X < state.contentBounds.Min.X || event.X >= state.contentBounds.Max.X ||
		event.Y < state.contentBounds.Min.Y || event.Y >= state.contentBounds.Max.Y {
		// For drag events, still process if we started the drag
		if event.Type != MouseDrag && event.Type != MouseDragEnd {
			return false
		}
		if !state.isDragging {
			return false
		}
	}

	state.mouseHandler.HandleEvent(event)
	return true
}

func (t *textAreaView) renderBordered(ctx *RenderContext, w, h int, content *scrollView, scrollY *int, borderStyle, titleStyle Style) {
	border := t.border

	if t.leftBorderOnly {
		// Only draw the left border
		for y := 0; y < h; y++ {
			ctx.PrintTruncated(0, y, border.Vertical, borderStyle)
		}

		// Inner content area (offset by 1 for left border)
		innerBounds := image.Rect(1, 0, w, h)
		if innerBounds.Dx() > 0 && innerBounds.Dy() > 0 {
			innerCtx := ctx.SubContext(innerBounds)
			content.render(innerCtx)
		}
		return
	}

	// Draw full border (original behavior)
	// Draw top border with title
	ctx.PrintTruncated(0, 0, border.TopLeft, borderStyle)
	bx := 1

	if t.title != "" && w > 4 {
		ctx.PrintTruncated(bx, 0, border.Horizontal, borderStyle)
		bx++
		titleText := " " + t.title + " "
		titleW, _ := MeasureText(titleText)
		maxTitleW := w - 4
		if titleW > maxTitleW {
			titleW = maxTitleW
		}
		ctx.PrintTruncated(bx, 0, titleText, titleStyle)
		bx += titleW
	}

	for ; bx < w-1; bx++ {
		ctx.PrintTruncated(bx, 0, border.Horizontal, borderStyle)
	}
	if w > 1 {
		ctx.PrintTruncated(w-1, 0, border.TopRight, borderStyle)
	}

	// Side borders
	for y := 1; y < h-1; y++ {
		ctx.PrintTruncated(0, y, border.Vertical, borderStyle)
		if w > 1 {
			ctx.PrintTruncated(w-1, y, border.Vertical, borderStyle)
		}
	}

	// Bottom border
	if h > 1 {
		ctx.PrintTruncated(0, h-1, border.BottomLeft, borderStyle)
		for bx := 1; bx < w-1; bx++ {
			ctx.PrintTruncated(bx, h-1, border.Horizontal, borderStyle)
		}
		if w > 1 {
			ctx.PrintTruncated(w-1, h-1, border.BottomRight, borderStyle)
		}
	}

	// Inner content area
	innerBounds := image.Rect(1, 1, w-1, h-1)
	if innerBounds.Dx() > 0 && innerBounds.Dy() > 0 {
		innerCtx := ctx.SubContext(innerBounds)
		content.render(innerCtx)
	}
}

// textAreaFocusHandler implements Focusable for TextArea
type textAreaFocusHandler struct {
	area    *textAreaView
	bounds  image.Rectangle
	focused bool
}

func (h *textAreaFocusHandler) FocusID() string {
	return h.area.id
}

func (h *textAreaFocusHandler) IsFocused() bool {
	return h.focused
}

func (h *textAreaFocusHandler) SetFocused(focused bool) {
	h.focused = focused
}

func (h *textAreaFocusHandler) FocusBounds() image.Rectangle {
	return h.bounds
}

func (h *textAreaFocusHandler) HandleKeyEvent(event KeyEvent) bool {
	scrollY := h.area.getScrollY()
	cursorLine := h.area.getCursorLine()
	content := h.area.getContent()
	lines := strings.Split(content, "\n")
	maxLine := len(lines) - 1
	handled := false

	// Handle Ctrl+C for copy if selection is enabled and there's an active selection
	if event.Key == KeyCtrlC && h.area.selectionEnabled {
		if sel := h.area.getSelection(); sel != nil && sel.Active && !sel.IsEmpty() {
			CopySelectionToClipboard(lines, *sel)
			return true
		}
		// No active selection - don't consume, let app handle Ctrl+C (e.g., for quit)
	}

	// Handle Escape to clear selection
	if event.Key == KeyEscape && h.area.selectionEnabled {
		h.area.clearSelection()
		return true
	}

	switch event.Key {
	case KeyArrowUp:
		// Move cursor line up if current line highlighting is enabled
		if h.area.highlightCurrentLine && cursorLine > 0 {
			cursorLine--
			handled = true
			// Auto-scroll if needed
			if cursorLine < scrollY {
				scrollY = cursorLine
			}
		} else if scrollY > 0 {
			scrollY--
			handled = true
		}
	case KeyArrowDown:
		// Move cursor line down if current line highlighting is enabled
		if h.area.highlightCurrentLine && cursorLine < maxLine {
			cursorLine++
			handled = true
			// Auto-scroll if needed
			_, viewHeight := h.area.size(0, 0)
			if h.area.bordered && !h.area.leftBorderOnly {
				viewHeight -= 2 // account for border
			}
			if cursorLine >= scrollY+viewHeight {
				scrollY = cursorLine - viewHeight + 1
			}
		} else {
			scrollY++
			handled = true
		}
	case KeyPageUp:
		if h.area.highlightCurrentLine {
			cursorLine -= 5
			if cursorLine < 0 {
				cursorLine = 0
			}
			scrollY = cursorLine
		} else {
			scrollY -= 5
		}
		if scrollY < 0 {
			scrollY = 0
		}
		handled = true
	case KeyPageDown:
		if h.area.highlightCurrentLine {
			cursorLine += 5
			if cursorLine > maxLine {
				cursorLine = maxLine
			}
			scrollY = cursorLine
		} else {
			scrollY += 5
		}
		handled = true
	case KeyHome:
		if h.area.highlightCurrentLine {
			cursorLine = 0
		}
		scrollY = 0
		handled = true
	case KeyEnd:
		if h.area.highlightCurrentLine {
			cursorLine = maxLine
			// Scroll to the bottom
			_, viewHeight := h.area.size(0, 0)
			if h.area.bordered && !h.area.leftBorderOnly {
				viewHeight -= 2 // account for border
			}
			scrollY = maxLine - viewHeight + 1
			if scrollY < 0 {
				scrollY = 0
			}
		}
		handled = true
	}

	if handled {
		h.area.setScrollY(scrollY)
		if h.area.highlightCurrentLine {
			h.area.setCursorLine(cursorLine)
		}
	}
	return handled
}
