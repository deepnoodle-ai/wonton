package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/crawler/cache"
	"github.com/deepnoodle-ai/wonton/fetch"
)

func TestMemoryFrontierPriorityAndClose(t *testing.T) {
	frontier := NewMemoryFrontier(0)
	ctx := context.Background()
	assert.NoError(t, frontier.Push(ctx,
		URLItem{URL: "https://example.com/deep", Depth: 2},
		URLItem{URL: "https://example.com/scored", Depth: 5, Score: 10},
		URLItem{URL: "https://example.com/shallow", Depth: 1},
	))

	for _, expected := range []string{
		"https://example.com/scored",
		"https://example.com/shallow",
		"https://example.com/deep",
	} {
		item, ok, err := frontier.Next(ctx)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, expected, item.URL)
	}

	assert.NoError(t, frontier.Close())
	_, ok, err := frontier.Next(ctx)
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.ErrorIs(t, frontier.Push(ctx, URLItem{URL: "https://example.com/late"}), ErrFrontierClosed)
}

func TestMemoryFrontierBackpressure(t *testing.T) {
	frontier := NewMemoryFrontier(1)
	ctx := context.Background()
	assert.NoError(t, frontier.Push(ctx, URLItem{URL: "https://example.com/one"}))

	pushDone := make(chan error, 1)
	go func() {
		pushDone <- frontier.Push(ctx, URLItem{URL: "https://example.com/two"})
	}()

	select {
	case err := <-pushDone:
		t.Fatalf("Push returned before capacity was available: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	_, ok, err := frontier.Next(ctx)
	assert.NoError(t, err)
	assert.True(t, ok)

	select {
	case err := <-pushDone:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Push did not resume after capacity became available")
	}
}

func TestMemoryFrontierCapacityIncludesSchedulerLease(t *testing.T) {
	frontier := NewMemoryFrontier(1)
	assert.NoError(t, frontier.Push(context.Background(), URLItem{URL: "https://example.com/one"}))

	_, ok, err := frontier.nextForScheduling(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = frontier.Push(ctx, URLItem{URL: "https://example.com/two"})
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	frontier.releaseScheduled()
	assert.NoError(t, frontier.Push(context.Background(), URLItem{URL: "https://example.com/two"}))
}

func TestMemoryFrontierPushCancellation(t *testing.T) {
	frontier := NewMemoryFrontier(1)
	assert.NoError(t, frontier.Push(context.Background(), URLItem{URL: "https://example.com/one"}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := frontier.Push(ctx, URLItem{URL: "https://example.com/two"})
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, frontier.Len())
}

func TestCrawlerSmallFrontierDoesNotDropDiscoveredURLs(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	links := make([]fetch.Link, 64)
	for i := range links {
		path := fmt.Sprintf("/page/%02d", i)
		links[i] = fetch.Link{URL: path}
		pageURL := "https://example.com" + path
		mockFetcher.AddResponse(pageURL, &fetch.Response{URL: pageURL, HTML: "<html></html>"})
	}
	mockFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:   "https://example.com",
		HTML:  "<html></html>",
		Links: links,
	})
	mockFetcher.AddResponse("https://example.com/seed-two", &fetch.Response{
		URL:  "https://example.com/seed-two",
		HTML: "<html></html>",
	})

	c, err := New(Options{
		Workers:          1,
		QueueSize:        1,
		DefaultFetcher:   mockFetcher,
		FollowBehavior:   FollowSameDomain,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	visited := make(map[string]bool)
	maxPending := 0
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = c.Crawl(ctx, []string{"https://example.com", "https://example.com/seed-two"}, func(_ context.Context, result *Result) {
		mu.Lock()
		visited[result.URL.String()] = true
		if pending := c.Pending(); pending > maxPending {
			maxPending = pending
		}
		mu.Unlock()
	})
	assert.NoError(t, err)
	assert.Len(t, visited, 66)
	if maxPending > 1 {
		t.Fatalf("QueueSize=1 allowed %d URLs to wait", maxPending)
	}
	assert.Equal(t, 0, c.Pending())
}

type cancellationFetcher struct {
	started chan struct{}
	once    sync.Once
}

func (f *cancellationFetcher) Fetch(ctx context.Context, _ *fetch.Request) (*fetch.Response, error) {
	f.once.Do(func() { close(f.started) })
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCrawlerReportsParentContextCancellation(t *testing.T) {
	fetcher := &cancellationFetcher{started: make(chan struct{})}
	c, err := New(Options{
		DefaultFetcher:   fetcher,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- c.Crawl(ctx, []string{"https://example.com"}, func(context.Context, *Result) {})
	}()
	select {
	case <-fetcher.started:
	case <-time.After(time.Second):
		t.Fatal("fetch did not start")
	}
	cancel()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("crawl did not return after parent cancellation")
	}
}

func TestCrawlerProcessesPreloadedFrontier(t *testing.T) {
	frontier := NewMemoryFrontier(0)
	discoveredAt := time.Now().Add(-time.Minute)
	assert.NoError(t, frontier.Push(context.Background(), URLItem{
		URL:          "https://example.com/preloaded",
		Depth:        3,
		Referrer:     "https://example.com/parent",
		DiscoveredAt: discoveredAt,
	}))

	mockFetcher := fetch.NewMockFetcher()
	mockFetcher.AddResponse("https://example.com/preloaded", &fetch.Response{
		URL:  "https://example.com/preloaded",
		HTML: "<html></html>",
	})
	c, err := New(Options{
		Frontier:         frontier,
		DefaultFetcher:   mockFetcher,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var result *Result
	assert.NoError(t, c.Crawl(context.Background(), nil, func(_ context.Context, got *Result) {
		result = got
	}))
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.Depth)
	assert.Equal(t, "https://example.com/parent", result.Referrer)
	assert.Equal(t, discoveredAt, result.DiscoveredAt)
}

func TestCrawlerReportsDepthReferrerAndDiscoveryTime(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	mockFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:   "https://example.com",
		HTML:  "<html></html>",
		Links: []fetch.Link{{URL: "/a"}, {URL: "/b"}},
	})
	mockFetcher.AddResponse("https://example.com/a", &fetch.Response{
		URL:   "https://example.com/a",
		HTML:  "<html></html>",
		Links: []fetch.Link{{URL: "/deep"}},
	})
	mockFetcher.AddResponse("https://example.com/b", &fetch.Response{URL: "https://example.com/b", HTML: "<html></html>"})
	mockFetcher.AddResponse("https://example.com/deep", &fetch.Response{URL: "https://example.com/deep", HTML: "<html></html>"})

	c, err := New(Options{
		Workers:          2,
		DefaultFetcher:   mockFetcher,
		FollowBehavior:   FollowSameDomain,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	results := make(map[string]*Result)
	err = c.Crawl(context.Background(), []string{"https://example.com"}, func(_ context.Context, result *Result) {
		mu.Lock()
		copy := *result
		results[result.URL.String()] = &copy
		mu.Unlock()
	})
	assert.NoError(t, err)

	assert.Equal(t, 0, results["https://example.com"].Depth)
	assert.Equal(t, "", results["https://example.com"].Referrer)
	assert.False(t, results["https://example.com"].DiscoveredAt.IsZero())
	assert.Equal(t, 1, results["https://example.com/a"].Depth)
	assert.Equal(t, "https://example.com", results["https://example.com/a"].Referrer)
	assert.Equal(t, 2, results["https://example.com/deep"].Depth)
	assert.Equal(t, "https://example.com/a", results["https://example.com/deep"].Referrer)
}

func TestCrawlerCacheHitDoesNotRequireFetcher(t *testing.T) {
	htmlCache := cache.NewInMemoryCache()
	assert.NoError(t, htmlCache.Set(context.Background(), "https://example.com", []byte("<html>cached</html>")))

	c, err := New(Options{
		Cache:            htmlCache,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var result *Result
	err = c.Crawl(context.Background(), []string{"https://example.com"}, func(_ context.Context, got *Result) {
		result = got
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NoError(t, result.Error)
	assert.Equal(t, "<html>cached</html>", result.Response.HTML)
}

type concurrencyTrackingFetcher struct {
	mu      sync.Mutex
	current map[string]int
	maximum map[string]int
	delay   time.Duration
}

func (f *concurrencyTrackingFetcher) Fetch(ctx context.Context, request *fetch.Request) (*fetch.Response, error) {
	parsed, err := url.Parse(request.URL)
	if err != nil {
		return nil, err
	}
	host := parsed.Host
	f.mu.Lock()
	if f.current == nil {
		f.current = make(map[string]int)
	}
	f.current[host]++
	if f.maximum == nil {
		f.maximum = make(map[string]int)
	}
	if f.current[host] > f.maximum[host] {
		f.maximum[host] = f.current[host]
	}
	f.mu.Unlock()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-time.After(f.delay):
	}

	f.mu.Lock()
	f.current[host]--
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &fetch.Response{URL: request.URL, StatusCode: 200, HTML: "<html></html>"}, nil
}

func TestCrawlerEnforcesPerHostConcurrency(t *testing.T) {
	tracker := &concurrencyTrackingFetcher{delay: 20 * time.Millisecond}
	urls := make([]string, 12)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.com/page/%d", i)
	}

	c, err := New(Options{
		Workers:          8,
		DefaultFetcher:   tracker,
		FollowBehavior:   FollowNone,
		HostPolicy:       HostPolicy{MaxConcurrent: 2},
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)
	assert.NoError(t, c.Crawl(context.Background(), urls, func(context.Context, *Result) {}))

	tracker.mu.Lock()
	maximum := tracker.maximum["example.com"]
	tracker.mu.Unlock()
	assert.Equal(t, 2, maximum)
}

type hostGateFetcher struct {
	slowRelease chan struct{}
	fastStarted chan struct{}
	fastOnce    sync.Once
}

func (f *hostGateFetcher) Fetch(ctx context.Context, request *fetch.Request) (*fetch.Response, error) {
	parsed, err := url.Parse(request.URL)
	if err != nil {
		return nil, err
	}
	if parsed.Host == "slow.example" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-f.slowRelease:
		}
	} else {
		f.fastOnce.Do(func() { close(f.fastStarted) })
	}
	return &fetch.Response{URL: request.URL, StatusCode: 200, HTML: "<html></html>"}, nil
}

func TestCrawlerSlowHostDoesNotBlockAnotherHost(t *testing.T) {
	fetcher := &hostGateFetcher{
		slowRelease: make(chan struct{}),
		fastStarted: make(chan struct{}),
	}
	c, err := New(Options{
		Workers:          2,
		DefaultFetcher:   fetcher,
		FollowBehavior:   FollowNone,
		HostPolicy:       HostPolicy{MaxConcurrent: 1},
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	crawlDone := make(chan error, 1)
	go func() {
		crawlDone <- c.Crawl(context.Background(), []string{
			"https://slow.example/one",
			"https://slow.example/two",
			"https://fast.example/one",
		}, func(context.Context, *Result) {})
	}()

	select {
	case <-fetcher.fastStarted:
	case <-time.After(time.Second):
		close(fetcher.slowRelease)
		<-crawlDone
		t.Fatal("fast host did not start while the slow host was blocked")
	}
	close(fetcher.slowRelease)
	assert.NoError(t, <-crawlDone)
}

type startTimeFetcher struct {
	mu     sync.Mutex
	starts []time.Time
}

func (f *startTimeFetcher) Fetch(_ context.Context, request *fetch.Request) (*fetch.Response, error) {
	f.mu.Lock()
	f.starts = append(f.starts, time.Now())
	f.mu.Unlock()
	return &fetch.Response{URL: request.URL, StatusCode: 200, HTML: "<html></html>"}, nil
}

func TestCrawlerRequestDelayIsPerHost(t *testing.T) {
	fetcher := &startTimeFetcher{}
	c, err := New(Options{
		Workers:          2,
		RequestDelay:     50 * time.Millisecond,
		DefaultFetcher:   fetcher,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)
	assert.NoError(t, c.Crawl(context.Background(), []string{
		"https://example.com/one",
		"https://example.com/two",
	}, func(context.Context, *Result) {}))

	fetcher.mu.Lock()
	starts := append([]time.Time(nil), fetcher.starts...)
	fetcher.mu.Unlock()
	assert.Len(t, starts, 2)
	if spacing := starts[1].Sub(starts[0]); spacing < 45*time.Millisecond {
		t.Fatalf("same-host requests were only %s apart", spacing)
	}
}

type robotsDelayFetcher struct {
	mu         sync.Mutex
	pageStarts []time.Time
}

func (f *robotsDelayFetcher) Fetch(_ context.Context, request *fetch.Request) (*fetch.Response, error) {
	if request.URL == "https://example.com/robots.txt" {
		return &fetch.Response{
			URL:  request.URL,
			HTML: "User-agent: *\nCrawl-delay: 0.05",
		}, nil
	}
	f.mu.Lock()
	f.pageStarts = append(f.pageStarts, time.Now())
	f.mu.Unlock()
	return &fetch.Response{URL: request.URL, StatusCode: 200, HTML: "<html></html>"}, nil
}

func TestCrawlerRobotsDelayIsAppliedPerHost(t *testing.T) {
	fetcher := &robotsDelayFetcher{}
	c, err := New(Options{
		Workers:        2,
		DefaultFetcher: fetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)
	assert.NoError(t, c.Crawl(context.Background(), []string{
		"https://example.com/one",
		"https://example.com/two",
	}, func(context.Context, *Result) {}))

	fetcher.mu.Lock()
	starts := append([]time.Time(nil), fetcher.pageStarts...)
	fetcher.mu.Unlock()
	assert.Len(t, starts, 2)
	if spacing := starts[1].Sub(starts[0]); spacing < 45*time.Millisecond {
		t.Fatalf("robots.txt Crawl-delay produced only %s spacing", spacing)
	}
}

func TestParseRobotsTxtIgnoresEmptyUserAgent(t *testing.T) {
	data := parseRobotsTxt(`User-agent:
Disallow: /everything

User-agent: *
Disallow: /public`, "WontonBot")
	assert.Equal(t, []string{"/public"}, data.disallowRules)
}

func TestMemoryFrontierNextCancellation(t *testing.T) {
	frontier := NewMemoryFrontier(0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := frontier.Next(ctx)
	assert.True(t, errors.Is(err, context.Canceled))
}

type failingFrontier struct {
	err error
}

func (f *failingFrontier) Push(context.Context, ...URLItem) error { return nil }
func (f *failingFrontier) Next(context.Context) (URLItem, bool, error) {
	return URLItem{}, false, f.err
}
func (f *failingFrontier) Len() int     { return 0 }
func (f *failingFrontier) Close() error { return nil }

func TestCrawlerReturnsFrontierFailure(t *testing.T) {
	expected := errors.New("frontier unavailable")
	c, err := New(Options{
		Frontier:         &failingFrontier{err: expected},
		DefaultFetcher:   fetch.NewMockFetcher(),
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)
	err = c.Crawl(context.Background(), []string{"https://example.com"}, func(context.Context, *Result) {})
	assert.ErrorIs(t, err, expected)
}

type closeAfterPushFrontier struct {
	*MemoryFrontier
}

func (f *closeAfterPushFrontier) Push(ctx context.Context, items ...URLItem) error {
	if err := f.MemoryFrontier.Push(ctx, items...); err != nil {
		return err
	}
	return f.Close()
}

func TestCrawlerDrainsClosedFrontier(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	mockFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:  "https://example.com",
		HTML: "<html></html>",
	})
	frontier := &closeAfterPushFrontier{MemoryFrontier: NewMemoryFrontier(1)}
	c, err := New(Options{
		Frontier:         frontier,
		DefaultFetcher:   mockFetcher,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	assert.NoError(t, c.Crawl(ctx, []string{"https://example.com"}, func(context.Context, *Result) {}))
	assert.ErrorIs(t, frontier.Push(ctx, URLItem{URL: "https://example.com/late"}), ErrFrontierClosed)
}
