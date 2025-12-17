package tui

import "image"

// gridView displays a grid of clickable cells
type gridView struct {
	cols       int
	rows       int
	cellWidth  int
	cellHeight int
	gap        int
	cells      [][]gridCell
	onClick    func(col, row int)
}

type gridCell struct {
	char  rune
	style Style
}

// CellGrid creates a grid of clickable cells.
//
// Example:
//
//	CellGrid(5, 5).CellSize(6, 3).Gap(1).OnClick(func(c, r int) { ... })
func CellGrid(cols, rows int) *gridView {
	// Initialize cells with default values
	cells := make([][]gridCell, rows)
	for r := range cells {
		cells[r] = make([]gridCell, cols)
		for c := range cells[r] {
			cells[r][c] = gridCell{
				char:  ' ',
				style: NewStyle().WithBackground(ColorBrightBlack),
			}
		}
	}

	return &gridView{
		cols:       cols,
		rows:       rows,
		cellWidth:  4,
		cellHeight: 2,
		gap:        1,
		cells:      cells,
	}
}

// CellSize sets the size of each cell.
func (g *gridView) CellSize(width, height int) *gridView {
	g.cellWidth = width
	g.cellHeight = height
	return g
}

// Gap sets the gap between cells.
func (g *gridView) Gap(gap int) *gridView {
	g.gap = gap
	return g
}

// OnClick sets the callback when a cell is clicked.
func (g *gridView) OnClick(fn func(col, row int)) *gridView {
	g.onClick = fn
	return g
}

// SetCell sets the style for a specific cell.
func (g *gridView) SetCell(col, row int, style Style) *gridView {
	if row >= 0 && row < g.rows && col >= 0 && col < g.cols {
		g.cells[row][col].style = style
	}
	return g
}

// SetCellChar sets the character and style for a specific cell.
func (g *gridView) SetCellChar(col, row int, char rune, style Style) *gridView {
	if row >= 0 && row < g.rows && col >= 0 && col < g.cols {
		g.cells[row][col].char = char
		g.cells[row][col].style = style
	}
	return g
}

// SetAllCells sets all cells using a callback.
func (g *gridView) SetAllCells(fn func(col, row int) Style) *gridView {
	for r := 0; r < g.rows; r++ {
		for c := 0; c < g.cols; c++ {
			g.cells[r][c].style = fn(c, r)
		}
	}
	return g
}

func (g *gridView) size(maxWidth, maxHeight int) (int, int) {
	w := g.cols*g.cellWidth + (g.cols-1)*g.gap
	h := g.rows*g.cellHeight + (g.rows-1)*g.gap

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (g *gridView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	for row := 0; row < g.rows; row++ {
		for col := 0; col < g.cols; col++ {
			// Calculate cell position
			cellX := col * (g.cellWidth + g.gap)
			cellY := row * (g.cellHeight + g.gap)

			// Get cell properties
			cell := g.cells[row][col]

			// Draw cell
			for y := 0; y < g.cellHeight; y++ {
				for x := 0; x < g.cellWidth; x++ {
					if cellX+x < width && cellY+y < height {
						ctx.SetCell(cellX+x, cellY+y, cell.char, cell.style)
					}
				}
			}

			// Register click region if callback provided
			if g.onClick != nil {
				bounds := ctx.AbsoluteBounds()
				clickBounds := image.Rect(
					bounds.Min.X+cellX,
					bounds.Min.Y+cellY,
					bounds.Min.X+cellX+g.cellWidth,
					bounds.Min.Y+cellY+g.cellHeight,
				)
				c, r := col, row // capture for closure
				interactiveRegistry.RegisterButton(clickBounds, func() {
					g.onClick(c, r)
				})
			}
		}
	}
}

// colorGridView displays a grid that cycles through colors on click
type colorGridView struct {
	cols          int
	rows          int
	cellWidth     int
	cellHeight    int
	gap           int
	colors        []Color
	state         [][]int // color index for each cell
	onStateChange func(col, row, colorIndex int)
}

// ColorGrid creates a grid where clicking cycles through colors.
// state should be a 2D slice tracking the color index for each cell.
//
// Example:
//
//	ColorGrid(5, 5, app.gridState, []Color{ColorBlack, ColorRed, ColorGreen, ColorBlue})
func ColorGrid(cols, rows int, state [][]int, colors []Color) *colorGridView {
	if len(colors) == 0 {
		colors = []Color{ColorBrightBlack, ColorRed, ColorGreen, ColorBlue, ColorYellow}
	}
	return &colorGridView{
		cols:       cols,
		rows:       rows,
		cellWidth:  4,
		cellHeight: 2,
		gap:        1,
		colors:     colors,
		state:      state,
	}
}

// CellSize sets the size of each cell.
func (g *colorGridView) CellSize(width, height int) *colorGridView {
	g.cellWidth = width
	g.cellHeight = height
	return g
}

// Gap sets the gap between cells.
func (g *colorGridView) Gap(gap int) *colorGridView {
	g.gap = gap
	return g
}

// OnStateChange sets a callback when a cell's color changes.
func (g *colorGridView) OnStateChange(fn func(col, row, colorIndex int)) *colorGridView {
	g.onStateChange = fn
	return g
}

func (g *colorGridView) size(maxWidth, maxHeight int) (int, int) {
	w := g.cols*g.cellWidth + (g.cols-1)*g.gap
	h := g.rows*g.cellHeight + (g.rows-1)*g.gap

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (g *colorGridView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	for row := 0; row < g.rows; row++ {
		for col := 0; col < g.cols; col++ {
			// Calculate cell position
			cellX := col * (g.cellWidth + g.gap)
			cellY := row * (g.cellHeight + g.gap)

			// Get color from state
			colorIdx := 0
			if row < len(g.state) && col < len(g.state[row]) {
				colorIdx = g.state[row][col]
				if colorIdx < 0 || colorIdx >= len(g.colors) {
					colorIdx = 0
				}
			}
			style := NewStyle().WithBackground(g.colors[colorIdx])

			// Draw cell
			for y := 0; y < g.cellHeight; y++ {
				for x := 0; x < g.cellWidth; x++ {
					if cellX+x < width && cellY+y < height {
						ctx.SetCell(cellX+x, cellY+y, ' ', style)
					}
				}
			}

			// Register click to cycle color
			bounds := ctx.AbsoluteBounds()
			clickBounds := image.Rect(
				bounds.Min.X+cellX,
				bounds.Min.Y+cellY,
				bounds.Min.X+cellX+g.cellWidth,
				bounds.Min.Y+cellY+g.cellHeight,
			)
			c, r := col, row // capture for closure
			interactiveRegistry.RegisterButton(clickBounds, func() {
				if r < len(g.state) && c < len(g.state[r]) {
					g.state[r][c] = (g.state[r][c] + 1) % len(g.colors)
					if g.onStateChange != nil {
						g.onStateChange(c, r, g.state[r][c])
					}
				}
			})
		}
	}
}

// charGridView displays a grid of characters
type charGridView struct {
	data      [][]rune
	style     Style
	cellWidth int
}

// CharGrid creates a grid displaying characters.
//
// Example:
//
//	CharGrid([][]rune{
//		{'X', 'O', 'X'},
//		{'O', 'X', 'O'},
//		{'X', 'O', 'X'},
//	})
func CharGrid(data [][]rune) *charGridView {
	return &charGridView{
		data:      data,
		style:     NewStyle(),
		cellWidth: 1,
	}
}

// Style sets the style for all characters.
func (g *charGridView) Style(s Style) *charGridView {
	g.style = s
	return g
}

// Fg sets the foreground color.
func (g *charGridView) Fg(c Color) *charGridView {
	g.style = g.style.WithForeground(c)
	return g
}

// CellWidth sets the width per character cell (for spacing).
func (g *charGridView) CellWidth(w int) *charGridView {
	g.cellWidth = w
	return g
}

func (g *charGridView) size(maxWidth, maxHeight int) (int, int) {
	h := len(g.data)
	w := 0
	for _, row := range g.data {
		rowW := len(row) * g.cellWidth
		if rowW > w {
			w = rowW
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

func (g *charGridView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(g.data) == 0 {
		return
	}

	for y, row := range g.data {
		if y >= height {
			break
		}
		for x, char := range row {
			cellX := x * g.cellWidth
			if cellX >= width {
				break
			}
			ctx.SetCell(cellX, y, char, g.style)
		}
	}
}
