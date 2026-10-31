package htmltomd

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// convertCase is one HTML input and the Markdown it must produce.
type convertCase struct {
	name string // subtest name; the HTML is used when empty
	html string
	opts *Options // nil means default options
	want string
}

func runCases(t *testing.T, cases []convertCase) {
	t.Helper()
	for _, tc := range cases {
		name := tc.name
		if name == "" {
			name = tc.html
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, ConvertWithOptions(tc.html, tc.opts), tc.want)
		})
	}
}

// Plain text, paragraphs, line breaks, entities, and Unicode.
func TestConvert_TextAndWhitespace(t *testing.T) {
	runCases(t, []convertCase{
		{name: "plain text", html: "Hello world", want: "Hello world"},
		{name: "text with whitespace", html: "  Hello   world  ", want: "Hello world"},
		{name: "multiline text", html: "Hello\nworld", want: "Hello world"},
		{name: "single paragraph", html: "<p>Hello world</p>", want: "Hello world"},
		{name: "multiple paragraphs", html: "<p>First</p><p>Second</p>", want: "First\n\nSecond"},
		{name: "paragraph with text", html: "Before<p>Inside</p>After", want: "Before\n\nInside\n\nAfter"},
		{name: "hr tag", html: "<hr>", want: "---"},
		{name: "hr with content", html: "<p>Above</p><hr><p>Below</p>", want: "Above\n\n---\n\nBelow"},
		{name: "br tag", html: "Line 1<br>Line 2", want: "Line 1\nLine 2"},
		{name: "br self-closing", html: "Line 1<br/>Line 2", want: "Line 1\nLine 2"},
		{name: "amp", html: "Tom &amp; Jerry", want: "Tom & Jerry"},
		{name: "lt gt", html: "1 &lt; 2 &gt; 0", want: "1 < 2 > 0"},
		{name: "nbsp", html: "Hello&nbsp;world", want: "Hello world"},
		{name: "quote", html: "&quot;quoted&quot;", want: "\"quoted\""},
		{name: "plain span", html: "<span>text</span>", want: "text"},
		{name: "nested spans", html: "<span><span>text</span></span>", want: "text"},
		{name: "span in paragraph", html: "<p>Hello <span>world</span>!</p>", want: "Hello world!"},
		{name: "chinese text", html: "<p>你好世界</p>", want: "你好世界"},
		{name: "japanese text", html: "<p>こんにちは</p>", want: "こんにちは"},
		{name: "korean text", html: "<p>안녕하세요</p>", want: "안녕하세요"},
		{name: "arabic text", html: "<p>مرحبا</p>", want: "مرحبا"},
		{name: "emoji", html: "<p>Hello 👋 World 🌍</p>", want: "Hello 👋 World 🌍"},
		{name: "emoji in heading", html: "<h1>🚀 Launch</h1>", want: "# 🚀 Launch"},
		{name: "mixed scripts", html: "<p>Hello 你好 مرحبا</p>", want: "Hello 你好 مرحبا"},
		{name: "unicode math", html: "<p>∑(1/n²) = π²/6</p>", want: "∑(1/n²) = π²/6"},
		{name: "greek letters", html: "<p>α β γ δ ε</p>", want: "α β γ δ ε"},
		{name: "combining characters", html: "<p>café naïve résumé</p>", want: "café naïve résumé"},
		{name: "rtl text in ltr context", html: "<p>Hello مرحبا World</p>", want: "Hello مرحبا World"},
		{name: "zero-width chars", html: "<p>word\u200Bword</p>", want: "word\u200Bword"},
		{name: "named entities", html: "&copy; &reg; &trade;", want: "© ® ™"},
		{name: "numeric entities decimal", html: "&#169; &#174;", want: "© ®"},
		{name: "numeric entities hex", html: "&#x00A9; &#x00AE;", want: "© ®"},
		{name: "unknown entity", html: "&unknown;", want: "&unknown;"},
		{name: "incomplete entity", html: "Tom & Jerry", want: "Tom & Jerry"},
		{name: "double encoded", html: "&amp;amp;", want: "&amp;"},
		{name: "entity at end", html: "Hello&nbsp;", want: "Hello"},
		{name: "multiple nbsp", html: "Hello&nbsp;&nbsp;&nbsp;World", want: "Hello World"},
		{name: "mixed entities", html: "&lt;div class=&quot;test&quot;&gt;", want: "<div class=\"test\">"},
		{name: "apos entity", html: "&apos;quoted&apos;", want: "'quoted'"},
		{name: "mdash and ndash", html: "2020&ndash;2023 &mdash; long", want: "2020–2023 — long"},
		{name: "bullet entity", html: "&bull; item", want: "• item"},
		// Seen on docs.python.org/3/library/json.html.
		{
			name: "inter tag newlines do not trigger indented code block",
			html: `<section>
<dl class="py function">
<dd>
<div><p>Previous description.</p>
</div>
</dd></dl>

<dl class="py function">
<dt class="sig sig-object py" id="json.dumps">
<span class="sig-prename">json.</span><span class="sig-name">dumps</span>(<em>obj</em>)
</dt>
</dl>
</section>`,
			want: "Previous description.\n\njson.dumps(*obj*)",
		},
		{name: "empty input", html: "", want: ""},
		{name: "whitespace-only input", html: "   ", want: ""},
		{name: "newline-only input", html: "\n\n\n", want: ""},
	})
}

// Headings in both styles.
func TestConvert_Headings(t *testing.T) {
	runCases(t, []convertCase{
		{name: "h1", html: "<h1>Title</h1>", want: "# Title"},
		{name: "h2", html: "<h2>Subtitle</h2>", want: "## Subtitle"},
		{name: "h3", html: "<h3>Section</h3>", want: "### Section"},
		{name: "h4", html: "<h4>Subsection</h4>", want: "#### Subsection"},
		{name: "h5", html: "<h5>Minor</h5>", want: "##### Minor"},
		{name: "h6", html: "<h6>Smallest</h6>", want: "###### Smallest"},
		{name: "heading with inline", html: "<h1>Hello <strong>world</strong></h1>", want: "# Hello **world**"},
		{
			name: "setext heading unicode underline length",
			html: `<h1>日本語</h1>`,
			opts: &Options{HeadingStyle: HeadingStyleSetext},
			want: "日本語\n======",
		},
		// Seen on docs.python.org, whose headings end in a ¶ permalink.
		{
			name: "heading permalink anchor dropped",
			html: `<h2 id="x">Title<a class="headerlink" href="#x">¶</a></h2>`,
			want: "## Title",
		},
		{name: "br inside heading becomes a space", html: "<h1>Title<br>Subtitle</h1>", want: "# Title Subtitle"},
		{
			name: "setext h1",
			html: "<h1>Title</h1>",
			opts: &Options{HeadingStyle: HeadingStyleSetext},
			want: "Title\n=====",
		},
		{
			name: "setext h2",
			html: "<h2>Subtitle</h2>",
			opts: &Options{HeadingStyle: HeadingStyleSetext},
			want: "Subtitle\n--------",
		},
	})
}

// Bold, italic, strikethrough, quotes, superscript, and their edge whitespace.
func TestConvert_InlineFormatting(t *testing.T) {
	runCases(t, []convertCase{
		{name: "strong", html: "<strong>bold</strong>", want: "**bold**"},
		{name: "b tag", html: "<b>bold</b>", want: "**bold**"},
		{name: "inline bold", html: "Hello <strong>world</strong>!", want: "Hello **world**!"},
		{name: "em", html: "<em>italic</em>", want: "*italic*"},
		{name: "i tag", html: "<i>italic</i>", want: "*italic*"},
		{name: "inline italic", html: "Hello <em>world</em>!", want: "Hello *world*!"},
		{name: "del tag", html: "<del>deleted</del>", want: "~~deleted~~"},
		{name: "s tag", html: "<s>strikethrough</s>", want: "~~strikethrough~~"},
		{name: "strike tag", html: "<strike>old</strike>", want: "~~old~~"},
		{name: "bold in italic", html: "<em><strong>text</strong></em>", want: "***text***"},
		{name: "italic in bold", html: "<strong><em>text</em></strong>", want: "***text***"},
		{name: "code in bold", html: "<strong><code>code</code></strong>", want: "**`code`**"},
		{name: "link in bold", html: `<strong><a href="url">link</a></strong>`, want: "**[link](url)**"},
		{
			name: "nested formatting",
			html: "<strong><em><strong><em>text</em></strong></em></strong>",
			want: "***text***",
		},
		{
			name: "complex nesting",
			html: "<div><p><strong>Hello <em>nested <code>world</code></em></strong></p></div>",
			want: "**Hello *nested `world`***",
		},
		{html: "<p>Hello<strong> world</strong></p>", want: "Hello **world**"},
		{html: "<p><em>it </em>works</p>", want: "*it* works"},
		{html: "<p>a<del> b </del>c</p>", want: "a ~~b~~ c"},
		{html: "<p>See<a href=\"/x\"> link </a>here</p>", want: "See [link](/x) here"},
		{html: "<p>Hello <b> world</b> and <i>x </i> y</p>", want: "Hello **world** and *x* y"},
		{html: "<p>a<b> </b>b</p>", want: "a b"},
		{html: "<p>Hello <b> </b>world</p>", want: "Hello world"},
		{html: "<p><em>x </em><strong> y</strong></p>", want: "*x* **y**"},
		{html: "<p>Hello <span><b> world</b></span></p>", want: "Hello **world**"},
		// Seen on docs.python.org/3/library/json.html.
		{
			name: "em wrapping literal asterisk does not produce triple star",
			html: `<p>json.dump(<em>obj</em>, <em>*</em>, <em>skipkeys=False</em>)</p>`,
			want: `json.dump(*obj*, \*, *skipkeys=False*)`,
		},
		{
			name: "nested identical bold tags do not cancel out",
			html: `<p><b>Outer <b>inner</b> outer</b></p>`,
			want: "**Outer inner outer**",
		},
		{
			name: "br inside strong preserves line break",
			html: `<p><strong>Line 1<br>Line 2</strong></p>`,
			want: "**Line 1**\n**Line 2**",
		},
		{
			name: "inline q adds quotes",
			html: `<p>He said <q>hello world</q> to the room.</p>`,
			want: `He said "hello world" to the room.`,
		},
		{
			name: "br inside em preserves line break",
			html: `<p><em>Line 1<br>Line 2</em> after</p>`,
			want: "*Line 1*\n*Line 2* after",
		},
		// Seen on docs.python.org signatures such as **kw.
		{name: "em starting with asterisks escapes them", html: `<p><em>**kw</em></p>`, want: `*\*\*kw*`},
		{
			name: "code inside bold is not escaped",
			html: `<p><strong>bold <code>a*b</code></strong></p>`,
			want: "**bold `a*b`**",
		},
		// Seen on martinfowler.com.
		{
			name: "sup and sub kept as inline html",
			html: `<p>E = mc<sup>2</sup> and H<sub>2</sub>O</p>`,
			want: "E = mc<sup>2</sup> and H<sub>2</sub>O",
		},
	})
}

// Inline code and code blocks.
func TestConvert_Code(t *testing.T) {
	runCases(t, []convertCase{
		{name: "code tag", html: "<code>code</code>", want: "`code`"},
		{name: "inline code", html: "Use <code>fmt.Println</code> here", want: "Use `fmt.Println` here"},
		{name: "pre tag", html: "<pre>code block</pre>", want: "```\ncode block\n```"},
		{name: "pre with code", html: "<pre><code>func main() {}</code></pre>", want: "```\nfunc main() {}\n```"},
		{
			name: "pre with language class",
			html: `<pre><code class="language-go">func main() {}</code></pre>`,
			want: "```go\nfunc main() {}\n```",
		},
		{name: "backticks in inline code", html: "<code>foo`bar</code>", want: "``foo`bar``"},
		{name: "code starting with backtick", html: "<code>`quoted`</code>", want: "`` `quoted` ``"},
		{name: "code with html entities", html: "<code>&lt;div&gt;</code>", want: "`<div>`"},
		{name: "code with ampersand", html: "<code>a &amp; b</code>", want: "`a & b`"},
		{name: "pre with backticks", html: "<pre>```go\ncode\n```</pre>", want: "````\n```go\ncode\n```\n````"},
		{name: "code with unicode", html: "<code>λ → α</code>", want: "`λ → α`"},
		{
			name: "code with newlines in pre",
			html: "<pre>line1\nline2\nline3</pre>",
			want: "```\nline1\nline2\nline3\n```",
		},
		{
			name: "code preserves whitespace",
			html: "<pre>    indented\n    more</pre>",
			want: "```\n    indented\n    more\n```",
		},
		{
			name: "language class among other classes",
			html: `<pre><code class="hljs language-go">x := 1</code></pre>`,
			want: "```go\nx := 1\n```",
		},
		{
			name: "lang- prefix",
			html: `<pre><code class="lang-python">x = 1</code></pre>`,
			want: "```python\nx = 1\n```",
		},
		{
			name: "language class on pre element",
			html: `<pre class="language-rust"><code>let x = 1;</code></pre>`,
			want: "```rust\nlet x = 1;\n```",
		},
		{
			name: "indented code block preserves multiple blank lines",
			html: "<pre><code>line 1\n\n\nline 2</code></pre>",
			opts: &Options{CodeBlockStyle: CodeBlockStyleIndented},
			want: "    line 1\n\n\n    line 2",
		},
		// Seen on pkg.go.dev/net/http and doc.rust-lang.org.
		{
			name: "links inside pre render as plain text",
			html: `<pre><code>// See <a href="https://rfc-editor.org/rfc/rfc9110.html">RFC 9110</a> for details.</code></pre>`,
			want: "```\n// See RFC 9110 for details.\n```",
		},
		// Seen on pkg.go.dev/net/http.
		{
			name: "code wrapping single link inverts to linked code",
			html: `<p>Use <code><a href="https://pkg.go.dev/net/http#Client">http.Client</a></code> for requests.</p>`,
			want: "Use [`http.Client`](https://pkg.go.dev/net/http#Client) for requests.",
		},
		// Seen on doc.rust-lang.org/std/vec/struct.Vec.html.
		{
			name: "code with mixed text and multiple links strips inner link syntax",
			html: `<p>When <code><a href="../mem/fn.size_of.html">size_of::&lt;T&gt;</a>() * <a href="#method.capacity">capacity</a>() &gt; 0</code> holds.</p>`,
			want: "When `size_of::<T>() * capacity() > 0` holds.",
		},
		{name: "empty code tag produces nothing", html: `<p>Before <code></code> after.</p>`, want: "Before after."},
		// Seen on lwn.net/Articles/.
		{
			name: "tt tag treated as inline code",
			html: `<p>Address <tt>` + "`touch${IFS}/tmp/pwn`" + `@attacker.example</tt> failed.</p>`,
			want: "Address `` `touch${IFS}/tmp/pwn`@attacker.example `` failed.",
		},
		// Seen on developer.mozilla.org.
		{
			name: "kbd tag treated as inline code",
			html: `<p>Press <kbd>Ctrl+C</kbd> to stop.</p>`,
			want: "Press `Ctrl+C` to stop.",
		},
		{
			name: "pre data lang attribute detected",
			html: `<pre data-lang="rust"><code>fn main() {}</code></pre>`,
			want: "```rust\nfn main() {}\n```",
		},
		// Seen on docs.python.org.
		{
			name: "sphinx highlight wrapper language",
			html: `<div class="highlight-python notranslate"><div class="highlight"><pre>x = 1</pre></div></div>`,
			want: "```python\nx = 1\n```",
		},
		// Seen on github.com.
		{
			name: "github highlight source language",
			html: `<pre class="highlight-source-go">x</pre>`,
			want: "```go\nx\n```",
		},
		{
			name: "fenced code block keeps indentation and blank lines",
			html: "<pre><code>{\n  \"a\": 1,\n   \"b\": [\n\n\n    2\n  ]\n}</code></pre>",
			want: "```\n{\n  \"a\": 1,\n   \"b\": [\n\n\n    2\n  ]\n}\n```",
		},
		{name: "indentation outside code is trimmed", html: "<p>  text</p>", want: "text"},
		{
			name: "indented code block style",
			html: "<pre><code>code</code></pre>",
			opts: &Options{CodeBlockStyle: CodeBlockStyleIndented},
			want: "    code",
		},
	})
}

// Links, images, URL handling, and link styles.
func TestConvert_LinksAndImages(t *testing.T) {
	runCases(t, []convertCase{
		{
			name: "simple link",
			html: `<a href="https://example.com">Example</a>`,
			want: "[Example](https://example.com)",
		},
		{name: "link without href", html: `<a>No link</a>`, want: "No link"},
		{
			name: "link in text",
			html: `Check <a href="https://go.dev">Go</a> out`,
			want: "Check [Go](https://go.dev) out",
		},
		{name: "image with alt", html: `<img src="img.png" alt="description">`, want: "![description](img.png)"},
		{name: "image without alt", html: `<img src="img.png">`, want: "![](img.png)"},
		{name: "image in text", html: `See <img src="icon.png" alt="icon"> here`, want: "See ![icon](icon.png) here"},
		{
			name: "link with query params",
			html: `<a href="https://example.com?a=1&b=2">link</a>`,
			want: "[link](https://example.com?a=1&b=2)",
		},
		{
			name: "link with fragment",
			html: `<a href="https://example.com#section">link</a>`,
			want: "[link](https://example.com#section)",
		},
		{
			name: "link with spaces in url",
			html: `<a href="https://example.com/path%20with%20spaces">link</a>`,
			want: "[link](https://example.com/path%20with%20spaces)",
		},
		{
			name: "mailto link",
			html: `<a href="mailto:test@example.com">email</a>`,
			want: "[email](mailto:test@example.com)",
		},
		{name: "tel link", html: `<a href="tel:+1234567890">call</a>`, want: "[call](tel:+1234567890)"},
		{name: "javascript link ignored", html: `<a href="javascript:void(0)">click</a>`, want: "click"},
		{name: "empty href", html: `<a href="">link</a>`, want: "link"},
		{name: "hash only href", html: `<a href="#">link</a>`, want: "[link](#)"},
		{name: "relative url", html: `<a href="/path/to/page">link</a>`, want: "[link](/path/to/page)"},
		{name: "link with title", html: `<a href="url" title="tooltip">link</a>`, want: `[link](url "tooltip")`},
		{
			name: "nested elements in link",
			html: `<a href="url"><strong>bold link</strong></a>`,
			want: "[**bold link**](url)",
		},
		{
			name: "image in link",
			html: `<a href="url"><img src="img.png" alt="img"></a>`,
			want: "[![img](img.png)](url)",
		},
		// Seen on danluu.com.
		{
			name: "link text with brackets escaped",
			html: `<p><a href="https://danluu.com/why-video/">Why do people post on [bad platform] instead of [good platform]?</a></p>`,
			want: `[Why do people post on \[bad platform\] instead of \[good platform\]?](https://danluu.com/why-video/)`,
		},
		{
			name: "link url with spaces encoded",
			html: `<p><a href="/files/annual report 2026.pdf">Annual Report</a></p>`,
			want: "[Annual Report](/files/annual%20report%202026.pdf)",
		},
		{
			name: "javascript void link renders text only",
			html: `<p><a href="javascript:void(0)">Expand section</a></p>`,
			want: "Expand section",
		},
		{
			name: "empty img without src or alt skipped",
			html: `<p>Text before <img> text after.</p>`,
			want: "Text before text after.",
		},
		{
			name: "lazy loaded img falls back to data src",
			html: `<img data-src="https://example.com/diagram.png" alt="Architecture">`,
			want: "![Architecture](https://example.com/diagram.png)",
		},
		{
			name: "card link wrapping heading and paragraph",
			html: `<a href="/post/1"><div><h3>Post Title</h3><p>Post summary text</p></div></a>`,
			want: "### [Post Title](/post/1)\n\nPost summary text",
		},
		// Seen on en.wikipedia.org.
		{
			name: "link url with unbalanced parens encoded",
			html: `<a href="/files/report (final.pdf">R</a> <a href="https://en.wikipedia.org/wiki/Convention_(norm)">C</a>`,
			want: "[R](/files/report%20%28final.pdf) [C](https://en.wikipedia.org/wiki/Convention_(norm))",
		},
		{
			name: "obfuscated javascript scheme renders text only",
			html: "<a href=\" JaVa\tScript:alert(1)\">x</a>",
			want: "x",
		},
		// Seen on danluu.com.
		{
			name: "image alt brackets escaped inside link",
			html: `<a href="/x"><img alt="[logo]" src="l.png"></a>`,
			want: `[![\[logo\]](l.png)](/x)`,
		},
		{
			name: "lazy loaded img with data uri placeholder uses srcset",
			html: `<img src="data:image/gif;base64,R0l" data-srcset="https://ex.com/a.png 1x, b.png 2x" alt="A">`,
			want: "![A](https://ex.com/a.png)",
		},
		{
			name: "card link without heading links first paragraph",
			html: `<a href="/p"><div><p>Title</p><p>Desc</p></div></a>`,
			want: "[Title](/p)\n\nDesc",
		},
		{
			name: "card link with referenced style",
			html: `<a href="/p"><div><h3>Title</h3></div></a>`,
			opts: &Options{LinkStyle: LinkStyleReferenced},
			want: "### [Title][1]\n\n[1]: /p",
		},
		{
			name: "base href resolves relative urls",
			html: `<html><head><base href="https://ex.com/docs/"></head><body><a href="../a">A</a> <img src="i.png" alt="I"></body></html>`,
			want: "[A](https://ex.com/a) ![I](https://ex.com/docs/i.png)",
		},
		{
			name: "base url option resolves relative urls",
			html: `<a href="b/c">rel</a>`,
			opts: &Options{BaseURL: "https://ex.com/a/"},
			want: "[rel](https://ex.com/a/b/c)",
		},
		{
			name: "image with title",
			html: `<img src="a.png" alt="A" title="The title">`,
			want: `![A](a.png "The title")`,
		},
		{
			name: "quotes in link title are escaped",
			html: `<a href="/x" title='say "hi"'>link</a>`,
			want: `[link](/x "say \"hi\"")`,
		},
		{
			name: "newlines in alt text collapse",
			html: `<img src="/i.png" alt="line one
line two" title='a "b"'>`,
			want: `![line one line two](/i.png "a \"b\"")`,
		},
		{
			name: "referenced link",
			html: `<a href="https://example.com">Example</a>`,
			opts: &Options{LinkStyle: LinkStyleReferenced},
			want: "[Example][1]\n\n[1]: https://example.com",
		},
		{
			name: "referenced links share a number per URL",
			html: `<p><a href="https://example.com">First</a> and <a href="https://example.com">Second</a> and <a href="https://other.com">Third</a></p>`,
			opts: &Options{LinkStyle: LinkStyleReferenced},
			want: "[First][1] and [Second][1] and [Third][2]\n\n[1]: https://example.com\n[2]: https://other.com",
		},
		{
			name: "referenced links with different titles get separate numbers",
			html: `<p><a href="https://example.com" title="Foo">First</a> and <a href="https://example.com" title="Bar">Second</a></p>`,
			opts: &Options{LinkStyle: LinkStyleReferenced},
			want: "[First][1] and [Second][2]\n\n[1]: https://example.com \"Foo\"\n[2]: https://example.com \"Bar\"",
		},
		{
			name: "quotes in referenced link title are escaped",
			html: `<a href="/x" title='q "t"'>l</a>`,
			opts: &Options{LinkStyle: LinkStyleReferenced},
			want: `[l][1]

[1]: /x "q \"t\""`,
		},
	})
}

// Unordered, ordered, nested, and task lists.
func TestConvert_Lists(t *testing.T) {
	runCases(t, []convertCase{
		{name: "simple list", html: "<ul><li>One</li><li>Two</li><li>Three</li></ul>", want: "- One\n- Two\n- Three"},
		{
			name: "nested list",
			html: "<ul><li>One<ul><li>Nested</li></ul></li><li>Two</li></ul>",
			want: "- One\n  - Nested\n- Two",
		},
		{
			name: "simple list",
			html: "<ol><li>First</li><li>Second</li><li>Third</li></ol>",
			want: "1. First\n2. Second\n3. Third",
		},
		{
			name: "nested in unordered",
			html: "<ul><li>Item<ol><li>Sub one</li><li>Sub two</li></ol></li></ul>",
			want: "- Item\n  1. Sub one\n  2. Sub two",
		},
		{
			name: "deeply nested lists",
			html: "<ul><li>a<ul><li>b<ul><li>c<ul><li>d</li></ul></li></ul></li></ul></li></ul>",
			want: "- a\n  - b\n    - c\n      - d",
		},
		{name: "orphan li element", html: "<li>Orphan item</li>", want: "- Orphan item"},
		{
			name: "list with paragraphs",
			html: "<ul><li><p>Item 1</p></li><li><p>Item 2</p></li></ul>",
			want: "- Item 1\n- Item 2",
		},
		{
			name: "mixed list types nested",
			html: "<ol><li>First<ul><li>Nested bullet</li></ul></li><li>Second</li></ol>",
			want: "1. First\n   - Nested bullet\n2. Second",
		},
		{
			name: "deeply nested mixed",
			html: "<ul><li>A<ol><li>1<ul><li>x</li></ul></li></ol></li></ul>",
			want: "- A\n  1. 1\n     - x",
		},
		{
			name: "list with inline formatting",
			html: "<ul><li><strong>Bold</strong> item</li><li>Item with <code>code</code></li></ul>",
			want: "- **Bold** item\n- Item with `code`",
		},
		{
			name: "list starting at different number",
			html: `<ol start="5"><li>Five</li><li>Six</li></ol>`,
			want: "5. Five\n6. Six",
		},
		// Seen on the pkg.go.dev/net/http index.
		{
			name: "li wrapping only nested list omits empty bullet",
			html: `<ul>
<li><a href="#Client">type Client</a></li>
<li><ul>
  <li><a href="#Client.Do">func (c *Client) Do(req *Request)</a></li>
</ul></li>
</ul>`,
			want: "- [type Client](#Client)\n  - [func (c *Client) Do(req *Request)](#Client.Do)",
		},
		// Seen on lobste.rs and go.dev.
		{
			name: "li with div wrapping nested list preserves indent",
			html: `<ul>
<li>Parent item
  <div>
    <ul>
      <li>First child</li>
      <li>Second child</li>
    </ul>
  </div>
</li>
</ul>`,
			want: "- Parent item\n  - First child\n  - Second child",
		},
		// Seen on en.wikipedia.org and wiki.archlinux.org.
		{
			name: "li with multiple paragraphs indents continuation",
			html: `<ol>
<li><p>First paragraph of item 1.</p><p>Second paragraph of item 1.</p></li>
<li><p>Item 2.</p></li>
</ol>`,
			want: "1. First paragraph of item 1.\n\n   Second paragraph of item 1.\n2. Item 2.",
		},
		{
			name: "li value attribute overrides number",
			html: `<ol><li>First</li><li value="10">Tenth</li><li>Eleventh</li></ol>`,
			want: "1. First\n10. Tenth\n11. Eleventh",
		},
		{
			name: "task list checkboxes",
			html: `<ul>
<li><input type="checkbox" checked> Completed task</li>
<li><input type="checkbox"> Pending task</li>
</ul>`,
			want: "- [x] Completed task\n- [ ] Pending task",
		},
		{
			name: "paragraph then nested list stays tight",
			html: `<ul><li><p>Text</p><ul><li>Child</li></ul></li><li>Next</li></ul>`,
			want: "- Text\n  - Child\n- Next",
		},
		{
			name: "code block inside list item is indented",
			html: "<ul><li>Item<pre><code>x := 1\ny := 2</code></pre></li></ul>",
			want: "- Item\n\n  ```\n  x := 1\n  y := 2\n  ```",
		},
		{
			name: "indented code block inside list item",
			html: "<ul><li>x<pre>a\n\nb</pre></li></ul>",
			opts: &Options{CodeBlockStyle: CodeBlockStyleIndented},
			want: "- x\n\n      a\n\n      b",
		},
		{
			name: "blockquote inside list item",
			html: `<ul><li><blockquote>q1<p>q2</p></blockquote></li></ul>`,
			want: "- > q1\n  >\n  > q2",
		},
		{
			name: "nested ordered list indents to marker width",
			html: `<ol><li>a</li><li><ol><li>b</li></ol></li><li>c</li></ol>`,
			want: "1. a\n   1. b\n2. c",
		},
		{
			name: "list nested directly in a list",
			html: "<ul><li>One</li><ul><li>Nested</li></ul><li>Two</li></ul>",
			want: "- One\n  - Nested\n- Two",
		},
		{
			name: "bullet char *",
			html: `<ul><li>One</li><li>Two</li></ul>`,
			opts: &Options{BulletChar: "*"},
			want: "* One\n* Two",
		},
		{
			name: "bullet char +",
			html: `<ul><li>One</li><li>Two</li></ul>`,
			opts: &Options{BulletChar: "+"},
			want: "+ One\n+ Two",
		},
	})
}

// Definition lists.
func TestConvert_DefinitionLists(t *testing.T) {
	runCases(t, []convertCase{
		// Seen on go.dev/doc/effective_go.
		{
			name: "definition list dl dt dd separated",
			html: `<dl>
<dt>Indentation</dt>
<dd>We use tabs for indentation and gofmt emits them by default.</dd>
<dt>Line length</dt>
<dd>Go has no line length limit.</dd>
</dl>`,
			want: "**Indentation**\n: We use tabs for indentation and gofmt emits them by default.\n\n**Line length**\n: Go has no line length limit.",
		},
		// Seen on go.dev/doc/effective_go.
		{
			name: "definition list multiple terms and definitions",
			html: `<dl><dt>a</dt><dt>b</dt><dd><p>one</p><p>two</p></dd><dd>three</dd></dl>`,
			want: "**a**\n**b**\n: one\n\n  two\n: three",
		},
	})
}

// Blockquotes.
func TestConvert_Blockquotes(t *testing.T) {
	runCases(t, []convertCase{
		{name: "simple quote", html: "<blockquote>Quote</blockquote>", want: "> Quote"},
		{
			name: "multiline quote",
			html: "<blockquote><p>Line 1</p><p>Line 2</p></blockquote>",
			want: "> Line 1\n>\n> Line 2",
		},
		{name: "nested quote", html: "<blockquote><blockquote>Nested</blockquote></blockquote>", want: "> > Nested"},
		{
			name: "nested blockquotes 3 deep",
			html: "<blockquote><blockquote><blockquote>deep</blockquote></blockquote></blockquote>",
			want: "> > > deep",
		},
		{name: "blockquote with list", html: "<blockquote><ul><li>item</li></ul></blockquote>", want: "> - item"},
		{
			name: "blockquote with code",
			html: "<blockquote><pre>code</pre></blockquote>",
			want: "> ```\n> code\n> ```",
		},
		{
			name: "blockquote with heading",
			html: "<blockquote><h2>Title</h2><p>Text</p></blockquote>",
			want: "> ## Title\n>\n> Text",
		},
		{
			name: "multiple paragraphs in blockquote",
			html: "<blockquote><p>Para 1</p><p>Para 2</p><p>Para 3</p></blockquote>",
			want: "> Para 1\n>\n> Para 2\n>\n> Para 3",
		},
		// Seen on lwn.net/Articles/.
		{
			name: "blockquote with direct inline elements stays single paragraph",
			html: `<blockquote class="bq">
Using Hacktron, we found that an email address like
<code>user@attacker.example</code> gave any registered user
<strong>code execution</strong> on its backend.
</blockquote>`,
			want: "> Using Hacktron, we found that an email address like `user@attacker.example` gave any registered user **code execution** on its backend.",
		},
		{
			name: "blockquote with div wrapping multiple paragraphs",
			html: `<blockquote><div><p>First paragraph.</p><p>Second paragraph.</p></div></blockquote>`,
			want: "> First paragraph.\n>\n> Second paragraph.",
		},
		{
			name: "nested blockquote with paragraphs",
			html: `<blockquote><p>a</p><blockquote><p>b</p><p>c</p></blockquote></blockquote>`,
			want: "> a\n>\n> > b\n> >\n> > c",
		},
		{
			name: "code block in blockquote keeps blank lines",
			html: "<blockquote><pre>a\n\nb</pre></blockquote>",
			want: "> ```\n> a\n>\n> b\n> ```",
		},
	})
}

// Data tables and layout tables.
func TestConvert_Tables(t *testing.T) {
	runCases(t, []convertCase{
		{
			name: "simple table",
			html: "<table><tr><th>Name</th><th>Age</th></tr><tr><td>Alice</td><td>30</td></tr></table>",
			want: "| Name | Age |\n| --- | --- |\n| Alice | 30 |",
		},
		{
			name: "table without headers",
			html: "<table><tr><td>A</td><td>B</td></tr><tr><td>C</td><td>D</td></tr></table>",
			want: "| A | B |\n| --- | --- |\n| C | D |",
		},
		{
			name: "uneven columns",
			html: "<table><tr><td>A</td><td>B</td><td>C</td></tr><tr><td>D</td></tr></table>",
			want: "| A | B | C |\n| --- | --- | --- |\n| D |  |  |",
		},
		{
			name: "with thead and tbody",
			html: "<table><thead><tr><th>H1</th></tr></thead><tbody><tr><td>D1</td></tr></tbody></table>",
			want: "| H1 |\n| --- |\n| D1 |",
		},
		{
			name: "formatted content in cells",
			html: "<table><tr><td><strong>bold</strong></td><td><em>italic</em></td></tr></table>",
			want: "| **bold** | *italic* |\n| --- | --- |",
		},
		{
			name: "links in table cells",
			html: `<table><tr><td><a href="url">link</a></td></tr></table>`,
			want: "| [link](url) |\n| --- |",
		},
		{
			name: "whitespace in cells",
			html: "<table><tr><td>  spaced  </td></tr></table>",
			want: "| spaced |\n| --- |",
		},
		{
			name: "pipe in cell",
			html: "<table><tr><th>op</th></tr><tr><td>a | b</td></tr></table>",
			want: "| op |\n| --- |\n| a \\| b |",
		},
		{
			name: "newline in cell",
			html: "<table><tr><th>h</th></tr><tr><td>line1<br>line2</td></tr></table>",
			want: "| h |\n| --- |\n| line1<br>line2 |",
		},
		// Seen on paulgraham.com/avg.html.
		{
			name: "single column layout table unwrapped",
			html: `<table width="435" cellspacing="0" cellpadding="0">
<tr><td>
<h1>Beating the Averages</h1>
<p>First paragraph of the essay.</p>
<p>Second paragraph of the essay.</p>
</td></tr>
</table>`,
			want: "# Beating the Averages\n\nFirst paragraph of the essay.\n\nSecond paragraph of the essay.",
		},
		// Seen on news.ycombinator.com.
		{
			name: "nested layout table unwrapped",
			html: `<table id="hnmain">
<tr><td>
  <table class="itemlist">
    <tr><td>1.</td><td><a href="https://example.com/1">Story One</a></td></tr>
    <tr><td>2.</td><td><a href="https://example.com/2">Story Two</a></td></tr>
  </table>
</td></tr>
</table>`,
			want: "| 1. | [Story One](https://example.com/1) |\n| --- | --- |\n| 2. | [Story Two](https://example.com/2) |",
		},
		{
			name: "presentation role table unwrapped",
			html: `<table role="presentation">
<tr><td><p>Section A</p></td></tr>
<tr><td><p>Section B</p></td></tr>
</table>`,
			want: "Section A\n\nSection B",
		},
		// Seen on developer.mozilla.org/en-US/docs/Web/HTML/Element/table.
		{
			name: "table caption preserved",
			html: `<table>
<caption>Server Inventory</caption>
<thead><tr><th>Host</th><th>Role</th></tr></thead>
<tbody><tr><td>alpha</td><td>db</td></tr></tbody>
</table>`,
			want: "Server Inventory\n\n| Host | Role |\n| --- | --- |\n| alpha | db |",
		},
		// Seen on developer.mozilla.org/en-US/docs/Web/HTML/Element/table.
		{
			name: "table cell colspan padding",
			html: `<table>
<tr><th>A</th><th>B</th><th>C</th></tr>
<tr><td colspan="2">Span AB</td><td>Cell C</td></tr>
</table>`,
			want: "| A | B | C |\n| --- | --- | --- |\n| Span AB |  | Cell C |",
		},
		// Seen on developer.mozilla.org and en.wikipedia.org.
		{
			name: "table cell with br preserves line break",
			html: `<table>
<tr><th>Name</th><th>Notes</th></tr>
<tr><td>alpha</td><td>Line 1<br>Line 2</td></tr>
</table>`,
			want: "| Name | Notes |\n| --- | --- |\n| alpha | Line 1<br>Line 2 |",
		},
		// Seen on developer.mozilla.org.
		{
			name: "table cell rowspan padding",
			html: `<table><tr><th>A</th><th>B</th></tr><tr><td rowspan="2">x</td><td>1</td></tr><tr><td>2</td></tr></table>`,
			want: "| A | B |\n| --- | --- |\n| x | 1 |\n|  | 2 |",
		},
		// Seen on paulgraham.com, whose essay body uses <br><br>, not <p>.
		{
			name: "single column table with br content unwrapped",
			html: `<table><tr><td><img src="a.png"></td></tr><tr><td><font>Essay line one<br><br>line two</font></td></tr></table>`,
			want: "![](a.png)\n\nEssay line one\n\nline two",
		},
		// Seen on en.wikipedia.org.
		{
			name: "pre inside data table cell",
			html: "<table><tr><th>Code</th></tr><tr><td><pre>a\nb</pre></td></tr></table>",
			want: "| Code |\n| --- |\n| `a`<br>`b` |",
		},
		// Seen on developer.mozilla.org.
		{name: "caption only table", html: `<table><caption>Only</caption></table>`, want: "Only"},
		{
			name: "tfoot rows are kept",
			html: "<table><thead><tr><th>H</th></tr></thead><tbody><tr><td>B</td></tr></tbody><tfoot><tr><td>F</td></tr></tfoot></table>",
			want: "| H |\n| --- |\n| B |\n| F |",
		},
		{
			name: "empty leading row does not become the header",
			html: "<table><tr></tr><tr><th>H</th></tr><tr><td>B</td></tr></table>",
			want: "| H |\n| --- |\n| B |",
		},
	})
}

// Block containers and HTML5 semantic elements.
func TestConvert_Containers(t *testing.T) {
	runCases(t, []convertCase{
		{name: "div ignored", html: "<div>Content</div>", want: "Content"},
		{name: "nested divs", html: "<div><div>Nested</div></div>", want: "Nested"},
		{name: "div with other elements", html: "<div><p>Para</p></div>", want: "Para"},
		{
			name: "deeply nested divs",
			html: "<div><div><div><div><div>text</div></div></div></div></div>",
			want: "text",
		},
		{name: "article", html: "<article><h1>Title</h1><p>Content</p></article>", want: "# Title\n\nContent"},
		{name: "section", html: "<section><h2>Section</h2><p>Text</p></section>", want: "## Section\n\nText"},
		{name: "aside", html: "<aside><p>Sidebar</p></aside>", want: "Sidebar"},
		{
			name: "figure with figcaption",
			html: "<figure><img src=\"img.png\" alt=\"image\"><figcaption>Caption</figcaption></figure>",
			want: "![image](img.png)\n\nCaption",
		},
		{
			name: "details and summary",
			html: "<details><summary>More info</summary><p>Hidden content</p></details>",
			want: "More info\n\nHidden content",
		},
		{name: "mark element", html: "<p>This is <mark>highlighted</mark></p>", want: "This is highlighted"},
		{
			name: "time element",
			html: "<p>Posted <time datetime=\"2024-01-01\">January 1st</time></p>",
			want: "Posted January 1st",
		},
		{name: "abbr element", html: `<p>The <abbr title="World Wide Web">WWW</abbr></p>`, want: "The WWW"},
		{name: "address element", html: "<address>123 Street</address>", want: "123 Street"},
		{name: "cite element", html: "<p>From <cite>The Book</cite></p>", want: "From The Book"},
	})
}

// Whole documents.
func TestConvert_Documents(t *testing.T) {
	runCases(t, []convertCase{
		{
			name: "blog post structure",
			html: `<article>
<header>
<h1>Blog Title</h1>
</header>
<p>2024-01-01</p>
<p>First paragraph.</p>
<p>Second paragraph with <a href="https://example.com">link</a>.</p>
<footer><p>Author: John</p></footer>
</article>`,
			want: "# Blog Title\n\n2024-01-01\n\nFirst paragraph.\n\nSecond paragraph with [link](https://example.com).\n\nAuthor: John",
		},
		{
			name: "navigation stripped",
			html: `<nav><a href="/">Home</a> <a href="/about">About</a></nav><main><p>Content</p></main>`,
			want: "[Home](/) [About](/about)\n\nContent",
		},
		{
			name: "complex document",
			html: `<!DOCTYPE html>
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
			want: "# Welcome\n\nHello **world**!\n\n- One\n- Two",
		},
		{
			name: "full document",
			html: `<!DOCTYPE html>
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
</html>`,
			want: "# Welcome\n\nThis is a **test** document.\n\n- Item 1\n- Item 2",
		},
	})
}

// Content that never reaches the output: skipped tags, hidden elements, and comments.
func TestConvert_SkippedContent(t *testing.T) {
	runCases(t, []convertCase{
		{name: "script", html: "<p>Hello</p><script>alert('hi')</script><p>World</p>", want: "Hello\n\nWorld"},
		{name: "style", html: "<style>body{}</style><p>Content</p>", want: "Content"},
		{name: "head", html: "<html><head><title>Test</title></head><body><p>Body</p></body></html>", want: "Body"},
		{name: "simple comment", html: "<!-- comment -->text", want: "text"},
		{
			name: "comment between elements",
			html: "<p>before</p><!-- comment --><p>after</p>",
			want: "before\n\nafter",
		},
		{name: "comment inside element", html: "<p>before<!-- comment -->after</p>", want: "beforeafter"},
		{name: "multiline comment", html: "text<!-- line1\nline2 -->more", want: "textmore"},
		{name: "unclosed comment", html: "text<!-- comment", want: "text"},
		{name: "empty comment", html: "text<!---->more", want: "textmore"},
		{name: "nested dashes in comment", html: "text<!-- -- -->more", want: "textmore"},
		{name: "conditional comment", html: "<!--[if IE]>IE only<![endif]-->text", want: "text"},
		// Seen on go.dev and GitHub octicons.
		{
			name: "svg and hidden elements skipped by default",
			html: `<p><svg><title>Icon</title><path d="M0 0"/></svg>Visible text<span hidden>secret</span></p>`,
			want: "Visible text",
		},
		{
			name: "internal marker characters in input are stripped",
			html: "<p>a\x01b\x05c\x02</p><pre>p\x03q</pre>",
			want: "abc\n\n```\npq\n```",
		},
		{
			name: "SkipTags option",
			html: `<p>Hello</p><nav>Navigation</nav><p>World</p>`,
			opts: &Options{SkipTags: []string{"nav"}},
			want: "Hello\n\nWorld",
		},
	})
}

// Malformed HTML.
func TestConvert_MalformedHTML(t *testing.T) {
	runCases(t, []convertCase{
		{name: "unclosed tag", html: "<p>Hello <b>world", want: "Hello **world**"},
		{name: "mismatched tags", html: "<div>Hello <b>world</div></b>", want: "Hello **world**"},
		{name: "no closing", html: "<p>Just text", want: "Just text"},
		{name: "weird nesting", html: "<b><i>text</b></i>", want: "***text***"},
		{name: "unclosed nested tags", html: "<div><p><b>text", want: "**text**"},
		{name: "random closing tags", html: "text</b></i></div>", want: "text"},
		{name: "extra closing tags", html: "<p>text</p></p></p>", want: "text"},
		{name: "double open tags", html: "<p><p>text</p></p>", want: "text"},
		{name: "broken tag syntax", html: "<div text", want: ""},
		{name: "angle brackets in text", html: "1 < 2 > 0", want: "1 < 2 > 0"},
		{name: "lone angle brackets", html: "a < b and c > d", want: "a < b and c > d"},
		{name: "empty tags", html: "<p></p>", want: ""},
		{name: "empty nested tags", html: "<div><p><span></span></p></div>", want: ""},
		{name: "whitespace only tags", html: "<p>   </p>", want: ""},
		{name: "tabs and newlines", html: "<p>\t\n\t</p>", want: ""},
		{name: "self-closing div", html: "<div/>text", want: "text"},
		{name: "improperly self-closed p", html: "<p/>text", want: "text"},
		{name: "xhtml style br", html: "<br />", want: ""},
		{name: "mixed case tags", html: "<DIV><P>text</P></DIV>", want: "text"},
		{name: "mixed case with attributes", html: `<A HREF="url">link</A>`, want: "[link](url)"},
		{name: "missing quotes on attributes", html: `<a href=url>link</a>`, want: "[link](url)"},
		{name: "single quotes on attributes", html: `<a href='url'>link</a>`, want: "[link](url)"},
		{
			name: "multiple spaces between attributes",
			html: `<a    href="url"    title="t">link</a>`,
			want: `[link](url "t")`,
		},
		{name: "newlines in tags", html: "<a\nhref=\"url\"\n>link</a>", want: "[link](url)"},
		{name: "tabs in tags", html: "<a\thref=\"url\">link</a>", want: "[link](url)"},
	})
}

// Elements with no content.
func TestConvert_EmptyElements(t *testing.T) {
	runCases(t, []convertCase{
		{name: "empty heading", html: "<h1></h1>", want: ""},
		{name: "empty bold", html: "<strong></strong>", want: ""},
		{name: "empty italic", html: "<em></em>", want: ""},
		{name: "empty link", html: `<a href="url"></a>`, want: ""},
		{name: "empty list", html: "<ul></ul>", want: ""},
		{name: "empty list item", html: "<ul><li></li></ul>", want: ""},
		{name: "empty blockquote", html: "<blockquote></blockquote>", want: ""},
		{name: "empty table", html: "<table></table>", want: ""},
		{name: "empty table row", html: "<table><tr></tr></table>", want: ""},
		{name: "empty code", html: "<code></code>", want: ""},
		{name: "empty pre", html: "<pre></pre>", want: "```\n\n```"},
		{name: "whitespace-only bold", html: "<strong>   </strong>", want: ""},
	})
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	assert.Equal(t, opts.LinkStyle, LinkStyleInline)
	assert.Equal(t, opts.HeadingStyle, HeadingStyleATX)
	assert.Equal(t, opts.CodeBlockStyle, CodeBlockStyleFenced)
	assert.Equal(t, opts.BulletChar, "-")
	assert.Nil(t, opts.SkipTags)
}

// ConvertWithOptions does not modify the caller's Options, so one Options
// value can be shared across goroutines.
func TestConvertWithOptionsDoesNotMutateOptions(t *testing.T) {
	opts := &Options{}
	assert.Equal(t, ConvertWithOptions("<ul><li>a</li></ul>", opts), "- a")
	assert.Equal(t, opts.BulletChar, "")
}
