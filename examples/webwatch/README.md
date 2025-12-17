# webwatch

Monitor web pages for content changes. Periodically fetches URLs, converts to markdown, and displays colorized unified diffs when changes are detected.

## Features

- Periodic monitoring with configurable check intervals
- Automatic retry with exponential backoff on fetch failures
- Converts HTML to clean markdown for better diff quality
- Colorized unified diff output highlighting additions/deletions
- Quiet mode to only show output when changes detected
- One-time check mode for CI/CD integration
- Graceful shutdown with Ctrl+C

## Usage

```bash
# Monitor a URL with default 60-second interval
go run examples/webwatch/main.go watch https://example.com

# Monitor with custom interval (30 seconds)
go run examples/webwatch/main.go watch --interval 30 https://example.com/pricing

# Check once and exit (useful for scripts/CI)
go run examples/webwatch/main.go watch --once https://docs.example.com

# Quiet mode - only show output when changes detected
go run examples/webwatch/main.go watch --quiet --interval 120 https://example.com/blog

# Short flags
go run examples/webwatch/main.go watch -i 60 -q https://example.com
```

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--interval` | `-i` | Check interval in seconds | 60 |
| `--once` | | Check once and exit | false |
| `--quiet` | `-q` | Only show output when changes detected | false |

## Example Output

### Initial Check
```
Monitoring: https://example.com
Interval: 60s
────────────────────────────────────────────────────────────────────────────────

✓ Check #1 - 2025-12-17 14:28:03
→ Initial snapshot captured (5 lines, 167 B)

→ Next check in 1m (Press Ctrl+C to stop)
```

### When Changes Detected
```
⚡ Check #3 - 2025-12-17 14:30:15
CHANGES DETECTED! Generating diff...

→ Statistics:
  Files changed: 1
  Additions:     +3
  Deletions:     -2

→ Diff:
────────────────────────────────────────────────────────────────────────────────
content.md
@@ -1,5 +1,6 @@
 # Example Domain

-This domain is for use in illustrative examples.
+This domain is for use in illustrative examples in documents.
+You may use this domain in examples.

 [More information...](https://www.iana.org/domains/example)
────────────────────────────────────────────────────────────────────────────────
```

## Use Cases

- **Documentation monitoring**: Track when documentation pages are updated
- **Pricing page tracking**: Monitor competitor pricing changes
- **Blog updates**: Get notified when new blog posts appear
- **API documentation**: Track API changes from documentation
- **Terms of Service**: Monitor changes to legal documents
- **CI/CD integration**: Check for unexpected changes in deployed sites

## Packages Used

This example demonstrates the integration of several Wonton packages:

- **cli**: Command-line interface framework with flags and arguments
- **fetch**: HTTP fetching with content extraction
- **htmltomd**: HTML to Markdown conversion for clean diffs
- **unidiff**: Unified diff parsing and display
- **color**: Colorized terminal output
- **humanize**: Human-readable formatting (bytes, durations, times)
- **retry**: Automatic retry with exponential backoff

## Implementation Notes

- Uses `fetch` with `OnlyMainContent: true` to extract just the main content area
- Converts HTML to markdown before comparison for cleaner, more meaningful diffs
- Implements retry logic with exponential backoff for resilient fetching
- Uses signal handling for graceful shutdown with summary statistics
- Stores previous content in memory for comparison (not persistent across restarts)

## Future Enhancements

Possible improvements for production use:

- Persistent storage of previous content (file or database)
- Multiple URL monitoring in parallel
- Webhook notifications when changes detected
- Email alerts for changes
- Configurable selectors to monitor specific page sections
- Change history with timestamps
- Diff filtering to ignore certain types of changes
