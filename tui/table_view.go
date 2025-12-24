package tui

import (
	"image"
	"strings"

	"github.com/mattn/go-runewidth"
)

// TableColumn represents a table column configuration.
type TableColumn struct {
	Title    string
	Width    int // If 0, auto-calculated based on content
	MinWidth int // Minimum width (won't shrink below this)
}

// tableView displays a scrollable data table as a declarative view.
type tableView struct {
	columns              []TableColumn
	rows                 [][]string
	selected             *int
	onSelect             func(row int)
	scrollY              int // internal scroll position
	style                Style
	headerStyle          Style
	selectedStyle        Style
	showHeader           bool
	width                int
	height               int
	columnWidths         []int // calculated column widths
	uppercaseHeaders     bool
	maxColumnWidth       int  // 0 = no limit
	invertSelectedColors bool // invert fg/bg on selected row
	headerBottomBorder   bool
	columnGap            int  // gap between columns (default 2)
	fillWidth            bool // expand to fill container width
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
		columns:            columns,
		selected:           selected,
		style:              NewStyle(),
		headerStyle:        NewStyle().WithBold(),
		selectedStyle:      NewStyle().WithReverse(),
		showHeader:         true,
		headerBottomBorder: true,
		columnGap:          2,
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

// Size sets both width and height at once.
func (t *tableView) Size(w, h int) *tableView {
	t.width = w
	t.height = h
	return t
}

// UppercaseHeaders enables or disables automatic uppercasing of header text.
func (t *tableView) UppercaseHeaders(uppercase bool) *tableView {
	t.uppercaseHeaders = uppercase
	return t
}

// MaxColumnWidth sets the maximum width for any column.
// If a column's calculated width exceeds this value, it will be truncated.
// Set to 0 for no limit (default).
func (t *tableView) MaxColumnWidth(maxWidth int) *tableView {
	t.maxColumnWidth = maxWidth
	return t
}

// InvertSelectedColors enables or disables color inversion on the selected row.
// When enabled, foreground and background colors are swapped for better visibility.
func (t *tableView) InvertSelectedColors(invert bool) *tableView {
	t.invertSelectedColors = invert
	return t
}

// HeaderBottomBorder enables or disables a bottom border line under the header row.
func (t *tableView) HeaderBottomBorder(show bool) *tableView {
	t.headerBottomBorder = show
	return t
}

// ColumnGap sets the gap (in spaces) between columns. Default is 2.
func (t *tableView) ColumnGap(gap int) *tableView {
	t.columnGap = gap
	return t
}

// FillWidth causes the table to expand to fill its container width.
// Extra space is distributed to auto-sized columns (those without explicit Width).
// If all columns have explicit widths, extra space goes to the largest column.
func (t *tableView) FillWidth() *tableView {
	t.fillWidth = true
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

		// Apply maxColumnWidth limit if set
		if t.maxColumnWidth > 0 && t.columnWidths[i] > t.maxColumnWidth {
			t.columnWidths[i] = t.maxColumnWidth
		}
	}
}

// expandColumns distributes extra space to columns.
// Prefers auto-sized columns (Width=0), falls back to largest column.
func (t *tableView) expandColumns(extraSpace int) {
	if extraSpace <= 0 || len(t.columnWidths) == 0 {
		return
	}

	// Find auto-sized columns (those without explicit Width)
	var autoSizedIdxs []int
	for i, col := range t.columns {
		if col.Width == 0 {
			autoSizedIdxs = append(autoSizedIdxs, i)
		}
	}

	// Distribute extra space
	if len(autoSizedIdxs) > 0 {
		// Distribute evenly among auto-sized columns
		perColumn := extraSpace / len(autoSizedIdxs)
		remainder := extraSpace % len(autoSizedIdxs)
		for i, idx := range autoSizedIdxs {
			t.columnWidths[idx] += perColumn
			if i < remainder {
				t.columnWidths[idx]++
			}
		}
	} else {
		// All columns have explicit widths; give extra to the largest
		maxIdx := 0
		for i, w := range t.columnWidths {
			if w > t.columnWidths[maxIdx] {
				maxIdx = i
			}
		}
		t.columnWidths[maxIdx] += extraSpace
	}
}

// effectiveMinWidth returns the minimum width for a column, using sensible defaults:
// 1. Explicit MinWidth if set
// 2. Explicit Width if set (you specified a size, you want it)
// 3. Header title width (never narrower than header)
func (t *tableView) effectiveMinWidth(colIdx int) int {
	if colIdx >= len(t.columns) {
		return 1
	}
	col := t.columns[colIdx]

	if col.MinWidth > 0 {
		return col.MinWidth
	}
	if col.Width > 0 {
		return col.Width
	}
	// Default to header width
	headerWidth := runewidth.StringWidth(col.Title)
	if headerWidth < 1 {
		return 1
	}
	return headerWidth
}

// fitColumnWidths adjusts column widths to fit within availableWidth.
// It shrinks columns proportionally when total width exceeds available space,
// but respects MinWidth constraints on columns.
func (t *tableView) fitColumnWidths(availableWidth int) {
	if len(t.columnWidths) == 0 || availableWidth <= 0 {
		return
	}

	// Calculate total width including gaps between columns
	totalGaps := 0
	if len(t.columnWidths) > 1 {
		totalGaps = (len(t.columnWidths) - 1) * t.columnGap
	}

	totalWidth := totalGaps
	for _, w := range t.columnWidths {
		totalWidth += w
	}

	// Available space for columns (excluding gaps)
	availableForColumns := availableWidth - totalGaps
	if availableForColumns < len(t.columnWidths) {
		availableForColumns = len(t.columnWidths)
	}

	// If we have extra space and fillWidth is enabled, expand columns
	if totalWidth < availableWidth && t.fillWidth {
		extraSpace := availableForColumns - (totalWidth - totalGaps)
		t.expandColumns(extraSpace)
		return
	}

	if totalWidth <= availableWidth {
		return // Already fits, no expansion needed
	}

	// Calculate total of min widths and shrinkable space
	totalMinWidth := 0
	for i := range t.columns {
		totalMinWidth += t.effectiveMinWidth(i)
	}

	// If min widths exceed available, just use min widths
	if totalMinWidth >= availableForColumns {
		for i := range t.columnWidths {
			t.columnWidths[i] = t.effectiveMinWidth(i)
		}
		return
	}

	// Shrink columns proportionally, respecting min widths
	// First pass: calculate how much we need to shrink
	columnTotal := totalWidth - totalGaps
	excess := columnTotal - availableForColumns

	// Shrink columns that can be shrunk (above their min width)
	for excess > 0 {
		// Find shrinkable columns and their total shrinkable space
		shrinkable := 0
		shrinkableWidth := 0
		for i, w := range t.columnWidths {
			minW := t.effectiveMinWidth(i)
			if w > minW {
				shrinkable++
				shrinkableWidth += w - minW
			}
		}

		if shrinkable == 0 || shrinkableWidth == 0 {
			break // Nothing more to shrink
		}

		// Shrink proportionally
		shrinkAmount := excess
		if shrinkAmount > shrinkableWidth {
			shrinkAmount = shrinkableWidth
		}

		for i := range t.columnWidths {
			minW := t.effectiveMinWidth(i)
			canShrink := t.columnWidths[i] - minW
			if canShrink > 0 {
				// Proportional share of shrinking
				share := int(float64(canShrink) / float64(shrinkableWidth) * float64(shrinkAmount))
				if share > canShrink {
					share = canShrink
				}
				t.columnWidths[i] -= share
				excess -= share
			}
		}
	}

	// Distribute any remaining space to the largest column
	newTotal := 0
	for _, w := range t.columnWidths {
		newTotal += w
	}
	if newTotal < availableForColumns && len(t.columnWidths) > 0 {
		maxIdx := 0
		for i, w := range t.columnWidths {
			if w > t.columnWidths[maxIdx] {
				maxIdx = i
			}
		}
		t.columnWidths[maxIdx] += availableForColumns - newTotal
	}
}

func (t *tableView) size(maxWidth, maxHeight int) (int, int) {
	t.calculateColumnWidths()

	// Calculate total width including gaps
	w := t.width
	if w == 0 {
		for _, cw := range t.columnWidths {
			w += cw
		}
		// Add gaps between columns
		if len(t.columnWidths) > 1 {
			w += (len(t.columnWidths) - 1) * t.columnGap
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

func (t *tableView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(t.columns) == 0 {
		return
	}

	t.calculateColumnWidths()
	t.fitColumnWidths(width) // Shrink columns to fit container

	currentY := 0

	// Draw header
	if t.showHeader {
		currentX := 0
		for i, col := range t.columns {
			w := t.columnWidths[i]
			title := col.Title

			// Apply uppercase if enabled
			if t.uppercaseHeaders {
				title = strings.ToUpper(title)
			}

			// Truncate if needed
			if runewidth.StringWidth(title) > w {
				title = runewidth.Truncate(title, w, "…")
			}
			// Pad
			padding := w - runewidth.StringWidth(title)
			if padding > 0 {
				title += repeatStr(" ", padding)
			}

			ctx.PrintStyled(currentX, currentY, title, t.headerStyle)
			currentX += w
			// Add gap after column (except last)
			if i < len(t.columns)-1 {
				currentX += t.columnGap
			}
		}
		currentY++

		// Draw header bottom border if enabled
		if t.headerBottomBorder {
			// Use actual table width (sum of columns + gaps)
			tableWidth := 0
			for _, cw := range t.columnWidths {
				tableWidth += cw
			}
			if len(t.columnWidths) > 1 {
				tableWidth += (len(t.columnWidths) - 1) * t.columnGap
			}
			if tableWidth > width {
				tableWidth = width
			}
			border := repeatStr("─", tableWidth)
			ctx.PrintStyled(0, currentY, border, t.style)
			currentY++
		}
	}

	// Calculate available height for rows
	availableHeight := height
	if t.showHeader {
		availableHeight--
		if t.headerBottomBorder {
			availableHeight--
		}
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
			// Apply color inversion if enabled
			if t.invertSelectedColors {
				style = invertColors(style)
			}
		}

		currentX := 0
		for colIdx, cell := range row {
			if colIdx >= len(t.columnWidths) {
				break
			}
			w := t.columnWidths[colIdx]

			// Truncate/Pad
			if runewidth.StringWidth(cell) > w {
				cell = runewidth.Truncate(cell, w, "…")
			}
			padding := w - runewidth.StringWidth(cell)
			paddedCell := cell
			if padding > 0 {
				paddedCell += repeatStr(" ", padding)
			}

			ctx.PrintStyled(currentX, currentY+i, paddedCell, style)
			currentX += w
			// Add gap after column (except last), filled with row style
			if colIdx < len(t.columnWidths)-1 && t.columnGap > 0 {
				ctx.PrintStyled(currentX, currentY+i, repeatStr(" ", t.columnGap), style)
				currentX += t.columnGap
			}
		}

		// Register clickable region for this row
		bounds := ctx.AbsoluteBounds()
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

// invertColors swaps the foreground and background colors of a style.
func invertColors(s Style) Style {
	// Swap standard colors
	s.Foreground, s.Background = s.Background, s.Foreground

	// Swap RGB colors if present
	s.FgRGB, s.BgRGB = s.BgRGB, s.FgRGB

	return s
}
