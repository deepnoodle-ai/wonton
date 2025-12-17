package htmltomd

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestConvert_PlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text", "Hello world", "Hello world"},
		{"text with whitespace", "  Hello   world  ", "Hello world"},
		{"multiline text", "Hello\nworld", "Hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Paragraphs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single paragraph", "<p>Hello world</p>", "Hello world"},
		{"multiple paragraphs", "<p>First</p><p>Second</p>", "First\n\nSecond"},
		{"paragraph with text", "Before<p>Inside</p>After", "Before\n\nInside\n\nAfter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Headings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"h1", "<h1>Title</h1>", "# Title"},
		{"h2", "<h2>Subtitle</h2>", "## Subtitle"},
		{"h3", "<h3>Section</h3>", "### Section"},
		{"h4", "<h4>Subsection</h4>", "#### Subsection"},
		{"h5", "<h5>Minor</h5>", "##### Minor"},
		{"h6", "<h6>Smallest</h6>", "###### Smallest"},
		{"heading with inline", "<h1>Hello <strong>world</strong></h1>", "# Hello **world**"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Bold(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strong", "<strong>bold</strong>", "**bold**"},
		{"b tag", "<b>bold</b>", "**bold**"},
		{"inline bold", "Hello <strong>world</strong>!", "Hello **world**!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Italic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"em", "<em>italic</em>", "*italic*"},
		{"i tag", "<i>italic</i>", "*italic*"},
		{"inline italic", "Hello <em>world</em>!", "Hello *world*!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_InlineCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"code tag", "<code>code</code>", "`code`"},
		{"inline code", "Use <code>fmt.Println</code> here", "Use `fmt.Println` here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Links(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple link", `<a href="https://example.com">Example</a>`, "[Example](https://example.com)"},
		{"link without href", `<a>No link</a>`, "No link"},
		{"link in text", `Check <a href="https://go.dev">Go</a> out`, "Check [Go](https://go.dev) out"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Images(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"image with alt", `<img src="img.png" alt="description">`, "![description](img.png)"},
		{"image without alt", `<img src="img.png">`, "![](img.png)"},
		{"image in text", `See <img src="icon.png" alt="icon"> here`, "See ![icon](icon.png) here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_UnorderedLists(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"simple list",
			"<ul><li>One</li><li>Two</li><li>Three</li></ul>",
			"- One\n- Two\n- Three",
		},
		{
			"nested list",
			"<ul><li>One<ul><li>Nested</li></ul></li><li>Two</li></ul>",
			"- One\n  - Nested\n- Two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_OrderedLists(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"simple list",
			"<ol><li>First</li><li>Second</li><li>Third</li></ol>",
			"1. First\n2. Second\n3. Third",
		},
		{
			"nested in unordered",
			"<ul><li>Item<ol><li>Sub one</li><li>Sub two</li></ol></li></ul>",
			"- Item\n  1. Sub one\n  2. Sub two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Blockquote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple quote", "<blockquote>Quote</blockquote>", "> Quote"},
		{"multiline quote", "<blockquote><p>Line 1</p><p>Line 2</p></blockquote>", "> Line 1\n>\n> Line 2"},
		{"nested quote", "<blockquote><blockquote>Nested</blockquote></blockquote>", "> > Nested"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_PreformattedCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"pre tag", "<pre>code block</pre>", "```\ncode block\n```"},
		{"pre with code", "<pre><code>func main() {}</code></pre>", "```\nfunc main() {}\n```"},
		{
			"pre with language class",
			`<pre><code class="language-go">func main() {}</code></pre>`,
			"```go\nfunc main() {}\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_HorizontalRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hr tag", "<hr>", "---"},
		{"hr with content", "<p>Above</p><hr><p>Below</p>", "Above\n\n---\n\nBelow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_LineBreaks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"br tag", "Line 1<br>Line 2", "Line 1\nLine 2"},
		{"br self-closing", "Line 1<br/>Line 2", "Line 1\nLine 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Tables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"simple table",
			"<table><tr><th>Name</th><th>Age</th></tr><tr><td>Alice</td><td>30</td></tr></table>",
			"| Name | Age |\n| --- | --- |\n| Alice | 30 |",
		},
		{
			"table without headers",
			"<table><tr><td>A</td><td>B</td></tr><tr><td>C</td><td>D</td></tr></table>",
			"| A | B |\n| --- | --- |\n| C | D |",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_HTMLEntities(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"amp", "Tom &amp; Jerry", "Tom & Jerry"},
		{"lt gt", "1 &lt; 2 &gt; 0", "1 < 2 > 0"},
		{"nbsp", "Hello&nbsp;world", "Hello world"},
		{"quote", "&quot;quoted&quot;", "\"quoted\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_MalformedHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"unclosed tag", "<p>Hello <b>world", "Hello **world**"},
		{"mismatched tags", "<div>Hello <b>world</div></b>", "Hello **world**"},
		{"no closing", "<p>Just text", "Just text"},
		{"weird nesting", "<b><i>text</b></i>", "***text***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Divs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"div ignored", "<div>Content</div>", "Content"},
		{"nested divs", "<div><div>Nested</div></div>", "Nested"},
		{"div with other elements", "<div><p>Para</p></div>", "Para"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_IgnoredElements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"script", "<p>Hello</p><script>alert('hi')</script><p>World</p>", "Hello\n\nWorld"},
		{"style", "<style>body{}</style><p>Content</p>", "Content"},
		{"head", "<html><head><title>Test</title></head><body><p>Body</p></body></html>", "Body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_FullDocument(t *testing.T) {
	input := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Welcome</h1>
<p>This is a <strong>test</strong> document.</p>
<ul>
<li>Item 1</li>
<li>Item 2</li>
</ul>
</body>
</html>`

	expected := `# Welcome

This is a **test** document.

- Item 1
- Item 2`

	result := Convert(input)
	assert.Equal(t, expected, result)
}

func TestConvertWithOptions_LinkStyle(t *testing.T) {
	input := `<a href="https://example.com">Example</a>`

	// Default inline style
	result := ConvertWithOptions(input, &Options{})
	assert.Equal(t, "[Example](https://example.com)", result)

	// Referenced style
	result = ConvertWithOptions(input, &Options{LinkStyle: LinkStyleReferenced})
	assert.Contains(t, result, "[Example][1]")
	assert.Contains(t, result, "[1]: https://example.com")
}

func TestConvertWithOptions_HeadingStyle(t *testing.T) {
	input := `<h1>Title</h1>`

	// Default ATX style
	result := ConvertWithOptions(input, &Options{})
	assert.Equal(t, "# Title", result)

	// Setext style
	result = ConvertWithOptions(input, &Options{HeadingStyle: HeadingStyleSetext})
	assert.Equal(t, "Title\n=====", result)

	input = `<h2>Subtitle</h2>`
	result = ConvertWithOptions(input, &Options{HeadingStyle: HeadingStyleSetext})
	assert.Equal(t, "Subtitle\n--------", result)
}

func TestConvertWithOptions_BulletChar(t *testing.T) {
	input := `<ul><li>One</li><li>Two</li></ul>`

	result := ConvertWithOptions(input, &Options{BulletChar: "*"})
	assert.Equal(t, "* One\n* Two", result)

	result = ConvertWithOptions(input, &Options{BulletChar: "+"})
	assert.Equal(t, "+ One\n+ Two", result)
}

func TestConvertWithOptions_CodeBlockStyle(t *testing.T) {
	input := `<pre><code>code</code></pre>`

	// Default fenced
	result := ConvertWithOptions(input, &Options{})
	assert.Equal(t, "```\ncode\n```", result)

	// Indented
	result = ConvertWithOptions(input, &Options{CodeBlockStyle: CodeBlockStyleIndented})
	assert.Equal(t, "    code", result)
}

func TestConvertWithOptions_SkipTags(t *testing.T) {
	input := `<p>Hello</p><nav>Navigation</nav><p>World</p>`

	result := ConvertWithOptions(input, &Options{SkipTags: []string{"nav"}})
	assert.Equal(t, "Hello\n\nWorld", result)
}

func TestConvert_Strikethrough(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"del tag", "<del>deleted</del>", "~~deleted~~"},
		{"s tag", "<s>strikethrough</s>", "~~strikethrough~~"},
		{"strike tag", "<strike>old</strike>", "~~old~~"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_NestedFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"bold in italic", "<em><strong>text</strong></em>", "***text***"},
		{"italic in bold", "<strong><em>text</em></strong>", "***text***"},
		{"code in bold", "<strong><code>code</code></strong>", "**`code`**"},
		{"link in bold", `<strong><a href="url">link</a></strong>`, "**[link](url)**"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_Span(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain span", "<span>text</span>", "text"},
		{"nested spans", "<span><span>text</span></span>", "text"},
		{"span in paragraph", "<p>Hello <span>world</span>!</p>", "Hello world!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_EmptyInput(t *testing.T) {
	assert.Equal(t, "", Convert(""))
	assert.Equal(t, "", Convert("   "))
	assert.Equal(t, "", Convert("\n\n\n"))
}

// Additional tests for malformed HTML and edge cases

func TestConvert_MalformedHTML_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"unclosed nested tags", "<div><p><b>text", "**text**"},
		{"reversed closing tags", "<b><i>text</b></i>", "***text***"},
		{"random closing tags", "text</b></i></div>", "text"},
		{"extra closing tags", "<p>text</p></p></p>", "text"},
		{"double open tags", "<p><p>text</p></p>", "text"},
		{"broken tag syntax", "<div text", ""},
		{"angle brackets in text", "1 < 2 > 0", "1 < 2 > 0"},
		{"lone angle brackets", "a < b and c > d", "a < b and c > d"},
		{"empty tags", "<p></p>", ""},
		{"empty nested tags", "<div><p><span></span></p></div>", ""},
		{"whitespace only tags", "<p>   </p>", ""},
		{"tabs and newlines", "<p>\t\n\t</p>", ""},
		{"self-closing div", "<div/>text", "text"},
		{"improperly self-closed p", "<p/>text", "text"},
		{"xhtml style br", "<br />", ""},
		{"mixed case tags", "<DIV><P>text</P></DIV>", "text"},
		{"mixed case with attributes", `<A HREF="url">link</A>`, "[link](url)"},
		{"missing quotes on attributes", `<a href=url>link</a>`, "[link](url)"},
		{"single quotes on attributes", `<a href='url'>link</a>`, "[link](url)"},
		{"multiple spaces between attributes", `<a    href="url"    title="t">link</a>`, `[link](url "t")`},
		{"newlines in tags", "<a\nhref=\"url\"\n>link</a>", "[link](url)"},
		{"tabs in tags", "<a\thref=\"url\">link</a>", "[link](url)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_HTMLComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple comment", "<!-- comment -->text", "text"},
		{"comment between elements", "<p>before</p><!-- comment --><p>after</p>", "before\n\nafter"},
		{"comment inside element", "<p>before<!-- comment -->after</p>", "beforeafter"},
		{"multiline comment", "text<!-- line1\nline2 -->more", "textmore"},
		{"unclosed comment", "text<!-- comment", "text"},
		{"empty comment", "text<!---->more", "textmore"},
		{"nested dashes in comment", "text<!-- -- -->more", "textmore"},
		{"conditional comment", "<!--[if IE]>IE only<![endif]-->text", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_EmptyElements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty paragraph", "<p></p>", ""},
		{"empty heading", "<h1></h1>", ""},
		{"empty bold", "<strong></strong>", ""},
		{"empty italic", "<em></em>", ""},
		{"empty link", `<a href="url"></a>`, ""},
		{"empty list", "<ul></ul>", ""},
		{"empty list item", "<ul><li></li></ul>", ""},
		{"empty blockquote", "<blockquote></blockquote>", ""},
		{"empty table", "<table></table>", ""},
		{"empty table row", "<table><tr></tr></table>", ""},
		{"empty code", "<code></code>", "``"},
		{"empty pre", "<pre></pre>", "```\n\n```"},
		{"whitespace-only bold", "<strong>   </strong>", ""},
		{"whitespace-only paragraph", "<p>   </p>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_SpecialCharactersInCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"backticks in inline code", "<code>foo`bar</code>", "`foo`bar`"},
		{"code with html entities", "<code>&lt;div&gt;</code>", "`<div>`"},
		{"code with ampersand", "<code>a &amp; b</code>", "`a & b`"},
		{"pre with backticks", "<pre>```go\ncode\n```</pre>", "```\n```go\ncode\n```\n```"},
		{"code with unicode", "<code>λ → α</code>", "`λ → α`"},
		{"code with newlines in pre", "<pre>line1\nline2\nline3</pre>", "```\nline1\nline2\nline3\n```"},
		{"code preserves whitespace", "<pre>    indented\n    more</pre>", "```\n    indented\n    more\n```"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_DeeplyNested(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"deeply nested divs",
			"<div><div><div><div><div>text</div></div></div></div></div>",
			"text",
		},
		{
			"deeply nested lists",
			"<ul><li>a<ul><li>b<ul><li>c<ul><li>d</li></ul></li></ul></li></ul></li></ul>",
			"- a\n  - b\n    - c\n      - d",
		},
		{
			"nested formatting",
			"<strong><em><strong><em>text</em></strong></em></strong>",
			"******text******",
		},
		{
			"nested blockquotes 3 deep",
			"<blockquote><blockquote><blockquote>deep</blockquote></blockquote></blockquote>",
			"> > > deep",
		},
		{
			"complex nesting",
			"<div><p><strong>Hello <em>nested <code>world</code></em></strong></p></div>",
			"**Hello *nested `world`***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_UnicodeAndEmoji(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"chinese text", "<p>你好世界</p>", "你好世界"},
		{"japanese text", "<p>こんにちは</p>", "こんにちは"},
		{"korean text", "<p>안녕하세요</p>", "안녕하세요"},
		{"arabic text", "<p>مرحبا</p>", "مرحبا"},
		{"emoji", "<p>Hello 👋 World 🌍</p>", "Hello 👋 World 🌍"},
		{"emoji in heading", "<h1>🚀 Launch</h1>", "# 🚀 Launch"},
		{"mixed scripts", "<p>Hello 你好 مرحبا</p>", "Hello 你好 مرحبا"},
		{"unicode math", "<p>∑(1/n²) = π²/6</p>", "∑(1/n²) = π²/6"},
		{"greek letters", "<p>α β γ δ ε</p>", "α β γ δ ε"},
		{"combining characters", "<p>café naïve résumé</p>", "café naïve résumé"},
		{"rtl text in ltr context", "<p>Hello مرحبا World</p>", "Hello مرحبا World"},
		{"zero-width chars", "<p>word\u200Bword</p>", "word\u200Bword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_HTMLEntities_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"named entities", "&copy; &reg; &trade;", "© ® ™"},
		{"numeric entities decimal", "&#169; &#174;", "© ®"},
		{"numeric entities hex", "&#x00A9; &#x00AE;", "© ®"},
		{"unknown entity", "&unknown;", "&unknown;"},
		{"incomplete entity", "Tom & Jerry", "Tom & Jerry"},
		{"double encoded", "&amp;amp;", "&amp;"},
		{"entity at end", "Hello&nbsp;", "Hello"},
		{"multiple nbsp", "Hello&nbsp;&nbsp;&nbsp;World", "Hello World"},
		{"mixed entities", "&lt;div class=&quot;test&quot;&gt;", "<div class=\"test\">"},
		{"apos entity", "&apos;quoted&apos;", "'quoted'"},
		{"mdash and ndash", "2020&ndash;2023 &mdash; long", "2020–2023 — long"},
		{"bullet entity", "&bull; item", "• item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_TableEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"uneven columns",
			"<table><tr><td>A</td><td>B</td><td>C</td></tr><tr><td>D</td></tr></table>",
			"| A | B | C |\n| --- | --- | --- |\n| D |  |  |",
		},
		{
			"with thead and tbody",
			"<table><thead><tr><th>H1</th></tr></thead><tbody><tr><td>D1</td></tr></tbody></table>",
			"| H1 |\n| --- |\n| D1 |",
		},
		{
			"formatted content in cells",
			"<table><tr><td><strong>bold</strong></td><td><em>italic</em></td></tr></table>",
			"| **bold** | *italic* |\n| --- | --- |",
		},
		{
			"links in table cells",
			`<table><tr><td><a href="url">link</a></td></tr></table>`,
			"| [link](url) |\n| --- |",
		},
		{
			"whitespace in cells",
			"<table><tr><td>  spaced  </td></tr></table>",
			"| spaced |\n| --- |",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_ListEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"orphan li element",
			"<li>Orphan item</li>",
			"- Orphan item",
		},
		{
			"list with paragraphs",
			"<ul><li><p>Item 1</p></li><li><p>Item 2</p></li></ul>",
			"- Item 1\n- Item 2",
		},
		{
			"mixed list types nested",
			"<ol><li>First<ul><li>Nested bullet</li></ul></li><li>Second</li></ol>",
			"1. First\n  - Nested bullet\n2. Second",
		},
		{
			"deeply nested mixed",
			"<ul><li>A<ol><li>1<ul><li>x</li></ul></li></ol></li></ul>",
			"- A\n  1. 1\n    - x",
		},
		{
			"list with inline formatting",
			"<ul><li><strong>Bold</strong> item</li><li>Item with <code>code</code></li></ul>",
			"- **Bold** item\n- Item with `code`",
		},
		{
			"list starting at different number",
			`<ol start="5"><li>Five</li><li>Six</li></ol>`,
			"1. Five\n2. Six",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_BlockquoteEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"blockquote with list",
			"<blockquote><ul><li>item</li></ul></blockquote>",
			"> - item",
		},
		{
			"blockquote with code",
			"<blockquote><pre>code</pre></blockquote>",
			"> ```\n> code\n> ```",
		},
		{
			"blockquote with heading",
			"<blockquote><h2>Title</h2><p>Text</p></blockquote>",
			"> ## Title\n>\n> Text",
		},
		{
			"multiple paragraphs in blockquote",
			"<blockquote><p>Para 1</p><p>Para 2</p><p>Para 3</p></blockquote>",
			"> Para 1\n>\n> Para 2\n>\n> Para 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_LinkEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"link with query params", `<a href="https://example.com?a=1&b=2">link</a>`, "[link](https://example.com?a=1&b=2)"},
		{"link with fragment", `<a href="https://example.com#section">link</a>`, "[link](https://example.com#section)"},
		{"link with spaces in url", `<a href="https://example.com/path%20with%20spaces">link</a>`, "[link](https://example.com/path%20with%20spaces)"},
		{"mailto link", `<a href="mailto:test@example.com">email</a>`, "[email](mailto:test@example.com)"},
		{"tel link", `<a href="tel:+1234567890">call</a>`, "[call](tel:+1234567890)"},
		{"javascript link ignored", `<a href="javascript:void(0)">click</a>`, "[click](javascript:void(0))"},
		{"empty href", `<a href="">link</a>`, "link"},
		{"hash only href", `<a href="#">link</a>`, "[link](#)"},
		{"relative url", `<a href="/path/to/page">link</a>`, "[link](/path/to/page)"},
		{"link with title", `<a href="url" title="tooltip">link</a>`, `[link](url "tooltip")`},
		{"nested elements in link", `<a href="url"><strong>bold link</strong></a>`, "[**bold link**](url)"},
		{"image in link", `<a href="url"><img src="img.png" alt="img"></a>`, "[![img](img.png)](url)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_HTML5SemanticElements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"article", "<article><h1>Title</h1><p>Content</p></article>", "# Title\n\nContent"},
		{"section", "<section><h2>Section</h2><p>Text</p></section>", "## Section\n\nText"},
		{"aside", "<aside><p>Sidebar</p></aside>", "Sidebar"},
		{"figure with figcaption", "<figure><img src=\"img.png\" alt=\"image\"><figcaption>Caption</figcaption></figure>", "![image](img.png)\n\nCaption"},
		{"details and summary", "<details><summary>More info</summary><p>Hidden content</p></details>", "More info\n\nHidden content"},
		{"mark element", "<p>This is <mark>highlighted</mark></p>", "This is highlighted"},
		{"time element", "<p>Posted <time datetime=\"2024-01-01\">January 1st</time></p>", "Posted January 1st"},
		{"abbr element", `<p>The <abbr title="World Wide Web">WWW</abbr></p>`, "The WWW"},
		{"address element", "<address>123 Street</address>", "123 Street"},
		{"cite element", "<p>From <cite>The Book</cite></p>", "From The Book"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_RealWorldHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"blog post structure",
			`<article>
<header>
<h1>Blog Title</h1>
</header>
<p>2024-01-01</p>
<p>First paragraph.</p>
<p>Second paragraph with <a href="https://example.com">link</a>.</p>
<footer><p>Author: John</p></footer>
</article>`,
			"# Blog Title\n\n2024-01-01\n\nFirst paragraph.\n\nSecond paragraph with [link](https://example.com).\n\nAuthor: John",
		},
		{
			"navigation stripped",
			`<nav><a href="/">Home</a> <a href="/about">About</a></nav><main><p>Content</p></main>`,
			"[Home](/) [About](/about)\n\nContent",
		},
		{
			"complex document",
			`<!DOCTYPE html>
			<html>
			<head><title>Page</title><style>body{}</style></head>
			<body>
				<header><h1>Welcome</h1></header>
				<main>
					<p>Hello <strong>world</strong>!</p>
					<ul><li>One</li><li>Two</li></ul>
				</main>
				<script>console.log("hi");</script>
			</body>
			</html>`,
			"# Welcome\n\nHello **world**!\n\n- One\n- Two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvert_ReferencedLinkDeduplication(t *testing.T) {
	input := `<p><a href="https://example.com">First</a> and <a href="https://example.com">Second</a> and <a href="https://other.com">Third</a></p>`
	result := ConvertWithOptions(input, &Options{LinkStyle: LinkStyleReferenced})

	// Same URL should reuse the same reference number
	assert.Contains(t, result, "[First][1]")
	assert.Contains(t, result, "[Second][1]")
	assert.Contains(t, result, "[Third][2]")

	// References should appear at end
	assert.Contains(t, result, "[1]: https://example.com")
	assert.Contains(t, result, "[2]: https://other.com")
}

func TestConvert_ReferencedLinkDifferentTitles(t *testing.T) {
	// Same URL but different titles should create separate references
	input := `<p><a href="https://example.com" title="Foo">First</a> and <a href="https://example.com" title="Bar">Second</a></p>`
	result := ConvertWithOptions(input, &Options{LinkStyle: LinkStyleReferenced})

	// Different titles should create different reference numbers
	assert.Contains(t, result, "[First][1]")
	assert.Contains(t, result, "[Second][2]")

	// Both references should appear with their respective titles
	assert.Contains(t, result, `[1]: https://example.com "Foo"`)
	assert.Contains(t, result, `[2]: https://example.com "Bar"`)
}

func TestConvert_NilOptions(t *testing.T) {
	result := ConvertWithOptions("<p>text</p>", nil)
	assert.Equal(t, "text", result)
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	assert.Equal(t, LinkStyleInline, opts.LinkStyle)
	assert.Equal(t, HeadingStyleATX, opts.HeadingStyle)
	assert.Equal(t, CodeBlockStyleFenced, opts.CodeBlockStyle)
	assert.Equal(t, "-", opts.BulletChar)
	assert.Nil(t, opts.SkipTags)
}
