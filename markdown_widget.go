package gooey

import (
	"image"
	"strings"

	"github.com/mattn/go-runewidth"
)

// MarkdownWidget is a composable widget that renders markdown content
type MarkdownWidget struct {
	BaseWidget
	content       string
	renderer      *MarkdownRenderer
	rendered      *RenderedMarkdown
	scrollY       int // Vertical scroll offset
	needsRerender bool
}

// NewMarkdownWidget creates a new markdown widget with the given content
func NewMarkdownWidget(content string) *MarkdownWidget {
	w := &MarkdownWidget{
		content:       content,
		renderer:      NewMarkdownRenderer(),
		needsRerender: true,
	}
	w.BaseWidget = NewBaseWidget()
	return w
}

// SetContent updates the markdown content
func (w *MarkdownWidget) SetContent(content string) {
	if w.content != content {
		w.content = content
		w.needsRerender = true
		w.MarkDirty()
	}
}

// GetContent returns the current markdown content
func (w *MarkdownWidget) GetContent() string {
	return w.content
}

// SetRenderer sets a custom markdown renderer
func (w *MarkdownWidget) SetRenderer(renderer *MarkdownRenderer) {
	w.renderer = renderer
	w.needsRerender = true
	w.MarkDirty()
}

// SetTheme sets the markdown theme
func (w *MarkdownWidget) SetTheme(theme MarkdownTheme) {
	w.renderer.Theme = theme
	w.needsRerender = true
	w.MarkDirty()
}

// ScrollTo scrolls to the specified line (0-indexed)
func (w *MarkdownWidget) ScrollTo(line int) {
	if w.scrollY != line {
		w.scrollY = line
		w.MarkDirty()
	}
}

// ScrollBy scrolls by the specified number of lines (positive = down, negative = up)
func (w *MarkdownWidget) ScrollBy(delta int) {
	w.ScrollTo(w.scrollY + delta)
}

// GetScrollPosition returns the current scroll position
func (w *MarkdownWidget) GetScrollPosition() int {
	return w.scrollY
}

// Init initializes the widget
func (w *MarkdownWidget) Init() {
	w.needsRerender = true
}

// Destroy cleans up the widget
func (w *MarkdownWidget) Destroy() {
	// Nothing to clean up
}

// Draw renders the markdown content to the frame
func (w *MarkdownWidget) Draw(frame RenderFrame) {
	bounds := w.GetBounds()
	if bounds.Empty() {
		return
	}

	// Re-render markdown if needed
	if w.needsRerender {
		w.renderMarkdown(bounds.Dx())
		w.needsRerender = false
	}

	if w.rendered == nil {
		return
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
	maxScroll := len(w.rendered.Lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if w.scrollY > maxScroll {
		w.scrollY = maxScroll
	}
	if w.scrollY < 0 {
		w.scrollY = 0
	}

	// Render visible lines
	endLine := w.scrollY + height
	if endLine > len(w.rendered.Lines) {
		endLine = len(w.rendered.Lines)
	}

	y := startY
	for i := w.scrollY; i < endLine && y < startY+height; i++ {
		line := w.rendered.Lines[i]

		// Apply indentation
		x := startX + line.Indent

		// Render segments
		for _, seg := range line.Segments {
			if seg.Hyperlink != nil {
				// Render as hyperlink
				frame.PrintHyperlink(x, y, *seg.Hyperlink)
			} else {
				// Render as styled text
				frame.PrintStyled(x, y, seg.Text, seg.Style)
			}

			x += runewidth.StringWidth(seg.Text)

			// Check if we've exceeded the width
			if x >= startX+width {
				break
			}
		}

		y++
	}
}

// renderMarkdown renders the markdown content using the renderer
func (w *MarkdownWidget) renderMarkdown(maxWidth int) {
	// Set max width based on widget bounds
	w.renderer.MaxWidth = maxWidth

	// Render the markdown
	rendered, err := w.renderer.Render(w.content)
	if err != nil {
		// On error, show the error message
		w.rendered = &RenderedMarkdown{
			Lines: []StyledLine{
				{
					Segments: []StyledSegment{
						{
							Text:  "Error rendering markdown: " + err.Error(),
							Style: NewStyle().WithForeground(ColorRed),
						},
					},
				},
			},
		}
		return
	}

	w.rendered = rendered
}

// GetLineCount returns the total number of rendered lines
func (w *MarkdownWidget) GetLineCount() int {
	if w.rendered == nil {
		return 0
	}
	return len(w.rendered.Lines)
}

// CanScrollUp returns true if the widget can scroll up
func (w *MarkdownWidget) CanScrollUp() bool {
	return w.scrollY > 0
}

// CanScrollDown returns true if the widget can scroll down
func (w *MarkdownWidget) CanScrollDown() bool {
	bounds := w.GetBounds()
	if bounds.Empty() || w.rendered == nil {
		return false
	}

	maxScroll := len(w.rendered.Lines) - bounds.Dy()
	return w.scrollY < maxScroll
}

// RenderToString renders the markdown to a plain string (useful for testing)
func (w *MarkdownWidget) RenderToString() string {
	if w.needsRerender {
		w.renderMarkdown(w.renderer.MaxWidth)
		w.needsRerender = false
	}

	if w.rendered == nil {
		return ""
	}

	var result strings.Builder
	for _, line := range w.rendered.Lines {
		// Add indentation
		if line.Indent > 0 {
			result.WriteString(strings.Repeat(" ", line.Indent))
		}

		// Add segments
		for _, seg := range line.Segments {
			result.WriteString(seg.Text)
		}

		result.WriteString("\n")
	}

	return result.String()
}

// MarkdownViewer is a higher-level widget that includes scrolling controls
type MarkdownViewer struct {
	BaseWidget
	content   *MarkdownWidget
	container *Container
}

// NewMarkdownViewer creates a new markdown viewer with scroll support
func NewMarkdownViewer(content string) *MarkdownViewer {
	viewer := &MarkdownViewer{
		content: NewMarkdownWidget(content),
	}
	viewer.BaseWidget = NewBaseWidget()

	// Create container with the markdown widget
	viewer.container = NewContainer(NewVBoxLayout(0))
	viewer.container.AddChild(viewer.content)

	return viewer
}

// SetContent updates the markdown content
func (v *MarkdownViewer) SetContent(content string) {
	v.content.SetContent(content)
}

// GetContent returns the current markdown content
func (v *MarkdownViewer) GetContent() string {
	return v.content.GetContent()
}

// SetTheme sets the markdown theme
func (v *MarkdownViewer) SetTheme(theme MarkdownTheme) {
	v.content.SetTheme(theme)
}

// ScrollTo scrolls to the specified line
func (v *MarkdownViewer) ScrollTo(line int) {
	v.content.ScrollTo(line)
}

// ScrollBy scrolls by the specified number of lines
func (v *MarkdownViewer) ScrollBy(delta int) {
	v.content.ScrollBy(delta)
}

// GetScrollPosition returns the current scroll position
func (v *MarkdownViewer) GetScrollPosition() int {
	return v.content.GetScrollPosition()
}

// Init initializes the viewer
func (v *MarkdownViewer) Init() {
	v.content.Init()
	v.container.Init()
}

// Destroy cleans up the viewer
func (v *MarkdownViewer) Destroy() {
	v.content.Destroy()
	v.container.Destroy()
}

// SetBounds sets the bounds of the viewer
func (v *MarkdownViewer) SetBounds(bounds image.Rectangle) {
	v.BaseWidget.SetBounds(bounds)
	v.container.SetBounds(bounds)
	v.content.SetBounds(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
}

// Draw renders the viewer
func (v *MarkdownViewer) Draw(frame RenderFrame) {
	v.container.Draw(frame)
}

// HandleKey handles keyboard events (required by Widget interface)
func (w *MarkdownWidget) HandleKey(event KeyEvent) bool {
	// By default, markdown widget doesn't handle key events
	// The MarkdownViewer will handle scrolling
	return false
}

// HandleKey handles keyboard events for scrolling
func (v *MarkdownViewer) HandleKey(event KeyEvent) bool {
	switch event.Key {
	case KeyArrowUp:
		if v.content.CanScrollUp() {
			v.content.ScrollBy(-1)
			return true
		}
	case KeyArrowDown:
		if v.content.CanScrollDown() {
			v.content.ScrollBy(1)
			return true
		}
	case KeyPageUp:
		bounds := v.GetBounds()
		v.content.ScrollBy(-bounds.Dy())
		return true
	case KeyPageDown:
		bounds := v.GetBounds()
		v.content.ScrollBy(bounds.Dy())
		return true
	case KeyHome:
		v.content.ScrollTo(0)
		return true
	case KeyEnd:
		v.content.ScrollTo(v.content.GetLineCount())
		return true
	}

	return false
}
