package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/fetch"
	"github.com/deepnoodle-ai/wonton/httpguard"
)

// hitRecorder is an httptest handler wrapper that records request URIs.
type hitRecorder struct {
	mu   sync.Mutex
	hits []string
}

func (h *hitRecorder) record(r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hits = append(h.hits, r.URL.RequestURI())
}

func (h *hitRecorder) count(uri string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, hit := range h.hits {
		if hit == uri {
			n++
		}
	}
	return n
}

func newSiteServer(t *testing.T, rec *hitRecorder, routes map[string]func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		if handler, ok := routes[r.URL.Path]; ok {
			handler(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func htmlPage(body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<html><body>%s</body></html>", body)
	}
}

func newHTTPCrawler(t *testing.T, server *httptest.Server, opts Options) *Crawler {
	t.Helper()
	opts.DefaultFetcher = fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{Client: server.Client()})
	opts.AllowHTTP = true
	if opts.Workers == 0 {
		opts.Workers = 1
	}
	c, err := New(opts)
	assert.NoError(t, err)
	return c
}

// BUG-010: robots.txt is served as text/plain, which HTTPFetcher used to
// reject, so the crawler silently allowed everything.
func TestRobotsTxtEnforcedWithHTTPFetcher(t *testing.T) {
	rec := &hitRecorder{}
	server := newSiteServer(t, rec, map[string]func(http.ResponseWriter, *http.Request){
		"/robots.txt": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "User-agent: *\nDisallow: /private\n")
		},
		"/":        htmlPage(`<a href="/private">p</a><a href="/public">q</a>`),
		"/public":  htmlPage("public"),
		"/private": htmlPage("private"),
	})
	c := newHTTPCrawler(t, server, Options{FollowBehavior: FollowSameDomain})

	var mu sync.Mutex
	var blocked []string
	err := c.Crawl(context.Background(), []string{server.URL + "/"}, func(_ context.Context, r *Result) {
		if r.Error != nil {
			mu.Lock()
			blocked = append(blocked, r.URL.Path)
			mu.Unlock()
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, rec.count("/private"), "disallowed path was fetched")
	assert.Equal(t, 1, rec.count("/public"))
	assert.Equal(t, []string{"/private"}, blocked)
}

// A 5xx robots.txt means the rules are unavailable; RFC 9309 requires a
// complete disallow rather than crawling everything.
func TestRobotsTxtServerErrorDisallowsAll(t *testing.T) {
	rec := &hitRecorder{}
	server := newSiteServer(t, rec, map[string]func(http.ResponseWriter, *http.Request){
		"/robots.txt": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		},
		"/": htmlPage("home"),
	})
	c := newHTTPCrawler(t, server, Options{})

	var blocked int
	err := c.Crawl(context.Background(), []string{server.URL + "/"}, func(_ context.Context, r *Result) {
		if r.Error != nil {
			blocked++
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, rec.count("/"), "page fetched despite 5xx robots.txt")
	assert.Equal(t, 1, blocked)
}

// BUG-017: relative links resolve against the page URL as served, keeping a
// directory's trailing slash and following redirects.
func TestRelativeLinksResolveAgainstDirectoryAndRedirect(t *testing.T) {
	rec := &hitRecorder{}
	server := newSiteServer(t, rec, map[string]func(http.ResponseWriter, *http.Request){
		"/docs/guide/":       htmlPage(`<a href="intro">intro</a>`),
		"/docs/guide/intro":  htmlPage("intro"),
		"/old":               func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/new/", http.StatusFound) },
		"/new/":              htmlPage(`<a href="page">page</a>`),
		"/new/page":          htmlPage("page"),
		"/search":            htmlPage("search"),
		"/robots.txt":        func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
		"/docs/intro":        htmlPage("wrong level"),
		"/page":              htmlPage("wrong level"),
		"/docs/guide/intro/": htmlPage("wrong"),
	})
	c := newHTTPCrawler(t, server, Options{FollowBehavior: FollowSameDomain, PreserveQueryParams: true})

	var mu sync.Mutex
	var links []string
	err := c.Crawl(context.Background(), []string{
		server.URL + "/docs/guide/",
		server.URL + "/old",
		server.URL + "/search?path=/a/b/",
	}, func(_ context.Context, r *Result) {
		mu.Lock()
		links = append(links, r.Links...)
		mu.Unlock()
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, rec.count("/docs/guide/"), "trailing slash dropped from seed")
	assert.Equal(t, 1, rec.count("/docs/guide/intro"))
	assert.Equal(t, 0, rec.count("/docs/intro"))
	assert.Equal(t, 1, rec.count("/new/page"))
	assert.Equal(t, 0, rec.count("/page"))
	assert.Equal(t, 1, rec.count("/search?path=/a/b/"), "query value trailing slash stripped")
	sort.Strings(links)
	assert.Equal(t, []string{server.URL + "/docs/guide/intro", server.URL + "/new/page"}, links)
}

// BUG-058: consecutive User-agent lines share one group, and inline comments
// are not part of the value.
func TestParseRobotsTxtGroupsAndComments(t *testing.T) {
	content := "User-agent: MyBot\nUser-agent: OtherBot\nDisallow: /\n\nUser-agent: *\nDisallow: /tmp\n"
	data := parseRobotsTxt(content, "MyBot")
	assert.Equal(t, []string{"/"}, data.disallowRules)

	data = parseRobotsTxt(content, "OtherBot")
	assert.Equal(t, []string{"/"}, data.disallowRules)

	data = parseRobotsTxt(content, "ThirdBot")
	assert.Equal(t, []string{"/tmp"}, data.disallowRules)

	// A User-agent line after rules starts a new group.
	content = "User-agent: MyBot\nDisallow: /a\nUser-agent: OtherBot\nDisallow: /b\n"
	data = parseRobotsTxt(content, "MyBot")
	assert.Equal(t, []string{"/a"}, data.disallowRules)

	content = "User-agent: * # everyone\nDisallow: /admin # internal\nAllow: /admin/public#x\nCrawl-delay: 5 # sec\n"
	data = parseRobotsTxt(content, "AnyBot")
	assert.Equal(t, []string{"/admin"}, data.disallowRules)
	assert.Equal(t, []string{"/admin/public"}, data.allowRules)
	assert.Equal(t, 5*time.Second, data.crawlDelay)
}

// BUG-059: HTTP error pages are failures: 5xx is retried, 4xx is not, and
// neither has its links followed or counts as a success.
func TestHTTPErrorPagesAreFailures(t *testing.T) {
	rec := &hitRecorder{}
	server := newSiteServer(t, rec, map[string]func(http.ResponseWriter, *http.Request){
		"/boom": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `<html><body><a href="/from-500">x</a></body></html>`)
		},
		"/missing": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "TEXT/HTML")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `<html><body><a href="/from-404">x</a></body></html>`)
		},
		"/from-500": htmlPage("x"),
		"/from-404": htmlPage("x"),
	})
	c := newHTTPCrawler(t, server, Options{
		FollowBehavior:   FollowSameDomain,
		RespectRobotsTxt: BoolPtr(false),
		RetryOptions:     &RetryOptions{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond},
	})

	var mu sync.Mutex
	errs := map[string]error{}
	err := c.Crawl(context.Background(), []string{server.URL + "/boom", server.URL + "/missing"}, func(_ context.Context, r *Result) {
		mu.Lock()
		errs[r.URL.Path] = r.Error
		mu.Unlock()
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, rec.count("/boom"), "500 should be retried")
	assert.Equal(t, 1, rec.count("/missing"), "404 should not be retried")
	assert.Equal(t, 0, rec.count("/from-500"))
	assert.Equal(t, 0, rec.count("/from-404"))
	var fetchErr *fetch.Error
	assert.True(t, errors.As(errs["/boom"], &fetchErr), "boom error: %v", errs["/boom"])
	assert.Equal(t, http.StatusInternalServerError, fetchErr.StatusCode)
	assert.True(t, errors.As(errs["/missing"], &fetchErr), "missing error: %v", errs["/missing"])
	assert.Equal(t, http.StatusNotFound, fetchErr.StatusCode)
	assert.Equal(t, int64(0), c.GetStats().GetSucceeded())
	assert.Equal(t, int64(2), c.GetStats().GetFailed())
}

func TestThrottledResponsesRetriedWithoutAdaptive(t *testing.T) {
	var calls int32
	rec := &hitRecorder{}
	server := newSiteServer(t, rec, map[string]func(http.ResponseWriter, *http.Request){
		"/busy": func(w http.ResponseWriter, _ *http.Request) {
			if atomic.AddInt32(&calls, 1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			htmlPage("ok")(w, nil)
		},
	})
	c := newHTTPCrawler(t, server, Options{
		RespectRobotsTxt: BoolPtr(false),
		RetryOptions:     &RetryOptions{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond},
	})
	var gotErr error
	assert.NoError(t, c.Crawl(context.Background(), []string{server.URL + "/busy"}, func(_ context.Context, r *Result) {
		gotErr = r.Error
	}))
	assert.Equal(t, 2, rec.count("/busy"))
	assert.NoError(t, gotErr)
}

type countingFetcher struct {
	calls int32
	err   error
}

func (f *countingFetcher) Fetch(context.Context, *fetch.Request) (*fetch.Response, error) {
	atomic.AddInt32(&f.calls, 1)
	return nil, f.err
}

func TestPermanentFetchErrorsAreNotRetried(t *testing.T) {
	for name, permanent := range map[string]error{
		"non-public":   fmt.Errorf("dial: %w", httpguard.ErrNonPublicAddress),
		"redirect":     fmt.Errorf("get: %w", httpguard.ErrRedirectRefused),
		"content type": fmt.Errorf("%w: application/pdf", fetch.ErrUnsupportedContentType),
		"404":          &fetch.Error{StatusCode: http.StatusNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			fetcher := &countingFetcher{err: permanent}
			c, err := New(Options{
				DefaultFetcher:   fetcher,
				RespectRobotsTxt: BoolPtr(false),
				RetryOptions:     &RetryOptions{MaxAttempts: 3, InitialBackoff: time.Millisecond},
			})
			assert.NoError(t, err)
			var gotErr error
			assert.NoError(t, c.Crawl(context.Background(), []string{"https://example.com/x"}, func(_ context.Context, r *Result) {
				gotErr = r.Error
			}))
			assert.Equal(t, int32(1), atomic.LoadInt32(&fetcher.calls))
			assert.ErrorIs(t, gotErr, permanent)
		})
	}

	// Errors of unknown kind keep the old behavior and are retried.
	fetcher := &countingFetcher{err: errors.New("flaky")}
	c, err := New(Options{
		DefaultFetcher:   fetcher,
		RespectRobotsTxt: BoolPtr(false),
		RetryOptions:     &RetryOptions{MaxAttempts: 3, InitialBackoff: time.Millisecond},
	})
	assert.NoError(t, err)
	assert.NoError(t, c.Crawl(context.Background(), []string{"https://example.com/x"}, func(context.Context, *Result) {}))
	assert.Equal(t, int32(3), atomic.LoadInt32(&fetcher.calls))
}

// BUG-073: items preloaded into a caller's frontier honor MaxURLs,
// KnownURLs, normalization, and deduplication.
func TestPreloadedFrontierHonorsLimits(t *testing.T) {
	frontier := NewMemoryFrontier(0)
	for _, u := range []string{
		"https://example.com/known",
		"https://example.com/a",
		"https://EXAMPLE.com/a#frag",
		"https://example.com/b",
		"https://example.com/c",
		"https://example.com/d",
	} {
		assert.NoError(t, frontier.Push(context.Background(), URLItem{URL: u}))
	}
	mock := fetch.NewMockFetcher()
	for _, p := range []string{"known", "a", "b", "c", "d"} {
		mock.AddResponse("https://example.com/"+p, &fetch.Response{HTML: "<html></html>"})
	}
	c, err := New(Options{
		Frontier:         frontier,
		DefaultFetcher:   mock,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(false),
		MaxURLs:          2,
		Workers:          1,
		KnownURLs:        []string{"https://example.com/known"},
	})
	assert.NoError(t, err)

	var mu sync.Mutex
	var crawled []string
	assert.NoError(t, c.Crawl(context.Background(), nil, func(_ context.Context, r *Result) {
		mu.Lock()
		crawled = append(crawled, r.URL.String())
		mu.Unlock()
	}))
	assert.Equal(t, []string{"https://example.com/a", "https://example.com/b"}, crawled)
}
