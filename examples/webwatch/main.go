// Example: Web Page Monitor
//
// Monitors web pages for content changes by periodically fetching URLs,
// converting to markdown, and displaying colorized unified diffs in the
// terminal when changes are detected.
//
// Useful for tracking documentation updates, pricing page changes, or
// monitoring competitors' websites.
//
// Run with:
//
//	go run examples/webwatch/main.go watch https://example.com
//	go run examples/webwatch/main.go watch --interval 30 https://example.com/pricing
//	go run examples/webwatch/main.go watch -i 60 --once https://docs.example.com
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
	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/fetch"
	"github.com/deepnoodle-ai/wonton/humanize"
	"github.com/deepnoodle-ai/wonton/retry"
	"github.com/deepnoodle-ai/wonton/unidiff"
)

// Config defines the CLI flags for webwatch
type Config struct {
	Interval int  `flag:"interval,i" default:"60" help:"Check interval in seconds"`
	Once     bool `flag:"once" help:"Check once and exit"`
	Quiet    bool `flag:"quiet,q" help:"Only show output when changes detected"`
}

func main() {
	app := cli.New("webwatch").
		Description("Monitor web pages for content changes").
		Version("1.0.0")

	cmd := app.Command("watch").
		Description("Monitor a URL for changes").
		Args("url")

	cli.ParseFlags[Config](cmd)

	cmd.Run(func(ctx *cli.Context) error {
		config, err := cli.BindFlags[Config](ctx)
		if err != nil {
			return err
		}

		url := ctx.Arg(0)
		if url == "" {
			return cli.Error("URL is required")
		}

		return monitorURL(url, *config)
	})

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func monitorURL(url string, config Config) error {
	interval := time.Duration(config.Interval) * time.Second

	// Create HTTP fetcher
	fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
		Timeout: 30 * time.Second,
	})

	// Store previous content
	var previousContent string
	var lastCheck time.Time
	checkCount := 0

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	printHeader(url, interval, config.Once)

	for {
		checkCount++
		lastCheck = time.Now()

		// Fetch and convert to markdown with retry
		content, err := fetchWithRetry(fetcher, url)
		if err != nil {
			printError(checkCount, err)
			if config.Once {
				return err
			}
			// Continue to next iteration on error
		} else {
			// Check for changes
			if previousContent == "" {
				// First fetch
				printInitial(checkCount, content)
				previousContent = content
			} else if content != previousContent {
				// Content changed
				printChange(checkCount, previousContent, content)
				previousContent = content
			} else {
				// No change
				if !config.Quiet {
					printNoChange(checkCount)
				}
			}
		}

		// Exit if running in once mode
		if config.Once {
			break
		}

		// Wait for next check or signal
		if !config.Quiet {
			fmt.Printf("\n%s Next check in %s (Press Ctrl+C to stop)\n",
				color.Cyan.Apply("→"),
				humanize.Duration(interval))
		}

		select {
		case <-time.After(interval):
			// Continue to next check
		case <-sigChan:
			// Graceful shutdown
			fmt.Printf("\n%s Stopping monitoring...\n",
				color.Yellow.Apply("⚠"))
			printSummary(checkCount, lastCheck)
			return nil
		}
	}

	if config.Once {
		printSummary(checkCount, lastCheck)
	}

	return nil
}

func fetchWithRetry(fetcher fetch.Fetcher, url string) (string, error) {
	ctx := context.Background()

	result, err := retry.Do(ctx, func() (string, error) {
		req := &fetch.Request{
			URL:             url,
			OnlyMainContent: true,
			Formats:         []string{"markdown"},
		}

		resp, err := fetcher.Fetch(ctx, req)
		if err != nil {
			return "", err
		}

		return resp.Markdown, nil
	},
		retry.WithMaxAttempts(3),
		retry.WithBackoff(500*time.Millisecond, 5*time.Second),
		retry.WithOnRetry(func(attempt int, err error, delay time.Duration) {
			fmt.Printf("%s Retry attempt %d after error: %v (waiting %s)\n",
				color.Yellow.Apply("⚠"),
				attempt,
				err,
				humanize.Duration(delay))
		}),
	)

	return result, err
}

func printHeader(url string, interval time.Duration, once bool) {
	fmt.Printf("%s %s\n",
		color.BrightBlue.Apply("Monitoring:"),
		url)

	if !once {
		fmt.Printf("%s %s\n",
			color.BrightBlue.Apply("Interval:"),
			humanize.Duration(interval))
	}

	fmt.Println(strings.Repeat("─", 80))
}

func printInitial(checkCount int, content string) {
	lines := strings.Count(content, "\n") + 1
	bytes := len(content)

	fmt.Printf("\n%s Check #%d - %s\n",
		color.Green.Apply("✓"),
		checkCount,
		time.Now().Format("2006-01-02 15:04:05"))

	fmt.Printf("%s Initial snapshot captured (%d lines, %s)\n",
		color.Cyan.Apply("→"),
		lines,
		humanize.Bytes(int64(bytes)))
}

func printNoChange(checkCount int) {
	fmt.Printf("\n%s Check #%d - %s\n",
		color.Green.Apply("✓"),
		checkCount,
		time.Now().Format("2006-01-02 15:04:05"))

	fmt.Printf("%s No changes detected\n",
		color.Cyan.Apply("→"))
}

func printChange(checkCount int, oldContent, newContent string) {
	fmt.Printf("\n%s Check #%d - %s\n",
		color.BrightYellow.Apply("⚡"),
		checkCount,
		time.Now().Format("2006-01-02 15:04:05"))

	fmt.Printf("%s %s\n",
		color.BrightYellow.Apply("CHANGES DETECTED!"),
		"Generating diff...")

	// Generate diff
	diffText := generateDiff(oldContent, newContent)

	// Parse and display diff
	diff, err := unidiff.Parse(diffText)
	if err != nil {
		fmt.Printf("%s Failed to parse diff: %v\n",
			color.Red.Apply("✗"),
			err)
		return
	}

	// Print statistics
	stats := diff.Stats()
	fmt.Printf("\n%s Statistics:\n",
		color.Cyan.Apply("→"))
	fmt.Printf("  Files changed: %d\n", stats.FilesChanged)
	fmt.Printf("  Additions:     %s\n", color.Green.Apply(fmt.Sprintf("+%d", stats.Additions)))
	fmt.Printf("  Deletions:     %s\n", color.Red.Apply(fmt.Sprintf("-%d", stats.Deletions)))

	// Print colorized diff
	fmt.Printf("\n%s Diff:\n", color.Cyan.Apply("→"))
	fmt.Println(strings.Repeat("─", 80))
	printColorizedDiff(diff)
	fmt.Println(strings.Repeat("─", 80))
}

func printError(checkCount int, err error) {
	fmt.Printf("\n%s Check #%d - %s\n",
		color.Red.Apply("✗"),
		checkCount,
		time.Now().Format("2006-01-02 15:04:05"))

	fmt.Printf("%s Error: %v\n",
		color.Red.Apply("✗"),
		err)
}

func printSummary(checkCount int, lastCheck time.Time) {
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%s Summary:\n", color.Cyan.Apply("→"))
	fmt.Printf("  Total checks: %d\n", checkCount)
	fmt.Printf("  Last check:   %s (%s)\n",
		lastCheck.Format("2006-01-02 15:04:05"),
		humanize.Time(lastCheck))
}

func generateDiff(oldContent, newContent string) string {
	// Simple unified diff generation
	// In a real implementation, you might use a proper diff algorithm
	// For now, we'll create a basic diff format that unidiff can parse

	var diff strings.Builder

	diff.WriteString("diff --git a/content.md b/content.md\n")
	diff.WriteString("--- a/content.md\n")
	diff.WriteString("+++ b/content.md\n")

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Simple line-by-line comparison
	// This is a basic implementation - a real diff would use LCS algorithm
	diff.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(oldLines), len(newLines)))

	// Find common prefix
	commonPrefix := 0
	for i := 0; i < len(oldLines) && i < len(newLines); i++ {
		if oldLines[i] == newLines[i] {
			commonPrefix++
		} else {
			break
		}
	}

	// Find common suffix
	commonSuffix := 0
	for i := 0; i < len(oldLines)-commonPrefix && i < len(newLines)-commonPrefix; i++ {
		if oldLines[len(oldLines)-1-i] == newLines[len(newLines)-1-i] {
			commonSuffix++
		} else {
			break
		}
	}

	// Print context lines before changes
	contextStart := max(0, commonPrefix-3)
	for i := contextStart; i < commonPrefix; i++ {
		diff.WriteString(" " + oldLines[i] + "\n")
	}

	// Print removed lines
	for i := commonPrefix; i < len(oldLines)-commonSuffix; i++ {
		diff.WriteString("-" + oldLines[i] + "\n")
	}

	// Print added lines
	for i := commonPrefix; i < len(newLines)-commonSuffix; i++ {
		diff.WriteString("+" + newLines[i] + "\n")
	}

	// Print context lines after changes
	contextEnd := min(len(oldLines), len(oldLines)-commonSuffix+3)
	for i := len(oldLines) - commonSuffix; i < contextEnd; i++ {
		diff.WriteString(" " + oldLines[i] + "\n")
	}

	return diff.String()
}

func printColorizedDiff(diff *unidiff.Diff) {
	for _, file := range diff.Files {
		// File header
		fmt.Printf("%s\n", color.ApplyBold(file.NewPath))

		for _, hunk := range file.Hunks {
			// Hunk header in cyan
			fmt.Printf("%s\n", color.Cyan.Apply(hunk.Header))

			for _, line := range hunk.Lines {
				switch line.Type {
				case unidiff.LineAdded:
					// Green for additions
					fmt.Printf("%s\n", color.Green.Apply("+"+line.Content))
				case unidiff.LineRemoved:
					// Red for deletions
					fmt.Printf("%s\n", color.Red.Apply("-"+line.Content))
				case unidiff.LineContext:
					// Normal for context
					fmt.Printf(" %s\n", line.Content)
				}
			}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
