package gooey

import (
	"image"
)

// Grid represents a flexible grid layout container.
// It manages the positioning and sizing of widgets in a grid-like structure.
type Grid struct {
	rows    []GridRow
	columns []GridCol
	cells   [][]*GridCell // 2D slice of cells [row][col]

	terminal *Terminal
	bounds   image.Rectangle // The area this grid occupies

	// Optional: Focus management for interactive grids
	focusedRow int
	focusedCol int
}

// GridRow defines properties for a row in the grid.
type GridRow struct {
	Height int // Fixed height in lines. If 0, uses weight.
	Weight int // Relative height weight. Used if Height is 0.
}

// GridCol defines properties for a column in the grid.
type GridCol struct {
	Width  int // Fixed width in characters. If 0, uses weight.
	Weight int // Relative width weight. Used if Width is 0.
}

// GridCell holds a widget and its position within the grid.
type GridCell struct {
	Widget  Widget
	Row     int
	Col     int
	RowSpan int             // Number of rows this cell spans (default 1)
	ColSpan int             // Number of columns this cell spans (default 1)
	bounds  image.Rectangle // Calculated bounds for the widget to draw within
}

// NewGrid creates a new Grid layout.
func NewGrid(terminal *Terminal) *Grid {
	return &Grid{
		terminal: terminal,
		rows:     make([]GridRow, 0),
		columns:  make([]GridCol, 0),
		cells:    make([][]*GridCell, 0),
		bounds:   image.Rectangle{}, // Will be set during Draw
	}
}

// AddRow adds a new row definition to the grid.
// height: fixed height in lines. Use 0 for dynamic sizing based on weight.
// weight: relative weight for dynamic height calculation. Only used if height is 0.
func (g *Grid) AddRow(height, weight int) *Grid {
	g.rows = append(g.rows, GridRow{Height: height, Weight: weight})
	// Expand cells slice to accommodate new row
	newRow := make([]*GridCell, len(g.columns))
	g.cells = append(g.cells, newRow)
	return g
}

// AddCol adds a new column definition to the grid.
// width: fixed width in characters. Use 0 for dynamic sizing based on weight.
// weight: relative weight for dynamic width calculation. Only used if width is 0.
func (g *Grid) AddCol(width, weight int) *Grid {
	g.columns = append(g.columns, GridCol{Width: width, Weight: weight})
	// Expand existing rows to accommodate new column
	for rIdx := range g.cells {
		g.cells[rIdx] = append(g.cells[rIdx], nil)
	}
	return g
}

// AddWidget adds a widget to a specific cell in the grid.
// row, col are 0-indexed.
func (g *Grid) AddWidget(widget Widget, row, col int) error {
	return g.AddWidgetSpan(widget, row, col, 1, 1)
}

// AddWidgetSpan adds a widget to a specific cell in the grid with row and column spanning.
// row, col are 0-indexed. rowSpan and colSpan specify how many rows/columns the widget occupies.
func (g *Grid) AddWidgetSpan(widget Widget, row, col, rowSpan, colSpan int) error {
	if row >= len(g.rows) || col >= len(g.columns) {
		return ErrInvalidGridPosition
	}
	if row+rowSpan > len(g.rows) || col+colSpan > len(g.columns) {
		return ErrInvalidGridPosition // Span exceeds grid bounds
	}

	// Check if any of the cells in the span are already occupied
	for r := row; r < row+rowSpan; r++ {
		for c := col; c < col+colSpan; c++ {
			if g.cells[r][c] != nil {
				return ErrGridCellOccupied
			}
		}
	}

	// Create the main cell
	cell := &GridCell{
		Widget:  widget,
		Row:     row,
		Col:     col,
		RowSpan: rowSpan,
		ColSpan: colSpan,
	}

	// Mark all cells in the span (the first cell gets the actual widget, others get a placeholder)
	for r := row; r < row+rowSpan; r++ {
		for c := col; c < col+colSpan; c++ {
			if r == row && c == col {
				g.cells[r][c] = cell
			} else {
				// Mark as occupied by a spanning cell
				g.cells[r][c] = &GridCell{
					Widget:  nil, // Placeholder
					Row:     row,
					Col:     col,
					RowSpan: -1, // Marker to indicate this is part of a span
					ColSpan: -1,
				}
			}
		}
	}

	return nil
}

// ErrInvalidGridPosition is returned when trying to add a widget to a non-existent grid cell.
var ErrInvalidGridPosition = gooeyError("invalid grid position: row or column out of bounds")

// ErrGridCellOccupied is returned when trying to add a widget to an already occupied grid cell.
var ErrGridCellOccupied = gooeyError("grid cell already occupied")

// Draw renders the grid and its widgets to the provided RenderFrame.
// It calculates the bounds for each cell and passes a clipped frame to each widget.
func (g *Grid) Draw(frame RenderFrame) {
	g.bounds = frame.GetBounds() // Set the grid's total drawing area

	width, height := g.bounds.Dx(), g.bounds.Dy()
	startX, startY := g.bounds.Min.X, g.bounds.Min.Y

	// Calculate row heights
	rowHeights := g.calculateRowHeights(height)
	// Calculate column widths
	colWidths := g.calculateColWidths(width)

	// Build cumulative position arrays for easier span calculation
	rowYPositions := make([]int, len(rowHeights)+1)
	rowYPositions[0] = startY
	for i, h := range rowHeights {
		rowYPositions[i+1] = rowYPositions[i] + h
	}

	colXPositions := make([]int, len(colWidths)+1)
	colXPositions[0] = startX
	for i, w := range colWidths {
		colXPositions[i+1] = colXPositions[i] + w
	}

	// Draw each cell
	for rIdx := range g.cells {
		for cIdx := range g.cells[rIdx] {
			cell := g.cells[rIdx][cIdx]
			if cell == nil {
				continue
			}

			// Skip placeholder cells (part of a span)
			if cell.RowSpan == -1 || cell.ColSpan == -1 {
				continue
			}

			// Skip cells with no widget
			if cell.Widget == nil {
				continue
			}

			// Calculate bounds including span
			rowSpan := cell.RowSpan
			colSpan := cell.ColSpan
			if rowSpan == 0 {
				rowSpan = 1
			}
			if colSpan == 0 {
				colSpan = 1
			}

			x1 := colXPositions[cIdx]
			y1 := rowYPositions[rIdx]
			x2 := colXPositions[cIdx+colSpan]
			y2 := rowYPositions[rIdx+rowSpan]

			cell.bounds = image.Rect(x1, y1, x2, y2)

			// Create a sub-frame for the cell and draw the widget
			cellFrame := frame.SubFrame(cell.bounds)
			cell.Widget.Draw(cellFrame)
		}
	}
}

// HandleKey delegates key events to the appropriate widget.
// For now, it will simply iterate through all widgets. Later, we can add focus management.
func (g *Grid) HandleKey(event KeyEvent) bool {
	for rIdx := range g.cells {
		for cIdx := range g.cells[rIdx] {
			cell := g.cells[rIdx][cIdx]
			if cell == nil {
				continue
			}
			// Skip placeholder cells and cells without widgets
			if cell.RowSpan == -1 || cell.ColSpan == -1 || cell.Widget == nil {
				continue
			}
			if cell.Widget.HandleKey(event) {
				return true
			}
		}
	}
	return false
}

func (g *Grid) calculateRowHeights(totalHeight int) []int {
	numRows := len(g.rows)
	if numRows == 0 {
		return []int{}
	}

	calculatedHeights := make([]int, numRows)
	fixedHeightSum := 0
	weightedHeightSum := 0
	totalWeight := 0

	// First pass: handle fixed heights and sum weights
	for i, row := range g.rows {
		if row.Height > 0 {
			calculatedHeights[i] = row.Height
			fixedHeightSum += row.Height
		} else {
			totalWeight += row.Weight
		}
	}

	remainingHeight := totalHeight - fixedHeightSum
	if remainingHeight < 0 {
		remainingHeight = 0 // Prevent negative heights if fixed heights exceed total
	}

	// Second pass: distribute remaining height by weight
	if totalWeight > 0 {
		for i, row := range g.rows {
			if row.Height == 0 {
				h := (remainingHeight * row.Weight) / totalWeight
				calculatedHeights[i] = h
				weightedHeightSum += h
			}
		}
	}

	// Adjust for rounding errors
	diff := remainingHeight - weightedHeightSum
	for i := 0; i < diff; i++ {
		if i < numRows { // Distribute remainder to first 'diff' rows
			calculatedHeights[i]++
		}
	}

	return calculatedHeights
}

func (g *Grid) calculateColWidths(totalWidth int) []int {
	numCols := len(g.columns)
	if numCols == 0 {
		return []int{}
	}

	calculatedWidths := make([]int, numCols)
	fixedWidthSum := 0
	weightedWidthSum := 0
	totalWeight := 0

	// First pass: handle fixed widths and sum weights
	for i, col := range g.columns {
		if col.Width > 0 {
			calculatedWidths[i] = col.Width
			fixedWidthSum += col.Width
		} else {
			totalWeight += col.Weight
		}
	}

	remainingWidth := totalWidth - fixedWidthSum
	if remainingWidth < 0 {
		remainingWidth = 0 // Prevent negative widths if fixed widths exceed total
	}

	// Second pass: distribute remaining width by weight
	if totalWeight > 0 {
		for i, col := range g.columns {
			if col.Width == 0 {
				w := (remainingWidth * col.Weight) / totalWeight
				calculatedWidths[i] = w
				weightedWidthSum += w
			}
		}
	}

	// Adjust for rounding errors
	diff := remainingWidth - weightedWidthSum
	for i := 0; i < diff; i++ {
		if i < numCols { // Distribute remainder to first 'diff' columns
			calculatedWidths[i]++
		}
	}

	return calculatedWidths
}
