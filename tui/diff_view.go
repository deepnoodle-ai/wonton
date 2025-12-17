package tui

import (
	"github.com/mattn/go-runewidth"
)

// diffView displays a file diff with syntax highlighting.
type diffView struct {
	diff      *Diff
	scrollY   *int
	language  string
	theme     DiffTheme
	renderer  *DiffRenderer
	rendered  []RenderedDiffLine
	lastDiff  *Diff // for cache invalidation
	height    int
	showLines bool
}

// DiffView creates a diff view from a parsed Diff.
// scrollY should be a pointer to the scroll position (optional, can be nil).
//
// Example:
//
//	diff, _ := tui.ParseUnifiedDiff(diffText)
//	DiffView(diff, "go", &app.scrollY)
func DiffView(diff *Diff, language string, scrollY *int) *diffView {
	return &diffView{
		diff:      diff,
		scrollY:   scrollY,
		language:  language,
		theme:     DefaultDiffTheme(),
		renderer:  NewDiffRenderer(),
		showLines: true,
	}
}

// DiffViewFromText creates a diff view from diff text.
// Returns an error if the diff cannot be parsed.
func DiffViewFromText(diffText, language string, scrollY *int) (*diffView, error) {
	diff, err := ParseUnifiedDiff(diffText)
	if err != nil {
		return nil, err
	}
	return DiffView(diff, language, scrollY), nil
}

// Theme sets the diff theme.
func (d *diffView) Theme(theme DiffTheme) *diffView {
	d.theme = theme
	d.renderer.Theme = theme
	d.rendered = nil // invalidate cache
	return d
}

// Language sets the programming language for syntax highlighting.
func (d *diffView) Language(lang string) *diffView {
	d.language = lang
	d.rendered = nil // invalidate cache
	return d
}

// ShowLineNumbers enables or disables line numbers.
func (d *diffView) ShowLineNumbers(show bool) *diffView {
	d.showLines = show
	d.renderer.ShowLineNums = show
	d.rendered = nil // invalidate cache
	return d
}

// SyntaxHighlight enables or disables syntax highlighting.
func (d *diffView) SyntaxHighlight(enable bool) *diffView {
	d.renderer.SyntaxHighlight = enable
	d.rendered = nil // invalidate cache
	return d
}

// Height sets a fixed height for the view.
func (d *diffView) Height(h int) *diffView {
	d.height = h
	return d
}

// renderContent renders the diff if needed.
func (d *diffView) renderContent() {
	if d.rendered != nil && d.lastDiff == d.diff {
		return // use cached render
	}

	d.renderer.ShowLineNums = d.showLines
	d.rendered = d.renderer.RenderDiff(d.diff, d.language)
	d.lastDiff = d.diff
}

func (d *diffView) size(maxWidth, maxHeight int) (int, int) {
	// Render to get line count
	d.renderContent()

	w := maxWidth
	if w == 0 {
		w = 80
	}

	h := d.height
	if h == 0 && d.rendered != nil {
		h = len(d.rendered)
	}

	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (d *diffView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Render diff content
	d.renderContent()

	if len(d.rendered) == 0 {
		return
	}

	// Get scroll position
	scrollY := 0
	if d.scrollY != nil {
		scrollY = *d.scrollY
	}

	// Clamp scroll position
	maxScroll := len(d.rendered) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	if scrollY < 0 {
		scrollY = 0
	}

	// Update scroll pointer if clamped
	if d.scrollY != nil && *d.scrollY != scrollY {
		*d.scrollY = scrollY
	}

	// Render visible lines
	endLine := scrollY + height
	if endLine > len(d.rendered) {
		endLine = len(d.rendered)
	}

	y := 0
	for i := scrollY; i < endLine && y < height; i++ {
		line := d.rendered[i]
		x := 0

		// Draw background if specified
		if line.BgColor != nil {
			bgStyle := NewStyle().WithBgRGB(*line.BgColor)
			// Fill the entire line width with background
			for col := 0; col < width; col++ {
				ctx.SetCell(col, y, ' ', bgStyle)
			}
		}

		// Draw line numbers if enabled
		if d.renderer.ShowLineNums {
			lineNumStyle := d.renderer.Theme.LineNumStyle
			if line.BgColor != nil {
				lineNumStyle = lineNumStyle.WithBgRGB(*line.BgColor)
			}

			// Old line number
			ctx.PrintStyled(x, y, line.LineNumOld, lineNumStyle)
			x += runewidth.StringWidth(line.LineNumOld)

			// Separator
			ctx.PrintStyled(x, y, " ", lineNumStyle)
			x++

			// New line number
			ctx.PrintStyled(x, y, line.LineNumNew, lineNumStyle)
			x += runewidth.StringWidth(line.LineNumNew)

			// Separator
			ctx.PrintStyled(x, y, " │ ", lineNumStyle)
			x += 3
		}

		// Draw content segments
		for _, seg := range line.Segments {
			if x >= width {
				break
			}

			// Apply background color if specified
			style := seg.Style
			if line.BgColor != nil {
				style = style.WithBgRGB(*line.BgColor)
			}

			ctx.PrintStyled(x, y, seg.Text, style)
			x += runewidth.StringWidth(seg.Text)
		}

		y++
	}
}

// GetLineCount returns the total number of rendered lines.
func (d *diffView) GetLineCount() int {
	d.renderContent()
	return len(d.rendered)
}
