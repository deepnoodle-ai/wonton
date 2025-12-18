# htmltomd

Convert HTML to Markdown with graceful handling of malformed input. Ideal for preparing web content for LLM consumption or generating clean documentation from HTML sources.

## Features

- Robust HTML parsing using golang.org/x/net/html
- Handles malformed HTML gracefully
- Customizable output styles (ATX vs Setext headings, inline vs referenced links, etc.)
- Supports common HTML elements (headings, lists, tables, code blocks, blockquotes)
- Preserves wide character display properties
- Tag skipping for unwanted content

## Usage Examples

### Basic Conversion

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/htmltomd"
)

func main() {
    html := `
        <h1>Getting Started</h1>
        <p>Welcome to our <strong>API</strong>.</p>
        <ul>
            <li>Fast</li>
            <li>Reliable</li>
        </ul>
    `

    md := htmltomd.Convert(html)
    fmt.Println(md)
    // Output:
    // # Getting Started
    //
    // Welcome to our **API**.
    //
    // - Fast
    // - Reliable
}
```

### Custom Options

```go
// Use referenced links and Setext-style headings
opts := &htmltomd.Options{
    LinkStyle:      htmltomd.LinkStyleReferenced,
    HeadingStyle:   htmltomd.HeadingStyleSetext,
    CodeBlockStyle: htmltomd.CodeBlockStyleIndented,
    BulletChar:     "*",
}

md := htmltomd.ConvertWithOptions(html, opts)
```

### Skip Unwanted Tags

```go
opts := &htmltomd.Options{
    SkipTags: []string{"nav", "footer", "aside"},
}

md := htmltomd.ConvertWithOptions(html, opts)
```

### Convert Tables

```go
html := `
    <table>
        <tr><th>Name</th><th>Age</th></tr>
        <tr><td>Alice</td><td>30</td></tr>
        <tr><td>Bob</td><td>25</td></tr>
    </table>
`

md := htmltomd.Convert(html)
// Output:
// | Name | Age |
// | --- | --- |
// | Alice | 30 |
// | Bob | 25 |
```

### Handle Code Blocks

```go
html := `
    <pre><code class="language-go">
    func main() {
        fmt.Println("Hello")
    }
    </code></pre>
`

md := htmltomd.Convert(html)
// Output:
// ```go
// func main() {
//     fmt.Println("Hello")
// }
// ```
```

## API Reference

### Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Convert` | Converts HTML to Markdown using default options | `htmlContent string` | `string` |
| `ConvertWithOptions` | Converts HTML with custom options | `htmlContent string, opts *Options` | `string` |
| `DefaultOptions` | Returns default conversion options | None | `*Options` |

### Types

#### Options

Configuration for HTML to Markdown conversion.

```go
type Options struct {
    LinkStyle      LinkStyle      // How links are rendered (inline/referenced)
    HeadingStyle   HeadingStyle   // How headings are rendered (ATX/Setext)
    CodeBlockStyle CodeBlockStyle // How code blocks are rendered (fenced/indented)
    BulletChar     string         // Character for unordered lists ("-", "*", "+")
    SkipTags       []string       // HTML tags to skip entirely
}
```

#### Constants

**LinkStyle:**
- `LinkStyleInline` - Renders links as `[text](url)` (default)
- `LinkStyleReferenced` - Renders links as `[text][n]` with references at end

**HeadingStyle:**
- `HeadingStyleATX` - Renders headings with `#` prefix (default)
- `HeadingStyleSetext` - Renders h1/h2 with underlines (`===` or `---`)

**CodeBlockStyle:**
- `CodeBlockStyleFenced` - Renders code blocks with ` ``` ` fences (default)
- `CodeBlockStyleIndented` - Renders code blocks with 4-space indentation

## Supported HTML Elements

- **Headings:** h1-h6
- **Text formatting:** strong, b, em, i, del, s, strike, code
- **Links:** a (with href and title)
- **Images:** img (with src and alt)
- **Lists:** ul, ol, li (with nesting)
- **Blockquotes:** blockquote (with nesting)
- **Code blocks:** pre, code (with language detection)
- **Tables:** table, thead, tbody, tr, th, td
- **Horizontal rules:** hr
- **Line breaks:** br
- **Containers:** div, article, section, main, aside, header, footer, nav

## Related Packages

- [htmlparse](../htmlparse) - Parse HTML for metadata extraction and link discovery
- [fetch](../fetch) - Fetch HTML content from URLs
- [crawler](../crawler) - Crawl websites with HTML parsing

## Design Notes

The converter prioritizes graceful degradation over strict correctness. Malformed HTML is parsed as best as possible rather than failing. Script, style, head, and noscript tags are always skipped regardless of configuration.

Wide characters (like CJK characters) are handled correctly, but the Markdown output does not preserve width hints since Markdown has no equivalent concept.
