package tui

import (
	"strings"

	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/mattn/go-runewidth"
)

// TextPosition represents a position in text content.
type TextPosition struct {
	Line int // 0-indexed line number
	Col  int // 0-indexed column (byte offset within line)
}

// Before returns true if p comes before other in document order.
func (p TextPosition) Before(other TextPosition) bool {
	if p.Line < other.Line {
		return true
	}
	if p.Line > other.Line {
		return false
	}
	return p.Col < other.Col
}

// Equal returns true if positions are the same.
func (p TextPosition) Equal(other TextPosition) bool {
	return p.Line == other.Line && p.Col == other.Col
}

// TextSelection represents a text selection range.
// Start and End can be in any order; use Normalized() to get ordered positions.
type TextSelection struct {
	Start  TextPosition
	End    TextPosition
	Active bool // true if a selection is active
}

// Normalized returns the selection with Start before End.
func (s TextSelection) Normalized() (start, end TextPosition) {
	if s.Start.Before(s.End) {
		return s.Start, s.End
	}
	return s.End, s.Start
}

// IsEmpty returns true if no text is selected.
func (s TextSelection) IsEmpty() bool {
	return !s.Active || s.Start.Equal(s.End)
}

// Clear clears the selection.
func (s *TextSelection) Clear() {
	s.Active = false
	s.Start = TextPosition{}
	s.End = TextPosition{}
}

// SetStart sets the start position and activates selection.
func (s *TextSelection) SetStart(pos TextPosition) {
	s.Start = pos
	s.End = pos
	s.Active = true
}

// SetEnd updates the end position of an active selection.
func (s *TextSelection) SetEnd(pos TextPosition) {
	s.End = pos
}

// Contains returns true if the given position is within the selection.
func (s TextSelection) Contains(pos TextPosition) bool {
	if !s.Active {
		return false
	}
	start, end := s.Normalized()

	// Before start
	if pos.Line < start.Line || (pos.Line == start.Line && pos.Col < start.Col) {
		return false
	}
	// After end
	if pos.Line > end.Line || (pos.Line == end.Line && pos.Col >= end.Col) {
		return false
	}
	return true
}

// ContainsLine returns true if any part of the line is selected.
func (s TextSelection) ContainsLine(line int) bool {
	if !s.Active {
		return false
	}
	start, end := s.Normalized()
	return line >= start.Line && line <= end.Line
}

// LineRange returns the column range selected on a given line.
// Returns (0, 0) if the line is not selected.
// If the entire line is selected, returns (0, lineLen).
func (s TextSelection) LineRange(line int, lineLen int) (startCol, endCol int) {
	if !s.Active {
		return 0, 0
	}

	start, end := s.Normalized()

	// Line not in selection
	if line < start.Line || line > end.Line {
		return 0, 0
	}

	// Determine start column for this line
	if line == start.Line {
		startCol = start.Col
	} else {
		startCol = 0
	}

	// Determine end column for this line
	if line == end.Line {
		endCol = end.Col
	} else {
		endCol = lineLen
	}

	return startCol, endCol
}

// ExtractSelectedText extracts the selected text from the given lines.
func ExtractSelectedText(lines []string, sel TextSelection) string {
	if sel.IsEmpty() {
		return ""
	}

	start, end := sel.Normalized()

	// Clamp to valid line range
	if start.Line >= len(lines) {
		return ""
	}
	if end.Line >= len(lines) {
		end.Line = len(lines) - 1
		end.Col = len(lines[end.Line])
	}

	// Single line selection
	if start.Line == end.Line {
		line := lines[start.Line]
		startCol := clampCol(start.Col, len(line))
		endCol := clampCol(end.Col, len(line))
		if startCol >= endCol {
			return ""
		}
		return line[startCol:endCol]
	}

	// Multi-line selection
	var result strings.Builder

	// First line (from start.Col to end)
	firstLine := lines[start.Line]
	startCol := clampCol(start.Col, len(firstLine))
	result.WriteString(firstLine[startCol:])
	result.WriteByte('\n')

	// Middle lines (full lines)
	for i := start.Line + 1; i < end.Line; i++ {
		result.WriteString(lines[i])
		result.WriteByte('\n')
	}

	// Last line (from start to end.Col)
	lastLine := lines[end.Line]
	endCol := clampCol(end.Col, len(lastLine))
	result.WriteString(lastLine[:endCol])

	return result.String()
}

// clampCol clamps a column value to valid range [0, lineLen].
func clampCol(col, lineLen int) int {
	if col < 0 {
		return 0
	}
	if col > lineLen {
		return lineLen
	}
	return col
}

// SelectionStyle is the default style for selected text.
var SelectionStyle = NewStyle().WithBackground(ColorBlue).WithForeground(ColorWhite)

// ScreenToTextPosition converts screen coordinates to text position.
// This helper handles scrolled content and line number offsets.
func ScreenToTextPosition(
	screenX, screenY int,
	scrollY int,
	lineNumberWidth int,
	lines []string,
) TextPosition {
	// Adjust for scroll
	contentY := screenY + scrollY

	// Adjust for line numbers
	textX := screenX - lineNumberWidth
	if textX < 0 {
		textX = 0
	}

	// Clamp line
	line := contentY
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		line = len(lines) - 1
		if line < 0 {
			return TextPosition{Line: 0, Col: 0}
		}
	}

	// Convert screen X to byte offset, accounting for wide characters
	col := screenXToByteOffset(lines[line], textX)

	return TextPosition{Line: line, Col: col}
}

// screenXToByteOffset converts a screen X position to a byte offset in a string,
// properly handling wide characters (emoji, CJK, etc.).
func screenXToByteOffset(s string, screenX int) int {
	currentX := 0
	for i, r := range s {
		if currentX >= screenX {
			return i
		}
		currentX += runewidth.RuneWidth(r)
	}
	return len(s)
}

// byteOffsetToScreenX converts a byte offset to screen X position.
func byteOffsetToScreenX(s string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset >= len(s) {
		return runewidth.StringWidth(s)
	}
	return runewidth.StringWidth(s[:byteOffset])
}

// CopySelectionToClipboard copies the selected text to the system clipboard.
// Returns true if text was copied, false if selection was empty or copy failed.
func CopySelectionToClipboard(lines []string, sel TextSelection) bool {
	text := ExtractSelectedText(lines, sel)
	if text == "" {
		return false
	}

	err := clipboard.Write(text)
	return err == nil
}

// SelectWord selects the word at the given position.
// Returns the updated selection.
func SelectWord(lines []string, pos TextPosition) TextSelection {
	if pos.Line >= len(lines) {
		return TextSelection{}
	}

	line := lines[pos.Line]
	if pos.Col >= len(line) {
		pos.Col = len(line)
	}

	// Find word boundaries
	start := pos.Col
	end := pos.Col

	// Scan backward to find start of word
	for start > 0 {
		r, size := decodeLastRune(line[:start])
		if !isWordRune(r) {
			break
		}
		start -= size
	}

	// Scan forward to find end of word
	for end < len(line) {
		r, size := decodeRune(line[end:])
		if !isWordRune(r) {
			break
		}
		end += size
	}

	return TextSelection{
		Start:  TextPosition{Line: pos.Line, Col: start},
		End:    TextPosition{Line: pos.Line, Col: end},
		Active: true,
	}
}

// SelectLine selects the entire line at the given position.
func SelectLine(lines []string, lineNum int) TextSelection {
	if lineNum >= len(lines) || lineNum < 0 {
		return TextSelection{}
	}

	return TextSelection{
		Start:  TextPosition{Line: lineNum, Col: 0},
		End:    TextPosition{Line: lineNum, Col: len(lines[lineNum])},
		Active: true,
	}
}

// isWordRune returns true if the rune is part of a word.
func isWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}

// decodeRune decodes the first rune from a string.
func decodeRune(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}
	for i, r := range s {
		_ = i
		return r, len(string(r))
	}
	return 0, 0
}

// decodeLastRune decodes the last rune from a string.
func decodeLastRune(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}
	var lastRune rune
	var lastSize int
	for _, r := range s {
		lastRune = r
		lastSize = len(string(r))
	}
	return lastRune, lastSize
}
