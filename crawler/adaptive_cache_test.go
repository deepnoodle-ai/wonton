package crawler

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/crawler/cache"
	"github.com/deepnoodle-ai/wonton/fetch"
)

type scriptedFetcher struct {
	mu        sync.Mutex
	responses []*fetch.Response
	requests  []*fetch.Request
	starts    []time.Time
}

func (f *scriptedFetcher) Fetch(_ context.Context, request *fetch.Request) (*fetch.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copy := *request
	copy.Headers = cloneHeaders(request.Headers)
	f.requests = append(f.requests, &copy)
	f.starts = append(f.starts, time.Now())
	index := len(f.requests) - 1
	if index >= len(f.responses) {
		index = len(f.responses) - 1
	}
	if index < 0 {
		return nil, errors.New("no scripted response")
	}
	return f.responses[index], nil
}

func TestCrawlerAdaptiveRetryAfterAndHostStats(t *testing.T) {
	fetcher := &scriptedFetcher{responses: []*fetch.Response{
		{
			URL:        "https://example.com/page",
			StatusCode: http.StatusTooManyRequests,
			Headers:    map[string]string{"Retry-After": "1"},
		},
		{
			URL:        "https://example.com/page",
			StatusCode: http.StatusOK,
			HTML:       "<html>healthy</html>",
		},
	}}
	c, err := New(Options{
		Workers:          2,
		DefaultFetcher:   fetcher,
		FollowBehavior:   FollowNone,
		Adaptive:         true,
		HostPolicy:       HostPolicy{MaxConcurrent: 1, MaxDelay: 40 * time.Millisecond},
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var results []*Result
	err = c.Crawl(context.Background(), []string{"https://example.com/page"}, func(_ context.Context, result *Result) {
		results = append(results, result)
	})
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NoError(t, results[0].Error)

	fetcher.mu.Lock()
	starts := append([]time.Time(nil), fetcher.starts...)
	fetcher.mu.Unlock()
	assert.Len(t, starts, 2)
	if spacing := starts[1].Sub(starts[0]); spacing < 35*time.Millisecond {
		t.Fatalf("Retry-After retry started after only %s", spacing)
	}

	stats := c.HostStats()
	assert.Len(t, stats, 1)
	assert.Equal(t, "example.com", stats[0].Host)
	assert.Equal(t, int64(2), stats[0].Requests)
	assert.Equal(t, int64(1), stats[0].Errors)
	assert.Equal(t, int64(1), stats[0].StatusCodes[http.StatusTooManyRequests])
	assert.Equal(t, int64(1), stats[0].StatusCodes[http.StatusOK])
	assert.Equal(t, int64(len("<html>healthy</html>")), stats[0].Bytes)
	assert.Equal(t, 40*time.Millisecond, stats[0].FinalDelay)
	assert.Equal(t, int64(1), c.GetStats().GetProcessed())
	assert.Equal(t, int64(1), c.GetStats().GetSucceeded())
}

func TestCrawlerAdaptiveRetriesAreBounded(t *testing.T) {
	fetcher := &scriptedFetcher{responses: []*fetch.Response{{
		URL:        "https://example.com/busy",
		StatusCode: http.StatusServiceUnavailable,
		Headers:    map[string]string{"Retry-After": "0"},
	}}}
	c, err := New(Options{
		DefaultFetcher:   fetcher,
		FollowBehavior:   FollowNone,
		Adaptive:         true,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var result *Result
	err = c.Crawl(context.Background(), []string{"https://example.com/busy"}, func(_ context.Context, got *Result) {
		result = got
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Error(t, result.Error)
	assert.Equal(t, http.StatusServiceUnavailable, result.Response.StatusCode)
	assert.Equal(t, int64(1), c.GetStats().GetProcessed())
	assert.Equal(t, int64(1), c.GetStats().GetFailed())

	fetcher.mu.Lock()
	requests := len(fetcher.requests)
	fetcher.mu.Unlock()
	assert.Equal(t, 3, requests)
	stats := c.HostStats()
	assert.Len(t, stats, 1)
	assert.Equal(t, int64(3), stats[0].Requests)
	assert.Equal(t, int64(3), stats[0].Errors)
	assert.Equal(t, int64(3), stats[0].StatusCodes[http.StatusServiceUnavailable])
}

func TestAdaptiveLatencyBackoffAndRecovery(t *testing.T) {
	scheduler := newHostScheduler(
		NewMemoryFrontier(0),
		HostPolicy{MinDelay: 100 * time.Millisecond, MaxDelay: time.Second},
		true,
		nil,
		0,
	)
	state := &schedulerHostState{
		minDelay:     100 * time.Millisecond,
		currentDelay: 100 * time.Millisecond,
	}
	scheduler.applyObservation(state, requestObservation{statusCode: 200, latency: 10 * time.Millisecond})
	scheduler.applyObservation(state, requestObservation{statusCode: 200, latency: 100 * time.Millisecond})
	assert.Equal(t, 125*time.Millisecond, state.currentDelay)

	for range 8 {
		scheduler.applyObservation(state, requestObservation{statusCode: 200, latency: 10 * time.Millisecond})
	}
	assert.Equal(t, 100*time.Millisecond, state.currentDelay)
}

func TestAdaptiveThrottleWithoutRetryAfterDoublesDelay(t *testing.T) {
	scheduler := newHostScheduler(
		NewMemoryFrontier(0),
		HostPolicy{MinDelay: 10 * time.Millisecond, MaxDelay: 35 * time.Millisecond},
		true,
		nil,
		0,
	)
	state := &schedulerHostState{
		minDelay:     10 * time.Millisecond,
		currentDelay: 10 * time.Millisecond,
	}
	// Status-driven backoff does not depend on timer resolution; Windows can
	// report zero for a very fast in-memory fetch.
	scheduler.applyObservation(state, requestObservation{statusCode: 503})
	assert.Equal(t, 20*time.Millisecond, state.currentDelay)
	scheduler.applyObservation(state, requestObservation{statusCode: 503, latency: time.Millisecond})
	assert.Equal(t, 35*time.Millisecond, state.currentDelay)
}

func TestCrawlerFreshTypedCacheAvoidsNetworkAndKeepsLinks(t *testing.T) {
	memory := cache.NewInMemoryCache()
	entry := &cache.Entry{
		URL:           "https://example.com",
		StatusCode:    http.StatusOK,
		Headers:       map[string]string{"ETag": `"v1"`},
		Body:          []byte("not parseable as useful HTML"),
		Links:         []fetch.Link{{URL: "/child"}},
		ETag:          `"v1"`,
		FetchedAt:     time.Now(),
		SchemaVersion: cache.ResponseSchemaVersion,
	}
	assert.NoError(t, memory.SetEntry(context.Background(), cache.ResponseKey("https://example.com"), entry))

	childFetcher := fetch.NewMockFetcher()
	childFetcher.AddResponse("https://example.com/child", &fetch.Response{
		URL:        "https://example.com/child",
		StatusCode: http.StatusOK,
		HTML:       "<html>child</html>",
	})
	c, err := New(Options{
		Cache:            memory,
		CacheTTL:         time.Hour,
		DefaultFetcher:   childFetcher,
		FollowBehavior:   FollowSameDomain,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var visited []string
	assert.NoError(t, c.Crawl(context.Background(), []string{"https://example.com"}, func(_ context.Context, result *Result) {
		visited = append(visited, result.URL.String())
	}))
	assert.Contains(t, visited, "https://example.com")
	assert.Contains(t, visited, "https://example.com/child")
	// Only the child required a request; the root and its extracted links came
	// directly from the typed entry.
	stats := c.HostStats()
	assert.Len(t, stats, 1)
	assert.Equal(t, int64(1), stats[0].Requests)
}

func TestCrawlerRevalidatesStaleTypedCache(t *testing.T) {
	memory := cache.NewInMemoryCache()
	staleAt := time.Now().Add(-2 * time.Hour)
	entry := &cache.Entry{
		URL:           "https://example.com/page",
		StatusCode:    http.StatusOK,
		Headers:       map[string]string{"ETag": `"v1"`, "Last-Modified": "Tue, 02 Sep 2025 12:00:00 GMT"},
		Body:          []byte("<html>cached</html>"),
		Links:         []fetch.Link{{URL: "/cached-link"}},
		ETag:          `"v1"`,
		LastModified:  "Tue, 02 Sep 2025 12:00:00 GMT",
		FetchedAt:     staleAt,
		SchemaVersion: cache.ResponseSchemaVersion,
	}
	key := cache.ResponseKey("https://example.com/page")
	assert.NoError(t, memory.SetEntry(context.Background(), key, entry))
	fetcher := &scriptedFetcher{responses: []*fetch.Response{{
		URL:        "https://example.com/page",
		StatusCode: http.StatusNotModified,
		Headers:    map[string]string{"ETag": `"v1"`},
	}}}
	c, err := New(Options{
		Cache:            memory,
		CacheTTL:         time.Minute,
		Revalidate:       true,
		DefaultFetcher:   fetcher,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var result *Result
	assert.NoError(t, c.Crawl(context.Background(), []string{"https://example.com/page"}, func(_ context.Context, got *Result) {
		result = got
	}))
	assert.NotNil(t, result)
	assert.Equal(t, "<html>cached</html>", result.Response.HTML)
	assert.Equal(t, http.StatusOK, result.Response.StatusCode)
	assert.Len(t, result.Response.Links, 1)

	fetcher.mu.Lock()
	request := fetcher.requests[0]
	fetcher.mu.Unlock()
	assert.Equal(t, `"v1"`, request.Headers["If-None-Match"])
	assert.Equal(t, entry.LastModified, request.Headers["If-Modified-Since"])
	refreshed, err := memory.GetEntry(context.Background(), key)
	assert.NoError(t, err)
	if !refreshed.FetchedAt.After(staleAt) {
		t.Fatalf("304 did not refresh FetchedAt: got %s, old %s", refreshed.FetchedAt, staleAt)
	}
	stats := c.HostStats()
	assert.Len(t, stats, 1)
	assert.Equal(t, int64(1), stats[0].StatusCodes[http.StatusNotModified])
}

func TestCrawlerRefreshesStaleCacheWithoutValidators(t *testing.T) {
	memory := cache.NewInMemoryCache()
	key := cache.ResponseKey("https://example.com/page")
	assert.NoError(t, memory.SetEntry(context.Background(), key, &cache.Entry{
		URL:           "https://example.com/page",
		StatusCode:    http.StatusOK,
		Body:          []byte("<html>old</html>"),
		FetchedAt:     time.Now().Add(-time.Hour),
		SchemaVersion: cache.ResponseSchemaVersion,
	}))
	fetcher := &scriptedFetcher{responses: []*fetch.Response{{
		URL:        "https://example.com/page",
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"ETag": `"v2"`},
		HTML:       "<html>new</html>",
	}}}
	c, err := New(Options{
		Cache:            memory,
		CacheTTL:         time.Minute,
		DefaultFetcher:   fetcher,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
	})
	assert.NoError(t, err)

	var result *Result
	assert.NoError(t, c.Crawl(context.Background(), []string{"https://example.com/page"}, func(_ context.Context, got *Result) {
		result = got
	}))
	assert.Equal(t, "<html>new</html>", result.Response.HTML)
	fetcher.mu.Lock()
	assert.Len(t, fetcher.requests[0].Headers, 0)
	fetcher.mu.Unlock()
	refreshed, err := memory.GetEntry(context.Background(), key)
	assert.NoError(t, err)
	assert.Equal(t, "<html>new</html>", string(refreshed.Body))
	assert.Equal(t, `"v2"`, refreshed.ETag)
}

func TestStoreFetchedResponseSkipsUncacheableStatuses(t *testing.T) {
	for _, statusCode := range []int{
		http.StatusNotModified,
		http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
	} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			memory := cache.NewInMemoryCache()
			c, err := New(Options{Cache: memory})
			assert.NoError(t, err)

			rawURL := "https://example.com/uncacheable"
			c.storeFetchedResponse(context.Background(), rawURL, &fetch.Response{
				URL:        rawURL,
				StatusCode: statusCode,
				HTML:       "do not replay",
			})

			_, err = memory.GetEntry(context.Background(), cache.ResponseKey(rawURL))
			assert.True(t, cache.IsNotFound(err))
		})
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)
	delay, ok := parseRetryAfter(now.Add(5*time.Second).Format(http.TimeFormat), now)
	assert.True(t, ok)
	assert.Equal(t, 5*time.Second, delay)
}
