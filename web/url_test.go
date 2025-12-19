package web

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "simple https URL",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "http URL converted to https",
			input:    "http://example.com",
			expected: "https://example.com",
		},
		{
			name:     "URL without protocol",
			input:    "example.com",
			expected: "https://example.com",
		},
		{
			name:     "URL with path",
			input:    "https://example.com/path",
			expected: "https://example.com/path",
		},
		{
			name:     "URL with root path removed",
			input:    "https://example.com/",
			expected: "https://example.com",
		},
		{
			name:     "URL with query and fragment removed",
			input:    "https://example.com/path?query=1#fragment",
			expected: "https://example.com/path",
		},
		{
			name:     "URL with whitespace",
			input:    "  https://example.com  ",
			expected: "https://example.com",
		},
		{
			name:        "empty URL",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid protocol",
			input:       "ftp://example.com",
			expectError: true,
		},
		{
			name:        "malformed URL",
			input:       "ht tp://example.com",
			expectError: true,
		},
		// Edge cases from feedback
		{
			name:     "httpbin.org - starts with http but no scheme",
			input:    "httpbin.org",
			expected: "https://httpbin.org",
		},
		{
			name:     "httpbin.org with path",
			input:    "httpbin.org/get",
			expected: "https://httpbin.org/get",
		},
		{
			name:        "mailto URL should be rejected",
			input:       "mailto:test@example.com",
			expectError: true,
		},
		{
			name:        "javascript URL should be rejected",
			input:       "javascript:void(0)",
			expectError: true,
		},
		{
			name:        "tel URL should be rejected",
			input:       "tel:+1234567890",
			expectError: true,
		},
		{
			name:        "data URL should be rejected",
			input:       "data:text/html,<h1>Hello</h1>",
			expectError: true,
		},
		{
			name:     "protocol-relative URL",
			input:    "//example.com/path",
			expected: "https://example.com/path",
		},
		{
			name:     "URL with port",
			input:    "https://example.com:8080/path",
			expected: "https://example.com:8080/path",
		},
		{
			name:     "http URL with port converted to https",
			input:    "http://example.com:8080/path",
			expected: "https://example.com:8080/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeURL(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

func TestAreSameHost(t *testing.T) {
	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{
			name:     "same domain",
			url1:     "https://example.com/path1",
			url2:     "https://example.com/path2",
			expected: true,
		},
		{
			name:     "different domains",
			url1:     "https://example.com",
			url2:     "https://google.com",
			expected: false,
		},
		{
			name:     "same domain different subdomains",
			url1:     "https://www.example.com",
			url2:     "https://api.example.com",
			expected: false,
		},
		{
			name:     "nil URLs",
			url1:     "",
			url2:     "",
			expected: false,
		},
		// Edge cases: port handling and case sensitivity
		{
			name:     "same host with and without port",
			url1:     "https://example.com:443/path",
			url2:     "https://example.com/path",
			expected: true,
		},
		{
			name:     "same host different ports",
			url1:     "https://example.com:8080/path",
			url2:     "https://example.com:9090/path",
			expected: true,
		},
		{
			name:     "case insensitive host comparison",
			url1:     "https://EXAMPLE.COM/path",
			url2:     "https://example.com/path",
			expected: true,
		},
		{
			name:     "mixed case comparison",
			url1:     "https://Example.Com/path",
			url2:     "https://EXAMPLE.COM/path",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u1, u2 *url.URL
			if tt.url1 != "" {
				u1, _ = url.Parse(tt.url1)
			}
			if tt.url2 != "" {
				u2, _ = url.Parse(tt.url2)
			}
			result := AreSameHost(u1, u2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAreRelatedHosts(t *testing.T) {
	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{
			name:     "same domain",
			url1:     "https://example.com",
			url2:     "https://example.com",
			expected: true,
		},
		{
			name:     "related subdomains",
			url1:     "https://www.example.com",
			url2:     "https://api.example.com",
			expected: true,
		},
		{
			name:     "different base domains",
			url1:     "https://example.com",
			url2:     "https://google.com",
			expected: false,
		},
		{
			name:     "one URL is nil",
			url1:     "https://example.com",
			url2:     "",
			expected: false,
		},
		{
			name:     "both URLs are nil",
			url1:     "",
			url2:     "",
			expected: false,
		},
		{
			name:     "single part domains",
			url1:     "https://localhost",
			url2:     "https://localhost",
			expected: false,
		},
		// Edge cases: public suffixes
		{
			name:     "co.uk - same registrable domain",
			url1:     "https://www.example.co.uk",
			url2:     "https://api.example.co.uk",
			expected: true,
		},
		{
			name:     "co.uk - different registrable domains",
			url1:     "https://example.co.uk",
			url2:     "https://other.co.uk",
			expected: false,
		},
		{
			name:     "com.au - same registrable domain",
			url1:     "https://www.example.com.au",
			url2:     "https://shop.example.com.au",
			expected: true,
		},
		{
			name:     "com.au - different registrable domains",
			url1:     "https://example.com.au",
			url2:     "https://other.com.au",
			expected: false,
		},
		{
			name:     "hosts with ports - same domain",
			url1:     "https://www.example.com:8080",
			url2:     "https://api.example.com:9090",
			expected: true,
		},
		{
			name:     "case insensitive domain comparison",
			url1:     "https://WWW.EXAMPLE.COM",
			url2:     "https://api.example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u1, u2 *url.URL
			if tt.url1 != "" {
				u1, _ = url.Parse(tt.url1)
			}
			if tt.url2 != "" {
				u2, _ = url.Parse(tt.url2)
			}
			result := AreRelatedHosts(u1, u2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "sort URLs alphabetically",
			input:    []string{"https://z.com", "https://a.com", "https://m.com"},
			expected: []string{"https://a.com", "https://m.com", "https://z.com"},
		},
		{
			name:     "already sorted",
			input:    []string{"https://a.com", "https://b.com", "https://c.com"},
			expected: []string{"https://a.com", "https://b.com", "https://c.com"},
		},
		{
			name:     "single URL",
			input:    []string{"https://example.com"},
			expected: []string{"https://example.com"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert strings to URLs
			urls := make([]*url.URL, len(tt.input))
			for i, u := range tt.input {
				urls[i], _ = url.Parse(u)
			}

			// Sort the URLs
			SortURLs(urls)

			// Convert back to strings for comparison
			result := make([]string, len(urls))
			for i, u := range urls {
				result[i] = u.String()
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortURLsWithNilEntries(t *testing.T) {
	// Test that nil entries are sorted to the end
	urls := []*url.URL{
		mustParse("https://z.com"),
		nil,
		mustParse("https://a.com"),
		nil,
		mustParse("https://m.com"),
	}

	SortURLs(urls)

	// Non-nil entries should be sorted, nils at end
	assert.Equal(t, "https://a.com", urls[0].String())
	assert.Equal(t, "https://m.com", urls[1].String())
	assert.Equal(t, "https://z.com", urls[2].String())
	assert.Nil(t, urls[3])
	assert.Nil(t, urls[4])
}

func TestResolveLink(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		link     string
		expected string
		valid    bool
	}{
		{
			name:     "absolute HTTPS URL",
			domain:   "example.com",
			link:     "https://example.com/page",
			expected: "https://example.com/page",
			valid:    true,
		},
		{
			name:     "absolute HTTP URL",
			domain:   "example.com",
			link:     "http://example.com/page",
			expected: "https://example.com/page",
			valid:    true,
		},
		{
			name:     "relative URL with leading slash",
			domain:   "example.com",
			link:     "/about",
			expected: "https://example.com/about",
			valid:    true,
		},
		{
			name:     "relative URL without leading slash",
			domain:   "example.com",
			link:     "about",
			expected: "https://example.com/about",
			valid:    true,
		},
		{
			name:   "invalid scheme",
			domain: "example.com",
			link:   "ftp://example.com/file",
			valid:  false,
		},
		{
			name:   "javascript URL",
			domain: "example.com",
			link:   "javascript:void(0)",
			valid:  false,
		},
		{
			name:   "mailto URL",
			domain: "example.com",
			link:   "mailto:test@example.com",
			valid:  false,
		},
		{
			name:     "URL with fragment",
			domain:   "example.com",
			link:     "https://example.com/page#section",
			expected: "https://example.com/page",
			valid:    true,
		},
		{
			name:     "domain with https prefix",
			domain:   "https://example.com",
			link:     "/page",
			expected: "https://example.com/page",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := ResolveLink(tt.domain, tt.link)
			assert.Equal(t, tt.valid, valid)
			if valid {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Example demonstrates basic URL normalization.
func ExampleNormalizeURL() {
	// Normalize a URL with query parameters and fragment
	url, _ := NormalizeURL("example.com/path?query=1#fragment")
	fmt.Println(url.String())

	// Convert http to https
	url, _ = NormalizeURL("http://example.com")
	fmt.Println(url.String())

	// Add https prefix when missing
	url, _ = NormalizeURL("example.com")
	fmt.Println(url.String())

	// Output:
	// https://example.com/path
	// https://example.com
	// https://example.com
}

// Example demonstrates resolving relative URLs against a base domain.
func ExampleResolveLink() {
	baseDomain := "example.com"

	// Resolve absolute path
	resolved, ok := ResolveLink(baseDomain, "/about")
	fmt.Printf("%s: %v\n", resolved, ok)

	// Resolve relative path
	resolved, ok = ResolveLink(baseDomain, "contact")
	fmt.Printf("%s: %v\n", resolved, ok)

	// Reject non-HTTP schemes
	resolved, ok = ResolveLink(baseDomain, "mailto:test@example.com")
	fmt.Printf("valid: %v\n", ok)

	// Output:
	// https://example.com/about: true
	// https://example.com/contact: true
	// valid: false
}

// Example demonstrates comparing URL hosts.
func ExampleAreSameHost() {
	url1, _ := url.Parse("https://example.com/page1")
	url2, _ := url.Parse("https://example.com/page2")
	url3, _ := url.Parse("https://sub.example.com/page")

	fmt.Println(AreSameHost(url1, url2))
	fmt.Println(AreSameHost(url1, url3))

	// Output:
	// true
	// false
}

// Example demonstrates checking if URLs share a common domain.
func ExampleAreRelatedHosts() {
	url1, _ := url.Parse("https://www.example.com")
	url2, _ := url.Parse("https://api.example.com")
	url3, _ := url.Parse("https://other.com")

	fmt.Println(AreRelatedHosts(url1, url2))
	fmt.Println(AreRelatedHosts(url1, url3))

	// Output:
	// true
	// false
}

// Example demonstrates sorting URLs alphabetically.
func ExampleSortURLs() {
	urls := []*url.URL{
		mustParse("https://z.com/page"),
		mustParse("https://a.com/page"),
		mustParse("https://m.com/page"),
	}

	SortURLs(urls)

	for _, u := range urls {
		fmt.Println(u.String())
	}

	// Output:
	// https://a.com/page
	// https://m.com/page
	// https://z.com/page
}

// mustParse is a helper function for examples.
func mustParse(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
