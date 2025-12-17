// Example: Site Link Checker
//
// Crawls a website and checks every link for broken references.
// Shows live progress in a TUI with color-coded results and human-readable stats.
//
// Run with:
//
//	go run examples/sitecheck/main.go https://example.com
//	go run examples/sitecheck/main.go --max-urls 50 https://example.com
//	go run examples/sitecheck/main.go --workers 10 https://example.com
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/crawler"
	"github.com/deepnoodle-ai/wonton/fetch"
	"github.com/deepnoodle-ai/wonton/humanize"
	"github.com/deepnoodle-ai/wonton/retry"
	"github.com/deepnoodle-ai/wonton/tui"
	"github.com/deepnoodle-ai/wonton/web"
)

// LinkStatus represents the status of a checked link
type LinkStatus struct {
	URL        string
	StatusCode int
	Error      error
	Timestamp  time.Time
}

// SiteCheckApp holds the state for the TUI
type SiteCheckApp struct {
	mu sync.Mutex

	// Crawling state
	totalChecked int
	totalOK      int
	totalBroken  int
	currentURL   string
	startTime    time.Time
	done         bool

	// Recent results
	recentResults []LinkStatus
	maxRecent     int

	// Track checked URLs to avoid duplicates
	checkedURLs map[string]bool

	// Error for display
	fatalError error
}

// NewSiteCheckApp creates a new app instance
func NewSiteCheckApp() *SiteCheckApp {
	return &SiteCheckApp{
		maxRecent:     10,
		recentResults: make([]LinkStatus, 0),
		checkedURLs:   make(map[string]bool),
		startTime:     time.Now(),
	}
}

// Update adds a link check result
func (app *SiteCheckApp) Update(status LinkStatus) {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.totalChecked++
	if status.Error == nil && status.StatusCode >= 200 && status.StatusCode < 400 {
		app.totalOK++
	} else {
		app.totalBroken++
	}

	// Add to recent results (keep last N)
	app.recentResults = append(app.recentResults, status)
	if len(app.recentResults) > app.maxRecent {
		app.recentResults = app.recentResults[1:]
	}
}

// SetCurrent updates the currently processing URL
func (app *SiteCheckApp) SetCurrent(url string) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.currentURL = url
}

// SetDone marks the crawl as complete
func (app *SiteCheckApp) SetDone() {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.done = true
}

// SetError sets a fatal error
func (app *SiteCheckApp) SetError(err error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.fatalError = err
	app.done = true
}

// HandleEvent processes TUI events
func (app *SiteCheckApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

// View renders the TUI
func (app *SiteCheckApp) View() tui.View {
	app.mu.Lock()
	defer app.mu.Unlock()

	elapsed := time.Since(app.startTime)

	// Header
	header := tui.HeaderBar("Site Link Checker").
		Bg(tui.ColorBlue).
		Fg(tui.ColorWhite)

	// Stats section
	statsColor := tui.ColorGreen
	if app.totalBroken > 0 {
		statsColor = tui.ColorYellow
	}
	if app.totalBroken > 10 {
		statsColor = tui.ColorRed
	}

	stats := tui.Group(
		tui.Text("Checked: %s", humanize.Number(int64(app.totalChecked))).
			Fg(statsColor).Bold(),
		tui.Spacer().MinWidth(3),
		tui.Text("OK: %s", humanize.Number(int64(app.totalOK))).
			Fg(tui.ColorGreen),
		tui.Spacer().MinWidth(3),
		tui.Text("Broken: %s", humanize.Number(int64(app.totalBroken))).
			Fg(tui.ColorRed),
		tui.Spacer().MinWidth(3),
		tui.Text("Elapsed: %s", humanize.Duration(elapsed)).
			Fg(tui.ColorCyan),
	)

	// Current URL being processed
	var currentView tui.View
	if app.done {
		if app.fatalError != nil {
			currentView = tui.Text("Error: %v", app.fatalError).Fg(tui.ColorRed).Bold()
		} else {
			currentView = tui.Text("Crawl complete!").Fg(tui.ColorGreen).Bold()
		}
	} else {
		if app.currentURL != "" {
			currentView = tui.Group(
				tui.Text("Checking:").Fg(tui.ColorBrightBlack),
				tui.Spacer().MinWidth(1),
				tui.Text("%s", app.currentURL).Fg(tui.ColorWhite).MaxWidth(80),
			)
		} else {
			currentView = tui.Text("Starting...").Fg(tui.ColorBrightBlack)
		}
	}

	// Recent results
	var resultViews []tui.View
	resultViews = append(resultViews, tui.Text("Recent results:").Bold().Fg(tui.ColorWhite))
	resultViews = append(resultViews, tui.Spacer().MinHeight(1))

	if len(app.recentResults) == 0 {
		resultViews = append(resultViews, tui.Text("No results yet...").Fg(tui.ColorBrightBlack))
	} else {
		for i := len(app.recentResults) - 1; i >= 0; i-- {
			result := app.recentResults[i]
			resultViews = append(resultViews, app.formatResult(result))
		}
	}

	// Footer
	footer := tui.Text("Press 'q' or ESC to quit").Fg(tui.ColorBrightBlack)

	// Build the layout
	return tui.Stack(
		header,
		tui.Spacer().MinHeight(1),
		stats,
		tui.Spacer().MinHeight(1),
		currentView,
		tui.Spacer().MinHeight(1),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
		tui.Stack(resultViews...),
		tui.Spacer(),
		footer,
	).Padding(2)
}

// formatResult formats a single link check result
func (app *SiteCheckApp) formatResult(status LinkStatus) tui.View {
	// Determine status icon and color
	var statusIcon string
	var statusColor color.Color
	var statusText string

	if status.Error != nil {
		statusIcon = "✗"
		statusColor = tui.ColorRed
		statusText = fmt.Sprintf("Error: %v", status.Error)
	} else if status.StatusCode >= 200 && status.StatusCode < 300 {
		statusIcon = "✓"
		statusColor = tui.ColorGreen
		statusText = fmt.Sprintf("OK (%d)", status.StatusCode)
	} else if status.StatusCode >= 300 && status.StatusCode < 400 {
		statusIcon = "→"
		statusColor = tui.ColorYellow
		statusText = fmt.Sprintf("Redirect (%d)", status.StatusCode)
	} else if status.StatusCode >= 400 && status.StatusCode < 500 {
		statusIcon = "✗"
		statusColor = tui.ColorRed
		statusText = fmt.Sprintf("Client Error (%d)", status.StatusCode)
	} else if status.StatusCode >= 500 {
		statusIcon = "✗"
		statusColor = tui.ColorRed
		statusText = fmt.Sprintf("Server Error (%d)", status.StatusCode)
	} else {
		statusIcon = "?"
		statusColor = tui.ColorYellow
		statusText = fmt.Sprintf("Unknown (%d)", status.StatusCode)
	}

	// Format status text with fixed width for alignment
	paddedStatusText := fmt.Sprintf("%-20s", statusText)

	return tui.Group(
		tui.Text("%s", statusIcon).Fg(statusColor).Bold(),
		tui.Spacer().MinWidth(1),
		tui.Text("%s", paddedStatusText).Fg(statusColor),
		tui.Spacer().MinWidth(1),
		tui.Text("%s", status.URL).Fg(tui.ColorWhite),
	)
}

// checkLink checks a single link and returns its status
func checkLink(ctx context.Context, url string) LinkStatus {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, just record them
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return LinkStatus{
			URL:       url,
			Error:     err,
			Timestamp: time.Now(),
		}
	}

	// Set a reasonable user agent
	req.Header.Set("User-Agent", "SiteCheck/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return LinkStatus{
			URL:       url,
			Error:     err,
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()

	return LinkStatus{
		URL:        url,
		StatusCode: resp.StatusCode,
		Timestamp:  time.Now(),
	}
}

// runCrawler starts the crawler in the background
func runCrawler(ctx context.Context, app *SiteCheckApp, startURL string, maxURLs, workers int) {
	go func() {
		// Normalize the start URL
		normalizedURL, err := web.NormalizeURL(startURL)
		if err != nil {
			app.SetError(fmt.Errorf("invalid URL: %w", err))
			return
		}

		// Create crawler
		c, err := crawler.New(crawler.Options{
			MaxURLs:        maxURLs,
			Workers:        workers,
			DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
			FollowBehavior: crawler.FollowSameDomain,
			RequestDelay:   500 * time.Millisecond, // Be polite
		})
		if err != nil {
			app.SetError(fmt.Errorf("failed to create crawler: %w", err))
			return
		}

		// Crawl and check each link
		err = c.Crawl(ctx, []string{normalizedURL.String()}, func(ctx context.Context, result *crawler.Result) {
			if result.Error != nil {
				app.Update(LinkStatus{
					URL:       result.URL.String(),
					Error:     result.Error,
					Timestamp: time.Now(),
				})
				return
			}

			app.SetCurrent(result.URL.String())

			// Check the main URL if not already checked
			pageURL := result.URL.String()
			app.mu.Lock()
			alreadyChecked := app.checkedURLs[pageURL]
			if !alreadyChecked {
				app.checkedURLs[pageURL] = true
			}
			app.mu.Unlock()

			if !alreadyChecked {
				status := checkLink(ctx, pageURL)
				app.Update(status)
			}

			// Check all links on this page (including external links)
			for _, link := range result.Links {
				// Skip media files
				linkURL, err := web.NormalizeURL(link)
				if err != nil {
					continue
				}
				if web.IsMediaURL(linkURL) {
					continue
				}

				// Check if we've already checked this URL
				normalizedLink := linkURL.String()
				app.mu.Lock()
				alreadyCheckedLink := app.checkedURLs[normalizedLink]
				if !alreadyCheckedLink {
					app.checkedURLs[normalizedLink] = true
				}
				app.mu.Unlock()

				if alreadyCheckedLink {
					continue
				}

				app.SetCurrent(normalizedLink)

				// Use retry for checking links
				err = retry.DoSimple(ctx, func() error {
					linkStatus := checkLink(ctx, normalizedLink)
					app.Update(linkStatus)
					return nil
				},
					retry.WithMaxAttempts(2),
					retry.WithBackoff(100*time.Millisecond, 1*time.Second),
				)
				if err != nil {
					// Context cancelled, stop
					return
				}
			}
		})

		if err != nil && err != context.Canceled {
			app.SetError(fmt.Errorf("crawl error: %w", err))
			return
		}

		app.SetDone()
	}()
}

func main() {
	app := cli.New("sitecheck").
		Description("Crawl a website and check for broken links").
		Version("1.0.0")

	// Main command
	app.Command("").
		Description("Check a website for broken links").
		Args("url").
		Flags(
			cli.Int("max-urls", "m").
				Default(100).
				Help("Maximum number of URLs to crawl"),
			cli.Int("workers", "w").
				Default(5).
				Help("Number of concurrent workers"),
		).
		Run(func(ctx *cli.Context) error {
			startURL := ctx.Arg(0)
			maxURLs := ctx.Int("max-urls")
			workers := ctx.Int("workers")

			// Validate URL
			if startURL == "" {
				return cli.Error("URL is required").
					Hint("Usage: sitecheck https://example.com")
			}

			// Create the TUI app
			tuiApp := NewSiteCheckApp()

			// Start the crawler in the background
			crawlerCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

			runCrawler(crawlerCtx, tuiApp, startURL, maxURLs, workers)

			// Run the TUI
			if err := tui.Run(tuiApp); err != nil {
				return err
			}

			// Print final summary
			tuiApp.mu.Lock()
			defer tuiApp.mu.Unlock()

			fmt.Println()
			fmt.Println("=== Final Summary ===")
			fmt.Printf("Total checked: %s links\n", humanize.Number(int64(tuiApp.totalChecked)))
			fmt.Printf("OK: %s\n", humanize.Number(int64(tuiApp.totalOK)))
			fmt.Printf("Broken: %s\n", humanize.Number(int64(tuiApp.totalBroken)))
			fmt.Printf("Time: %s\n", humanize.Duration(time.Since(tuiApp.startTime)))

			if tuiApp.totalBroken > 0 {
				return cli.Exit(1)
			}

			return nil
		})

	// Run the CLI
	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}
