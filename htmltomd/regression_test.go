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
