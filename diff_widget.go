package gooey

import (
	"github.com/mattn/go-runewidth"
)

// DiffViewer is a widget that displays file diffs with syntax highlighting
type DiffViewer struct {
	BaseWidget
	diff     *Diff
	renderer *DiffRenderer
	rendered []RenderedDiffLine
	scrollY  int
	language string // Programming language for syntax highlighting
}

// NewDiffViewer creates a new diff viewer widget
func NewDiffViewer(diffText, language string) (*DiffViewer, error) {
	diff, err := ParseUnifiedDiff(diffText)
	if err != nil {
		return nil, err
	}

	viewer := &DiffViewer{
		diff:     diff,
		renderer: NewDiffRenderer(),
		language: language,
		scrollY:  0,
	}
	viewer.BaseWidget = NewBaseWidget()

	return viewer, nil
}

// NewDiffViewerFromDiff creates a diff viewer from a parsed Diff
func NewDiffViewerFromDiff(diff *Diff, language string) *DiffViewer {
	viewer := &DiffViewer{
		diff:     diff,
		renderer: NewDiffRenderer(),
		language: language,
		scrollY:  0,
	}
	viewer.BaseWidget = NewBaseWidget()

	return viewer
}

// SetLanguage sets the programming language for syntax highlighting
func (dv *DiffViewer) SetLanguage(language string) {
	if dv.language != language {
		dv.language = language
		dv.rendered = nil // Force re-render
		dv.MarkDirty()
	}
}

// SetTheme sets the diff theme
func (dv *DiffViewer) SetTheme(theme DiffTheme) {
	dv.renderer.Theme = theme
	dv.rendered = nil // Force re-render
	dv.MarkDirty()
}

// SetRenderer sets a custom renderer
func (dv *DiffViewer) SetRenderer(renderer *DiffRenderer) {
	dv.renderer = renderer
	dv.rendered = nil // Force re-render
	dv.MarkDirty()
}

// ScrollTo scrolls to the specified line
func (dv *DiffViewer) ScrollTo(line int) {
	if dv.scrollY != line {
		dv.scrollY = line
		dv.MarkDirty()
	}
}

// ScrollBy scrolls by the specified number of lines
func (dv *DiffViewer) ScrollBy(delta int) {
	dv.ScrollTo(dv.scrollY + delta)
}

// GetScrollPosition returns the current scroll position
func (dv *DiffViewer) GetScrollPosition() int {
	return dv.scrollY
}

// Init initializes the widget
func (dv *DiffViewer) Init() {
	dv.rendered = nil
}

// Destroy cleans up the widget
func (dv *DiffViewer) Destroy() {
	// Nothing to clean up
}

// HandleKey handles keyboard events (required by Widget interface)
func (dv *DiffViewer) HandleKey(event KeyEvent) bool {
	switch event.Key {
	case KeyArrowUp:
		if dv.CanScrollUp() {
			dv.ScrollBy(-1)
			return true
		}
	case KeyArrowDown:
		if dv.CanScrollDown() {
			dv.ScrollBy(1)
			return true
		}
	case KeyPageUp:
		bounds := dv.GetBounds()
		dv.ScrollBy(-bounds.Dy())
		return true
	case KeyPageDown:
		bounds := dv.GetBounds()
		dv.ScrollBy(bounds.Dy())
		return true
	case KeyHome:
		dv.ScrollTo(0)
		return true
	case KeyEnd:
		dv.ScrollTo(dv.GetLineCount())
		return true
	}

	return false
}

// Draw renders the diff viewer
func (dv *DiffViewer) Draw(frame RenderFrame) {
	bounds := dv.GetBounds()
	if bounds.Empty() {
		return
	}

	// Render diff if not already rendered
	if dv.rendered == nil {
		dv.rendered = dv.renderer.RenderDiff(dv.diff, dv.language)
	}

	// Determine drawing area
	frameWidth, frameHeight := frame.Size()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if we're in a SubFrame
	inSubFrame := (frameWidth == width && frameHeight == height)

	startX := bounds.Min.X
	startY := bounds.Min.Y

	if inSubFrame {
		startX = 0
		startY = 0
	}

	// Clamp scroll position
	maxScroll := len(dv.rendered) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if dv.scrollY > maxScroll {
		dv.scrollY = maxScroll
	}
	if dv.scrollY < 0 {
		dv.scrollY = 0
	}

	// Render visible lines
	endLine := dv.scrollY + height
	if endLine > len(dv.rendered) {
		endLine = len(dv.rendered)
	}

	y := startY
	for i := dv.scrollY; i < endLine && y < startY+height; i++ {
		line := dv.rendered[i]
		x := startX

		// Draw background if specified
		if line.BgColor != nil {
			bgStyle := NewStyle().WithBgRGB(*line.BgColor)
			// Fill the entire line width with background
			for col := 0; col < width; col++ {
				frame.SetCell(startX+col, y, ' ', bgStyle)
			}
		}

		// Draw line numbers if enabled
		if dv.renderer.ShowLineNums {
			lineNumStyle := dv.renderer.Theme.LineNumStyle
			if line.BgColor != nil {
				lineNumStyle = lineNumStyle.WithBgRGB(*line.BgColor)
			}

			// Old line number
			frame.PrintStyled(x, y, line.LineNumOld, lineNumStyle)
			x += runewidth.StringWidth(line.LineNumOld)

			// Separator
			frame.PrintStyled(x, y, " ", lineNumStyle)
			x++

			// New line number
			frame.PrintStyled(x, y, line.LineNumNew, lineNumStyle)
			x += runewidth.StringWidth(line.LineNumNew)

			// Separator
			frame.PrintStyled(x, y, " │ ", lineNumStyle)
			x += 3
		}

		// Draw content segments
		for _, seg := range line.Segments {
			if x >= startX+width {
				break
			}

			// Apply background color if specified
			style := seg.Style
			if line.BgColor != nil {
				style = style.WithBgRGB(*line.BgColor)
			}

			frame.PrintStyled(x, y, seg.Text, style)
			x += runewidth.StringWidth(seg.Text)
		}

		y++
	}
}

// GetLineCount returns the total number of rendered lines
func (dv *DiffViewer) GetLineCount() int {
	if dv.rendered == nil {
		dv.rendered = dv.renderer.RenderDiff(dv.diff, dv.language)
	}
	return len(dv.rendered)
}

// CanScrollUp returns true if the viewer can scroll up
func (dv *DiffViewer) CanScrollUp() bool {
	return dv.scrollY > 0
}

// CanScrollDown returns true if the viewer can scroll down
func (dv *DiffViewer) CanScrollDown() bool {
	bounds := dv.GetBounds()
	if bounds.Empty() || dv.rendered == nil {
		return false
	}

	maxScroll := len(dv.rendered) - bounds.Dy()
	return dv.scrollY < maxScroll
}

// GetDiff returns the underlying Diff object
func (dv *DiffViewer) GetDiff() *Diff {
	return dv.diff
}

// SetDiff updates the diff content
func (dv *DiffViewer) SetDiff(diff *Diff) {
	dv.diff = diff
	dv.rendered = nil
	dv.scrollY = 0
	dv.MarkDirty()
}

// SetDiffText updates the diff content from text
func (dv *DiffViewer) SetDiffText(diffText string) error {
	diff, err := ParseUnifiedDiff(diffText)
	if err != nil {
		return err
	}

	dv.SetDiff(diff)
	return nil
}
