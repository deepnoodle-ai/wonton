# web — fetching, crawling, and content extraction

Packages: `fetch`, `crawler`, `httpguard`, `htmlparse`, `htmltomd`, `web`, `sse`.

## Safety first: httpguard

`fetch.DefaultHTTPClient` and `fetch.DefaultDownloadClient` are built by `httpguard`. They refuse
loopback, private, link-local, and other non-public addresses, ignore `HTTP_PROXY`/`HTTPS_PROXY`,
and validate every redirect hop — the defaults are safe for URLs a user or a model supplied.

That also means **`localhost` and intranet hosts fail by default**. To reach them, pass your own
client:

```go
fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
	Client: &http.Client{Timeout: 30 * time.Second},
})
```

Build a guarded client directly when you make requests yourself:

```go
client := httpguard.NewClient(
	httpguard.WithConnectTimeout(5*time.Second),
	httpguard.WithMaxRedirects(3),        // redirects are refused unless enabled
)
```

Other options: `WithDNSTimeout`, `WithTLSHandshakeTimeout`, `WithResponseHeaderTimeout`,
`WithHTTPRedirects`, `WithResolver`, `WithDialContext`, `WithAddressValidator`. The predicate
`httpguard.ValidatePublicIP(ip)` is exported for custom checks.

## fetch

`HTTPFetcher` is content-oriented and GET-only. Ask for the formats you want and it returns them
in one response.

```go
fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{Timeout: 30 * time.Second})

resp, err := fetcher.Fetch(ctx, &fetch.Request{
	URL:             "https://example.com",
	Formats:         []string{"html", "markdown", "links"},   // also "raw_html"
	OnlyMainContent: true,                                    // strips nav/header/footer
	ExcludeTags:     []string{"script", "style"},
})
// resp.URL (final), resp.StatusCode, resp.Headers, resp.HTML, resp.RawHTML,
// resp.Markdown, resp.Links, resp.Metadata.Title
```

`fetch.Download(ctx, url, &fetch.DownloadOptions{...})` saves a file. `fetch.IsRetryable(err)`
pairs well with the `retry` package. `fetch.NewMockFetcher()` stands in for tests. For HEAD
requests use `net/http` directly.

## crawler

```go
c, err := crawler.New(crawler.Options{
	Workers:        4,
	MaxURLs:        100,
	FollowBehavior: crawler.FollowSameDomain,   // FollowAny, FollowRelatedSubdomains, FollowNone
	DefaultFetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{}),
	RequestDelay:   500 * time.Millisecond,     // be polite
})

err = c.Crawl(ctx, []string{"https://example.com"},
	func(ctx context.Context, result *crawler.Result) {
		if result.Error != nil {
			return
		}
		fmt.Println(result.URL.String(), result.Response.Metadata.Title)
	})
```

The callback runs concurrently across workers — guard your own shared state. `c.Stop()` halts a
crawl, `c.GetStats()` reports progress. Per-domain behavior comes from `AddParserRules` and
`AddFetcherRules`.

## htmlparse and htmltomd

```go
doc, err := htmlparse.Parse(htmlContent)     // or ParseReader(r)
meta := doc.Metadata()                       // title, description, canonical, og:* …
links := doc.Links()                         // also Images(), FilteredLinks(filter), Branding()
text := doc.Text()                           // visible text
clean := doc.Transform(&htmlparse.TransformOptions{
	OnlyMainContent: true,
	Exclude:         []string{"nav", "footer"},
})
```

```go
md := htmltomd.Convert(htmlContent)
md = htmltomd.ConvertWithOptions(htmlContent, &htmltomd.Options{
	LinkStyle:    htmltomd.LinkStyleReferenced,
	HeadingStyle: htmltomd.HeadingStyleSetext,
})
```

## web

URL and text helpers that take real `*url.URL` values:

```go
u, err := web.NormalizeURL("example.com/path?q=1#frag")   // → https://example.com/path
u, err = web.NormalizeURL(raw, web.KeepQuery(), web.KeepHTTP())

resolved, err := web.ResolveLink(base, "/about")          // base is a *url.URL

web.AreSameHost(a, b)        // exact host match
web.AreRelatedHosts(a, b)    // same registrable domain, e.g. www. vs api.
web.IsBinaryURL(u)           // media/archive/document extensions
web.IsBinaryExtension(".pdf")
web.NormalizeText("  Hello &amp; goodbye  ")              // → "Hello & goodbye"
```

`web.NewExtensionSet(exts...)` builds a custom extension filter. `web.Searcher` is the interface
to implement for a search provider:

```go
type Searcher interface {
	Search(ctx context.Context, input *web.SearchInput) (*web.SearchOutput, error)
}
```

For downloads use `fetch.Download`, not `web` — `web` is URL and text logic only.

## sse

Server-Sent Events, the transport most LLM streaming APIs use.

```go
client := sse.NewClient("https://api.example.com/stream")
client.Headers.Set("Authorization", "Bearer "+token)

events, errs := client.Connect(ctx)
for event := range events {
	var msg response
	if err := event.JSON(&msg); err != nil {   // Event: Event, Data, ID, Retry
		continue
	}
}
if err := <-errs; err != nil {
	return err
}
```

Also `sse.Stream(reader, callback)` for an existing body, and `sse.ParseString`/`sse.ParseBytes`
for buffered payloads.

## Gotchas

- Don't reach for `crawler` when one page is enough — `fetch` is simpler and cheaper.
- `Formats` is opt-in: `resp.Markdown` is empty unless you asked for `"markdown"`.
- Crawl callbacks run synchronously on worker goroutines: keep them fast, and guard shared
  state yourself. `result.URL` is a `*url.URL`; check `result.Error` before the other fields.
- `web.ResolveLink` and `web.NormalizeURL` return `(*url.URL, error)` — not strings.
- Set `RequestDelay` and `MaxURLs` on every crawl of a site you do not own.
