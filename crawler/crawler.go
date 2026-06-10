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

	"github.com/deepnoodle-ai/wonton/crawler/cache"
	"github.com/deepnoodle-ai/wonton/fetch"
	"github.com/deepnoodle-ai/wonton/htmlparse"
	"github.com/deepnoodle-ai/wonton/retry"
	"github.com/deepnoodle-ai/wonton/web"
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
	// Defaults to 1 if not specified.
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
	// If the queue fills up, newly discovered URLs are dropped (and logged);
	// they may be picked up again if rediscovered later in the crawl.
	// Defaults to 10000 if not specified.
	QueueSize int

	// AllowHTTP permits crawling HTTP URLs without upgrading to HTTPS.
	// By default, all URLs are normalized to HTTPS for security.
	// Enable this for crawling HTTP-only sites or internal networks.
	AllowHTTP bool

	// PreserveQueryParams keeps URL query parameters during normalization.
	// By default, query parameters are stripped to reduce URL duplication.
	// Enable this for sites that use query parameters for pagination or content.
	PreserveQueryParams bool

	// RetryOptions configures retry behavior for failed fetch requests.
	// If nil, no retries are performed.
	RetryOptions *RetryOptions

	// RespectRobotsTxt enables robots.txt compliance.
	// When enabled, the crawler will fetch and respect robots.txt rules,
	// including Crawl-delay directives (applied per worker, in place of
	// RequestDelay when larger).
	// Defaults to true. Set to false to disable robots.txt checking.
	RespectRobotsTxt *bool

	// RobotsTxtUserAgent is the user agent string used when checking robots.txt rules.
	// Defaults to "*" if not specified.
	RobotsTxtUserAgent string
}

// RetryOptions configures retry behavior for failed fetch requests.
type RetryOptions struct {
	// MaxAttempts is the maximum number of retry attempts (including the first).
	// Defaults to 3 if not specified.
	MaxAttempts int

	// InitialBackoff is the delay before the first retry.
	// Defaults to 1 second if not specified.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum delay between retries.
	// Defaults to 30 seconds if not specified.
	MaxBackoff time.Duration
}

// BoolPtr returns a pointer to the given bool value.
// This is a helper for setting optional bool fields in Options.
//
// Example:
//
//	crawler.New(crawler.Options{
//		RespectRobotsTxt: crawler.BoolPtr(false), // disable robots.txt
//	})
func BoolPtr(b bool) *bool {
	return &b
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
	queueSize            int
	maxURLs              int
	workers              int
	requestDelay         time.Duration
	cache                cache.Cache
	parserRules          []*ParserRule
	defaultParser        Parser
	fetcherRules         []*FetcherRule
	defaultFetcher       fetch.Fetcher
	followBehavior       FollowBehavior
	stats                *CrawlerStats
	logger               *slog.Logger
	showProgress         bool
	showProgressInterval time.Duration

	// mu protects running and cancel
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc

	// pendingURLs counts URLs that have been queued but not fully processed.
	// The crawl is complete when this reaches zero.
	pendingURLs int64

	// enqueuedURLs counts URLs admitted to the queue, used to enforce maxURLs.
	enqueuedURLs int64

	// URL normalization options, derived from AllowHTTP and
	// PreserveQueryParams at construction time
	normalizeOpts []web.NormalizeOption

	// Retry configuration
	retryOptions *RetryOptions

	// robots.txt support
	respectRobotsTxt   bool
	robotsTxtUserAgent string
	robotsCache        sync.Map // map[string]*robotsTxtData
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
	if opts.Workers <= 0 {
		opts.Workers = 1
	}
	if opts.QueueSize <= 0 {
		opts.QueueSize = 10000
	}
	if opts.FollowBehavior == "" {
		opts.FollowBehavior = FollowSameDomain
	}
	if opts.RobotsTxtUserAgent == "" {
		opts.RobotsTxtUserAgent = "*"
	}
	// Default RespectRobotsTxt to true
	respectRobotsTxt := true
	if opts.RespectRobotsTxt != nil {
		respectRobotsTxt = *opts.RespectRobotsTxt
	}
	var normalizeOpts []web.NormalizeOption
	if opts.AllowHTTP {
		normalizeOpts = append(normalizeOpts, web.KeepHTTP())
	}
	if opts.PreserveQueryParams {
		normalizeOpts = append(normalizeOpts, web.KeepQuery())
	}
	c := &Crawler{
		cache:                opts.Cache,
		maxURLs:              opts.MaxURLs,
		workers:              opts.Workers,
		requestDelay:         opts.RequestDelay,
		defaultFetcher:       opts.DefaultFetcher,
		defaultParser:        opts.DefaultParser,
		followBehavior:       opts.FollowBehavior,
		stats:                &CrawlerStats{},
		logger:               logger,
		showProgress:         opts.ShowProgress,
		showProgressInterval: opts.ShowProgressInterval,
		queueSize:            opts.QueueSize,
		normalizeOpts:        normalizeOpts,
		retryOptions:         opts.RetryOptions,
		respectRobotsTxt:     respectRobotsTxt,
		robotsTxtUserAgent:   opts.RobotsTxtUserAgent,
	}
	// Seed the processed set with already-known URLs so they are not
	// crawled again. This supports resuming interrupted crawls.
	for _, rawURL := range opts.KnownURLs {
		if normalized, err := c.normalizeURL(rawURL); err == nil {
			c.processedURLs.Store(strings.TrimSuffix(normalized.String(), "/"), true)
		}
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
// A Crawler can be reused: once Crawl returns, it may be called again. URLs
// processed by earlier calls are remembered and will not be crawled again,
// and statistics continue to accumulate.
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
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return errors.New("crawler is already running")
	}
	c.running = true
	// Create a fresh queue for each crawl so the Crawler can be reused.
	c.queue = make(chan string, c.queueSize)
	atomic.StoreInt64(&c.pendingURLs, 0)
	// This context is used to stop workers when the work is done
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.mu.Unlock()

	defer func() {
		cancel()
		c.mu.Lock()
		c.running = false
		c.cancel = nil
		c.mu.Unlock()
	}()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.worker(ctx, &wg, callback)
	}

	// Optionally start the progress reporter
	if c.showProgress {
		go c.progressReporter(ctx)
	}

	// Queue initial URLs
	count, err := c.enqueue(ctx, urls)
	if err != nil || count == 0 {
		// Nothing to crawl (or the context was canceled): stop the workers
		// and wait for them to exit.
		cancel()
	} else {
		// Start idle monitor to detect when all queued work is complete
		go c.idleMonitor(ctx, cancel)
	}

	// Wait for workers to complete
	wg.Wait()
	return err
}

// Stop gracefully stops the crawler by canceling its context. This signals
// workers to finish their current tasks and stop processing new URLs.
//
// Stop is safe to call concurrently and can be called multiple times.
// It does nothing if the crawler is not currently running.
func (c *Crawler) Stop() {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Crawler) enqueue(ctx context.Context, urls []string) (int, error) {
	queued := 0
	for _, rawURL := range urls {
		normalizedURL, err := c.normalizeURL(rawURL)
		if err != nil {
			c.logger.Warn("invalid url",
				slog.String("url", rawURL),
				slog.String("error", err.Error()))
			continue
		}
		value := strings.TrimSuffix(normalizedURL.String(), "/")
		// Atomically check-and-mark as seen so concurrent workers cannot
		// enqueue the same URL twice.
		if _, seen := c.processedURLs.LoadOrStore(value, true); seen {
			continue
		}
		// Reserve a slot against the max URLs limit before queueing.
		if c.maxURLs > 0 && atomic.AddInt64(&c.enqueuedURLs, 1) > int64(c.maxURLs) {
			atomic.AddInt64(&c.enqueuedURLs, -1)
			c.processedURLs.Delete(value)
			return queued, nil
		}
		select {
		case c.queue <- value:
			atomic.AddInt64(&c.pendingURLs, 1)
			queued++
		case <-ctx.Done():
			c.releaseURL(value)
			return queued, ctx.Err()
		default:
			// Queue is full - unmark the URL so it can be retried if it is
			// discovered again later.
			c.releaseURL(value)
			c.logger.Warn("queue full, dropping url",
				slog.String("url", value))
		}
	}
	return queued, nil
}

// releaseURL undoes the bookkeeping performed before a URL is queued, used
// when queueing fails.
func (c *Crawler) releaseURL(value string) {
	c.processedURLs.Delete(value)
	if c.maxURLs > 0 {
		atomic.AddInt64(&c.enqueuedURLs, -1)
	}
}

func (c *Crawler) worker(ctx context.Context, wg *sync.WaitGroup, callback Callback) {
	defer wg.Done()
	queue := c.queue
	for {
		select {
		case <-ctx.Done():
			return
		case rawURL, ok := <-queue:
			if !ok {
				return
			}
			delay := c.processURL(ctx, rawURL, callback)
			// Decrement pending only after processing completes, so that any
			// links discovered (and queued) by processURL are counted first
			// and the idle monitor never observes a false zero.
			atomic.AddInt64(&c.pendingURLs, -1)
			if delay > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
				}
			}
		}
	}
}

// processURL fetches, parses, and reports a single URL. It returns the delay
// the worker should apply before processing its next URL, which is the larger
// of the configured RequestDelay and any robots.txt Crawl-delay for the host.
func (c *Crawler) processURL(ctx context.Context, rawURL string, callback Callback) time.Duration {
	c.stats.IncrementProcessed()

	// Parse the url to get its domain
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		c.logger.Warn("invalid url",
			slog.String("url", rawURL),
			slog.String("error", err.Error()))
		return c.requestDelay
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
			// The cache stores only HTML, not the links discovered at fetch
			// time, so re-extract them to keep link following working on
			// cache hits.
			if doc, err := htmlparse.Parse(response.HTML); err == nil {
				response.Links = doc.Links()
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
		return c.requestDelay
	}

	// Fetch if there was not a cache hit
	if response == nil {
		// Check robots.txt if enabled. Cache hits skip this check (and the
		// robots.txt fetch it may trigger) since no network request is made.
		if c.respectRobotsTxt && !c.isAllowedByRobots(ctx, parsedURL) {
			c.logger.Debug("blocked by robots.txt",
				slog.String("url", rawURL))
			callback(ctx, &Result{URL: parsedURL, Error: errors.New("blocked by robots.txt")})
			c.stats.IncrementFailed()
			return c.delayFor(parsedURL)
		}

		req := &fetch.Request{
			URL:             rawURL,
			Prettify:        false,
			OnlyMainContent: false,
			Formats:         []string{"html", "links"},
		}
		c.logger.Debug("fetching", slog.String("url", rawURL))

		// Use retry logic if configured
		if c.retryOptions != nil {
			maxAttempts := c.retryOptions.MaxAttempts
			if maxAttempts <= 0 {
				maxAttempts = 3
			}
			initialBackoff := c.retryOptions.InitialBackoff
			if initialBackoff <= 0 {
				initialBackoff = time.Second
			}
			maxBackoff := c.retryOptions.MaxBackoff
			if maxBackoff <= 0 {
				maxBackoff = 30 * time.Second
			}

			response, err = retry.Do(ctx, func() (*fetch.Response, error) {
				return fetcher.Fetch(ctx, req)
			},
				retry.WithMaxAttempts(maxAttempts),
				retry.WithBackoff(initialBackoff, maxBackoff),
				retry.WithOnRetry(func(attempt int, err error, delay time.Duration) {
					c.logger.Warn("retrying fetch",
						slog.String("url", rawURL),
						slog.Int("attempt", attempt),
						slog.String("error", err.Error()),
						slog.Duration("delay", delay))
				}),
			)
		} else {
			response, err = fetcher.Fetch(ctx, req)
		}

		if err != nil {
			callback(ctx, &Result{URL: parsedURL, Error: err})
			c.stats.IncrementFailed()
			return c.delayFor(parsedURL)
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
		discoveredLinks = c.extractURLs(response.Links, parsedURL)
	}
	callback(ctx, &Result{
		URL:      parsedURL,
		Parsed:   parsed,
		Links:    discoveredLinks,
		Response: response,
		Error:    parseErr,
	})
	if parseErr != nil {
		c.stats.IncrementFailed()
	} else {
		c.stats.IncrementSucceeded()
	}

	filteredURLs := c.filterLinks(parsedURL, discoveredLinks)
	if _, err := c.enqueue(ctx, filteredURLs); err != nil {
		c.logger.Warn("failed to enqueue discovered urls",
			slog.String("url", rawURL),
			slog.String("error", err.Error()))
	}
	return c.delayFor(parsedURL)
}

// delayFor returns the delay a worker should apply after processing a URL:
// the configured RequestDelay, or the host's robots.txt Crawl-delay if that
// is larger. Only already-cached robots.txt data is consulted; this never
// triggers a robots.txt fetch.
func (c *Crawler) delayFor(parsedURL *url.URL) time.Duration {
	delay := c.requestDelay
	if c.respectRobotsTxt {
		if d := c.cachedRobotsCrawlDelay(parsedURL); d > delay {
			delay = d
		}
	}
	return delay
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
		u, err := c.normalizeURL(rawURL)
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

func (c *Crawler) extractURLs(links []fetch.Link, baseURL *url.URL) []string {
	urlMap := make(map[string]bool)
	for _, link := range links {
		if resolved, ok := c.resolveLink(baseURL, link.URL); ok {
			urlMap[resolved] = true
		}
	}
	var results []string
	for u := range urlMap {
		results = append(results, u)
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
	// Check periodically for idle state
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// The crawl is complete when no queued URLs remain. The counter
			// is decremented only after a URL is fully processed (including
			// enqueueing any discovered links), so a zero reading is final.
			if atomic.LoadInt64(&c.pendingURLs) == 0 {
				c.logger.Debug("no more work available, stopping crawler")
				cancel() // Cancel context to stop all workers
				return
			}
		}
	}
}

// normalizeURL parses and normalizes a URL according to the crawler's configuration.
func (c *Crawler) normalizeURL(rawURL string) (*url.URL, error) {
	return web.NormalizeURL(rawURL, c.normalizeOpts...)
}

// resolveLink resolves a relative or absolute URL against the page URL it
// appeared on, applying the crawler's normalization configuration.
func (c *Crawler) resolveLink(baseURL *url.URL, link string) (string, bool) {
	resolved, err := web.ResolveLink(baseURL, link, c.normalizeOpts...)
	if err != nil {
		return "", false
	}
	return resolved.String(), true
}
