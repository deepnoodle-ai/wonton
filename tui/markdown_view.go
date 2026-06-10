package tui

import (
	"github.com/deepnoodle-ai/wonton/runewidth"
)

// MarkdownView displays rendered markdown content.
type MarkdownView struct {
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
func Markdown(content string, scrollY *int) *MarkdownView {
	return &MarkdownView{
		content:  content,
		scrollY:  scrollY,
		theme:    DefaultMarkdownTheme(),
		maxWidth: 0,
		renderer: NewMarkdownRenderer(),
	}
}

// Theme sets the markdown theme.
func (m *MarkdownView) Theme(theme MarkdownTheme) *MarkdownView {
	m.theme = theme
	m.renderer.Theme = theme
	m.rendered = nil // invalidate cache
	return m
}

// MaxWidth sets the maximum width for text wrapping.
func (m *MarkdownView) MaxWidth(w int) *MarkdownView {
	m.maxWidth = w
	m.renderer.MaxWidth = w
	m.rendered = nil // invalidate cache
	return m
}

// Height sets a fixed height for the view.
func (m *MarkdownView) Height(h int) *MarkdownView {
	m.height = h
	return m
}

// renderContent renders the markdown if needed.
func (m *MarkdownView) renderContent(width int) {
	if m.rendered != nil && m.lastWidth == width {
		return // use cached render
	}

	// Use the configured maxWidth for paragraph text wrapping (readability),
	// but allow tables to use the full available width.
	wrapWidth := width
	if m.maxWidth > 0 && m.maxWidth < width {
		wrapWidth = m.maxWidth
	}
	m.renderer.MaxWidth = wrapWidth
	m.renderer.TableMaxWidth = width
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

func (m *MarkdownView) size(maxWidth, maxHeight int) (int, int) {
	w := maxWidth
	if w <= 0 {
		w = m.maxWidth
	}

	// Render to determine dimensions
	m.renderContent(w)

	// When no width constraint is given, measure the widest rendered line
	if w <= 0 && m.rendered != nil {
		for _, line := range m.rendered.Lines {
			lw := line.Indent
			for _, seg := range line.Segments {
				lw += runewidth.StringWidth(seg.Text)
			}
			if lw > w {
				w = lw
			}
		}
	}

	h := m.height
	if h == 0 && m.rendered != nil {
		h = len(m.rendered.Lines)
	}

	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (m *MarkdownView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Render markdown content
	m.renderContent(width)

	if m.rendered == nil {
		return
	}

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
				// Render as hyperlink using segment's text (which may be a
				// single word after wrapping), not the full hyperlink text
				link := *seg.Hyperlink
				link.Text = seg.Text
				ctx.PrintHyperlink(x, y, link)
			} else {
				// Render as styled text (truncated at frame edge)
				ctx.PrintTruncated(x, y, seg.Text, seg.Style)
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
func (m *MarkdownView) GetLineCount() int {
	if m.rendered == nil {
		return 0
	}
	return len(m.rendered.Lines)
}
