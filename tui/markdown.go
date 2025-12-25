package tui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/mattn/go-runewidth"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownTheme defines the colors and styles for markdown rendering
type MarkdownTheme struct {
	// Heading styles by level (1-6)
	H1Style Style
	H2Style Style
	H3Style Style
	H4Style Style
	H5Style Style
	H6Style Style

	// Text styles
	BoldStyle          Style
	ItalicStyle        Style
	CodeStyle          Style
	LinkStyle          Style
	StrikethroughStyle Style

	// Block styles
	BlockQuoteStyle Style
	CodeBlockStyle  Style

	// List styles
	BulletChar string
	NumberFmt  string // Format string for numbered lists, e.g., "%d. "

	// Other
	HorizontalRuleChar  string
	HorizontalRuleStyle Style

	// Syntax highlighting theme name for code blocks
	SyntaxTheme string
}

// DefaultMarkdownTheme returns a default markdown theme with good contrast
func DefaultMarkdownTheme() MarkdownTheme {
	return MarkdownTheme{
		H1Style: NewStyle().WithBold().WithForeground(ColorCyan).WithUnderline(),
		H2Style: NewStyle().WithBold().WithForeground(ColorCyan),
		H3Style: NewStyle().WithBold().WithForeground(ColorBlue),
		H4Style: NewStyle().WithBold().WithForeground(ColorBlue),
		H5Style: NewStyle().WithForeground(ColorBlue),
		H6Style: NewStyle().WithForeground(ColorBlue),

		BoldStyle:          NewStyle().WithBold(),
		ItalicStyle:        NewStyle().WithItalic(),
		CodeStyle:          NewStyle().WithForeground(ColorYellow),
		LinkStyle:          NewStyle().WithForeground(ColorBlue).WithUnderline(),
		StrikethroughStyle: NewStyle().WithStrikethrough(),

		BlockQuoteStyle: NewStyle().WithForeground(ColorBrightBlack),
		CodeBlockStyle:  NewStyle().WithForeground(ColorGreen),

		BulletChar:          "•",
		NumberFmt:           "%d. ",
		HorizontalRuleChar:  "─",
		HorizontalRuleStyle: NewStyle().WithForeground(ColorBrightBlack),

		SyntaxTheme: "monokai",
	}
}

// MarkdownRenderer renders markdown content to styled terminal output
type MarkdownRenderer struct {
	Theme    MarkdownTheme
	MaxWidth int // Maximum width for text wrapping (0 = no limit)
	TabWidth int // Width of tab character in spaces
	parser   goldmark.Markdown
}

// NewMarkdownRenderer creates a new markdown renderer with the default theme
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		Theme:    DefaultMarkdownTheme(),
		MaxWidth: 80,
		TabWidth: 4,
		parser:   goldmark.New(),
	}
}

// WithTheme sets a custom theme for the renderer
func (mr *MarkdownRenderer) WithTheme(theme MarkdownTheme) *MarkdownRenderer {
	mr.Theme = theme
	return mr
}

// WithMaxWidth sets the maximum width for text wrapping
func (mr *MarkdownRenderer) WithMaxWidth(width int) *MarkdownRenderer {
	mr.MaxWidth = width
	return mr
}

// StyledLine represents a single line of rendered markdown with styled segments
type StyledLine struct {
	Segments []StyledSegment
	Indent   int // Indentation level in spaces
}

// StyledSegment represents a portion of text with a specific style
type StyledSegment struct {
	Text      string
	Style     Style
	Hyperlink *Hyperlink // Optional hyperlink
}

// RenderedMarkdown contains the fully rendered markdown content
type RenderedMarkdown struct {
	Lines []StyledLine
}

// Render parses and renders markdown content
func (mr *MarkdownRenderer) Render(markdown string) (*RenderedMarkdown, error) {
	source := []byte(markdown)
	reader := text.NewReader(source)

	doc := mr.parser.Parser().Parse(reader)

	result := &RenderedMarkdown{
		Lines: []StyledLine{},
	}

	ctx := &renderContext{
		renderer: mr,
		source:   source,
		result:   result,
		indent:   0,
	}

	mr.renderNode(doc, ctx)

	return result, nil
}

type renderContext struct {
	renderer     *MarkdownRenderer
	source       []byte
	result       *RenderedMarkdown
	indent       int
	listLevel    int
	listCounters []int // Stack of list counters for nested numbered lists
}

func (mr *MarkdownRenderer) renderNode(node ast.Node, ctx *renderContext) {
	switch n := node.(type) {
	case *ast.Document:
		mr.renderChildren(node, ctx)

	case *ast.Heading:
		mr.renderHeading(n, ctx)

	case *ast.Paragraph:
		mr.renderParagraph(n, ctx)

	case *ast.List:
		mr.renderList(n, ctx)

	case *ast.ListItem:
		mr.renderListItem(n, ctx)

	case *ast.FencedCodeBlock:
		mr.renderFencedCodeBlock(n, ctx)

	case *ast.CodeBlock:
		mr.renderCodeBlock(n, ctx)

	case *ast.Blockquote:
		mr.renderBlockquote(n, ctx)

	case *ast.ThematicBreak:
		mr.renderHorizontalRule(ctx)

	default:
		mr.renderChildren(node, ctx)
	}
}

func (mr *MarkdownRenderer) renderChildren(node ast.Node, ctx *renderContext) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		mr.renderNode(child, ctx)
	}
}

func (mr *MarkdownRenderer) renderHeading(node *ast.Heading, ctx *renderContext) {
	// Get heading text
	segments := mr.extractInlineSegments(node, ctx)

	// Apply heading style
	var style Style
	switch node.Level {
	case 1:
		style = mr.Theme.H1Style
	case 2:
		style = mr.Theme.H2Style
	case 3:
		style = mr.Theme.H3Style
	case 4:
		style = mr.Theme.H4Style
	case 5:
		style = mr.Theme.H5Style
	default:
		style = mr.Theme.H6Style
	}

	// Apply style to all segments
	for i := range segments {
		segments[i].Style = mr.mergeStyles(segments[i].Style, style)
	}

	ctx.result.Lines = append(ctx.result.Lines, StyledLine{
		Segments: segments,
		Indent:   ctx.indent,
	})

	// Add blank line after heading
	ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
}

func (mr *MarkdownRenderer) renderParagraph(node *ast.Paragraph, ctx *renderContext) {
	segments := mr.extractInlineSegments(node, ctx)

	// Word wrap if needed
	if mr.MaxWidth > 0 {
		lines := mr.wrapSegments(segments, mr.MaxWidth-ctx.indent)
		for _, line := range lines {
			ctx.result.Lines = append(ctx.result.Lines, StyledLine{
				Segments: line,
				Indent:   ctx.indent,
			})
		}
	} else {
		ctx.result.Lines = append(ctx.result.Lines, StyledLine{
			Segments: segments,
			Indent:   ctx.indent,
		})
	}

	// Add blank line after paragraph
	ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
}

func (mr *MarkdownRenderer) renderList(node *ast.List, ctx *renderContext) {
	ctx.listLevel++
	if node.IsOrdered() {
		ctx.listCounters = append(ctx.listCounters, node.Start)
	}

	mr.renderChildren(node, ctx)

	if node.IsOrdered() && len(ctx.listCounters) > 0 {
		ctx.listCounters = ctx.listCounters[:len(ctx.listCounters)-1]
	}
	ctx.listLevel--

	// Add blank line after list
	if ctx.listLevel == 0 {
		ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
	}
}

func (mr *MarkdownRenderer) renderListItem(node *ast.ListItem, ctx *renderContext) {
	parent := node.Parent().(*ast.List)

	// Determine list marker
	var marker string
	if parent.IsOrdered() {
		if len(ctx.listCounters) > 0 {
			marker = fmt.Sprintf(mr.Theme.NumberFmt, ctx.listCounters[len(ctx.listCounters)-1])
			ctx.listCounters[len(ctx.listCounters)-1]++
		}
	} else {
		marker = mr.Theme.BulletChar + " "
	}

	markerWidth := runewidth.StringWidth(marker)

	// Render first line with marker
	firstChild := true
	savedIndent := ctx.indent

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if firstChild {
			// Add marker before first element
			segments := []StyledSegment{{Text: marker, Style: NewStyle()}}

			// Extract inline segments from the first child
			if para, ok := child.(*ast.Paragraph); ok {
				paraSegments := mr.extractInlineSegments(para, ctx)
				segments = append(segments, paraSegments...)

				// Wrap the combined marker + content if needed
				if mr.MaxWidth > 0 {
					lines := mr.wrapSegments(segments, mr.MaxWidth-ctx.indent)
					for i, line := range lines {
						indent := ctx.indent
						if i > 0 {
							// Continuation lines get extra indent to align with text after marker
							indent += markerWidth
						}
						ctx.result.Lines = append(ctx.result.Lines, StyledLine{
							Segments: line,
							Indent:   indent,
						})
					}
				} else {
					ctx.result.Lines = append(ctx.result.Lines, StyledLine{
						Segments: segments,
						Indent:   ctx.indent,
					})
				}
			} else if textBlock, ok := child.(*ast.TextBlock); ok {
				// TextBlock is used for simple list items
				textSegments := mr.extractInlineSegments(textBlock, ctx)
				segments = append(segments, textSegments...)

				// Wrap the combined marker + content if needed
				if mr.MaxWidth > 0 {
					lines := mr.wrapSegments(segments, mr.MaxWidth-ctx.indent)
					for i, line := range lines {
						indent := ctx.indent
						if i > 0 {
							// Continuation lines get extra indent to align with text after marker
							indent += markerWidth
						}
						ctx.result.Lines = append(ctx.result.Lines, StyledLine{
							Segments: line,
							Indent:   indent,
						})
					}
				} else {
					ctx.result.Lines = append(ctx.result.Lines, StyledLine{
						Segments: segments,
						Indent:   ctx.indent,
					})
				}
			} else {
				ctx.result.Lines = append(ctx.result.Lines, StyledLine{
					Segments: segments,
					Indent:   ctx.indent,
				})
				ctx.indent += markerWidth
				mr.renderNode(child, ctx)
				ctx.indent = savedIndent
			}
			firstChild = false
		} else {
			// Subsequent children are indented
			ctx.indent = savedIndent + markerWidth
			mr.renderNode(child, ctx)
			ctx.indent = savedIndent
		}
	}
}

func (mr *MarkdownRenderer) renderCodeBlock(node *ast.CodeBlock, ctx *renderContext) {
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		text := string(line.Value(ctx.source))

		ctx.result.Lines = append(ctx.result.Lines, StyledLine{
			Segments: []StyledSegment{{
				Text:  text,
				Style: mr.Theme.CodeBlockStyle,
			}},
			Indent: ctx.indent + mr.TabWidth,
		})
	}

	// Add blank line after code block
	ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
}

func (mr *MarkdownRenderer) renderFencedCodeBlock(node *ast.FencedCodeBlock, ctx *renderContext) {
	// Get language for syntax highlighting
	language := string(node.Language(ctx.source))

	// Collect code lines
	var code strings.Builder
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code.Write(line.Value(ctx.source))
	}

	// Try syntax highlighting if language is specified
	if language != "" {
		if highlighted := mr.highlightCode(code.String(), language); highlighted != nil {
			for _, line := range highlighted {
				ctx.result.Lines = append(ctx.result.Lines, StyledLine{
					Segments: line,
					Indent:   ctx.indent + mr.TabWidth,
				})
			}
			ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
			return
		}
	}

	// Fallback to plain code block styling
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		text := string(line.Value(ctx.source))

		ctx.result.Lines = append(ctx.result.Lines, StyledLine{
			Segments: []StyledSegment{{
				Text:  text,
				Style: mr.Theme.CodeBlockStyle,
			}},
			Indent: ctx.indent + mr.TabWidth,
		})
	}

	// Add blank line after code block
	ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
}

func (mr *MarkdownRenderer) renderBlockquote(node *ast.Blockquote, ctx *renderContext) {
	savedIndent := ctx.indent
	ctx.indent += 2

	mr.renderChildren(node, ctx)

	// Apply blockquote style to all lines in the blockquote
	// Note: This is simplified - in practice we'd track which lines belong to the blockquote

	ctx.indent = savedIndent
}

func (mr *MarkdownRenderer) renderHorizontalRule(ctx *renderContext) {
	width := mr.MaxWidth
	if width == 0 {
		width = 80
	}

	rule := strings.Repeat(mr.Theme.HorizontalRuleChar, width-ctx.indent)

	ctx.result.Lines = append(ctx.result.Lines, StyledLine{
		Segments: []StyledSegment{{
			Text:  rule,
			Style: mr.Theme.HorizontalRuleStyle,
		}},
		Indent: ctx.indent,
	})

	// Add blank line after rule
	ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
}

func (mr *MarkdownRenderer) extractInlineSegments(node ast.Node, ctx *renderContext) []StyledSegment {
	var segments []StyledSegment
	baseStyle := NewStyle()

	mr.extractInlineSegmentsRec(node, ctx, baseStyle, &segments)

	// Merge adjacent segments with the same style and no hyperlinks.
	// This fixes cases where goldmark splits text at underscore boundaries
	// (e.g., "read_file" becomes ["read_", "file"]) even when no emphasis
	// is applied.
	segments = mergeAdjacentSegments(segments)

	return segments
}

// mergeAdjacentSegments combines consecutive segments that have the same
// style and no hyperlinks. This is needed because goldmark's emphasis parser
// splits text at potential delimiter boundaries (like underscores) even when
// no emphasis is applied.
func mergeAdjacentSegments(segments []StyledSegment) []StyledSegment {
	if len(segments) <= 1 {
		return segments
	}

	result := make([]StyledSegment, 0, len(segments))
	current := segments[0]

	for i := 1; i < len(segments); i++ {
		next := segments[i]

		// Merge if same style and neither has a hyperlink
		if current.Style == next.Style && current.Hyperlink == nil && next.Hyperlink == nil {
			current.Text += next.Text
		} else {
			result = append(result, current)
			current = next
		}
	}
	result = append(result, current)

	return result
}

func (mr *MarkdownRenderer) extractInlineSegmentsRec(node ast.Node, ctx *renderContext, currentStyle Style, segments *[]StyledSegment) {
	switch n := node.(type) {
	case *ast.Text:
		text := string(n.Segment.Value(ctx.source))
		*segments = append(*segments, StyledSegment{
			Text:  text,
			Style: currentStyle,
		})
		// Check if this text node ends with a hard line break
		if n.HardLineBreak() {
			*segments = append(*segments, StyledSegment{
				Text:  "\n",
				Style: currentStyle,
			})
		}

	case *ast.String:
		*segments = append(*segments, StyledSegment{
			Text:  string(n.Value),
			Style: currentStyle,
		})

	case *ast.CodeSpan:
		// CodeSpan contains Text children - extract their content
		var codeText strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				codeText.Write(textNode.Segment.Value(ctx.source))
			}
		}
		style := mr.mergeStyles(currentStyle, mr.Theme.CodeStyle)
		*segments = append(*segments, StyledSegment{
			Text:  codeText.String(),
			Style: style,
		})

	case *ast.Emphasis:
		style := currentStyle
		switch n.Level {
		case 1:
			style = mr.mergeStyles(style, mr.Theme.ItalicStyle)
		case 2:
			style = mr.mergeStyles(style, mr.Theme.BoldStyle)
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			mr.extractInlineSegmentsRec(child, ctx, style, segments)
		}

	case *ast.Link:
		dest := string(n.Destination)

		// Extract link text
		var linkText strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if text, ok := child.(*ast.Text); ok {
				linkText.Write(text.Segment.Value(ctx.source))
			}
		}

		style := mr.mergeStyles(currentStyle, mr.Theme.LinkStyle)

		*segments = append(*segments, StyledSegment{
			Text:  linkText.String(),
			Style: style,
			Hyperlink: &Hyperlink{
				URL:   dest,
				Text:  linkText.String(),
				Style: style,
			},
		})

	default:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			mr.extractInlineSegmentsRec(child, ctx, currentStyle, segments)
		}
	}
}

func (mr *MarkdownRenderer) mergeStyles(base Style, overlay Style) Style {
	result := base

	if overlay.Bold {
		result.Bold = true
	}
	if overlay.Italic {
		result.Italic = true
	}
	if overlay.Underline {
		result.Underline = true
	}
	if overlay.Strikethrough {
		result.Strikethrough = true
	}
	if overlay.Foreground != ColorDefault {
		result.Foreground = overlay.Foreground
	}
	if overlay.Background != ColorDefault {
		result.Background = overlay.Background
	}
	if overlay.FgRGB != nil {
		result.FgRGB = overlay.FgRGB
	}
	if overlay.BgRGB != nil {
		result.BgRGB = overlay.BgRGB
	}

	return result
}

// isPunctuation returns true if the string consists only of punctuation
// that should not have a leading space
func isPunctuation(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch r {
		case '.', ',', '!', '?', ':', ';', ')', ']', '}', '"', '\'':
			// These are trailing punctuation that shouldn't have a leading space
		default:
			return false
		}
	}
	return true
}

func (mr *MarkdownRenderer) wrapSegments(segments []StyledSegment, maxWidth int) [][]StyledSegment {
	if maxWidth <= 0 {
		return [][]StyledSegment{segments}
	}

	var lines [][]StyledSegment
	var currentLine []StyledSegment
	currentWidth := 0

	for _, seg := range segments {
		// Check for hard line breaks (newline characters)
		if strings.Contains(seg.Text, "\n") {
			// Split the segment at newlines
			parts := strings.Split(seg.Text, "\n")
			for i, part := range parts {
				// Process the text before the newline
				words := strings.Fields(part)
				for _, word := range words {
					wordWidth := runewidth.StringWidth(word)
					spaceWidth := 1

					// Don't count space for punctuation since we won't add one
					spaceNeeded := spaceWidth
					if isPunctuation(word) {
						spaceNeeded = 0
					}

					if currentWidth+wordWidth+spaceNeeded > maxWidth && len(currentLine) > 0 {
						lines = append(lines, currentLine)
						currentLine = nil
						currentWidth = 0
					}

					// Add space before word if not at start of line (but not before punctuation)
					needsSpace := len(currentLine) > 0 && !isPunctuation(word)
					if needsSpace {
						currentLine = append(currentLine, StyledSegment{
							Text:  " ",
							Style: seg.Style,
						})
						currentWidth++
					}

					currentLine = append(currentLine, StyledSegment{
						Text:      word,
						Style:     seg.Style,
						Hyperlink: seg.Hyperlink,
					})
					currentWidth += wordWidth
				}

				// If this isn't the last part, we hit a newline - force a line break
				if i < len(parts)-1 {
					if len(currentLine) > 0 {
						lines = append(lines, currentLine)
					} else {
						// Empty line from consecutive newlines
						lines = append(lines, []StyledSegment{})
					}
					currentLine = nil
					currentWidth = 0
				}
			}
			continue
		}

		// Normal text without newlines
		words := strings.Fields(seg.Text)

		for _, word := range words {
			wordWidth := runewidth.StringWidth(word)
			spaceWidth := 1

			// Check if adding this word would exceed the limit
			// Don't count space for punctuation since we won't add one
			spaceNeeded := spaceWidth
			if isPunctuation(word) {
				spaceNeeded = 0
			}
			if currentWidth+wordWidth+spaceNeeded > maxWidth && len(currentLine) > 0 {
				// Start a new line
				lines = append(lines, currentLine)
				currentLine = nil
				currentWidth = 0
			}

			// Add space before word if not at start of line (but not before punctuation)
			needsSpace := len(currentLine) > 0 && !isPunctuation(word)
			if needsSpace {
				currentLine = append(currentLine, StyledSegment{
					Text:  " ",
					Style: seg.Style,
				})
				currentWidth++
			}

			// Add the word
			currentLine = append(currentLine, StyledSegment{
				Text:      word,
				Style:     seg.Style,
				Hyperlink: seg.Hyperlink,
			})
			currentWidth += wordWidth
		}
	}

	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

func (mr *MarkdownRenderer) highlightCode(code, language string) [][]StyledSegment {
	// Get lexer for the language
	lexer := lexers.Get(language)
	if lexer == nil {
		// If we can't find a lexer, return unstyled code
		return nil
	}

	// Get chroma style
	chromaStyle := styles.Get(mr.Theme.SyntaxTheme)
	if chromaStyle == nil {
		chromaStyle = styles.Fallback
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return nil
	}

	// Convert tokens to our styled segments
	var result [][]StyledSegment
	var currentLine []StyledSegment

	for _, token := range iterator.Tokens() {
		tokenText := token.Value

		// Get the style entry for this token type
		styleEntry := chromaStyle.Get(token.Type)

		// Convert chroma style to our Style
		style := NewStyle()

		// Handle foreground color
		if styleEntry.Colour.IsSet() {
			r, g, b := styleEntry.Colour.Red(), styleEntry.Colour.Green(), styleEntry.Colour.Blue()
			style = style.WithFgRGB(RGB{R: r, G: g, B: b})
		}

		// Note: We intentionally do NOT use background colors from the syntax theme
		// to allow the terminal's default background to show through

		// Handle text attributes
		if styleEntry.Bold == 1 {
			style = style.WithBold()
		}
		if styleEntry.Italic == 1 {
			style = style.WithItalic()
		}
		if styleEntry.Underline == 1 {
			style = style.WithUnderline()
		}

		// Split token text by newlines to handle multi-line tokens
		lines := strings.Split(tokenText, "\n")
		for i, line := range lines {
			if i > 0 {
				// New line - append current line to result and start new one
				result = append(result, currentLine)
				currentLine = []StyledSegment{}
			}

			if line != "" {
				currentLine = append(currentLine, StyledSegment{
					Text:  line,
					Style: style,
				})
			}
		}
	}

	// Append final line if not empty
	if len(currentLine) > 0 {
		result = append(result, currentLine)
	}

	return result
}
