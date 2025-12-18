# crawler

Pluggable web crawler with configurable fetchers, parsers, and caching.

## Summary

The crawler package provides a concurrent web crawler that can fetch and parse web pages at scale. It supports pluggable fetchers for different domains, custom parsers for extracting structured data, optional caching to reduce redundant fetches, and flexible link-following behavior (same domain, related subdomains, or any domain). The crawler uses worker pools for concurrency, tracks statistics, and provides progress reporting. Rules can be configured with priority-based matching using exact, regex, glob, prefix, or suffix patterns.

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
        DefaultFetcher: fetch.NewHTTPFetcher(),
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
        DefaultFetcher: fetch.NewHTTPFetcher(),
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
        DefaultFetcher: fetch.NewHTTPFetcher(),
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

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/crawler/cache"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    // Create memory cache
    memCache := cache.NewMemoryCache(1000) // Max 1000 entries

    c, err := crawler.New(crawler.Options{
        MaxURLs:        500,
        Workers:        5,
        Cache:          memCache,
        DefaultFetcher: fetch.NewHTTPFetcher(),
        FollowBehavior: crawler.FollowSameDomain,
    })
    if err != nil {
        log.Fatal(err)
    }

    // First crawl - fetches from web
    c.Crawl(context.Background(), []string{"https://example.com"}, processResult)

    // Second crawl - uses cache
    c.Crawl(context.Background(), []string{"https://example.com"}, processResult)
}
```

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
    // Add delay between requests to be polite
    c, err := crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        3,
        RequestDelay:   2 * time.Second, // 2 second delay between requests
        DefaultFetcher: fetch.NewHTTPFetcher(),
        FollowBehavior: crawler.FollowSameDomain,
    })
    if err != nil {
        log.Fatal(err)
    }

    c.Crawl(context.Background(), []string{"https://example.com"}, processResult)
}
```

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
        DefaultFetcher:       fetch.NewHTTPFetcher(),
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
    fetcher := fetch.NewHTTPFetcher()

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
        DefaultFetcher: fetch.NewHTTPFetcher(),
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
    "context"

    "github.com/deepnoodle-ai/wonton/crawler"
    "github.com/deepnoodle-ai/wonton/fetch"
)

func main() {
    // Different fetchers for different domains
    httpFetcher := fetch.NewHTTPFetcher()
    jsFetcher := fetch.NewBrowserFetcher() // Hypothetical JS-capable fetcher

    fetcherRules := []*crawler.FetcherRule{
        // Use JS fetcher for SPA sites
        crawler.NewFetcherRule("app.example.com", jsFetcher,
            crawler.WithFetcherPriority(100)),

        // Use regular HTTP for static sites
        crawler.NewFetcherRule("*.example.com", httpFetcher,
            crawler.WithFetcherMatchType(crawler.MatchGlob),
            crawler.WithFetcherPriority(50)),
    }

    c, err := crawler.New(crawler.Options{
        MaxURLs:        100,
        Workers:        5,
        DefaultFetcher: httpFetcher,
        FetcherRules:   fetcherRules,
        FollowBehavior: crawler.FollowSameDomain,
    })
    if err != nil {
        log.Fatal(err)
    }

    c.Crawl(context.Background(), []string{"https://app.example.com"}, processResult)
}
```

## API Reference

### Types

#### Options

| Field | Type | Description |
|-------|------|-------------|
| `MaxURLs` | `int` | Maximum number of URLs to crawl (0 = unlimited) |
| `Workers` | `int` | Number of concurrent worker goroutines |
| `Cache` | `cache.Cache` | Optional cache for storing fetched HTML |
| `RequestDelay` | `time.Duration` | Delay between requests (per worker) |
| `KnownURLs` | `[]string` | Pre-populate list of known URLs |
| `ParserRules` | `[]*ParserRule` | Domain-specific parser rules |
| `DefaultParser` | `Parser` | Parser used when no rule matches |
| `FetcherRules` | `[]*FetcherRule` | Domain-specific fetcher rules |
| `DefaultFetcher` | `fetch.Fetcher` | Fetcher used when no rule matches |
| `FollowBehavior` | `FollowBehavior` | How to follow discovered links |
| `Logger` | `*slog.Logger` | Logger for crawler events |
| `ShowProgress` | `bool` | Enable periodic progress reporting |
| `ShowProgressInterval` | `time.Duration` | How often to report progress (default: 30s) |
| `QueueSize` | `int` | Size of URL queue (default: 10000) |

#### Result

| Field | Type | Description |
|-------|------|-------------|
| `URL` | `*url.URL` | The URL that was crawled |
| `Parsed` | `any` | Parsed data from parser (if parser exists) |
| `Links` | `[]string` | Discovered links on the page |
| `Response` | `*fetch.Response` | Full fetch response with HTML and metadata |
| `Error` | `error` | Error encountered during crawl (if any) |

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
- **[htmlparse](../htmlparse/)** - HTML parsing for extracting data and links
- **[web](../web/)** - URL normalization and manipulation utilities
- **[retry](../retry/)** - Retry logic for failed requests
