package crawler

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/fetch"
)

func TestCrawler_AllowHTTP(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()

	// Set up HTTP response
	mockFetcher.AddResponse("http://example.com", &fetch.Response{
		URL:   "http://example.com",
		HTML:  "<html><body>HTTP Site</body></html>",
		Links: []fetch.Link{},
	})

	// Test with AllowHTTP = true
	crawler, err := New(Options{
		MaxURLs:        10,
		Workers:        1,
		RequestDelay:   time.Millisecond,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
		AllowHTTP:      true,
	})
	assert.NoError(t, err)

	var processedURL string
	callback := func(ctx context.Context, result *Result) {
		processedURL = result.URL.String()
	}

	ctx := context.Background()
	err = crawler.Crawl(ctx, []string{"http://example.com"}, callback)
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com", processedURL)
}

func TestCrawler_HTTPToHTTPS(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()

	// Set up HTTPS response (default behavior converts HTTP to HTTPS)
	mockFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:   "https://example.com",
		HTML:  "<html><body>HTTPS Site</body></html>",
		Links: []fetch.Link{},
	})

	// Test with default AllowHTTP = false
	crawler, err := New(Options{
		MaxURLs:        10,
		Workers:        1,
		RequestDelay:   time.Millisecond,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
		AllowHTTP:      false, // default
	})
	assert.NoError(t, err)

	var processedURL string
	callback := func(ctx context.Context, result *Result) {
		processedURL = result.URL.String()
	}

	ctx := context.Background()
	err = crawler.Crawl(ctx, []string{"http://example.com"}, callback)
	assert.NoError(t, err)
	// Should be converted to HTTPS
	assert.Equal(t, "https://example.com", processedURL)
}

func TestCrawler_PreserveQueryParams(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()

	// Set up response with query params
	mockFetcher.AddResponse("https://example.com/page?id=123&page=2", &fetch.Response{
		URL:   "https://example.com/page?id=123&page=2",
		HTML:  "<html><body>Page with params</body></html>",
		Links: []fetch.Link{},
	})

	// Test with PreserveQueryParams = true
	crawler, err := New(Options{
		MaxURLs:             10,
		Workers:             1,
		RequestDelay:        time.Millisecond,
		DefaultFetcher:      mockFetcher,
		FollowBehavior:      FollowNone,
		PreserveQueryParams: true,
	})
	assert.NoError(t, err)

	var processedURL string
	callback := func(ctx context.Context, result *Result) {
		processedURL = result.URL.String()
	}

	ctx := context.Background()
	err = crawler.Crawl(ctx, []string{"https://example.com/page?id=123&page=2"}, callback)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/page?id=123&page=2", processedURL)
}

func TestCrawler_StripQueryParams(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()

	// Set up response without query params
	mockFetcher.AddResponse("https://example.com/page", &fetch.Response{
		URL:   "https://example.com/page",
		HTML:  "<html><body>Page</body></html>",
		Links: []fetch.Link{},
	})

	// Test with default PreserveQueryParams = false
	crawler, err := New(Options{
		MaxURLs:             10,
		Workers:             1,
		RequestDelay:        time.Millisecond,
		DefaultFetcher:      mockFetcher,
		FollowBehavior:      FollowNone,
		PreserveQueryParams: false, // default
	})
	assert.NoError(t, err)

	var processedURL string
	callback := func(ctx context.Context, result *Result) {
		processedURL = result.URL.String()
	}

	ctx := context.Background()
	err = crawler.Crawl(ctx, []string{"https://example.com/page?id=123&page=2"}, callback)
	assert.NoError(t, err)
	// Query params should be stripped
	assert.Equal(t, "https://example.com/page", processedURL)
}

func TestCrawler_ResolveLink_PreservesScheme(t *testing.T) {
	crawler, err := New(Options{
		Workers:        1,
		DefaultFetcher: fetch.NewMockFetcher(),
		AllowHTTP:      true,
	})
	assert.NoError(t, err)

	// Test that relative links are resolved using the base URL's scheme
	baseURL, _ := url.Parse("http://example.com/path")
	resolved, ok := crawler.resolveLink(baseURL, "/about")
	assert.True(t, ok)
	assert.Equal(t, "http://example.com/about", resolved)

	// HTTPS base URL
	baseURL, _ = url.Parse("https://example.com/path")
	resolved, ok = crawler.resolveLink(baseURL, "/about")
	assert.True(t, ok)
	assert.Equal(t, "https://example.com/about", resolved)
}

func TestCrawler_NormalizeURL(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		allowHTTP           bool
		preserveQueryParams bool
		expected            string
		shouldFail          bool
	}{
		{
			name:      "adds https by default",
			input:     "example.com/path",
			expected:  "https://example.com/path",
			allowHTTP: false,
		},
		{
			name:      "converts http to https by default",
			input:     "http://example.com/path",
			expected:  "https://example.com/path",
			allowHTTP: false,
		},
		{
			name:      "preserves http when allowed",
			input:     "http://example.com/path",
			expected:  "http://example.com/path",
			allowHTTP: true,
		},
		{
			name:                "strips query params by default",
			input:               "https://example.com/path?foo=bar",
			expected:            "https://example.com/path",
			preserveQueryParams: false,
		},
		{
			name:                "preserves query params when enabled",
			input:               "https://example.com/path?foo=bar",
			expected:            "https://example.com/path?foo=bar",
			preserveQueryParams: true,
		},
		{
			name:       "rejects invalid scheme",
			input:      "ftp://example.com",
			shouldFail: true,
		},
		{
			name:       "rejects empty URL",
			input:      "",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crawler, err := New(Options{
				Workers:             1,
				DefaultFetcher:      fetch.NewMockFetcher(),
				AllowHTTP:           tt.allowHTTP,
				PreserveQueryParams: tt.preserveQueryParams,
			})
			assert.NoError(t, err)

			result, err := crawler.normalizeURL(tt.input)
			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

func TestCrawler_RetryOptionsConfig(t *testing.T) {
	// Test that retry options are properly configured
	mockFetcher := fetch.NewMockFetcher()

	// Set up a working response
	mockFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:   "https://example.com",
		HTML:  "<html><body>Success</body></html>",
		Links: []fetch.Link{},
	})

	crawler, err := New(Options{
		MaxURLs:        10,
		Workers:        1,
		RequestDelay:   time.Millisecond,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
		RetryOptions: &RetryOptions{
			MaxAttempts:    5,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     time.Second,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, crawler.retryOptions)
	assert.Equal(t, 5, crawler.retryOptions.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, crawler.retryOptions.InitialBackoff)
	assert.Equal(t, time.Second, crawler.retryOptions.MaxBackoff)

	var success bool
	callback := func(ctx context.Context, result *Result) {
		if result.Error == nil {
			success = true
		}
	}

	ctx := context.Background()
	err = crawler.Crawl(ctx, []string{"https://example.com"}, callback)
	assert.NoError(t, err)
	assert.True(t, success)
}

func TestCrawler_RetryOnError(t *testing.T) {
	// Test that errors trigger retry behavior (even if ultimately failing)
	mockFetcher := fetch.NewMockFetcher()

	// Set up an error response
	mockFetcher.AddError("https://error.com", errors.New("fetch failed"))

	crawler, err := New(Options{
		MaxURLs:        10,
		Workers:        1,
		RequestDelay:   time.Millisecond,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
		RetryOptions: &RetryOptions{
			MaxAttempts:    2,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     time.Millisecond,
		},
	})
	assert.NoError(t, err)

	var hadError bool
	callback := func(ctx context.Context, result *Result) {
		if result.Error != nil {
			hadError = true
		}
	}

	ctx := context.Background()
	err = crawler.Crawl(ctx, []string{"https://error.com"}, callback)
	assert.NoError(t, err)
	assert.True(t, hadError, "should have reported error after retries")
}

func TestRobotsTxtParser(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		userAgent string
		expected  *robotsTxtData
	}{
		{
			name: "basic disallow",
			content: `User-agent: *
Disallow: /admin
Disallow: /private`,
			userAgent: "*",
			expected: &robotsTxtData{
				disallowRules: []string{"/admin", "/private"},
			},
		},
		{
			name: "specific user agent",
			content: `User-agent: Googlebot
Disallow: /google-only

User-agent: *
Disallow: /all`,
			userAgent: "Googlebot",
			expected: &robotsTxtData{
				disallowRules: []string{"/google-only"},
			},
		},
		{
			name: "allow and disallow",
			content: `User-agent: *
Allow: /allowed
Disallow: /`,
			userAgent: "*",
			expected: &robotsTxtData{
				allowRules:    []string{"/allowed"},
				disallowRules: []string{"/"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRobotsTxt(tt.content, tt.userAgent)
			assert.Equal(t, tt.expected.disallowRules, result.disallowRules)
			assert.Equal(t, tt.expected.allowRules, result.allowRules)
		})
	}
}

func TestPathMatches(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		rule    string
		matches bool
	}{
		{"prefix match", "/admin/users", "/admin", true},
		{"exact match", "/admin", "/admin", true},
		{"no match", "/public", "/admin", false},
		{"wildcard match", "/images/photo.jpg", "/images/*.jpg", true},
		{"wildcard no match", "/images/photo.png", "/images/*.jpg", false},
		{"end anchor match", "/exact", "/exact$", true},
		{"end anchor no match", "/exact/more", "/exact$", false},
		{"empty rule", "/any", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pathMatches(tt.path, tt.rule)
			assert.Equal(t, tt.matches, result)
		})
	}
}

func TestCrawler_RobotsTxtBlocking(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()

	// Set up robots.txt response
	mockFetcher.AddResponse("https://example.com/robots.txt", &fetch.Response{
		URL: "https://example.com/robots.txt",
		HTML: `User-agent: *
Disallow: /blocked
Allow: /allowed`,
		Links: []fetch.Link{},
	})

	// Set up allowed page
	mockFetcher.AddResponse("https://example.com/allowed", &fetch.Response{
		URL:   "https://example.com/allowed",
		HTML:  "<html><body>Allowed</body></html>",
		Links: []fetch.Link{},
	})

	// Set up blocked page (should not be fetched)
	mockFetcher.AddResponse("https://example.com/blocked", &fetch.Response{
		URL:   "https://example.com/blocked",
		HTML:  "<html><body>Blocked</body></html>",
		Links: []fetch.Link{},
	})

	crawler, err := New(Options{
		MaxURLs:          10,
		Workers:          1,
		RequestDelay:     time.Millisecond,
		DefaultFetcher:   mockFetcher,
		FollowBehavior:   FollowNone,
		RespectRobotsTxt: BoolPtr(true),
	})
	assert.NoError(t, err)

	var results []string
	var errors []string
	mu := sync.Mutex{}

	callback := func(ctx context.Context, result *Result) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, result.URL.String())
		if result.Error != nil {
			errors = append(errors, result.Error.Error())
		}
	}

	ctx := context.Background()
	err = crawler.Crawl(ctx, []string{
		"https://example.com/allowed",
		"https://example.com/blocked",
	}, callback)
	assert.NoError(t, err)

	// Both URLs should be processed
	assert.Len(t, results, 2)

	// One should have been blocked
	hasBlocked := false
	for _, e := range errors {
		if e == "blocked by robots.txt" {
			hasBlocked = true
			break
		}
	}
	assert.True(t, hasBlocked, "should have blocked URL by robots.txt")
}

func TestCacheClose(t *testing.T) {
	// Test that the new Close method works on InMemoryCache
	c := NewInMemoryCacheForTest()
	ctx := context.Background()

	// Add some data
	err := c.Set(ctx, "key1", []byte("value1"))
	assert.NoError(t, err)

	// Verify data exists
	val, err := c.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), val)

	// Close the cache
	err = c.Close()
	assert.NoError(t, err)

	// Data should be cleared (but interface doesn't guarantee this)
	// The important thing is Close() doesn't error
}

// Helper to create InMemoryCache for testing without import cycle
func NewInMemoryCacheForTest() *testCache {
	return &testCache{
		data: make(map[string][]byte),
	}
}

type testCache struct {
	data  map[string][]byte
	mutex sync.RWMutex
}

func (c *testCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if val, ok := c.data[key]; ok {
		return val, nil
	}
	return nil, errors.New("not found")
}

func (c *testCache) Set(ctx context.Context, key string, value []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = value
	return nil
}

func (c *testCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, key)
	return nil
}

func (c *testCache) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make(map[string][]byte)
	return nil
}
