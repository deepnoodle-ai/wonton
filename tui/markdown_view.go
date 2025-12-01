package tui

import (
	"image"

	"github.com/mattn/go-runewidth"
)

// markdownView displays rendered markdown content.
type markdownView struct {
	content   string
	scrollY   *int
	theme     MarkdownTheme
	maxWidth  int
	height    int
	renderer  *MarkdownRenderer
	rendered  *RenderedMarkdown
	lastWidth int // track last render width for cache invalidation
}

// Markdown creates a markdown view with the given content.
// scrollY should be a pointer to the scroll position (optional, can be nil).
//
// Example:
//
//	Markdown(content, &app.scrollY).Theme(tui.DefaultMarkdownTheme())
func Markdown(content string, scrollY *int) *markdownView {
	return &markdownView{
		content:  content,
		scrollY:  scrollY,
		theme:    DefaultMarkdownTheme(),
		maxWidth: 80,
		renderer: NewMarkdownRenderer(),
	}
}

// Theme sets the markdown theme.
func (m *markdownView) Theme(theme MarkdownTheme) *markdownView {
	m.theme = theme
	m.renderer.Theme = theme
	m.rendered = nil // invalidate cache
	return m
}

// MaxWidth sets the maximum width for text wrapping.
func (m *markdownView) MaxWidth(w int) *markdownView {
	m.maxWidth = w
	m.renderer.MaxWidth = w
	m.rendered = nil // invalidate cache
	return m
}

// Height sets a fixed height for the view.
func (m *markdownView) Height(h int) *markdownView {
	m.height = h
	return m
}

// renderContent renders the markdown if needed.
func (m *markdownView) renderContent(width int) {
	if m.rendered != nil && m.lastWidth == width {
		return // use cached render
	}

	m.renderer.MaxWidth = width
	rendered, err := m.renderer.Render(m.content)
	if err != nil {
		m.rendered = &RenderedMarkdown{
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
	} else {
		m.rendered = rendered
	}
	m.lastWidth = width
}

func (m *markdownView) size(maxWidth, maxHeight int) (int, int) {
	w := m.maxWidth
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}

	// Render to get line count
	m.renderContent(w)

	h := m.height
	if h == 0 && m.rendered != nil {
		h = len(m.rendered.Lines)
	}

	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (m *markdownView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	width := bounds.Dx()
	height := bounds.Dy()

	// Render markdown content
	m.renderContent(width)

	if m.rendered == nil {
		return
	}

	subFrame := frame.SubFrame(bounds)

	// Get scroll position
	scrollY := 0
	if m.scrollY != nil {
		scrollY = *m.scrollY
	}

	// Clamp scroll position
	maxScroll := len(m.rendered.Lines) - height
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
	if m.scrollY != nil && *m.scrollY != scrollY {
		*m.scrollY = scrollY
	}

	// Render visible lines
	endLine := scrollY + height
	if endLine > len(m.rendered.Lines) {
		endLine = len(m.rendered.Lines)
	}

	y := 0
	for i := scrollY; i < endLine && y < height; i++ {
		line := m.rendered.Lines[i]

		// Apply indentation
		x := line.Indent

		// Render segments
		for _, seg := range line.Segments {
			if seg.Hyperlink != nil {
				// Render as hyperlink
				subFrame.PrintHyperlink(x, y, *seg.Hyperlink)
			} else {
				// Render as styled text
				subFrame.PrintStyled(x, y, seg.Text, seg.Style)
			}

			x += runewidth.StringWidth(seg.Text)

			// Check if we've exceeded the width
			if x >= width {
				break
			}
		}

		y++
	}
}

// GetLineCount returns the total number of rendered lines.
// This is useful for scroll calculations in HandleEvent.
func (m *markdownView) GetLineCount() int {
	if m.rendered == nil {
		return 0
	}
	return len(m.rendered.Lines)
}
