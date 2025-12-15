package tui

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/mattn/go-runewidth"
)

// codeView displays syntax-highlighted code.
type codeView struct {
	code        string
	language    string
	theme       string
	showNumbers bool
	startLine   int
	scrollY     *int
	height      int
	highlighted [][]StyledSegment
	lastHash    uint64
}

// Code creates a code view with syntax highlighting.
//
// Example:
//
//	Code(`func main() {
//	    fmt.Println("Hello")
//	}`, "go")
func Code(code string, language string) *codeView {
	return &codeView{
		code:        code,
		language:    language,
		theme:       "monokai",
		showNumbers: true,
		startLine:   1,
	}
}

// Language sets the programming language for syntax highlighting.
// If not set or unknown, falls back to plain text.
func (c *codeView) Language(lang string) *codeView {
	c.language = lang
	c.highlighted = nil // invalidate cache
	return c
}

// Theme sets the syntax highlighting theme.
// Available themes: monokai, dracula, github, vs, solarized-dark, solarized-light, etc.
func (c *codeView) Theme(theme string) *codeView {
	c.theme = theme
	c.highlighted = nil // invalidate cache
	return c
}

// LineNumbers enables or disables line numbers.
func (c *codeView) LineNumbers(show bool) *codeView {
	c.showNumbers = show
	return c
}

// StartLine sets the starting line number (default: 1).
func (c *codeView) StartLine(n int) *codeView {
	c.startLine = n
	return c
}

// ScrollY sets the scroll position pointer.
func (c *codeView) ScrollY(scrollY *int) *codeView {
	c.scrollY = scrollY
	return c
}

// Height sets a fixed height for the view.
func (c *codeView) Height(h int) *codeView {
	c.height = h
	return c
}

// highlight performs syntax highlighting and caches the result.
func (c *codeView) highlight() {
	if c.highlighted != nil {
		return
	}

	// Get lexer
	lexer := lexers.Get(c.language)
	if lexer == nil {
		lexer = lexers.Analyse(c.code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Get style
	style := styles.Get(c.theme)
	if style == nil {
		style = styles.Fallback
	}

	// Tokenize
	iterator, err := lexer.Tokenise(nil, c.code)
	if err != nil {
		// Fallback to plain text
		c.highlighted = c.plainLines()
		return
	}

	// Convert tokens to styled segments
	c.highlighted = make([][]StyledSegment, 0)
	var currentLine []StyledSegment

	for _, token := range iterator.Tokens() {
		tokenStyle := c.chromaToStyle(style.Get(token.Type))

		// Handle newlines in token
		parts := strings.Split(token.Value, "\n")
		for i, part := range parts {
			if i > 0 {
				c.highlighted = append(c.highlighted, currentLine)
				currentLine = nil
			}
			if part != "" {
				currentLine = append(currentLine, StyledSegment{
					Text:  part,
					Style: tokenStyle,
				})
			}
		}
	}

	// Add final line
	if len(currentLine) > 0 || len(c.highlighted) == 0 {
		c.highlighted = append(c.highlighted, currentLine)
	}
}

// plainLines creates unhighlighted lines for fallback.
func (c *codeView) plainLines() [][]StyledSegment {
	lines := strings.Split(c.code, "\n")
	result := make([][]StyledSegment, len(lines))
	for i, line := range lines {
		result[i] = []StyledSegment{{Text: line, Style: NewStyle()}}
	}
	return result
}

// chromaToStyle converts a chroma style entry to our Style.
func (c *codeView) chromaToStyle(entry chroma.StyleEntry) Style {
	style := NewStyle()

	if entry.Colour.IsSet() {
		style = style.WithFgRGB(RGB{
			R: entry.Colour.Red(),
			G: entry.Colour.Green(),
			B: entry.Colour.Blue(),
		})
	}

	if entry.Bold == chroma.Yes {
		style = style.WithBold()
	}
	if entry.Italic == chroma.Yes {
		style = style.WithItalic()
	}
	if entry.Underline == chroma.Yes {
		style = style.WithUnderline()
	}

	return style
}

// lineNumberWidth calculates the width needed for line numbers.
func (c *codeView) lineNumberWidth() int {
	if !c.showNumbers {
		return 0
	}
	c.highlight()
	maxLine := c.startLine + len(c.highlighted) - 1
	width := 1
	for maxLine >= 10 {
		maxLine /= 10
		width++
	}
	return width + 2 // number + space + separator
}

func (c *codeView) size(maxWidth, maxHeight int) (int, int) {
	c.highlight()

	// Calculate width
	lnWidth := c.lineNumberWidth()
	maxCodeWidth := 0
	for _, line := range c.highlighted {
		lineWidth := 0
		for _, seg := range line {
			lineWidth += runewidth.StringWidth(seg.Text)
		}
		if lineWidth > maxCodeWidth {
			maxCodeWidth = lineWidth
		}
	}

	w := lnWidth + maxCodeWidth
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}

	h := c.height
	if h == 0 {
		h = len(c.highlighted)
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}

	return w, h
}

func (c *codeView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	c.highlight()

	lnWidth := c.lineNumberWidth()
	lnStyle := NewStyle().WithForeground(ColorBrightBlack)
	separatorStyle := NewStyle().WithForeground(ColorBrightBlack)

	// Get scroll position
	scrollY := 0
	if c.scrollY != nil {
		scrollY = *c.scrollY
	}

	// Clamp scroll
	maxScroll := len(c.highlighted) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	if scrollY < 0 {
		scrollY = 0
	}

	// Update scroll pointer
	if c.scrollY != nil && *c.scrollY != scrollY {
		*c.scrollY = scrollY
	}

	// Render visible lines
	for y := 0; y < height && scrollY+y < len(c.highlighted); y++ {
		lineIdx := scrollY + y
		line := c.highlighted[lineIdx]

		x := 0

		// Render line number
		if c.showNumbers {
			lineNum := c.startLine + lineIdx
			numStr := padLeft(lineNum, lnWidth-2)
			ctx.PrintStyled(x, y, numStr, lnStyle)
			x += len(numStr)
			ctx.PrintStyled(x, y, " ", separatorStyle)
			x++
		}

		// Render code
		for _, seg := range line {
			if x >= width {
				break
			}
			text := seg.Text
			segWidth := runewidth.StringWidth(text)
			if x+segWidth > width {
				// Truncate
				text = truncateToWidth(text, width-x)
			}
			ctx.PrintStyled(x, y, text, seg.Style)
			x += runewidth.StringWidth(text)
		}
	}
}

// GetLineCount returns the total number of lines.
func (c *codeView) GetLineCount() int {
	c.highlight()
	return len(c.highlighted)
}

// padLeft pads a number with spaces on the left.
func padLeft(n, width int) string {
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if s == "" {
		s = "0"
	}
	for len(s) < width {
		s = " " + s
	}
	return s
}

// truncateToWidth truncates a string to fit within a given display width.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	var result strings.Builder
	width := 0

	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if width+rw > maxWidth {
			break
		}
		result.WriteRune(r)
		width += rw
	}

	return result.String()
}

// AvailableThemes returns a list of available syntax highlighting themes.
func AvailableThemes() []string {
	return styles.Names()
}

// AvailableLanguages returns a list of available language lexers.
func AvailableLanguages() []string {
	names := lexers.Names(false)
	return names
}
