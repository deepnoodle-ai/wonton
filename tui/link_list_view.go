package tui

// hyperlinkView displays a clickable hyperlink (declarative view)
type linkListView struct {
	links   []Hyperlink
	style   Style
	spacing int
}

// LinkList creates a vertical list of hyperlinks.
//
// Example:
//
//	LinkList(
//		NewHyperlink("https://go.dev", "Go"),
//		NewHyperlink("https://github.com", "GitHub"),
//	)
func LinkList(links ...Hyperlink) *linkListView {
	return &linkListView{
		links:   links,
		style:   NewStyle().WithUnderline().WithForeground(ColorBlue),
		spacing: 0,
	}
}

// Style sets the style for all links.
func (l *linkListView) Style(s Style) *linkListView {
	l.style = s
	return l
}

// Spacing sets vertical spacing between links.
func (l *linkListView) Spacing(s int) *linkListView {
	l.spacing = s
	return l
}

func (l *linkListView) size(maxWidth, maxHeight int) (int, int) {
	w := 0
	for _, link := range l.links {
		linkW, _ := MeasureText(link.Text)
		if linkW > w {
			w = linkW
		}
	}
	h := len(l.links)
	if l.spacing > 0 && len(l.links) > 1 {
		h += (len(l.links) - 1) * l.spacing
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (l *linkListView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(l.links) == 0 {
		return
	}

	y := 0
	rowHeight := 1 + l.spacing

	for _, link := range l.links {
		if y >= height {
			break
		}
		styledLink := link.WithStyle(l.style)
		ctx.PrintHyperlink(0, y, styledLink)
		y += rowHeight
	}
}

// inlineLinkView displays multiple links on a single line with separators
