// Package htmlparse provides HTML parsing, metadata extraction, and transformation utilities.
//
// This package is designed for extracting structured data from HTML documents
// and transforming HTML content for various use cases like web scraping,
// content extraction for LLMs, and metadata analysis.
//
// # Core Features
//
//   - Parse HTML from strings or io.Reader
//   - Extract metadata (title, description, Open Graph, Twitter Cards)
//   - Extract links with filtering (internal/external)
//   - Extract images and branding elements
//   - Transform HTML with flexible filtering
//   - Convert HTML to plain text
//
// # Basic Usage
//
//	doc, err := htmlparse.Parse("<html>...</html>")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Extract metadata
//	meta := doc.Metadata()
//	fmt.Println(meta.Title, meta.Description)
//
//	// Extract links
//	links := doc.Links()
//	for _, link := range links {
//	    fmt.Println(link.URL, link.Text)
//	}
//
// # Filtering Links
//
// Extract only internal or external links by providing a base URL:
//
//	// Get only internal links
//	internal := doc.FilteredLinks(htmlparse.LinkFilter{
//	    BaseURL:  "https://example.com",
//	    Internal: true,
//	})
//
//	// Get only external links
//	external := doc.FilteredLinks(htmlparse.LinkFilter{
//	    BaseURL:  "https://example.com",
//	    External: true,
//	})
//
// # Transforming HTML
//
// Transform HTML to extract clean content, exclude unwanted elements,
// or prepare content for LLM consumption:
//
//	// Extract only main content, exclude navigation and ads
//	clean := doc.Transform(&htmlparse.TransformOptions{
//	    OnlyMainContent: true,
//	    Exclude:         []string{"nav", "footer"},
//	    ExcludeFilters:  htmlparse.StandardExcludeFilters,
//	    PrettyPrint:     true,
//	})
//
// # Advanced Element Filtering
//
// Use ElementFilter for sophisticated element exclusion based on
// attributes and values:
//
//	// Exclude cookie banners, modals, and ads
//	clean := doc.Transform(&htmlparse.TransformOptions{
//	    ExcludeFilters: []htmlparse.ElementFilter{
//	        {Attr: "role", AttrEquals: "dialog"},
//	        {Attr: "id", AttrContains: "cookie"},
//	        {Attr: "class", AttrContains: "modal"},
//	    },
//	})
package htmlparse

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Document represents a parsed HTML document and provides methods
// for extracting metadata, links, images, and transforming content.
//
// Create a Document using Parse or ParseReader, then call methods
// to extract data or transform the HTML.
type Document struct {
	root *html.Node
}

// Parse parses HTML content from a string into a Document.
//
// This is a convenience wrapper around ParseReader for string input.
// The HTML is parsed using the golang.org/x/net/html parser, which is
// lenient and handles malformed HTML gracefully.
//
// Example:
//
//	doc, err := htmlparse.Parse("<html><body><h1>Hello</h1></body></html>")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(doc.Text()) // Output: Hello
func Parse(htmlContent string) (*Document, error) {
	return ParseReader(strings.NewReader(htmlContent))
}

// ParseReader parses HTML from an io.Reader into a Document.
//
// This is useful for parsing HTML from HTTP responses, files, or other
// streaming sources without loading the entire content into memory first.
//
// Example:
//
//	resp, _ := http.Get("https://example.com")
//	defer resp.Body.Close()
//	doc, err := htmlparse.ParseReader(resp.Body)
func ParseReader(r io.Reader) (*Document, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	return &Document{root: root}, nil
}

// Metadata contains extracted page metadata from HTML <head> elements.
//
// This includes standard meta tags, Open Graph protocol data, and
// Twitter Card data. Use Document.Metadata() to extract this information.
//
// Fields are omitted from JSON if empty (omitempty tags).
type Metadata struct {
	Title       string     `json:"title,omitempty"`       // Page title from <title> tag
	Description string     `json:"description,omitempty"` // Meta description
	Author      string     `json:"author,omitempty"`      // Meta author
	Keywords    []string   `json:"keywords,omitempty"`    // Meta keywords (comma-separated)
	Canonical   string     `json:"canonical,omitempty"`   // Canonical URL from <link rel="canonical">
	Charset     string     `json:"charset,omitempty"`     // Character encoding
	Viewport    string     `json:"viewport,omitempty"`    // Viewport configuration
	Robots      string     `json:"robots,omitempty"`      // Robot indexing directives
	OpenGraph   *OpenGraph `json:"opengraph,omitempty"`   // Open Graph protocol metadata
	Twitter     *Twitter   `json:"twitter,omitempty"`     // Twitter Card metadata
}

// OpenGraph contains Open Graph protocol metadata.
//
// Open Graph is used by social media platforms (Facebook, LinkedIn, etc.)
// to display rich previews when a URL is shared. These are extracted from
// <meta property="og:*"> tags.
type OpenGraph struct {
	Title       string `json:"title,omitempty"`       // og:title
	Description string `json:"description,omitempty"` // og:description
	Image       string `json:"image,omitempty"`       // og:image
	URL         string `json:"url,omitempty"`         // og:url
	Type        string `json:"type,omitempty"`        // og:type (e.g., "website", "article")
	SiteName    string `json:"siteName,omitempty"`    // og:site_name
}

// Twitter contains Twitter Card metadata.
//
// Twitter Cards control how links are displayed when shared on Twitter/X.
// These are extracted from <meta name="twitter:*"> tags.
type Twitter struct {
	Card        string `json:"card,omitempty"`        // twitter:card (e.g., "summary", "summary_large_image")
	Title       string `json:"title,omitempty"`       // twitter:title
	Description string `json:"description,omitempty"` // twitter:description
	Image       string `json:"image,omitempty"`       // twitter:image
	Site        string `json:"site,omitempty"`        // twitter:site (e.g., "@username")
	Creator     string `json:"creator,omitempty"`     // twitter:creator (e.g., "@username")
}

// Metadata extracts page metadata from the document.
//
// This method extracts standard meta tags, Open Graph data, and Twitter Card
// data from the HTML <head> section. It returns a Metadata struct containing
// all discovered values.
//
// Example:
//
//	doc, _ := htmlparse.Parse(html)
//	meta := doc.Metadata()
//	fmt.Println(meta.Title)
//	if meta.OpenGraph != nil {
//	    fmt.Println(meta.OpenGraph.Image)
//	}
func (d *Document) Metadata() Metadata {
	var m Metadata
	var og OpenGraph
	var tw Twitter

	d.walkNodes(d.root, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		tag := strings.ToLower(n.Data)

		switch tag {
		case "title":
			m.Title = getTextContent(n)
		case "meta":
			name := strings.ToLower(getAttr(n, "name"))
			property := strings.ToLower(getAttr(n, "property"))
			content := getAttr(n, "content")
			charset := getAttr(n, "charset")

			if charset != "" {
				m.Charset = charset
			}

			switch name {
			case "description":
				m.Description = content
			case "author":
				m.Author = content
			case "keywords":
				m.Keywords = parseKeywords(content)
			case "viewport":
				m.Viewport = content
			case "robots":
				m.Robots = content
			}

			switch property {
			case "og:title":
				og.Title = content
			case "og:description":
				og.Description = content
			case "og:image":
				og.Image = content
			case "og:url":
				og.URL = content
			case "og:type":
				og.Type = content
			case "og:site_name":
				og.SiteName = content
			}

			switch name {
			case "twitter:card":
				tw.Card = content
			case "twitter:title":
				tw.Title = content
			case "twitter:description":
				tw.Description = content
			case "twitter:image":
				tw.Image = content
			case "twitter:site":
				tw.Site = content
			case "twitter:creator":
				tw.Creator = content
			}
		case "link":
			rel := strings.ToLower(getAttr(n, "rel"))
			if rel == "canonical" {
				m.Canonical = getAttr(n, "href")
			}
		}
		return true
	})

	if og != (OpenGraph{}) {
		m.OpenGraph = &og
	}
	if tw != (Twitter{}) {
		m.Twitter = &tw
	}

	return m
}

// Link represents a hyperlink (<a> tag) found in the document.
type Link struct {
	URL   string `json:"url"`             // The href attribute value
	Text  string `json:"text,omitempty"`  // The link text content
	Title string `json:"title,omitempty"` // The title attribute value
}

// Links extracts all links (<a> tags with href) from the document.
//
// This is equivalent to calling FilteredLinks with an empty filter.
// Links without an href attribute are excluded.
//
// Example:
//
//	doc, _ := htmlparse.Parse(html)
//	links := doc.Links()
//	for _, link := range links {
//	    fmt.Printf("%s -> %s\n", link.Text, link.URL)
//	}
func (d *Document) Links() []Link {
	return d.FilteredLinks(LinkFilter{})
}

// LinkFilter configures which links to extract and how to process them.
//
// Use BaseURL to resolve relative links to absolute URLs. Set Internal or
// External (but not both) to filter links by whether they point to the same
// host as BaseURL.
type LinkFilter struct {
	// BaseURL for resolving relative links. If empty, relative URLs are kept as-is.
	// Example: "https://example.com" will resolve "/page" to "https://example.com/page"
	BaseURL string

	// Internal includes only links to the same host as BaseURL.
	// Requires BaseURL to be set. Mutually exclusive with External.
	Internal bool

	// External includes only links to different hosts than BaseURL.
	// Requires BaseURL to be set. Mutually exclusive with Internal.
	External bool
}

// Image represents an image (<img> tag) found in the document.
type Image struct {
	URL   string `json:"url"`             // The src attribute value
	Alt   string `json:"alt,omitempty"`   // The alt text
	Title string `json:"title,omitempty"` // The title attribute value
}

// Images extracts all images (<img> tags with src) from the document.
//
// Images without a src attribute are excluded.
//
// Example:
//
//	doc, _ := htmlparse.Parse(html)
//	images := doc.Images()
//	for _, img := range images {
//	    fmt.Printf("%s: %s\n", img.Alt, img.URL)
//	}
func (d *Document) Images() []Image {
	var images []Image
	d.walkNodes(d.root, func(n *html.Node) bool {
		if n.Type != html.ElementNode || strings.ToLower(n.Data) != "img" {
			return true
		}
		src := getAttr(n, "src")
		if src != "" {
			images = append(images, Image{
				URL:   src,
				Alt:   getAttr(n, "alt"),
				Title: getAttr(n, "title"),
			})
		}
		return true
	})
	return images
}

// FilteredLinks extracts links matching the filter criteria.
//
// This method allows you to filter links by internal/external status and
// resolve relative URLs to absolute URLs using a base URL.
//
// Example:
//
//	// Get only external links
//	external := doc.FilteredLinks(htmlparse.LinkFilter{
//	    BaseURL:  "https://example.com",
//	    External: true,
//	})
func (d *Document) FilteredLinks(f LinkFilter) []Link {
	var baseURL *url.URL
	if f.BaseURL != "" {
		baseURL, _ = url.Parse(f.BaseURL)
	}

	var links []Link
	d.walkNodes(d.root, func(n *html.Node) bool {
		if n.Type != html.ElementNode || strings.ToLower(n.Data) != "a" {
			return true
		}

		href := getAttr(n, "href")
		if href == "" {
			return true
		}

		link := Link{
			URL:   href,
			Text:  strings.TrimSpace(getTextContent(n)),
			Title: getAttr(n, "title"),
		}

		// Resolve relative URL if base is provided
		if baseURL != nil {
			resolved, err := baseURL.Parse(href)
			if err == nil {
				link.URL = resolved.String()
			}
		}

		// Apply internal/external filter
		if f.Internal || f.External {
			if baseURL == nil {
				return true
			}
			linkURL, err := url.Parse(link.URL)
			if err != nil {
				return true
			}
			isInternal := linkURL.Host == "" || linkURL.Host == baseURL.Host
			if f.Internal && !isInternal {
				return true
			}
			if f.External && isInternal {
				return true
			}
		}

		links = append(links, link)
		return true
	})
	return links
}

// ElementFilter defines criteria for matching HTML elements during transformation.
//
// All non-empty fields must match for the filter to match an element.
// This enables sophisticated filtering based on tag names and attribute values.
//
// Example filters:
//
//	// Match all <script> tags
//	{Tag: "script"}
//
//	// Match elements with role="dialog"
//	{Attr: "role", AttrEquals: "dialog"}
//
//	// Match elements with "modal" in class attribute
//	{Attr: "class", AttrContains: "modal"}
//
//	// Match <img> tags with data-cookieconsent attribute
//	{Tag: "img", Attr: "data-cookieconsent"}
type ElementFilter struct {
	// Tag matches the element tag name (e.g., "script", "nav").
	// Empty matches any tag. Case-insensitive.
	Tag string `json:"tag,omitempty"`

	// Attr specifies the attribute name to check.
	// If set alone, matches elements that have this attribute (any value).
	// Case-insensitive attribute name matching.
	Attr string `json:"attr,omitempty"`

	// AttrEquals matches elements where Attr equals this value exactly.
	// Case-sensitive value matching.
	AttrEquals string `json:"attr_equals,omitempty"`

	// AttrContains matches elements where Attr contains this substring.
	// Case-insensitive substring matching.
	AttrContains string `json:"attr_contains,omitempty"`
}

// Matches returns true if the filter matches the given element.
//
// This method is used internally during transformation and can be used
// directly for testing filter logic.
//
// Parameters:
//   - tag: The element tag name (e.g., "div", "script")
//   - attrs: Map of attribute names to values
//
// Returns true only if all non-empty filter fields match.
func (f ElementFilter) Matches(tag string, attrs map[string]string) bool {
	// Check tag if specified
	if f.Tag != "" && !strings.EqualFold(f.Tag, tag) {
		return false
	}

	// If no attribute filter, we're done (tag-only filter)
	if f.Attr == "" {
		return f.Tag != "" // Must have matched something
	}

	// Get attribute value (case-insensitive key lookup)
	var attrVal string
	var hasAttr bool
	for k, v := range attrs {
		if strings.EqualFold(k, f.Attr) {
			attrVal = v
			hasAttr = true
			break
		}
	}

	// Attribute must exist
	if !hasAttr {
		return false
	}

	// Check value constraints
	if f.AttrEquals != "" && attrVal != f.AttrEquals {
		return false
	}
	if f.AttrContains != "" && !strings.Contains(strings.ToLower(attrVal), strings.ToLower(f.AttrContains)) {
		return false
	}

	return true
}

// StandardExcludeFilters contains commonly excluded elements for clean content extraction.
//
// This pre-configured filter list excludes:
//   - Modal dialogs and popups (role="dialog", aria-modal="true", cookie banners)
//   - Non-content elements (script, style, noscript, iframe, svg)
//   - Form elements (select, input, button, form)
//   - Navigation elements (nav, footer, hr)
//
// Use this with TransformOptions.ExcludeFilters to quickly clean HTML for
// content extraction or LLM consumption.
//
// Example:
//
//	clean := doc.Transform(&htmlparse.TransformOptions{
//	    ExcludeFilters: htmlparse.StandardExcludeFilters,
//	})
var StandardExcludeFilters = []ElementFilter{
	// Modal/dialog elements
	{Attr: "role", AttrEquals: "dialog"},
	{Attr: "aria-modal", AttrEquals: "true"},
	{Attr: "id", AttrContains: "cookie"},
	{Attr: "id", AttrContains: "popup"},
	{Attr: "id", AttrContains: "modal"},
	{Attr: "class", AttrContains: "modal"},
	{Attr: "class", AttrContains: "dialog"},
	{Tag: "img", Attr: "data-cookieconsent"},
	// Non-content elements
	{Tag: "script"},
	{Tag: "style"},
	{Tag: "noscript"},
	{Tag: "iframe"},
	{Tag: "svg"},
	// Form elements
	{Tag: "select"},
	{Tag: "input"},
	{Tag: "button"},
	{Tag: "form"},
	// Navigation
	{Tag: "nav"},
	{Tag: "footer"},
	{Tag: "hr"},
}

// TransformOptions configures HTML transformation and filtering.
//
// Use these options to extract clean content from HTML documents,
// remove unwanted elements, or prepare HTML for specific use cases
// like LLM consumption or content analysis.
type TransformOptions struct {
	// Include only elements with these tag names.
	// If empty, all tags are included (except those excluded).
	// Example: []string{"p", "h1", "h2"} to keep only paragraphs and headings.
	Include []string

	// Exclude elements with these tag names (simple tag-based exclusion).
	// Example: []string{"nav", "footer"} to remove navigation and footers.
	Exclude []string

	// ExcludeFilters provides advanced element filtering based on attributes.
	// Elements matching any filter will be excluded.
	// See StandardExcludeFilters for common exclusion patterns.
	ExcludeFilters []ElementFilter

	// OnlyMainContent extracts only the main content area.
	// Uses <main> if present, otherwise uses <body> and excludes
	// nav, header, footer, aside elements.
	OnlyMainContent bool

	// PrettyPrint formats the output HTML with indentation for readability.
	// Useful for debugging or human review.
	PrettyPrint bool
}

// renderOpts holds rendering configuration.
type renderOpts struct {
	includeSet     map[string]bool
	excludeSet     map[string]bool
	excludeFilters []ElementFilter
}

// isExcluded checks if an element node should be excluded.
func (r *renderOpts) isExcluded(n *html.Node) bool {
	tag := strings.ToLower(n.Data)

	// Check simple tag exclusion
	if r.excludeSet[tag] {
		return true
	}

	// Check element filters
	if len(r.excludeFilters) > 0 {
		attrs := nodeAttrs(n)
		for _, f := range r.excludeFilters {
			if f.Matches(tag, attrs) {
				return true
			}
		}
	}

	return false
}

// nodeAttrs extracts attributes from a node into a map.
func nodeAttrs(n *html.Node) map[string]string {
	attrs := make(map[string]string, len(n.Attr))
	for _, attr := range n.Attr {
		attrs[attr.Key] = attr.Val
	}
	return attrs
}

// Transform returns the document HTML with transformations applied.
//
// This method applies filtering and formatting options to produce
// a transformed version of the HTML. Common use cases include:
//   - Extracting main content for LLM processing
//   - Removing ads, modals, and navigation
//   - Cleaning HTML for content analysis
//   - Preparing HTML for display or archival
//
// If opts is nil, returns the full document with only script/style removed.
//
// Example:
//
//	// Extract clean main content
//	clean := doc.Transform(&htmlparse.TransformOptions{
//	    OnlyMainContent: true,
//	    ExcludeFilters:  htmlparse.StandardExcludeFilters,
//	    PrettyPrint:     true,
//	})
func (d *Document) Transform(opts *TransformOptions) string {
	if opts == nil {
		opts = &TransformOptions{}
	}

	rOpts := &renderOpts{
		includeSet:     toSet(opts.Include),
		excludeSet:     toSet(opts.Exclude),
		excludeFilters: opts.ExcludeFilters,
	}

	// Add default exclusions for main content mode
	if opts.OnlyMainContent {
		for _, tag := range []string{"nav", "header", "footer", "aside", "script", "style", "noscript"} {
			rOpts.excludeSet[tag] = true
		}
	}

	// Always exclude these
	for _, tag := range []string{"script", "style", "noscript"} {
		rOpts.excludeSet[tag] = true
	}

	// Find the content root
	root := d.root
	if opts.OnlyMainContent {
		if main := d.findElement("main"); main != nil {
			root = main
		} else if body := d.findElement("body"); body != nil {
			root = body
		}
	}

	var buf strings.Builder
	if opts.PrettyPrint {
		d.renderPretty(&buf, root, 0, rOpts)
	} else {
		d.renderCompact(&buf, root, rOpts)
	}

	result := buf.String()
	if opts.PrettyPrint {
		result = strings.TrimSpace(result)
	}
	return result
}

// HTML returns the full document HTML without transformation.
//
// This returns the complete HTML document as parsed, including
// all elements, scripts, styles, and formatting. For a cleaned
// version, use Transform instead.
func (d *Document) HTML() string {
	var buf strings.Builder
	html.Render(&buf, d.root)
	return buf.String()
}

// Text returns the plain text content of the document.
//
// This extracts all visible text from the document, excluding:
//   - Script and style tag contents
//   - Head section contents
//   - HTML tags and attributes
//
// Text from different elements is separated by spaces.
//
// Example:
//
//	doc, _ := htmlparse.Parse("<html><body><h1>Title</h1><p>Content</p></body></html>")
//	text := doc.Text()
//	fmt.Println(text) // Output: Title Content
func (d *Document) Text() string {
	var buf strings.Builder
	skipTags := map[string]bool{"script": true, "style": true, "noscript": true, "head": true}

	d.walkNodes(d.root, func(n *html.Node) bool {
		if n.Type == html.ElementNode && skipTags[strings.ToLower(n.Data)] {
			return false // skip this subtree
		}
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				if buf.Len() > 0 {
					buf.WriteString(" ")
				}
				buf.WriteString(text)
			}
		}
		return true
	})

	return buf.String()
}

// walkNodes traverses the DOM tree, calling fn for each node.
// If fn returns false, the node's children are skipped.
func (d *Document) walkNodes(n *html.Node, fn func(*html.Node) bool) {
	if n == nil {
		return
	}
	if !fn(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		d.walkNodes(c, fn)
	}
}

func (d *Document) findElement(tag string) *html.Node {
	var found *html.Node
	d.walkNodes(d.root, func(n *html.Node) bool {
		if found != nil {
			return false
		}
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == tag {
			found = n
			return false
		}
		return true
	})
	return found
}

func (d *Document) renderCompact(buf *strings.Builder, n *html.Node, opts *renderOpts) {
	if n == nil {
		return
	}

	switch n.Type {
	case html.TextNode:
		buf.WriteString(n.Data)
	case html.ElementNode:
		tag := strings.ToLower(n.Data)

		// Check exclusion
		if opts.isExcluded(n) {
			return
		}

		// Check inclusion (if include list is provided)
		if len(opts.includeSet) > 0 && !opts.includeSet[tag] {
			// Still process children for included tags
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				d.renderCompact(buf, c, opts)
			}
			return
		}

		// Render opening tag
		buf.WriteString("<")
		buf.WriteString(n.Data)
		for _, attr := range n.Attr {
			buf.WriteString(" ")
			buf.WriteString(attr.Key)
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(attr.Val))
			buf.WriteString(`"`)
		}

		if isVoidElement(tag) {
			buf.WriteString(">")
			return
		}
		buf.WriteString(">")

		// Render children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			d.renderCompact(buf, c, opts)
		}

		// Render closing tag
		buf.WriteString("</")
		buf.WriteString(n.Data)
		buf.WriteString(">")
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			d.renderCompact(buf, c, opts)
		}
	case html.CommentNode:
		// Skip comments
	case html.DoctypeNode:
		buf.WriteString("<!DOCTYPE ")
		buf.WriteString(n.Data)
		buf.WriteString(">")
	}
}

func (d *Document) renderPretty(buf *strings.Builder, n *html.Node, depth int, opts *renderOpts) {
	if n == nil {
		return
	}

	indent := strings.Repeat("  ", depth)

	switch n.Type {
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text != "" {
			buf.WriteString(indent)
			buf.WriteString(text)
			buf.WriteString("\n")
		}
	case html.ElementNode:
		tag := strings.ToLower(n.Data)

		// Check exclusion
		if opts.isExcluded(n) {
			return
		}

		// Check inclusion (if include list is provided)
		if len(opts.includeSet) > 0 && !opts.includeSet[tag] {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				d.renderPretty(buf, c, depth, opts)
			}
			return
		}

		// Render opening tag
		buf.WriteString(indent)
		buf.WriteString("<")
		buf.WriteString(n.Data)
		for _, attr := range n.Attr {
			buf.WriteString(" ")
			buf.WriteString(attr.Key)
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(attr.Val))
			buf.WriteString(`"`)
		}

		if isVoidElement(tag) {
			buf.WriteString(">\n")
			return
		}

		// Check if this is an inline element with only text content
		if isInlineTextOnly(n) {
			buf.WriteString(">")
			buf.WriteString(getTextContent(n))
			buf.WriteString("</")
			buf.WriteString(n.Data)
			buf.WriteString(">\n")
			return
		}

		buf.WriteString(">\n")

		// Render children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			d.renderPretty(buf, c, depth+1, opts)
		}

		// Render closing tag
		buf.WriteString(indent)
		buf.WriteString("</")
		buf.WriteString(n.Data)
		buf.WriteString(">\n")
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			d.renderPretty(buf, c, depth, opts)
		}
	case html.DoctypeNode:
		buf.WriteString("<!DOCTYPE ")
		buf.WriteString(n.Data)
		buf.WriteString(">\n")
	}
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return buf.String()
}

func parseKeywords(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var keywords []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keywords = append(keywords, p)
		}
	}
	return keywords
}

func toSet(slice []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range slice {
		m[strings.ToLower(s)] = true
	}
	return m
}

func isVoidElement(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}

func isInlineTextOnly(n *html.Node) bool {
	// Check if element has only text node children (no nested elements)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			return false
		}
	}
	return true
}

// Branding contains brand identity information extracted from a page.
//
// This includes visual branding elements like logos, icons, and theme colors
// that can be used to identify or represent the website.
type Branding struct {
	ColorScheme string `json:"color_scheme,omitempty"` // Color scheme preference (e.g., "light dark")
	ThemeColor  string `json:"theme_color,omitempty"`  // Primary theme color (e.g., "#ff5500")
	Logo        string `json:"logo,omitempty"`         // Logo image URL (detected heuristically)
	Favicon     string `json:"favicon,omitempty"`      // Favicon URL from <link rel="icon">
	AppleIcon   string `json:"apple_icon,omitempty"`   // Apple touch icon URL
}

// Branding extracts brand identity information from the document.
//
// This method attempts to identify visual branding elements including:
//   - Theme colors from meta tags
//   - Favicon and Apple touch icons from link tags
//   - Logo images (detected heuristically from img tags with "logo" in attributes)
//
// Example:
//
//	doc, _ := htmlparse.Parse(html)
//	brand := doc.Branding()
//	fmt.Println(brand.ThemeColor, brand.Logo)
func (d *Document) Branding() Branding {
	var b Branding

	d.walkNodes(d.root, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		tag := strings.ToLower(n.Data)

		switch tag {
		case "meta":
			name := strings.ToLower(getAttr(n, "name"))
			content := getAttr(n, "content")

			switch name {
			case "theme-color":
				b.ThemeColor = content
			case "color-scheme":
				b.ColorScheme = content
			}

		case "link":
			rel := strings.ToLower(getAttr(n, "rel"))
			href := getAttr(n, "href")

			switch rel {
			case "icon", "shortcut icon":
				if b.Favicon == "" {
					b.Favicon = href
				}
			case "apple-touch-icon", "apple-touch-icon-precomposed":
				if b.AppleIcon == "" {
					b.AppleIcon = href
				}
			}

		case "img":
			// Look for logo in img elements
			if b.Logo != "" {
				return true
			}
			src := getAttr(n, "src")
			if src == "" {
				return true
			}
			// Check if this looks like a logo
			if isLikelyLogo(n) {
				b.Logo = src
			}
		}
		return true
	})

	return b
}

// isLikelyLogo checks if an img element is likely a logo based on attributes.
func isLikelyLogo(n *html.Node) bool {
	// Check various attributes for "logo" indicators
	for _, attr := range n.Attr {
		key := strings.ToLower(attr.Key)
		val := strings.ToLower(attr.Val)

		switch key {
		case "class", "id", "alt", "title":
			if strings.Contains(val, "logo") {
				return true
			}
		case "src":
			// Check if filename contains "logo"
			if strings.Contains(val, "logo") {
				return true
			}
		}
	}
	return false
}
