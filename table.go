package gooey

import (
	"github.com/mattn/go-runewidth"
)

// Column represents a table column
type Column struct {
	Title string
	Width int // If 0, will be auto-calculated based on content
}

// Table represents a scrollable data table
type Table struct {
	X, Y          int
	Width, Height int
	Columns       []Column
	Rows          [][]string
	ScrollY       int // Index of the first visible row
	SelectedRow   int // Index of the selected row (relative to data)
	Style         Style
	HeaderStyle   Style
	SelectedStyle Style
	Focused       bool
	ShowHeader    bool
	columnWidths  []int // Calculated widths
}

// NewTable creates a new table
func NewTable(x, y, width, height int) *Table {
	return &Table{
		X:             x,
		Y:             y,
		Width:         width,
		Height:        height,
		Rows:          make([][]string, 0),
		Style:         NewStyle(),
		HeaderStyle:   NewStyle().WithBold().WithUnderline(),
		SelectedStyle: NewStyle().WithReverse(),
		Focused:       true,
		ShowHeader:    true,
	}
}

// SetColumns sets the table columns
func (t *Table) SetColumns(cols []Column) *Table {
	t.Columns = cols
	return t
}

// SetRows sets the table data
func (t *Table) SetRows(rows [][]string) *Table {
	t.Rows = rows
	// Reset selection/scroll if needed, or clamp
	if t.SelectedRow >= len(rows) {
		t.SelectedRow = len(rows) - 1
	}
	if t.SelectedRow < 0 && len(rows) > 0 {
		t.SelectedRow = 0
	}
	return t
}

// calculateColumnWidths computes the actual width for each column
func (t *Table) calculateColumnWidths() {
	if len(t.Columns) == 0 {
		return
	}

	t.columnWidths = make([]int, len(t.Columns))

	// 1. Respect explicit widths
	// 2. Auto-calculate based on content if width is 0
	// 3. Distribute remaining space? For now, just use calculated/explicit.

	for i, col := range t.Columns {
		if col.Width > 0 {
			t.columnWidths[i] = col.Width
		} else {
			// Auto-width: max(header, content max width)
			maxW := runewidth.StringWidth(col.Title)
			for _, row := range t.Rows {
				if i < len(row) {
					w := runewidth.StringWidth(row[i])
					if w > maxW {
						maxW = w
					}
				}
			}
			t.columnWidths[i] = maxW + 2 // Add padding
		}
	}
}

// Draw renders the table
func (t *Table) Draw(frame RenderFrame) {
	t.calculateColumnWidths() // Recalculate every draw? Or cache? Light enough for now.

	currentY := t.Y

	// Draw Header
	if t.ShowHeader {
		currentX := t.X
		for i, col := range t.Columns {
			width := t.columnWidths[i]
			title := col.Title
			// Truncate if needed
			if runewidth.StringWidth(title) > width {
				title = runewidth.Truncate(title, width, "…")
			}
			// Pad
			padding := width - runewidth.StringWidth(title)
			if padding > 0 {
				title += repeat(" ", padding)
			}

			frame.PrintStyled(currentX, currentY, title, t.HeaderStyle)
			currentX += width
		}
		currentY++
	}

	// Calculate visible rows
	availableHeight := t.Height
	if t.ShowHeader {
		availableHeight--
	}
	if availableHeight <= 0 {
		return
	}

	// Adjust ScrollY to ensure SelectedRow is visible
	if t.SelectedRow < t.ScrollY {
		t.ScrollY = t.SelectedRow
	}
	if t.SelectedRow >= t.ScrollY+availableHeight {
		t.ScrollY = t.SelectedRow - availableHeight + 1
	}

	// Clamp ScrollY
	if t.ScrollY < 0 {
		t.ScrollY = 0
	}
	maxScroll := len(t.Rows) - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if t.ScrollY > maxScroll {
		t.ScrollY = maxScroll
	}

	// Draw Rows
	for i := 0; i < availableHeight; i++ {
		rowIndex := t.ScrollY + i
		if rowIndex >= len(t.Rows) {
			break
		}

		row := t.Rows[rowIndex]
		style := t.Style
		if t.Focused && rowIndex == t.SelectedRow {
			style = t.SelectedStyle
		}

		currentX := t.X
		for colIdx, cell := range row {
			if colIdx >= len(t.columnWidths) {
				break
			}
			width := t.columnWidths[colIdx]

			// Truncate/Pad
			if runewidth.StringWidth(cell) > width {
				cell = runewidth.Truncate(cell, width, "…")
			}
			padding := width - runewidth.StringWidth(cell)
			paddedCell := cell
			if padding > 0 {
				paddedCell += repeat(" ", padding)
			}

			frame.PrintStyled(currentX, currentY+i, paddedCell, style)
			currentX += width
		}
	}

	// Scrollbar hints?
	if t.ScrollY > 0 {
		// Up arrow indicator
	}
	if t.ScrollY < maxScroll {
		// Down arrow indicator
	}
}

// HandleKey handles key events
func (t *Table) HandleKey(event KeyEvent) bool {
	if !t.Focused {
		return false
	}

	switch event.Key {
	case KeyArrowUp:
		if t.SelectedRow > 0 {
			t.SelectedRow--
			return true
		}
	case KeyArrowDown:
		if t.SelectedRow < len(t.Rows)-1 {
			t.SelectedRow++
			return true
		}
	case KeyPageUp:
		t.SelectedRow -= t.Height
		if t.SelectedRow < 0 {
			t.SelectedRow = 0
		}
		return true
	case KeyPageDown:
		t.SelectedRow += t.Height
		if t.SelectedRow >= len(t.Rows) {
			t.SelectedRow = len(t.Rows) - 1
		}
		return true
	case KeyHome:
		t.SelectedRow = 0
		return true
	case KeyEnd:
		t.SelectedRow = len(t.Rows) - 1
		return true
	}
	return false
}

// Helper
func repeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	var result string
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
