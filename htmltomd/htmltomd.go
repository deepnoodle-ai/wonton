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
//   - Text formatting: strong, b, em, i, del, s, strike, q, sup, sub
//   - Inline code: code, tt, kbd, samp
//   - Links: a (preserves href and title attributes; card links that wrap
//     a heading link the heading)
//   - Images: img (preserves src, alt, and title; falls back to data-src
//     and srcset for lazy-loaded images)
//   - Lists: ul, ol, li with nesting, multi-paragraph items, li value, and
//     GFM task list checkboxes
//   - Definition lists: dl, dt, dd
//   - Blockquotes: blockquote with nesting and any block content
//   - Code blocks: pre, with language detection from class and data-lang
//     attributes
//   - Tables: table, caption, thead, tbody, tfoot, tr, th, td with colspan
//     and rowspan; layout tables are unwrapped into ordinary blocks
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
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/deepnoodle-ai/wonton/runewidth"
	"golang.org/x/net/html"
)

// Precompiled regexes for efficiency
var (
	whitespaceRegex  = regexp.MustCompile(`\s+`)
	inlineSpaceRegex = regexp.MustCompile(`[ \t\r\f\v]+`)
	multiSpaceRegex  = regexp.MustCompile(` {2,}`)
	tagStripRegex    = regexp.MustCompile(`<[^>]*>`)
)

// alwaysSkipTags are elements whose content is never prose.
var alwaysSkipTags = []string{
	"script", "style", "head", "noscript", "template", "svg", "canvas", "select",
}

// Converted fragments carry a few control characters that normalize and
// finalize interpret and remove. They are stripped from input text and
// attributes, so they can only come from the converter itself.
//
// Structural prefixes (markIndent, markQuote) let nested blocks indent their
// lines without normalize mistaking that indentation for stray whitespace
// from the HTML source, which it trims.
const (
	markIndent    = '\x01' // one column of list or definition indentation
	markCodeStart = "\x02" // line opening a code block; lines up to markCodeEnd are verbatim
	markCodeEnd   = "\x03"
	markQuote     = '\x05' // a blockquote ">"
	markTight     = "\x06" // line that removes the blank lines before it
)

var quotePrefix = string([]byte{markQuote, markIndent})

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
	// "noscript", "template", "svg", "canvas", and "select" are always
	// skipped regardless of this setting, as are elements with the hidden
	// attribute or aria-hidden="true".
	//
	// Example: []string{"nav", "footer", "aside"}
	SkipTags []string

	// BaseURL, when set, resolves relative link and image URLs against it.
	// A <base href> in the document is honored too, resolved against
	// BaseURL when it is itself relative.
	BaseURL string
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
// Note that script, style, head, noscript, template, svg, canvas, and
// select tags are always skipped regardless of options.
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
	// Work on a copy so the caller's Options are never modified.
	o := *opts
	opts = &o
	if opts.BulletChar == "" {
		opts.BulletChar = "-"
	}

	skipTags := make(map[string]bool)
	for _, tag := range opts.SkipTags {
		skipTags[strings.ToLower(tag)] = true
	}
	for _, tag := range alwaysSkipTags {
		skipTags[tag] = true
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// If parsing fails completely, return cleaned text
		return cleanText(htmlContent)
	}

	c := &converter{
		opts:       opts,
		skipTags:   skipTags,
		linkRefMap: make(map[string]int),
		base:       resolveBase(opts.BaseURL, doc),
	}

	result := finalize(normalize(c.convertNode(doc, context{})))

	if opts.LinkStyle == LinkStyleReferenced && len(c.linkRefs) > 0 {
		result = result + "\n\n" + strings.Join(c.linkRefs, "\n")
	}

	return result
}

type context struct {
	listItems int  // depth of enclosing <li> elements
	linkText  bool // inside link text, where brackets are escaped
	emphasis  bool // inside em or strong, where asterisks are escaped
	bold      bool
	italic    bool
	strike    bool
	tableCell bool
	cardLink  *cardLink
}

// cardLink is the target of an <a> that wraps block content. The first
// heading or paragraph inside it claims the link.
type cardLink struct {
	href, title string
	used        bool
}

// claimCardLink returns the pending card link, if any, and marks it used.
// The context then renders link text.
func (ctx *context) claimCardLink() *cardLink {
	cl := ctx.cardLink
	if cl == nil || cl.used {
		return nil
	}
	cl.used = true
	ctx.cardLink = nil
	ctx.linkText = true
	return cl
}

type converter struct {
	opts       *Options
	skipTags   map[string]bool
	linkRefs   []string
	linkRefMap map[string]int
	base       *url.URL
}

func (c *converter) convertNode(n *html.Node, ctx context) string {
	switch n.Type {
	case html.TextNode:
		return c.handleText(n, ctx)
	case html.ElementNode:
		return c.handleElement(n, ctx)
	default:
		return c.convertChildren(n, ctx)
	}
}

func (c *converter) handleText(n *html.Node, ctx context) string {
	text := stripMarks(n.Data)
	text = strings.ReplaceAll(text, " ", " ")
	text = whitespaceRegex.ReplaceAllString(text, " ")
	if ctx.linkText {
		text = escapeBrackets(text)
	}
	if ctx.emphasis {
		text = strings.ReplaceAll(text, "*", `\*`)
	}
	return text
}

func (c *converter) skip(n *html.Node) bool {
	return c.skipTags[strings.ToLower(n.Data)] || hasAttr(n, "hidden") ||
		strings.EqualFold(strings.TrimSpace(getAttr(n, "aria-hidden")), "true")
}

func (c *converter) handleElement(n *html.Node, ctx context) string {
	if c.skip(n) {
		return ""
	}
	tag := strings.ToLower(n.Data)

	switch tag {
	case "p":
		return c.handleParagraph(n, ctx)
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return c.handleHeading(n, ctx, tag)
	case "strong", "b":
		return c.handleEmphasis(n, ctx, "**")
	case "em", "i":
		return c.handleEmphasis(n, ctx, "*")
	case "del", "s", "strike":
		return c.handleEmphasis(n, ctx, "~~")
	case "code", "tt", "kbd", "samp":
		return c.handleCode(n, ctx)
	case "a":
		return c.handleLink(n, ctx)
	case "img":
		return c.handleImage(n, ctx)
	case "ul", "ol":
		return c.handleList(n, ctx, tag == "ol")
	case "li":
		// An orphan <li> outside any list.
		if item := c.renderListItem(n, ctx, c.opts.BulletChar+" "); item != "" {
			return "\n\n" + item + "\n\n"
		}
		return ""
	case "dl":
		return c.handleDefinitionList(n, ctx)
	case "blockquote":
		return c.handleBlockquote(n, ctx)
	case "pre":
		return c.handlePre(n, ctx)
	case "hr":
		return "\n\n---\n\n"
	case "br":
		if ctx.tableCell {
			return "<br>"
		}
		return "\n"
	case "table":
		return c.handleTable(n, ctx)
	case "q":
		return `"` + c.convertChildren(n, ctx) + `"`
	case "sup", "sub":
		content := strings.TrimSpace(c.convertChildren(n, ctx))
		if content == "" {
			return ""
		}
		return "<" + tag + ">" + content + "</" + tag + ">"
	case "input":
		// Form controls carry no prose; task list checkboxes are rendered
		// by the list item.
		return ""
	case "div", "article", "section", "main", "aside", "header", "footer", "nav", "figure", "figcaption", "details", "summary":
		return c.handleBlockContainer(n, ctx)
	default:
		return c.convertChildren(n, ctx)
	}
}

func (c *converter) convertChildren(n *html.Node, ctx context) string {
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(c.convertNode(child, ctx))
	}
	return b.String()
}

// textContent returns the raw text of n with <br> as a newline, skipping
// ignored and hidden elements. It is used where Markdown syntax cannot
// appear: code blocks and code spans.
func (c *converter) textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			switch ch.Type {
			case html.TextNode:
				b.WriteString(stripMarks(ch.Data))
			case html.ElementNode:
				if c.skip(ch) {
					continue
				}
				if strings.EqualFold(ch.Data, "br") {
					b.WriteString("\n")
					continue
				}
				walk(ch)
			}
		}
	}
	walk(n)
	return b.String()
}

func (c *converter) handleBlockContainer(n *html.Node, ctx context) string {
	content := strings.TrimSpace(c.convertChildren(n, ctx))
	if content == "" {
		return ""
	}
	return "\n\n" + content + "\n\n"
}

func (c *converter) handleParagraph(n *html.Node, ctx context) string {
	cl := ctx.claimCardLink()
	content := strings.TrimSpace(c.convertChildren(n, ctx))
	if content == "" {
		if cl != nil {
			cl.used = false
		}
		return ""
	}
	if cl != nil {
		content = c.formatLink(collapseSpace(content), cl.href, cl.title)
	}
	return "\n\n" + content + "\n\n"
}

func (c *converter) handleHeading(n *html.Node, ctx context, tag string) string {
	cl := ctx.claimCardLink()
	// A Markdown heading is one line, so line breaks (<br>) become spaces.
	content := collapseSpace(c.convertChildren(n, ctx))
	if content == "" {
		if cl != nil {
			cl.used = false
		}
		return ""
	}
	if cl != nil {
		content = c.formatLink(content, cl.href, cl.title)
	}

	level := int(tag[1] - '0')

	var result string
	if c.opts.HeadingStyle == HeadingStyleSetext && level <= 2 {
		char := "="
		if level == 2 {
			char = "-"
		}
		underline := strings.Repeat(char, max(runewidth.StringWidth(content), 1))
		result = content + "\n" + underline
	} else {
		result = strings.Repeat("#", level) + " " + content
	}

	return "\n\n" + result + "\n\n"
}

// handleEmphasis renders bold, italic, and strikethrough. An element nested
// in one of the same kind adds no delimiters, since "**a **b** c**" would
// close the outer one early.
func (c *converter) handleEmphasis(n *html.Node, ctx context, delim string) string {
	var nested bool
	switch delim {
	case "**":
		nested, ctx.bold = ctx.bold, true
		ctx.emphasis = true
	case "*":
		nested, ctx.italic = ctx.italic, true
		ctx.emphasis = true
	default:
		nested, ctx.strike = ctx.strike, true
	}
	content := c.convertChildren(n, ctx)
	if nested {
		delim = ""
	}
	return wrapInline(n, content, delim)
}

// wrapInline wraps the inline content of n in a Markdown delimiter such as
// "**". Delimiters must touch the text, so edge whitespace moves outside
// them: "Hello<b> world</b>" becomes "Hello **world**". Each line is wrapped
// separately, so a <br> inside the element survives.
func wrapInline(n *html.Node, content, delim string) string {
	lead, text, trail := splitEdgeSpace(n, content, true)
	if text == "" {
		return lead + trail
	}
	// Emphasizing only escaped asterisks ("<em>*</em>") would read as a
	// run of delimiters, and block content cannot sit inside delimiters.
	if delim == "" || strings.ReplaceAll(text, `\*`, "") == "" || strings.Contains(text, markCodeStart) {
		return lead + text + trail
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if t := strings.TrimSpace(line); t != "" {
			lines[i] = delim + t + delim
		} else {
			lines[i] = ""
		}
	}
	return lead + strings.Join(lines, "\n") + trail
}

// splitEdgeSpace collapses whitespace in the inline content of n and splits
// off leading and trailing whitespace. An edge space is dropped when the
// adjacent text node already supplies one, so no double space appears. With
// keepNewlines, line breaks inside and at the edges are kept.
func splitEdgeSpace(n *html.Node, content string, keepNewlines bool) (lead, text, trail string) {
	if keepNewlines {
		content = inlineSpaceRegex.ReplaceAllString(content, " ")
	} else {
		content = whitespaceRegex.ReplaceAllString(content, " ")
	}
	text = strings.TrimSpace(content)
	prevSpace, nextSpace := textEndsWithSpace(n.PrevSibling), textStartsWithSpace(n.NextSibling)
	if text == "" {
		switch {
		case strings.Contains(content, "\n"):
			lead = "\n"
		case content != "" && !prevSpace && !nextSpace:
			// Whitespace-only content is one space, unless a neighbor has one.
			lead = " "
		}
		return lead, "", ""
	}
	leading := content[:len(content)-len(strings.TrimLeft(content, " \n"))]
	trailing := content[len(strings.TrimRight(content, " \n")):]
	lead = edgeSpace(leading, prevSpace)
	trail = edgeSpace(trailing, nextSpace)
	return lead, text, trail
}

func edgeSpace(ws string, neighborSpace bool) string {
	switch {
	case strings.Contains(ws, "\n"):
		return "\n"
	case ws != "" && !neighborSpace:
		return " "
	}
	return ""
}

func textEndsWithSpace(n *html.Node) bool {
	return n != nil && n.Type == html.TextNode && strings.TrimRight(n.Data, " \t\r\n ") != n.Data
}

func textStartsWithSpace(n *html.Node) bool {
	return n != nil && n.Type == html.TextNode && strings.TrimLeft(n.Data, " \t\r\n ") != n.Data
}

// handleCode renders inline code. Markup inside is reduced to its text,
// except that code wrapping a single link becomes a link around the code.
func (c *converter) handleCode(n *html.Node, ctx context) string {
	if a := soleLinkChild(n); a != nil {
		if href := c.linkHref(getAttr(a, "href")); href != "" {
			if code := codeSpan(whitespaceRegex.ReplaceAllString(c.textContent(a), " ")); code != "" {
				return c.formatLink(code, href, getAttr(a, "title"))
			}
			return ""
		}
	}
	text := whitespaceRegex.ReplaceAllString(strings.ReplaceAll(c.textContent(n), " ", " "), " ")
	if strings.TrimSpace(text) == "" {
		if text != "" {
			return " "
		}
		return ""
	}
	return codeSpan(text)
}

// codeSpan wraps text in backticks. Per CommonMark, the delimiter must be
// longer than any backtick run in the text, and text starting or ending with
// a backtick needs a space inside the delimiters (stripped on render).
func codeSpan(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	delim := strings.Repeat("`", longestBacktickRun(text)+1)
	if strings.HasPrefix(text, "`") || strings.HasSuffix(text, "`") {
		return delim + " " + text + " " + delim
	}
	return delim + text + delim
}

// soleLinkChild returns the <a> that is the only content of n, if any.
func soleLinkChild(n *html.Node) *html.Node {
	var link *html.Node
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		switch ch.Type {
		case html.TextNode:
			if strings.TrimSpace(ch.Data) != "" {
				return nil
			}
		case html.ElementNode:
			if link != nil || !strings.EqualFold(ch.Data, "a") {
				return nil
			}
			link = ch
		}
	}
	return link
}

// longestBacktickRun returns the length of the longest run of consecutive
// backticks in s.
func longestBacktickRun(s string) int {
	longest, run := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	return longest
}

var cardLinkTags = map[string]bool{"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "p": true}

func (c *converter) handleLink(n *html.Node, ctx context) string {
	rawHref := strings.TrimSpace(getAttr(n, "href"))
	href := c.linkHref(rawHref)
	title := getAttr(n, "title")
	ctx.cardLink = nil

	var content string
	switch {
	case href == "":
		content = c.convertChildren(n, ctx)
		// Without a target the content stands on its own, block
		// structure included.
		if strings.Contains(content, "\n") {
			return content
		}
	case hasDescendant(n, cardLinkTags):
		// A card link wraps a heading or paragraph: link that block's
		// text and keep the rest as ordinary blocks.
		cl := &cardLink{href: href, title: title}
		inner := ctx
		inner.cardLink = cl
		content = c.convertChildren(n, inner)
		if cl.used {
			return content
		}
	default:
		ctx.linkText = true
		content = c.convertChildren(n, ctx)
	}

	// Link text is one line. Leading and trailing spaces move outside the
	// brackets.
	lead, text, trail := splitEdgeSpace(n, content, false)
	if href == "" || text == "" {
		return lead + text + trail
	}
	// Permalink anchors next to headings ("¶", "§", "#") are noise.
	if strings.HasPrefix(rawHref, "#") && (text == "¶" || text == "§" || text == "#") {
		return lead + trail
	}
	return lead + c.formatLink(text, href, title) + trail
}

// formatLink renders a link in the configured style. text is already
// Markdown; href is an absolute or relative URL.
func (c *converter) formatLink(text, href, title string) string {
	dest := formatDestination(href)
	if c.opts.LinkStyle == LinkStyleReferenced {
		// Key on URL+title to avoid losing different titles for the same URL
		refKey := dest + "\x00" + title
		refNum, exists := c.linkRefMap[refKey]
		if !exists {
			refNum = len(c.linkRefs) + 1
			c.linkRefMap[refKey] = refNum
			if title != "" {
				c.linkRefs = append(c.linkRefs, fmt.Sprintf("[%d]: %s \"%s\"", refNum, dest, escapeTitle(title)))
			} else {
				c.linkRefs = append(c.linkRefs, fmt.Sprintf("[%d]: %s", refNum, dest))
			}
		}
		return fmt.Sprintf("[%s][%d]", text, refNum)
	}
	if title != "" {
		return fmt.Sprintf("[%s](%s \"%s\")", text, dest, escapeTitle(title))
	}
	return "[" + text + "](" + dest + ")"
}

// linkHref returns the URL an href should link to, or "" when it should not
// be linked: empty, or a script or data URL.
func (c *converter) linkHref(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || isScriptOrDataURL(raw) {
		return ""
	}
	return c.resolveURL(raw)
}

// isScriptOrDataURL reports whether u uses the javascript:, vbscript:, or
// data: scheme. Browsers ignore whitespace and control characters inside the
// scheme, so they are ignored here too.
func isScriptOrDataURL(u string) bool {
	s := strings.Map(func(r rune) rune {
		if r <= ' ' {
			return -1
		}
		return unicode.ToLower(r)
	}, u)
	return strings.HasPrefix(s, "javascript:") || strings.HasPrefix(s, "vbscript:") || strings.HasPrefix(s, "data:")
}

func (c *converter) resolveURL(ref string) string {
	ref = strings.TrimSpace(ref)
	if c.base == nil || ref == "" {
		return ref
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return c.base.ResolveReference(u).String()
}

// resolveBase returns the URL that relative references resolve against: the
// document's <base href> (itself resolved against baseURL), else baseURL.
func resolveBase(baseURL string, doc *html.Node) *url.URL {
	var base *url.URL
	if baseURL != "" {
		if u, err := url.Parse(strings.TrimSpace(baseURL)); err == nil && u.IsAbs() {
			base = u
		}
	}
	if href := findBaseHref(doc); href != "" {
		if u, err := url.Parse(href); err == nil {
			if base != nil {
				u = base.ResolveReference(u)
			}
			if u.IsAbs() {
				base = u
			}
		}
	}
	return base
}

func findBaseHref(n *html.Node) string {
	if n.Type == html.ElementNode && strings.EqualFold(n.Data, "base") {
		if href := strings.TrimSpace(getAttr(n, "href")); href != "" {
			return href
		}
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if href := findBaseHref(ch); href != "" {
			return href
		}
	}
	return ""
}

// formatDestination makes a URL safe as an inline link destination: spaces,
// control characters, angle brackets, and unbalanced parentheses are
// percent-encoded.
func formatDestination(u string) string {
	depth, balanced := 0, true
	for _, r := range u {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				balanced = false
			}
		}
	}
	balanced = balanced && depth == 0

	var b strings.Builder
	for _, r := range u {
		switch {
		case r <= ' ' || r == 0x7f:
			fmt.Fprintf(&b, "%%%02X", r)
		case r == '<':
			b.WriteString("%3C")
		case r == '>':
			b.WriteString("%3E")
		case r == '(' && !balanced:
			b.WriteString("%28")
		case r == ')' && !balanced:
			b.WriteString("%29")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeTitle makes a link or image title safe inside double quotes: quotes
// and backslashes are escaped and line breaks become spaces.
func escapeTitle(title string) string {
	title = strings.ReplaceAll(title, "\\", "\\\\")
	title = strings.ReplaceAll(title, "\"", "\\\"")
	return collapseSpace(title)
}

func escapeBrackets(s string) string {
	return strings.NewReplacer("[", `\[`, "]", `\]`).Replace(s)
}

func (c *converter) handleImage(n *html.Node, ctx context) string {
	// Alt text must stay on one line to remain part of the image syntax.
	alt := collapseSpace(getAttr(n, "alt"))
	src := imageSource(n)
	if src == "" {
		// Without a usable source, the alt text stands in for the image.
		if ctx.linkText {
			return escapeBrackets(alt)
		}
		return alt
	}
	dest := formatDestination(c.resolveURL(src))
	alt = escapeBrackets(alt)
	if title := getAttr(n, "title"); title != "" {
		return fmt.Sprintf("![%s](%s \"%s\")", alt, dest, escapeTitle(title))
	}
	return "![" + alt + "](" + dest + ")"
}

// imageSource returns the URL of an image, looking past the data: URI
// placeholders that lazy-loading scripts swap out.
func imageSource(n *html.Node) string {
	if src := strings.TrimSpace(getAttr(n, "src")); src != "" && !isScriptOrDataURL(src) {
		return src
	}
	for _, attr := range []string{"data-src", "data-lazy-src", "data-original"} {
		if src := strings.TrimSpace(getAttr(n, attr)); src != "" && !isScriptOrDataURL(src) {
			return src
		}
	}
	for _, attr := range []string{"srcset", "data-srcset"} {
		first, _, _ := strings.Cut(getAttr(n, attr), ",")
		if f := strings.Fields(first); len(f) > 0 && !isScriptOrDataURL(f[0]) {
			return f[0]
		}
	}
	return ""
}

func (c *converter) handleList(n *html.Node, ctx context, ordered bool) string {
	content := c.convertListChildren(n, ctx, ordered)
	if content == "" {
		return ""
	}
	if ctx.listItems > 0 {
		// A nested list hugs the item text before it.
		return "\n" + markTight + "\n" + content + "\n"
	}
	return "\n\n" + content + "\n\n"
}

func (c *converter) convertListChildren(n *html.Node, ctx context, ordered bool) string {
	num := 0
	// Honor the start attribute: CommonMark numbers an ordered list from the
	// first item's marker.
	if ordered {
		if v, err := strconv.Atoi(strings.TrimSpace(getAttr(n, "start"))); err == nil && v >= 0 {
			num = v - 1
		}
	}
	marker := func() string {
		if ordered {
			return strconv.Itoa(num) + ". "
		}
		return c.opts.BulletChar + " "
	}

	var items []string
	width := len(c.opts.BulletChar) + 1 // marker width of the previous item
	// attach nests a list under the previous item.
	attach := func(nested string) {
		nested = prefixLines(nested, indent(width), indent(width))
		if len(items) == 0 {
			items = append(items, nested)
		} else {
			items[len(items)-1] += "\n" + nested
		}
	}

	hasListItems := false
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		switch tag := strings.ToLower(child.Data); tag {
		case "li":
			hasListItems = true
			if c.skip(child) {
				continue
			}
			// An item holding only a list (<li><ul>...</ul></li>) nests
			// that list under the previous item, without an empty bullet.
			if onlyNestedLists(child) {
				inner := ctx
				inner.listItems++
				if nested := normalize(c.convertChildren(child, inner)); nested != "" {
					attach(nested)
				}
				continue
			}
			num++
			if ordered {
				if v, err := strconv.Atoi(strings.TrimSpace(getAttr(child, "value"))); err == nil && v >= 0 {
					num = v
				}
			}
			m := marker()
			if item := c.renderListItem(child, ctx, m); item != "" {
				items = append(items, item)
				width = len(m)
			}
		case "ul", "ol":
			// A list placed directly in a list (invalid but common in
			// WYSIWYG exports) nests under the preceding item.
			if nested := normalize(c.handleList(child, ctx, tag == "ol")); nested != "" {
				attach(nested)
			}
		}
	}

	// If no <li> elements found, treat other children as implicit list
	// items (handles malformed HTML like <ul><a>...</a></ul>).
	if !hasListItems {
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode || c.skip(child) {
				continue
			}
			num++
			content := normalize(c.convertNode(child, ctx))
			if content != "" {
				m := marker()
				items = append(items, prefixLines(content, m, indent(len(m))))
			}
		}
	}

	return strings.Join(items, "\n")
}

// renderListItem renders an <li> as marker plus content, with continuation
// lines indented to the content column.
func (c *converter) renderListItem(n *html.Node, ctx context, marker string) string {
	ctx.listItems++
	if box := leadingCheckbox(n); box != nil {
		if hasAttr(box, "checked") {
			marker += "[x] "
		} else {
			marker += "[ ] "
		}
	}
	content := normalize(c.convertChildren(n, ctx))
	if content == "" {
		return ""
	}
	return prefixLines(content, marker, indent(len(strings.TrimSuffix(strings.TrimSuffix(marker, "[x] "), "[ ] "))))
}

// leadingCheckbox returns the checkbox <input> that starts a GFM task list
// item, if any.
func leadingCheckbox(li *html.Node) *html.Node {
	for ch := li.FirstChild; ch != nil; ch = ch.NextSibling {
		switch ch.Type {
		case html.TextNode:
			if strings.TrimSpace(ch.Data) != "" {
				return nil
			}
		case html.ElementNode:
			if strings.EqualFold(ch.Data, "input") && strings.EqualFold(getAttr(ch, "type"), "checkbox") {
				return ch
			}
			return nil
		}
	}
	return nil
}

// onlyNestedLists reports whether n contains lists and nothing else but
// whitespace.
func onlyNestedLists(n *html.Node) bool {
	found := false
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		switch ch.Type {
		case html.TextNode:
			if strings.TrimSpace(ch.Data) != "" {
				return false
			}
		case html.ElementNode:
			tag := strings.ToLower(ch.Data)
			if tag != "ul" && tag != "ol" {
				return false
			}
			found = true
		}
	}
	return found
}

// handleDefinitionList renders terms in bold, each followed by its
// definitions as ": definition" lines. A <dt> or <dd> without its
// counterpart renders as a plain block.
func (c *converter) handleDefinitionList(n *html.Node, ctx context) string {
	var items []*html.Node
	var collect func(*html.Node)
	collect = func(n *html.Node) {
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type != html.ElementNode || c.skip(ch) {
				continue
			}
			switch strings.ToLower(ch.Data) {
			case "dt", "dd":
				items = append(items, ch)
			case "div":
				// HTML allows a <div> around each term and its definitions.
				collect(ch)
			}
		}
	}
	collect(n)

	hasDefinition := false
	for _, it := range items {
		if strings.EqualFold(it.Data, "dd") {
			hasDefinition = true
		}
	}

	var b strings.Builder
	prev := ""
	sawTerm := false
	for _, it := range items {
		if strings.EqualFold(it.Data, "dt") {
			inner := ctx
			if hasDefinition {
				inner.bold, inner.emphasis = true, true
			}
			term := collapseSpace(c.convertChildren(it, inner))
			if term == "" {
				continue
			}
			if hasDefinition && strings.ReplaceAll(term, `\*`, "") != "" {
				term = "**" + term + "**"
			}
			switch prev {
			case "dt":
				b.WriteString("\n")
			case "dd":
				b.WriteString("\n\n")
			}
			b.WriteString(term)
			prev, sawTerm = "dt", true
			continue
		}
		body := normalize(c.convertChildren(it, ctx))
		if body == "" {
			continue
		}
		switch {
		case !sawTerm:
			if prev != "" {
				b.WriteString("\n\n")
			}
			b.WriteString(body)
		default:
			b.WriteString("\n")
			b.WriteString(prefixLines(body, ": ", indent(2)))
		}
		prev = "dd"
	}
	if b.Len() == 0 {
		return ""
	}
	return "\n\n" + b.String() + "\n\n"
}

// handleBlockquote converts the quote's content with ordinary block flow,
// then prefixes every line with "> ".
func (c *converter) handleBlockquote(n *html.Node, ctx context) string {
	content := normalize(c.convertChildren(n, ctx))
	if content == "" {
		return ""
	}
	return "\n\n" + prefixLines(content, quotePrefix, quotePrefix) + "\n\n"
}

func (c *converter) handlePre(n *html.Node, ctx context) string {
	// Markup inside <pre> (links, highlighting spans) reduces to its text,
	// since Markdown inside a code block renders literally.
	content := strings.Trim(c.textContent(n), "\n")

	if ctx.tableCell {
		// A table cell holds one line: each code line becomes a code span.
		var spans []string
		for _, line := range strings.Split(content, "\n") {
			if span := codeSpan(line); span != "" {
				spans = append(spans, span)
			}
		}
		return strings.Join(spans, "<br>")
	}

	var body string
	if c.opts.CodeBlockStyle == CodeBlockStyleIndented {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if line != "" {
				lines[i] = "    " + line
			}
		}
		body = strings.Join(lines, "\n")
	} else {
		// The fence must be longer than any backtick run in the content so
		// that code containing ``` doesn't terminate the block early.
		fence := "```"
		if run := longestBacktickRun(content); run >= 3 {
			fence = strings.Repeat("`", run+1)
		}
		body = fence + c.codeLanguage(n) + "\n" + content + "\n" + fence
	}

	return "\n\n" + markCodeStart + "\n" + body + "\n" + markCodeEnd + "\n\n"
}

// codeLanguage finds the language of a <pre> block from a <code> child, the
// <pre> itself, or a nearby wrapper such as Sphinx's
// <div class="highlight-python">.
func (c *converter) codeLanguage(pre *html.Node) string {
	var nodes []*html.Node
	for ch := pre.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type == html.ElementNode && strings.EqualFold(ch.Data, "code") {
			nodes = append(nodes, ch)
		}
	}
	nodes = append(nodes, pre)
	for p, i := pre.Parent, 0; p != nil && p.Type == html.ElementNode && i < 3; p, i = p.Parent, i+1 {
		nodes = append(nodes, p)
	}
	for _, nd := range nodes {
		for _, attr := range []string{"data-lang", "data-language"} {
			if l := strings.TrimSpace(getAttr(nd, attr)); validLanguage(l) {
				return l
			}
		}
		if l := classLanguage(getAttr(nd, "class")); l != "" {
			return l
		}
	}
	return ""
}

// classLanguage extracts a code language from a class attribute, recognizing
// the conventions of common syntax highlighters: "language-x", "lang-x",
// GitHub's "highlight-source-x", and Sphinx's "highlight-x". The class may
// contain other tokens (e.g. "hljs language-go").
func classLanguage(class string) string {
	for _, cls := range strings.Fields(class) {
		for _, prefix := range []string{"language-", "lang-", "highlight-source-", "highlight-"} {
			if l := strings.TrimPrefix(cls, prefix); l != cls {
				if validLanguage(l) {
					return l
				}
				break
			}
		}
	}
	return ""
}

func validLanguage(l string) bool {
	return l != "" && l != "default" && l != "none" && !strings.ContainsAny(l, "` \t\n")
}

func (c *converter) handleTable(n *html.Node, ctx context) string {
	caption := ""
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type == html.ElementNode && strings.EqualFold(ch.Data, "caption") && !c.skip(ch) {
			caption = collapseSpace(finalize(normalize(c.convertChildren(ch, ctx))))
			break
		}
	}
	captionBlock := ""
	if caption != "" {
		captionBlock = "\n\n" + caption + "\n\n"
	}

	trs := tableRowNodes(n)
	if c.isLayoutTable(n, trs) {
		// A table used for page layout holds ordinary content: render each
		// cell as a block.
		var b strings.Builder
		b.WriteString(captionBlock)
		for _, tr := range trs {
			for _, cell := range cellNodes(tr) {
				if c.skip(cell) {
					continue
				}
				b.WriteString("\n\n" + c.convertChildren(cell, ctx) + "\n\n")
			}
		}
		return b.String()
	}

	rows := c.tableRows(trs)
	colCount := 0
	for _, row := range rows {
		colCount = max(colCount, len(row))
	}
	if colCount == 0 {
		return captionBlock
	}
	for i := range rows {
		for len(rows[i]) < colCount {
			rows[i] = append(rows[i], "")
		}
	}

	var lines []string
	lines = append(lines, "| "+strings.Join(rows[0], " | ")+" |")
	sep := make([]string, colCount)
	for i := range sep {
		sep[i] = "---"
	}
	lines = append(lines, "| "+strings.Join(sep, " | ")+" |")
	for _, row := range rows[1:] {
		lines = append(lines, "| "+strings.Join(row, " | ")+" |")
	}

	return captionBlock + "\n\n" + strings.Join(lines, "\n") + "\n\n"
}

var (
	// layoutCellTags cannot be represented in a pipe table cell.
	layoutCellTags = map[string]bool{
		"table": true, "pre": true, "blockquote": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	}
	// multilineCellTags mark a single-column table as a content wrapper.
	multilineCellTags = map[string]bool{"p": true, "ul": true, "ol": true, "dl": true, "br": true}
)

// isLayoutTable reports whether a table arranges page content rather than
// holding data: it is marked presentational, or it has no header cells and
// its cells hold content a pipe table cannot (nested tables, code blocks,
// headings), or it is a single column of multi-line content.
func (c *converter) isLayoutTable(table *html.Node, trs []*html.Node) bool {
	switch strings.ToLower(strings.TrimSpace(getAttr(table, "role"))) {
	case "presentation", "none":
		return true
	}
	cols := 0
	for _, tr := range trs {
		if tr.Parent != nil && strings.EqualFold(tr.Parent.Data, "thead") {
			return false
		}
		n := 0
		for _, cell := range cellNodes(tr) {
			if strings.EqualFold(cell.Data, "th") {
				return false
			}
			n += spanAttr(cell, "colspan", 1000)
		}
		cols = max(cols, n)
	}
	for _, tr := range trs {
		for _, cell := range cellNodes(tr) {
			if hasDescendant(cell, layoutCellTags) || (cols == 1 && hasDescendant(cell, multilineCellTags)) {
				return true
			}
		}
	}
	return false
}

// tableRowNodes returns the rows of a table, not those of nested tables.
func tableRowNodes(table *html.Node) []*html.Node {
	var trs []*html.Node
	for ch := table.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type != html.ElementNode {
			continue
		}
		switch strings.ToLower(ch.Data) {
		case "thead", "tbody", "tfoot":
			trs = append(trs, tableRowNodes(ch)...)
		case "tr":
			trs = append(trs, ch)
		}
	}
	return trs
}

func cellNodes(tr *html.Node) []*html.Node {
	var cells []*html.Node
	for ch := tr.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type == html.ElementNode && (strings.EqualFold(ch.Data, "td") || strings.EqualFold(ch.Data, "th")) {
			cells = append(cells, ch)
		}
	}
	return cells
}

// tableRows renders table rows as cell strings. A cell spanning columns or
// rows is followed by, or leaves below it, empty cells so later cells stay
// under their headers.
func (c *converter) tableRows(trs []*html.Node) [][]string {
	var rows [][]string
	pending := map[int]int{} // column -> rows still covered by a rowspan above
	for _, tr := range trs {
		var row []string
		hasCell := false
		col := 0
		fill := func() {
			for pending[col] > 0 {
				pending[col]--
				row = append(row, "")
				col++
			}
		}
		for _, cell := range cellNodes(tr) {
			fill()
			hasCell = true
			content := ""
			if !c.skip(cell) {
				content = c.cellContent(cell)
			}
			colspan := spanAttr(cell, "colspan", 1000)
			rowspan := spanAttr(cell, "rowspan", 65534)
			for i := 0; i < colspan; i++ {
				if i == 0 {
					row = append(row, content)
				} else {
					row = append(row, "")
				}
				if rowspan > 1 {
					pending[col] = rowspan - 1
				}
				col++
			}
		}
		// Columns past the last cell may still be covered from above.
		last := -1
		for k, v := range pending {
			if v > 0 && k > last {
				last = k
			}
		}
		for ; col <= last; col++ {
			if pending[col] > 0 {
				pending[col]--
			}
			row = append(row, "")
		}
		// Skip rows without cells, so an empty leading <tr> does not
		// become the header row.
		if hasCell {
			rows = append(rows, row)
		}
	}
	return rows
}

// cellContent renders a table cell on one line: line breaks become <br>,
// blocks are joined by spaces, and pipes are escaped.
func (c *converter) cellContent(cell *html.Node) string {
	s := finalize(normalize(c.convertChildren(cell, context{tableCell: true})))
	var parts []string
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			parts = append(parts, t)
		}
	}
	s = strings.Join(parts, " ")
	for {
		t := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "<br>"), "<br>"))
		if t == s {
			break
		}
		s = t
	}
	return strings.ReplaceAll(s, "|", `\|`)
}

// spanAttr parses a colspan or rowspan attribute, clamped to [1, limit].
func spanAttr(n *html.Node, key string, limit int) int {
	v, err := strconv.Atoi(strings.TrimSpace(getAttr(n, key)))
	if err != nil || v < 1 {
		return 1
	}
	return min(v, limit)
}

func hasDescendant(n *html.Node, tags map[string]bool) bool {
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type == html.ElementNode && (tags[strings.ToLower(ch.Data)] || hasDescendant(ch, tags)) {
			return true
		}
	}
	return false
}

// structuralPrefix splits line into its leading structural markers and the
// rest.
func structuralPrefix(line string) (prefix, rest string) {
	i := 0
	for i < len(line) && (line[i] == markIndent || line[i] == markQuote) {
		i++
	}
	return line[:i], line[i:]
}

func isMarkerLine(line string) bool {
	_, rest := structuralPrefix(line)
	return rest == markCodeStart || rest == markCodeEnd
}

func isBlankLine(line string) bool {
	return strings.Trim(line, string([]byte{markIndent, markQuote})) == ""
}

func indent(width int) string {
	return strings.Repeat(string(rune(markIndent)), width)
}

// prefixLines prefixes the first content line of a normalized fragment with
// first and every other line with rest. Code block marker lines take the
// rest prefix, so first lands on the fence.
func prefixLines(s, first, rest string) string {
	lines := strings.Split(s, "\n")
	done := false
	for i, line := range lines {
		switch {
		case !done && !isMarkerLine(line):
			lines[i] = first + line
			done = true
		case isBlankLine(line):
			lines[i] = strings.TrimRight(rest+line, string(rune(markIndent)))
		default:
			lines[i] = rest + line
		}
	}
	return strings.Join(lines, "\n")
}

// normalize cleans up a converted fragment: whitespace from the HTML source
// is trimmed from each line (structural prefixes are kept), runs of spaces
// and blank lines collapse, and leading and trailing blank lines go. Lines
// inside code blocks are kept exactly. normalize is idempotent, so nested
// blocks can normalize their content before prefixing it.
func normalize(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	inCode := false
	for _, line := range lines {
		prefix, rest := structuralPrefix(line)
		if inCode {
			out = append(out, line)
			if rest == markCodeEnd {
				inCode = false
			}
			continue
		}
		switch rest {
		case markCodeStart:
			inCode = true
			out = append(out, line)
			continue
		case markTight:
			for len(out) > 0 && isBlankLine(out[len(out)-1]) {
				out = out[:len(out)-1]
			}
			continue
		}
		rest = strings.Trim(rest, " \t")
		if rest == "" {
			if len(out) == 0 || isBlankLine(out[len(out)-1]) {
				continue
			}
			out = append(out, strings.TrimRight(prefix, string(rune(markIndent))))
			continue
		}
		// Inline elements can leave two spaces where their edges meet,
		// as in "*x*  **y**". Text nodes are already collapsed, so any run
		// of spaces is one of these. Table rows are left alone, since
		// their cells may hold code.
		if !strings.HasPrefix(rest, "|") {
			rest = multiSpaceRegex.ReplaceAllString(rest, " ")
		}
		out = append(out, prefix+rest)
	}
	for len(out) > 0 && isBlankLine(out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// finalize turns a normalized fragment into Markdown: code block marker
// lines are dropped and structural prefixes become spaces and ">".
func finalize(s string) string {
	lines := strings.Split(s, "\n")
	out := lines[:0]
	for _, line := range lines {
		if !isMarkerLine(line) {
			out = append(out, line)
		}
	}
	s = strings.Join(out, "\n")
	s = strings.NewReplacer(string(rune(markIndent)), " ", string(rune(markQuote)), ">").Replace(s)
	return strings.Trim(s, "\n")
}

// stripMarks removes the control characters the converter uses as markers.
func stripMarks(s string) string {
	if !strings.ContainsAny(s, "\x01\x02\x03\x05\x06") {
		return s
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case markIndent, '\x02', '\x03', markQuote, '\x06':
			return -1
		}
		return r
	}, s)
}

func collapseSpace(s string) string {
	return strings.TrimSpace(whitespaceRegex.ReplaceAllString(s, " "))
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == key {
			return stripMarks(attr.Val)
		}
	}
	return ""
}

func hasAttr(n *html.Node, key string) bool {
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == key {
			return true
		}
	}
	return false
}

func cleanText(s string) string {
	// Simple fallback for completely unparseable input
	s = tagStripRegex.ReplaceAllString(s, "")
	s = whitespaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
