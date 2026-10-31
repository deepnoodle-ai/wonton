package htmltomd

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// deficiencyCase pins the fix for a concrete HTML-to-Markdown deficiency
// identified in docs/htmltomd-improvements.md.
type deficiencyCase struct {
	name   string
	source string // real-world site or section in docs/htmltomd-improvements.md
	html   string
	opts   *Options
	want   string
}

func runDeficiencyCases(t *testing.T, cases []deficiencyCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ConvertWithOptions(tc.html, tc.opts)
			assert.Equal(t, got, tc.want, "%s (%s)", tc.name, tc.source)
		})
	}
}

// Section 2.1: Layout Tables, Nested Tables, Captions, and Cell Formatting
func TestDeficiency_Tables(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "single_column_layout_table_unwrapped",
			source: "paulgraham.com/avg.html (2.1)",
			html: `<table width="435" cellspacing="0" cellpadding="0">
<tr><td>
<h1>Beating the Averages</h1>
<p>First paragraph of the essay.</p>
<p>Second paragraph of the essay.</p>
</td></tr>
</table>`,
			want: "# Beating the Averages\n\nFirst paragraph of the essay.\n\nSecond paragraph of the essay.",
		},
		{
			name:   "nested_layout_table_unwrapped",
			source: "news.ycombinator.com (2.1)",
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
			name:   "presentation_role_table_unwrapped",
			source: "HTML ARIA role=presentation (2.1)",
			html: `<table role="presentation">
<tr><td><p>Section A</p></td></tr>
<tr><td><p>Section B</p></td></tr>
</table>`,
			want: "Section A\n\nSection B",
		},
		{
			name:   "table_caption_preserved",
			source: "developer.mozilla.org/en-US/docs/Web/HTML/Element/table (2.1)",
			html: `<table>
<caption>Server Inventory</caption>
<thead><tr><th>Host</th><th>Role</th></tr></thead>
<tbody><tr><td>alpha</td><td>db</td></tr></tbody>
</table>`,
			want: "Server Inventory\n\n| Host | Role |\n| --- | --- |\n| alpha | db |",
		},
		{
			name:   "table_cell_colspan_padding",
			source: "developer.mozilla.org/en-US/docs/Web/HTML/Element/table (2.1)",
			html: `<table>
<tr><th>A</th><th>B</th><th>C</th></tr>
<tr><td colspan="2">Span AB</td><td>Cell C</td></tr>
</table>`,
			want: "| A | B | C |\n| --- | --- | --- |\n| Span AB |  | Cell C |",
		},
		{
			name:   "table_cell_with_br_preserves_line_break",
			source: "developer.mozilla.org & en.wikipedia.org (2.1)",
			html: `<table>
<tr><th>Name</th><th>Notes</th></tr>
<tr><td>alpha</td><td>Line 1<br>Line 2</td></tr>
</table>`,
			want: "| Name | Notes |\n| --- | --- |\n| alpha | Line 1<br>Line 2 |",
		},
	})
}

// Section 2.2: Inter-Tag Newlines Accumulating into 4-Space Code Blocks
func TestDeficiency_InterTagWhitespaceCodeBlocks(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "inter_tag_newlines_do_not_trigger_indented_code_block",
			source: "docs.python.org/3/library/json.html (2.2)",
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
		{
			name:   "indented_code_block_preserves_multiple_blank_lines",
			source: "htmltomd.go postProcess (2.2)",
			html:   "<pre><code>line 1\n\n\nline 2</code></pre>",
			opts:   &Options{CodeBlockStyle: CodeBlockStyleIndented},
			want:   "    line 1\n\n\n    line 2",
		},
	})
}

// Section 2.3: Blockquote Inline Grouping & Nested Paragraph Spacing
func TestDeficiency_Blockquote(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "blockquote_with_direct_inline_elements_stays_single_paragraph",
			source: "lwn.net/Articles/ (2.3)",
			html: `<blockquote class="bq">
Using Hacktron, we found that an email address like
<code>user@attacker.example</code> gave any registered user
<strong>code execution</strong> on its backend.
</blockquote>`,
			want: "> Using Hacktron, we found that an email address like `user@attacker.example` gave any registered user **code execution** on its backend.",
		},
		{
			name:   "blockquote_with_div_wrapping_multiple_paragraphs",
			source: "htmltomd.go handleParagraph inBlockquote (2.3)",
			html:   `<blockquote><div><p>First paragraph.</p><p>Second paragraph.</p></div></blockquote>`,
			want:   "> First paragraph.\n>\n> Second paragraph.",
		},
	})
}

// Section 2.4: Markdown Syntax Emitted Inside <pre> and <code>
func TestDeficiency_PreAndCode(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "links_inside_pre_render_as_plain_text",
			source: "pkg.go.dev/net/http & doc.rust-lang.org (2.4)",
			html:   `<pre><code>// See <a href="https://rfc-editor.org/rfc/rfc9110.html">RFC 9110</a> for details.</code></pre>`,
			want:   "```\n// See RFC 9110 for details.\n```",
		},
		{
			name:   "code_wrapping_single_link_inverts_to_linked_code",
			source: "pkg.go.dev/net/http (2.4)",
			html:   `<p>Use <code><a href="https://pkg.go.dev/net/http#Client">http.Client</a></code> for requests.</p>`,
			want:   "Use [`http.Client`](https://pkg.go.dev/net/http#Client) for requests.",
		},
		{
			name:   "code_with_mixed_text_and_multiple_links_strips_inner_link_syntax",
			source: "doc.rust-lang.org/std/vec/struct.Vec.html (2.4)",
			html:   `<p>When <code><a href="../mem/fn.size_of.html">size_of::&lt;T&gt;</a>() * <a href="#method.capacity">capacity</a>() &gt; 0</code> holds.</p>`,
			want:   "When `size_of::<T>() * capacity() > 0` holds.",
		},
		{
			name:   "empty_code_tag_produces_nothing",
			source: "htmltomd.go handleCode (2.4)",
			html:   `<p>Before <code></code> after.</p>`,
			want:   "Before after.",
		},
		{
			name:   "tt_tag_treated_as_inline_code",
			source: "lwn.net/Articles/ (2.4 & 3)",
			html:   `<p>Address <tt>` + "`touch${IFS}/tmp/pwn`" + `@attacker.example</tt> failed.</p>`,
			want:   "Address `` `touch${IFS}/tmp/pwn`@attacker.example `` failed.",
		},
		{
			name:   "kbd_tag_treated_as_inline_code",
			source: "developer.mozilla.org (3)",
			html:   `<p>Press <kbd>Ctrl+C</kbd> to stop.</p>`,
			want:   "Press `Ctrl+C` to stop.",
		},
		{
			name:   "pre_data_lang_attribute_detected",
			source: "Syntax highlighters data-lang (2.4 & 3)",
			html:   `<pre data-lang="rust"><code>fn main() {}</code></pre>`,
			want:   "```rust\nfn main() {}\n```",
		},
	})
}

// Section 2.5: List Nesting, Wrapper <div>s, and Multi-Paragraph <li>
func TestDeficiency_Lists(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "li_wrapping_only_nested_list_omits_empty_bullet",
			source: "pkg.go.dev/net/http Index (2.5)",
			html: `<ul>
<li><a href="#Client">type Client</a></li>
<li><ul>
  <li><a href="#Client.Do">func (c *Client) Do(req *Request)</a></li>
</ul></li>
</ul>`,
			want: "- [type Client](#Client)\n  - [func (c *Client) Do(req *Request)](#Client.Do)",
		},
		{
			name:   "li_with_div_wrapping_nested_list_preserves_indent",
			source: "lobste.rs & go.dev (2.5)",
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
		{
			name:   "li_with_multiple_paragraphs_indents_continuation",
			source: "en.wikipedia.org & wiki.archlinux.org (2.5)",
			html: `<ol>
<li><p>First paragraph of item 1.</p><p>Second paragraph of item 1.</p></li>
<li><p>Item 2.</p></li>
</ol>`,
			want: "1. First paragraph of item 1.\n\n   Second paragraph of item 1.\n2. Item 2.",
		},
		{
			name:   "li_value_attribute_overrides_number",
			source: "HTML <li value=N> (2.5)",
			html:   `<ol><li>First</li><li value="10">Tenth</li><li>Eleventh</li></ol>`,
			want:   "1. First\n10. Tenth\n11. Eleventh",
		},
		{
			name:   "task_list_checkboxes",
			source: "GFM task list (2.5)",
			html: `<ul>
<li><input type="checkbox" checked> Completed task</li>
<li><input type="checkbox"> Pending task</li>
</ul>`,
			want: "- [x] Completed task\n- [ ] Pending task",
		},
	})
}

// Section 2.6: Links, Images, Bracket Escaping, and URLs with Spaces/Parens
func TestDeficiency_LinksAndImages(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "link_text_with_brackets_escaped",
			source: "danluu.com (2.6)",
			html:   `<p><a href="https://danluu.com/why-video/">Why do people post on [bad platform] instead of [good platform]?</a></p>`,
			want:   `[Why do people post on \[bad platform\] instead of \[good platform\]?](https://danluu.com/why-video/)`,
		},
		{
			name:   "link_url_with_spaces_encoded",
			source: "CommonMark link destination (2.6)",
			html:   `<p><a href="/files/annual report 2026.pdf">Annual Report</a></p>`,
			want:   "[Annual Report](/files/annual%20report%202026.pdf)",
		},
		{
			name:   "javascript_void_link_renders_text_only",
			source: "Common web pattern (2.6)",
			html:   `<p><a href="javascript:void(0)">Expand section</a></p>`,
			want:   "Expand section",
		},
		{
			name:   "empty_img_without_src_or_alt_skipped",
			source: "htmltomd.go handleImage (2.6)",
			html:   `<p>Text before <img> text after.</p>`,
			want:   "Text before text after.",
		},
		{
			name:   "lazy_loaded_img_falls_back_to_data_src",
			source: "Common lazy-load pattern (2.6)",
			html:   `<img data-src="https://example.com/diagram.png" alt="Architecture">`,
			want:   "![Architecture](https://example.com/diagram.png)",
		},
		{
			name:   "card_link_wrapping_heading_and_paragraph",
			source: "Modern card link pattern (2.6)",
			html:   `<a href="/post/1"><div><h3>Post Title</h3><p>Post summary text</p></div></a>`,
			want:   "### [Post Title](/post/1)\n\nPost summary text",
		},
	})
}

// Section 2.7: Inline Formatting Edge Cases & Setext Heading Width
func TestDeficiency_InlineAndHeadings(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "em_wrapping_literal_asterisk_does_not_produce_triple_star",
			source: "docs.python.org/3/library/json.html (2.7)",
			html:   `<p>json.dump(<em>obj</em>, <em>*</em>, <em>skipkeys=False</em>)</p>`,
			want:   `json.dump(*obj*, \*, *skipkeys=False*)`,
		},
		{
			name:   "nested_identical_bold_tags_do_not_cancel_out",
			source: "Malformed HTML nested bold (2.7)",
			html:   `<p><b>Outer <b>inner</b> outer</b></p>`,
			want:   "**Outer inner outer**",
		},
		{
			name:   "br_inside_strong_preserves_line_break",
			source: "htmltomd.go splitEdgeSpace (2.7)",
			html:   `<p><strong>Line 1<br>Line 2</strong></p>`,
			want:   "**Line 1**\n**Line 2**",
		},
		{
			name:   "setext_heading_unicode_underline_length",
			source: "htmltomd.go handleHeading (2.7)",
			html:   `<h1>日本語</h1>`,
			opts:   &Options{HeadingStyle: HeadingStyleSetext},
			want:   "日本語\n======",
		},
	})
}

// Section 3: Missing HTML Elements (<dl>/<dt>/<dd>, <q>, <sup>, <svg>, [hidden])
func TestDeficiency_MissingElementsAndNoise(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "definition_list_dl_dt_dd_separated",
			source: "go.dev/doc/effective_go (3)",
			html: `<dl>
<dt>Indentation</dt>
<dd>We use tabs for indentation and gofmt emits them by default.</dd>
<dt>Line length</dt>
<dd>Go has no line length limit.</dd>
</dl>`,
			want: "**Indentation**\n: We use tabs for indentation and gofmt emits them by default.\n\n**Line length**\n: Go has no line length limit.",
		},
		{
			name:   "inline_q_adds_quotes",
			source: "HTML <q> element (3)",
			html:   `<p>He said <q>hello world</q> to the room.</p>`,
			want:   `He said "hello world" to the room.`,
		},
		{
			name:   "svg_and_hidden_elements_skipped_by_default",
			source: "go.dev & GitHub octicons (3)",
			html:   `<p><svg><title>Icon</title><path d="M0 0"/></svg>Visible text<span hidden>secret</span></p>`,
			want:   "Visible text",
		},
	})
}

// Edge cases uncovered while fixing the deficiencies above.
func TestDeficiency_ImplementationEdgeCases(t *testing.T) {
	runDeficiencyCases(t, []deficiencyCase{
		{
			name:   "paragraph_then_nested_list_stays_tight",
			source: "list item with <p> and nested <ul> (2.5)",
			html:   `<ul><li><p>Text</p><ul><li>Child</li></ul></li><li>Next</li></ul>`,
			want:   "- Text\n  - Child\n- Next",
		},
		{
			name:   "code_block_inside_list_item_is_indented",
			source: "multi-block list item (2.5)",
			html:   "<ul><li>Item<pre><code>x := 1\ny := 2</code></pre></li></ul>",
			want:   "- Item\n\n  ```\n  x := 1\n  y := 2\n  ```",
		},
		{
			name:   "indented_code_block_inside_list_item",
			source: "multi-block list item (2.2 & 2.5)",
			html:   "<ul><li>x<pre>a\n\nb</pre></li></ul>",
			opts:   &Options{CodeBlockStyle: CodeBlockStyleIndented},
			want:   "- x\n\n      a\n\n      b",
		},
		{
			name:   "blockquote_inside_list_item",
			source: "multi-block list item (2.3 & 2.5)",
			html:   `<ul><li><blockquote>q1<p>q2</p></blockquote></li></ul>`,
			want:   "- > q1\n  >\n  > q2",
		},
		{
			name:   "nested_blockquote_with_paragraphs",
			source: "blockquote block flow (2.3)",
			html:   `<blockquote><p>a</p><blockquote><p>b</p><p>c</p></blockquote></blockquote>`,
			want:   "> a\n>\n> > b\n> >\n> > c",
		},
		{
			name:   "code_block_in_blockquote_keeps_blank_lines",
			source: "blockquote code fences (2.3)",
			html:   "<blockquote><pre>a\n\nb</pre></blockquote>",
			want:   "> ```\n> a\n>\n> b\n> ```",
		},
		{
			name:   "nested_ordered_list_indents_to_marker_width",
			source: "CommonMark list item content column (2.5)",
			html:   `<ol><li>a</li><li><ol><li>b</li></ol></li><li>c</li></ol>`,
			want:   "1. a\n   1. b\n2. c",
		},
		{
			name:   "table_cell_rowspan_padding",
			source: "developer.mozilla.org (2.1)",
			html:   `<table><tr><th>A</th><th>B</th></tr><tr><td rowspan="2">x</td><td>1</td></tr><tr><td>2</td></tr></table>`,
			want:   "| A | B |\n| --- | --- |\n| x | 1 |\n|  | 2 |",
		},
		{
			name:   "single_column_table_with_br_content_unwrapped",
			source: "paulgraham.com essay body uses <br><br> (2.1)",
			html:   `<table><tr><td><img src="a.png"></td></tr><tr><td><font>Essay line one<br><br>line two</font></td></tr></table>`,
			want:   "![](a.png)\n\nEssay line one\n\nline two",
		},
		{
			name:   "pre_inside_data_table_cell",
			source: "en.wikipedia.org (2.1)",
			html:   "<table><tr><th>Code</th></tr><tr><td><pre>a\nb</pre></td></tr></table>",
			want:   "| Code |\n| --- |\n| `a`<br>`b` |",
		},
		{
			name:   "caption_only_table",
			source: "developer.mozilla.org (2.1)",
			html:   `<table><caption>Only</caption></table>`,
			want:   "Only",
		},
		{
			name:   "link_url_with_unbalanced_parens_encoded",
			source: "en.wikipedia.org (2.6)",
			html:   `<a href="/files/report (final.pdf">R</a> <a href="https://en.wikipedia.org/wiki/Convention_(norm)">C</a>`,
			want:   "[R](/files/report%20%28final.pdf) [C](https://en.wikipedia.org/wiki/Convention_(norm))",
		},
		{
			name:   "obfuscated_javascript_scheme_renders_text_only",
			source: "Common web pattern (2.6)",
			html:   "<a href=\" JaVa\tScript:alert(1)\">x</a>",
			want:   "x",
		},
		{
			name:   "image_alt_brackets_escaped_inside_link",
			source: "danluu.com (2.6)",
			html:   `<a href="/x"><img alt="[logo]" src="l.png"></a>`,
			want:   `[![\[logo\]](l.png)](/x)`,
		},
		{
			name:   "lazy_loaded_img_with_data_uri_placeholder_uses_srcset",
			source: "Common lazy-load pattern (2.6)",
			html:   `<img src="data:image/gif;base64,R0l" data-srcset="https://ex.com/a.png 1x, b.png 2x" alt="A">`,
			want:   "![A](https://ex.com/a.png)",
		},
		{
			name:   "card_link_without_heading_links_first_paragraph",
			source: "Modern card link pattern (2.6)",
			html:   `<a href="/p"><div><p>Title</p><p>Desc</p></div></a>`,
			want:   "[Title](/p)\n\nDesc",
		},
		{
			name:   "card_link_with_referenced_style",
			source: "Modern card link pattern (2.6)",
			html:   `<a href="/p"><div><h3>Title</h3></div></a>`,
			opts:   &Options{LinkStyle: LinkStyleReferenced},
			want:   "### [Title][1]\n\n[1]: /p",
		},
		{
			name:   "heading_permalink_anchor_dropped",
			source: "docs.python.org headerlink (4.4)",
			html:   `<h2 id="x">Title<a class="headerlink" href="#x">¶</a></h2>`,
			want:   "## Title",
		},
		{
			name:   "br_inside_em_preserves_line_break",
			source: "htmltomd.go splitEdgeSpace (2.7)",
			html:   `<p><em>Line 1<br>Line 2</em> after</p>`,
			want:   "*Line 1*\n*Line 2* after",
		},
		{
			name:   "em_starting_with_asterisks_escapes_them",
			source: "docs.python.org **kw (2.7)",
			html:   `<p><em>**kw</em></p>`,
			want:   `*\*\*kw*`,
		},
		{
			name:   "code_inside_bold_is_not_escaped",
			source: "inline code literal text (2.7)",
			html:   `<p><strong>bold <code>a*b</code></strong></p>`,
			want:   "**bold `a*b`**",
		},
		{
			name:   "sup_and_sub_kept_as_inline_html",
			source: "martinfowler.com (3)",
			html:   `<p>E = mc<sup>2</sup> and H<sub>2</sub>O</p>`,
			want:   "E = mc<sup>2</sup> and H<sub>2</sub>O",
		},
		{
			name:   "definition_list_multiple_terms_and_definitions",
			source: "go.dev/doc/effective_go (3)",
			html:   `<dl><dt>a</dt><dt>b</dt><dd><p>one</p><p>two</p></dd><dd>three</dd></dl>`,
			want:   "**a**\n**b**\n: one\n\n  two\n: three",
		},
		{
			name:   "sphinx_highlight_wrapper_language",
			source: "docs.python.org (3)",
			html:   `<div class="highlight-python notranslate"><div class="highlight"><pre>x = 1</pre></div></div>`,
			want:   "```python\nx = 1\n```",
		},
		{
			name:   "github_highlight_source_language",
			source: "github.com (3)",
			html:   `<pre class="highlight-source-go">x</pre>`,
			want:   "```go\nx\n```",
		},
		{
			name:   "base_href_resolves_relative_urls",
			source: "<base href> (3)",
			html:   `<html><head><base href="https://ex.com/docs/"></head><body><a href="../a">A</a> <img src="i.png" alt="I"></body></html>`,
			want:   "[A](https://ex.com/a) ![I](https://ex.com/docs/i.png)",
		},
		{
			name:   "base_url_option_resolves_relative_urls",
			source: "Options.BaseURL (3)",
			html:   `<a href="b/c">rel</a>`,
			opts:   &Options{BaseURL: "https://ex.com/a/"},
			want:   "[rel](https://ex.com/a/b/c)",
		},
		{
			name:   "internal_marker_characters_in_input_are_stripped",
			source: "converter control characters",
			html:   "<p>a\x01b\x05c\x02</p><pre>p\x03q</pre>",
			want:   "abc\n\n```\npq\n```",
		},
	})
}
