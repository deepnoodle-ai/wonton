package gooey

import (
	"fmt"
	"image"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// inputSegment represents a portion of input text
type inputSegment struct {
	display string // What is shown to the user
	actual  string // The actual value (same as display for typed text, full content for pastes)
	isPaste bool   // True if this is a paste placeholder
}

// TextInput is a simple single-line text input widget
type TextInput struct {
	BaseWidget
	Placeholder string
	CursorPos   int // Cursor position in display text

	// Styles
	Style            Style
	PlaceholderStyle Style
	CursorStyle      Style
	PasteStyle       Style // Style for paste placeholders

	// Callbacks
	OnChange func(value string)
	OnSubmit func(value string)

	// Password/masking support
	MaskChar  rune // If non-zero, display this character instead of actual text
	MaxLength int  // If non-zero, limit input to this many runes

	// Paste placeholder mode
	PastePlaceholderMode bool // When true, multi-line pastes show as "[pasted N lines]"

	// Multiline mode
	MultilineMode  bool // When true, Shift+Enter inserts newlines
	SubmitOnEnter   bool   // When true, Enter triggers OnSubmit (default: true)

	// Internal
	focused  bool
	segments []inputSegment // Segments of typed text and paste placeholders
}

// NewTextInput creates a new text input widget
func NewTextInput() *TextInput {
	t := &TextInput{
		BaseWidget:       NewBaseWidget(),
		CursorPos:        0,
		Style:            NewStyle().WithForeground(ColorWhite),
		PlaceholderStyle: NewStyle().WithForeground(ColorBrightBlack),
		CursorStyle:      NewStyle().WithBackground(ColorWhite).WithForeground(ColorBlack),
		PasteStyle:       NewStyle().WithForeground(ColorBrightBlack).WithItalic(),
		segments:      []inputSegment{},
		SubmitOnEnter: true,
	}
	t.SetMinSize(image.Point{X: 10, Y: 1})
	return t
}

// Value returns the actual value (with full paste content, not placeholders)
func (t *TextInput) Value() string {
	var result string
	for _, seg := range t.segments {
		result += seg.actual
	}
	return result
}

// SetValue sets the input value, clearing any segments
func (t *TextInput) SetValue(value string) {
	if value == "" {
		t.segments = []inputSegment{}
	} else {
		t.segments = []inputSegment{{display: value, actual: value, isPaste: false}}
	}
	t.CursorPos = len(value)
	t.MarkDirty()
}

// DisplayText returns the text shown to the user (with placeholders for pastes)
func (t *TextInput) DisplayText() string {
	var result string
	for _, seg := range t.segments {
		result += seg.display
	}
	return result
}

// displayLen returns the total length of display text
func (t *TextInput) displayLen() int {
	total := 0
	for _, seg := range t.segments {
		total += len(seg.display)
	}
	return total
}

// WithPastePlaceholderMode enables/disables paste placeholder mode
func (t *TextInput) WithPastePlaceholderMode(enabled bool) *TextInput {
	t.PastePlaceholderMode = enabled
	return t
}

// WithMultilineMode enables multiline input where Shift+Enter inserts newlines.
// Newlines are displayed as the NewlineDisplay string (default "↵").
func (t *TextInput) WithMultilineMode(enabled bool) *TextInput {
	t.MultilineMode = enabled
	return t
}

// WithSubmitOnEnter sets whether Enter triggers OnSubmit.
// Default is true. Set to false if you want Enter to do nothing (useful with MultilineMode).
func (t *TextInput) WithSubmitOnEnter(enabled bool) *TextInput {
	t.SubmitOnEnter = enabled
	return t
}

// WithMask sets a mask character for password input.
// When set, all characters are displayed as this character instead of the actual text.
func (t *TextInput) WithMask(char rune) *TextInput {
	t.MaskChar = char
	return t
}

// WithMaxLength sets the maximum number of runes allowed in the input.
func (t *TextInput) WithMaxLength(n int) *TextInput {
	t.MaxLength = n
	return t
}

// WithPlaceholder sets the placeholder text shown when input is empty.
func (t *TextInput) WithPlaceholder(placeholder string) *TextInput {
	t.Placeholder = placeholder
	return t
}

// WithStyle sets the style for the input text.
func (t *TextInput) WithStyle(style Style) *TextInput {
	t.Style = style
	return t
}

// Draw renders the input
func (t *TextInput) Draw(frame RenderFrame) {
	bounds := t.GetBounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if height <= 0 {
		return
	}

	// Determine if we are in a SubFrame
	frameW, frameH := frame.Size()
	drawX, drawY := bounds.Min.X, bounds.Min.Y
	if frameW == width && frameH == height {
		drawX, drawY = 0, 0
	}

	// Clear background
	frame.FillStyled(drawX, drawY, width, height, ' ', t.Style)

	displayText := t.DisplayText()
	showingPlaceholder := displayText == "" && t.Placeholder != ""

	if showingPlaceholder {
		// Show placeholder text
		placeholderText := t.Placeholder
		if runewidth.StringWidth(placeholderText) > width {
			placeholderText = runewidth.Truncate(placeholderText, width, "…")
		}
		frame.PrintStyled(drawX, drawY, placeholderText, t.PlaceholderStyle)
	} else if t.MaskChar != 0 && displayText != "" {
		// Mask the text for password input (single line only)
		masked := make([]rune, utf8.RuneCountInString(displayText))
		for i := range masked {
			masked[i] = t.MaskChar
		}
		maskedText := string(masked)
		if runewidth.StringWidth(maskedText) > width {
			maskedText = runewidth.Truncate(maskedText, width, "…")
		}
		frame.PrintStyled(drawX, drawY, maskedText, t.Style)
	} else {
		// Draw segments with appropriate styles, handling newlines
		x := drawX
		y := drawY
		for _, seg := range t.segments {
			style := t.Style
			if seg.isPaste {
				style = t.PasteStyle
			}

			// Handle segment character by character to deal with newlines
			for _, r := range seg.display {
				if r == '\n' {
					// Move to next line
					y++
					x = drawX
					if y >= drawY+height {
						break
					}
					continue
				}

				charWidth := runewidth.RuneWidth(r)
				if x+charWidth > drawX+width {
					// Would overflow line - truncate
					break
				}

				frame.PrintStyled(x, y, string(r), style)
				x += charWidth
			}

			if y >= drawY+height {
				break
			}
		}
	}

	// Draw cursor if focused
	if t.focused {
		// Calculate cursor position accounting for newlines
		cursorX, cursorY := t.getCursorXY(drawX, drawY, width)

		if cursorY < drawY+height && cursorX < drawX+width {
			charUnderCursor := " "
			if showingPlaceholder {
				// Show first char of placeholder under cursor
				r, _ := utf8.DecodeRuneInString(t.Placeholder)
				charUnderCursor = string(r)
			} else if t.CursorPos < len(displayText) {
				r, _ := utf8.DecodeRuneInString(displayText[t.CursorPos:])
				if r != '\n' {
					if t.MaskChar != 0 {
						charUnderCursor = string(t.MaskChar)
					} else {
						charUnderCursor = string(r)
					}
				}
			}
			frame.PrintStyled(cursorX, cursorY, charUnderCursor, t.CursorStyle)
		}
	}
}

// getCursorXY calculates the visual x,y position of the cursor
func (t *TextInput) getCursorXY(startX, startY, width int) (x, y int) {
	displayText := t.DisplayText()
	x = startX
	y = startY

	for i, r := range displayText {
		if i >= t.CursorPos {
			break
		}
		if r == '\n' {
			y++
			x = startX
		} else {
			x += runewidth.RuneWidth(r)
			if x >= startX+width {
				// Wrap would occur - for now just stop at edge
				x = startX + width - 1
			}
		}
	}
	return x, y
}

// findSegmentAtPos returns the segment index and offset within segment for a display position
func (t *TextInput) findSegmentAtPos(pos int) (segIndex int, offset int) {
	remaining := pos
	for i, seg := range t.segments {
		if remaining <= len(seg.display) {
			return i, remaining
		}
		remaining -= len(seg.display)
	}
	// Past end - return last segment
	if len(t.segments) == 0 {
		return 0, 0
	}
	lastIdx := len(t.segments) - 1
	return lastIdx, len(t.segments[lastIdx].display)
}

// insertAtCursor inserts text at the current cursor position
func (t *TextInput) insertAtCursor(text string) {
	t.insertSegmentAtCursor(inputSegment{display: text, actual: text, isPaste: false})
}

// insertNewline inserts a newline at cursor position (for multiline mode)
func (t *TextInput) insertNewline() {
	t.insertSegmentAtCursor(inputSegment{display: "\n", actual: "\n", isPaste: false})
}

// insertSegmentAtCursor inserts a segment at the current cursor position
func (t *TextInput) insertSegmentAtCursor(newSeg inputSegment) {
	if len(t.segments) == 0 {
		t.segments = []inputSegment{newSeg}
		t.CursorPos = len(newSeg.display)
		return
	}

	segIdx, offset := t.findSegmentAtPos(t.CursorPos)
	seg := t.segments[segIdx]

	// Check if this is a "special" segment (display != actual, like newlines or pastes)
	isSpecialSeg := newSeg.display != newSeg.actual || newSeg.isPaste

	if seg.isPaste || seg.display != seg.actual {
		// Current segment is special (paste or newline) - insert new segment adjacent
		insertIdx := segIdx
		if offset > 0 {
			insertIdx = segIdx + 1
		}
		t.segments = append(t.segments[:insertIdx], append([]inputSegment{newSeg}, t.segments[insertIdx:]...)...)
	} else if isSpecialSeg {
		// New segment is special - need to split current text segment
		if offset == 0 {
			// Insert before
			t.segments = append(t.segments[:segIdx], append([]inputSegment{newSeg}, t.segments[segIdx:]...)...)
		} else if offset >= len(seg.display) {
			// Insert after
			t.segments = append(t.segments[:segIdx+1], append([]inputSegment{newSeg}, t.segments[segIdx+1:]...)...)
		} else {
			// Split text segment
			before := inputSegment{display: seg.display[:offset], actual: seg.actual[:offset], isPaste: false}
			after := inputSegment{display: seg.display[offset:], actual: seg.actual[offset:], isPaste: false}

			newSegments := append([]inputSegment{}, t.segments[:segIdx]...)
			if before.display != "" {
				newSegments = append(newSegments, before)
			}
			newSegments = append(newSegments, newSeg)
			if after.display != "" {
				newSegments = append(newSegments, after)
			}
			newSegments = append(newSegments, t.segments[segIdx+1:]...)
			t.segments = newSegments
		}
	} else {
		// Both are regular text - merge into current segment
		before := seg.display[:offset]
		after := seg.display[offset:]
		seg.display = before + newSeg.display + after
		seg.actual = before + newSeg.actual + after
		t.segments[segIdx] = seg
	}

	t.CursorPos += len(newSeg.display)
	t.mergeAdjacentTextSegments()
}

// isSpecialSegment returns true if a segment should be deleted atomically
// (pastes, newlines, or any segment where display differs from actual)
func (seg *inputSegment) isSpecial() bool {
	return seg.isPaste || seg.display != seg.actual
}

// deleteBackward deletes the character/segment before cursor
func (t *TextInput) deleteBackward() bool {
	if t.CursorPos == 0 || len(t.segments) == 0 {
		return false
	}

	segIdx, offset := t.findSegmentAtPos(t.CursorPos)
	seg := t.segments[segIdx]

	// Check if we're at the start of a segment and need to look at previous
	if offset == 0 && segIdx > 0 {
		segIdx--
		seg = t.segments[segIdx]
		offset = len(seg.display)
	}

	if seg.isSpecial() {
		// Delete entire special segment atomically (paste or newline)
		deletedDisplayLen := len(seg.display)
		t.segments = append(t.segments[:segIdx], t.segments[segIdx+1:]...)
		t.CursorPos -= deletedDisplayLen
	} else {
		// Delete single character from text segment
		if offset > 0 {
			_, w := utf8.DecodeLastRuneInString(seg.display[:offset])
			seg.display = seg.display[:offset-w] + seg.display[offset:]
			seg.actual = seg.display
			t.CursorPos -= w

			if seg.display == "" {
				// Remove empty segment
				t.segments = append(t.segments[:segIdx], t.segments[segIdx+1:]...)
			} else {
				t.segments[segIdx] = seg
			}
		}
	}

	t.mergeAdjacentTextSegments()
	return true
}

// deleteForward deletes the character/segment at cursor
func (t *TextInput) deleteForward() bool {
	displayLen := t.displayLen()
	if t.CursorPos >= displayLen || len(t.segments) == 0 {
		return false
	}

	segIdx, offset := t.findSegmentAtPos(t.CursorPos)
	seg := t.segments[segIdx]

	if seg.isSpecial() {
		// Delete entire special segment atomically (paste or newline)
		t.segments = append(t.segments[:segIdx], t.segments[segIdx+1:]...)
	} else {
		// Delete single character from text segment
		if offset < len(seg.display) {
			_, w := utf8.DecodeRuneInString(seg.display[offset:])
			seg.display = seg.display[:offset] + seg.display[offset+w:]
			seg.actual = seg.display

			if seg.display == "" {
				// Remove empty segment
				t.segments = append(t.segments[:segIdx], t.segments[segIdx+1:]...)
			} else {
				t.segments[segIdx] = seg
			}
		}
	}

	t.mergeAdjacentTextSegments()
	return true
}

// mergeAdjacentTextSegments combines adjacent regular text segments
// (not special segments like pastes or newlines)
func (t *TextInput) mergeAdjacentTextSegments() {
	if len(t.segments) < 2 {
		return
	}

	merged := []inputSegment{t.segments[0]}
	for i := 1; i < len(t.segments); i++ {
		last := &merged[len(merged)-1]
		curr := t.segments[i]

		if !last.isSpecial() && !curr.isSpecial() {
			// Merge regular text segments
			last.display += curr.display
			last.actual += curr.actual
		} else {
			merged = append(merged, curr)
		}
	}
	t.segments = merged
}

// HandleKey handles key events
func (t *TextInput) HandleKey(event KeyEvent) bool {
	if !t.focused {
		return false
	}

	displayText := t.DisplayText()

	switch event.Key {
	case KeyArrowLeft:
		if t.CursorPos > 0 {
			_, w := utf8.DecodeLastRuneInString(displayText[:t.CursorPos])
			t.CursorPos -= w
			t.MarkDirty()
		}
		return true
	case KeyArrowRight:
		if t.CursorPos < len(displayText) {
			_, w := utf8.DecodeRuneInString(displayText[t.CursorPos:])
			t.CursorPos += w
			t.MarkDirty()
		}
		return true
	case KeyBackspace:
		if t.deleteBackward() {
			if t.OnChange != nil {
				t.OnChange(t.Value())
			}
			t.MarkDirty()
		}
		return true
	case KeyDelete:
		if t.deleteForward() {
			if t.OnChange != nil {
				t.OnChange(t.Value())
			}
			t.MarkDirty()
		}
		return true
	case KeyHome:
		t.CursorPos = 0
		t.MarkDirty()
		return true
	case KeyEnd:
		t.CursorPos = t.displayLen()
		t.MarkDirty()
		return true
	case KeyEnter:
		if event.Shift && t.MultilineMode {
			// Shift+Enter in multiline mode: insert newline
			t.insertNewline()
			if t.OnChange != nil {
				t.OnChange(t.Value())
			}
			t.MarkDirty()
			return true
		}
		if t.SubmitOnEnter && t.OnSubmit != nil {
			t.OnSubmit(t.Value())
		}
		return true
	}

	if event.Rune != 0 && event.Rune >= 32 { // Printable characters
		// Check max length before inserting
		if t.MaxLength > 0 && utf8.RuneCountInString(t.Value()) >= t.MaxLength {
			return true // Consumed but ignored
		}
		t.insertAtCursor(string(event.Rune))
		if t.OnChange != nil {
			t.OnChange(t.Value())
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

// HandlePaste handles pasted content, using placeholder mode if enabled for multi-line pastes.
// Returns true if the paste was handled.
func (t *TextInput) HandlePaste(content string) bool {
	if content == "" {
		return false
	}

	lines := strings.Split(content, "\n")
	lineCount := len(lines)

	// Determine if we should use placeholder
	usePlaceholder := t.PastePlaceholderMode && lineCount > 1

	var newSeg inputSegment
	if usePlaceholder {
		newSeg = inputSegment{
			display: fmt.Sprintf("[pasted %d lines]", lineCount),
			actual:  content,
			isPaste: true,
		}
	} else {
		// Single line or placeholder mode disabled - show actual content
		newSeg = inputSegment{
			display: content,
			actual:  content,
			isPaste: false,
		}
	}

	// Insert the segment at cursor position
	if len(t.segments) == 0 {
		t.segments = []inputSegment{newSeg}
		t.CursorPos = len(newSeg.display)
	} else {
		segIdx, offset := t.findSegmentAtPos(t.CursorPos)
		seg := t.segments[segIdx]

		if seg.isPaste || offset == 0 || offset == len(seg.display) {
			// Insert as new segment (before, after, or between paste segments)
			insertIdx := segIdx
			if offset > 0 {
				insertIdx = segIdx + 1
			}
			t.segments = append(t.segments[:insertIdx], append([]inputSegment{newSeg}, t.segments[insertIdx:]...)...)
		} else {
			// Split text segment
			before := inputSegment{display: seg.display[:offset], actual: seg.actual[:offset], isPaste: false}
			after := inputSegment{display: seg.display[offset:], actual: seg.actual[offset:], isPaste: false}

			newSegments := append([]inputSegment{}, t.segments[:segIdx]...)
			if before.display != "" {
				newSegments = append(newSegments, before)
			}
			newSegments = append(newSegments, newSeg)
			if after.display != "" {
				newSegments = append(newSegments, after)
			}
			newSegments = append(newSegments, t.segments[segIdx+1:]...)
			t.segments = newSegments
		}
		t.CursorPos += len(newSeg.display)
	}

	t.mergeAdjacentTextSegments()

	if t.OnChange != nil {
		t.OnChange(t.Value())
	}
	t.MarkDirty()

	return true
}

// Clear clears all input
func (t *TextInput) Clear() {
	t.segments = []inputSegment{}
	t.CursorPos = 0
	t.MarkDirty()
}
