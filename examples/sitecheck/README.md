# Site Link Checker Example

A command-line tool that crawls a website and checks every link for broken references. Shows live progress in a TUI with color-coded results and human-readable stats.

## Features

- Crawls websites following same-domain links
- Checks all links (including external links) for broken references
- Live TUI display with real-time progress
- Color-coded results:
  - ✓ Green: OK (200-299 status codes)
  - → Yellow: Redirects (300-399 status codes)
  - ✗ Red: Errors (400-599 status codes or network errors)
- Human-readable statistics (e.g., "Checked 1,234 links in 2m 30s")
- Automatic retry with exponential backoff for transient failures
- Skips media files (images, PDFs, videos, etc.)
- Deduplicates URL checks to avoid checking the same link multiple times

## Usage

Run the tool by providing a URL to check:

```bash
go run examples/sitecheck/main.go https://example.com
```

### Options

- `--max-urls`, `-m`: Maximum number of URLs to crawl (default: 100)
- `--workers`, `-w`: Number of concurrent workers (default: 5)

### Examples

Check a website with default settings:
```bash
go run examples/sitecheck/main.go https://example.com
```

Check up to 50 URLs:
```bash
go run examples/sitecheck/main.go --max-urls 50 https://example.com
```

Use 10 concurrent workers for faster crawling:
```bash
go run examples/sitecheck/main.go --workers 10 https://example.com
```

Check a specific subdomain:
```bash
go run examples/sitecheck/main.go https://blog.example.com
```

## TUI Controls

- `q` or `ESC`: Quit the application
- `Ctrl+C`: Force quit

## Packages Used

This example demonstrates the following Wonton packages:

- **cli**: Command-line interface framework with flags and arguments
- **crawler**: Web crawler that follows links and crawls pages
- **fetch**: HTTP fetching with configurable options
- **htmlparse**: HTML parsing and link extraction (used by crawler)
- **web**: URL normalization and validation
- **retry**: Retry logic with exponential backoff
- **tui**: Terminal UI framework for live progress display
- **color**: ANSI color types for styling
- **humanize**: Human-readable formatting for numbers and durations

## Implementation Notes

- The crawler respects a 500ms delay between requests to be polite
- HEAD requests are used for link checking (faster than GET)
- Links are checked with a 10-second timeout
- Failed requests are retried once with exponential backoff
- The crawler only follows links on the same domain
- All discovered links (including external) are checked for validity
- Media files are automatically skipped based on file extension
- URLs are deduplicated to avoid redundant checks

## Exit Codes

- `0`: Success (no broken links found)
- `1`: Failure (broken links found or error occurred)
