// Command crawl is a web crawler CLI with TUI display.
//
// Usage:
//
//	crawl <command> [options]
//
// Commands:
//
//	crawl           Crawl a website starting from seed URLs
//	fetch           Fetch and display a single URL
//	links           Extract and display links from a URL
//	meta            Extract and display metadata from a URL
package main

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/crawler"
	"github.com/deepnoodle-ai/wonton/fetch"
	"github.com/deepnoodle-ai/wonton/htmltomd"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	app := cli.New("crawl").
		Description("Web crawler CLI with rich terminal display").
		Version("1.0.0")

	// crawl command - crawl a website
	app.Command("crawl").
		Description("Crawl a website starting from seed URLs").
		Args("urls...").
		Flags(
			cli.Int("workers", "w").Default(4).Help("Number of concurrent workers"),
			cli.Int("max", "m").Default(100).Help("Maximum URLs to crawl"),
			cli.String("delay", "d").Default("100ms").Help("Delay between requests"),
			cli.String("follow", "f").Default("same-domain").
				Enum("none", "same-domain", "subdomains", "any").
				Help("Link following behavior"),
			cli.Bool("interactive", "i").Help("Show interactive TUI display"),
		).
		Run(runCrawl)

	// fetch command - fetch a single URL
	app.Command("fetch").
		Description("Fetch and display a single URL").
		Args("url").
		Flags(
			cli.Bool("markdown", "m").Help("Convert to markdown"),
			cli.Bool("raw", "r").Help("Show raw HTML"),
			cli.Bool("main", "M").Help("Extract only main content"),
			cli.String("timeout", "t").Default("30s").Help("Request timeout"),
		).
		Run(runFetch)

	// links command - extract links
	app.Command("links").
		Description("Extract and display links from a URL").
		Args("url").
		Flags(
			cli.Bool("internal", "i").Help("Show only internal links"),
			cli.Bool("external", "e").Help("Show only external links"),
			cli.Bool("interactive", "I").Help("Show interactive TUI display"),
		).
		Run(runLinks)

	// meta command - extract metadata
	app.Command("meta").
		Description("Extract and display metadata from a URL").
		Args("url").
		Flags(
			cli.Bool("json", "j").Help("Output as JSON"),
		).
		Run(runMeta)

	app.Execute()
}

// runCrawl handles the crawl command
func runCrawl(ctx *cli.Context) error {
	urls := ctx.Args()
	if len(urls) == 0 {
		return fmt.Errorf("at least one URL is required")
	}

	workers := ctx.Int("workers")
	maxURLs := ctx.Int("max")
	delayStr := ctx.String("delay")
	delay, err := time.ParseDuration(delayStr)
	if err != nil {
		return fmt.Errorf("invalid delay: %w", err)
	}
	followStr := ctx.String("follow")
	interactive := ctx.Bool("interactive")

	var follow crawler.FollowBehavior
	switch followStr {
	case "none":
		follow = crawler.FollowNone
	case "same-domain":
		follow = crawler.FollowSameDomain
	case "subdomains":
		follow = crawler.FollowRelatedSubdomains
	case "any":
		follow = crawler.FollowAny
	}

	fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{})

	c, err := crawler.New(crawler.Options{
		Workers:        workers,
		MaxURLs:        maxURLs,
		RequestDelay:   delay,
		DefaultFetcher: fetcher,
		FollowBehavior: follow,
	})
	if err != nil {
		return fmt.Errorf("failed to create crawler: %w", err)
	}

	if interactive && ctx.Interactive() {
		return runCrawlTUI(ctx.Context(), c, urls)
	}

	return runCrawlSimple(ctx, c, urls)
}

// runCrawlSimple runs the crawler with simple text output
func runCrawlSimple(ctx *cli.Context, c *crawler.Crawler, urls []string) error {
	var mu sync.Mutex
	var succeeded, failed int

	err := c.Crawl(ctx.Context(), urls, func(_ context.Context, result *crawler.Result) {
		mu.Lock()
		defer mu.Unlock()

		if result.Error != nil {
			failed++
			ctx.Fail("  %s: %v", result.URL, result.Error)
		} else {
			succeeded++
			title := ""
			if result.Response != nil {
				title = result.Response.Metadata.Title
			}
			if title != "" {
				ctx.Success("  %s - %s", result.URL, title)
			} else {
				ctx.Success("  %s", result.URL)
			}
		}
	})
	if err != nil {
		return err
	}

	ctx.Info("\nCrawl complete: %d succeeded, %d failed", succeeded, failed)
	return nil
}

// CrawlApp is the TUI application for crawling
type CrawlApp struct {
	crawler    *crawler.Crawler
	urls       []string
	results    []crawlResult
	mu         sync.Mutex
	done       bool
	startTime  time.Time
	cancelFunc context.CancelFunc
}

type crawlResult struct {
	url     string
	title   string
	status  string
	links   int
	errMsg  string
	fetched time.Time
}

func (app *CrawlApp) View() tui.View {
	app.mu.Lock()
	defer app.mu.Unlock()

	stats := app.crawler.GetStats()
	elapsed := time.Since(app.startTime).Round(time.Second)

	var statusText string
	var statusColor tui.Color
	if app.done {
		statusText = "Complete"
		statusColor = tui.ColorGreen
	} else {
		statusText = "Crawling..."
		statusColor = tui.ColorYellow
	}

	// Build table rows
	rows := make([][]string, len(app.results))
	for i, r := range app.results {
		statusIcon := "+"
		if r.errMsg != "" {
			statusIcon = "x"
		}
		title := r.title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		rows[i] = []string{statusIcon, r.url, title, fmt.Sprintf("%d", r.links)}
	}

	var selected int
	return tui.Stack(
		tui.Group(
			tui.Text("Web Crawler").Bold().Fg(tui.ColorCyan),
			tui.Spacer(),
			tui.Text("%s", statusText).Fg(statusColor),
		),
		tui.Text(""),
		tui.Group(
			tui.Text("Processed: %d", stats.GetProcessed()).Fg(tui.ColorWhite),
			tui.Text(" | "),
			tui.Text("Succeeded: %d", stats.GetSucceeded()).Fg(tui.ColorGreen),
			tui.Text(" | "),
			tui.Text("Failed: %d", stats.GetFailed()).Fg(tui.ColorRed),
			tui.Text(" | "),
			tui.Text("Elapsed: %s", elapsed).Fg(tui.ColorBlue),
		),
		tui.Text(""),
		tui.Table(
			[]tui.TableColumn{{Title: ""}, {Title: "URL"}, {Title: "Title"}, {Title: "Links"}},
			&selected,
		).Rows(rows).Height(20).ShowHeader(true),
		tui.Text(""),
		tui.Text("Press 'q' to quit, 's' to stop crawling").Dim(),
	).Padding(1)
}

func (app *CrawlApp) HandleEvent(event tui.Event) []tui.Cmd {
	if key, ok := event.(tui.KeyEvent); ok {
		switch key.Rune {
		case 'q':
			app.cancelFunc()
			return []tui.Cmd{tui.Quit()}
		case 's':
			app.crawler.Stop()
		}
	}
	return nil
}

func runCrawlTUI(ctx context.Context, c *crawler.Crawler, urls []string) error {
	ctx, cancel := context.WithCancel(ctx)

	app := &CrawlApp{
		crawler:    c,
		urls:       urls,
		startTime:  time.Now(),
		cancelFunc: cancel,
	}

	// Run crawler in background
	go func() {
		c.Crawl(ctx, urls, func(_ context.Context, result *crawler.Result) {
			app.mu.Lock()
			defer app.mu.Unlock()

			r := crawlResult{
				url:     result.URL.String(),
				links:   len(result.Links),
				fetched: time.Now(),
			}
			if result.Error != nil {
				r.status = "error"
				r.errMsg = result.Error.Error()
			} else {
				r.status = "ok"
				if result.Response != nil {
					r.title = result.Response.Metadata.Title
				}
			}
			app.results = append(app.results, r)
		})
		app.mu.Lock()
		app.done = true
		app.mu.Unlock()
	}()

	return tui.Run(app,
		tui.WithAlternateScreen(true),
		tui.WithFPS(10),
	)
}

// runFetch handles the fetch command
func runFetch(ctx *cli.Context) error {
	rawURL := ctx.Arg(0)
	if rawURL == "" {
		return fmt.Errorf("URL is required")
	}

	// Ensure URL has scheme
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	showMarkdown := ctx.Bool("markdown")
	showRaw := ctx.Bool("raw")
	mainOnly := ctx.Bool("main")
	timeoutStr := ctx.String("timeout")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}

	fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
		Timeout: timeout,
	})

	formats := []string{"html", "links"}
	if showMarkdown {
		formats = append(formats, "markdown")
	}
	if showRaw {
		formats = append(formats, "raw_html")
	}

	req := &fetch.Request{
		URL:             rawURL,
		OnlyMainContent: mainOnly,
		Formats:         formats,
	}

	resp, err := fetcher.Fetch(ctx.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Display results
	ctx.Success("Fetched: %s", resp.URL)
	ctx.Info("Status: %d", resp.StatusCode)

	if resp.Metadata.Title != "" {
		ctx.Info("Title: %s", resp.Metadata.Title)
	}
	if resp.Metadata.Description != "" {
		ctx.Info("Description: %s", resp.Metadata.Description)
	}

	fmt.Println()

	if showMarkdown && resp.Markdown != "" {
		fmt.Println("--- Markdown ---")
		fmt.Println(resp.Markdown)
	} else if showRaw && resp.RawHTML != "" {
		fmt.Println("--- Raw HTML ---")
		fmt.Println(resp.RawHTML)
	} else if resp.HTML != "" {
		// Convert to markdown for display
		md := htmltomd.Convert(resp.HTML)
		fmt.Println(md)
	}

	return nil
}

// runLinks handles the links command
func runLinks(ctx *cli.Context) error {
	rawURL := ctx.Arg(0)
	if rawURL == "" {
		return fmt.Errorf("URL is required")
	}

	// Ensure URL has scheme
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	internalOnly := ctx.Bool("internal")
	externalOnly := ctx.Bool("external")
	interactive := ctx.Bool("interactive")

	fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{})

	req := &fetch.Request{
		URL:     rawURL,
		Formats: []string{"links"},
	}

	resp, err := fetcher.Fetch(ctx.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Parse base URL for filtering
	baseURL, _ := url.Parse(rawURL)

	// Filter and deduplicate links
	linkMap := make(map[string]fetch.Link)
	for _, link := range resp.Links {
		if link.URL == "" {
			continue
		}

		// Parse link URL
		linkURL, err := url.Parse(link.URL)
		if err != nil {
			continue
		}

		// Resolve relative URLs
		if !linkURL.IsAbs() {
			linkURL = baseURL.ResolveReference(linkURL)
		}

		isInternal := linkURL.Host == baseURL.Host

		if internalOnly && !isInternal {
			continue
		}
		if externalOnly && isInternal {
			continue
		}

		link.URL = linkURL.String()
		linkMap[link.URL] = link
	}

	// Sort links
	var links []fetch.Link
	for _, link := range linkMap {
		links = append(links, link)
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].URL < links[j].URL
	})

	if interactive && ctx.Interactive() {
		return runLinksTUI(links, baseURL)
	}

	// Simple output
	ctx.Success("Found %d links on %s", len(links), rawURL)
	fmt.Println()

	for _, link := range links {
		if link.Text != "" {
			fmt.Printf("  %s\n    %s\n", link.Text, link.URL)
		} else {
			fmt.Printf("  %s\n", link.URL)
		}
	}

	return nil
}

// LinksApp is the TUI application for displaying links
type LinksApp struct {
	links    []fetch.Link
	baseURL  *url.URL
	selected int
}

func (app *LinksApp) View() tui.View {
	// Prepare items for the list
	items := make([]string, len(app.links))
	for i, link := range app.links {
		linkURL, _ := url.Parse(link.URL)
		var prefix string
		if linkURL != nil && linkURL.Host == app.baseURL.Host {
			prefix = "[int]"
		} else {
			prefix = "[ext]"
		}

		text := link.Text
		if len(text) > 30 {
			text = text[:27] + "..."
		}
		if text == "" {
			text = "(no text)"
		}

		items[i] = fmt.Sprintf("%s %s - %s", prefix, text, link.URL)
	}

	return tui.Stack(
		tui.Group(
			tui.Text("Links").Bold().Fg(tui.ColorCyan),
			tui.Spacer(),
			tui.Text("%d total", len(app.links)).Dim(),
		),
		tui.Text(""),
		tui.FilterableListStrings(items, &app.selected).Height(20),
		tui.Text(""),
		tui.Text("Type to filter, arrow keys to navigate, 'q' to quit").Dim(),
	).Padding(1)
}

func (app *LinksApp) HandleEvent(event tui.Event) []tui.Cmd {
	if key, ok := event.(tui.KeyEvent); ok {
		if key.Rune == 'q' {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func runLinksTUI(links []fetch.Link, baseURL *url.URL) error {
	app := &LinksApp{
		links:   links,
		baseURL: baseURL,
	}
	return tui.Run(app,
		tui.WithAlternateScreen(true),
		tui.WithFPS(30),
	)
}

// runMeta handles the meta command
func runMeta(ctx *cli.Context) error {
	rawURL := ctx.Arg(0)
	if rawURL == "" {
		return fmt.Errorf("URL is required")
	}

	// Ensure URL has scheme
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	outputJSON := ctx.Bool("json")

	fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{})

	req := &fetch.Request{
		URL:     rawURL,
		Formats: []string{"branding"},
	}

	resp, err := fetcher.Fetch(ctx.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	meta := resp.Metadata

	if outputJSON {
		// Simple JSON-like output
		fmt.Printf("{\n")
		fmt.Printf("  \"url\": %q,\n", rawURL)
		fmt.Printf("  \"title\": %q,\n", meta.Title)
		fmt.Printf("  \"description\": %q,\n", meta.Description)
		fmt.Printf("  \"author\": %q,\n", meta.Author)
		fmt.Printf("  \"canonical\": %q,\n", meta.Canonical)
		fmt.Printf("  \"charset\": %q,\n", meta.Charset)
		fmt.Printf("  \"robots\": %q", meta.Robots)
		if meta.OpenGraph != nil {
			fmt.Printf(",\n  \"opengraph\": {\n")
			fmt.Printf("    \"title\": %q,\n", meta.OpenGraph.Title)
			fmt.Printf("    \"description\": %q,\n", meta.OpenGraph.Description)
			fmt.Printf("    \"image\": %q,\n", meta.OpenGraph.Image)
			fmt.Printf("    \"type\": %q\n", meta.OpenGraph.Type)
			fmt.Printf("  }")
		}
		if meta.Twitter != nil {
			fmt.Printf(",\n  \"twitter\": {\n")
			fmt.Printf("    \"card\": %q,\n", meta.Twitter.Card)
			fmt.Printf("    \"title\": %q,\n", meta.Twitter.Title)
			fmt.Printf("    \"description\": %q,\n", meta.Twitter.Description)
			fmt.Printf("    \"image\": %q\n", meta.Twitter.Image)
			fmt.Printf("  }")
		}
		fmt.Printf("\n}\n")
		return nil
	}

	// Pretty output
	ctx.Success("Metadata for %s", rawURL)
	fmt.Println()

	printMeta := func(label, value string) {
		if value != "" {
			fmt.Printf("  %-14s %s\n", label+":", value)
		}
	}

	printMeta("Title", meta.Title)
	printMeta("Description", meta.Description)
	printMeta("Author", meta.Author)
	printMeta("Canonical", meta.Canonical)
	printMeta("Charset", meta.Charset)
	printMeta("Viewport", meta.Viewport)
	printMeta("Robots", meta.Robots)

	if len(meta.Keywords) > 0 {
		printMeta("Keywords", strings.Join(meta.Keywords, ", "))
	}

	if meta.OpenGraph != nil {
		fmt.Println()
		ctx.Info("Open Graph:")
		printMeta("  Title", meta.OpenGraph.Title)
		printMeta("  Description", meta.OpenGraph.Description)
		printMeta("  Image", meta.OpenGraph.Image)
		printMeta("  URL", meta.OpenGraph.URL)
		printMeta("  Type", meta.OpenGraph.Type)
		printMeta("  Site Name", meta.OpenGraph.SiteName)
	}

	if meta.Twitter != nil {
		fmt.Println()
		ctx.Info("Twitter Card:")
		printMeta("  Card", meta.Twitter.Card)
		printMeta("  Title", meta.Twitter.Title)
		printMeta("  Description", meta.Twitter.Description)
		printMeta("  Image", meta.Twitter.Image)
		printMeta("  Site", meta.Twitter.Site)
		printMeta("  Creator", meta.Twitter.Creator)
	}

	if resp.Branding != nil {
		fmt.Println()
		ctx.Info("Branding:")
		printMeta("  Logo", resp.Branding.Logo)
		printMeta("  Color Scheme", resp.Branding.ColorScheme)
		if resp.Branding.Colors != nil {
			printMeta("  Primary Color", resp.Branding.Colors.Primary)
		}
		if resp.Branding.Images != nil {
			printMeta("  Favicon", resp.Branding.Images.Favicon)
			printMeta("  OG Image", resp.Branding.Images.OGImage)
		}
	}

	return nil
}
