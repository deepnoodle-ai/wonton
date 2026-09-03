package crawler

import (
	"bufio"
	"context"
	"net/url"
	"strconv"
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
			// A malformed bare User-agent directive must not match every agent:
			// strings.Contains(s, "") is otherwise always true.
			isSpecificMatch = currentUserAgent != "" &&
				(currentUserAgent == userAgentLower || strings.Contains(userAgentLower, currentUserAgent))
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
			if seconds, err := strconv.ParseFloat(value, 64); err == nil && seconds >= 0 {
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

	// Per the robots.txt standard, the most specific (longest) matching rule
	// wins. If an Allow and Disallow rule match with equal specificity, the
	// Allow rule wins.
	bestAllow, bestDisallow := -1, -1
	for _, rule := range data.allowRules {
		if len(rule) > bestAllow && pathMatches(path, rule) {
			bestAllow = len(rule)
		}
	}
	for _, rule := range data.disallowRules {
		if len(rule) > bestDisallow && pathMatches(path, rule) {
			bestDisallow = len(rule)
		}
	}
	if bestDisallow == -1 {
		return true
	}
	return bestAllow >= bestDisallow
}

// pathMatches checks if a path matches a robots.txt rule pattern. Rules are
// anchored at the start of the path, may contain * wildcards, and may end
// with $ to anchor the match at the end of the path.
func pathMatches(path, rule string) bool {
	// Empty rule matches nothing
	if rule == "" {
		return false
	}

	endAnchor := strings.HasSuffix(rule, "$")
	if endAnchor {
		rule = rule[:len(rule)-1]
	}

	parts := strings.Split(rule, "*")

	// No wildcards: simple prefix (or exact, with $) matching
	if len(parts) == 1 {
		if endAnchor {
			return path == rule
		}
		return strings.HasPrefix(path, rule)
	}

	// The first segment must match at the start of the path
	if !strings.HasPrefix(path, parts[0]) {
		return false
	}
	pos := len(parts[0])

	// With an end anchor, the final segment must match at the end of the path
	end := len(path)
	if endAnchor {
		last := parts[len(parts)-1]
		if len(path)-pos < len(last) || !strings.HasSuffix(path, last) {
			return false
		}
		end = len(path) - len(last)
		parts = parts[:len(parts)-1]
	}

	// Remaining segments must appear in order between pos and end
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		idx := strings.Index(path[pos:end], part)
		if idx == -1 {
			return false
		}
		pos += idx + len(part)
	}
	return true
}

// cachedRobotsCrawlDelay returns the Crawl-delay for the URL's host if
// robots.txt data is already cached, or 0 otherwise. It never triggers a
// robots.txt fetch.
func (c *Crawler) cachedRobotsCrawlDelay(targetURL *url.URL) time.Duration {
	host := targetURL.Scheme + "://" + targetURL.Host
	if cached, ok := c.robotsCache.Load(host); ok {
		return cached.(*robotsTxtData).crawlDelay
	}
	return 0
}
