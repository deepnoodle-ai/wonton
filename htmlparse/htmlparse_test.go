package htmlparse

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestParse(t *testing.T) {
	doc, err := Parse("<html><body><p>Hello</p></body></html>")
	assert.NoError(t, err)
	assert.NotNil(t, doc)
}

func TestParseReader(t *testing.T) {
	r := strings.NewReader("<html><body><p>Hello</p></body></html>")
	doc, err := ParseReader(r)
	assert.NoError(t, err)
	assert.NotNil(t, doc)
}

func TestMetadata_Title(t *testing.T) {
	doc, _ := Parse("<html><head><title>My Page</title></head></html>")
	m := doc.Metadata()
	assert.Equal(t, "My Page", m.Title)
}

func TestMetadata_Description(t *testing.T) {
	doc, _ := Parse(`<html><head><meta name="description" content="A test page"></head></html>`)
	m := doc.Metadata()
	assert.Equal(t, "A test page", m.Description)
}

func TestMetadata_Author(t *testing.T) {
	doc, _ := Parse(`<html><head><meta name="author" content="John Doe"></head></html>`)
	m := doc.Metadata()
	assert.Equal(t, "John Doe", m.Author)
}

func TestMetadata_Keywords(t *testing.T) {
	doc, _ := Parse(`<html><head><meta name="keywords" content="go, html, parser"></head></html>`)
	m := doc.Metadata()
	assert.Equal(t, []string{"go", "html", "parser"}, m.Keywords)
}

func TestMetadata_Canonical(t *testing.T) {
	doc, _ := Parse(`<html><head><link rel="canonical" href="https://example.com/page"></head></html>`)
	m := doc.Metadata()
	assert.Equal(t, "https://example.com/page", m.Canonical)
}

func TestMetadata_Charset(t *testing.T) {
	doc, _ := Parse(`<html><head><meta charset="UTF-8"></head></html>`)
	m := doc.Metadata()
	assert.Equal(t, "UTF-8", m.Charset)
}

func TestMetadata_Viewport(t *testing.T) {
	doc, _ := Parse(`<html><head><meta name="viewport" content="width=device-width, initial-scale=1"></head></html>`)
	m := doc.Metadata()
	assert.Equal(t, "width=device-width, initial-scale=1", m.Viewport)
}

func TestMetadata_Robots(t *testing.T) {
	doc, _ := Parse(`<html><head><meta name="robots" content="noindex, nofollow"></head></html>`)
	m := doc.Metadata()
	assert.Equal(t, "noindex, nofollow", m.Robots)
}

func TestMetadata_OpenGraph(t *testing.T) {
	html := `<html><head>
		<meta property="og:title" content="OG Title">
		<meta property="og:description" content="OG Description">
		<meta property="og:image" content="https://example.com/image.png">
		<meta property="og:url" content="https://example.com">
		<meta property="og:type" content="website">
		<meta property="og:site_name" content="Example Site">
	</head></html>`
	doc, _ := Parse(html)
	m := doc.Metadata()
	assert.Equal(t, "OG Title", m.OpenGraph.Title)
	assert.Equal(t, "OG Description", m.OpenGraph.Description)
	assert.Equal(t, "https://example.com/image.png", m.OpenGraph.Image)
	assert.Equal(t, "https://example.com", m.OpenGraph.URL)
	assert.Equal(t, "website", m.OpenGraph.Type)
	assert.Equal(t, "Example Site", m.OpenGraph.SiteName)
}

func TestMetadata_Twitter(t *testing.T) {
	html := `<html><head>
		<meta name="twitter:card" content="summary_large_image">
		<meta name="twitter:title" content="Twitter Title">
		<meta name="twitter:description" content="Twitter Description">
		<meta name="twitter:image" content="https://example.com/twitter.png">
		<meta name="twitter:site" content="@example">
		<meta name="twitter:creator" content="@author">
	</head></html>`
	doc, _ := Parse(html)
	m := doc.Metadata()
	assert.Equal(t, "summary_large_image", m.Twitter.Card)
	assert.Equal(t, "Twitter Title", m.Twitter.Title)
	assert.Equal(t, "Twitter Description", m.Twitter.Description)
	assert.Equal(t, "https://example.com/twitter.png", m.Twitter.Image)
	assert.Equal(t, "@example", m.Twitter.Site)
	assert.Equal(t, "@author", m.Twitter.Creator)
}

func TestMetadata_Full(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Full Test Page</title>
		<meta name="description" content="A comprehensive test">
		<meta name="author" content="Test Author">
		<meta name="keywords" content="test, comprehensive, metadata">
		<meta name="viewport" content="width=device-width">
		<meta name="robots" content="index, follow">
		<link rel="canonical" href="https://example.com/test">
		<meta property="og:title" content="OG Test">
		<meta name="twitter:card" content="summary">
	</head>
	<body><p>Content</p></body>
	</html>`

	doc, _ := Parse(html)
	m := doc.Metadata()

	assert.Equal(t, "Full Test Page", m.Title)
	assert.Equal(t, "A comprehensive test", m.Description)
	assert.Equal(t, "Test Author", m.Author)
	assert.Equal(t, []string{"test", "comprehensive", "metadata"}, m.Keywords)
	assert.Equal(t, "UTF-8", m.Charset)
	assert.Equal(t, "width=device-width", m.Viewport)
	assert.Equal(t, "index, follow", m.Robots)
	assert.Equal(t, "https://example.com/test", m.Canonical)
	assert.Equal(t, "OG Test", m.OpenGraph.Title)
	assert.Equal(t, "summary", m.Twitter.Card)
}

func TestLinks_Basic(t *testing.T) {
	html := `<html><body>
		<a href="https://example.com">Example</a>
		<a href="/about">About</a>
		<a href="#section">Section</a>
	</body></html>`

	doc, _ := Parse(html)
	links := doc.Links()

	assert.Equal(t, 3, len(links))
	assert.Equal(t, "https://example.com", links[0].URL)
	assert.Equal(t, "Example", links[0].Text)
	assert.Equal(t, "/about", links[1].URL)
	assert.Equal(t, "About", links[1].Text)
	assert.Equal(t, "#section", links[2].URL)
	assert.Equal(t, "Section", links[2].Text)
}

func TestLinks_WithTitle(t *testing.T) {
	html := `<a href="https://example.com" title="Click here">Link</a>`
	doc, _ := Parse(html)
	links := doc.Links()

	assert.Equal(t, 1, len(links))
	assert.Equal(t, "Click here", links[0].Title)
}

func TestLinks_EmptyHref(t *testing.T) {
	html := `<a href="">Empty</a><a>No href</a><a href="valid">Valid</a>`
	doc, _ := Parse(html)
	links := doc.Links()

	assert.Equal(t, 1, len(links))
	assert.Equal(t, "valid", links[0].URL)
}

func TestLinks_NestedElements(t *testing.T) {
	html := `<a href="url"><strong>Bold</strong> link</a>`
	doc, _ := Parse(html)
	links := doc.Links()

	assert.Equal(t, 1, len(links))
	assert.Equal(t, "Bold link", links[0].Text)
}

func TestFilteredLinks_BaseURL(t *testing.T) {
	html := `<a href="/page">Relative</a><a href="https://other.com">External</a>`
	doc, _ := Parse(html)
	links := doc.FilteredLinks(LinkFilter{BaseURL: "https://example.com"})

	assert.Equal(t, 2, len(links))
	assert.Equal(t, "https://example.com/page", links[0].URL)
	assert.Equal(t, "https://other.com", links[1].URL)
}

func TestFilteredLinks_InternalOnly(t *testing.T) {
	html := `
		<a href="/page">Internal</a>
		<a href="https://example.com/other">Also Internal</a>
		<a href="https://other.com">External</a>
	`
	doc, _ := Parse(html)
	links := doc.FilteredLinks(LinkFilter{
		BaseURL:  "https://example.com",
		Internal: true,
	})

	assert.Equal(t, 2, len(links))
	assert.Equal(t, "https://example.com/page", links[0].URL)
	assert.Equal(t, "https://example.com/other", links[1].URL)
}

func TestFilteredLinks_ExternalOnly(t *testing.T) {
	html := `
		<a href="/page">Internal</a>
		<a href="https://example.com/other">Also Internal</a>
		<a href="https://other.com">External</a>
	`
	doc, _ := Parse(html)
	links := doc.FilteredLinks(LinkFilter{
		BaseURL:  "https://example.com",
		External: true,
	})

	assert.Equal(t, 1, len(links))
	assert.Equal(t, "https://other.com", links[0].URL)
}

func TestTransform_Basic(t *testing.T) {
	html := `<html><body><p>Hello</p></body></html>`
	doc, _ := Parse(html)
	result := doc.Transform(nil)
	assert.Contains(t, result, "<p>Hello</p>")
}

func TestTransform_Include(t *testing.T) {
	html := `<html><body><p>Keep</p><div>Remove</div><p>Also keep</p></body></html>`
	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{Include: []string{"p"}})

	assert.Contains(t, result, "<p>Keep</p>")
	assert.Contains(t, result, "<p>Also keep</p>")
	assert.NotContains(t, result, "<div>")
}

func TestTransform_Exclude(t *testing.T) {
	html := `<html><body><nav>Navigation</nav><p>Content</p><footer>Footer</footer></body></html>`
	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{Exclude: []string{"nav", "footer"}})

	assert.Contains(t, result, "<p>Content</p>")
	assert.NotContains(t, result, "<nav>")
	assert.NotContains(t, result, "<footer>")
}

func TestTransform_OnlyMainContent(t *testing.T) {
	html := `<html><body>
		<nav>Navigation</nav>
		<main><p>Main content</p></main>
		<footer>Footer</footer>
	</body></html>`
	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{OnlyMainContent: true})

	assert.Contains(t, result, "<p>Main content</p>")
	assert.NotContains(t, result, "<nav>")
	assert.NotContains(t, result, "<footer>")
}

func TestTransform_OnlyMainContent_NoMain(t *testing.T) {
	html := `<html><body>
		<nav>Navigation</nav>
		<p>Body content</p>
		<footer>Footer</footer>
	</body></html>`
	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{OnlyMainContent: true})

	assert.Contains(t, result, "<p>Body content</p>")
	assert.NotContains(t, result, "<nav>")
	assert.NotContains(t, result, "<footer>")
}

func TestTransform_PrettyPrint(t *testing.T) {
	html := `<html><body><div><p>Hello</p></div></body></html>`
	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{PrettyPrint: true})

	// Should have proper indentation
	assert.Contains(t, result, "\n")
	lines := strings.Split(result, "\n")
	foundIndented := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") {
			foundIndented = true
			break
		}
	}
	assert.True(t, foundIndented)
}

func TestTransform_ScriptStyleRemoved(t *testing.T) {
	html := `<html><head><style>body{}</style></head><body><script>alert(1)</script><p>Content</p></body></html>`
	doc, _ := Parse(html)
	result := doc.Transform(nil)

	assert.Contains(t, result, "<p>Content</p>")
	assert.NotContains(t, result, "<script>")
	assert.NotContains(t, result, "<style>")
	assert.NotContains(t, result, "alert")
}

func TestHTML(t *testing.T) {
	input := `<html><head></head><body><p>Test</p></body></html>`
	doc, _ := Parse(input)
	result := doc.HTML()

	// Should return valid HTML
	assert.Contains(t, result, "<p>Test</p>")
	assert.Contains(t, result, "<html>")
}

func TestText(t *testing.T) {
	html := `<html><body><h1>Title</h1><p>First paragraph.</p><p>Second paragraph.</p></body></html>`
	doc, _ := Parse(html)
	text := doc.Text()

	assert.Contains(t, text, "Title")
	assert.Contains(t, text, "First paragraph.")
	assert.Contains(t, text, "Second paragraph.")
}

func TestText_IgnoresScriptStyle(t *testing.T) {
	html := `<html><head><style>body{color:red}</style></head><body><script>alert(1)</script><p>Visible</p></body></html>`
	doc, _ := Parse(html)
	text := doc.Text()

	assert.Contains(t, text, "Visible")
	assert.NotContains(t, text, "alert")
	assert.NotContains(t, text, "color")
}

func TestText_IgnoresHead(t *testing.T) {
	html := `<html><head><title>Page Title</title></head><body><p>Body content</p></body></html>`
	doc, _ := Parse(html)
	text := doc.Text()

	assert.Contains(t, text, "Body content")
	assert.NotContains(t, text, "Page Title")
}

func TestVoidElements(t *testing.T) {
	html := `<p>Hello<br>World</p><img src="test.png"><input type="text">`
	doc, _ := Parse(html)
	result := doc.Transform(nil)

	// Void elements should not have closing tags
	assert.NotContains(t, result, "</br>")
	assert.NotContains(t, result, "</img>")
	assert.NotContains(t, result, "</input>")
}

func TestParseKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"one", []string{"one"}},
		{"one, two, three", []string{"one", "two", "three"}},
		{"  spaced  ,  values  ", []string{"spaced", "values"}},
		{"trailing,", []string{"trailing"}},
	}

	for _, tt := range tests {
		result := parseKeywords(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMalformedHTML(t *testing.T) {
	tests := []string{
		"<p>Unclosed",
		"<div><p>Mismatched</div></p>",
		"Just text",
		"<>Invalid</>",
		"",
	}

	for _, html := range tests {
		doc, err := Parse(html)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		// Should not panic
		_ = doc.Metadata()
		_ = doc.Links()
		_ = doc.Text()
		_ = doc.HTML()
		_ = doc.Transform(nil)
	}
}

func TestRealWorldHTML(t *testing.T) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Example Page</title>
    <meta name="description" content="An example page for testing">
    <meta property="og:title" content="Example OG Title">
    <link rel="canonical" href="https://example.com/page">
</head>
<body>
    <header>
        <nav>
            <a href="/">Home</a>
            <a href="/about">About</a>
        </nav>
    </header>
    <main>
        <article>
            <h1>Welcome</h1>
            <p>This is the main content with a <a href="https://external.com">link</a>.</p>
        </article>
    </main>
    <footer>
        <p>Copyright 2024</p>
    </footer>
    <script>console.log("hi");</script>
</body>
</html>`

	doc, err := Parse(html)
	assert.NoError(t, err)

	// Metadata
	meta := doc.Metadata()
	assert.Equal(t, "Example Page", meta.Title)
	assert.Equal(t, "An example page for testing", meta.Description)
	assert.Equal(t, "Example OG Title", meta.OpenGraph.Title)
	assert.Equal(t, "https://example.com/page", meta.Canonical)

	// Links
	links := doc.Links()
	assert.Equal(t, 3, len(links))

	// Internal links only
	internal := doc.FilteredLinks(LinkFilter{
		BaseURL:  "https://example.com",
		Internal: true,
	})
	assert.Equal(t, 2, len(internal))

	// External links only
	external := doc.FilteredLinks(LinkFilter{
		BaseURL:  "https://example.com",
		External: true,
	})
	assert.Equal(t, 1, len(external))
	assert.Equal(t, "https://external.com", external[0].URL)

	// Main content
	mainContent := doc.Transform(&TransformOptions{OnlyMainContent: true})
	assert.Contains(t, mainContent, "Welcome")
	assert.Contains(t, mainContent, "main content")
	assert.NotContains(t, mainContent, "<nav>")
	assert.NotContains(t, mainContent, "<footer>")
	assert.NotContains(t, mainContent, "console.log")

	// Text
	text := doc.Text()
	assert.Contains(t, text, "Welcome")
	assert.NotContains(t, text, "console.log")
}

func TestTransform_CombinedOptions(t *testing.T) {
	html := `<html><body>
		<nav><a href="/">Nav</a></nav>
		<main>
			<h1>Title</h1>
			<p>Content</p>
			<div>Div content</div>
		</main>
		<footer>Footer</footer>
	</body></html>`

	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{
		OnlyMainContent: true,
		Include:         []string{"h1", "p"},
		PrettyPrint:     true,
	})

	assert.Contains(t, result, "<h1>Title</h1>")
	assert.Contains(t, result, "<p>Content</p>")
	assert.NotContains(t, result, "<div>")
	assert.NotContains(t, result, "<nav>")
}

func TestLinks_VariousProtocols(t *testing.T) {
	html := `
		<a href="https://example.com">HTTPS</a>
		<a href="http://example.com">HTTP</a>
		<a href="mailto:test@example.com">Email</a>
		<a href="tel:+1234567890">Phone</a>
		<a href="javascript:void(0)">JS</a>
		<a href="ftp://files.example.com">FTP</a>
	`
	doc, _ := Parse(html)
	links := doc.Links()
	assert.Equal(t, 6, len(links))
}

func TestTransform_PreservesAttributes(t *testing.T) {
	html := `<a href="https://example.com" class="link" id="main-link" target="_blank">Link</a>`
	doc, _ := Parse(html)
	result := doc.Transform(nil)

	assert.Contains(t, result, `href="https://example.com"`)
	assert.Contains(t, result, `class="link"`)
	assert.Contains(t, result, `id="main-link"`)
	assert.Contains(t, result, `target="_blank"`)
}

func TestTransform_EscapesAttributeValues(t *testing.T) {
	html := `<a href="url?a=1&b=2" title="Say &quot;hello&quot;">Link</a>`
	doc, _ := Parse(html)
	result := doc.Transform(nil)

	// Should properly escape special characters
	assert.Contains(t, result, `href="url?a=1&amp;b=2"`)
}

func TestMetadata_EmptyDocument(t *testing.T) {
	doc, _ := Parse("")
	m := doc.Metadata()

	assert.Equal(t, "", m.Title)
	assert.Equal(t, "", m.Description)
	assert.Nil(t, m.Keywords)
	assert.Nil(t, m.OpenGraph)
	assert.Nil(t, m.Twitter)
}

func TestMetadata_JSON(t *testing.T) {
	// With OpenGraph/Twitter
	html := `<html><head>
		<title>Test</title>
		<meta property="og:title" content="OG Title">
	</head></html>`
	doc, _ := Parse(html)
	m := doc.Metadata()

	assert.NotNil(t, m.OpenGraph)
	assert.Nil(t, m.Twitter)

	// Verify JSON omits nil pointers
	data, err := json.Marshal(m)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"opengraph"`)
	assert.NotContains(t, string(data), `"twitter"`)
}

func TestMetadata_JSON_Empty(t *testing.T) {
	doc, _ := Parse("<html></html>")
	m := doc.Metadata()

	data, err := json.Marshal(m)
	assert.NoError(t, err)
	// Should be minimal JSON with no opengraph or twitter
	assert.NotContains(t, string(data), `"opengraph"`)
	assert.NotContains(t, string(data), `"twitter"`)
}

func TestLinks_EmptyDocument(t *testing.T) {
	doc, _ := Parse("")
	links := doc.Links()
	assert.Equal(t, 0, len(links))
}

func TestTransform_EmptyDocument(t *testing.T) {
	doc, _ := Parse("")
	result := doc.Transform(nil)
	// Should not panic, returns empty or minimal HTML
	assert.NotNil(t, result)
}

// ElementFilter tests

func TestElementFilter_MatchesTag(t *testing.T) {
	f := ElementFilter{Tag: "div"}

	assert.True(t, f.Matches("div", nil))
	assert.True(t, f.Matches("DIV", nil)) // case insensitive
	assert.False(t, f.Matches("span", nil))
}

func TestElementFilter_MatchesAttrPresence(t *testing.T) {
	f := ElementFilter{Attr: "data-test"}

	assert.True(t, f.Matches("div", map[string]string{"data-test": ""}))
	assert.True(t, f.Matches("div", map[string]string{"data-test": "value"}))
	assert.True(t, f.Matches("span", map[string]string{"DATA-TEST": "value"})) // case insensitive
	assert.False(t, f.Matches("div", map[string]string{"other": "value"}))
	assert.False(t, f.Matches("div", nil))
}

func TestElementFilter_MatchesAttrEquals(t *testing.T) {
	f := ElementFilter{Attr: "role", AttrEquals: "dialog"}

	assert.True(t, f.Matches("div", map[string]string{"role": "dialog"}))
	assert.False(t, f.Matches("div", map[string]string{"role": "button"}))
	assert.False(t, f.Matches("div", map[string]string{"role": "dialog-box"}))
	assert.False(t, f.Matches("div", nil))
}

func TestElementFilter_MatchesAttrContains(t *testing.T) {
	f := ElementFilter{Attr: "class", AttrContains: "modal"}

	assert.True(t, f.Matches("div", map[string]string{"class": "modal"}))
	assert.True(t, f.Matches("div", map[string]string{"class": "my-modal-dialog"}))
	assert.True(t, f.Matches("div", map[string]string{"class": "Modal"})) // case insensitive
	assert.False(t, f.Matches("div", map[string]string{"class": "dialog"}))
	assert.False(t, f.Matches("div", nil))
}

func TestElementFilter_MatchesTagAndAttr(t *testing.T) {
	f := ElementFilter{Tag: "img", Attr: "data-cookieconsent"}

	assert.True(t, f.Matches("img", map[string]string{"data-cookieconsent": ""}))
	assert.False(t, f.Matches("div", map[string]string{"data-cookieconsent": ""}))
	assert.False(t, f.Matches("img", map[string]string{"src": "test.jpg"}))
}

func TestElementFilter_MatchesIdContains(t *testing.T) {
	f := ElementFilter{Attr: "id", AttrContains: "cookie"}

	assert.True(t, f.Matches("div", map[string]string{"id": "cookie-banner"}))
	assert.True(t, f.Matches("div", map[string]string{"id": "my-cookie-popup"}))
	assert.False(t, f.Matches("div", map[string]string{"id": "main-content"}))
}

func TestElementFilter_EmptyFilterMatchesNothing(t *testing.T) {
	f := ElementFilter{}

	assert.False(t, f.Matches("div", nil))
	assert.False(t, f.Matches("div", map[string]string{"class": "test"}))
}

// Transform with ExcludeFilters tests

func TestTransform_ExcludeFilters_ByRole(t *testing.T) {
	html := `<html><body>
		<div role="dialog">Modal content</div>
		<p>Regular content</p>
	</body></html>`

	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{
		ExcludeFilters: []ElementFilter{
			{Attr: "role", AttrEquals: "dialog"},
		},
	})

	assert.Contains(t, result, "Regular content")
	assert.NotContains(t, result, "Modal content")
	assert.NotContains(t, result, `role="dialog"`)
}

func TestTransform_ExcludeFilters_ByClassContains(t *testing.T) {
	html := `<html><body>
		<div class="cookie-banner">Cookie notice</div>
		<div class="ad-container">Advertisement</div>
		<p>Main content</p>
	</body></html>`

	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{
		ExcludeFilters: []ElementFilter{
			{Attr: "class", AttrContains: "cookie"},
			{Attr: "class", AttrContains: "ad-"},
		},
	})

	assert.Contains(t, result, "Main content")
	assert.NotContains(t, result, "Cookie notice")
	assert.NotContains(t, result, "Advertisement")
}

func TestTransform_ExcludeFilters_ByIdContains(t *testing.T) {
	html := `<html><body>
		<div id="popup-overlay">Popup</div>
		<div id="modal-container">Modal</div>
		<p>Content</p>
	</body></html>`

	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{
		ExcludeFilters: []ElementFilter{
			{Attr: "id", AttrContains: "popup"},
			{Attr: "id", AttrContains: "modal"},
		},
	})

	assert.Contains(t, result, "Content")
	assert.NotContains(t, result, "Popup")
	assert.NotContains(t, result, "Modal")
}

func TestTransform_ExcludeFilters_TagWithAttr(t *testing.T) {
	html := `<html><body>
		<img src="logo.png" alt="Logo">
		<img src="tracking.gif" data-cookieconsent="statistics">
		<p>Text</p>
	</body></html>`

	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{
		ExcludeFilters: []ElementFilter{
			{Tag: "img", Attr: "data-cookieconsent"},
		},
	})

	assert.Contains(t, result, "logo.png")
	assert.Contains(t, result, "Text")
	assert.NotContains(t, result, "tracking.gif")
}

func TestTransform_ExcludeFilters_AriaModal(t *testing.T) {
	html := `<html><body>
		<div aria-modal="true">Modal dialog</div>
		<p>Page content</p>
	</body></html>`

	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{
		ExcludeFilters: []ElementFilter{
			{Attr: "aria-modal", AttrEquals: "true"},
		},
	})

	assert.Contains(t, result, "Page content")
	assert.NotContains(t, result, "Modal dialog")
}

func TestTransform_ExcludeFilters_CombinedWithExcludeTags(t *testing.T) {
	html := `<html><body>
		<nav>Navigation</nav>
		<div class="modal">Modal</div>
		<p>Content</p>
	</body></html>`

	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{
		Exclude: []string{"nav"},
		ExcludeFilters: []ElementFilter{
			{Attr: "class", AttrContains: "modal"},
		},
	})

	assert.Contains(t, result, "Content")
	assert.NotContains(t, result, "Navigation")
	assert.NotContains(t, result, "Modal")
}

func TestTransform_StandardExcludeFilters(t *testing.T) {
	html := `<html><body>
		<div role="dialog">Dialog</div>
		<div aria-modal="true">Modal</div>
		<div id="cookie-consent">Cookies</div>
		<div id="popup-overlay">Popup</div>
		<div class="modal-container">ModalClass</div>
		<nav>Nav</nav>
		<footer>Footer</footer>
		<script>alert('hi')</script>
		<svg><circle/></svg>
		<form><input type="text"></form>
		<p>Real content here</p>
	</body></html>`

	doc, _ := Parse(html)
	result := doc.Transform(&TransformOptions{
		ExcludeFilters: StandardExcludeFilters,
	})

	assert.Contains(t, result, "Real content here")
	assert.NotContains(t, result, "Dialog")
	assert.NotContains(t, result, "Modal")
	assert.NotContains(t, result, "Cookies")
	assert.NotContains(t, result, "Popup")
	assert.NotContains(t, result, "ModalClass")
	assert.NotContains(t, result, "<nav>")
	assert.NotContains(t, result, "<footer>")
	assert.NotContains(t, result, "<script>")
	assert.NotContains(t, result, "<svg>")
	assert.NotContains(t, result, "<form>")
}

// Images tests

func TestImages_Basic(t *testing.T) {
	html := `<html><body>
		<img src="photo.jpg" alt="A photo" title="My Photo">
		<img src="icon.png">
	</body></html>`

	doc, _ := Parse(html)
	images := doc.Images()

	assert.Equal(t, 2, len(images))
	assert.Equal(t, "photo.jpg", images[0].URL)
	assert.Equal(t, "A photo", images[0].Alt)
	assert.Equal(t, "My Photo", images[0].Title)
	assert.Equal(t, "icon.png", images[1].URL)
	assert.Equal(t, "", images[1].Alt)
	assert.Equal(t, "", images[1].Title)
}

func TestImages_NoSrc(t *testing.T) {
	html := `<html><body><img alt="No source"></body></html>`

	doc, _ := Parse(html)
	images := doc.Images()

	assert.Equal(t, 0, len(images))
}

func TestImages_Empty(t *testing.T) {
	doc, _ := Parse("<html><body><p>No images</p></body></html>")
	images := doc.Images()

	assert.Equal(t, 0, len(images))
}

// Branding tests

func TestBranding_ThemeColor(t *testing.T) {
	html := `<html><head>
		<meta name="theme-color" content="#ff5500">
	</head></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "#ff5500", b.ThemeColor)
}

func TestBranding_ColorScheme(t *testing.T) {
	html := `<html><head>
		<meta name="color-scheme" content="light dark">
	</head></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "light dark", b.ColorScheme)
}

func TestBranding_Favicon(t *testing.T) {
	html := `<html><head>
		<link rel="icon" href="/favicon.ico">
	</head></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "/favicon.ico", b.Favicon)
}

func TestBranding_FaviconShortcut(t *testing.T) {
	html := `<html><head>
		<link rel="shortcut icon" href="/favicon.png">
	</head></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "/favicon.png", b.Favicon)
}

func TestBranding_AppleTouchIcon(t *testing.T) {
	html := `<html><head>
		<link rel="apple-touch-icon" href="/apple-icon.png">
	</head></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "/apple-icon.png", b.AppleIcon)
}

func TestBranding_LogoByClass(t *testing.T) {
	html := `<html><body>
		<img src="/images/site-logo.png" class="logo" alt="Site">
	</body></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "/images/site-logo.png", b.Logo)
}

func TestBranding_LogoById(t *testing.T) {
	html := `<html><body>
		<img src="/brand.svg" id="main-logo">
	</body></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "/brand.svg", b.Logo)
}

func TestBranding_LogoByAlt(t *testing.T) {
	html := `<html><body>
		<img src="/company.png" alt="Company Logo">
	</body></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "/company.png", b.Logo)
}

func TestBranding_LogoBySrc(t *testing.T) {
	html := `<html><body>
		<img src="/assets/logo.svg">
	</body></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "/assets/logo.svg", b.Logo)
}

func TestBranding_LogoFirstMatch(t *testing.T) {
	html := `<html><body>
		<img src="/first-logo.png" class="logo">
		<img src="/second-logo.png" class="logo">
	</body></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "/first-logo.png", b.Logo)
}

func TestBranding_NoLogoWithoutHint(t *testing.T) {
	html := `<html><body>
		<img src="/photo.jpg" alt="A regular photo">
	</body></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "", b.Logo)
}

func TestBranding_Complete(t *testing.T) {
	html := `<html><head>
		<meta name="theme-color" content="#0066cc">
		<meta name="color-scheme" content="dark">
		<link rel="icon" href="/favicon.ico">
		<link rel="apple-touch-icon" href="/apple-icon.png">
	</head><body>
		<img src="/logo.svg" class="site-logo" alt="Brand">
	</body></html>`

	doc, _ := Parse(html)
	b := doc.Branding()

	assert.Equal(t, "#0066cc", b.ThemeColor)
	assert.Equal(t, "dark", b.ColorScheme)
	assert.Equal(t, "/favicon.ico", b.Favicon)
	assert.Equal(t, "/apple-icon.png", b.AppleIcon)
	assert.Equal(t, "/logo.svg", b.Logo)
}

// Examples for godoc

func ExampleParse() {
	html := `<html><body><h1>Hello, World!</h1><p>This is a paragraph.</p></body></html>`
	doc, err := Parse(html)
	if err != nil {
		panic(err)
	}

	text := doc.Text()
	fmt.Println(text)
	// Output: Hello, World! This is a paragraph.
}

func ExampleDocument_Metadata() {
	html := `<html>
	<head>
		<title>My Page</title>
		<meta name="description" content="A sample page">
		<meta property="og:image" content="https://example.com/image.png">
	</head>
	<body>Content</body>
	</html>`

	doc, _ := Parse(html)
	meta := doc.Metadata()

	fmt.Println(meta.Title)
	fmt.Println(meta.Description)
	if meta.OpenGraph != nil {
		fmt.Println(meta.OpenGraph.Image)
	}
	// Output:
	// My Page
	// A sample page
	// https://example.com/image.png
}

func ExampleDocument_Links() {
	html := `<html><body>
		<a href="https://example.com">Example Site</a>
		<a href="/about">About Us</a>
	</body></html>`

	doc, _ := Parse(html)
	links := doc.Links()

	for _, link := range links {
		fmt.Printf("%s -> %s\n", link.Text, link.URL)
	}
	// Output:
	// Example Site -> https://example.com
	// About Us -> /about
}

func ExampleDocument_FilteredLinks() {
	html := `<html><body>
		<a href="/page1">Internal Page 1</a>
		<a href="https://example.com/page2">Internal Page 2</a>
		<a href="https://other.com">External Site</a>
	</body></html>`

	doc, _ := Parse(html)

	// Get only internal links
	internal := doc.FilteredLinks(LinkFilter{
		BaseURL:  "https://example.com",
		Internal: true,
	})

	for _, link := range internal {
		fmt.Println(link.URL)
	}
	// Output:
	// https://example.com/page1
	// https://example.com/page2
}

func ExampleDocument_Images() {
	html := `<html><body>
		<img src="photo.jpg" alt="A photo">
		<img src="icon.png" alt="Icon">
	</body></html>`

	doc, _ := Parse(html)
	images := doc.Images()

	for _, img := range images {
		fmt.Printf("%s: %s\n", img.Alt, img.URL)
	}
	// Output:
	// A photo: photo.jpg
	// Icon: icon.png
}

func ExampleDocument_Transform() {
	html := `<html>
	<body>
		<nav><a href="/">Home</a></nav>
		<main>
			<h1>Article Title</h1>
			<p>Main content here.</p>
		</main>
		<footer>Copyright 2024</footer>
		<script>console.log("hi")</script>
	</body>
	</html>`

	doc, _ := Parse(html)

	// Extract only main content, exclude navigation and scripts
	clean := doc.Transform(&TransformOptions{
		OnlyMainContent: true,
	})

	// The output will contain only the main content area
	fmt.Println(strings.Contains(clean, "Article Title"))
	fmt.Println(strings.Contains(clean, "Main content"))
	fmt.Println(strings.Contains(clean, "<nav>"))
	fmt.Println(strings.Contains(clean, "<script>"))
	// Output:
	// true
	// true
	// false
	// false
}

func ExampleDocument_Transform_withFilters() {
	html := `<html><body>
		<div role="dialog">Cookie notice</div>
		<div class="modal-popup">Subscribe!</div>
		<p>Real content here</p>
	</body></html>`

	doc, _ := Parse(html)

	// Exclude modals and dialogs
	clean := doc.Transform(&TransformOptions{
		ExcludeFilters: []ElementFilter{
			{Attr: "role", AttrEquals: "dialog"},
			{Attr: "class", AttrContains: "modal"},
		},
	})

	fmt.Println(strings.Contains(clean, "Real content"))
	fmt.Println(strings.Contains(clean, "Cookie notice"))
	fmt.Println(strings.Contains(clean, "Subscribe"))
	// Output:
	// true
	// false
	// false
}

func ExampleDocument_Text() {
	html := `<html>
	<head>
		<title>Page Title</title>
		<script>alert('hi')</script>
	</head>
	<body>
		<h1>Hello</h1>
		<p>This is <strong>bold</strong> text.</p>
	</body>
	</html>`

	doc, _ := Parse(html)
	text := doc.Text()

	fmt.Println(text)
	// Output: Hello This is bold text.
}

func ExampleDocument_Branding() {
	html := `<html>
	<head>
		<meta name="theme-color" content="#0066cc">
		<link rel="icon" href="/favicon.ico">
	</head>
	<body>
		<img src="/logo.svg" class="site-logo" alt="Company Logo">
	</body>
	</html>`

	doc, _ := Parse(html)
	brand := doc.Branding()

	fmt.Println(brand.ThemeColor)
	fmt.Println(brand.Favicon)
	fmt.Println(brand.Logo)
	// Output:
	// #0066cc
	// /favicon.ico
	// /logo.svg
}

func ExampleElementFilter_Matches() {
	filter := ElementFilter{
		Attr:         "class",
		AttrContains: "modal",
	}

	// Test various elements
	fmt.Println(filter.Matches("div", map[string]string{"class": "modal-dialog"}))
	fmt.Println(filter.Matches("div", map[string]string{"class": "container"}))
	fmt.Println(filter.Matches("span", map[string]string{"class": "my-modal"}))
	// Output:
	// true
	// false
	// true
}

func ExampleStandardExcludeFilters() {
	html := `<html><body>
		<nav>Navigation</nav>
		<div role="dialog">Cookie banner</div>
		<script>alert('ad')</script>
		<p>Real content</p>
	</body></html>`

	doc, _ := Parse(html)

	// Use standard filters to clean content
	clean := doc.Transform(&TransformOptions{
		ExcludeFilters: StandardExcludeFilters,
	})

	fmt.Println(strings.Contains(clean, "Real content"))
	fmt.Println(strings.Contains(clean, "Navigation"))
	fmt.Println(strings.Contains(clean, "Cookie banner"))
	// Output:
	// true
	// false
	// false
}
