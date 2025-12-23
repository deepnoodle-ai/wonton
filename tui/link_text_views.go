package tui

// hyperlinkView displays a clickable hyperlink (declarative view)
type hyperlinkView struct {
	url     string
	text    string
	style   Style
	showURL bool // show URL in fallback format
}

// Link creates a declarative hyperlink view.
// If text is empty, the URL will be used as the display text.
//
// Example:
//
//	Link("https://github.com", "GitHub")
//	Link("https://example.com", "").Fg(ColorCyan)  // uses URL as text
func Link(url, text string) *hyperlinkView {
	if text == "" {
		text = url
	}
	return &hyperlinkView{
		url:   url,
		text:  text,
		style: NewStyle().WithUnderline().WithForeground(ColorBlue),
	}
}

// Fg sets the foreground color.
func (h *hyperlinkView) Fg(c Color) *hyperlinkView {
	h.style = h.style.WithForeground(c)
	return h
}

// Bold makes the link text bold.
func (h *hyperlinkView) Bold() *hyperlinkView {
	h.style = h.style.WithBold()
	return h
}

// Underline sets whether the link is underlined (default true).
func (h *hyperlinkView) Underline(u bool) *hyperlinkView {
	if u {
		h.style = h.style.WithUnderline()
	} else {
		// Create a new style without underline, preserving other properties
		h.style = Style{
			Foreground:    h.style.Foreground,
			Background:    h.style.Background,
			Bold:          h.style.Bold,
			Dim:           h.style.Dim,
			Italic:        h.style.Italic,
			Underline:     false,
			Strikethrough: h.style.Strikethrough,
			Blink:         h.style.Blink,
			Reverse:       h.style.Reverse,
			Hidden:        h.style.Hidden,
			FgRGB:         h.style.FgRGB,
			BgRGB:         h.style.BgRGB,
			URL:           h.style.URL,
		}
	}
	return h
}

// Style sets the complete style.
func (h *hyperlinkView) Style(s Style) *hyperlinkView {
	h.style = s
	return h
}

// ShowURL enables fallback format showing URL in parentheses.
func (h *hyperlinkView) ShowURL() *hyperlinkView {
	h.showURL = true
	return h
}

func (h *hyperlinkView) size(maxWidth, maxHeight int) (int, int) {
	w, _ := MeasureText(h.text)
	if h.showURL {
		w += len(" ()") + len(h.url)
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (h *hyperlinkView) render(ctx *RenderContext) {
	w, ht := ctx.Size()
	if w == 0 || ht == 0 {
		return
	}

	// Create the hyperlink with OSC 8 support
	link := NewHyperlink(h.url, h.text).WithStyle(h.style)

	if h.showURL {
		ctx.PrintHyperlinkFallback(0, 0, link)
	} else {
		ctx.PrintHyperlink(0, 0, link)
	}
}

// linkRowView displays a label followed by a hyperlink
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

// wrappedTextView displays text with wrapping and alignment
type wrappedTextView struct {
	text       string
	style      Style
	align      Alignment
	truncate   bool
	fillBg     bool
	flexFactor int
}

// WrappedText creates a text view with automatic word wrapping.
// By default, WrappedText is flexible (flex factor 1) so it will expand
// to fill available space in Group/Stack layouts.
//
// Example:
//
//	WrappedText("Long text that wraps...").Align(AlignCenter)
//	WrappedText("Text").Bg(ColorBlue).FillBg()
func WrappedText(text string) *wrappedTextView {
	return &wrappedTextView{
		text:       text,
		style:      NewStyle(),
		align:      AlignLeft,
		flexFactor: 1, // Default to flexible for proper wrapping in layouts
	}
}

// Flex sets the flex factor for this view.
// A higher value means this view gets more of the available space.
// Set to 0 to make the view non-flexible (fixed size).
func (w *wrappedTextView) Flex(factor int) *wrappedTextView {
	w.flexFactor = factor
	return w
}

// flex implements the Flexible interface.
func (w *wrappedTextView) flex() int {
	return w.flexFactor
}

// Fg sets the foreground color.
func (w *wrappedTextView) Fg(c Color) *wrappedTextView {
	w.style = w.style.WithForeground(c)
	return w
}

// Bg sets the background color.
func (w *wrappedTextView) Bg(c Color) *wrappedTextView {
	w.style = w.style.WithBackground(c)
	return w
}

// Bold makes the text bold.
func (w *wrappedTextView) Bold() *wrappedTextView {
	w.style = w.style.WithBold()
	return w
}

// Dim makes the text dimmed.
func (w *wrappedTextView) Dim() *wrappedTextView {
	w.style = w.style.WithDim()
	return w
}

// Style sets the complete style.
func (w *wrappedTextView) Style(s Style) *wrappedTextView {
	w.style = s
	return w
}

// Align sets the text alignment.
func (w *wrappedTextView) Align(a Alignment) *wrappedTextView {
	w.align = a
	return w
}

// Center is a shorthand for Align(AlignCenter).
func (w *wrappedTextView) Center() *wrappedTextView {
	w.align = AlignCenter
	return w
}

// Right is a shorthand for Align(AlignRight).
func (w *wrappedTextView) Right() *wrappedTextView {
	w.align = AlignRight
	return w
}

// Truncate enables truncation instead of wrapping.
func (w *wrappedTextView) Truncate() *wrappedTextView {
	w.truncate = true
	return w
}

// FillBg fills the background with the background color.
func (w *wrappedTextView) FillBg() *wrappedTextView {
	w.fillBg = true
	return w
}

func (w *wrappedTextView) size(maxWidth, maxHeight int) (int, int) {
	// For wrapped text, we expand to fill available width
	width, _ := MeasureText(w.text)
	if maxWidth > 0 && width > maxWidth {
		width = maxWidth
	}

	// Calculate height based on wrapped lines
	height := 1
	if maxWidth > 0 && !w.truncate {
		wrapped := WrapText(w.text, maxWidth)
		lines := splitLinesSimple(wrapped)
		height = len(lines)
		if maxHeight > 0 && height > maxHeight {
			height = maxHeight
		}
	}

	return width, height
}

func (wt *wrappedTextView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Fill background if requested
	if wt.fillBg {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				ctx.SetCell(x, y, ' ', wt.style)
			}
		}
	}

	// Process text
	displayText := wt.text
	if !wt.truncate && width > 0 {
		displayText = WrapText(displayText, width)
	}

	// Align text
	displayText = AlignText(displayText, width, wt.align)

	// Render
	lines := splitLinesSimple(displayText)
	for y, line := range lines {
		if y >= height {
			break
		}
		if wt.truncate {
			ctx.PrintTruncated(0, y, line, wt.style)
		} else {
			ctx.PrintStyled(0, y, line, wt.style)
		}
	}
}

// splitLinesSimple splits text on newlines (used by wrappedTextView)
func splitLinesSimple(s string) []string {
	if s == "" {
		return []string{}
	}

	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	} else if start == len(s) && len(s) > 0 && s[len(s)-1] == '\n' {
		// Trailing newline creates empty last line
		lines = append(lines, "")
	}
	return lines
}

// linkListView displays a list of hyperlinks
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
