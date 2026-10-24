package htmlparse

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// BUG-033: an inline SVG <title> does not replace the page title.
func TestMetadataIgnoresSVGTitle(t *testing.T) {
	doc, err := Parse(`<html><head><title>Page</title></head><body><svg><title>Icon</title></svg></body></html>`)
	assert.NoError(t, err)
	assert.Equal(t, "Page", doc.Metadata().Title)

	doc, err = Parse(`<html><body><svg><title>Icon</title></svg></body></html>`)
	assert.NoError(t, err)
	assert.Equal(t, "", doc.Metadata().Title)
}

// BUG-033: non-http links are not internal.
func TestFilteredLinksInternalExcludesNonHTTPSchemes(t *testing.T) {
	doc, err := Parse(`<a href="/a">a</a><a href="mailto:x@other.com">m</a><a href="tel:123">t</a>` +
		`<a href="javascript:void(0)">j</a><a href="data:text/plain,hi">d</a><a href="https://other.com/b">b</a>`)
	assert.NoError(t, err)

	internal := doc.FilteredLinks(LinkFilter{BaseURL: "https://example.com", Internal: true})
	var urls []string
	for _, l := range internal {
		urls = append(urls, l.URL)
	}
	assert.Equal(t, []string{"https://example.com/a"}, urls)
}

// BUG-033: Transform with Include emits only text inside included elements.
func TestTransformIncludeDropsOutsideText(t *testing.T) {
	doc, err := Parse(`<html><body><div>outside <p>inside <b>bold</b></p> tail</div></body></html>`)
	assert.NoError(t, err)

	compact := doc.Transform(&TransformOptions{Include: []string{"p"}})
	assert.Contains(t, compact, "inside")
	assert.Contains(t, compact, "bold")
	assert.False(t, strings.Contains(compact, "outside"), "compact output: %q", compact)
	assert.False(t, strings.Contains(compact, "tail"), "compact output: %q", compact)

	pretty := doc.Transform(&TransformOptions{Include: []string{"p"}, PrettyPrint: true})
	assert.Contains(t, pretty, "inside")
	assert.False(t, strings.Contains(pretty, "outside"), "pretty output: %q", pretty)
}
