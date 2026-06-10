// Example: URL to Markdown Clipboard Watcher
//
// Watches your clipboard for URLs. When you copy a URL, it automatically
// fetches the page, converts it to clean markdown, and replaces your
// clipboard content - perfect for pasting into notes or LLM prompts.
//
// Run with:
//
//	go run examples/urlx/main.go
//	go run examples/urlx/main.go --interval 2s
//	go run examples/urlx/main.go --help
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/fetch"
	"github.com/deepnoodle-ai/wonton/htmltomd"
	"github.com/deepnoodle-ai/wonton/retry"
	"github.com/deepnoodle-ai/wonton/web"
)

func main() {
	app := cli.New("urlx").
		Description("Watch clipboard for URLs and convert them to markdown").
		Version("0.1.0")

	app.Main().
		Long(`Continuously monitors your clipboard for URLs. When a URL is detected,
it fetches the page content, converts it to clean markdown, and replaces
the clipboard content with the markdown version.

This is useful for quickly capturing web content for notes, documentation,
or feeding to LLMs.`).
		Flags(
			cli.Duration("interval", "i").
				Default(time.Second).
				Help("How often to check the clipboard"),
			cli.Duration("timeout", "t").
				Default(30*time.Second).
				Help("HTTP request timeout"),
			cli.Int("retries", "r").
				Default(3).
				Help("Number of retry attempts for failed fetches"),
		).
		Run(runWatch)

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func runWatch(ctx *cli.Context) error {
	// Check if clipboard is available
	if !clipboard.Available() {
		return fmt.Errorf("clipboard not available on this system")
	}

	interval := ctx.Duration("interval")
	if interval <= 0 {
		return cli.Error("invalid --interval: expected a positive duration like 2s")
	}

	timeout := ctx.Duration("timeout")
	if timeout <= 0 {
		return cli.Error("invalid --timeout: expected a positive duration like 30s")
	}

	maxRetries := ctx.Int("retries")

	ctx.Printf("Watching clipboard every %v...\n", interval)
	ctx.Println("Press Ctrl+C to stop")

	// Set up signal handling for graceful shutdown
	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create HTTP fetcher
	fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
		Timeout: timeout,
	})

	// Keep track of last processed content to avoid reprocessing
	var lastContent string

	// Create ticker for polling
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-sigCtx.Done():
			ctx.Println("\nShutting down...")
			return nil
		case <-ticker.C:
			// Read clipboard
			content, err := clipboard.Read()
			if err != nil {
				// Don't fail on read errors, just continue
				continue
			}

			// Skip if content hasn't changed
			if content == lastContent || content == "" {
				continue
			}

			// Check if content is a URL
			if !isURL(content) {
				lastContent = content
				continue
			}

			ctx.Printf("\nDetected URL: %s\n", content)

			// Fetch and convert
			markdown, err := fetchAndConvert(sigCtx, fetcher, content, maxRetries)
			if err != nil {
				ctx.Printf("Error: %v\n", err)
				lastContent = content
				continue
			}

			// Write markdown back to clipboard
			if err := clipboard.Write(markdown); err != nil {
				ctx.Printf("Error writing to clipboard: %v\n", err)
				lastContent = content
				continue
			}

			ctx.Println("Converted to markdown and updated clipboard!")
			lastContent = markdown
		}
	}
}

// isURL checks if the string looks like a URL
func isURL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \t\n") {
		return false
	}

	// web.NormalizeURL handles bare hostnames, missing schemes, etc.
	u, err := web.NormalizeURL(s)
	if err != nil {
		return false
	}

	// Require an explicit scheme or a dotted hostname so ordinary words
	// copied to the clipboard (e.g. "hello") aren't treated as URLs.
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.Contains(u.Hostname(), ".")
}

// fetchAndConvert fetches a URL and converts it to markdown with retries
func fetchAndConvert(ctx context.Context, fetcher fetch.Fetcher, urlStr string, maxRetries int) (string, error) {
	// Normalize the URL
	normalizedURL, err := web.NormalizeURL(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Create fetch request
	req := &fetch.Request{
		URL:     normalizedURL.String(),
		Formats: []string{"html"},
	}

	// Fetch with retries
	var resp *fetch.Response
	err = retry.DoSimple(ctx, func() error {
		var fetchErr error
		resp, fetchErr = fetcher.Fetch(ctx, req)
		return fetchErr
	},
		retry.WithMaxAttempts(maxRetries),
		retry.WithBackoff(500*time.Millisecond, 5*time.Second),
		retry.WithOnRetry(func(attempt int, err error, delay time.Duration) {
			fmt.Printf("Retry %d/%d after %v: %v\n", attempt, maxRetries-1, delay, err)
		}),
	)

	if err != nil {
		return "", fmt.Errorf("failed to fetch: %w", err)
	}

	// Check if we got HTML content
	if resp.HTML == "" {
		return "", fmt.Errorf("no HTML content received")
	}

	// Convert to markdown
	markdown := htmltomd.Convert(resp.HTML)

	// Add metadata header
	title := resp.Metadata.Title
	if title == "" {
		title = normalizedURL.Hostname()
	}
	header := fmt.Sprintf("# %s\n\nSource: %s\n\n---\n\n", title, urlStr)
	if resp.Metadata.Description != "" {
		header += fmt.Sprintf("*%s*\n\n---\n\n", resp.Metadata.Description)
	}

	return header + markdown, nil
}
