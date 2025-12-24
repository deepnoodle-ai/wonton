package tui

// hyperlinkView displays a clickable hyperlink (declarative view)
type inlineLinkView struct {
	links     []Hyperlink
	separator string
	style     Style
}

// InlineLinks creates a horizontal row of hyperlinks with separators.
//
// Example:
//
//	InlineLinks(" | ",
//		NewHyperlink("https://go.dev", "Go"),
//		NewHyperlink("https://github.com", "GitHub"),
//	)
func InlineLinks(separator string, links ...Hyperlink) *inlineLinkView {
	return &inlineLinkView{
		links:     links,
		separator: separator,
		style:     NewStyle().WithUnderline().WithForeground(ColorBlue),
	}
}

// Style sets the style for all links.
func (i *inlineLinkView) Style(s Style) *inlineLinkView {
	i.style = s
	return i
}

func (i *inlineLinkView) size(maxWidth, maxHeight int) (int, int) {
	w := 0
	sepW, _ := MeasureText(i.separator)
	for idx, link := range i.links {
		linkW, _ := MeasureText(link.Text)
		w += linkW
		if idx < len(i.links)-1 {
			w += sepW
		}
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (i *inlineLinkView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(i.links) == 0 {
		return
	}

	x := 0
	sepW, _ := MeasureText(i.separator)
	sepStyle := NewStyle() // plain style for separator

	for idx, link := range i.links {
		if x >= width {
			break
		}

		styledLink := link.WithStyle(i.style)
		ctx.PrintHyperlink(x, 0, styledLink)
		linkW, _ := MeasureText(link.Text)
		x += linkW

		// Add separator (except after last link)
		if idx < len(i.links)-1 {
			ctx.PrintStyled(x, 0, i.separator, sepStyle)
			x += sepW
		}
	}
}
