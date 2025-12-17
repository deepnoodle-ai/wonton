# fetch

The fetch package provides interfaces and implementations for fetching web pages with support for HTML parsing, content extraction, and format conversion. It includes a simple HTTP fetcher for direct requests and extensible interfaces for custom fetchers.

## Usage Examples

### Basic HTTP Fetching

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    // Create HTTP fetcher with defaults
    fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{})

    // Fetch a page
    req := &fetch.Request{
        URL: "https://example.com",
    }

    resp, err := fetcher.Fetch(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Title: %s\n", resp.Metadata.Title)
    fmt.Printf("Status: %d\n", resp.StatusCode)
}
```

### Content Extraction

```go
// Extract only main content, excluding navigation and footer
req := &fetch.Request{
    URL:             "https://example.com/article",
    OnlyMainContent: true,
    Formats:         []string{"html", "markdown"},
}

resp, err := fetcher.Fetch(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Println("HTML:", resp.HTML)
fmt.Println("Markdown:", resp.Markdown)
```

### Custom Filtering

```go
req := &fetch.Request{
    URL:         "https://example.com",
    ExcludeTags: []string{"script", "style", "nav", "footer"},
    IncludeTags: []string{"article", "main", "div"},
    Prettify:    true,
}

resp, err := fetcher.Fetch(ctx, req)
```

### Advanced Element Filtering

```go
req := &fetch.Request{
    URL: "https://example.com",
    ExcludeFilters: []fetch.ElementFilter{
        {Tag: "div", Attr: "class", AttrContains: "advertisement"},
        {Tag: "aside"},
        {Attr: "data-ad"},
    },
}

resp, err := fetcher.Fetch(ctx, req)
```

### Multiple Output Formats

```go
req := &fetch.Request{
    URL: "https://example.com",
    Formats: []string{
        "html",        // Transformed HTML
        "raw_html",    // Original HTML
        "markdown",    // Markdown conversion
        "links",       // Extract all links
        "images",      // Extract all images
        "branding",    // Extract brand colors/logos
    },
}

resp, err := fetcher.Fetch(ctx, req)

// Access different formats
for _, link := range resp.Links {
    fmt.Printf("Link: %s -> %s\n", link.Text, link.URL)
}

for _, img := range resp.Images {
    fmt.Printf("Image: %s (alt: %s)\n", img.URL, img.Alt)
}

if resp.Branding != nil {
    fmt.Printf("Logo: %s\n", resp.Branding.Logo)
    fmt.Printf("Theme Color: %s\n", resp.Branding.Colors.Primary)
}
```

### Custom HTTP Client

```go
import (
    "net/http"
    "time"
)

client := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:    10,
        IdleConnTimeout: 30 * time.Second,
    },
}

fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
    Client:      client,
    Timeout:     60 * time.Second,
    MaxBodySize: 20 * 1024 * 1024, // 20 MB
    Headers: map[string]string{
        "User-Agent": "MyApp/1.0",
    },
})
```

### Request Timeout

```go
req := &fetch.Request{
    URL:     "https://example.com",
    Timeout: 5000, // 5 seconds in milliseconds
}

resp, err := fetcher.Fetch(ctx, req)
```

### Custom Headers

```go
req := &fetch.Request{
    URL: "https://api.example.com",
    Headers: map[string]string{
        "Authorization": "Bearer token123",
        "Accept":        "text/html",
    },
}

resp, err := fetcher.Fetch(ctx, req)
```

### Processing HTML Directly

```go
// Process HTML without fetching
htmlContent := "<html><body><h1>Hello</h1></body></html>"

req := &fetch.Request{
    URL:             "https://example.com",
    OnlyMainContent: true,
    Formats:         []string{"markdown"},
}

resp, err := fetch.ProcessRequest(req, htmlContent)
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Markdown)
```

### Using Standard Exclude Filters

```go
// Use predefined filters for common elements to exclude
req := &fetch.Request{
    URL:            "https://example.com",
    ExcludeFilters: fetch.StandardExcludeFilters,
}

// StandardExcludeFilters includes:
// - Modal/dialog elements
// - Cookie consent banners
// - Scripts, styles, iframes
// - Forms and inputs
// - Navigation and footers
```

## API Reference

### Fetcher Interface

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `Fetch(ctx, req)` | Fetches a webpage and returns response | `context.Context`, `*Request` | `(*Response, error)` |

### HTTP Fetcher

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `NewHTTPFetcher(opts)` | Creates HTTP fetcher | `HTTPFetcherOptions` | `*HTTPFetcher` |

### HTTP Fetcher Options

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `Timeout` | `time.Duration` | Request timeout | 30s |
| `Headers` | `map[string]string` | Default headers | `{}` |
| `Client` | `*http.Client` | HTTP client to use | Default client |
| `MaxBodySize` | `int64` | Max response body size | 10 MB |

### Request Fields

| Field | Type | Description |
|-------|------|-------------|
| `URL` | `string` | URL to fetch (required) |
| `OnlyMainContent` | `bool` | Extract only main content area |
| `IncludeTags` | `[]string` | Only include these HTML tags |
| `ExcludeTags` | `[]string` | Exclude these HTML tags |
| `ExcludeFilters` | `[]ElementFilter` | Advanced element filtering |
| `Timeout` | `int` | Request timeout in milliseconds |
| `Prettify` | `bool` | Pretty-print HTML output |
| `Formats` | `[]string` | Output formats to generate |
| `Headers` | `map[string]string` | Custom request headers |

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `URL` | `string` | Final URL (after redirects) |
| `StatusCode` | `int` | HTTP status code |
| `Headers` | `map[string]string` | Response headers |
| `HTML` | `string` | Transformed HTML content |
| `RawHTML` | `string` | Original HTML content |
| `Markdown` | `string` | Markdown conversion |
| `Metadata` | `Metadata` | Page metadata (title, description, etc.) |
| `Links` | `[]Link` | Extracted links |
| `Images` | `[]Image` | Extracted images |
| `Branding` | `*BrandingProfile` | Brand colors, logos, fonts |
| `Timestamp` | `time.Time` | Fetch timestamp |
| `Error` | `string` | Error message if fetch failed |

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
| `Robots` | `string` | Robots meta tag |
| `OpenGraph` | `*OpenGraph` | Open Graph metadata |
| `Twitter` | `*Twitter` | Twitter Card metadata |

### Element Filter

| Field | Type | Description |
|-------|------|-------------|
| `Tag` | `string` | Element tag name to match |
| `Attr` | `string` | Attribute name to check |
| `AttrEquals` | `string` | Attribute must equal this value |
| `AttrContains` | `string` | Attribute must contain this substring |

### Processing Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `ProcessRequest(req, html)` | Processes HTML with request options | `*Request`, `string` | `(*Response, error)` |

### Supported Formats

| Format | Description |
|--------|-------------|
| `"html"` | Transformed HTML (default if no formats specified) |
| `"raw_html"` | Original, unmodified HTML |
| `"markdown"` | Markdown conversion of content |
| `"links"` | Extract all hyperlinks |
| `"images"` | Extract all images |
| `"branding"` | Extract brand identity (colors, logos) |

### Pre-defined Filter Sets

| Constant | Description |
|----------|-------------|
| `StandardExcludeFilters` | Common elements to exclude (modals, scripts, nav, forms) |

## Error Handling

The HTTP fetcher validates requests and returns errors for unsupported options:

```go
resp, err := fetcher.Fetch(ctx, req)
if err != nil {
    if errors.Is(err, fetch.ErrUnsupportedOption) {
        // Handle unsupported option error
    }
    // Handle other errors
}
```

Unsupported options for HTTPFetcher:
- `MaxAge` (caching)
- `WaitFor` (requires browser)
- `Mobile` (mobile emulation)
- `Actions` (browser automation)
- `StorageState` (cookies/localStorage)
- Format: `"screenshot"`, `"json"`, `"summary"`

## Implementing Custom Fetchers

```go
type MyFetcher struct {
    // Your fields
}

func (f *MyFetcher) Fetch(ctx context.Context, req *fetch.Request) (*fetch.Response, error) {
    // Your implementation
    // Can use fetch.ProcessRequest() to handle HTML transformation

    htmlContent := // ... fetch HTML ...

    return fetch.ProcessRequest(req, htmlContent)
}
```

## Related Packages

- [htmlparse](../htmlparse/) - HTML parsing and transformation
- [htmltomd](../htmltomd/) - HTML to Markdown conversion
- [web](../web/) - URL manipulation and normalization
- [crawler](../crawler/) - Web crawling with fetch integration

## Implementation Notes

- HTTP fetcher only supports text/html content type
- Response body size is limited to prevent memory exhaustion (default 10 MB)
- When no formats are specified, returns HTML by default
- When formats are specified, only requested formats are included
- Environment variables override transformations during HTML processing
- Element filters use case-insensitive matching for attributes
- Standard exclude filters remove common non-content elements
