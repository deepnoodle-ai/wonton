package gooey

import (
	"image"

	"github.com/mattn/go-runewidth"
)

// TableColumn represents a table column configuration.
type TableColumn struct {
	Title string
	Width int // If 0, auto-calculated based on content
}

// tableView displays a scrollable data table as a declarative view.
type tableView struct {
	columns       []TableColumn
	rows          [][]string
	selected      *int
	onSelect      func(row int)
	scrollY       int // internal scroll position
	style         Style
	headerStyle   Style
	selectedStyle Style
	showHeader    bool
	width         int
	height        int
	columnWidths  []int // calculated column widths
}

// Table creates a new table view with the given columns.
// selected should be a pointer to the currently selected row index.
//
// Example:
//
//	Table([]TableColumn{{Title: "Name"}, {Title: "Age"}}, &app.selected).
//	    Rows([][]string{{"Alice", "30"}, {"Bob", "25"}})
func Table(columns []TableColumn, selected *int) *tableView {
	return &tableView{
		columns:       columns,
		selected:      selected,
		style:         NewStyle(),
		headerStyle:   NewStyle().WithBold().WithUnderline(),
		selectedStyle: NewStyle().WithReverse(),
		showHeader:    true,
	}
}

// Rows sets the table data.
func (t *tableView) Rows(rows [][]string) *tableView {
	t.rows = rows
	return t
}

// OnSelect sets a callback when a row is clicked.
func (t *tableView) OnSelect(fn func(row int)) *tableView {
	t.onSelect = fn
	return t
}

// ShowHeader enables or disables the header row.
func (t *tableView) ShowHeader(show bool) *tableView {
	t.showHeader = show
	return t
}

// Fg sets the foreground color for normal rows.
func (t *tableView) Fg(c Color) *tableView {
	t.style = t.style.WithForeground(c)
	return t
}

// Bg sets the background color for normal rows.
func (t *tableView) Bg(c Color) *tableView {
	t.style = t.style.WithBackground(c)
	return t
}

// Style sets the style for normal rows.
func (t *tableView) Style(s Style) *tableView {
	t.style = s
	return t
}

// HeaderStyle sets the style for the header row.
func (t *tableView) HeaderStyle(s Style) *tableView {
	t.headerStyle = s
	return t
}

// HeaderFg sets the foreground color for the header.
func (t *tableView) HeaderFg(c Color) *tableView {
	t.headerStyle = t.headerStyle.WithForeground(c)
	return t
}

// SelectedStyle sets the style for the selected row.
func (t *tableView) SelectedStyle(s Style) *tableView {
	t.selectedStyle = s
	return t
}

// SelectedFg sets the foreground color for the selected row.
func (t *tableView) SelectedFg(c Color) *tableView {
	t.selectedStyle = t.selectedStyle.WithForeground(c)
	return t
}

// SelectedBg sets the background color for the selected row.
func (t *tableView) SelectedBg(c Color) *tableView {
	t.selectedStyle = t.selectedStyle.WithBackground(c)
	return t
}

// Width sets a fixed width for the table.
func (t *tableView) Width(w int) *tableView {
	t.width = w
	return t
}

// Height sets a fixed height for the table.
func (t *tableView) Height(h int) *tableView {
	t.height = h
	return t
}

// calculateColumnWidths computes the actual width for each column.
func (t *tableView) calculateColumnWidths() {
	if len(t.columns) == 0 {
		return
	}

	t.columnWidths = make([]int, len(t.columns))

	for i, col := range t.columns {
		if col.Width > 0 {
			t.columnWidths[i] = col.Width
		} else {
			// Auto-width: max(header, content max width)
			maxW := runewidth.StringWidth(col.Title)
			for _, row := range t.rows {
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

func (t *tableView) size(maxWidth, maxHeight int) (int, int) {
	t.calculateColumnWidths()

	// Calculate total width
	w := t.width
	if w == 0 {
		for _, cw := range t.columnWidths {
			w += cw
		}
	}

	// Calculate height
	h := t.height
	if h == 0 {
		h = len(t.rows)
		if t.showHeader {
			h++
		}
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (t *tableView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() || len(t.columns) == 0 {
		return
	}

	t.calculateColumnWidths()
	subFrame := frame.SubFrame(bounds)

	currentY := 0

	// Draw header
	if t.showHeader {
		currentX := 0
		for i, col := range t.columns {
			width := t.columnWidths[i]
			title := col.Title
			// Truncate if needed
			if runewidth.StringWidth(title) > width {
				title = runewidth.Truncate(title, width, "…")
			}
			// Pad
			padding := width - runewidth.StringWidth(title)
			if padding > 0 {
				title += repeatStr(" ", padding)
			}

			subFrame.PrintStyled(currentX, currentY, title, t.headerStyle)
			currentX += width
		}
		currentY++
	}

	// Calculate available height for rows
	availableHeight := bounds.Dy()
	if t.showHeader {
		availableHeight--
	}
	if availableHeight <= 0 {
		return
	}

	// Get selected row
	selectedRow := 0
	if t.selected != nil {
		selectedRow = *t.selected
	}

	// Adjust scrollY to ensure selected row is visible
	if selectedRow < t.scrollY {
		t.scrollY = selectedRow
	}
	if selectedRow >= t.scrollY+availableHeight {
		t.scrollY = selectedRow - availableHeight + 1
	}

	// Clamp scrollY
	if t.scrollY < 0 {
		t.scrollY = 0
	}
	maxScroll := len(t.rows) - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if t.scrollY > maxScroll {
		t.scrollY = maxScroll
	}

	// Draw rows
	for i := 0; i < availableHeight; i++ {
		rowIndex := t.scrollY + i
		if rowIndex >= len(t.rows) {
			break
		}

		row := t.rows[rowIndex]
		style := t.style
		if rowIndex == selectedRow {
			style = t.selectedStyle
		}

		currentX := 0
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
				paddedCell += repeatStr(" ", padding)
			}

			subFrame.PrintStyled(currentX, currentY+i, paddedCell, style)
			currentX += width
		}

		// Register clickable region for this row
		rowBounds := image.Rect(
			bounds.Min.X,
			bounds.Min.Y+currentY+i,
			bounds.Max.X,
			bounds.Min.Y+currentY+i+1,
		)
		idx := rowIndex // capture for closure
		interactiveRegistry.RegisterButton(rowBounds, func() {
			if t.selected != nil {
				*t.selected = idx
			}
			if t.onSelect != nil {
				t.onSelect(idx)
			}
		})
	}
}

// repeatStr repeats a string n times.
func repeatStr(s string, count int) string {
	if count <= 0 {
		return ""
	}
	var result string
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
