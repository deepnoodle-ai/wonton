// Package htmlparse provides HTML parsing and transformation utilities.
//
// It offers a simple API for extracting metadata, links, and content from HTML
// documents, with support for filtering and transformation.
//
// Basic usage:
//
//	doc, err := htmlparse.Parse("<html>...</html>")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	meta := doc.Metadata()
//	links := doc.Links()
//
// With transformation:
//
//	html := doc.Transform(&htmlparse.TransformOptions{
//	    OnlyMainContent: true,
//	    Exclude:         []string{"nav", "footer"},
//	    PrettyPrint:     true,
//	})
package htmlparse

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Document represents a parsed HTML document.
type Document struct {
	root *html.Node
}

// Parse parses HTML content into a Document.
func Parse(htmlContent string) (*Document, error) {
	return ParseReader(strings.NewReader(htmlContent))
}

// ParseReader parses HTML from an io.Reader into a Document.
func ParseReader(r io.Reader) (*Document, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	return &Document{root: root}, nil
}

// Metadata contains extracted page metadata.
type Metadata struct {
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Author      string     `json:"author,omitempty"`
	Keywords    []string   `json:"keywords,omitempty"`
	Canonical   string     `json:"canonical,omitempty"`
	Charset     string     `json:"charset,omitempty"`
	Viewport    string     `json:"viewport,omitempty"`
	Robots      string     `json:"robots,omitempty"`
	OpenGraph   *OpenGraph `json:"opengraph,omitempty"`
	Twitter     *Twitter   `json:"twitter,omitempty"`
}

// OpenGraph contains Open Graph protocol metadata.
type OpenGraph struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
	SiteName    string `json:"siteName,omitempty"`
}

// Twitter contains Twitter Card metadata.
type Twitter struct {
	Card        string `json:"card,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	Site        string `json:"site,omitempty"`
	Creator     string `json:"creator,omitempty"`
}

// Metadata extracts page metadata from the document.
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

// Link represents a hyperlink found in the document.
type Link struct {
	URL   string `json:"url"`
	Text  string `json:"text,omitempty"`
	Title string `json:"title,omitempty"`
}

// Links extracts all links from the document.
func (d *Document) Links() []Link {
	return d.FilteredLinks(LinkFilter{})
}

// LinkFilter configures which links to extract.
type LinkFilter struct {
	// BaseURL for resolving relative links. If empty, relative URLs are kept as-is.
	BaseURL string

	// Internal includes only links to the same host as BaseURL.
	// Requires BaseURL to be set.
	Internal bool

	// External includes only links to different hosts than BaseURL.
	// Requires BaseURL to be set.
	External bool
}

// Image represents an image found in the document.
type Image struct {
	URL   string `json:"url"`
	Alt   string `json:"alt,omitempty"`
	Title string `json:"title,omitempty"`
}

// Images extracts all images from the document.
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

// ElementFilter defines criteria for matching HTML elements.
// All non-empty fields must match for the filter to match an element.
type ElementFilter struct {
	// Tag matches the element tag name (e.g., "script", "nav").
	// Empty matches any tag.
	Tag string `json:"tag,omitempty"`

	// Attr specifies the attribute name to check.
	// If set alone, matches elements that have this attribute (any value).
	Attr string `json:"attr,omitempty"`

	// AttrEquals matches elements where Attr equals this value exactly.
	AttrEquals string `json:"attr_equals,omitempty"`

	// AttrContains matches elements where Attr contains this substring.
	AttrContains string `json:"attr_contains,omitempty"`
}

// Matches returns true if the filter matches the given element.
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

// TransformOptions configures HTML transformation.
type TransformOptions struct {
	// Include only elements with these tag names.
	// If empty, all tags are included (except those excluded).
	Include []string

	// Exclude elements with these tag names (simple exclusion).
	Exclude []string

	// ExcludeFilters provides advanced element filtering.
	// Elements matching any filter will be excluded.
	ExcludeFilters []ElementFilter

	// OnlyMainContent extracts only the main content area.
	// Uses <main> if present, otherwise excludes nav, header, footer, aside.
	OnlyMainContent bool

	// PrettyPrint formats the output HTML with indentation.
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
func (d *Document) HTML() string {
	var buf strings.Builder
	html.Render(&buf, d.root)
	return buf.String()
}

// Text returns the plain text content of the document.
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
type Branding struct {
	ColorScheme string `json:"color_scheme,omitempty"`
	ThemeColor  string `json:"theme_color,omitempty"`
	Logo        string `json:"logo,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
	AppleIcon   string `json:"apple_icon,omitempty"`
}

// Branding extracts brand identity information from the document.
// This includes logo, favicon, theme color, and color scheme when detectable.
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
