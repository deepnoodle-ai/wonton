// Package web provides URL manipulation, text normalization, and media detection
// utilities for web crawling and content processing.
package web

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// AreSameHost checks if two URLs have the same host value.
func AreSameHost(url1, url2 *url.URL) bool {
	return url1 != nil && url2 != nil && url1.Host == url2.Host
}

// AreRelatedHosts checks if two URLs are the same or are related by a common
// parent domain.
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

// NormalizeURL parses a URL string and returns a normalized URL. The following
// transformations are applied:
// - Trim whitespace
// - Convert http:// to https://
// - Add https:// prefix if missing
// - Remove any query parameters and URL fragments
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

// SortURLs sorts a slice of URLs by their string representation.
func SortURLs(urls []*url.URL) {
	sort.Slice(urls, func(i, j int) bool {
		return urls[i].String() < urls[j].String()
	})
}

// ResolveLink resolves a relative or absolute URL against a domain and returns
// the normalized result. Returns the resolved URL and true if successful, or
// an empty string and false if the URL is invalid.
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
