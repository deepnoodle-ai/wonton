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
}

type textAreaRegistryImpl struct {
	mu     sync.Mutex
	states map[string]*textAreaState
}

type textAreaState struct {
	scrollY    int
	cursorLine int
}

func (r *textAreaRegistryImpl) Clear() {
	// We do NOT clear the map here because we want state to persist across frames.
	// However, to prevent memory leaks from unused IDs, we might want a mechanism to prune.
	// For now, we follow the pattern of persisting state.
	// TODO: Implement garbage collection for stale IDs?
}

func (r *textAreaRegistryImpl) Get(id string) *textAreaState {
	r.mu.Lock()
	defer r.mu.Unlock()

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

			// Build the line content with optional line number
			if t.showLineNumbers {
				lineNum := i + 1
				lineNumText := fmt.Sprintf("%*d ", lnWidth-1, lineNum)
				lineNumView := Text("%s", lineNumText).Style(lnStyle)

				var textView View
				if line == "" {
					textView = Text(" ") // preserve empty lines
				} else {
					textView = Text("%s", line).Style(t.textStyle)
				}

				// Apply current line highlighting if enabled
				if t.highlightCurrentLine && i == cursorLine {
					textView = Text("%s", line).Style(t.textStyle.Merge(currentLineStyle))
					lineNumView = Text("%s", lineNumText).Style(lnStyle.Merge(currentLineStyle))
				}

				lineView = Group(lineNumView, textView)
			} else {
				// No line numbers, just the text
				if line == "" {
					lineView = Text(" ") // preserve empty lines
				} else {
					lineStyle := t.textStyle
					// Apply current line highlighting if enabled
					if t.highlightCurrentLine && i == cursorLine {
						lineStyle = lineStyle.Merge(currentLineStyle)
					}
					lineView = Text("%s", line).Style(lineStyle)
				}
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
