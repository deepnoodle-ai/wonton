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
		opts        []NormalizeOption
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
			name:     "uppercase host lowercased",
			input:    "https://EXAMPLE.com/Path",
			expected: "https://example.com/Path",
		},
		{
			name:     "mixed case host with port lowercased",
			input:    "https://Example.COM:8080/page",
			expected: "https://example.com:8080/page",
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
		// Bare host:port inputs (parse as scheme:opaque without help)
		{
			name:     "bare hostname with port",
			input:    "example.com:8080/path",
			expected: "https://example.com:8080/path",
		},
		{
			name:     "localhost with port",
			input:    "localhost:3000",
			expected: "https://localhost:3000",
		},
		{
			name:     "IP address with port",
			input:    "192.168.1.1:8080/admin",
			expected: "https://192.168.1.1:8080/admin",
		},
		// Canonicalization: userinfo, default ports, dot segments
		{
			name:     "userinfo removed",
			input:    "https://user:pass@example.com/secret",
			expected: "https://example.com/secret",
		},
		{
			name:     "default https port removed",
			input:    "https://example.com:443/path",
			expected: "https://example.com/path",
		},
		{
			name:     "default http port removed after upgrade is not assumed",
			input:    "http://example.com:80/path",
			expected: "https://example.com:80/path",
		},
		{
			name:     "default http port removed with KeepHTTP",
			input:    "http://example.com:80/path",
			opts:     []NormalizeOption{KeepHTTP()},
			expected: "http://example.com/path",
		},
		{
			name:     "dot segments resolved",
			input:    "https://example.com/a/../b/./c",
			expected: "https://example.com/b/c",
		},
		{
			name:     "trailing slash preserved when cleaning",
			input:    "https://example.com/a/../docs/",
			expected: "https://example.com/docs/",
		},
		// Options
		{
			name:     "KeepQuery preserves query parameters",
			input:    "https://example.com/watch?v=abc123",
			opts:     []NormalizeOption{KeepQuery()},
			expected: "https://example.com/watch?v=abc123",
		},
		{
			name:     "KeepQuery still removes fragment",
			input:    "https://example.com/page?a=1#section",
			opts:     []NormalizeOption{KeepQuery()},
			expected: "https://example.com/page?a=1",
		},
		{
			name:     "KeepHTTP preserves http scheme",
			input:    "http://example.com/page",
			opts:     []NormalizeOption{KeepHTTP()},
			expected: "http://example.com/page",
		},
		{
			name:     "KeepHTTP does not affect schemeless input",
			input:    "example.com/page",
			opts:     []NormalizeOption{KeepHTTP()},
			expected: "https://example.com/page",
		},
		{
			name:     "combined options",
			input:    "http://example.com/search?q=go",
			opts:     []NormalizeOption{KeepHTTP(), KeepQuery()},
			expected: "http://example.com/search?q=go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeURL(tt.input, tt.opts...)
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

func TestResolveLink(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		link        string
		opts        []NormalizeOption
		expected    string
		expectError bool
	}{
		{
			name:     "absolute HTTPS URL",
			base:     "https://example.com",
			link:     "https://example.com/page",
			expected: "https://example.com/page",
		},
		{
			name:     "absolute HTTP URL upgraded",
			base:     "https://example.com",
			link:     "http://example.com/page",
			expected: "https://example.com/page",
		},
		{
			name:     "relative URL with leading slash",
			base:     "https://example.com",
			link:     "/about",
			expected: "https://example.com/about",
		},
		{
			name:     "relative URL without leading slash",
			base:     "https://example.com",
			link:     "about",
			expected: "https://example.com/about",
		},
		// Relative links resolve against the page URL per RFC 3986: the last
		// path segment of the base is dropped unless it ends with "/".
		{
			name:     "relative link against page URL",
			base:     "https://example.com/blog/post",
			link:     "other-post",
			expected: "https://example.com/blog/other-post",
		},
		{
			name:     "relative link against directory URL",
			base:     "https://example.com/docs/guide/",
			link:     "intro",
			expected: "https://example.com/docs/guide/intro",
		},
		{
			name:     "parent directory link",
			base:     "https://example.com/blog/post",
			link:     "../about",
			expected: "https://example.com/about",
		},
		{
			name:     "fragment-only link resolves to page",
			base:     "https://example.com/blog/post",
			link:     "#section",
			expected: "https://example.com/blog/post",
		},
		{
			name:     "protocol-relative link uses base scheme",
			base:     "https://example.com/page",
			link:     "//cdn.example.com/lib",
			expected: "https://cdn.example.com/lib",
		},
		{
			name:        "invalid scheme",
			base:        "https://example.com",
			link:        "ftp://example.com/file",
			expectError: true,
		},
		{
			name:        "javascript URL",
			base:        "https://example.com",
			link:        "javascript:void(0)",
			expectError: true,
		},
		{
			name:        "mailto URL",
			base:        "https://example.com",
			link:        "mailto:test@example.com",
			expectError: true,
		},
		{
			name:     "URL with fragment",
			base:     "https://example.com",
			link:     "https://example.com/page#section",
			expected: "https://example.com/page",
		},
		{
			name:     "query removed by default",
			base:     "https://example.com",
			link:     "/search?q=go",
			expected: "https://example.com/search",
		},
		{
			name:     "KeepQuery preserves link query",
			base:     "https://example.com",
			link:     "/search?q=go",
			opts:     []NormalizeOption{KeepQuery()},
			expected: "https://example.com/search?q=go",
		},
		{
			name:     "http base preserved with KeepHTTP",
			base:     "http://example.com/page",
			link:     "/about",
			opts:     []NormalizeOption{KeepHTTP()},
			expected: "http://example.com/about",
		},
		{
			name:     "http base upgraded by default",
			base:     "http://example.com/page",
			link:     "/about",
			expected: "https://example.com/about",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, err := url.Parse(tt.base)
			assert.NoError(t, err)
			result, err := ResolveLink(base, tt.link, tt.opts...)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

func TestResolveLinkInvalidBase(t *testing.T) {
	_, err := ResolveLink(nil, "/about")
	assert.Error(t, err)

	ftpBase, _ := url.Parse("ftp://example.com")
	_, err = ResolveLink(ftpBase, "/about")
	assert.Error(t, err)
}

// Example demonstrates basic URL normalization.
func ExampleNormalizeURL() {
	// Normalize a URL with query parameters and fragment
	u, _ := NormalizeURL("example.com/path?query=1#fragment")
	fmt.Println(u.String())

	// Convert http to https
	u, _ = NormalizeURL("http://example.com")
	fmt.Println(u.String())

	// Preserve query parameters when they matter
	u, _ = NormalizeURL("example.com/watch?v=abc123", KeepQuery())
	fmt.Println(u.String())

	// Output:
	// https://example.com/path
	// https://example.com
	// https://example.com/watch?v=abc123
}

// Example demonstrates resolving links against the page they appear on.
func ExampleResolveLink() {
	page, _ := url.Parse("https://example.com/blog/post")

	// Relative links resolve against the page URL
	u, _ := ResolveLink(page, "../about")
	fmt.Println(u.String())

	// Absolute links are validated and normalized
	u, _ = ResolveLink(page, "https://other.com/page")
	fmt.Println(u.String())

	// Non-HTTP schemes are rejected
	_, err := ResolveLink(page, "mailto:test@example.com")
	fmt.Println(err != nil)

	// Output:
	// https://example.com/about
	// https://other.com/page
	// true
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
