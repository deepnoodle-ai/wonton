# htmlparse

The htmlparse package provides HTML parsing and transformation utilities with support for metadata extraction, link and image discovery, content filtering, and clean HTML output optimized for LLM consumption.

## Usage Examples

### Basic HTML Parsing

```go
package main

import (
    "fmt"
    "log"

    "github.com/deepnoodle-ai/wonton/htmlparse"
)

func main() {
    html := `
    <html>
        <head>
            <title>My Page</title>
            <meta name="description" content="A sample page">
        </head>
        <body>
            <h1>Hello World</h1>
            <p>This is a paragraph.</p>
        </body>
    </html>
    `

    doc, err := htmlparse.Parse(html)
    if err != nil {
        log.Fatal(err)
    }

    // Extract metadata
    meta := doc.Metadata()
    fmt.Printf("Title: %s\n", meta.Title)
    fmt.Printf("Description: %s\n", meta.Description)

    // Get plain text
    text := doc.Text()
    fmt.Println("Text:", text)
}
```

### Extracting Links

```go
doc, err := htmlparse.Parse(html)
if err != nil {
    log.Fatal(err)
}

// Get all links
links := doc.Links()
for _, link := range links {
    fmt.Printf("%s -> %s\n", link.Text, link.URL)
}
```

### Filtering Links

```go
// Resolve relative URLs and filter by host
links := doc.FilteredLinks(htmlparse.LinkFilter{
    BaseURL:  "https://example.com",
    Internal: true, // Only internal links
})

// Or get only external links
externalLinks := doc.FilteredLinks(htmlparse.LinkFilter{
    BaseURL:  "https://example.com",
    External: true,
})
```

### Extracting Images

```go
images := doc.Images()
for _, img := range images {
    fmt.Printf("URL: %s\n", img.URL)
    fmt.Printf("Alt: %s\n", img.Alt)
    fmt.Printf("Title: %s\n", img.Title)
}
```

### Transforming HTML

```go
// Remove scripts, styles, and navigation
transformed := doc.Transform(&htmlparse.TransformOptions{
    Exclude: []string{"script", "style", "nav", "footer"},
})

fmt.Println(transformed)
```

### Pretty Printing

```go
// Format HTML with indentation
pretty := doc.Transform(&htmlparse.TransformOptions{
    PrettyPrint: true,
})

fmt.Println(pretty)
```

### Extracting Main Content

```go
// Get only the main content area
content := doc.Transform(&htmlparse.TransformOptions{
    OnlyMainContent: true,
})

// This finds <main> or excludes nav/header/footer/aside
```

### Including Specific Tags

```go
// Only keep article and paragraph tags
transformed := doc.Transform(&htmlparse.TransformOptions{
    Include: []string{"article", "p", "h1", "h2", "h3"},
})
```

### Advanced Element Filtering

```go
// Remove elements by attributes
transformed := doc.Transform(&htmlparse.TransformOptions{
    ExcludeFilters: []htmlparse.ElementFilter{
        // Remove all divs with class containing "ad"
        {Tag: "div", Attr: "class", AttrContains: "ad"},
        // Remove elements with specific attribute
        {Attr: "data-tracking"},
        // Remove specific class
        {Attr: "class", AttrEquals: "popup"},
    },
})
```

### Using Standard Exclude Filters

```go
// Remove common non-content elements (modals, scripts, forms, nav)
clean := doc.Transform(&htmlparse.TransformOptions{
    ExcludeFilters: htmlparse.StandardExcludeFilters,
})
```

### Metadata Extraction

```go
meta := doc.Metadata()

// Basic metadata
fmt.Println("Title:", meta.Title)
fmt.Println("Description:", meta.Description)
fmt.Println("Author:", meta.Author)
fmt.Println("Keywords:", meta.Keywords)
fmt.Println("Canonical:", meta.Canonical)
fmt.Println("Charset:", meta.Charset)
fmt.Println("Viewport:", meta.Viewport)

// Open Graph metadata
if meta.OpenGraph != nil {
    og := meta.OpenGraph
    fmt.Println("OG Title:", og.Title)
    fmt.Println("OG Description:", og.Description)
    fmt.Println("OG Image:", og.Image)
    fmt.Println("OG URL:", og.URL)
    fmt.Println("OG Type:", og.Type)
}

// Twitter Card metadata
if meta.Twitter != nil {
    tw := meta.Twitter
    fmt.Println("Twitter Card:", tw.Card)
    fmt.Println("Twitter Title:", tw.Title)
    fmt.Println("Twitter Image:", tw.Image)
}
```

### Extracting Branding Information

```go
branding := doc.Branding()

fmt.Println("Logo:", branding.Logo)
fmt.Println("Favicon:", branding.Favicon)
fmt.Println("Apple Icon:", branding.AppleIcon)
fmt.Println("Theme Color:", branding.ThemeColor)
fmt.Println("Color Scheme:", branding.ColorScheme)
```

### Parsing from Reader

```go
import (
    "os"
)

file, err := os.Open("page.html")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

doc, err := htmlparse.ParseReader(file)
if err != nil {
    log.Fatal(err)
}
```

### Combining Transformations

```go
// Clean HTML for LLM consumption
clean := doc.Transform(&htmlparse.TransformOptions{
    OnlyMainContent: true,
    ExcludeFilters:  htmlparse.StandardExcludeFilters,
    Exclude:         []string{"aside", "figcaption"},
    PrettyPrint:     true,
})
```

### Custom Element Filter

```go
// Create custom filter logic
filter := htmlparse.ElementFilter{
    Tag:          "div",
    Attr:         "role",
    AttrEquals:   "banner",
}

// Check if filter matches
attrs := map[string]string{"role": "banner"}
if filter.Matches("div", attrs) {
    fmt.Println("Element matches filter")
}
```

### Getting Raw HTML

```go
// Get unmodified HTML
raw := doc.HTML()
fmt.Println(raw)
```

## API Reference

### Parsing Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `Parse(html)` | Parses HTML string | `string` | `(*Document, error)` |
| `ParseReader(r)` | Parses HTML from reader | `io.Reader` | `(*Document, error)` |

### Document Methods

| Method | Description | Returns |
|--------|-------------|---------|
| `Metadata()` | Extracts page metadata | `Metadata` |
| `Links()` | Gets all links | `[]Link` |
| `FilteredLinks(filter)` | Gets filtered links | `[]Link` |
| `Images()` | Gets all images | `[]Image` |
| `Branding()` | Extracts brand info | `Branding` |
| `Transform(opts)` | Transforms HTML | `string` |
| `HTML()` | Gets full HTML | `string` |
| `Text()` | Gets plain text | `string` |

### Transform Options

| Field | Type | Description |
|-------|------|-------------|
| `Include` | `[]string` | Only include these tags |
| `Exclude` | `[]string` | Exclude these tags |
| `ExcludeFilters` | `[]ElementFilter` | Advanced filtering |
| `OnlyMainContent` | `bool` | Extract main content only |
| `PrettyPrint` | `bool` | Format with indentation |

### Link Filter

| Field | Type | Description |
|-------|------|-------------|
| `BaseURL` | `string` | Base for resolving relative URLs |
| `Internal` | `bool` | Only same-host links |
| `External` | `bool` | Only different-host links |

### Element Filter

| Field | Type | Description |
|-------|------|-------------|
| `Tag` | `string` | Element tag to match |
| `Attr` | `string` | Attribute name to check |
| `AttrEquals` | `string` | Attribute must equal value |
| `AttrContains` | `string` | Attribute must contain substring |

Element filters use AND logic - all specified fields must match.

### Element Filter Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `Matches(tag, attrs)` | Tests if element matches | `string`, `map[string]string` | `bool` |

### Metadata Fields

| Field | Type | Description |
|-------|------|-------------|
| `Title` | `string` | Page title |
| `Description` | `string` | Meta description |
| `Author` | `string` | Page author |
| `Keywords` | `[]string` | Meta keywords |
| `Canonical` | `string` | Canonical URL |
| `Charset` | `string` | Character encoding |
| `Viewport` | `string` | Viewport settings |
| `Robots` | `string` | Robots directive |
| `OpenGraph` | `*OpenGraph` | Open Graph data |
| `Twitter` | `*Twitter` | Twitter Card data |

### Open Graph Fields

| Field | Type | Description |
|-------|------|-------------|
| `Title` | `string` | OG title |
| `Description` | `string` | OG description |
| `Image` | `string` | OG image URL |
| `URL` | `string` | OG URL |
| `Type` | `string` | OG type |
| `SiteName` | `string` | Site name |

### Twitter Fields

| Field | Type | Description |
|-------|------|-------------|
| `Card` | `string` | Card type |
| `Title` | `string` | Title |
| `Description` | `string` | Description |
| `Image` | `string` | Image URL |
| `Site` | `string` | @username of site |
| `Creator` | `string` | @username of creator |

### Link Fields

| Field | Type | Description |
|-------|------|-------------|
| `URL` | `string` | Link URL |
| `Text` | `string` | Link text |
| `Title` | `string` | Link title attribute |

### Image Fields

| Field | Type | Description |
|-------|------|-------------|
| `URL` | `string` | Image URL |
| `Alt` | `string` | Alt text |
| `Title` | `string` | Title attribute |

### Branding Fields

| Field | Type | Description |
|-------|------|-------------|
| `ColorScheme` | `string` | Color scheme (light/dark) |
| `ThemeColor` | `string` | Theme color |
| `Logo` | `string` | Logo image URL |
| `Favicon` | `string` | Favicon URL |
| `AppleIcon` | `string` | Apple touch icon URL |

### Standard Exclude Filters

Pre-configured filters for common elements to exclude:

**Modal/Dialog Elements:**
- `role="dialog"`
- `aria-modal="true"`
- IDs/classes containing: `cookie`, `popup`, `modal`, `dialog`

**Non-Content Elements:**
- `script`, `style`, `noscript`, `iframe`, `svg`

**Form Elements:**
- `select`, `input`, `button`, `form`

**Navigation:**
- `nav`, `footer`, `hr`

## Common Use Cases

### Clean HTML for LLM Processing

```go
// Remove clutter and keep only content
clean := doc.Transform(&htmlparse.TransformOptions{
    OnlyMainContent: true,
    ExcludeFilters:  htmlparse.StandardExcludeFilters,
})
```

### Extract Article Content

```go
// Get main article without navigation/ads
article := doc.Transform(&htmlparse.TransformOptions{
    Include:        []string{"article", "p", "h1", "h2", "h3", "h4", "ul", "ol", "li"},
    ExcludeFilters: htmlparse.StandardExcludeFilters,
    PrettyPrint:    true,
})
```

### Build Link Index

```go
links := doc.FilteredLinks(htmlparse.LinkFilter{
    BaseURL:  "https://example.com",
    Internal: true,
})

// Create sitemap or link graph
```

### SEO Analysis

```go
meta := doc.Metadata()

// Check SEO elements
if meta.Title == "" {
    fmt.Println("Missing title")
}
if meta.Description == "" {
    fmt.Println("Missing description")
}
if meta.Canonical == "" {
    fmt.Println("No canonical URL")
}

// Check Open Graph
if meta.OpenGraph == nil {
    fmt.Println("Missing Open Graph tags")
}
```

## Related Packages

- [htmltomd](../htmltomd/) - Convert HTML to Markdown
- [fetch](../fetch/) - Fetch web pages with HTML parsing
- [web](../web/) - URL manipulation and text normalization
- [crawler](../crawler/) - Web crawling with HTML parsing

## Implementation Notes

- Uses `golang.org/x/net/html` for parsing
- Handles malformed HTML gracefully
- Case-insensitive attribute matching
- Void elements recognized (img, br, hr, etc.)
- Comments are excluded from output
- Empty text nodes are filtered in pretty-print mode
- Links and images can have empty URLs (skipped in extraction)
- Relative URLs preserved unless BaseURL provided
- Standard exclude filters cover 99% of common page noise
- Transform operations are non-destructive (original doc unchanged)
