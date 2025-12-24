package tui

// linkListView displays a vertical list of hyperlinks.
type linkListView struct {
	links []Hyperlink
	style Style
	gap   int
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
		links: links,
		style: NewStyle().WithUnderline().WithForeground(ColorBlue),
		gap:   0,
	}
}

// Style sets the style for all links.
func (l *linkListView) Style(s Style) *linkListView {
	l.style = s
	return l
}

// Gap sets vertical spacing between links.
func (l *linkListView) Gap(g int) *linkListView {
	l.gap = g
	return l
}

// Spacing sets vertical spacing between links.
// Deprecated: Use Gap instead for consistency with other layout components.
func (l *linkListView) Spacing(s int) *linkListView {
	l.gap = s
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
	if l.gap > 0 && len(l.links) > 1 {
		h += (len(l.links) - 1) * l.gap
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
	rowHeight := 1 + l.gap

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
