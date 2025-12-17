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

// FollowBehavior is used to determine how to follow links.
type FollowBehavior string

const (
	FollowAny               FollowBehavior = "any"
	FollowSameDomain        FollowBehavior = "same-domain"
	FollowRelatedSubdomains FollowBehavior = "related-subdomains"
	FollowNone              FollowBehavior = "none"
)

// Result represents the result of one page being crawled.
type Result struct {
	URL      *url.URL
	Parsed   any
	Links    []string
	Response *fetch.Response
	Error    error
}

// ProcessCallback is called with the fetch request and parsed result (if any)
type Callback func(ctx context.Context, result *Result)

// Options used to configure a crawler.
type Options struct {
	MaxURLs              int
	Workers              int
	Cache                cache.Cache
	RequestDelay         time.Duration
	KnownURLs            []string
	ParserRules          []*ParserRule
	DefaultParser        Parser
	FetcherRules         []*FetcherRule
	DefaultFetcher       fetch.Fetcher
	FollowBehavior       FollowBehavior
	Logger               *slog.Logger
	ShowProgress         bool
	ShowProgressInterval time.Duration
	QueueSize            int
}

// Crawler is used to crawl the web.
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

// New creates a new crawler.
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

// Crawl the provided URLs and call the callback for each processed page.
// Links may be followed depending on the configured follow behavior.
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

// GetStats returns the current crawling statistics
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
