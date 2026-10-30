package web

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// NormalizeOption configures the behavior of [NormalizeURL] and
// [ResolveLink].
type NormalizeOption func(*normalizeConfig)

type normalizeConfig struct {
	keepQuery bool
	keepHTTP  bool
}

// KeepQuery preserves URL query parameters, which are removed by default.
func KeepQuery() NormalizeOption {
	return func(c *normalizeConfig) { c.keepQuery = true }
}

// KeepHTTP preserves the http scheme instead of upgrading it to https.
// Inputs without a scheme still default to https.
func KeepHTTP() NormalizeOption {
	return func(c *normalizeConfig) { c.keepHTTP = true }
}

func applyOptions(opts []NormalizeOption) normalizeConfig {
	var cfg normalizeConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// AreSameHost checks if two URLs have the same hostname. Returns false if
// either URL is nil.
//
// The comparison is case-insensitive and ignores ports, but subdomains are
// considered different hosts: "www.example.com" and "api.example.com" do not
// match. Use AreRelatedHosts to check for a shared parent domain.
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
	if url1 == nil || url2 == nil {
		return false
	}
	return strings.EqualFold(url1.Hostname(), url2.Hostname())
}

// AreRelatedHosts checks if two URLs share the same registrable domain
// (effective TLD + 1). Returns false if either URL is nil or cannot have its
// registrable domain determined.
//
// This function uses the Public Suffix List to correctly handle multi-part
// TLDs like "co.uk", "com.au", etc. For example, "example.co.uk" and
// "other.co.uk" are NOT related because they have different registrable
// domains.
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
//
//	url4, _ := url.Parse("https://foo.example.co.uk")
//	url5, _ := url.Parse("https://bar.example.co.uk")
//	web.AreRelatedHosts(url4, url5) // true (both share "example.co.uk")
func AreRelatedHosts(url1, url2 *url.URL) bool {
	if url1 == nil || url2 == nil {
		return false
	}

	host1 := url1.Hostname()
	host2 := url2.Hostname()

	// Get the registrable domain (eTLD+1) for each host
	domain1, err1 := publicsuffix.EffectiveTLDPlusOne(host1)
	domain2, err2 := publicsuffix.EffectiveTLDPlusOne(host2)

	// If either fails (e.g., localhost, IP addresses, invalid domains), return false
	if err1 != nil || err2 != nil {
		return false
	}

	return strings.EqualFold(domain1, domain2)
}

// NormalizeURL parses a URL string and returns a canonical URL suitable for
// comparison and deduplication.
//
// The following transformations are applied by default:
//   - Trim whitespace from the input
//   - Add https:// if the URL has no scheme (including host:port inputs
//     like "localhost:3000")
//   - Convert http:// to https:// (disable with [KeepHTTP])
//   - Lowercase the host
//   - Remove userinfo (credentials embedded in the URL)
//   - Remove default ports (:443 for https, :80 for http)
//   - Remove query parameters (disable with [KeepQuery]) and fragments
//   - Resolve dot segments in the path ("/a/../b" becomes "/b")
//   - Remove trailing "/" if the path is only "/"
//
// This function returns an error if the input is empty, has a non-http(s)
// scheme (mailto:, ftp:, javascript:, etc.), or cannot be parsed as a valid
// URL.
//
// Example:
//
//	u, _ := web.NormalizeURL("example.com/path?q=1#frag")
//	fmt.Println(u.String()) // "https://example.com/path"
//
//	u, _ = web.NormalizeURL("http://example.com", web.KeepHTTP())
//	fmt.Println(u.String()) // "http://example.com"
func NormalizeURL(value string, opts ...NormalizeOption) (*url.URL, error) {
	cfg := applyOptions(opts)

	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("invalid empty url")
	}

	u, err := url.Parse(value)
	if err != nil {
		// Inputs like "192.168.1.1:8080/admin" fail to parse because the
		// port reads as a colon in the first path segment. Retry with an
		// explicit scheme unless the input claimed one.
		if strings.Contains(value, "://") {
			return nil, fmt.Errorf("invalid url %q: %w", value, err)
		}
		u, err = url.Parse("https://" + value)
		if err != nil {
			return nil, fmt.Errorf("invalid url %q: %w", value, err)
		}
	}

	switch u.Scheme {
	case "":
		if u.Host != "" {
			// Protocol-relative URL (//example.com/path): just set the scheme
			u.Scheme = "https"
		} else {
			// Plain hostname like "example.com": add https:// and re-parse
			u, err = url.Parse("https://" + value)
			if err != nil {
				return nil, fmt.Errorf("invalid url %q: %w", value, err)
			}
		}
	case "http", "https":
		// canonicalize upgrades http after it strips the default port.
	default:
		// Inputs like "localhost:3000" parse as scheme "localhost" with
		// opaque "3000". An opaque that is only a valid port (optionally
		// followed by a path, query, or fragment) means the "scheme" was a
		// hostname. Known non-host schemes are always rejected, since
		// "tel:911" and "javascript:0" also look like host:port.
		if !looksLikeHostPort(u) {
			return nil, fmt.Errorf("invalid url scheme %q: %s", u.Scheme, value)
		}
		u, err = url.Parse("https://" + value)
		if err != nil {
			return nil, fmt.Errorf("invalid url %q: %w", value, err)
		}
	}

	return canonicalize(u, cfg)
}

// ResolveLink resolves a link found on a page against that page's URL and
// returns the canonical result. The base URL must be the URL of the page the
// link appeared on — relative links resolve against the page path, per RFC
// 3986. The same normalization as [NormalizeURL] is applied to the result,
// honoring any options.
//
// Absolute links are validated (only http and https are accepted) and
// normalized. Fragments are always removed.
//
// Returns an error if base is nil or not http(s), or if the link cannot be
// parsed or uses an unsupported scheme (mailto:, javascript:, etc.).
//
// Example:
//
//	base, _ := url.Parse("https://example.com/blog/post")
//	u, _ := web.ResolveLink(base, "../about")
//	fmt.Println(u.String()) // "https://example.com/about"
//
//	u, _ = web.ResolveLink(base, "https://other.com/page")
//	fmt.Println(u.String()) // "https://other.com/page"
//
//	_, err := web.ResolveLink(base, "mailto:test@example.com")
//	fmt.Println(err != nil) // true
func ResolveLink(base *url.URL, href string, opts ...NormalizeOption) (*url.URL, error) {
	if base == nil {
		return nil, errors.New("nil base url")
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("base url scheme must be http or https: %s", base)
	}

	ref, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return nil, fmt.Errorf("invalid link %q: %w", href, err)
	}
	if ref.IsAbs() && ref.Scheme != "http" && ref.Scheme != "https" {
		return nil, fmt.Errorf("unsupported link scheme %q: %s", ref.Scheme, href)
	}

	cfg := applyOptions(opts)
	resolved := base.ResolveReference(ref)
	return canonicalize(resolved, cfg)
}

// nonHostSchemes are URI schemes whose opaque part can start with digits but
// never names a host:port.
var nonHostSchemes = map[string]bool{
	"about": true, "blob": true, "callto": true, "data": true, "fax": true,
	"file": true, "geo": true, "intent": true, "javascript": true,
	"magnet": true, "mailto": true, "news": true, "sip": true, "sips": true,
	"skype": true, "sms": true, "tel": true, "urn": true, "vbscript": true,
	"xmpp": true,
}

// looksLikeHostPort reports whether a URL parsed with a non-http scheme is
// really a schemeless "host:port[/path]" input, such as "localhost:3000".
func looksLikeHostPort(u *url.URL) bool {
	if nonHostSchemes[u.Scheme] || u.Opaque == "" {
		return false
	}
	port := u.Opaque
	if i := strings.IndexAny(port, "/?#"); i >= 0 {
		port = port[:i]
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return false
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// canonicalize applies the shared normalization transformations to a parsed
// http(s) URL. The URL is modified in place and returned.
func canonicalize(u *url.URL, cfg normalizeConfig) (*url.URL, error) {
	// A hostname of only dots comes from a relative path such as "../about"
	// read as a bare hostname.
	if u.Hostname() == "" {
		return nil, fmt.Errorf("invalid url missing hostname: %s", u)
	}
	if strings.Trim(u.Hostname(), ".") == "" {
		return nil, fmt.Errorf("invalid url: hostname is only dots: %s", u)
	}

	// Credentials embedded in a URL are a fetch concern, not part of the
	// resource's identity; carrying them through canonicalization risks
	// leaking them into logs and stored URL sets.
	u.User = nil

	// Hostnames are case-insensitive; lowercase for stable comparisons and
	// deduplication. (u.Host includes the port but not userinfo.)
	u.Host = strings.ToLower(u.Host)

	// Default ports are redundant: example.com:443 and example.com are the
	// same resource over https. An http URL that will be upgraded drops
	// either default port, so "http://host:80" and "http://host:443" both
	// become "https://host".
	upgrade := u.Scheme == "http" && !cfg.keepHTTP
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (port == "443" && (u.Scheme == "https" || upgrade)) {
		host := u.Hostname()
		if strings.Contains(host, ":") {
			host = "[" + host + "]" // restore IPv6 brackets stripped by Hostname
		}
		u.Host = host
	}
	if upgrade {
		u.Scheme = "https"
	}

	if !cfg.keepQuery {
		u.ForceQuery = false
		u.RawQuery = ""
	}
	u.Fragment = ""
	u.RawFragment = ""

	// Resolve dot segments so "/a/../b" and "/b" compare equal. A path with
	// its own encoding (RawPath set) is cleaned in escaped form, so an
	// escaped slash such as %2F stays part of its segment.
	if u.RawPath != "" {
		raw := cleanPath(decodeDotSegments(u.EscapedPath()))
		if p, err := url.PathUnescape(raw); err == nil {
			u.Path, u.RawPath = p, raw
		}
	} else if u.Path != "" {
		u.Path = cleanPath(u.Path)
	}
	if u.Path == "/" {
		u.Path = ""
	}
	return u, nil
}

// decodeDotSegments decodes escaped dot segments such as "%2e" and "%2E%2e"
// in an escaped path, so cleanPath resolves them like "." and "..".
func decodeDotSegments(p string) string {
	segments := strings.Split(p, "/")
	for i, segment := range segments {
		switch strings.ToLower(segment) {
		case "%2e":
			segments[i] = "."
		case "%2e%2e", ".%2e", "%2e.":
			segments[i] = ".."
		}
	}
	return strings.Join(segments, "/")
}

// cleanPath resolves dot segments while preserving a trailing slash, which
// path.Clean would drop ("/docs/" and "/docs" may be different resources).
func cleanPath(p string) string {
	cleaned := path.Clean(p)
	if cleaned == "." {
		return ""
	}
	// A final "." or ".." segment names a directory, as in RFC 3986:
	// "/a/b/.." resolves to "/a/".
	endsInDir := strings.HasSuffix(p, "/") || strings.HasSuffix(p, "/.") || strings.HasSuffix(p, "/..")
	if endsInDir && !strings.HasSuffix(cleaned, "/") {
		cleaned += "/"
	}
	return cleaned
}
