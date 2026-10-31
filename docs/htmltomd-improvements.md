# `htmltomd` Review: Real-World Deficiencies & Improvement Plan

Date: October 31, 2026

## 1. Executive Summary

We conducted a line-by-line code audit of [`htmltomd/htmltomd.go`](../htmltomd/htmltomd.go) alongside empirical testing against **15 real-world web pages** spanning news aggregators, essays, technical documentation, wikis, and RFCs:

| Corpus Sample | Source URL | HTML Size | Converted MD Size / Lines | Primary Deficiencies Exposed |
|---|---|---|---|---|
| `paulgraham` | `https://paulgraham.com/avg.html` | 38.0 KB | 29.0 KB (**2 lines!**) | Layout `<table>` flattens entire essay into a single 29 KB table row |
| `hn` | `https://news.ycombinator.com/` | 34.4 KB | 10.7 KB (**4 lines!**) | Nested layout `<table>` flattens 30 stories into one pipe-escaped line |
| `python_json` | `https://docs.python.org/3/library/json.html` | 150.2 KB | 35.1 KB (650 lines) | Inter-tag newlines create accidental 4-space code blocks; `<em>*</em>` -> `***`; missing `<dl>/<dt>/<dd>` |
| `effective_go` | `https://go.dev/doc/effective_go` | 155.8 KB | 99.1 KB (2,504 lines) | `<dl>/<dt>/<dd>` definitions glued into run-on paragraphs; Material icon ligature text leaked |
| `pkg_godev_http` | `https://pkg.go.dev/net/http` | 538.9 KB | 147.8 KB (4,394 lines) | `<li><ul>...</ul></li>` emits empty `-` bullets; `<a>` inside `<pre>` emits raw `[RFC](url)` inside code fences |
| `rust_vec` | `https://doc.rust-lang.org/std/vec/struct.Vec.html` | 674.7 KB | 132.9 KB (3,204 lines) | `<code><a href="...">...</a></code>` emits raw Markdown link syntax inside backticks |
| `rfc2616` | `https://datatracker.ietf.org/doc/html/rfc2616` | 561.7 KB | 412.6 KB (9,708 lines) | `<pre>` containing `<a>`, `<span class="h1">`, `<hr>` emits raw Markdown inside code blocks; `<select><option>` text dump |
| `wikipedia_table` | `https://en.wikipedia.org/wiki/Markdown` | 217.9 KB | 52.8 KB (438 lines) | `<pre>` inside `<td>` collapses fences onto one line (` ``` ... ``` `); unescaped `[[1]](#cite)` brackets; URLs with `()` |
| `mdn_table` | `https://developer.mozilla.org/en-US/docs/Web/HTML/Element/table` | 140.4 KB | 24.4 KB (724 lines) | `<caption>` silently dropped (`extractTableRows`); `colspan`/`rowspan` column misalignment |
| `danluu` | `https://danluu.com/` | 89.0 KB | 31.3 KB (206 lines) | Adjacent `<a>` / custom `<d>` tags glued (`10/26["Title"](url)`); unescaped `[bad platform]` inside link text |
| `lwn` | `https://lwn.net/Articles/` | 59.0 KB | 16.4 KB (260 lines) | `<blockquote>` without `<p>` splits inline `<tt>` into separate paragraphs; `<tt>` not handled as `<code>` |
| `lobsters` | `https://lobste.rs/` | 68.7 KB | 15.6 KB (489 lines) | `<li>` containing `<div><ul>` strips indentation on first nested list item |
| `archwiki_systemd` | `https://wiki.archlinux.org/title/Systemd` | 238.5 KB | 59.0 KB (922 lines) | Multi-paragraph/block list items lose continuation indentation |
| `martinfowler` | `https://martinfowler.com/articles/microservices.html` | 115.6 KB | 60.6 KB (587 lines) | `<figure>`/`<figcaption>`, `<sub>`/`<sup>`, and sidebar callout structure flattened |
| `npr_text` | `https://text.npr.org/` | 8.2 KB | 2.6 KB (72 lines) | Clean baseline, but relative URLs remain unresolved without `BaseURL` |

---

## 2. Critical Correctness & Malformed HTML Deficiencies

### 2.1 Layout Tables & Nested Tables Collapse Entire Pages into 1–2 Giant Lines
- **Location**: `handleTable` (`htmltomd/htmltomd.go:860-910`), `extractTableRows` (`htmltomd/htmltomd.go:912-930`), `extractTableRow` (`htmltomd/htmltomd.go:932-949`)
- **Real-World Reproduction**:
  - `paulgraham.com/avg.html` wraps the entire 38 KB essay inside a layout `<table><tr><td>...</td></tr></table>`. `handleTable` converts the entire essay into the header row of a GFM table (`28,993 bytes` on line 1, followed by `| --- | |` on line 2).
  - `news.ycombinator.com` wraps the page in `<table id="hnmain">` with a nested `<table class="itemlist">`. `extractTableRow` (`htmltomd/htmltomd.go:940-945`) converts the inner table to a GFM table and then collapses all `\n` to `" "` and escapes every `|` to `\|`, producing a 10.7 KB single line full of `\| --- \|`.
  - `en.wikipedia.org/wiki/Markdown` places side-by-side `<pre><code>` examples inside a 2-column `<table>`. `extractTableRow` (`htmltomd/htmltomd.go:943`) collapses the newlines in ```` ```\ncode\n``` ```` into single-line ```` ``` code ``` ```` inside the table cell.
- **Additional Table Bugs**:
  1. **`<caption>` silently dropped**: `extractTableRows` (`htmltomd/htmltomd.go:918-928`) only inspects `thead`, `tbody`, `tfoot`, and `tr`. In `mdn_table.html`, `<caption>He-Man and Skeletor facts</caption>` is completely lost.
  2. **Headerless tables promote first data row to `<th>`**: Every `<table>` unconditionally treats `rows[0]` as a header row followed by `| --- |`, even when the table has no `<thead>` or `<th>` cells.
  3. **`<br>` in table cells**: `extractTableRow` (`htmltomd/htmltomd.go:943`) replaces `\n` (from `<br>`) with `" "` instead of `<br>` (or space when appropriate), losing multi-line cell structure.
  4. **`colspan` / `rowspan` ignored**: Cells with `colspan="2"` emit a single cell and get padded only at the far right of the row (`htmltomd/htmltomd.go:884-888`), shifting subsequent columns under the wrong headers.

---

### 2.2 Inter-Tag Newlines Accumulate into Accidental 4-Space Indented Code Blocks
- **Location**: `handleText` (`htmltomd/htmltomd.go:289-299`) and `postProcess` (`htmltomd/htmltomd.go:978-995`)
- **Real-World Reproduction** (`docs.python.org/3/library/json.html`):
  ```html
  ...</p>\n</div>\n</dd></dl>\n\n<dl class="py function">\n<dt class="sig sig-object py" id="json.dumps">\n<span class="sig-prename descclassname"><span class="pre">json.</span></span>
  ```
  Each `\n` between closing/opening non-block or unhandled tags (`</div>`, `</dd></dl>`, `<dl>`, `<dt>`, `<span>`) is a separate `html.TextNode` that `handleText` (`htmltomd/htmltomd.go:297`) normalizes to `" "`. Four consecutive inter-tag newlines after `</div>` (which emitted `\n\n`) produce `"    json.dumps(*obj*, ...)"`.
  Then `postProcess` (`htmltomd/htmltomd.go:984-986`) sees `leadingSpaces >= 4` and sets `keepIndent = true` **even when `c.opts.CodeBlockStyle == CodeBlockStyleFenced`**!
- **Result**: Every function and CLI flag signature after the first in `python_json.md` (lines 231, 235, 269, 326, 332, 422–434, 558–586) is rendered as a 4-space indented code block with raw unparsed `*` and `[¶](#...)`.
- **Related `CodeBlockStyleIndented` Bug in `postProcess`**:
  When `CodeBlockStyle == CodeBlockStyleIndented`, `handlePre` (`htmltomd/htmltomd.go:823-828`) prefixes every line with `"    "`. For empty lines inside `<pre>`, `"    "` has `strings.TrimSpace(line) == ""`, so `postProcess` (`htmltomd/htmltomd.go:966-970`) strips the 4 spaces and collapses runs of multiple blank lines inside the indented code block into a single blank line!

---

### 2.3 `<blockquote>` Splits Inline Elements into Separate Paragraphs & Concatenates Nested `<p>`s
- **Location**: `handleBlockquote` (`htmltomd/htmltomd.go:683-742`), `handleBlockquoteInner` (`htmltomd/htmltomd.go:744-797`), `handleParagraph` (`htmltomd/htmltomd.go:368-377`)
- **Bug 1 — Direct inline children split across paragraphs** (`lwn.net/Articles/`):
  ```html
  <blockquote class="bq">
    Using Hacktron, we found that an email address like
    <tt>`touch${IFS}/tmp/savane-rce-proof`@attacker.example</tt> gave any
    registered user code execution on its backend...
  </blockquote>
  ```
  Because `handleBlockquote` (`htmltomd/htmltomd.go:689-717`) treats *every* direct child of `<blockquote>` as a separate `blockPart` and joins `blockParts` with `\n>\n`, any `<blockquote>` without a wrapping `<p>` splits its sentence at every `<strong>`, `<em>`, `<a>`, `<code>`, or `<tt>` child:
  ```markdown
  > Using Hacktron, we found that an email address like
  >
  > `touch${IFS}/tmp/savane-rce-proof`@attacker.example
  >
  > gave any registered user code execution on its backend...
  ```
- **Bug 2 — `<p>` inside `<div>` inside `<blockquote>` has no paragraph breaks**:
  `handleParagraph` (`htmltomd/htmltomd.go:373-375`) returns `content` *without* `\n\n` whenever `ctx.inBlockquote > 0`. If `<p>` elements are nested inside a `<div>`, `<article>`, or `<li>` within a `<blockquote>` (`<blockquote><div><p>First.</p><p>Second.</p></div></blockquote>`), they concatenate directly into:
  ```markdown
  > First.Second.
  ```
- **Bug 3 — `postProcess` does not track code fences inside `<blockquote>`**:
  In `postProcess` (`htmltomd/htmltomd.go:959-1010`), `trimmed` still starts with `> `, so `> ``` ` is never recognized as opening a code fence.
- **Bug 4 — Dead duplicate function**:
  `handleBlockquoteInner` (`htmltomd/htmltomd.go:744-797`) is a verbatim copy of `handleBlockquote` (`htmltomd/htmltomd.go:683-742`) (since `ctx.inBlockquote > 0` already suppresses outer `\n\n` at line 738).

---

### 2.4 Markdown Syntax Emitted Inside `<pre>` and `<code>`
- **Location**: `handleElement` (`htmltomd/htmltomd.go:301-349`), `handlePre` (`htmltomd/htmltomd.go:799-840`), `handleCode` (`htmltomd/htmltomd.go:458-472`)
- **Real-World Reproduction**:
  1. **Links/headings/rules inside `<pre>`** (`pkg.go.dev/net/http`, `doc.rust-lang.org`, `datatracker.ietf.org/doc/html/rfc2616`):
     Documentation generators frequently hyperlink types and RFCs inside `<pre>` blocks (`<pre>// See <a href="...">RFC 9110</a></pre>` or `pub struct Vec<T, A = <a href="...">Global</a>>`). Because `ctx.inPre` is only checked in `handleText` and `handleCode`, `handleLink` still emits `[RFC 9110](https://...)` inside ` ``` ... ``` `, where Markdown displays the raw brackets and URL literally.
  2. **Links inside inline `<code>`** (`doc.rust-lang.org/std/vec/struct.Vec.html`):
     - When `<code>` wraps a single link (`<code><a href="url">http.Client</a></code>`), `htmltomd` outputs `` `[http.Client](url)` `` (broken literal brackets in code span) instead of inverting to `[`http.Client`](url)`.
     - When `<code>` mixes code text and multiple links (`<code><a href="...">size_of::&lt;T&gt;</a>() * <a href="...">capacity</a>() &gt; 0</code>`), `htmltomd` outputs `` `[size_of::<T>](...)() * [capacity](...)() > 0` `` instead of stripping the inner link markup to plain text `` `size_of::<T>() * capacity() > 0` ``.
  3. **Empty `<code></code>` emits literal `` `` ``**:
     `handleCode` (`htmltomd/htmltomd.go:463-471`) does not check for empty content; `Convert("<code></code>")` returns `` "``" ``.

---

### 2.5 List Nesting, Wrapper `<div>`s, and Multi-Paragraph `<li>` Bugs
- **Location**: `convertListChildren` (`htmltomd/htmltomd.go:580-633`), `handleListItem` (`htmltomd/htmltomd.go:635-681`)
- **Real-World Reproduction**:
  1. **Dangling empty bullet `-` on `<li><ul>...</ul></li>`** (`pkg.go.dev/net/http` Index, lines 493, 501, 512):
     `pkg.go.dev` nests method lists inside their own `<li><ul>...</ul></li>` sibling. In `handleListItem` (`htmltomd/htmltomd.go:670-678`), `content` is `""` and `len(nestedLists) > 0`, so it emits `indent + "- "` followed by `\n  - ...`, which `postProcess` (`htmltomd/htmltomd.go:973`) trims to a standalone `-` line:
     ```markdown
     - [type Client](#Client)
     -
       - [func (c *Client) CloseIdleConnections()](#Client.CloseIdleConnections)
     ```
  2. **`<li>` containing `<div><ul>...</ul></div>` loses first nested item's indentation** (`lobste.rs`, `go.dev`):
     `handleListItem` (`htmltomd/htmltomd.go:657`) only checks direct children for `ul`/`ol`. When `<ul>` is wrapped in a `<div>` inside `<li>`, `handleBlockContainer` calls `strings.TrimSpace` on `"\n  - Child 1\n  - Child 2"`, stripping the leading 2 spaces from `Child 1`:
     ```markdown
     - Parent

     - Child 1
       - Child 2
     ```
  3. **Multi-paragraph / block elements inside `<li>` break list continuity** (`wikipedia_table.md`, `archwiki_systemd.md`):
     `<ol><li><p>Para 1</p><p>Para 2</p></li><li><p>Item 2</p></li></ol>` outputs `Para 2` at column 0 without indentation, terminating the list in CommonMark.
  4. **`<li value="N">` and Task Lists (`<input type="checkbox">`)**:
     `<ol start="N">` is handled, but `<li value="N">` and GFM task list checkboxes (`<li><input type="checkbox" checked> Task</li>`) are ignored.

---

### 2.6 Link & Image Escaping, Broken URLs, and Card Links
- **Location**: `handleLink` (`htmltomd/htmltomd.go:491-525`), `handleImage` (`htmltomd/htmltomd.go:535-543`)
- **Deficiencies**:
  1. **Unescaped `[` and `]` in link text and image `alt`**:
     - `danluu.com`: `<a href="https://danluu.com/why-video/">Why do people post on [bad platform] instead of [good platform]?</a>` -> `[Why do people post on [bad platform] instead of [good platform]?](...)` (invalid CommonMark link text).
     - `en.wikipedia.org`: `<a href="#cite_note-1">[1]</a>` -> `[[1]](#cite_note-1)`.
  2. **URLs with spaces or unbalanced/nested parentheses**:
     - `<a href="/files/report (final).pdf">Report</a>` -> `[Report](/files/report (final).pdf)` (broken CommonMark link because of spaces).
     - `en.wikipedia.org`: URLs like `https://en.wikipedia.org/wiki/Convention_(norm)` or URLs with unmatched `)` break inline links unless parenthesized/spaced URLs are percent-encoded or wrapped in `<...>`.
  3. **`javascript:` / `vbscript:` / `data:` links and empty `<img>`**:
     - `<a href="javascript:void(0)">Toggle</a>` emits `[Toggle](javascript:void(0))`.
     - `<img>` without `src` emits `![]()`; `<img alt="Logo">` without `src` emits `![Logo]()`.
     - Lazy-loaded images (`<img data-src="..." src="data:image/gif;base64,...">` or `srcset`) ignore `data-src`/`srcset`.
  4. **Block/Card links (`<a><div><h3>Title</h3><p>Desc</p></div></a>`)**:
     - `handleLink` (`htmltomd/htmltomd.go:498`) collapses all newlines via `splitEdgeSpace`, turning headings inside `<a>` into `[### Title Desc](url)` where `###` is no longer a heading.
  5. **Adjacent `<a>` / custom inline tags glued together** (`danluu.com`):
     - `<div class=np><a href=...>Archive</a><a href=...>Patreon</a></div>` -> `[Archive](...)[Patreon](...)` and `<li><d>10/26</d><a href=...>Title</a></li>` -> `- 10/26[Title](...)`.

---

### 2.7 Inline Formatting Edge Cases (`wrapInline`, `splitEdgeSpace`, `handleHeading`)
- **Location**: `handleHeading` (`htmltomd/htmltomd.go:379-402`), `wrapInline` (`htmltomd/htmltomd.go:419-425`), `splitEdgeSpace` (`htmltomd/htmltomd.go:430-448`)
- **Deficiencies**:
  1. **`<br>` inside `<strong>`, `<em>`, `<del>` is lost**:
     `splitEdgeSpace` runs `whitespaceRegex.ReplaceAllString(content, " ")`, turning `\n` from `<br>` inside `<strong>Line 1<br>Line 2</strong>` into `" "` (`**Line 1 Line 2**`).
  2. **Nested identical emphasis tags & literal `*` inside `<em>`/`<strong>`**:
     - `<b>Bold <b>inner</b> end</b>` produces `**Bold **inner** end**` (which closes bold before `inner`).
     - In `python_json.md:199`, `<em class="sig-param">*</em>` produces `***` and `<em class="sig-param">**kw</em>` produces `***kw*`!
  3. **`HeadingStyleSetext` uses byte length `len(content)`**:
     `handleHeading` (`htmltomd/htmltomd.go:394`) computes `strings.Repeat(char, len(content))` using UTF-8 byte count rather than display width (or rune count).

---

## 3. Completeness Gaps (Missing HTML Elements & Attributes)

| Category | Elements / Attributes | Current Behavior | Impact on Real Sites |
|---|---|---|---|
| **Definition Lists** | `<dl>`, `<dt>`, `<dd>` | Falls through to `default: convertChildren` | **High**: `effective_go.md` concatenates `<dt>` and `<dd>` into run-on paragraphs (`Term 1Definition 1`); `python_json.md` & `mdn_table.md` lose definition hierarchy |
| **Monospace / Code-like** | `<tt>`, `<kbd>`, `<samp>`, `<var>` | Falls through to `default: convertChildren` | **Medium**: `lwn.md` loses `<tt>` code spans (leaving raw backticks unescaped); technical docs lose `<kbd>` keystrokes |
| **Inline Semantics** | `<q>`, `<sub>`, `<sup>`, `<ins>`, `<u>` | Falls through to `default: convertChildren` | **Medium**: `<q>` loses quotation marks; footnotes/math (`10<sup>6</sup>` -> `106`) corrupt numbers! |
| **Interactive / Disclosure** | `<details>`, `<summary>` | Both treated as `handleBlockContainer` (`<div>`) | **Low-Medium**: `<summary>` is indistinguishable from body text |
| **Media & Fallbacks** | `<picture>`, `<video>`, `<audio>`, `<iframe>` | Falls through to `default: convertChildren` | **Low-Medium**: `<video>`/`<audio>` fallback text or `src` links lost |
| **Non-Content / Hidden Noise** | `<svg>`, `<math>`, `<template>`, `<canvas>`, `<select>`, `<option>`, `<button>`, `<textarea>`, `[hidden]`, `aria-hidden="true"` | Only `script`, `style`, `head`, `noscript` are skipped | **High**: `rfc2616.md` dumps `<select><option>` menus; `effective_go.md` dumps `<i class="material-icons">arrow_drop_down</i>` and hidden menu text; inline `<svg><title>`/`<text>` leaks into Markdown |
| **Code Block Languages** | `data-lang`, `data-language`, `highlight-source-*`, parent `<div class="highlight-go"><pre>` | Only checks `language-*` / `lang-*` on `<code>` and `<pre>` | **Medium**: GitHub, Chroma, Pygments, and Sphinx code blocks lose syntax language tags |
| **Relative URLs** | `<base href="...">`, `Options.BaseURL` | Emits raw relative `href` / `src` | **Medium**: Scraped pages (`npr_text`, `python_json`, `pkg_godev`) have relative links (`../glossary.html`) that break outside the original origin |

---

## 4. Phased Improvement Plan

### Phase 1: Core Structural & Block/Inline Flow Fixes (Highest Impact)
1. **Eliminate Accidental 4-Space Code Blocks from Inter-Tag Whitespace**:
   - In `handleText` (`htmltomd/htmltomd.go:289-299`), drop whitespace-only text nodes (`strings.TrimSpace(n.Data) == ""`) when adjacent to block-level elements or at the start/end of a block container, and collapse adjacent spaces across inline boundaries before line splitting.
   - In `postProcess` (`htmltomd/htmltomd.go:984-986`), only preserve 4-space indentation as a code block when `c.opts.CodeBlockStyle == CodeBlockStyleIndented` (or track `<pre>` blocks explicitly via a sentinel/marker so `postProcess` never guesses whether 4 spaces came from HTML formatting vs `<pre>`, and preserves blank lines inside `CodeBlockStyleIndented`).
2. **Smart Table Handling (Data Tables vs Layout Tables)**:
   - Detect **layout tables**: unwrap `<table>` as normal block content instead of a GFM pipe table when:
     - The table has `role="presentation"` or `role="none"`, OR
     - The table contains nested `<table>`, `<pre>`, `<blockquote>`, or `<h1>`–`<h6>` and has no `<th>`/`<thead>`, OR
     - The table is a single-column (`colCount == 1`) headerless wrapper with multi-paragraph content (`paulgraham.com`, `news.ycombinator.com`).
   - Extract `<caption>` in `handleTable` (`htmltomd/htmltomd.go:860`) and render it as a paragraph/bold caption above the table.
   - Convert `<br>` inside table cells to `<br>` (or space for single-line trailing `<br>`) instead of losing line breaks, and honor `colspan` by inserting empty padding cells immediately after the spanning cell.
3. **Fix `<blockquote>` Inline Grouping & Nested `<p>` Spacing**:
   - Remove the duplicate `handleBlockquoteInner` (`htmltomd/htmltomd.go:744-797`) and the `if ctx.inBlockquote > 0` special case in `handleParagraph` (`htmltomd/htmltomd.go:373-375`).
   - Instead, convert `<blockquote>` children using normal block/inline flow (`c.convertChildren(n, newCtx)`), normalize via `postProcess`, and prefix the resulting lines with `> ` (or `>` for empty lines). This simultaneously fixes:
     - Inline elements inside `<p>`-less `<blockquote>` (`lwn.net`),
     - `<p>` inside `<div>` inside `<blockquote>`, and
     - Code fences inside `<blockquote>` (`postProcess` runs on the inner content before `> ` prefixing!).

### Phase 2: `<pre>`, `<code>`, and List Fidelity
1. **Plain-Text Extraction Inside `<pre>` and Mixed `<code>`**:
   - When `ctx.inPre` is true, do not emit Markdown syntax (`[text](url)`, `**`, `*`, `#`, `---`) for descendant elements; render `<br>` as `\n` and extract text content for inline/block children.
   - In `handleCode` (`htmltomd/htmltomd.go:458-472`):
     - Return `""` if `strings.TrimSpace(content) == ""`.
     - If `<code>` wraps a single `<a href="...">` child (with optional whitespace), invert it to `[`code`](href)` (matching `<a><code>...</code></a>`).
     - Otherwise, extract plain text inside `<code>` so inner `<a>` or `<span>` tags do not inject raw `[text](url)` inside backticks (`rust_vec.md`).
   - Support `<tt>`, `<kbd>`, and `<samp>` via `handleCode`.
   - Expand `classLanguage` to check `data-lang`, `data-language`, `highlight-source-*`, and parent `<div class="highlight ...">` wrappers.
2. **Fix List Item Edge Cases**:
   - In `handleListItem` (`htmltomd/htmltomd.go:670-680`), when `content == ""` and `len(nestedLists) > 0` (`<li><ul>...</ul></li>` on `pkg.go.dev`), emit the nested list without a dangling empty `-` bullet line.
   - Handle `<ul>`/`<ol>` nested inside a `<div>` within `<li>` (`lobste.rs`) without `handleBlockContainer`'s `strings.TrimSpace` stripping the first item's indentation.
   - Indent continuation paragraphs/blocks inside multi-paragraph `<li>` elements by matching the list marker width (`  ` or `   `).
   - Support `<li value="N">` and GFM task list checkboxes (`<input type="checkbox">`).

### Phase 3: Definition Lists, Inline Escaping, Links/Images, and Noise Filtering
1. **Add `<dl>`, `<dt>`, `<dd>` Support**:
   - Render `<dt>` as bold term (`**Term**`) on its own line and `<dd>` as `: Definition` or indented paragraph/block below `<dt>` (`effective_go.md`, `python_json.md`, `mdn_table.md`).
2. **Harden Link & Image Formatting**:
   - Escape unescaped `[` and `]` inside link text and image `alt` text (`\[`, `\]`).
   - Format URLs containing spaces or unbalanced/nested parentheses safely (encode spaces as `%20` or wrap destination in `<...>`).
   - Ignore `javascript:` / `vbscript:` `href`s (render text only).
   - Skip `<img>` when both `src` (with fallback to `data-src`) and `alt` are empty; if `src` is empty or a placeholder `data:` URI and `alt` is non-empty, avoid emitting broken `![alt]()`.
   - Handle card links (`<a>` wrapping headings/paragraphs) gracefully.
3. **Inline Formatting & Missing Semantic Tags**:
   - Prevent `<em>*</em>` / `<strong>**</strong>` from producing `***` / `******` (escape `*`/`_` inside emphasis when content starts/ends with the delimiter, and deduplicate directly nested `<b><b>` / `<em><em>`).
   - Preserve `<br>` inside `<strong>`, `<em>`, `<del>` by splitting emphasis delimiters across the line break (`**Line 1**\n**Line 2**`) or preserving `\n` inside `splitEdgeSpace`.
   - Add `<q>` (`"..."`), `<sup>` (`^text^` or `<sup>` / Unicode fallback), and `<details>`/`<summary>` (`**Summary**`).
4. **Default Non-Content Filtering & Optional `BaseURL`**:
   - Skip `<svg>`, `<template>`, `<canvas>`, and `[hidden]` elements by default (and optionally `form`/`select`/`button`/`nav`/`footer` or `aria-hidden="true"` anchor permalinks `¶`/`§`/`#` inside headings).
   - Add `BaseURL string` to `Options` (`htmltomd/htmltomd.go:123-150`) (and respect `<base href="...">`) to resolve relative `href` and `src` URLs.
