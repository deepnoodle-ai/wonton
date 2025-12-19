package web

import "context"

// SearchInput contains parameters for a web search query.
type SearchInput struct {
	// Query is the search query string.
	Query string `json:"query"`

	// Limit is the maximum number of results to return.
	// A value of 0 uses the implementation's default limit.
	Limit int `json:"limit,omitempty"`
}

// SearchOutput contains the results of a web search.
type SearchOutput struct {
	// Items contains the search results, ordered by relevance.
	Items []*SearchItem `json:"items"`
}

// SearchItem represents a single search result.
type SearchItem struct {
	// URL is the web address of the search result.
	URL string `json:"url"`

	// Title is the page title or headline.
	Title string `json:"title"`

	// Description is a summary or snippet from the page content.
	Description string `json:"description,omitempty"`

	// Icon is the URL of the site's favicon or icon.
	Icon string `json:"icon,omitempty"`

	// Image is the URL of a preview image, if available.
	Image string `json:"image,omitempty"`
}

// Searcher defines the interface for web search implementations.
//
// Implementations might include search engine APIs (Google, Bing, DuckDuckGo),
// site-specific search, or custom search indexes.
//
// Example implementation:
//
//	type GoogleSearcher struct {
//	    APIKey string
//	    CX     string
//	}
//
//	func (s *GoogleSearcher) Search(ctx context.Context, input *web.SearchInput) (*web.SearchOutput, error) {
//	    // Call Google Custom Search API
//	    // ...
//	    return &web.SearchOutput{Items: items}, nil
//	}
type Searcher interface {
	// Search performs a web search and returns matching results.
	// Returns an error if the search fails or the context is canceled.
	Search(ctx context.Context, input *SearchInput) (*SearchOutput, error)
}
