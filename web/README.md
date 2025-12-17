# web

URL manipulation, text normalization, and media type detection for web crawling and content processing. Provides utilities for normalizing URLs, resolving relative links, cleaning text, and identifying media files.

## Usage Examples

### URL Normalization

```go
package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/web"
)

func main() {
	// Normalize URL (adds https://, removes query params)
	url, err := web.NormalizeURL("example.com/path?query=1#fragment")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(url.String())
	// Output: https://example.com/path

	// Normalize with http:// (converts to https://)
	url, _ = web.NormalizeURL("http://example.com")
	fmt.Println(url.String())
	// Output: https://example.com

	// Already normalized URL
	url, _ = web.NormalizeURL("https://example.com/path")
	fmt.Println(url.String())
	// Output: https://example.com/path

	// Trim whitespace
	url, _ = web.NormalizeURL("  example.com  ")
	fmt.Println(url.String())
	// Output: https://example.com
}
```

### Resolving Relative Links

```go
func resolveLinks(baseDomain string, links []string) {
	for _, link := range links {
		resolved, ok := web.ResolveLink(baseDomain, link)
		if ok {
			fmt.Printf("%s -> %s\n", link, resolved)
		} else {
			fmt.Printf("%s -> (invalid)\n", link)
		}
	}
}

func main() {
	// Resolve relative URLs
	baseDomain := "https://example.com/blog"
	links := []string{
		"/about",                    // Absolute path
		"post/123",                  // Relative path
		"../contact",                // Parent path
		"https://other.com/page",    // Absolute URL
		"mailto:test@example.com",   // Non-HTTP (rejected)
		"#section",                  // Fragment (removed)
	}

	resolveLinks(baseDomain, links)
	// Output:
	// /about -> https://example.com/about
	// post/123 -> https://example.com/blog/post/123
	// ../contact -> https://example.com/contact
	// https://other.com/page -> https://other.com/page
	// mailto:test@example.com -> (invalid)
	// #section -> https://example.com/blog
}
```

### Host Comparison

```go
func compareHosts() {
	url1, _ := web.NormalizeURL("https://example.com/path1")
	url2, _ := web.NormalizeURL("https://example.com/path2")
	url3, _ := web.NormalizeURL("https://sub.example.com/path")
	url4, _ := web.NormalizeURL("https://other.com/path")

	// Check if same host
	fmt.Println(web.AreSameHost(url1, url2))
	// Output: true

	fmt.Println(web.AreSameHost(url1, url3))
	// Output: false

	// Check if related hosts (same domain)
	fmt.Println(web.AreRelatedHosts(url1, url3))
	// Output: true (both *.example.com)

	fmt.Println(web.AreRelatedHosts(url1, url4))
	// Output: false (different domains)
}
```

### Sorting URLs

```go
func sortURLs(urls []string) {
	// Parse URLs
	var parsedURLs []*url.URL
	for _, u := range urls {
		parsed, err := web.NormalizeURL(u)
		if err == nil {
			parsedURLs = append(parsedURLs, parsed)
		}
	}

	// Sort alphabetically
	web.SortURLs(parsedURLs)

	// Print sorted URLs
	for _, u := range parsedURLs {
		fmt.Println(u.String())
	}
}

func main() {
	urls := []string{
		"example.com/zebra",
		"example.com/alpha",
		"example.com/beta",
	}
	sortURLs(urls)
	// Output:
	// https://example.com/alpha
	// https://example.com/beta
	// https://example.com/zebra
}
```

### Text Normalization

```go
func cleanText(input string) string {
	return web.NormalizeText(input)
}

func main() {
	// Trim whitespace
	fmt.Println(cleanText("  Hello  "))
	// Output: Hello

	// Unescape HTML entities
	fmt.Println(cleanText("Hello &amp; goodbye"))
	// Output: Hello & goodbye

	fmt.Println(cleanText("&lt;div&gt;"))
	// Output: <div>

	// Remove non-printable characters
	fmt.Println(cleanText("Hello\x00\x01World"))
	// Output: Hello  World

	// Combined transformations
	fmt.Println(cleanText("  &quot;Hello&quot; \x00"))
	// Output: "Hello"
}
```

### Checking Punctuation

```go
func checkPunctuation() {
	fmt.Println(web.EndsWithPunctuation("Hello."))
	// Output: true

	fmt.Println(web.EndsWithPunctuation("Hello?"))
	// Output: true

	fmt.Println(web.EndsWithPunctuation("Hello"))
	// Output: false

	fmt.Println(web.EndsWithPunctuation("Hello!"))
	// Output: true

	fmt.Println(web.EndsWithPunctuation(""))
	// Output: false
}
```

### Media URL Detection

```go
func filterMediaURLs(urls []string) {
	var mediaURLs []string
	var pageURLs []string

	for _, rawURL := range urls {
		parsed, err := web.NormalizeURL(rawURL)
		if err != nil {
			continue
		}

		if web.IsMediaURL(parsed) {
			mediaURLs = append(mediaURLs, parsed.String())
		} else {
			pageURLs = append(pageURLs, parsed.String())
		}
	}

	fmt.Println("Page URLs:")
	for _, u := range pageURLs {
		fmt.Printf("  %s\n", u)
	}

	fmt.Println("\nMedia URLs:")
	for _, u := range mediaURLs {
		fmt.Printf("  %s\n", u)
	}
}

func main() {
	urls := []string{
		"example.com/page.html",
		"example.com/image.jpg",
		"example.com/document.pdf",
		"example.com/video.mp4",
		"example.com/about",
	}

	filterMediaURLs(urls)
	// Output:
	// Page URLs:
	//   https://example.com/page.html
	//   https://example.com/about
	//
	// Media URLs:
	//   https://example.com/image.jpg
	//   https://example.com/document.pdf
	//   https://example.com/video.mp4
}
```

### Web Crawler URL Processing

```go
type Crawler struct {
	baseDomain string
	visited    map[string]bool
	queue      []string
}

func (c *Crawler) AddLink(link string) {
	// Resolve relative link
	resolved, ok := web.ResolveLink(c.baseDomain, link)
	if !ok {
		return
	}

	// Parse normalized URL
	url, err := web.NormalizeURL(resolved)
	if err != nil {
		return
	}

	// Skip media files
	if web.IsMediaURL(url) {
		return
	}

	// Check if same domain
	baseURL, _ := web.NormalizeURL(c.baseDomain)
	if !web.AreSameHost(url, baseURL) {
		return
	}

	// Add to queue if not visited
	urlStr := url.String()
	if !c.visited[urlStr] {
		c.visited[urlStr] = true
		c.queue = append(c.queue, urlStr)
	}
}
```

### Link Extraction with Filtering

```go
func extractPageLinks(baseURL string, htmlLinks []string) []string {
	var validLinks []string

	for _, link := range htmlLinks {
		// Resolve and normalize
		resolved, ok := web.ResolveLink(baseURL, link)
		if !ok {
			continue
		}

		url, err := web.NormalizeURL(resolved)
		if err != nil {
			continue
		}

		// Skip media files
		if web.IsMediaURL(url) {
			continue
		}

		validLinks = append(validLinks, url.String())
	}

	return validLinks
}
```

### HTML Text Cleaning

```go
func extractCleanText(htmlText string) string {
	// Remove HTML tags (simplified - use proper parser in production)
	text := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(htmlText, "")

	// Normalize the text
	text = web.NormalizeText(text)

	return text
}

func main() {
	html := `<p>Hello &amp; welcome!</p><div>This is a test.</div>`
	fmt.Println(extractCleanText(html))
	// Output: Hello & welcome! This is a test.
}
```

### Deduplicate URLs

```go
func deduplicateURLs(urls []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, rawURL := range urls {
		// Normalize to ensure consistent comparison
		normalized, err := web.NormalizeURL(rawURL)
		if err != nil {
			continue
		}

		urlStr := normalized.String()
		if !seen[urlStr] {
			seen[urlStr] = true
			unique = append(unique, urlStr)
		}
	}

	return unique
}

func main() {
	urls := []string{
		"example.com",
		"http://example.com",
		"https://example.com",
		"example.com?foo=bar",
		"example.com#section",
	}

	deduplicated := deduplicateURLs(urls)
	for _, u := range deduplicated {
		fmt.Println(u)
	}
	// Output:
	// https://example.com
}
```

## API Reference

### URL Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NormalizeURL` | Normalizes URL with transformations | `value string` | `*url.URL, error` |
| `ResolveLink` | Resolves relative URL against base | `domain, value string` | `string, bool` |
| `AreSameHost` | Checks if URLs have same host | `url1, url2 *url.URL` | `bool` |
| `AreRelatedHosts` | Checks if URLs share domain | `url1, url2 *url.URL` | `bool` |
| `SortURLs` | Sorts URLs alphabetically | `urls []*url.URL` | none (in-place) |
| `IsMediaURL` | Checks if URL points to media file | `u *url.URL` | `bool` |

### Text Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NormalizeText` | Cleans and normalizes text | `text string` | `string` |
| `EndsWithPunctuation` | Checks if string ends with punctuation | `s string` | `bool` |

### Variables

| Variable | Description |
|----------|-------------|
| `MediaExtensions` | Map of media file extensions (`.jpg`, `.pdf`, `.mp4`, etc.) |

## URL Normalization Behavior

`NormalizeURL` applies these transformations:

1. Trims whitespace
2. Adds `https://` prefix if missing
3. Converts `http://` to `https://`
4. Removes query parameters
5. Removes URL fragments
6. Removes trailing `/` if path is just `/`

Examples:
- `"example.com"` → `"https://example.com"`
- `"http://example.com"` → `"https://example.com"`
- `"example.com/path?q=1#frag"` → `"https://example.com/path"`
- `"  example.com/  "` → `"https://example.com"`

## Text Normalization Behavior

`NormalizeText` applies these transformations:

1. Trims whitespace
2. Unescapes HTML entities (`&amp;` → `&`)
3. Removes non-printable characters

Examples:
- `"  text  "` → `"text"`
- `"Hello &amp; goodbye"` → `"Hello & goodbye"`
- `"&lt;div&gt;"` → `"<div>"`
- `"text\x00\x01here"` → `"text  here"`

## Supported Media Extensions

The `MediaExtensions` map includes common file types:

- **Images**: `.jpg`, `.jpeg`, `.png`, `.gif`, `.svg`, `.webp`, `.bmp`, `.ico`, `.tiff`
- **Documents**: `.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.ppt`, `.pptx`
- **Video**: `.mp4`, `.avi`, `.mov`, `.wmv`, `.flv`, `.mkv`, `.m4v`
- **Audio**: `.mp3`, `.wav`, `.aac`, `.ogg`, `.flac`, `.m4a`
- **Archives**: `.zip`, `.tar`, `.gz`, `.rar`, `.7z`, `.iso`
- **Fonts**: `.ttf`, `.otf`, `.woff`, `.woff2`, `.eot`
- **Executables**: `.exe`, `.dmg`, `.apk`, `.deb`, `.rpm`, `.msi`, `.bin`, `.pkg`
- **Other**: `.css`, `.torrent`

## Related Packages

- [crawler](../crawler) - Web crawler that uses these utilities
- [fetch](../fetch) - HTTP page fetching
- [htmlparse](../htmlparse) - HTML parsing and link extraction
- [htmltomd](../htmltomd) - HTML to Markdown conversion
