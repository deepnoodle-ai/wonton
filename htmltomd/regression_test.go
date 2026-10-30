package htmltomd

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// BUG-033: indentation and blank lines inside fenced code blocks survive.
func TestCodeBlockIndentationPreserved(t *testing.T) {
	got := Convert("<pre><code>{\n  \"a\": 1,\n   \"b\": [\n\n\n    2\n  ]\n}</code></pre>")
	assert.Equal(t, "```\n{\n  \"a\": 1,\n   \"b\": [\n\n\n    2\n  ]\n}\n```", got)

	// Outside code blocks, stray indentation is still trimmed.
	got = Convert("<p>  text</p>")
	assert.Equal(t, "text", got)
}

// BUG-072: <tfoot> rows are kept, and an empty leading row does not demote
// the header row.
func TestTableFooterAndEmptyRow(t *testing.T) {
	got := Convert("<table><thead><tr><th>H</th></tr></thead><tbody><tr><td>B</td></tr></tbody><tfoot><tr><td>F</td></tr></tfoot></table>")
	assert.Contains(t, got, "| H |")
	assert.Contains(t, got, "| B |")
	assert.Contains(t, got, "| F |")

	got = Convert("<table><tr></tr><tr><th>H</th></tr><tr><td>B</td></tr></table>")
	lines := strings.Split(got, "\n")
	assert.Equal(t, "| H |", lines[0])
}

// BUG-072: a list nested directly in a list (no wrapping <li>) is kept.
func TestSiblingNestedList(t *testing.T) {
	got := Convert("<ul><li>One</li><ul><li>Nested</li></ul><li>Two</li></ul>")
	assert.Equal(t, "- One\n  - Nested\n- Two", got)
}

// BUG-072: quotes in titles are escaped, and newlines in alt text collapse.
func TestLinkAndImageTitlesWithQuotes(t *testing.T) {
	got := Convert(`<a href="/x" title='say "hi"'>link</a>`)
	assert.Equal(t, `[link](/x "say \"hi\"")`, got)

	got = Convert(`<img src="/i.png" alt="line one
line two" title='a "b"'>`)
	assert.Equal(t, `![line one line two](/i.png "a \"b\"")`, got)

	got = ConvertWithOptions(`<a href="/x" title='q "t"'>l</a>`, &Options{LinkStyle: LinkStyleReferenced})
	assert.Contains(t, got, `[1]: /x "q \"t\""`)
}

// H24: ConvertWithOptions does not modify the caller's Options, so one
// Options value can be shared across goroutines.
func TestConvertWithOptionsDoesNotMutateOptions(t *testing.T) {
	opts := &Options{}
	got := ConvertWithOptions("<ul><li>a</li></ul>", opts)
	assert.Equal(t, got, "- a")
	assert.Equal(t, opts.BulletChar, "")
}

// M38: whitespace at the edge of inline elements moves outside the
// delimiters instead of being dropped, without doubling spaces.
func TestInlineEdgeWhitespace(t *testing.T) {
	tests := []struct{ input, want string }{
		{"<p>Hello<strong> world</strong></p>", "Hello **world**"},
		{"<p><em>it </em>works</p>", "*it* works"},
		{"<p>a<del> b </del>c</p>", "a ~~b~~ c"},
		{"<p>See<a href=\"/x\"> link </a>here</p>", "See [link](/x) here"},
		{"<p>Hello <b> world</b> and <i>x </i> y</p>", "Hello **world** and *x* y"},
		{"<p>a<b> </b>b</p>", "a b"},
		{"<p>Hello <b> </b>world</p>", "Hello world"},
		{"<p><em>x </em><strong> y</strong></p>", "*x* **y**"},
		{"<p>Hello <span><b> world</b></span></p>", "Hello **world**"},
	}
	for _, tt := range tests {
		assert.Equal(t, Convert(tt.input), tt.want, tt.input)
	}
}

// M38: a <br> inside a heading does not split the heading line.
func TestHeadingLineBreak(t *testing.T) {
	assert.Equal(t, Convert("<h1>Title<br>Subtitle</h1>"), "# Title Subtitle")
}
