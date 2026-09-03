# crawler: ideas

Scratch list of directions for this package. Nothing here is committed to —
it's a menu, roughly ordered by how much it unlocks.

Current status: the in-memory frontier, per-host scheduler, adaptive
politeness/host statistics, HTTP-semantics cache, and the depth/referrer
metadata portion of the crawl graph are implemented. Durable frontiers and
graph export remain roadmap work.

## Observations on the current implementation

The crawler knows about *URLs*, but not about *hosts*, *time*, *content*, or
*the graph*. Most gaps below point the same direction.

- `enqueue`'s `default:` branch silently drops URLs when the queue fills.
- Politeness is per-worker, not per-host: `FollowAny` with 10 workers can
  hammer one server, while a single slow host's crawl-delay penalizes every
  other host.
- The cache stores only HTML, so status, headers, `ETag` and links are
  discarded and re-derived by re-parsing. No TTL, no conditional
  revalidation, no revisit policy.
- `Sitemap:` in robots.txt is parsed past and ignored.
- `<meta name="robots">`, `X-Robots-Tag`, and `<link rel="canonical">` are
  unhandled.
- No depth, no referrer, no link graph. A `Result` can't say how it was found.
- `web.IsBinaryURL` exists and isn't used; a large PDF is fetched as HTML.
- A parse error increments `Failed` even when the fetch succeeded.
- `parseRobotsTxt`: a malformed bare `User-agent:` line yields an empty value,
  and `strings.Contains(ua, "")` is true, so it matches every agent.
- `processedURLs` and `robotsCache` are unbounded `sync.Map`s.
- The README advertises `fetch.NewBrowserFetcher()`, which does not exist.

Reverse test — how would you make this crawler maximally rude and useless?
Drop discovered URLs silently. Point every worker at one host. Download
500MB binaries as HTML. Re-download unchanged pages forever. Lose all
progress on Ctrl-C. Follow infinite calendar links until OOM. Every one of
those is currently reachable, which is the real priority ordering.

## Structure

1. **`Frontier` as a pluggable interface (implemented).** `Push/Pop/Len/Close`. In-memory
   priority heap by default; BoltDB/SQLite for resumable crawls; Redis for
   distributed. Removes the silent drop, obsoletes `KnownURLs`, and makes
   most ideas below cheap.
2. **Per-host scheduler (implemented).** Host-keyed sub-queues with per-host concurrency,
   delay, and token bucket. Workers pull the next *eligible* host. Biggest
   behavioral-correctness win in the list.
3. **Adaptive politeness (implemented).** Track per-host latency and 429/503 + `Retry-After`;
   speed up on healthy hosts, back off on struggling ones. Emit an impact
   report: requests, bytes, peak RPS per host.
4. **Depth, referrer, and the crawl graph (metadata implemented).** `Result.Depth`,
   `Result.Referrer`, `Result.DiscoveredAt`, plus link-graph export
   (DOT/JSON), orphan detection, broken-link report, PageRank.
5. **HTTP-semantics cache (implemented).** Store status, headers, body, and extracted links
   under a schema-versioned key. Revalidate with `If-None-Match` /
   `If-Modified-Since`. Recrawling a docs site becomes nearly free.
6. **Sitemap ingestion.** Follow `Sitemap:` from robots.txt, recurse sitemap
   indexes, handle gzip, use `<lastmod>` for incremental recrawls.

## Content intelligence

7. **Dedup in three tiers.** Exact content hash, then `rel=canonical`, then
   near-duplicate detection (simhash over shingles).
8. **Trap detection.** Query-param explosion, infinite calendars, session IDs
   in paths, repeating path segments (`/a/b/a/b/a/b`), soft-404s. Quarantine
   the pattern and report it rather than dying quietly.
9. **Focused crawling via a `Scorer` interface.** Score frontier URLs by
   anchor text, URL shape, and parent relevance; crawl best-first. Extreme
   version: an embedding or LLM scorer — "crawl for everything about pricing,
   stop when the score decays."
10. **Content-type gating.** Use `web.IsBinaryURL`, HEAD/Range probes, and
    `Content-Length` caps before committing to a fetch.

## Wonton-native superpowers

11. **LLM corpus mode.** `htmltomd` + main-content extraction + dedup, to
    crawl a docs site into a clean markdown bundle with front matter and a
    generated `llms.txt`. Matches the module's stated mission.
12. **Live TUI dashboard.** Per-host throughput sparklines, live frontier
    depth, status-code histogram, in-flight requests, top errors, with keys
    to pause, throttle, or blacklist a host mid-crawl. Record with
    `termsession`, export via `gif`.
13. **`wonton crawl` CLI.** Dogfoods `cli` + `crawler` + `fetch` + `tui` in
    one binary, and surfaces ergonomic problems no unit test will.
14. **Change monitoring via `unidiff`.** Crawl now, crawl later, diff the
    extracted text: "3 pages changed, 1 added, 2 gone."
15. **Record and replay.** Persist every response, then replay a whole crawl
    offline, deterministically. Same philosophy as `termsession`; makes
    parser development and eval reproduction trivial.
16. **Visual contact sheet.** `thumbnail` + `gif`: crawl a site, emit a grid
    of page thumbnails or an animated flythrough.

## API ergonomics

17. **Iterator API.** `for res := range c.All(ctx, seeds)` (`iter.Seq`)
    alongside the callback. Easier to test and to compose.
18. **Middleware pipeline**, mirroring `cli`'s: `OnRequest`, `OnResponse`,
    `URLFilter`, `ResultFilter`.
19. **Named profiles.** `crawler.Polite()`, `crawler.Archive()`,
    `crawler.LLMCorpus()`, `crawler.Aggressive()`. Highest-leverage change
    for "an agent writes correct code on the first try."
20. **Extract `crawler/robots` as a public sub-package.** Proper agent-group
    merging, sitemap directives, meta-robots and `X-Robots-Tag` support.
    Fixes the empty-user-agent match bug on the way.

## Picked for specs

- **A** — host-aware scheduling on a pluggable `Frontier` (1, 2, 3, 5)
- **B** — LLM corpus mode (11, 7, 6)
- **C** — `wonton crawl` with a live TUI dashboard (12, 13)

Second tier, high fun per line: change monitoring (14) and record/replay (15).
Wildcard: focused crawling with an LLM scorer (9).
