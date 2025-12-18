package crawler

import (
	"bufio"
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/fetch"
)

// robotsTxtData holds parsed robots.txt rules for a host
type robotsTxtData struct {
	disallowRules []string
	allowRules    []string
	crawlDelay    time.Duration
	fetchedAt     time.Time
}

// robotsTxtCacheExpiry is how long to cache robots.txt data
const robotsTxtCacheExpiry = 24 * time.Hour

// fetchRobotsTxt fetches and parses robots.txt for the given URL's host
func (c *Crawler) fetchRobotsTxt(ctx context.Context, targetURL *url.URL) (*robotsTxtData, error) {
	host := targetURL.Scheme + "://" + targetURL.Host

	// Check cache first
	if cached, ok := c.robotsCache.Load(host); ok {
		data := cached.(*robotsTxtData)
		if time.Since(data.fetchedAt) < robotsTxtCacheExpiry {
			return data, nil
		}
		// Expired, fetch again
		c.robotsCache.Delete(host)
	}

	// Construct robots.txt URL
	robotsURL := host + "/robots.txt"

	// Get fetcher for this host
	fetcher, exists := c.getFetcher(targetURL.Hostname())
	if !exists {
		// No fetcher configured, allow all by default
		data := &robotsTxtData{fetchedAt: time.Now()}
		c.robotsCache.Store(host, data)
		return data, nil
	}

	// Fetch robots.txt
	req := &fetch.Request{
		URL:             robotsURL,
		Prettify:        false,
		OnlyMainContent: false,
	}

	response, err := fetcher.Fetch(ctx, req)
	if err != nil {
		// If fetch fails, allow all (permissive behavior)
		c.logger.Debug("failed to fetch robots.txt, allowing all",
			"host", host,
			"error", err.Error())
		data := &robotsTxtData{fetchedAt: time.Now()}
		c.robotsCache.Store(host, data)
		return data, nil
	}

	// Parse robots.txt
	data := parseRobotsTxt(response.HTML, c.robotsTxtUserAgent)
	data.fetchedAt = time.Now()

	// Cache the result
	c.robotsCache.Store(host, data)

	return data, nil
}

// parseRobotsTxt parses robots.txt content for the given user agent.
// It follows the standard robots.txt spec: if there's a specific user-agent
// match, only those rules apply. Otherwise, wildcard (*) rules apply.
func parseRobotsTxt(content string, userAgent string) *robotsTxtData {
	// First pass: collect rules for specific user agent and wildcard
	specificData := &robotsTxtData{}
	wildcardData := &robotsTxtData{}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentUserAgent string
	var isSpecificMatch bool
	var isWildcardMatch bool
	hasSpecificRules := false

	userAgentLower := strings.ToLower(userAgent)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse directive
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		directive := strings.TrimSpace(strings.ToLower(line[:colonIdx]))
		value := strings.TrimSpace(line[colonIdx+1:])

		switch directive {
		case "user-agent":
			currentUserAgent = strings.ToLower(value)
			isWildcardMatch = currentUserAgent == "*"
			isSpecificMatch = currentUserAgent == userAgentLower ||
				strings.Contains(userAgentLower, currentUserAgent)
			// Don't count wildcard as specific
			if isSpecificMatch && !isWildcardMatch {
				hasSpecificRules = true
			}

		case "disallow":
			if value == "" {
				continue
			}
			if isSpecificMatch && !isWildcardMatch {
				specificData.disallowRules = append(specificData.disallowRules, value)
			} else if isWildcardMatch {
				wildcardData.disallowRules = append(wildcardData.disallowRules, value)
			}

		case "allow":
			if value == "" {
				continue
			}
			if isSpecificMatch && !isWildcardMatch {
				specificData.allowRules = append(specificData.allowRules, value)
			} else if isWildcardMatch {
				wildcardData.allowRules = append(wildcardData.allowRules, value)
			}

		case "crawl-delay":
			var seconds float64
			if _, err := stringToFloat(value); err == nil {
				seconds, _ = stringToFloat(value)
				delay := time.Duration(seconds * float64(time.Second))
				if isSpecificMatch && !isWildcardMatch {
					specificData.crawlDelay = delay
				} else if isWildcardMatch {
					wildcardData.crawlDelay = delay
				}
			}
		}
	}

	// Return specific rules if we found any, otherwise return wildcard rules
	if hasSpecificRules {
		return specificData
	}
	return wildcardData
}

// stringToFloat converts a string to float64, returning 0 on error
func stringToFloat(s string) (float64, error) {
	var result float64
	_, err := scanFloat(s, &result)
	return result, err
}

// scanFloat is a simple float parser
func scanFloat(s string, result *float64) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	var val float64
	var decimal float64 = 0.1
	var negative bool
	var seenDot bool
	var count int

	for i, c := range s {
		if i == 0 && c == '-' {
			negative = true
			continue
		}
		if c == '.' {
			if seenDot {
				break
			}
			seenDot = true
			continue
		}
		if c < '0' || c > '9' {
			break
		}
		count++
		if seenDot {
			val += float64(c-'0') * decimal
			decimal *= 0.1
		} else {
			val = val*10 + float64(c-'0')
		}
	}

	if negative {
		val = -val
	}
	*result = val
	return count, nil
}

// isAllowedByRobots checks if a URL is allowed by robots.txt rules
func (c *Crawler) isAllowedByRobots(ctx context.Context, targetURL *url.URL) bool {
	if !c.respectRobotsTxt {
		return true
	}

	data, err := c.fetchRobotsTxt(ctx, targetURL)
	if err != nil {
		// On error, allow by default (permissive)
		return true
	}

	path := targetURL.Path
	if path == "" {
		path = "/"
	}
	if targetURL.RawQuery != "" {
		path += "?" + targetURL.RawQuery
	}

	// Check allow rules first (they take precedence for longer matches)
	for _, rule := range data.allowRules {
		if pathMatches(path, rule) {
			return true
		}
	}

	// Check disallow rules
	for _, rule := range data.disallowRules {
		if pathMatches(path, rule) {
			return false
		}
	}

	// Default: allow
	return true
}

// pathMatches checks if a path matches a robots.txt rule pattern
func pathMatches(path, rule string) bool {
	// Empty rule matches nothing
	if rule == "" {
		return false
	}

	// Handle wildcards
	if strings.Contains(rule, "*") {
		// Split by wildcards and check if all parts match in order
		parts := strings.Split(rule, "*")
		pos := 0
		for _, part := range parts {
			if part == "" {
				continue
			}
			idx := strings.Index(path[pos:], part)
			if idx == -1 {
				return false
			}
			pos += idx + len(part)
		}
		return true
	}

	// Handle end-of-string anchor
	if strings.HasSuffix(rule, "$") {
		rule = strings.TrimSuffix(rule, "$")
		return path == rule
	}

	// Standard prefix matching
	return strings.HasPrefix(path, rule)
}

// getRobotsCrawlDelay returns the crawl delay from robots.txt, or 0 if not set
func (c *Crawler) getRobotsCrawlDelay(ctx context.Context, targetURL *url.URL) time.Duration {
	if !c.respectRobotsTxt {
		return 0
	}

	data, err := c.fetchRobotsTxt(ctx, targetURL)
	if err != nil {
		return 0
	}

	return data.crawlDelay
}
