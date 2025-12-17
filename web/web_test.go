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

func TestRemoveNonPrintableChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text with tabs and newlines",
			input:    "Hello\tWorld\n",
			expected: "Hello\tWorld\n",
		},
		{
			name:     "text with non-printable chars",
			input:    "Hello\x00\x01World",
			expected: "Hello  World",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only non-printable chars",
			input:    "\x00\x01\x02",
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeNonPrintableChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text needing trimming",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "text with HTML entities",
			input:    "Hello &amp; World &lt;test&gt;",
			expected: "Hello & World <test>",
		},
		{
			name:     "text with special quotes",
			input:    `"Hello" 'World'`,
			expected: "\"Hello\" 'World'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n  ",
			expected: "",
		},
		{
			name:     "text with non-printable chars",
			input:    "Hello\x00World",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEndsWithPunctuation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "ends with period",
			input:    "Hello.",
			expected: true,
		},
		{
			name:     "ends with comma",
			input:    "Hello,",
			expected: true,
		},
		{
			name:     "ends with question mark",
			input:    "Hello?",
			expected: true,
		},
		{
			name:     "ends with exclamation",
			input:    "Hello!",
			expected: true,
		},
		{
			name:     "ends with quote",
			input:    "Hello\"",
			expected: true,
		},
		{
			name:     "ends with apostrophe",
			input:    "Hello'",
			expected: true,
		},
		{
			name:     "ends with letter",
			input:    "Hello",
			expected: false,
		},
		{
			name:     "ends with number",
			input:    "Hello123",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "single punctuation",
			input:    ".",
			expected: true,
		},
		{
			name:     "unicode characters",
			input:    "Hello世界.",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EndsWithPunctuation(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMediaURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "image file",
			url:      "https://example.com/image.jpg",
			expected: true,
		},
		{
			name:     "video file",
			url:      "https://example.com/video.mp4",
			expected: true,
		},
		{
			name:     "audio file",
			url:      "https://example.com/audio.mp3",
			expected: true,
		},
		{
			name:     "document file",
			url:      "https://example.com/doc.pdf",
			expected: true,
		},
		{
			name:     "uppercase extension",
			url:      "https://example.com/IMAGE.JPG",
			expected: true,
		},
		{
			name:     "html file",
			url:      "https://example.com/page.html",
			expected: false,
		},
		{
			name:     "no extension",
			url:      "https://example.com/page",
			expected: false,
		},
		{
			name:     "path with dot but no extension",
			url:      "https://example.com/path.with.dots/page",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			result := IsMediaURL(u)
			assert.Equal(t, tt.expected, result)
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

// Example demonstrates text normalization for web content.
func ExampleNormalizeText() {
	// Trim whitespace
	fmt.Println(NormalizeText("  Hello  "))

	// Unescape HTML entities
	fmt.Println(NormalizeText("Hello &amp; goodbye"))

	// Convert HTML tags (entities)
	fmt.Println(NormalizeText("&lt;div&gt;"))

	// Remove non-printable characters
	fmt.Println(NormalizeText("Hello\x00World"))

	// Output:
	// Hello
	// Hello & goodbye
	// <div>
	// Hello World
}

// Example demonstrates checking for punctuation at the end of strings.
func ExampleEndsWithPunctuation() {
	fmt.Println(EndsWithPunctuation("Hello."))
	fmt.Println(EndsWithPunctuation("Hello?"))
	fmt.Println(EndsWithPunctuation("Hello"))
	fmt.Println(EndsWithPunctuation(""))

	// Output:
	// true
	// true
	// false
	// false
}

// Example demonstrates detecting media files from URLs.
func ExampleIsMediaURL() {
	imageURL, _ := url.Parse("https://example.com/photo.jpg")
	fmt.Println(IsMediaURL(imageURL))

	videoURL, _ := url.Parse("https://example.com/video.mp4")
	fmt.Println(IsMediaURL(videoURL))

	pageURL, _ := url.Parse("https://example.com/page.html")
	fmt.Println(IsMediaURL(pageURL))

	// Output:
	// true
	// true
	// false
}

// mustParse is a helper function for examples.
func mustParse(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
