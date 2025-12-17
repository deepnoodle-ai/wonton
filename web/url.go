// Package web provides URL manipulation, text normalization, and media detection
// utilities for web crawling and content processing.
//
// The package offers three main categories of functionality:
//
// URL Operations:
//   - NormalizeURL: Parse and standardize URLs (add https://, remove query params)
//   - ResolveLink: Resolve relative URLs against a base domain
//   - AreSameHost: Compare if two URLs have identical hosts
//   - AreRelatedHosts: Check if URLs share a common parent domain
//   - SortURLs: Sort URLs alphabetically by their string representation
//
// Text Processing:
//   - NormalizeText: Clean web text (trim, unescape HTML, remove non-printable chars)
//   - EndsWithPunctuation: Check if text ends with common punctuation marks
//
// Media Detection:
//   - IsMediaURL: Identify URLs pointing to media files
//   - MediaExtensions: Predefined set of common media file extensions
//
// This package is particularly useful when building web crawlers, content extractors,
// or any application that needs to process URLs and text from web pages.
package web

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// AreSameHost checks if two URLs have the same host value.
// Returns false if either URL is nil.
//
// This function performs an exact host comparison, meaning subdomains are considered
// different hosts. For example, "www.example.com" and "api.example.com" are not
// the same host. Use AreRelatedHosts if you need to check for shared parent domains.
//
// Example:
//
//	url1, _ := url.Parse("https://example.com/page1")
//	url2, _ := url.Parse("https://example.com/page2")
//	web.AreSameHost(url1, url2) // true
//
//	url3, _ := url.Parse("https://sub.example.com/page")
//	web.AreSameHost(url1, url3) // false
func AreSameHost(url1, url2 *url.URL) bool {
	return url1 != nil && url2 != nil && url1.Host == url2.Host
}

// AreRelatedHosts checks if two URLs share the same base domain (the last two
// parts of the hostname). Returns false if either URL is nil or has fewer than
// two domain parts.
//
// This function is useful for determining if URLs belong to the same website
// family, even if they use different subdomains.
//
// Example:
//
//	url1, _ := url.Parse("https://www.example.com")
//	url2, _ := url.Parse("https://api.example.com")
//	web.AreRelatedHosts(url1, url2) // true (both share "example.com")
//
//	url3, _ := url.Parse("https://example.org")
//	web.AreRelatedHosts(url1, url3) // false (different base domains)
func AreRelatedHosts(url1, url2 *url.URL) bool {
	if url1 == nil || url2 == nil {
		return false
	}
	parts1 := strings.Split(url1.Host, ".")
	parts2 := strings.Split(url2.Host, ".")

	// Get the base domain (last two parts)
	if len(parts1) < 2 || len(parts2) < 2 {
		return false
	}
	base1 := strings.Join(parts1[len(parts1)-2:], ".")
	base2 := strings.Join(parts2[len(parts2)-2:], ".")
	return base1 == base2
}

// NormalizeURL parses a URL string and returns a normalized URL.
//
// The following transformations are applied:
//   - Trim whitespace from the input
//   - Add https:// prefix if the URL has no scheme
//   - Convert http:// to https://
//   - Remove query parameters and URL fragments
//   - Remove trailing "/" if the path is only "/"
//
// This function returns an error if the input is empty, has an invalid scheme
// (anything other than http/https), or cannot be parsed as a valid URL.
//
// Example:
//
//	url, _ := web.NormalizeURL("example.com/path?q=1#frag")
//	fmt.Println(url.String()) // "https://example.com/path"
//
//	url, _ = web.NormalizeURL("http://example.com")
//	fmt.Println(url.String()) // "https://example.com"
func NormalizeURL(value string) (*url.URL, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("invalid empty url")
	}
	if !strings.HasPrefix(value, "http") {
		if strings.Contains(value, "://") {
			return nil, fmt.Errorf("invalid url: %s", value)
		}
		value = "https://" + value
	}
	if strings.HasPrefix(value, "http://") {
		value = "https://" + value[7:]
	}
	u, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("invalid url %q: %w", value, err)
	}
	u.ForceQuery = false
	u.RawQuery = ""
	u.Fragment = ""
	if u.Path == "/" {
		u.Path = ""
	}
	return u, nil
}

// SortURLs sorts a slice of URLs alphabetically by their string representation.
// The slice is sorted in place.
//
// Example:
//
//	urls := []*url.URL{
//	    mustParse("https://z.com"),
//	    mustParse("https://a.com"),
//	    mustParse("https://m.com"),
//	}
//	web.SortURLs(urls)
//	// urls is now ordered: a.com, m.com, z.com
func SortURLs(urls []*url.URL) {
	sort.Slice(urls, func(i, j int) bool {
		return urls[i].String() < urls[j].String()
	})
}

// ResolveLink resolves a relative or absolute URL against a base domain and returns
// the normalized result.
//
// For absolute URLs, this function validates the scheme (only http/https are accepted)
// and normalizes the URL. For relative URLs, it resolves them against the provided
// domain. URL fragments are always removed.
//
// Returns the resolved URL string and true if successful, or an empty string and
// false if the URL is invalid (e.g., unsupported scheme, parse error).
//
// The domain parameter can be specified with or without a scheme. If no scheme is
// provided, https:// is assumed.
//
// Example:
//
//	// Resolve relative URL
//	resolved, ok := web.ResolveLink("example.com", "/about")
//	// resolved: "https://example.com/about", ok: true
//
//	// Validate absolute URL
//	resolved, ok = web.ResolveLink("example.com", "https://other.com/page")
//	// resolved: "https://other.com/page", ok: true
//
//	// Reject non-http schemes
//	resolved, ok = web.ResolveLink("example.com", "ftp://files.com")
//	// resolved: "", ok: false
func ResolveLink(domain, value string) (string, bool) {
	// Parse the input URL
	parsedURL, err := url.Parse(value)
	if err != nil {
		return "", false
	}

	// Remove fragment
	parsedURL.Fragment = ""

	// Check if it's already absolute
	if parsedURL.IsAbs() {
		// Only accept HTTP/HTTPS schemes
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return "", false
		}
		// Normalize and return
		normalizedURL, err := NormalizeURL(parsedURL.String())
		if err != nil {
			return "", false
		}
		return normalizedURL.String(), true
	}

	// For relative URLs, we need to resolve against the domain
	// First, ensure domain has a scheme
	baseDomain := domain
	if !strings.HasPrefix(baseDomain, "http://") && !strings.HasPrefix(baseDomain, "https://") {
		baseDomain = "https://" + baseDomain
	}

	// Parse the base domain
	baseURL, err := url.Parse(baseDomain)
	if err != nil {
		return "", false
	}

	// Resolve the relative URL against the base
	resolvedURL := baseURL.ResolveReference(parsedURL)

	// Normalize and return
	normalizedURL, err := NormalizeURL(resolvedURL.String())
	if err != nil {
		return "", false
	}
	return normalizedURL.String(), true
}
