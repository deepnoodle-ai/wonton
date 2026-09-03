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
	"net/http"
	"net/url"
	"sort"
	"strconv"
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

	// Depth is the number of links between this page and its seed URL.
	Depth int

	// Referrer is the URL of the page where this URL was discovered. It is
	// empty for seed URLs.
	Referrer string

	// DiscoveredAt is when this URL was admitted to the crawl frontier.
	DiscoveredAt time.Time

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

	// Cache stores fetched pages to avoid redundant requests. Caches that also
	// implement cache.ResponseCache preserve HTTP metadata and extracted links.
	// If nil, no caching is used.
	Cache cache.Cache

	// CacheTTL controls how long typed response-cache entries are served
	// without a network request. Zero preserves the legacy behavior and treats
	// entries as fresh indefinitely.
	CacheTTL time.Duration

	// Revalidate conditionally refreshes stale typed cache entries with ETag or
	// Last-Modified validators. It has no effect on HTML-only Cache
	// implementations or when CacheTTL is zero.
	Revalidate bool

	// RequestDelay sets a minimum delay between requests to the same host.
	// The delay is enforced per host, so an unrelated host does not stall while
	// another host cools down. HostPolicy.MinDelay is used when it is larger.
	RequestDelay time.Duration

	// Frontier stores pending crawl work. If nil, a MemoryFrontier is created
	// for each call to Crawl so the Crawler remains reusable. A provided
	// frontier may be preloaded and crawled with an empty seed list.
	Frontier Frontier

	// HostPolicy controls per-host concurrency and request spacing.
	HostPolicy HostPolicy

	// Adaptive enables per-host backoff for 429/503 responses and elevated
	// latency. Retry-After is honored and throttled URLs are retried up to the
	// configured RetryOptions.MaxAttempts, or three attempts by default.
	Adaptive bool

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
	// It may be nil when every crawled URL can be served from Cache.
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

	// QueueSize bounds normalized URLs across the default MemoryFrontier, host
	// scheduler, active requests, and retries. Custom frontiers control their
	// own storage capacity while the scheduler applies this limit to staged and
	// active work. Raw link batches await normalization in a separate nonblocking
	// broker so workers cannot deadlock behind frontier backpressure. A fetcher
	// may return an arbitrarily large raw batch, which is not part of this limit;
	// MaxURLs separately limits total admissions. URLs are never silently
	// dropped. Defaults to 10000 if not specified.
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
	// including per-host Crawl-delay directives when they are larger than the
	// configured minimum delay.
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
	frontier             Frontier
	frontierProvided     bool
	queueSize            int
	maxURLs              int
	workers              int
	requestDelay         time.Duration
	hostPolicy           HostPolicy
	cache                cache.Cache
	responseCache        cache.ResponseCache
	cacheTTL             time.Duration
	revalidate           bool
	adaptive             bool
	hostStats            *hostStatsCollector
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

	// inFlightURLs is used with pendingURLs to report the number of items that
	// have not started yet.
	inFlightURLs int64

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
// Returns an error if any parser or fetcher rules have invalid patterns.
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
	if opts.HostPolicy.MaxConcurrent <= 0 {
		opts.HostPolicy.MaxConcurrent = 1
	}
	if opts.Adaptive && opts.HostPolicy.MaxDelay <= 0 {
		opts.HostPolicy.MaxDelay = defaultAdaptiveMaxDelay
	}
	if opts.RequestDelay > opts.HostPolicy.MinDelay {
		opts.HostPolicy.MinDelay = opts.RequestDelay
	}
	if opts.HostPolicy.MaxDelay < opts.HostPolicy.MinDelay {
		opts.HostPolicy.MaxDelay = opts.HostPolicy.MinDelay
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
	frontier := opts.Frontier
	frontierProvided := frontier != nil
	if frontier == nil {
		frontier = NewMemoryFrontier(opts.QueueSize)
	}
	c := &Crawler{
		cache:                opts.Cache,
		cacheTTL:             opts.CacheTTL,
		revalidate:           opts.Revalidate,
		adaptive:             opts.Adaptive,
		hostStats:            newHostStatsCollector(),
		frontier:             frontier,
		frontierProvided:     frontierProvided,
		maxURLs:              opts.MaxURLs,
		workers:              opts.Workers,
		requestDelay:         opts.RequestDelay,
		hostPolicy:           opts.HostPolicy,
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
	if responseCache, ok := opts.Cache.(cache.ResponseCache); ok {
		c.responseCache = responseCache
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
	parentCtx := ctx
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return errors.New("crawler is already running")
	}
	c.running = true
	// Create a fresh default frontier for each crawl so canceled work from one
	// run cannot leak into the next. Caller-provided frontiers retain their own
	// lifecycle and must be empty before the first crawl.
	if !c.frontierProvided {
		c.frontier = NewMemoryFrontier(c.queueSize)
	}
	initialPending := c.frontier.Len()
	atomic.StoreInt64(&c.pendingURLs, int64(initialPending))
	atomic.StoreInt64(&c.inFlightURLs, 0)
	c.hostStats.reset()
	// This context is used to stop workers when the work is done
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	frontier := c.frontier
	c.mu.Unlock()

	defer func() {
		cancel()
		c.mu.Lock()
		c.running = false
		c.cancel = nil
		c.mu.Unlock()
	}()

	scheduler := newHostScheduler(frontier, c.hostPolicy, c.adaptive, c.hostStats, c.queueSize)
	scheduler.start(ctx, cancel)
	admitter := newFrontierAdmitter(c, frontier, cancel)
	admitter.start(ctx)

	if len(urls) == 0 && initialPending == 0 {
		cancel()
		admitter.close()
		admitter.wait()
		scheduler.wait()
		if err := scheduler.failure(); err != nil {
			return err
		}
		return parentCtx.Err()
	}

	// Start workers before admitting seeds so a bounded frontier can drain while
	// the admission goroutine applies backpressure.
	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.worker(ctx, cancel, scheduler, admitter, &wg, callback)
	}
	seedErr := admitter.submit(ctx, urls, nil)
	if seedErr != nil {
		cancel()
	}

	if c.showProgress {
		go c.progressReporter(ctx)
	}

	// Wait for workers to complete
	wg.Wait()
	cancel()
	admitter.close()
	admitter.wait()
	scheduler.wait()
	if err := scheduler.failure(); err != nil {
		return err
	}
	if err := admitter.failure(); err != nil {
		return err
	}
	if seedErr != nil {
		return seedErr
	}
	return parentCtx.Err()
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

func (c *Crawler) prepareURL(rawURL string, parent *URLItem) (URLItem, bool, bool) {
	normalizedURL, err := c.normalizeURL(rawURL)
	if err != nil {
		c.logger.Warn("invalid url",
			slog.String("url", rawURL),
			slog.String("error", err.Error()))
		return URLItem{}, false, false
	}
	value := strings.TrimSuffix(normalizedURL.String(), "/")
	// Atomically check-and-mark as seen so concurrent workers cannot enqueue
	// the same URL twice.
	if _, seen := c.processedURLs.LoadOrStore(value, true); seen {
		return URLItem{}, false, false
	}
	// Reserve a slot against the max URLs limit before queueing.
	if c.maxURLs > 0 && atomic.AddInt64(&c.enqueuedURLs, 1) > int64(c.maxURLs) {
		atomic.AddInt64(&c.enqueuedURLs, -1)
		c.processedURLs.Delete(value)
		return URLItem{}, false, true
	}

	item := URLItem{
		URL:          value,
		DiscoveredAt: time.Now(),
	}
	if parent != nil {
		item.Depth = parent.Depth + 1
		item.Referrer = parent.URL
	}
	return item, true, false
}

// releaseURL undoes the bookkeeping performed before a URL is queued, used
// when queueing fails.
func (c *Crawler) releaseURL(value string) {
	c.processedURLs.Delete(value)
	if c.maxURLs > 0 {
		atomic.AddInt64(&c.enqueuedURLs, -1)
	}
}

func (c *Crawler) worker(
	ctx context.Context,
	cancel context.CancelFunc,
	scheduler *hostScheduler,
	admitter *frontierAdmitter,
	wg *sync.WaitGroup,
	callback Callback,
) {
	defer wg.Done()
	for {
		scheduled, ok := scheduler.next(ctx)
		if !ok {
			return
		}
		// Caller-provided frontiers may be preloaded rather than populated by
		// enqueue, so mark every dispatched item as seen before processing it.
		c.processedURLs.Store(scheduled.item.URL, true)
		atomic.AddInt64(&c.inFlightURLs, 1)
		outcome := c.processURL(ctx, scheduled, callback)
		if outcome.attemptErrors != nil {
			scheduled.attemptErrors = outcome.attemptErrors
		}
		scheduler.complete(ctx, completedItem{
			scheduled:   scheduled,
			crawlDelay:  outcome.crawlDelay,
			observation: outcome.observation,
			retry:       outcome.retry,
			retryDelay:  outcome.retryDelay,
		})
		atomic.AddInt64(&c.inFlightURLs, -1)
		if err := admitter.submit(ctx, outcome.discovered, &scheduled.item); err != nil && ctx.Err() == nil {
			c.logger.Warn("failed to admit discovered urls",
				slog.String("url", scheduled.item.URL),
				slog.String("error", err.Error()))
		}
		if outcome.retry {
			continue
		}
		c.stats.IncrementProcessed()

		// Discovered links are handed to the admission goroutine before this
		// decrement, so zero means no worker or queued item can produce more work.
		admitter.completeURL()
		if ctx.Err() != nil {
			return
		}
	}
}

type processOutcome struct {
	crawlDelay    time.Duration
	observation   requestObservation
	retry         bool
	retryDelay    time.Duration
	attemptErrors []error
	discovered    []string
}

// processURL fetches, parses, and reports one attempt for a URL. Retryable
// attempts are handed back to the scheduler without invoking the callback;
// the URL remains pending until its final attempt.
func (c *Crawler) processURL(ctx context.Context, scheduled scheduledItem, callback Callback) processOutcome {
	item := scheduled.item
	rawURL := item.URL

	// Parse the url to get its domain
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		c.logger.Warn("invalid url",
			slog.String("url", rawURL),
			slog.String("error", err.Error()))
		return processOutcome{}
	}
	domain := parsedURL.Hostname()

	var response *fetch.Response
	var cachedEntry *cache.Entry
	var observation requestObservation
	if c.responseCache != nil {
		entry, cacheErr := c.responseCache.GetEntry(ctx, cache.ResponseKey(rawURL))
		if cacheErr == nil && entry != nil && entry.SchemaVersion == cache.ResponseSchemaVersion {
			cachedEntry = entry
			if c.cacheTTL <= 0 || !entry.FetchedAt.IsZero() && time.Since(entry.FetchedAt) <= c.cacheTTL {
				c.logger.Debug("typed cache hit", slog.String("url", rawURL))
				response = responseFromCacheEntry(entry, rawURL)
			}
		} else if cacheErr != nil && !cache.IsNotFound(cacheErr) {
			c.logger.Warn("failed to read cached response",
				slog.String("url", rawURL),
				slog.String("error", cacheErr.Error()))
		}
	}
	// A typed stale entry must not fall through to its legacy HTML shadow,
	// which would bypass TTL and revalidation. Legacy-only entries retain the
	// original cache behavior and remain fresh indefinitely.
	if response == nil && cachedEntry == nil && c.cache != nil {
		if cachedHTML, err := c.cache.Get(ctx, rawURL); err == nil {
			c.logger.Debug("cache hit", slog.String("url", rawURL))
			response = &fetch.Response{
				URL:        rawURL,
				StatusCode: http.StatusOK,
				HTML:       string(cachedHTML),
			}
			// The cache stores only HTML, not the links discovered at fetch
			// time, so re-extract them to keep link following working on
			// cache hits.
			if doc, err := htmlparse.Parse(response.HTML); err == nil {
				response.Links = doc.Links()
			}
		}
	}

	// Fetch if there was not a cache hit
	if response == nil {
		// A cache hit does not need a fetcher at all. This permits replaying a
		// cached crawl even when its original fetcher is unavailable.
		fetcher, exists := c.getFetcher(domain)
		if !exists {
			c.logger.Error("no fetcher configured",
				slog.String("url", rawURL),
				slog.String("domain", domain))
			callback(ctx, c.resultWithError(item, parsedURL, errors.New("no fetcher configured for domain")))
			c.stats.IncrementFailed()
			return processOutcome{}
		}

		// Check robots.txt if enabled. Cache hits skip this check (and the
		// robots.txt fetch it may trigger) since no network request is made.
		if c.respectRobotsTxt && !c.isAllowedByRobots(ctx, parsedURL) {
			c.logger.Debug("blocked by robots.txt",
				slog.String("url", rawURL))
			callback(ctx, c.resultWithError(item, parsedURL, errors.New("blocked by robots.txt")))
			c.stats.IncrementFailed()
			return processOutcome{crawlDelay: c.robotsCrawlDelayFor(parsedURL)}
		}

		req := &fetch.Request{
			URL:             rawURL,
			Prettify:        false,
			OnlyMainContent: false,
			Formats:         []string{"html", "links"},
		}
		if cachedEntry != nil && c.revalidate {
			req.Headers = make(map[string]string, 2)
			etag := cachedEntry.ETag
			if etag == "" {
				etag = headerValue(cachedEntry.Headers, "ETag")
			}
			if etag != "" {
				req.Headers["If-None-Match"] = etag
			}
			lastModified := cachedEntry.LastModified
			if lastModified == "" {
				lastModified = headerValue(cachedEntry.Headers, "Last-Modified")
			}
			if lastModified != "" {
				req.Headers["If-Modified-Since"] = lastModified
			}
		}
		c.logger.Debug("fetching", slog.String("url", rawURL))
		started := time.Now()
		c.hostStats.requestStarted(scheduled.host, started)
		response, err = fetcher.Fetch(ctx, req)
		latency := time.Since(started)
		c.hostStats.requestFinished(scheduled.host, latency, response, err)
		observation = requestObservation{
			latency: latency,
			failed:  err != nil,
		}
		if response != nil {
			observation.statusCode = response.StatusCode
		}

		if err != nil {
			attemptErrors := append(append([]error(nil), scheduled.attemptErrors...), err)
			if c.retryOptions != nil && ctx.Err() == nil && scheduled.attempt < c.retryMaxAttempts() {
				delay := c.retryBackoff(scheduled.attempt)
				c.logger.Warn("retrying fetch",
					slog.String("url", rawURL),
					slog.Int("attempt", scheduled.attempt),
					slog.String("error", err.Error()),
					slog.Duration("delay", delay))
				return processOutcome{
					crawlDelay:    c.robotsCrawlDelayFor(parsedURL),
					observation:   observation,
					retry:         true,
					retryDelay:    delay,
					attemptErrors: attemptErrors,
				}
			}
			if c.retryOptions != nil {
				err = &retry.Error{
					Last:     err,
					Attempts: len(attemptErrors),
					Errors:   attemptErrors,
				}
			}
			callback(ctx, c.resultWithError(item, parsedURL, err))
			c.stats.IncrementFailed()
			return processOutcome{
				crawlDelay:  c.robotsCrawlDelayFor(parsedURL),
				observation: observation,
			}
		}

		if response.StatusCode == http.StatusNotModified && cachedEntry != nil {
			cachedEntry.FetchedAt = time.Now().UTC()
			cachedEntry.Headers = mergeHeaders(cachedEntry.Headers, response.Headers)
			if value := headerValue(response.Headers, "ETag"); value != "" {
				cachedEntry.ETag = value
			}
			if value := headerValue(response.Headers, "Last-Modified"); value != "" {
				cachedEntry.LastModified = value
			}
			c.storeResponseCacheEntry(ctx, rawURL, cachedEntry)
			response = responseFromCacheEntry(cachedEntry, rawURL)
		} else {
			if response.StatusCode == http.StatusTooManyRequests || response.StatusCode == http.StatusServiceUnavailable {
				if retryAfter, ok := parseRetryAfter(headerValue(response.Headers, "Retry-After"), time.Now()); ok {
					observation.retryAfter = retryAfter
					observation.hasRetryAfter = true
				}
				if c.adaptive && scheduled.attempt < c.retryMaxAttempts() {
					c.logger.Warn("host asked crawler to back off",
						slog.String("url", rawURL),
						slog.Int("status", response.StatusCode),
						slog.Int("attempt", scheduled.attempt))
					return processOutcome{
						crawlDelay:  c.robotsCrawlDelayFor(parsedURL),
						observation: observation,
						retry:       true,
					}
				}
				fetchErr := &fetch.Error{StatusCode: response.StatusCode, URL: rawURL}
				result := c.resultWithError(item, parsedURL, fetchErr)
				result.Response = response
				callback(ctx, result)
				c.stats.IncrementFailed()
				return processOutcome{
					crawlDelay:  c.robotsCrawlDelayFor(parsedURL),
					observation: observation,
				}
			}
			c.storeFetchedResponse(ctx, rawURL, response)
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
		URL:          parsedURL,
		Depth:        item.Depth,
		Referrer:     item.Referrer,
		DiscoveredAt: item.DiscoveredAt,
		Parsed:       parsed,
		Links:        discoveredLinks,
		Response:     response,
		Error:        parseErr,
	})
	if parseErr != nil {
		c.stats.IncrementFailed()
	} else {
		c.stats.IncrementSucceeded()
	}

	filteredURLs := c.filterLinks(parsedURL, discoveredLinks)
	return processOutcome{
		crawlDelay:  c.robotsCrawlDelayFor(parsedURL),
		observation: observation,
		discovered:  filteredURLs,
	}
}

func (c *Crawler) resultWithError(item URLItem, parsedURL *url.URL, err error) *Result {
	return &Result{
		URL:          parsedURL,
		Depth:        item.Depth,
		Referrer:     item.Referrer,
		DiscoveredAt: item.DiscoveredAt,
		Error:        err,
	}
}

func responseFromCacheEntry(entry *cache.Entry, fallbackURL string) *fetch.Response {
	responseURL := entry.URL
	if responseURL == "" {
		responseURL = fallbackURL
	}
	statusCode := entry.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return &fetch.Response{
		URL:        responseURL,
		StatusCode: statusCode,
		Headers:    cloneHeaders(entry.Headers),
		HTML:       string(entry.Body),
		Links:      append([]fetch.Link(nil), entry.Links...),
		Timestamp:  entry.FetchedAt,
	}
}

func (c *Crawler) storeFetchedResponse(ctx context.Context, rawURL string, response *fetch.Response) {
	if c.cache == nil || response == nil || !isCacheableStatus(response.StatusCode) {
		return
	}
	entry := &cache.Entry{
		URL:           response.URL,
		StatusCode:    response.StatusCode,
		Headers:       cloneHeaders(response.Headers),
		Body:          []byte(response.HTML),
		Links:         append([]fetch.Link(nil), response.Links...),
		ETag:          headerValue(response.Headers, "ETag"),
		LastModified:  headerValue(response.Headers, "Last-Modified"),
		FetchedAt:     time.Now().UTC(),
		SchemaVersion: cache.ResponseSchemaVersion,
	}
	c.storeResponseCacheEntry(ctx, rawURL, entry)
}

func isCacheableStatus(statusCode int) bool {
	return statusCode == 0 || statusCode == http.StatusOK
}

func (c *Crawler) storeResponseCacheEntry(ctx context.Context, rawURL string, entry *cache.Entry) {
	// Keep writing the original URL/HTML pair as well. Existing callers can
	// continue reading entries through Cache.Get, and legacy Cache
	// implementations remain fully supported. Write it first because coherent
	// dual-interface caches invalidate a typed entry on a legacy update.
	if c.cache != nil && len(entry.Body) > 0 {
		if err := c.cache.Set(ctx, rawURL, entry.Body); err != nil {
			c.logger.Warn("failed to cache html",
				slog.String("url", rawURL),
				slog.String("error", err.Error()))
		}
	}
	if c.responseCache != nil {
		if err := c.responseCache.SetEntry(ctx, cache.ResponseKey(rawURL), entry); err != nil {
			c.logger.Warn("failed to cache response",
				slog.String("url", rawURL),
				slog.String("error", err.Error()))
		}
	}
}

func cloneHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	copy := make(map[string]string, len(headers))
	for key, value := range headers {
		copy[key] = value
	}
	return copy
}

func mergeHeaders(base, update map[string]string) map[string]string {
	merged := cloneHeaders(base)
	if merged == nil {
		merged = make(map[string]string, len(update))
	}
	for key, value := range update {
		for existing := range merged {
			if strings.EqualFold(existing, key) {
				delete(merged, existing)
			}
		}
		merged[key] = value
	}
	return merged
}

func headerValue(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}
	when, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	delay := when.Sub(now)
	if delay < 0 {
		delay = 0
	}
	return delay, true
}

func (c *Crawler) retryMaxAttempts() int {
	if c.retryOptions != nil && c.retryOptions.MaxAttempts > 0 {
		return c.retryOptions.MaxAttempts
	}
	return 3
}

func (c *Crawler) retryBackoff(attempt int) time.Duration {
	initialBackoff := c.retryOptions.InitialBackoff
	if initialBackoff <= 0 {
		initialBackoff = time.Second
	}
	maxBackoff := c.retryOptions.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}
	config := retry.Config{
		InitialBackoff:    initialBackoff,
		MaxBackoff:        maxBackoff,
		BackoffMultiplier: 2,
		Jitter:            0.1,
	}
	return retry.ExponentialBackoff(attempt, &config)
}

// robotsCrawlDelayFor returns the cached robots.txt Crawl-delay for a host.
// The host scheduler combines it with HostPolicy.MinDelay.
func (c *Crawler) robotsCrawlDelayFor(parsedURL *url.URL) time.Duration {
	if !c.respectRobotsTxt {
		return 0
	}
	return c.cachedRobotsCrawlDelay(parsedURL)
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

// HostStats returns a stable, host-sorted snapshot of network activity for
// the current or most recent Crawl call. The maps in the returned values are
// copies and may be safely modified by the caller.
func (c *Crawler) HostStats() []HostStats {
	return c.hostStats.snapshot()
}

// Pending returns the number of queued URLs that have not started processing.
// It is safe to call while a crawl is running and is intended for progress
// reporting and observability integrations.
func (c *Crawler) Pending() int {
	pending := atomic.LoadInt64(&c.pendingURLs) - atomic.LoadInt64(&c.inFlightURLs)
	if pending < 0 {
		return 0
	}
	return int(pending)
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
