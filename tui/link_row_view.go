package tui

// hyperlinkView displays a clickable hyperlink (declarative view)
type linkRowView struct {
	label      string
	url        string
	linkText   string
	labelStyle Style
	linkStyle  Style
	gap        int
}

// LinkRow creates a label + hyperlink pair, useful for tables of links.
//
// Example:
//
//	LinkRow("Documentation", "https://docs.example.com", "docs.example.com")
func LinkRow(label, url, linkText string) *linkRowView {
	return &linkRowView{
		label:      label,
		url:        url,
		linkText:   linkText,
		labelStyle: NewStyle(),
		linkStyle:  NewStyle().WithUnderline().WithForeground(ColorBlue),
		gap:        2,
	}
}

// LabelFg sets the label foreground color.
func (l *linkRowView) LabelFg(c Color) *linkRowView {
	l.labelStyle = l.labelStyle.WithForeground(c)
	return l
}

// LinkFg sets the link foreground color.
func (l *linkRowView) LinkFg(c Color) *linkRowView {
	l.linkStyle = l.linkStyle.WithForeground(c)
	return l
}

// LabelStyle sets the label style.
func (l *linkRowView) LabelStyle(s Style) *linkRowView {
	l.labelStyle = s
	return l
}

// LinkStyle sets the link style.
func (l *linkRowView) LinkStyle(s Style) *linkRowView {
	l.linkStyle = s
	return l
}

// Gap sets the space between label and link.
func (l *linkRowView) Gap(g int) *linkRowView {
	l.gap = g
	return l
}

func (l *linkRowView) size(maxWidth, maxHeight int) (int, int) {
	labelW, _ := MeasureText(l.label)
	linkW, _ := MeasureText(l.linkText)
	w := labelW + l.gap + linkW
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (l *linkRowView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	x := 0

	// Draw label
	ctx.PrintStyled(x, 0, l.label, l.labelStyle)
	labelW, _ := MeasureText(l.label)
	x += labelW + l.gap

	// Draw link
	link := NewHyperlink(l.url, l.linkText).WithStyle(l.linkStyle)
	ctx.PrintHyperlink(x, 0, link)
}

// linkListView displays a list of hyperlinks
