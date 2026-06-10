# web

URL canonicalization, link resolution, text normalization, and web search
abstractions for crawlers and content processing. This package contains no
I/O — for fetching pages and downloading files, see the
[fetch](../fetch) package.

## Usage Examples

### URL Normalization

`NormalizeURL` produces a canonical URL suitable for comparison and
deduplication:

```go
package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/web"
)

func main() {
	// Adds https://, removes query params and fragments
	u, err := web.NormalizeURL("example.com/path?query=1#fragment")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(u.String())
	// Output: https://example.com/path

	// http upgraded to https
	u, _ = web.NormalizeURL("http://example.com")
	fmt.Println(u.String())
	// Output: https://example.com

	// Bare host:port inputs work
	u, _ = web.NormalizeURL("localhost:3000")
	fmt.Println(u.String())
	// Output: https://localhost:3000

	// Credentials, default ports, and dot segments are cleaned up
	u, _ = web.NormalizeURL("https://user:pass@example.com:443/a/../b")
	fmt.Println(u.String())
	// Output: https://example.com/b
}
```

Use options when the defaults don't fit:

```go
// Keep query parameters (they matter for sites like YouTube)
u, _ := web.NormalizeURL("youtube.com/watch?v=abc123", web.KeepQuery())
fmt.Println(u.String())
// Output: https://youtube.com/watch?v=abc123

// Don't upgrade http to https
u, _ = web.NormalizeURL("http://internal.example.com", web.KeepHTTP())
fmt.Println(u.String())
// Output: http://internal.example.com
```

### Resolving Links

`ResolveLink` resolves a link found on a page against that page's URL, the
way a browser would, and returns the canonical result:

```go
page, _ := url.Parse("https://example.com/blog/post")

links := []string{
	"/about",                  // absolute path
	"other-post",              // relative to the page
	"../contact",              // parent path
	"https://other.com/page",  // absolute URL
	"mailto:test@example.com", // non-HTTP (rejected)
	"#section",                // fragment (removed)
}

for _, link := range links {
	resolved, err := web.ResolveLink(page, link)
	if err != nil {
		fmt.Printf("%s -> (invalid: %v)\n", link, err)
		continue
	}
	fmt.Printf("%s -> %s\n", link, resolved)
}
// Output:
// /about -> https://example.com/about
// other-post -> https://example.com/blog/other-post
// ../contact -> https://example.com/contact
// https://other.com/page -> https://other.com/page
// mailto:test@example.com -> (invalid: unsupported link scheme "mailto": mailto:test@example.com)
// #section -> https://example.com/blog/post
```

`ResolveLink` accepts the same options as `NormalizeURL`:

```go
resolved, _ := web.ResolveLink(page, "/search?q=go", web.KeepQuery())
// https://example.com/search?q=go
```

### Host Comparison

```go
url1, _ := web.NormalizeURL("https://example.com/path1")
url2, _ := web.NormalizeURL("https://example.com/path2")
url3, _ := web.NormalizeURL("https://sub.example.com/path")
url4, _ := web.NormalizeURL("https://other.com/path")

// Exact hostname match (ports ignored, case-insensitive)
fmt.Println(web.AreSameHost(url1, url2)) // true
fmt.Println(web.AreSameHost(url1, url3)) // false

// Same registrable domain (uses the Public Suffix List)
fmt.Println(web.AreRelatedHosts(url1, url3)) // true (both *.example.com)
fmt.Println(web.AreRelatedHosts(url1, url4)) // false (different domains)
```

`AreRelatedHosts` correctly handles multi-part TLDs: `foo.example.co.uk` and
`bar.example.co.uk` are related, but `example.co.uk` and `other.co.uk` are
not.

### Text Normalization

```go
// Trim whitespace
fmt.Println(web.NormalizeText("  Hello  "))
// Output: Hello

// Unescape HTML entities
fmt.Println(web.NormalizeText("Hello &amp; goodbye"))
// Output: Hello & goodbye

// Replace non-printable characters and non-breaking spaces
fmt.Println(web.NormalizeText("Hello\x00World"))
// Output: Hello World
```

### Detecting File Downloads

`IsBinaryURL` identifies URLs that point to file downloads and page
subresources rather than web pages — the things a crawler extracting text or
following links typically skips:

```go
for _, rawURL := range urls {
	u, err := web.NormalizeURL(rawURL)
	if err != nil {
		continue
	}
	if web.IsBinaryURL(u) {
		continue // skip images, archives, PDFs, scripts, ...
	}
	crawl(u)
}
```

The default set is `web.BinaryExtensions`. Clone it to customize:

```go
exts := web.BinaryExtensions.Clone()
exts.Remove(".pdf") // crawl PDFs too
exts.Add(".dat")

if exts.ContainsURL(u) {
	// skip
}
```

### Crawler Link Processing

The pieces compose into a typical crawler loop:

```go
func extractPageLinks(pageURL *url.URL, htmlLinks []string) []string {
	var validLinks []string
	for _, link := range htmlLinks {
		resolved, err := web.ResolveLink(pageURL, link)
		if err != nil {
			continue // mailto:, javascript:, malformed, ...
		}
		if web.IsBinaryURL(resolved) {
			continue // skip file downloads
		}
		if !web.AreRelatedHosts(pageURL, resolved) {
			continue // stay on this site
		}
		validLinks = append(validLinks, resolved.String())
	}
	return validLinks
}
```

(For a full implementation, see the [crawler](../crawler) package, which is
built on these functions.)

### Deduplicating URLs

```go
func deduplicateURLs(urls []string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, rawURL := range urls {
		normalized, err := web.NormalizeURL(rawURL)
		if err != nil {
			continue
		}
		key := normalized.String()
		if !seen[key] {
			seen[key] = true
			unique = append(unique, key)
		}
	}
	return unique
}

// "example.com", "http://example.com", "https://example.com",
// "example.com?foo=bar", and "example.com#section" all collapse
// to "https://example.com"
```

### Implementing a Search Provider

The `Searcher` interface is the contract between applications and web search
backends (Google, Kagi, etc.):

```go
type MySearcher struct {
	apiKey string
}

func (s *MySearcher) Search(ctx context.Context, input *web.SearchInput) (*web.SearchOutput, error) {
	// Call your search API...
	return &web.SearchOutput{
		Items: []web.SearchItem{
			{
				URL:         "https://example.com/result1",
				Title:       "First Result",
				Description: "Description of the first result",
			},
		},
	}, nil
}

func searchExample() {
	var searcher web.Searcher = &MySearcher{apiKey: "..."}

	results, err := searcher.Search(context.Background(), &web.SearchInput{
		Query: "golang web scraping",
		Limit: 10,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range results.Items {
		fmt.Printf("%s: %s\n", item.Title, item.URL)
	}
}
```

## API Reference

### URL Functions

| Function          | Description                                  | Inputs                                       | Outputs           |
| ----------------- | -------------------------------------------- | -------------------------------------------- | ----------------- |
| `NormalizeURL`    | Canonicalizes a URL string                   | `value string, opts ...NormalizeOption`      | `*url.URL, error` |
| `ResolveLink`     | Resolves a link against the page URL         | `base *url.URL, href string, opts ...`       | `*url.URL, error` |
| `AreSameHost`     | Checks if URLs have the same hostname        | `url1, url2 *url.URL`                        | `bool`            |
| `AreRelatedHosts` | Checks if URLs share a registrable domain    | `url1, url2 *url.URL`                        | `bool`            |

### Normalization Options

| Option        | Effect                                            |
| ------------- | ------------------------------------------------- |
| `KeepQuery()` | Preserve query parameters (removed by default)    |
| `KeepHTTP()`  | Don't upgrade http to https (upgraded by default) |

### File Extension Functions

| Function / Variable  | Description                                                  |
| -------------------- | ------------------------------------------------------------ |
| `IsBinaryURL`        | Checks if a URL points to a file download or subresource     |
| `IsBinaryExtension`  | Checks an extension against the default set                  |
| `BinaryExtensions`   | The default `ExtensionSet`                                   |
| `ExtensionSet`       | Case-insensitive extension set: `Add`, `Remove`, `Contains`, `ContainsURL`, `Clone` |

### Text Functions

| Function        | Description                  | Inputs        | Outputs  |
| --------------- | ---------------------------- | ------------- | -------- |
| `NormalizeText` | Cleans and normalizes text   | `text string` | `string` |

### Search Types

| Type           | Description                                       |
| -------------- | ------------------------------------------------- |
| `Searcher`     | Interface for web search implementations          |
| `SearchInput`  | Search query and limit                            |
| `SearchOutput` | Container for search results                      |
| `SearchItem`   | Individual result: URL, title, description, icon, image |

## URL Normalization Behavior

`NormalizeURL` and `ResolveLink` apply these transformations:

1. Trim whitespace
2. Add `https://` if there is no scheme (including `host:port` inputs)
3. Convert `http://` to `https://` (disable with `KeepHTTP()`)
4. Lowercase the host
5. Remove userinfo (embedded credentials)
6. Remove default ports (`:443` for https, `:80` for http)
7. Remove query parameters (disable with `KeepQuery()`) and fragments
8. Resolve dot segments in the path (`/a/../b` → `/b`)
9. Remove trailing `/` if the path is just `/`

Examples:

- `"example.com"` → `"https://example.com"`
- `"http://example.com"` → `"https://example.com"`
- `"localhost:3000"` → `"https://localhost:3000"`
- `"example.com/path?q=1#frag"` → `"https://example.com/path"`
- `"https://user:pass@example.com:443/"` → `"https://example.com"`

Non-http(s) schemes (`mailto:`, `ftp:`, `javascript:`, `data:`, ...) are
rejected with an error.

## Supported Binary Extensions

`BinaryExtensions` recognizes these common file types:

- **Images**: `.jpg`, `.jpeg`, `.png`, `.gif`, `.svg`, `.webp`, `.avif`, `.heic`, `.heif`, `.bmp`, `.ico`, `.tif`, `.tiff`
- **Video**: `.mp4`, `.webm`, `.mkv`, `.mov`, `.avi`, `.wmv`, `.flv`, `.m4v`, `.mpg`, `.mpeg`, `.ogv`
- **Audio**: `.mp3`, `.wav`, `.aac`, `.ogg`, `.opus`, `.flac`, `.m4a`, `.weba`, `.mid`, `.midi`
- **Documents**: `.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.ppt`, `.pptx`, `.odt`, `.ods`, `.odp`, `.epub`
- **Archives**: `.zip`, `.tar`, `.gz`, `.tgz`, `.bz2`, `.xz`, `.zst`, `.rar`, `.7z`, `.iso`
- **Fonts**: `.ttf`, `.otf`, `.woff`, `.woff2`, `.eot`
- **Executables**: `.exe`, `.dmg`, `.apk`, `.deb`, `.rpm`, `.msi`, `.bin`, `.pkg`, `.img`, `.jar`
- **Page subresources**: `.css`, `.js`, `.mjs`
- **Other**: `.torrent`, `.swf`

## Related Packages

- [crawler](../crawler) - Web crawler built on these utilities
- [fetch](../fetch) - HTTP page fetching and file downloads
- [htmlparse](../htmlparse) - HTML parsing and link extraction
- [htmltomd](../htmltomd) - HTML to Markdown conversion
