package crawler

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/crawler/cache"
	"github.com/deepnoodle-ai/wonton/fetch"
)

// TestCrawler_Reuse verifies that a Crawler can run multiple crawls. Earlier
// versions closed the URL queue after the first crawl, so a second call to
// Crawl panicked.
func TestCrawler_Reuse(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	mockFetcher.AddResponse("https://example.com/a", &fetch.Response{
		URL:   "https://example.com/a",
		HTML:  "<html><body>A</body></html>",
		Links: []fetch.Link{},
	})
	mockFetcher.AddResponse("https://example.com/b", &fetch.Response{
		URL:   "https://example.com/b",
		HTML:  "<html><body>B</body></html>",
		Links: []fetch.Link{},
	})

	c, err := New(Options{
		Workers:        1,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	var crawled []string
	callback := func(ctx context.Context, result *Result) {
		mu.Lock()
		defer mu.Unlock()
		crawled = append(crawled, result.URL.String())
	}

	ctx := context.Background()
	assert.NoError(t, c.Crawl(ctx, []string{"https://example.com/a"}, callback))
	assert.NoError(t, c.Crawl(ctx, []string{"https://example.com/b"}, callback))

	assert.Equal(t, []string{"https://example.com/a", "https://example.com/b"}, crawled)

	// URLs from earlier crawls are remembered and not crawled again
	assert.NoError(t, c.Crawl(ctx, []string{"https://example.com/a"}, callback))
	assert.Len(t, crawled, 2)
}

// TestCrawler_MaxURLsWithDiscoveredLinks verifies that MaxURLs strictly bounds
// the number of URLs processed, including links discovered during the crawl.
// Earlier versions enforced the limit against the processed count at enqueue
// time, which admitted far more URLs than the limit while pages were in flight.
func TestCrawler_MaxURLsWithDiscoveredLinks(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	links := []fetch.Link{
		{URL: "/p1"}, {URL: "/p2"}, {URL: "/p3"}, {URL: "/p4"}, {URL: "/p5"},
	}
	mockFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:   "https://example.com",
		HTML:  "<html><body>root</body></html>",
		Links: links,
	})
	for _, l := range links {
		u := "https://example.com" + l.URL
		mockFetcher.AddResponse(u, &fetch.Response{
			URL:   u,
			HTML:  "<html><body>page</body></html>",
			Links: links, // every page links to all the others
		})
	}

	c, err := New(Options{
		MaxURLs:          3,
		Workers:          2,
		DefaultFetcher:   mockFetcher,
		FollowBehavior:   FollowSameDomain,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	processed := 0
	err = c.Crawl(context.Background(), []string{"https://example.com"}, func(ctx context.Context, result *Result) {
		mu.Lock()
		defer mu.Unlock()
		processed++
	})
	assert.NoError(t, err)
	assert.LessOrEqual(t, processed, 3)
	assert.LessOrEqual(t, c.GetStats().GetProcessed(), int64(3))
}

// TestCrawler_CacheHitFollowsLinks verifies that links are still discovered
// and followed when a page is served from the cache. Earlier versions lost
// all links on cache hits because only the HTML is cached.
func TestCrawler_CacheHitFollowsLinks(t *testing.T) {
	htmlCache := cache.NewInMemoryCache()
	ctx := context.Background()

	// The root page exists only in the cache and links to /child
	rootHTML := `<html><body><a href="/child">Child</a></body></html>`
	assert.NoError(t, htmlCache.Set(ctx, "https://example.com", []byte(rootHTML)))

	mockFetcher := fetch.NewMockFetcher()
	mockFetcher.AddResponse("https://example.com/child", &fetch.Response{
		URL:   "https://example.com/child",
		HTML:  "<html><body>Child</body></html>",
		Links: []fetch.Link{},
	})

	c, err := New(Options{
		Workers:          1,
		DefaultFetcher:   mockFetcher,
		Cache:            htmlCache,
		FollowBehavior:   FollowSameDomain,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	var crawled []string
	err = c.Crawl(ctx, []string{"https://example.com"}, func(ctx context.Context, result *Result) {
		mu.Lock()
		defer mu.Unlock()
		crawled = append(crawled, result.URL.String())
	})
	assert.NoError(t, err)
	assert.Contains(t, crawled, "https://example.com")
	assert.Contains(t, crawled, "https://example.com/child")
}

// TestCrawler_KnownURLsSkipped verifies that URLs listed in KnownURLs are not
// crawled again. Earlier versions stored KnownURLs but never consulted them.
func TestCrawler_KnownURLsSkipped(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	mockFetcher.AddResponse("https://example.com/new", &fetch.Response{
		URL:   "https://example.com/new",
		HTML:  "<html><body>New</body></html>",
		Links: []fetch.Link{},
	})

	c, err := New(Options{
		Workers:        1,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
		KnownURLs:      []string{"https://example.com/known"},
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	var crawled []string
	err = c.Crawl(context.Background(), []string{
		"https://example.com/known",
		"https://example.com/new",
	}, func(ctx context.Context, result *Result) {
		mu.Lock()
		defer mu.Unlock()
		crawled = append(crawled, result.URL.String())
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/new"}, crawled)
}

// TestRobotsLongestMatchPrecedence verifies that the most specific (longest)
// matching rule wins, per the robots.txt standard. Earlier versions let any
// matching Allow rule override all Disallow rules.
func TestRobotsLongestMatchPrecedence(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	mockFetcher.AddResponse("https://example.com/robots.txt", &fetch.Response{
		URL: "https://example.com/robots.txt",
		HTML: `User-agent: *
Allow: /
Disallow: /private`,
	})
	mockFetcher.AddResponse("https://example.com/public", &fetch.Response{
		URL:   "https://example.com/public",
		HTML:  "<html><body>Public</body></html>",
		Links: []fetch.Link{},
	})
	mockFetcher.AddResponse("https://example.com/private", &fetch.Response{
		URL:   "https://example.com/private",
		HTML:  "<html><body>Private</body></html>",
		Links: []fetch.Link{},
	})

	c, err := New(Options{
		Workers:        1,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	results := map[string]error{}
	err = c.Crawl(context.Background(), []string{
		"https://example.com/public",
		"https://example.com/private",
	}, func(ctx context.Context, result *Result) {
		mu.Lock()
		defer mu.Unlock()
		results[result.URL.String()] = result.Error
	})
	assert.NoError(t, err)

	assert.Nil(t, results["https://example.com/public"])
	assert.NotNil(t, results["https://example.com/private"])
	assert.ErrorContains(t, results["https://example.com/private"], "robots.txt")
}

// TestPathMatches_Anchoring verifies that robots.txt rules are anchored at
// the start of the path, including rules with wildcards and end anchors.
func TestPathMatches_Anchoring(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		rule    string
		matches bool
	}{
		{"wildcard rule anchored at start", "/foo/private", "/private*", false},
		{"wildcard rule matches at start", "/private/data", "/private*", true},
		{"trailing wildcard alone", "/private", "/private*", true},
		{"end anchor with wildcard", "/files/report.pdf", "/files/*.pdf$", true},
		{"end anchor with wildcard no match", "/files/report.pdf.html", "/files/*.pdf$", false},
		{"end anchor finds last occurrence", "/axbyb", "/a*b$", true},
		{"middle wildcard", "/a/b/c", "/a/*/c", true},
		{"rule ending in wildcard with end anchor", "/anything/here", "/any*$", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.matches, pathMatches(tt.path, tt.rule),
				"path=%q rule=%q", tt.path, tt.rule)
		})
	}
}

// TestParseRobotsTxt_CrawlDelay verifies Crawl-delay parsing.
func TestParseRobotsTxt_CrawlDelay(t *testing.T) {
	data := parseRobotsTxt(`User-agent: *
Crawl-delay: 1.5
Disallow: /x`, "*")
	assert.Equal(t, 1500*time.Millisecond, data.crawlDelay)

	// Invalid and negative values are ignored
	data = parseRobotsTxt("User-agent: *\nCrawl-delay: -2", "*")
	assert.Equal(t, time.Duration(0), data.crawlDelay)
	data = parseRobotsTxt("User-agent: *\nCrawl-delay: abc", "*")
	assert.Equal(t, time.Duration(0), data.crawlDelay)
}

// TestCrawler_StopIsSafe verifies Stop can be called at any time, including
// when the crawler is not running.
func TestCrawler_StopIsSafe(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	c, err := New(Options{
		Workers:        1,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)
	c.Stop() // not running: should be a no-op
	c.Stop()
}

// TestCrawler_ReuseAfterStopCrawlsQueuedURLs verifies that URLs queued but
// never crawled before Stop can be crawled by a later Crawl call, and that
// they do not use up MaxURLs.
func TestCrawler_ReuseAfterStopCrawlsQueuedURLs(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	links := []fetch.Link{{URL: "/p1"}, {URL: "/p2"}, {URL: "/p3"}}
	pages := []string{"https://example.com", "https://example.com/p1", "https://example.com/p2", "https://example.com/p3"}
	for i, u := range pages {
		resp := &fetch.Response{URL: u, HTML: "<html><body>page</body></html>"}
		if i == 0 {
			resp.Links = links
		}
		mockFetcher.AddResponse(u, resp)
	}

	c, err := New(Options{
		Workers:        1,
		MaxURLs:        len(pages),
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowSameDomain,
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	crawled := map[string]int{}
	callback := func(ctx context.Context, result *Result) {
		mu.Lock()
		crawled[result.URL.String()]++
		mu.Unlock()
		if result.URL.Path == "/p1" {
			c.Stop()
		}
	}

	ctx := context.Background()
	_ = c.Crawl(ctx, []string{pages[0]}, callback)
	assert.Equal(t, c.Pending(), 0)

	assert.NoError(t, c.Crawl(ctx, pages, callback))
	for _, u := range pages {
		assert.Equal(t, crawled[u], 1, u)
	}
}

// TestCrawler_FailedFirstRequestOpensHostGate verifies that with robots.txt
// respected, a failed first request to a host still lets the rest of that
// host's pages run.
func TestCrawler_FailedFirstRequestOpensHostGate(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	mockFetcher.AddError("https://example.com/robots.txt", &fetch.Error{StatusCode: http.StatusNotFound})
	mockFetcher.AddError("https://example.com/a", &fetch.Error{StatusCode: http.StatusNotFound})
	for _, u := range []string{"https://example.com/b", "https://example.com/c"} {
		mockFetcher.AddResponse(u, &fetch.Response{URL: u, HTML: "<html><body>ok</body></html>"})
	}
	c, err := New(Options{
		Workers:          2,
		DefaultFetcher:   mockFetcher,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(true),
		HostPolicy:       HostPolicy{MaxConcurrent: 2},
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	var ok, failed int
	err = c.Crawl(context.Background(), []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}, func(_ context.Context, r *Result) {
		mu.Lock()
		defer mu.Unlock()
		if r.Error != nil {
			failed++
		} else {
			ok++
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, failed, 1)
	assert.Equal(t, ok, 2)
}

// plainFrontier hides the MemoryFrontier type, so the crawler treats it as a
// caller's own Frontier implementation.
type plainFrontier struct{ Frontier }

// TestCrawler_StopKeepsCallerFrontierURLs verifies that URLs the scheduler
// took from a caller's frontier, but never dispatched, go back on it when
// the crawl stops.
func TestCrawler_StopKeepsCallerFrontierURLs(t *testing.T) {
	for name, frontier := range map[string]Frontier{
		"memory": NewMemoryFrontier(0),
		"plain":  plainFrontier{NewMemoryFrontier(0)},
	} {
		t.Run(name, func(t *testing.T) {
			mockFetcher := fetch.NewMockFetcher()
			mockFetcher.AddResponse("https://example.com", &fetch.Response{
				URL:   "https://example.com",
				HTML:  "<html><body>root</body></html>",
				Links: []fetch.Link{{URL: "/p1"}, {URL: "/p2"}, {URL: "/p3"}},
			})
			c, err := New(Options{
				Workers:        1,
				DefaultFetcher: mockFetcher,
				FollowBehavior: FollowSameDomain,
				Frontier:       frontier,
				// Hold the linked pages in the scheduler after the root.
				HostPolicy: HostPolicy{MinDelay: time.Hour},
			})
			assert.NoError(t, err)

			go func() {
				deadline := time.Now().Add(5 * time.Second)
				for c.Pending() < 3 && time.Now().Before(deadline) {
					time.Sleep(time.Millisecond)
				}
				c.Stop()
			}()
			_ = c.Crawl(context.Background(), []string{"https://example.com"}, func(context.Context, *Result) {})
			assert.Equal(t, frontier.Len(), 3)
			assert.Equal(t, c.Pending(), 3)
		})
	}
}

// flakyOnceFetcher fails its first request with a retryable error and
// succeeds after that.
type flakyOnceFetcher struct{ calls int32 }

func (f *flakyOnceFetcher) Fetch(_ context.Context, req *fetch.Request) (*fetch.Response, error) {
	if atomic.AddInt32(&f.calls, 1) == 1 {
		return nil, errors.New("flaky")
	}
	return &fetch.Response{URL: req.URL, HTML: "<html><body>ok</body></html>"}, nil
}

// TestCrawler_StopKeepsWaitingRetry verifies that a retry waiting for its
// backoff when the crawl stops goes back on a caller's frontier, and that a
// later Crawl crawls it.
func TestCrawler_StopKeepsWaitingRetry(t *testing.T) {
	fetcher := &flakyOnceFetcher{}
	frontier := NewMemoryFrontier(0)
	c, err := New(Options{
		Workers:          1,
		DefaultFetcher:   fetcher,
		FollowBehavior:   FollowNone,
		Frontier:         frontier,
		RespectRobotsTxt: BoolPtr(false),
		RetryOptions:     &RetryOptions{MaxAttempts: 3, InitialBackoff: time.Hour, MaxBackoff: time.Hour},
	})
	assert.NoError(t, err)

	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for atomic.LoadInt32(&fetcher.calls) < 1 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		time.Sleep(20 * time.Millisecond) // let the retry reach the scheduler
		c.Stop()
	}()
	_ = c.Crawl(context.Background(), []string{"https://example.com/x"}, func(context.Context, *Result) {})
	assert.Equal(t, frontier.Len(), 1)

	var crawled int
	assert.NoError(t, c.Crawl(context.Background(), nil, func(_ context.Context, r *Result) {
		if r.Error == nil {
			crawled++
		}
	}))
	assert.Equal(t, crawled, 1)
}

// failOncePushFrontier fails its first Push.
type failOncePushFrontier struct {
	Frontier
	failed bool
}

func (f *failOncePushFrontier) Push(ctx context.Context, items ...URLItem) error {
	if !f.failed {
		f.failed = true
		return errors.New("push failed")
	}
	return f.Frontier.Push(ctx, items...)
}

func TestCrawler_RequeueContinuesAfterPushError(t *testing.T) {
	c, err := New(Options{DefaultFetcher: fetch.NewMockFetcher()})
	assert.NoError(t, err)
	frontier := &failOncePushFrontier{Frontier: NewMemoryFrontier(0)}
	c.requeueUnsent(frontier, []scheduledItem{
		{item: URLItem{URL: "https://example.com/a"}, attempt: 1},
		{item: URLItem{URL: "https://example.com/b"}, attempt: 1},
		{item: URLItem{URL: "https://example.com/c"}, attempt: 1},
	})
	assert.Equal(t, frontier.Len(), 2)
}
