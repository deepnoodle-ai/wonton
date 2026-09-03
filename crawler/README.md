# crawler

Pluggable web crawler with configurable fetchers, parsers, and caching.

## Summary

The crawler package provides a concurrent web crawler that can fetch and parse web pages at scale. It supports a pluggable crawl frontier, adaptive per-host scheduling, pluggable fetchers for different domains, custom parsers, HTTP-aware caching, and flexible link-following behavior (same domain, related subdomains, or any domain). Rules can be configured with priority-based matching using exact, regex, glob, prefix, or suffix patterns.

## Fetching Is SSRF-Guarded by Default

A crawler follows links it did not choose, which makes it a natural SSRF
vector: a page can link to `http://169.254.169.254/` or to a host that
resolves inside your network. The crawler itself does not connect to anything
— it goes through whatever `fetch.Fetcher` you give it — and `fetch`'s default
clients refuse non-public addresses. So a crawler built on
`fetch.NewHTTPFetcher` inherits the guard, including for `robots.txt`.

The practical consequence is that crawling `localhost` or an intranet host
needs an explicit client:

```go
fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
    Client: &http.Client{Timeout: fetch.DefaultTimeout},
})
```

See the [fetch](../fetch/) and [httpguard](../httpguard/) READMEs for details.

## Usage Examples

### Basic Crawler

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    // Create a basic crawler
    c, err := crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        5,
        DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
        FollowBehavior: crawler.FollowSameDomain,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Crawl and process results
    urls := []string{"https://example.com"}
    err = c.Crawl(context.Background(), urls, func(ctx context.Context, result *crawler.Result) {
        if result.Error != nil {
            fmt.Printf("Error crawling %s: %v\n", result.URL, result.Error)
            return
        }
        fmt.Printf("Crawled: %s (found %d links)\n", result.URL, len(result.Links))
    })
    if err != nil {
        log.Fatal(err)
    }

    // Check statistics
    stats := c.GetStats()
    fmt.Printf("Processed: %d, Succeeded: %d, Failed: %d\n",
        stats.GetProcessed(), stats.GetSucceeded(), stats.GetFailed())
}
```

### With Custom Parser

```go
package main

import (
    "context"
    "fmt"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
    "github.com/deepnoodle-ai/wonton/htmlparse"
)

// Article represents parsed article data
type Article struct {
    Title   string
    Summary string
    Author  string
}

// ArticleParser extracts article data from HTML
type ArticleParser struct{}

func (p *ArticleParser) Parse(ctx context.Context, page *fetch.Response) (any, error) {
    parser := htmlparse.New(page.HTML)

    title := parser.GetMetaTag("og:title")
    summary := parser.GetMetaTag("og:description")
    author := parser.GetMetaTag("article:author")

    return &Article{
        Title:   title,
        Summary: summary,
        Author:  author,
    }, nil
}

func main() {
    // Create crawler with parser rule
    parserRule := crawler.NewParserRule(
        "blog.example.com",
        &ArticleParser{},
        crawler.WithParserPriority(10),
    )

    c, err := crawler.New(crawler.Options{
        MaxURLs:        50,
        Workers:        3,
        DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
        ParserRules:    []*crawler.ParserRule{parserRule},
        FollowBehavior: crawler.FollowSameDomain,
    })
    if err != nil {
        log.Fatal(err)
    }

    err = c.Crawl(context.Background(), []string{"https://blog.example.com"},
        func(ctx context.Context, result *crawler.Result) {
            if result.Parsed != nil {
                article := result.Parsed.(*Article)
                fmt.Printf("Article: %s by %s\n", article.Title, article.Author)
            }
        })
    if err != nil {
        log.Fatal(err)
    }
}
```

### With Multiple Domain Rules

```go
package main

import (
    "context"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    // Different parsers for different domains
    newsParser := &NewsParser{}
    blogParser := &BlogParser{}
    defaultParser := &GenericParser{}

    // Create rules with priorities
    parserRules := []*crawler.ParserRule{
        // Exact match - highest priority
        crawler.NewParserRule("news.example.com", newsParser,
            crawler.WithParserPriority(100)),

        // Glob pattern for all blog subdomains
        crawler.NewParserRule("*.blog.example.com", blogParser,
            crawler.WithParserMatchType(crawler.MatchGlob),
            crawler.WithParserPriority(50)),

        // Regex pattern for dated URLs
        crawler.NewParserRule(`/\d{4}/\d{2}/`, newsParser,
            crawler.WithParserMatchType(crawler.MatchRegex),
            crawler.WithParserPriority(25)),
    }

    c, err := crawler.New(crawler.Options{
        MaxURLs:        200,
        Workers:        10,
        DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
        DefaultParser:  defaultParser,
        ParserRules:    parserRules,
        FollowBehavior: crawler.FollowRelatedSubdomains,
    })
    if err != nil {
        log.Fatal(err)
    }

    urls := []string{
        "https://news.example.com",
        "https://tech.blog.example.com",
        "https://travel.blog.example.com",
    }

    c.Crawl(context.Background(), urls, processResult)
}
```

### With Caching

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/crawler/cache"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    // Create memory cache
    memCache := cache.NewInMemoryCache()

    options := crawler.Options{
        MaxURLs:        500,
        Workers:        5,
        Cache:          memCache,
        CacheTTL:       30 * time.Minute,
        Revalidate:     true,
        DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
        FollowBehavior: crawler.FollowSameDomain,
    }
    c, err := crawler.New(options)
    if err != nil {
        log.Fatal(err)
    }

    // First crawl - fetches from web
    c.Crawl(context.Background(), []string{"https://example.com"}, processResult)

    // Crawlers remember visited URLs, so use a new crawler for a later pass.
    // Fresh entries avoid the request; stale entries use ETag or
    // Last-Modified and reuse their body when the server returns 304.
    next, _ := crawler.New(options)
    next.Crawl(context.Background(), []string{"https://example.com"}, processResult)
}
```

`cache.InMemoryCache` implements both the original HTML-only `cache.Cache`
interface and `cache.ResponseCache`. Existing `cache.Cache` implementations
continue to work unchanged and retain their indefinite HTML caching behavior.
TTL and conditional revalidation are available when the supplied cache also
implements `cache.ResponseCache`. A zero `CacheTTL` means entries never expire.

### With Request Delay

```go
package main

import (
    "context"
    "time"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    // Add a per-host delay and limit each host to one in-flight request.
    // Other hosts can continue while this host is waiting.
    c, err := crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        3,
        RequestDelay:   2 * time.Second,
        HostPolicy: crawler.HostPolicy{
            MaxConcurrent: 1,
        },
        DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
        FollowBehavior: crawler.FollowSameDomain,
    })
    if err != nil {
        log.Fatal(err)
    }

    c.Crawl(context.Background(), []string{"https://example.com"}, processResult)
}
```

### With Adaptive Politeness

Adaptive mode requeues `429` and `503` responses, honors `Retry-After`, and
adjusts each host independently when its latency rises. Retries are bounded by
`RetryOptions.MaxAttempts` (three by default), and `MaxDelay` caps adaptive
backoff. Robots.txt and configured minimum delays are always respected.

```go
c, err := crawler.New(crawler.Options{
    Workers:        8,
    Adaptive:       true,
    DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
    HostPolicy: crawler.HostPolicy{
        MaxConcurrent: 2,
        MinDelay:      250 * time.Millisecond,
        MaxDelay:      30 * time.Second,
    },
})
if err != nil {
    log.Fatal(err)
}

if err := c.Crawl(context.Background(), seeds, processResult); err != nil {
    log.Fatal(err)
}
for _, host := range c.HostStats() {
    fmt.Printf("%s: %d requests, %d bytes, p95 %s, final delay %s\n",
        host.Host, host.Requests, host.Bytes, host.P95, host.FinalDelay)
}
```

`HostStats` covers page-network requests in the current or most recent crawl;
cache hits are excluded. `Errors` includes transport errors and HTTP statuses
of 400 or higher. `PeakRPS` is the largest rolling one-second request-start
count observed for the host.

### With Progress Reporting

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    c, err := crawler.New(crawler.Options{
        MaxURLs:              1000,
        Workers:              10,
        DefaultFetcher:       fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
        FollowBehavior:       crawler.FollowSameDomain,
        Logger:               logger,
        ShowProgress:         true,
        ShowProgressInterval: 10 * time.Second, // Report every 10 seconds
    })
    if err != nil {
        log.Fatal(err)
    }

    c.Crawl(context.Background(), []string{"https://example.com"}, processResult)
}
```

### Different Follow Behaviors

```go
package main

import (
    "context"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{})

    // Only follow links on the exact same domain
    sameDomain, _ := crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        5,
        DefaultFetcher: fetcher,
        FollowBehavior: crawler.FollowSameDomain, // Default
    })

    // Follow links on related subdomains (e.g., blog.example.com, api.example.com)
    relatedSubdomains, _ := crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        5,
        DefaultFetcher: fetcher,
        FollowBehavior: crawler.FollowRelatedSubdomains,
    })

    // Follow any link, regardless of domain
    anyDomain, _ := crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        5,
        DefaultFetcher: fetcher,
        FollowBehavior: crawler.FollowAny,
    })

    // Don't follow any links (just crawl specified URLs)
    noFollow, _ := crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        5,
        DefaultFetcher: fetcher,
        FollowBehavior: crawler.FollowNone,
    })
}
```

### Stopping a Crawler

```go
package main

import (
    "context"
    "time"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    c, _ := crawler.New(crawler.Options{
        MaxURLs:        10000,
        Workers:        10,
        DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
        FollowBehavior: crawler.FollowAny,
    })

    // Start crawling in background
    go c.Crawl(context.Background(), []string{"https://example.com"}, processResult)

    // Stop after 30 seconds
    time.Sleep(30 * time.Second)
    c.Stop()
}
```

### Custom Fetcher Rules

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func newCrawler(jsFetcher fetch.Fetcher) (*crawler.Crawler, error) {
    // jsFetcher is supplied by the application; any browser or remote fetching
    // service can participate by implementing fetch.Fetcher.
    httpFetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{})

    fetcherRules := []*crawler.FetcherRule{
        // Use JS fetcher for SPA sites
        crawler.NewFetcherRule("app.example.com", jsFetcher,
            crawler.WithFetcherPriority(100)),

        // Use regular HTTP for static sites
        crawler.NewFetcherRule("*.example.com", httpFetcher,
            crawler.WithFetcherMatchType(crawler.MatchGlob),
            crawler.WithFetcherPriority(50)),
    }

    return crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        5,
        DefaultFetcher: httpFetcher,
        FetcherRules:   fetcherRules,
        FollowBehavior: crawler.FollowSameDomain,
    })
}
```

### Custom Frontier

`MemoryFrontier` prioritizes higher scores, then shallower URLs, then insertion
order. `QueueSize` bounds pending items across both the frontier and the
crawler's per-host staging queues, applying backpressure instead of dropping
links. Provide another `Frontier` implementation for application-specific
storage or ordering; its own storage capacity remains implementation-defined,
while `QueueSize` still bounds scheduler staging. A provided frontier may be
preloaded; pass an empty seed list to `Crawl` to process only that existing
work.

```go
frontier := crawler.NewMemoryFrontier(1000)

c, err := crawler.New(crawler.Options{
    Workers:        8,
    Frontier:       frontier,
    DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
    HostPolicy: crawler.HostPolicy{
        MaxConcurrent: 2,
        MinDelay:      250 * time.Millisecond,
    },
})
```

## API Reference

### Types

#### Options

| Field | Type | Description |
|-------|------|-------------|
| `MaxURLs` | `int` | Maximum number of URLs to crawl (0 = unlimited) |
| `Workers` | `int` | Number of concurrent worker goroutines |
| `Cache` | `cache.Cache` | Optional HTML cache; typed implementations also preserve HTTP metadata and links |
| `CacheTTL` | `time.Duration` | Freshness lifetime for typed cache entries (0 = never expire) |
| `Revalidate` | `bool` | Revalidate stale typed entries with ETag or Last-Modified |
| `RequestDelay` | `time.Duration` | Minimum delay between requests to the same host |
| `Frontier` | `Frontier` | Pending-work store (default: `MemoryFrontier`) |
| `HostPolicy` | `HostPolicy` | Per-host concurrency and delay limits |
| `Adaptive` | `bool` | Enable bounded 429/503 retries and latency-sensitive per-host backoff |
| `KnownURLs` | `[]string` | Pre-populate list of known URLs |
| `ParserRules` | `[]*ParserRule` | Domain-specific parser rules |
| `DefaultParser` | `Parser` | Parser used when no rule matches |
| `FetcherRules` | `[]*FetcherRule` | Domain-specific fetcher rules |
| `DefaultFetcher` | `fetch.Fetcher` | Fetcher used when no rule matches |
| `FollowBehavior` | `FollowBehavior` | How to follow discovered links |
| `Logger` | `*slog.Logger` | Logger for crawler events |
| `ShowProgress` | `bool` | Enable periodic progress reporting |
| `ShowProgressInterval` | `time.Duration` | How often to report progress (default: 30s) |
| `QueueSize` | `int` | Shared pending capacity for the default frontier and scheduler staging; with custom frontiers, bounds scheduler staging (default: 10000) |

#### Result

| Field | Type | Description |
|-------|------|-------------|
| `URL` | `*url.URL` | The URL that was crawled |
| `Depth` | `int` | Link distance from the seed URL |
| `Referrer` | `string` | Page where the URL was discovered |
| `DiscoveredAt` | `time.Time` | Time the URL entered the frontier |
| `Parsed` | `any` | Parsed data from parser (if parser exists) |
| `Links` | `[]string` | Discovered links on the page |
| `Response` | `*fetch.Response` | Full fetch response with HTML and metadata |
| `Error` | `error` | Error encountered during crawl (if any) |

#### HostStats

| Field | Type | Description |
|-------|------|-------------|
| `Host` | `string` | Lowercase host name, including a non-default port |
| `Requests` | `int64` | Page-network requests started |
| `Bytes` | `int64` | Response body bytes observed |
| `Errors` | `int64` | Transport errors and HTTP responses of 400 or higher |
| `StatusCodes` | `map[int]int64` | Response count by HTTP status |
| `PeakRPS` | `float64` | Peak rolling one-second request-start count |
| `P50`, `P95` | `time.Duration` | Request latency percentiles |
| `FinalDelay` | `time.Duration` | Effective host delay after policy, robots, and adaptation |

#### FollowBehavior

| Constant | Description |
|----------|-------------|
| `FollowAny` | Follow all discovered links |
| `FollowSameDomain` | Only follow links on the same domain (default) |
| `FollowRelatedSubdomains` | Follow links on related subdomains |
| `FollowNone` | Don't follow any links |

#### MatchType

| Constant | Description |
|----------|-------------|
| `MatchExact` | Exact string match |
| `MatchRegex` | Regular expression match |
| `MatchGlob` | Glob pattern match (*, ?) |
| `MatchPrefix` | String prefix match |
| `MatchSuffix` | String suffix match |

### Functions

#### Crawler Creation

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `New(opts)` | Create a new crawler | `Options` | `*Crawler`, `error` |

#### Crawler Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `Crawl(ctx, urls, callback)` | Start crawling URLs | `context.Context`, `[]string`, `Callback` | `error` |
| `Stop()` | Stop the crawler | None | None |
| `GetStats()` | Get crawling statistics | None | `*CrawlerStats` |
| `HostStats()` | Get per-host network impact for the current or latest crawl | None | `[]HostStats` |
| `Pending()` | Get queued URLs that have not started | None | `int` |
| `AddParserRules(rules...)` | Add parser rules dynamically | `...*ParserRule` | `error` |
| `AddFetcherRules(rules...)` | Add fetcher rules dynamically | `...*FetcherRule` | `error` |

### Rule Functions

#### Parser Rules

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `NewParserRule(pattern, parser, opts...)` | Create parser rule | `string`, `Parser`, `...ParserRuleOption` | `*ParserRule` |
| `WithParserPriority(priority)` | Set rule priority | `int` | `ParserRuleOption` |
| `WithParserMatchType(matchType)` | Set match type | `MatchType` | `ParserRuleOption` |

#### Fetcher Rules

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `NewFetcherRule(pattern, fetcher, opts...)` | Create fetcher rule | `string`, `fetch.Fetcher`, `...FetcherRuleOption` | `*FetcherRule` |
| `WithFetcherPriority(priority)` | Set rule priority | `int` | `FetcherRuleOption` |
| `WithFetcherMatchType(matchType)` | Set match type | `MatchType` | `FetcherRuleOption` |

### Interfaces

#### ResponseCache

`cache.ResponseCache` augments `cache.Cache` with typed entries containing the
status, headers, body, extracted links, validators, and fetch time. Use
`cache.ResponseKey(url)` when seeding, inspecting, or deleting typed entries;
the key includes `cache.ResponseSchemaVersion`.

#### Parser

```go
type Parser interface {
    Parse(ctx context.Context, page *fetch.Response) (any, error)
}
```

Implement this interface to create custom parsers for extracting structured data from HTML.

#### Callback

```go
type Callback func(ctx context.Context, result *Result)
```

Called for each crawled page. Process the result and extract needed data.

### Statistics

| Method | Description | Returns |
|--------|-------------|---------|
| `GetProcessed()` | Number of URLs processed | `int64` |
| `GetSucceeded()` | Number of successful crawls | `int64` |
| `GetFailed()` | Number of failed crawls | `int64` |

## Related Packages

- **[fetch](../fetch/)** - HTTP page fetching used by the crawler
- **[httpguard](../httpguard/)** - The SSRF guard behind fetch's default clients
- **[htmlparse](../htmlparse/)** - HTML parsing for extracting data and links
- **[web](../web/)** - URL normalization and manipulation utilities
- **[retry](../retry/)** - Retry logic for failed requests
