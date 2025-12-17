package crawler

import (
	"context"
	"errors"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/fetch"
)

// TestCrawlerWithFetcherRules tests that the crawler correctly selects fetchers based on rules
func TestCrawlerWithFetcherRules(t *testing.T) {
	// Create mock fetchers with different behaviors
	defaultFetcher := fetch.NewMockFetcher()
	specialFetcher := fetch.NewMockFetcher()
	govFetcher := fetch.NewMockFetcher()
	apiFetcher := fetch.NewMockFetcher()

	// Add responses to track which fetcher was used
	defaultFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:        "https://example.com",
		StatusCode: 200,
		HTML:       "<html>Default fetcher</html>",
	})

	specialFetcher.AddResponse("https://special.example.com", &fetch.Response{
		URL:        "https://special.example.com",
		StatusCode: 200,
		HTML:       "<html>Special fetcher</html>",
	})

	govFetcher.AddResponse("https://agency.gov", &fetch.Response{
		URL:        "https://agency.gov",
		StatusCode: 200,
		HTML:       "<html>Gov fetcher</html>",
	})

	apiFetcher.AddResponse("https://api.example.com", &fetch.Response{
		URL:        "https://api.example.com",
		StatusCode: 200,
		HTML:       "<html>API fetcher</html>",
	})

	// Create fetcher rules
	fetcherRules := []*FetcherRule{
		NewFetcherRule("special.example.com", specialFetcher, WithFetcherPriority(100)),
		NewFetcherRule(".gov", govFetcher, WithFetcherMatchType(MatchSuffix), WithFetcherPriority(90)),
		NewFetcherRule("api.", apiFetcher, WithFetcherMatchType(MatchPrefix), WithFetcherPriority(80)),
	}

	// Create crawler with fetcher rules
	c, err := New(Options{
		MaxURLs:        10,
		Workers:        1,
		FetcherRules:   fetcherRules,
		DefaultFetcher: defaultFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	// Track which URLs were crawled and their responses
	results := make(map[string]string)
	callback := func(ctx context.Context, result *Result) {
		if result.Response != nil {
			results[result.URL.String()] = result.Response.HTML
		}
	}

	// Crawl URLs that should use different fetchers
	urls := []string{
		"https://example.com",         // Should use default fetcher
		"https://special.example.com", // Should use special fetcher (exact match)
		"https://agency.gov",          // Should use gov fetcher (suffix match)
		"https://api.example.com",     // Should use API fetcher (prefix match)
	}

	err = c.Crawl(context.Background(), urls, callback)
	assert.NoError(t, err)

	// Verify the correct fetchers were used
	assert.Equal(t, "<html>Default fetcher</html>", results["https://example.com"])
	assert.Equal(t, "<html>Special fetcher</html>", results["https://special.example.com"])
	assert.Equal(t, "<html>Gov fetcher</html>", results["https://agency.gov"])
	assert.Equal(t, "<html>API fetcher</html>", results["https://api.example.com"])
}

// TestCrawlerWithFetcherRulePriority tests that fetcher rules are applied in priority order
func TestCrawlerWithFetcherRulePriority(t *testing.T) {
	// Create mock fetchers
	highPriorityFetcher := fetch.NewMockFetcher()
	lowPriorityFetcher := fetch.NewMockFetcher()
	defaultFetcher := fetch.NewMockFetcher()

	// Both fetchers can handle the same URL
	testURL := "https://api.example.com"
	highPriorityFetcher.AddResponse(testURL, &fetch.Response{
		URL:        testURL,
		StatusCode: 200,
		HTML:       "<html>High priority</html>",
	})
	lowPriorityFetcher.AddResponse(testURL, &fetch.Response{
		URL:        testURL,
		StatusCode: 200,
		HTML:       "<html>Low priority</html>",
	})

	// Create overlapping rules with different priorities
	fetcherRules := []*FetcherRule{
		NewFetcherRule(".com", lowPriorityFetcher, WithFetcherMatchType(MatchSuffix), WithFetcherPriority(50)),   // Lower priority
		NewFetcherRule("api.", highPriorityFetcher, WithFetcherMatchType(MatchPrefix), WithFetcherPriority(100)), // Higher priority
	}

	// Create crawler
	c, err := New(Options{
		MaxURLs:        1,
		Workers:        1,
		FetcherRules:   fetcherRules,
		DefaultFetcher: defaultFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	// Track result
	var capturedHTML string
	callback := func(ctx context.Context, result *Result) {
		if result.Response != nil {
			capturedHTML = result.Response.HTML
		}
	}

	err = c.Crawl(context.Background(), []string{testURL}, callback)
	assert.NoError(t, err)

	// Verify the higher priority fetcher was used
	assert.Equal(t, "<html>High priority</html>", capturedHTML)
}

// TestCrawlerWithNoMatchingFetcher tests behavior when no fetcher matches
func TestCrawlerWithNoMatchingFetcher(t *testing.T) {
	// Create a crawler with no default fetcher and limited rules
	specialFetcher := fetch.NewMockFetcher()

	fetcherRules := []*FetcherRule{
		NewFetcherRule("special.example.com", specialFetcher, WithFetcherPriority(100)),
	}

	c, err := New(Options{
		MaxURLs:        1,
		Workers:        1,
		FetcherRules:   fetcherRules,
		DefaultFetcher: nil, // No default fetcher
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	// Track errors
	var capturedError error
	callback := func(ctx context.Context, result *Result) {
		if result.Error != nil {
			capturedError = result.Error
		}
	}

	// Try to crawl a URL that doesn't match any rule
	err = c.Crawl(context.Background(), []string{"https://nomatch.example.com"}, callback)
	assert.NoError(t, err)

	// Verify an error was captured
	assert.Error(t, capturedError)
	assert.Contains(t, capturedError.Error(), "no fetcher configured")
}

// TestCrawlerMixedRules tests a crawler with both parser and fetcher rules
func TestCrawlerMixedRules(t *testing.T) {
	// Create mock fetcher and parser
	mockFetcher := fetch.NewMockFetcher()
	mockParser := NewMockParser()

	// Setup fetcher response
	mockFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:        "https://example.com",
		StatusCode: 200,
		HTML:       "<html><body>Test content</body></html>",
	})

	// Setup parser response
	parsedData := map[string]string{"parsed": "data"}
	mockParser.SetParseFunc(func(ctx context.Context, page *fetch.Response) (any, error) {
		// Verify the response has the expected content
		if page.URL == "https://example.com" && page.HTML == "<html><body>Test content</body></html>" {
			return parsedData, nil
		}
		return nil, errors.New("unexpected response")
	})

	// Create rules
	fetcherRules := []*FetcherRule{
		NewFetcherRule("example.com", mockFetcher, WithFetcherPriority(100)),
	}
	parserRules := []*ParserRule{
		NewParserRule("example.com", mockParser, WithParserPriority(100)),
	}

	// Create crawler with both types of rules
	c, err := New(Options{
		MaxURLs:        1,
		Workers:        1,
		FetcherRules:   fetcherRules,
		ParserRules:    parserRules,
		DefaultFetcher: mockFetcher,
		DefaultParser:  mockParser,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	// Track results
	var capturedParsed any
	callback := func(ctx context.Context, result *Result) {
		if result.Parsed != nil {
			capturedParsed = result.Parsed
		}
	}

	err = c.Crawl(context.Background(), []string{"https://example.com"}, callback)
	assert.NoError(t, err)

	// Verify both fetcher and parser were used correctly
	assert.Equal(t, parsedData, capturedParsed)
}

// TestCrawlerWithRegexFetcherRule tests regex pattern matching for fetcher rules
func TestCrawlerWithRegexFetcherRule(t *testing.T) {
	// Create mock fetchers
	regexFetcher := fetch.NewMockFetcher()
	defaultFetcher := fetch.NewMockFetcher()

	// Setup responses
	urls := []string{
		"https://blog.example.com",
		"https://news.example.com",
		"https://api.example.com",
		"https://example.com",
	}

	for _, url := range urls {
		regexFetcher.AddResponse(url, &fetch.Response{
			URL:        url,
			StatusCode: 200,
			HTML:       "<html>Regex fetcher</html>",
		})
		defaultFetcher.AddResponse(url, &fetch.Response{
			URL:        url,
			StatusCode: 200,
			HTML:       "<html>Default fetcher</html>",
		})
	}

	// Create regex rule that matches subdomains with specific pattern
	fetcherRules := []*FetcherRule{
		NewFetcherRule(
			`^(blog|news)\..*\.com$`,
			regexFetcher,
			WithFetcherPriority(100),
			WithFetcherMatchType(MatchRegex),
		),
	}

	c, err := New(Options{
		MaxURLs:        10,
		Workers:        1,
		FetcherRules:   fetcherRules,
		DefaultFetcher: defaultFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	// Track results
	results := make(map[string]string)
	callback := func(ctx context.Context, result *Result) {
		if result.Response != nil {
			results[result.URL.String()] = result.Response.HTML
		}
	}

	err = c.Crawl(context.Background(), urls, callback)
	assert.NoError(t, err)

	// Verify regex matching worked correctly
	assert.Equal(t, "<html>Regex fetcher</html>", results["https://blog.example.com"])
	assert.Equal(t, "<html>Regex fetcher</html>", results["https://news.example.com"])
	assert.Equal(t, "<html>Default fetcher</html>", results["https://api.example.com"])
	assert.Equal(t, "<html>Default fetcher</html>", results["https://example.com"])
}

// TestCrawlerFetcherRuleCompileError tests handling of invalid regex patterns
func TestCrawlerFetcherRuleCompileError(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()

	// Create rule with invalid regex
	rule := &FetcherRule{
		MatchRule: MatchRule{
			Pattern:  "[invalid regex",
			Type:     MatchRegex,
			Priority: 100,
		},
		Fetcher: mockFetcher,
	}

	_, err := New(Options{
		MaxURLs:        1,
		Workers:        1,
		FetcherRules:   []*FetcherRule{rule},
		DefaultFetcher: mockFetcher,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing regexp")
}

// TestCrawlerWithGlobFetcherRule tests glob pattern matching for fetcher rules
func TestCrawlerWithGlobFetcherRule(t *testing.T) {
	// Create mock fetchers
	globFetcher := fetch.NewMockFetcher()
	defaultFetcher := fetch.NewMockFetcher()

	// Setup responses
	testCases := []struct {
		url      string
		expected string
	}{
		{"https://api.v1.example.com", "Glob fetcher"},
		{"https://api.v2.example.com", "Glob fetcher"},
		{"https://api.example.com", "Default fetcher"},
		{"https://example.com", "Default fetcher"},
	}

	for _, tc := range testCases {
		response := &fetch.Response{
			URL:        tc.url,
			StatusCode: 200,
			HTML:       "<html>" + tc.expected + "</html>",
		}
		if tc.expected == "Glob fetcher" {
			globFetcher.AddResponse(tc.url, response)
		} else {
			defaultFetcher.AddResponse(tc.url, response)
		}
	}

	// Create glob rule
	fetcherRules := []*FetcherRule{
		NewFetcherRule(
			"api.*.example.com",
			globFetcher,
			WithFetcherPriority(100),
			WithFetcherMatchType(MatchGlob),
		),
	}

	c, err := New(Options{
		MaxURLs:        10,
		Workers:        1,
		FetcherRules:   fetcherRules,
		DefaultFetcher: defaultFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	// Track results
	results := make(map[string]string)
	callback := func(ctx context.Context, result *Result) {
		if result.Response != nil {
			results[result.URL.String()] = result.Response.HTML
		}
	}

	// Extract URLs from test cases
	var urls []string
	for _, tc := range testCases {
		urls = append(urls, tc.url)
	}

	err = c.Crawl(context.Background(), urls, callback)
	assert.NoError(t, err)

	// Verify glob matching worked correctly
	for _, tc := range testCases {
		assert.Equal(t, "<html>"+tc.expected+"</html>", results[tc.url])
	}
}

// TestAddFetcherRulesAfterCreation tests adding fetcher rules after crawler creation
func TestAddFetcherRulesAfterCreation(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	additionalFetcher := fetch.NewMockFetcher()

	// Setup responses
	mockFetcher.AddResponse("https://example.com", &fetch.Response{
		URL:        "https://example.com",
		StatusCode: 200,
		HTML:       "<html>Original fetcher</html>",
	})
	additionalFetcher.AddResponse("https://special.example.com", &fetch.Response{
		URL:        "https://special.example.com",
		StatusCode: 200,
		HTML:       "<html>Additional fetcher</html>",
	})

	// Create crawler with minimal configuration
	c, err := New(Options{
		MaxURLs:        10,
		Workers:        1,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	// Add fetcher rule after creation
	err = c.AddFetcherRules(NewFetcherRule("special.example.com", additionalFetcher, WithFetcherPriority(100)))
	assert.NoError(t, err)

	// Track results
	results := make(map[string]string)
	callback := func(ctx context.Context, result *Result) {
		if result.Response != nil {
			results[result.URL.String()] = result.Response.HTML
		}
	}

	urls := []string{
		"https://example.com",
		"https://special.example.com",
	}

	err = c.Crawl(context.Background(), urls, callback)
	assert.NoError(t, err)

	// Verify both fetchers were used
	assert.Equal(t, "<html>Original fetcher</html>", results["https://example.com"])
	assert.Equal(t, "<html>Additional fetcher</html>", results["https://special.example.com"])
}

// TestCrawlerFetcherError tests proper error handling when fetcher returns an error
func TestCrawlerFetcherError(t *testing.T) {
	mockFetcher := fetch.NewMockFetcher()
	fetchError := errors.New("network timeout")

	// Setup fetcher to return error
	mockFetcher.AddError("https://error.example.com", fetchError)

	c, err := New(Options{
		MaxURLs:        1,
		Workers:        1,
		DefaultFetcher: mockFetcher,
		FollowBehavior: FollowNone,
	})
	assert.NoError(t, err)

	// Track errors
	var capturedError error
	callback := func(ctx context.Context, result *Result) {
		if result.Error != nil {
			capturedError = result.Error
		}
	}

	err = c.Crawl(context.Background(), []string{"https://error.example.com"}, callback)
	assert.NoError(t, err)

	// Verify error was properly captured
	assert.Equal(t, fetchError, capturedError)

	// Verify stats
	stats := c.GetStats()
	assert.Equal(t, int64(1), stats.GetProcessed())
	assert.Equal(t, int64(0), stats.GetSucceeded())
	assert.Equal(t, int64(1), stats.GetFailed())
}
