# fetch

The fetch package provides interfaces and implementations for fetching web pages with support for HTML parsing, content extraction, and format conversion. It includes a simple HTTP fetcher for direct requests, a `Download` function for fetching files, and extensible interfaces for custom fetchers.

## Default Clients Are SSRF-Guarded

A URL handed to a fetcher is often a URL the program did not choose — from a
user, a feed, or a link on a page it just crawled. So `DefaultHTTPClient` and
`DefaultDownloadClient` are built by [httpguard](../httpguard/): they validate
every address a hostname resolves to and connect only to public ones, which
means **they will not fetch from `localhost`, `127.0.0.1`, or an intranet
host**, and they ignore `HTTP_PROXY` / `HTTPS_PROXY`.

Redirects are still followed, up to the standard limit of 10, with each hop
address-validated like the original request. Plain-HTTP hops are allowed, since
plenty of legitimate sites still use them; credential headers you set
(`Authorization`, cookies) are dropped if such a hop downgrades a chain that
began over HTTPS, so a redirect cannot pull them onto the wire in the clear.

To fetch from a private address — a local dev server, a service on your own
network — supply a client:

```go
fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
    Client: &http.Client{Timeout: fetch.DefaultTimeout},
})

result, err := fetch.Download(ctx, url, &fetch.DownloadOptions{
    Client: &http.Client{},
})
```

Or change it once for the whole program, before any fetcher is constructed —
`NewHTTPFetcher` copies the variable at construction time, so a fetcher built
earlier keeps the client it was given:

```go
fetch.DefaultHTTPClient = &http.Client{Timeout: fetch.DefaultTimeout}
fetch.DefaultDownloadClient = &http.Client{}
```

Both of those give up the guard entirely. To keep it everywhere but one
network, widen the address check instead:

```go
_, devNet, _ := net.ParseCIDR("10.1.2.0/24")

fetch.DefaultHTTPClient = httpguard.NewClient(
    httpguard.WithMaxRedirects(10),
    httpguard.WithHTTPRedirects(),
    httpguard.WithAddressValidator(func(ip net.IP) error {
        if devNet.Contains(ip) {
            return nil
        }
        return httpguard.ValidatePublicIP(ip)
    }))
fetch.DefaultHTTPClient.Timeout = fetch.DefaultTimeout
```

The guard is a network boundary, not a whole policy: it does not enforce an
allowlist, restrict schemes, or cap the response. See the
[httpguard](../httpguard/) README for what it does and does not cover.

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

A supplied client replaces the guarded default entirely, so it reaches
whatever it is pointed at. Wrap it with `httpguard.NewClient` if you want the
guard and your own settings.

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
| `Client` | `*http.Client` | HTTP client to use | `DefaultHTTPClient` (guarded) |
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

## Downloading Files

`Download` fetches a file from a URL, either into memory or to disk:

```go
// Download to memory (result.Data holds the content)
result, err := fetch.Download(ctx, "https://example.com/doc.pdf", nil)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Downloaded %d bytes: %s\n", result.Size, result.Filename)

// Download to a file
result, err = fetch.Download(ctx, "https://example.com/image.png", &fetch.DownloadOptions{
    OutputPath: "/tmp/downloads/image.png",
    CreateDirs: true,
})

// Download to a directory with limits — the filename comes from the
// Content-Disposition header or the URL, sanitized against path traversal
result, err = fetch.Download(ctx, "https://example.com/file.pdf", &fetch.DownloadOptions{
    OutputPath:   "/tmp/downloads/",
    MaxSizeBytes: 10 * 1024 * 1024,  // 10MB limit
    ExpectedType: "application/pdf", // verify the Content-Type
})
fmt.Printf("Saved to: %s\n", result.Path)
```

Notes:

- An existing file at the destination is overwritten.
- The default client has no overall timeout (large downloads shouldn't race
  a wall clock) but bounds how long the server may take to start responding.
  Use the context to cancel or impose a deadline, or supply your own client
  via `DownloadOptions.Client`.

### Download Options

| Field | Type | Description |
|-------|------|-------------|
| `Headers` | `map[string]string` | Additional request headers |
| `OutputPath` | `string` | Destination file or directory; empty downloads to memory |
| `CreateDirs` | `bool` | Create parent directories if needed |
| `MaxSizeBytes` | `int64` | Maximum download size (0 = unlimited) |
| `ExpectedType` | `string` | Required MIME type (e.g. `"application/pdf"`) |
| `Client` | `*http.Client` | HTTP client (default: `DefaultDownloadClient`, which refuses non-public addresses) |

## Error Handling

HTTP failures are returned as `*fetch.Error`, which carries the status code
and URL. `fetch.IsRetryable` classifies both HTTP-level failures (408, 429,
5xx) and transport-level failures (timeouts, connection resets), making it a
natural fit for the [retry](../retry) package:

```go
result, err := retry.Do(ctx, func() (*fetch.DownloadResult, error) {
    return fetch.Download(ctx, url, nil)
}, retry.WithRetryIf(fetch.IsRetryable))

var fetchErr *fetch.Error
if errors.As(err, &fetchErr) {
    fmt.Printf("HTTP %d fetching %s\n", fetchErr.StatusCode, fetchErr.URL)
}
```

The HTTP fetcher validates requests and returns errors for unsupported options:

```go
resp, err := fetcher.Fetch(ctx, req)
if err != nil {
    if errors.Is(err, fetch.ErrUnsupported) {
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
- [httpguard](../httpguard/) - The SSRF guard behind the default clients

## Implementation Notes

- Default clients refuse non-public addresses; supply a `Client` to reach one
- HTTP fetcher only supports text/html content type
- Response body size is limited to prevent memory exhaustion (default 10 MB)
- When no formats are specified, returns HTML by default
- When formats are specified, only requested formats are included
- Element filters use case-insensitive matching for attributes
- Standard exclude filters remove common non-content elements
