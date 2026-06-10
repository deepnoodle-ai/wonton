// Package web provides URL canonicalization, link resolution, text
// normalization, and web search abstractions for crawlers and content
// processing.
//
// The URL functions parse and canonicalize URLs for stable comparison and
// deduplication ([NormalizeURL]), resolve links found on pages against the
// page URL ([ResolveLink]), and compare hosts ([AreSameHost],
// [AreRelatedHosts]).
//
// [NormalizeText] cleans text extracted from web pages, and [IsBinaryURL]
// identifies URLs that point to file downloads rather than pages.
//
// [Searcher] defines the contract for web search providers (Google, Kagi,
// and so on), with [SearchInput] and [SearchOutput] as the shared request
// and response types.
//
// This package contains no I/O. For fetching pages and downloading files,
// see the fetch package.
package web
