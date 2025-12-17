// Package crawler provides a concurrent web crawler with pluggable fetchers,
// parsers, and caching support. It is designed for extracting structured data
// from websites while respecting domain-specific crawling rules.
//
// The crawler supports multiple follow behaviors (same domain, related subdomains,
// or any domain), custom parsing logic per domain, and configurable rate limiting.
// It uses a worker pool architecture for efficient concurrent crawling.
//
// Basic usage:
//
//	// Create a crawler with basic options
//	crawler, err := crawler.New(crawler.Options{
//		Workers:        5,
//		MaxURLs:        1000,
//		RequestDelay:   time.Second,
//		DefaultFetcher: fetch.NewMockFetcher(),
//		FollowBehavior: crawler.FollowSameDomain,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Crawl URLs and process results
//	err = crawler.Crawl(ctx, []string{"https://example.com"}, func(ctx context.Context, result *crawler.Result) {
//		if result.Error != nil {
//			log.Printf("Error crawling %s: %v", result.URL, result.Error)
//			return
//		}
//		// Process the page content
//		fmt.Printf("Crawled: %s\n", result.URL)
//	})
//
// Advanced features include domain-specific parsers and fetchers using rules:
//
//	// Add a parser for a specific domain
//	rule := crawler.NewParserRule("*.example.com", myParser,
//		crawler.WithParserMatchType(crawler.MatchGlob),
//		crawler.WithParserPriority(10))
//	crawler.AddParserRules(rule)
//
// The crawler automatically discovers and follows links based on the configured
// follow behavior, handles caching to avoid redundant fetches, and provides
// real-time statistics about crawling progress.
package crawler

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deepnoodle-ai/wonton/web"
	"github.com/deepnoodle-ai/wonton/crawler/cache"
	"github.com/deepnoodle-ai/wonton/fetch"
)

// FollowBehavior determines which discovered links the crawler will follow.
// This controls how the crawler expands beyond the initial seed URLs.
type FollowBehavior string

const (
	// FollowAny follows all discovered links regardless of domain.
	// Use with caution as this can lead to crawling the entire web.
	FollowAny FollowBehavior = "any"

	// FollowSameDomain only follows links that share the exact same hostname
	// as the page they were discovered on. For example, links on example.com
	// will only follow to other example.com pages, not to sub.example.com.
	FollowSameDomain FollowBehavior = "same-domain"

	// FollowRelatedSubdomains follows links that share the same base domain,
	// including subdomains. For example, links on example.com will follow to
	// both example.com and sub.example.com pages.
	FollowRelatedSubdomains FollowBehavior = "related-subdomains"

	// FollowNone does not follow any discovered links. Only the initial
	// seed URLs provided to Crawl() will be processed.
	FollowNone FollowBehavior = "none"
)

// Result represents the outcome of crawling a single page. It contains
// the fetched content, any parsed data, discovered links, and potential errors.
type Result struct {
	// URL is the parsed URL that was crawled
	URL *url.URL

	// Parsed contains the result of running a Parser on the page, if a parser
	// was configured for this domain. The type depends on the Parser implementation.
	Parsed any

	// Links contains all URLs discovered on the page that passed the
	// follow behavior filter. These may be enqueued for future crawling.
	Links []string

	// Response is the raw fetch response including HTML content and metadata
	Response *fetch.Response

	// Error contains any error that occurred during fetching or parsing.
	// If non-nil, other fields may be incomplete or empty.
	Error error
}

// Callback is invoked for each page processed by the crawler. It receives
// the crawl result which includes the fetched page, parsed data (if a parser
// was configured), discovered links, and any errors.
//
// The callback is called synchronously by worker goroutines, so it should
// return quickly to avoid blocking crawling progress. For expensive processing,
// consider dispatching to a separate goroutine or queue.
type Callback func(ctx context.Context, result *Result)

// Options configures a Crawler instance. Use this to specify worker count,
// rate limiting, caching, parsers, fetchers, and link-following behavior.
type Options struct {
	// MaxURLs limits the total number of URLs that will be processed.
	// Set to 0 for unlimited (use with caution).
	MaxURLs int

	// Workers specifies the number of concurrent worker goroutines.
	// More workers increase throughput but also increase load on target sites.
	Workers int

	// Cache stores fetched pages to avoid redundant requests. If nil, no caching is used.
	Cache cache.Cache

	// RequestDelay adds a delay between requests from each worker.
	// Use this to be respectful of target servers and avoid overwhelming them.
	RequestDelay time.Duration

	// KnownURLs is a list of URLs that are already known and should not be processed again.
	// This is useful for resuming interrupted crawls.
	KnownURLs []string

	// ParserRules defines domain-specific parsers. When a URL matches a rule's pattern,
	// the associated parser is used to extract structured data from the page.
	ParserRules []*ParserRule

	// DefaultParser is used for domains that don't match any ParserRule.
	// If nil and no rule matches, pages are fetched but not parsed.
	DefaultParser Parser

	// FetcherRules defines domain-specific fetchers. When a URL matches a rule's pattern,
	// the associated fetcher is used to retrieve the page.
	FetcherRules []*FetcherRule

	// DefaultFetcher is used for domains that don't match any FetcherRule.
	// This field is required - the crawler cannot function without a default fetcher.
	DefaultFetcher fetch.Fetcher

	// FollowBehavior determines which discovered links will be followed.
	// Defaults to FollowSameDomain if not specified.
	FollowBehavior FollowBehavior

	// Logger is used for debug, info, and error messages. If nil, uses slog.Default().
	Logger *slog.Logger

	// ShowProgress enables periodic logging of crawl statistics.
	ShowProgress bool

	// ShowProgressInterval controls how often progress is logged.
	// Defaults to 30 seconds if ShowProgress is true and this is not set.
	ShowProgressInterval time.Duration

	// QueueSize sets the buffer size for the URL queue.
	// Larger queues can improve throughput but use more memory.
	// Defaults to 10000 if not specified.
	QueueSize int
}

// Crawler orchestrates concurrent web crawling with configurable fetchers,
// parsers, and caching. It manages a pool of workers that process URLs from
// a queue, following links according to the configured follow behavior.
//
// Crawler is safe for concurrent use after creation, but should not be modified
// while crawling is in progress. Use New() to create instances.
type Crawler struct {
	processedURLs        sync.Map
	queue                chan string
	maxURLs              int
	workers              int
	requestDelay         time.Duration
	cache                cache.Cache
	knownURLs            []string
	parserRules          []*ParserRule
	defaultParser        Parser
	fetcherRules         []*FetcherRule
	defaultFetcher       fetch.Fetcher
	followBehavior       FollowBehavior
	activeWorkers        int64
	stats                *CrawlerStats
	logger               *slog.Logger
	running              bool
	showProgress         bool
	showProgressInterval time.Duration
	cancel               context.CancelFunc
}

// New creates a new Crawler with the specified options. It validates and sets
// default values for optional fields, compiles rule patterns, and initializes
// the worker queue.
//
// Returns an error if any parser or fetcher rules have invalid patterns, or if
// required configuration is missing.
//
// Example:
//
//	crawler, err := crawler.New(crawler.Options{
//		Workers:        5,
//		MaxURLs:        1000,
//		RequestDelay:   time.Second,
//		DefaultFetcher: fetch.NewMockFetcher(),
//		FollowBehavior: crawler.FollowSameDomain,
//	})
func New(opts Options) (*Crawler, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if opts.ShowProgress && opts.ShowProgressInterval == 0 {
		opts.ShowProgressInterval = 30 * time.Second
	}
	if opts.QueueSize <= 0 {
		opts.QueueSize = 10000
	}
	if opts.FollowBehavior == "" {
		opts.FollowBehavior = FollowSameDomain
	}
	c := &Crawler{
		cache:                opts.Cache,
		maxURLs:              opts.MaxURLs,
		workers:              opts.Workers,
		requestDelay:         opts.RequestDelay,
		defaultFetcher:       opts.DefaultFetcher,
		knownURLs:            opts.KnownURLs,
		defaultParser:        opts.DefaultParser,
		followBehavior:       opts.FollowBehavior,
		stats:                &CrawlerStats{},
		logger:               logger,
		showProgress:         opts.ShowProgress,
		showProgressInterval: opts.ShowProgressInterval,
		queue:                make(chan string, opts.QueueSize),
	}
	if err := c.AddParserRules(opts.ParserRules...); err != nil {
		return nil, err
	}
	if err := c.AddFetcherRules(opts.FetcherRules...); err != nil {
		return nil, err
	}
	return c, nil
}

// sortRulesByPriority sorts parser rules by priority (higher priority first)
func (c *Crawler) sortRulesByPriority() {
	sort.Slice(c.parserRules, func(i, j int) bool {
		return c.parserRules[i].Priority > c.parserRules[j].Priority
	})
}

// AddParserRules adds new parser rules to the crawler. The rules will be
// re-sorted by priority after adding.
func (c *Crawler) AddParserRules(rule ...*ParserRule) error {
	for _, rule := range rule {
		// Compile regex patterns if needed
		if err := rule.Compile(); err != nil {
			return err
		}
		// Add the rule
		c.parserRules = append(c.parserRules, rule)
	}
	// Re-sort by priority
	c.sortRulesByPriority()
	return nil
}

// AddFetcherRules adds new fetcher rules to the crawler. The rules will be
// re-sorted by priority after adding.
func (c *Crawler) AddFetcherRules(rules ...*FetcherRule) error {
	for _, rule := range rules {
		// Compile regex patterns if needed
		if err := rule.Compile(); err != nil {
			return err
		}
		// Add the rule
		c.fetcherRules = append(c.fetcherRules, rule)
	}
	// Re-sort by priority
	c.sortFetcherRulesByPriority()
	return nil
}

// sortFetcherRulesByPriority sorts fetcher rules by priority (higher priority first)
func (c *Crawler) sortFetcherRulesByPriority() {
	sort.Slice(c.fetcherRules, func(i, j int) bool {
		return c.fetcherRules[i].Priority > c.fetcherRules[j].Priority
	})
}

// incrementActiveWorkers atomically increments the active workers counter
func (c *Crawler) incrementActiveWorkers() {
	atomic.AddInt64(&c.activeWorkers, 1)
}

// decrementActiveWorkers atomically decrements the active workers counter
func (c *Crawler) decrementActiveWorkers() {
	atomic.AddInt64(&c.activeWorkers, -1)
}

// getActiveWorkers atomically gets the current active workers count
func (c *Crawler) getActiveWorkers() int64 {
	return atomic.LoadInt64(&c.activeWorkers)
}

// Crawl begins crawling from the provided seed URLs, invoking the callback
// for each page processed. The crawler will follow discovered links according
// to the configured FollowBehavior.
//
// Crawl blocks until all reachable URLs have been processed, the context is
// canceled, MaxURLs is reached, or an unrecoverable error occurs. Workers will
// process pages concurrently according to the Workers setting.
//
// The callback is invoked synchronously by worker goroutines for each page.
// If the callback needs to perform expensive operations, it should dispatch
// work to separate goroutines to avoid blocking the crawler.
//
// Returns an error if the crawler is already running or if the context is
// canceled before crawling begins.
//
// Example:
//
//	err := crawler.Crawl(ctx, []string{"https://example.com"}, func(ctx context.Context, result *crawler.Result) {
//		if result.Error != nil {
//			log.Printf("Error: %v", result.Error)
//			return
//		}
//		fmt.Printf("Crawled %s: found %d links\n", result.URL, len(result.Links))
//	})
func (c *Crawler) Crawl(ctx context.Context, urls []string, callback Callback) error {
	if c.running {
		return errors.New("crawler is already running")
	}
	c.running = true

	// This context will be used to stop workers when the work is done
	ctx, c.cancel = context.WithCancel(ctx)
	defer func() {
		c.running = false
		c.cancel()
		c.cancel = nil
	}()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.worker(ctx, &wg, callback)
	}
	defer close(c.queue)

	// Optionally start the progress reporter
	if c.showProgress {
		go c.progressReporter(ctx)
	}

	// Start idle monitor to detect when no more work is available
	go c.idleMonitor(ctx, c.cancel)

	// Queue initial URLs
	count, err := c.enqueue(ctx, urls)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	// Wait for workers to complete
	wg.Wait()
	return nil
}

// Stop gracefully stops the crawler by canceling its context. This signals
// workers to finish their current tasks and stop processing new URLs.
//
// Stop is safe to call concurrently and can be called multiple times.
// It does nothing if the crawler is not currently running.
func (c *Crawler) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Crawler) enqueue(ctx context.Context, urls []string) (int, error) {
	// Prevent exceeding the max URLs limit
	if c.maxURLs > 0 {
		allowedCount := c.maxURLs - int(c.stats.GetProcessed())
		if allowedCount <= 0 {
			return 0, nil
		}
		if allowedCount < len(urls) {
			urls = urls[:allowedCount]
		}
	}
	// Normalize and enqueue the URLs
	queued := 0
	for _, rawURL := range urls {
		url, err := web.NormalizeURL(rawURL)
		if err != nil {
			c.logger.Warn("invalid url",
				slog.String("url", rawURL),
				slog.String("error", err.Error()))
			continue
		}
		value := strings.TrimSuffix(url.String(), "/")
		// Only enqueue if not already processed
		if _, exists := c.processedURLs.LoadOrStore(value, true); !exists {
			select {
			case c.queue <- value:
				queued++
			case <-ctx.Done():
				return queued, ctx.Err()
			default:
				// Queue is full, skip this URL
			}
		}
	}
	return queued, nil
}

func (c *Crawler) worker(ctx context.Context, wg *sync.WaitGroup, callback Callback) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case rawURL, ok := <-c.queue:
			if !ok {
				return
			}
			c.incrementActiveWorkers()
			c.processURL(ctx, rawURL, callback)
			c.decrementActiveWorkers()
			if c.requestDelay > 0 {
				time.Sleep(c.requestDelay)
			}
		}
	}
}

func (c *Crawler) processURL(ctx context.Context, rawURL string, callback Callback) {
	c.stats.IncrementProcessed()

	// Parse the url to get its domain
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		c.logger.Warn("invalid url",
			slog.String("url", rawURL),
			slog.String("error", err.Error()))
		return
	}
	domain := parsedURL.Hostname()

	// Check cache first if one is enabled
	var response *fetch.Response
	if c.cache != nil {
		if cachedHTML, err := c.cache.Get(ctx, rawURL); err == nil {
			c.logger.Debug("cache hit", slog.String("url", rawURL))
			response = &fetch.Response{
				URL:  rawURL,
				HTML: string(cachedHTML),
			}
		}
	}

	// Get the appropriate fetcher for this domain
	fetcher, exists := c.getFetcher(domain)
	if !exists {
		c.logger.Error("no fetcher configured",
			slog.String("url", rawURL),
			slog.String("domain", domain))
		callback(ctx, &Result{URL: parsedURL, Error: errors.New("no fetcher configured for domain")})
		c.stats.IncrementFailed()
		return
	}

	// Create fetch request
	req := &fetch.Request{
		URL:             rawURL,
		Prettify:        false,
		OnlyMainContent: false,
		// Note: The Fetcher field in Request is for specifying a fetcher name/type
		// We'll leave it empty and use the actual fetcher instance directly
	}

	// Fetch if there was not a cache hit
	if response == nil {
		c.logger.Debug("fetching", slog.String("url", rawURL))
		response, err = fetcher.Fetch(ctx, req)
		if err != nil {
			callback(ctx, &Result{URL: parsedURL, Error: err})
			c.stats.IncrementFailed()
			return
		}
		if c.cache != nil && response.HTML != "" {
			if err := c.cache.Set(ctx, rawURL, []byte(response.HTML)); err != nil {
				c.logger.Warn("failed to cache html",
					slog.String("url", rawURL),
					slog.String("error", err.Error()))
			}
		}
	}

	// Parse if a parser exists for the domain
	var parsed any
	var parseErr error
	parser, exists := c.getParser(domain)
	if exists {
		c.logger.Info("parsing with domain parser",
			slog.String("url", rawURL),
			slog.String("domain", domain))
		parsed, parseErr = parser.Parse(ctx, response)
		if parseErr != nil {
			c.logger.Error("failed to parse",
				slog.String("url", rawURL),
				slog.String("error", parseErr.Error()))
		}
	}

	// Extract URLs from the page
	var discoveredLinks []string
	if response.Links != nil {
		discoveredLinks = c.extractURLs(response.Links, domain)
	}
	callback(ctx, &Result{
		URL:      parsedURL,
		Parsed:   parsed,
		Links:    discoveredLinks,
		Response: response,
		Error:    parseErr,
	})
	c.stats.IncrementSucceeded()

	filteredURLs := c.filterLinks(parsedURL, discoveredLinks)
	if _, err := c.enqueue(ctx, filteredURLs); err != nil {
		c.logger.Warn("failed to enqueue discovered urls",
			slog.String("url", rawURL),
			slog.String("error", err.Error()))
	}
}

func (c *Crawler) getParser(domain string) (Parser, bool) {
	// Check parser rules (already sorted by priority)
	for _, rule := range c.parserRules {
		if rule.Matches(domain) {
			return rule.Parser, true
		}
	}
	// Fall back to default parser
	if c.defaultParser != nil {
		return c.defaultParser, true
	}
	return nil, false
}

// getFetcher returns the appropriate fetcher for the given domain based on rules
func (c *Crawler) getFetcher(domain string) (fetch.Fetcher, bool) {
	// Check fetcher rules (already sorted by priority)
	for _, rule := range c.fetcherRules {
		if rule.Matches(domain) {
			return rule.Fetcher, true
		}
	}
	// Fall back to default fetcher
	if c.defaultFetcher != nil {
		return c.defaultFetcher, true
	}
	return nil, false
}

func (c *Crawler) filterLinks(pageURL *url.URL, links []string) []string {
	if c.followBehavior == FollowNone {
		return nil
	}
	var filtered []string
	for _, rawURL := range links {
		u, err := web.NormalizeURL(rawURL)
		if err != nil {
			continue
		}
		switch c.followBehavior {
		case FollowAny:
			filtered = append(filtered, rawURL)
		case FollowSameDomain:
			if web.AreSameHost(u, pageURL) {
				filtered = append(filtered, rawURL)
			}
		case FollowRelatedSubdomains:
			if web.AreRelatedHosts(u, pageURL) {
				filtered = append(filtered, rawURL)
			}
		}
	}
	return filtered
}

func (c *Crawler) extractURLs(links []fetch.Link, domain string) []string {
	urlMap := make(map[string]bool)
	for _, link := range links {
		if url, ok := web.ResolveLink(domain, link.URL); ok {
			urlMap[url] = true
		}
	}
	var results []string
	for url := range urlMap {
		results = append(results, url)
	}
	sort.Strings(results)
	return results
}

func (c *Crawler) progressReporter(ctx context.Context) {
	ticker := time.NewTicker(c.showProgressInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.logger.Info("crawl progress",
				slog.Int64("processed", c.stats.GetProcessed()),
				slog.Int64("succeeded", c.stats.GetSucceeded()),
				slog.Int64("failed", c.stats.GetFailed()))
		}
	}
}

// GetStats returns the current crawling statistics, including counts of
// processed, succeeded, and failed URLs. The returned statistics are safe
// to read concurrently and reflect the latest state.
//
// The statistics continue to accumulate across multiple calls to Crawl()
// on the same Crawler instance.
func (c *Crawler) GetStats() *CrawlerStats {
	return c.stats
}

func (c *Crawler) idleMonitor(ctx context.Context, cancel context.CancelFunc) {
	// Check every second for idle state
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if we're idle: no active workers and queue is empty
			if c.getActiveWorkers() == 0 && len(c.queue) == 0 {
				c.logger.Info("no more work available, stopping crawler")
				cancel() // Cancel context to stop all workers
				return
			}
		}
	}
}
