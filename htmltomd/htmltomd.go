// Package htmltomd converts HTML to Markdown format with graceful handling
// of malformed input.
//
// This package is ideal for preparing web content for LLM consumption or
// generating clean documentation from HTML sources. It uses golang.org/x/net/html
// for robust parsing and handles malformed HTML gracefully, prioritizing best-effort
// conversion over strict correctness.
//
// # Features
//
// - Robust parsing of both well-formed and malformed HTML
// - Customizable output styles (ATX vs Setext headings, inline vs referenced links)
// - Support for all common HTML elements including tables, lists, code blocks
// - Automatic skipping of script, style, and other non-content tags
// - Configurable tag filtering to exclude unwanted content
// - Proper handling of Unicode, emoji, and wide characters
//
// # Basic Usage
//
// Convert HTML to Markdown using default options:
//
//	html := "<h1>Getting Started</h1><p>Welcome to our <strong>API</strong>.</p>"
//	md := htmltomd.Convert(html)
//	// Output: "# Getting Started\n\nWelcome to our **API**."
//
// # Custom Options
//
// Control the conversion style with Options:
//
//	opts := &htmltomd.Options{
//	    LinkStyle:      htmltomd.LinkStyleReferenced,  // Use [text][1] style links
//	    HeadingStyle:   htmltomd.HeadingStyleSetext,   // Use underlined h1/h2
//	    CodeBlockStyle: htmltomd.CodeBlockStyleIndented, // Use 4-space indentation
//	    BulletChar:     "*",                            // Use * for bullets
//	    SkipTags:       []string{"nav", "footer"},      // Skip these tags
//	}
//	md := htmltomd.ConvertWithOptions(html, opts)
//
// # Supported HTML Elements
//
// The converter handles a wide range of HTML elements:
//
//   - Headings: h1-h6
//   - Text formatting: strong, b, em, i, del, s, strike, code
//   - Links: a (preserves href and title attributes)
//   - Images: img (preserves src and alt attributes)
//   - Lists: ul, ol, li with full nesting support
//   - Blockquotes: blockquote with nesting
//   - Code blocks: pre, code with language detection from class attributes
//   - Tables: table, thead, tbody, tr, th, td
//   - Horizontal rules: hr
//   - Line breaks: br
//
// # Error Handling
//
// The converter never returns errors. Instead, it gracefully handles invalid
// HTML by using best-effort parsing. If HTML is completely unparseable, it
// falls back to stripping tags and normalizing whitespace.
package htmltomd

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Precompiled regexes for efficiency
var (
	whitespaceRegex   = regexp.MustCompile(`\s+`)
	multiNewlineRegex = regexp.MustCompile(`\n{3,}`)
	tagStripRegex     = regexp.MustCompile(`<[^>]*>`)
	listItemPattern   = regexp.MustCompile(`^[-*+][ ]|^\d+\.[ ]`)
)

// LinkStyle defines how links are rendered in Markdown.
type LinkStyle int

const (
	// LinkStyleInline renders links as [text](url).
	LinkStyleInline LinkStyle = iota
	// LinkStyleReferenced renders links as [text][n] with references at the end.
	LinkStyleReferenced
)

// HeadingStyle defines how headings are rendered in Markdown.
type HeadingStyle int

const (
	// HeadingStyleATX renders headings with # prefix (e.g., # Heading).
	HeadingStyleATX HeadingStyle = iota
	// HeadingStyleSetext renders h1/h2 with underlines (=== or ---).
	HeadingStyleSetext
)

// CodeBlockStyle defines how code blocks are rendered in Markdown.
type CodeBlockStyle int

const (
	// CodeBlockStyleFenced renders code blocks with ``` fences.
	CodeBlockStyleFenced CodeBlockStyle = iota
	// CodeBlockStyleIndented renders code blocks with 4-space indentation.
	CodeBlockStyleIndented
)

// Options configures the HTML to Markdown conversion process.
//
// All fields are optional and will use sensible defaults if not specified.
// The zero value of Options produces default behavior equivalent to
// DefaultOptions().
//
// Example customization:
//
//	opts := &htmltomd.Options{
//	    LinkStyle:      htmltomd.LinkStyleReferenced,  // Collect links at end
//	    HeadingStyle:   htmltomd.HeadingStyleSetext,   // Underline h1 and h2
//	    CodeBlockStyle: htmltomd.CodeBlockStyleIndented, // 4-space indented code
//	    BulletChar:     "*",                            // Use asterisks
//	    SkipTags:       []string{"nav", "footer"},      // Ignore navigation
//	}
type Options struct {
	// LinkStyle controls how links are rendered in the output.
	// Default is LinkStyleInline, which renders links as [text](url).
	// Use LinkStyleReferenced to collect link references at the end.
	LinkStyle LinkStyle

	// HeadingStyle controls how headings are rendered in the output.
	// Default is HeadingStyleATX, which uses # prefix (e.g., ## Heading).
	// Use HeadingStyleSetext for h1 and h2 with underlines (=== or ---).
	HeadingStyle HeadingStyle

	// CodeBlockStyle controls how code blocks are rendered in the output.
	// Default is CodeBlockStyleFenced, which uses ``` fences.
	// Use CodeBlockStyleIndented for 4-space indented code blocks.
	CodeBlockStyle CodeBlockStyle

	// BulletChar is the character used for unordered list items.
	// Valid values are "-", "*", or "+". Default is "-".
	BulletChar string

	// SkipTags is a list of HTML tag names to skip entirely during conversion.
	// Content within these tags will not appear in the output.
	// Tag names are case-insensitive. The tags "script", "style", "head",
	// and "noscript" are always skipped regardless of this setting.
	//
	// Example: []string{"nav", "footer", "aside"}
	SkipTags []string
}

// DefaultOptions returns the default conversion options.
//
// The defaults are:
//   - LinkStyle: LinkStyleInline (renders as [text](url))
//   - HeadingStyle: HeadingStyleATX (renders as # Heading)
//   - CodeBlockStyle: CodeBlockStyleFenced (renders with ``` fences)
//   - BulletChar: "-" (unordered lists use hyphens)
//   - SkipTags: nil (no additional tags skipped beyond defaults)
//
// Note that script, style, head, and noscript tags are always skipped
// regardless of options.
func DefaultOptions() *Options {
	return &Options{
		LinkStyle:      LinkStyleInline,
		HeadingStyle:   HeadingStyleATX,
		CodeBlockStyle: CodeBlockStyleFenced,
		BulletChar:     "-",
		SkipTags:       nil,
	}
}

// Convert converts HTML to Markdown using default options.
//
// This is a convenience function equivalent to ConvertWithOptions(htmlContent, DefaultOptions()).
// It handles malformed HTML gracefully and never returns an error.
//
// Example:
//
//	html := "<h1>Title</h1><p>Content with <strong>bold</strong> text.</p>"
//	md := htmltomd.Convert(html)
//	// Returns: "# Title\n\nContent with **bold** text."
func Convert(htmlContent string) string {
	return ConvertWithOptions(htmlContent, DefaultOptions())
}

// ConvertWithOptions converts HTML to Markdown with custom options.
//
// This function provides full control over the conversion process through the
// Options parameter. If opts is nil, default options are used.
//
// The conversion handles malformed HTML gracefully, using golang.org/x/net/html's
// error recovery. If HTML parsing fails completely, the function falls back to
// stripping tags and normalizing whitespace.
//
// Examples:
//
//	// Use referenced links instead of inline
//	opts := &htmltomd.Options{LinkStyle: htmltomd.LinkStyleReferenced}
//	md := htmltomd.ConvertWithOptions(html, opts)
//
//	// Skip navigation and footer elements
//	opts := &htmltomd.Options{SkipTags: []string{"nav", "footer"}}
//	md := htmltomd.ConvertWithOptions(html, opts)
//
//	// Use Setext-style headings (underlined)
//	opts := &htmltomd.Options{HeadingStyle: htmltomd.HeadingStyleSetext}
//	md := htmltomd.ConvertWithOptions(html, opts)
func ConvertWithOptions(htmlContent string, opts *Options) string {
	if opts == nil {
		opts = DefaultOptions()
	}
	if opts.BulletChar == "" {
		opts.BulletChar = "-"
	}

	// Build skip tags set
	skipTags := make(map[string]bool)
	for _, tag := range opts.SkipTags {
		skipTags[strings.ToLower(tag)] = true
	}
	// Always skip these
	skipTags["script"] = true
	skipTags["style"] = true
	skipTags["head"] = true
	skipTags["noscript"] = true

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// If parsing fails completely, return cleaned text
		return cleanText(htmlContent)
	}

	c := &converter{
		opts:       opts,
		skipTags:   skipTags,
		linkRefs:   make([]string, 0),
		linkRefMap: make(map[string]int),
	}

	result := c.convertNode(doc, &context{})
	result = c.postProcess(result)

	if opts.LinkStyle == LinkStyleReferenced && len(c.linkRefs) > 0 {
		result = result + "\n\n" + strings.Join(c.linkRefs, "\n")
	}

	return result
}

type context struct {
	listDepth      int
	listType       string // "ul" or "ol"
	listItemNum    int
	inPre          bool
	inBlockquote   int
	parentListType []string
}

func (ctx *context) copy() *context {
	newCtx := *ctx
	newCtx.parentListType = append([]string{}, ctx.parentListType...)
	return &newCtx
}

type converter struct {
	opts       *Options
	skipTags   map[string]bool
	linkRefs   []string
	linkRefMap map[string]int
}

func (c *converter) convertNode(n *html.Node, ctx *context) string {
	switch n.Type {
	case html.TextNode:
		return c.handleText(n, ctx)
	case html.ElementNode:
		return c.handleElement(n, ctx)
	case html.DocumentNode:
		return c.convertChildren(n, ctx)
	default:
		return c.convertChildren(n, ctx)
	}
}

func (c *converter) handleText(n *html.Node, ctx *context) string {
	text := n.Data
	if ctx.inPre {
		return text
	}
	// Replace nbsp with regular space
	text = strings.ReplaceAll(text, "\u00a0", " ")
	// Normalize whitespace
	text = whitespaceRegex.ReplaceAllString(text, " ")
	return text
}

func (c *converter) handleElement(n *html.Node, ctx *context) string {
	tag := strings.ToLower(n.Data)

	// Skip ignored tags entirely
	if c.skipTags[tag] {
		return ""
	}

	switch tag {
	case "p":
		return c.handleParagraph(n, ctx)
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return c.handleHeading(n, ctx, tag)
	case "strong", "b":
		return c.handleBold(n, ctx)
	case "em", "i":
		return c.handleItalic(n, ctx)
	case "del", "s", "strike":
		return c.handleStrikethrough(n, ctx)
	case "code":
		return c.handleCode(n, ctx)
	case "a":
		return c.handleLink(n, ctx)
	case "img":
		return c.handleImage(n, ctx)
	case "ul":
		return c.handleUnorderedList(n, ctx)
	case "ol":
		return c.handleOrderedList(n, ctx)
	case "li":
		return c.handleListItem(n, ctx)
	case "blockquote":
		return c.handleBlockquote(n, ctx)
	case "pre":
		return c.handlePre(n, ctx)
	case "hr":
		return c.handleHorizontalRule(n, ctx)
	case "br":
		return "\n"
	case "table":
		return c.handleTable(n, ctx)
	case "div", "article", "section", "main", "aside", "header", "footer", "nav", "figure", "figcaption", "details", "summary":
		return c.handleBlockContainer(n, ctx)
	case "span", "mark", "time", "abbr", "cite", "address", "label":
		return c.convertChildren(n, ctx)
	default:
		return c.convertChildren(n, ctx)
	}
}

func (c *converter) convertChildren(n *html.Node, ctx *context) string {
	var parts []string
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		part := c.convertNode(child, ctx)
		parts = append(parts, part)
	}
	return strings.Join(parts, "")
}

func (c *converter) handleBlockContainer(n *html.Node, ctx *context) string {
	content := strings.TrimSpace(c.convertChildren(n, ctx))
	if content == "" {
		return ""
	}
	return "\n\n" + content + "\n\n"
}

func (c *converter) handleParagraph(n *html.Node, ctx *context) string {
	content := strings.TrimSpace(c.convertChildren(n, ctx))
	if content == "" {
		return ""
	}
	if ctx.inBlockquote > 0 {
		return content
	}
	return "\n\n" + content + "\n\n"
}

func (c *converter) handleHeading(n *html.Node, ctx *context, tag string) string {
	content := strings.TrimSpace(c.convertChildren(n, ctx))
	if content == "" {
		return ""
	}

	level := int(tag[1] - '0')

	var result string
	if c.opts.HeadingStyle == HeadingStyleSetext && level <= 2 {
		char := "="
		if level == 2 {
			char = "-"
		}
		underline := strings.Repeat(char, len(content))
		result = content + "\n" + underline
	} else {
		prefix := strings.Repeat("#", level)
		result = prefix + " " + content
	}

	return "\n\n" + result + "\n\n"
}

func (c *converter) handleBold(n *html.Node, ctx *context) string {
	content := c.convertChildren(n, ctx)
	content = strings.TrimSpace(content)
	content = whitespaceRegex.ReplaceAllString(content, " ")
	if content == "" {
		return ""
	}
	return "**" + content + "**"
}

func (c *converter) handleItalic(n *html.Node, ctx *context) string {
	content := c.convertChildren(n, ctx)
	content = strings.TrimSpace(content)
	content = whitespaceRegex.ReplaceAllString(content, " ")
	if content == "" {
		return ""
	}
	return "*" + content + "*"
}

func (c *converter) handleStrikethrough(n *html.Node, ctx *context) string {
	content := c.convertChildren(n, ctx)
	content = strings.TrimSpace(content)
	content = whitespaceRegex.ReplaceAllString(content, " ")
	if content == "" {
		return ""
	}
	return "~~" + content + "~~"
}

func (c *converter) handleCode(n *html.Node, ctx *context) string {
	if ctx.inPre {
		// Inside pre, just return content
		return c.convertChildren(n, ctx)
	}
	content := c.convertChildren(n, ctx)
	return "`" + content + "`"
}

func (c *converter) handleLink(n *html.Node, ctx *context) string {
	href := getAttr(n, "href")
	title := getAttr(n, "title")
	content := c.convertChildren(n, ctx)

	// Normalize whitespace in link text (collapse newlines from nested block elements)
	content = strings.TrimSpace(content)
	content = whitespaceRegex.ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)

	// If no href or empty content, just return content (or nothing)
	if href == "" {
		return content
	}
	if content == "" {
		return ""
	}

	if c.opts.LinkStyle == LinkStyleReferenced {
		refNum, exists := c.linkRefMap[href]
		if !exists {
			refNum = len(c.linkRefs) + 1
			c.linkRefMap[href] = refNum
			if title != "" {
				c.linkRefs = append(c.linkRefs, fmt.Sprintf("[%d]: %s \"%s\"", refNum, href, title))
			} else {
				c.linkRefs = append(c.linkRefs, fmt.Sprintf("[%d]: %s", refNum, href))
			}
		}
		return fmt.Sprintf("[%s][%d]", content, refNum)
	}

	if title != "" {
		return fmt.Sprintf("[%s](%s \"%s\")", content, href, title)
	}
	return fmt.Sprintf("[%s](%s)", content, href)
}

func (c *converter) handleImage(n *html.Node, ctx *context) string {
	src := getAttr(n, "src")
	alt := getAttr(n, "alt")
	return fmt.Sprintf("![%s](%s)", alt, src)
}

func (c *converter) handleUnorderedList(n *html.Node, ctx *context) string {
	newCtx := ctx.copy()
	newCtx.listDepth++
	newCtx.listType = "ul"
	newCtx.listItemNum = 0
	newCtx.parentListType = append(newCtx.parentListType, "ul")

	content := c.convertListChildren(n, newCtx)
	if ctx.listDepth == 0 {
		return "\n\n" + content + "\n\n"
	}
	return "\n" + content
}

func (c *converter) handleOrderedList(n *html.Node, ctx *context) string {
	newCtx := ctx.copy()
	newCtx.listDepth++
	newCtx.listType = "ol"
	newCtx.listItemNum = 0
	newCtx.parentListType = append(newCtx.parentListType, "ol")

	content := c.convertListChildren(n, newCtx)
	if ctx.listDepth == 0 {
		return "\n\n" + content + "\n\n"
	}
	return "\n" + content
}

func (c *converter) convertListChildren(n *html.Node, ctx *context) string {
	var items []string
	var hasListItems bool

	// First pass: check for proper <li> elements
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && strings.ToLower(child.Data) == "li" {
			hasListItems = true
			ctx.listItemNum++
			item := c.handleListItem(child, ctx)
			if item != "" {
				items = append(items, item)
			}
		}
	}

	// If no <li> elements found, treat non-li children as implicit list items
	// (handles malformed HTML like <ul><a>...</a></ul>)
	if !hasListItems {
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.ElementNode {
				tag := strings.ToLower(child.Data)
				// Skip style/script tags
				if c.skipTags[tag] {
					continue
				}
				ctx.listItemNum++
				content := strings.TrimSpace(c.convertNode(child, ctx))
				if content != "" {
					indent := strings.Repeat("  ", ctx.listDepth-1)
					var prefix string
					if ctx.listType == "ol" {
						prefix = fmt.Sprintf("%d. ", ctx.listItemNum)
					} else {
						prefix = c.opts.BulletChar + " "
					}
					items = append(items, indent+prefix+content)
				}
			}
		}
	}

	return strings.Join(items, "\n")
}

func (c *converter) handleListItem(n *html.Node, ctx *context) string {
	// Handle orphan <li> elements (not inside ul/ol) gracefully
	depth := ctx.listDepth
	if depth < 1 {
		depth = 1
	}
	indent := strings.Repeat("  ", depth-1)

	var prefix string
	if ctx.listType == "ol" {
		prefix = fmt.Sprintf("%d. ", ctx.listItemNum)
	} else {
		prefix = c.opts.BulletChar + " "
	}

	// Process children, handling nested lists specially
	var textParts []string
	var nestedLists []string

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			childTag := strings.ToLower(child.Data)
			if childTag == "ul" || childTag == "ol" {
				nestedLists = append(nestedLists, c.convertNode(child, ctx))
			} else {
				textParts = append(textParts, c.convertNode(child, ctx))
			}
		} else {
			textParts = append(textParts, c.convertNode(child, ctx))
		}
	}

	content := strings.TrimSpace(strings.Join(textParts, ""))

	// Skip empty list items that have no nested lists
	if content == "" && len(nestedLists) == 0 {
		return ""
	}

	result := indent + prefix + content

	for _, nested := range nestedLists {
		result += nested
	}

	return result
}

func (c *converter) handleBlockquote(n *html.Node, ctx *context) string {
	newCtx := ctx.copy()
	newCtx.inBlockquote++

	// Process children, collecting paragraph-like content
	var blockParts []string
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			tag := strings.ToLower(child.Data)
			if tag == "p" {
				content := strings.TrimSpace(c.convertChildren(child, newCtx))
				if content != "" {
					blockParts = append(blockParts, content)
				}
			} else if tag == "blockquote" {
				// Nested blockquote - process it but don't add our prefix
				innerContent := c.handleBlockquoteInner(child, newCtx)
				if innerContent != "" {
					blockParts = append(blockParts, innerContent)
				}
			} else {
				part := c.convertNode(child, newCtx)
				part = strings.TrimSpace(part)
				if part != "" {
					blockParts = append(blockParts, part)
				}
			}
		} else {
			part := c.convertNode(child, newCtx)
			part = strings.TrimSpace(part)
			if part != "" {
				blockParts = append(blockParts, part)
			}
		}
	}

	if len(blockParts) == 0 {
		return ""
	}

	// Join parts with blank quoted lines between them
	prefix := "> "
	var resultLines []string
	for i, part := range blockParts {
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			resultLines = append(resultLines, prefix+line)
		}
		// Add blank quote line between parts (but not after the last)
		if i < len(blockParts)-1 {
			resultLines = append(resultLines, ">")
		}
	}

	result := strings.Join(resultLines, "\n")
	if ctx.inBlockquote == 0 {
		return "\n\n" + result + "\n\n"
	}
	return result
}

func (c *converter) handleBlockquoteInner(n *html.Node, ctx *context) string {
	// For nested blockquotes, just get the content with one level of ">" prefix
	newCtx := ctx.copy()
	newCtx.inBlockquote++

	var blockParts []string
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			tag := strings.ToLower(child.Data)
			if tag == "p" {
				content := strings.TrimSpace(c.convertChildren(child, newCtx))
				if content != "" {
					blockParts = append(blockParts, content)
				}
			} else if tag == "blockquote" {
				innerContent := c.handleBlockquoteInner(child, newCtx)
				if innerContent != "" {
					blockParts = append(blockParts, innerContent)
				}
			} else {
				part := c.convertNode(child, newCtx)
				part = strings.TrimSpace(part)
				if part != "" {
					blockParts = append(blockParts, part)
				}
			}
		} else {
			part := c.convertNode(child, newCtx)
			part = strings.TrimSpace(part)
			if part != "" {
				blockParts = append(blockParts, part)
			}
		}
	}

	if len(blockParts) == 0 {
		return ""
	}

	// For inner blockquotes, prefix each line with ">"
	prefix := "> "
	var resultLines []string
	for i, part := range blockParts {
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			resultLines = append(resultLines, prefix+line)
		}
		if i < len(blockParts)-1 {
			resultLines = append(resultLines, ">")
		}
	}

	return strings.Join(resultLines, "\n")
}

func (c *converter) handlePre(n *html.Node, ctx *context) string {
	newCtx := ctx.copy()
	newCtx.inPre = true

	// Check for language class
	lang := ""
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && strings.ToLower(child.Data) == "code" {
			class := getAttr(child, "class")
			if strings.HasPrefix(class, "language-") {
				lang = strings.TrimPrefix(class, "language-")
			}
		}
	}

	content := c.convertChildren(n, newCtx)
	// Only trim leading/trailing newlines, preserving internal whitespace
	content = strings.Trim(content, "\n")

	var result string
	if c.opts.CodeBlockStyle == CodeBlockStyleIndented {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = "    " + line
		}
		result = strings.Join(lines, "\n")
	} else {
		result = "```" + lang + "\n" + content + "\n```"
	}

	return "\n\n" + result + "\n\n"
}

func (c *converter) handleHorizontalRule(n *html.Node, ctx *context) string {
	return "\n\n---\n\n"
}

func (c *converter) handleTable(n *html.Node, ctx *context) string {
	var rows [][]string
	hasHeader := false

	// Find tbody, thead, or direct tr children
	c.extractTableRows(n, &rows, &hasHeader)

	if len(rows) == 0 {
		return ""
	}

	// Determine column count
	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}

	// If no columns, return empty
	if colCount == 0 {
		return ""
	}

	// Pad rows to same column count
	for i := range rows {
		for len(rows[i]) < colCount {
			rows[i] = append(rows[i], "")
		}
	}

	// Build markdown table
	var lines []string

	// Header row
	headerRow := rows[0]
	lines = append(lines, "| "+strings.Join(headerRow, " | ")+" |")

	// Separator
	sep := make([]string, colCount)
	for i := range sep {
		sep[i] = "---"
	}
	lines = append(lines, "| "+strings.Join(sep, " | ")+" |")

	// Data rows
	for _, row := range rows[1:] {
		lines = append(lines, "| "+strings.Join(row, " | ")+" |")
	}

	return "\n\n" + strings.Join(lines, "\n") + "\n\n"
}

func (c *converter) extractTableRows(n *html.Node, rows *[][]string, hasHeader *bool) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		tag := strings.ToLower(child.Data)
		switch tag {
		case "thead":
			*hasHeader = true
			c.extractTableRows(child, rows, hasHeader)
		case "tbody":
			c.extractTableRows(child, rows, hasHeader)
		case "tr":
			row := c.extractTableRow(child)
			*rows = append(*rows, row)
		}
	}
}

func (c *converter) extractTableRow(n *html.Node) []string {
	var cells []string
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		tag := strings.ToLower(child.Data)
		if tag == "th" || tag == "td" {
			content := strings.TrimSpace(c.convertChildren(child, &context{}))
			cells = append(cells, content)
		}
	}
	return cells
}

func (c *converter) postProcess(s string) string {
	// Remove lines that are only whitespace (but preserve indented lines like code blocks)
	lines := strings.Split(s, "\n")
	var cleanedLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			cleanedLines = append(cleanedLines, "")
		} else {
			// Trim trailing whitespace
			line = strings.TrimRight(line, " \t")
			// Trim leading whitespace unless it's intentional indentation:
			// - 4+ spaces for code blocks (indented style)
			// - 2+ spaces for list item nesting (must be followed by list marker)
			// - tab for code blocks
			trimmed := strings.TrimLeft(line, " ")
			leadingSpaces := len(line) - len(trimmed)
			keepIndent := false
			if strings.HasPrefix(line, "\t") {
				// Tab-indented code block
				keepIndent = true
			} else if leadingSpaces >= 4 {
				// 4+ space indented code block
				keepIndent = true
			} else if leadingSpaces >= 2 {
				// Check if this is a nested list item (indentation + list marker)
				if listItemPattern.MatchString(trimmed) {
					keepIndent = true
				}
			}
			if !keepIndent {
				line = trimmed
			}
			cleanedLines = append(cleanedLines, line)
		}
	}

	// Remove leading empty lines
	for len(cleanedLines) > 0 && cleanedLines[0] == "" {
		cleanedLines = cleanedLines[1:]
	}

	s = strings.Join(cleanedLines, "\n")

	// Clean up excessive newlines
	s = multiNewlineRegex.ReplaceAllString(s, "\n\n")

	// Trim trailing newlines
	s = strings.TrimRight(s, "\n")

	return s
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == key {
			return attr.Val
		}
	}
	return ""
}

func cleanText(s string) string {
	// Simple fallback for completely unparseable input
	s = tagStripRegex.ReplaceAllString(s, "")
	s = whitespaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
