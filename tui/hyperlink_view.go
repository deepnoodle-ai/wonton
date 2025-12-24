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

// Bg sets the background color.
func (h *hyperlinkView) Bg(c Color) *hyperlinkView {
	h.style = h.style.WithBackground(c)
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
