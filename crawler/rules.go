package crawler

import (
	"context"
	"regexp"
	"strings"

	"github.com/deepnoodle-ai/wonton/fetch"
)

// MatchType specifies how a rule's pattern should be matched against domains.
// Different match types offer varying levels of flexibility for domain matching.
type MatchType string

const (
	// MatchExact requires the domain to exactly match the pattern string.
	// Example: pattern "example.com" matches only "example.com"
	MatchExact MatchType = "exact"

	// MatchRegex treats the pattern as a regular expression.
	// Example: pattern "example\\.(com|org)" matches "example.com" and "example.org"
	MatchRegex MatchType = "regex"

	// MatchSuffix matches domains that end with the pattern.
	// Example: pattern ".com" matches "example.com", "test.com", etc.
	MatchSuffix MatchType = "suffix"

	// MatchPrefix matches domains that start with the pattern.
	// Example: pattern "blog." matches "blog.example.com", "blog.test.org", etc.
	MatchPrefix MatchType = "prefix"

	// MatchGlob uses glob-style wildcard matching (* and ?).
	// Example: pattern "*.example.com" matches "blog.example.com", "api.example.com", etc.
	MatchGlob MatchType = "glob"
)

// MatchRule defines the core pattern-matching logic used by both ParserRule
// and FetcherRule. It specifies how to match a domain string against a pattern.
type MatchRule struct {
	// Pattern is the string or expression to match against domain names.
	// Its interpretation depends on Type (exact string, regex, glob, etc.).
	Pattern string

	// Type specifies how Pattern should be interpreted (exact, regex, glob, etc.).
	Type MatchType

	// Priority determines the order rules are evaluated. Higher values are
	// checked first. When multiple rules match, the highest priority wins.
	Priority int

	// compiled stores the precompiled regex for MatchRegex and MatchGlob types.
	// This is populated by Compile() and should not be set manually.
	compiled *regexp.Regexp
}

// ParserRule associates a Parser with domains matching a pattern. When a URL's
// domain matches the rule's pattern, the associated parser is used to extract
// structured data from the page.
//
// Use NewParserRule() to create instances with functional options for setting
// priority and match type.
type ParserRule struct {
	MatchRule
	// Parser is the parser to invoke for domains matching this rule
	Parser Parser
}

// FetcherRule associates a Fetcher with domains matching a pattern. When a URL's
// domain matches the rule's pattern, the associated fetcher is used to retrieve
// the page content.
//
// This allows using different fetching strategies for different sites, such as
// using a headless browser for JavaScript-heavy sites while using simple HTTP
// fetching for static sites.
//
// Use NewFetcherRule() to create instances with functional options for setting
// priority and match type.
type FetcherRule struct {
	MatchRule
	// Fetcher is the fetcher to use for domains matching this rule
	Fetcher fetch.Fetcher
}

// Parser extracts structured data from a fetched web page. Implementations
// can parse HTML, extract specific fields, convert to custom types, or perform
// any other transformation of the raw page content.
//
// The returned value can be of any type and will be passed to the callback
// in Result.Parsed. Common return types include structs, maps, or slices.
//
// Example implementation:
//
//	type ArticleParser struct{}
//
//	func (p *ArticleParser) Parse(ctx context.Context, page *fetch.Response) (any, error) {
//		doc, err := htmlparse.Parse(page.HTML)
//		if err != nil {
//			return nil, err
//		}
//		return Article{
//			Title:   htmlparse.ExtractTitle(doc),
//			Content: htmlparse.ExtractMainContent(doc),
//		}, nil
//	}
type Parser interface {
	Parse(ctx context.Context, page *fetch.Response) (any, error)
}

// Compile prepares the MatchRule for use by compiling any necessary patterns.
// For MatchRegex, it compiles the pattern as a regular expression. For MatchGlob,
// it converts the glob pattern to a regex and compiles it. Other match types
// require no compilation.
//
// Returns an error if the pattern is invalid for the specified match type.
// This method is called automatically by AddParserRules and AddFetcherRules.
func (r *MatchRule) Compile() error {
	switch r.Type {
	case MatchRegex:
		compiled, err := regexp.Compile(r.Pattern)
		if err != nil {
			return err
		}
		r.compiled = compiled
	case MatchGlob:
		// Convert glob pattern to regex
		regexPattern := globToRegex(r.Pattern)
		compiled, err := regexp.Compile(regexPattern)
		if err != nil {
			return err
		}
		r.compiled = compiled
	}
	return nil
}

// Matches checks whether the given value matches this rule's pattern according
// to the configured match type. Returns true if the value matches, false otherwise.
//
// The rule must be compiled before calling Matches if it uses MatchRegex or MatchGlob.
func (r *MatchRule) Matches(value string) bool {
	switch r.Type {
	case MatchExact:
		return r.Pattern == value
	case MatchSuffix:
		return strings.HasSuffix(value, r.Pattern)
	case MatchPrefix:
		return strings.HasPrefix(value, r.Pattern)
	case MatchRegex, MatchGlob:
		if r.compiled != nil {
			return r.compiled.MatchString(value)
		}
	}
	return false
}

// globToRegex converts a glob pattern to a regular expression
func globToRegex(pattern string) string {
	// Escape special regex characters except * and ?
	pattern = regexp.QuoteMeta(pattern)
	// Replace escaped glob characters with regex equivalents
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")
	pattern = strings.ReplaceAll(pattern, "\\?", ".")
	// Anchor the pattern
	return "^" + pattern + "$"
}

// ParserRuleOption is a functional option for configuring ParserRule instances.
// Use with NewParserRule to customize priority and match type.
type ParserRuleOption func(*ParserRule)

// WithParserPriority sets the priority for a parser rule. Higher priority rules
// are evaluated first when multiple rules could match a domain.
//
// Default priority is 0.
func WithParserPriority(priority int) ParserRuleOption {
	return func(r *ParserRule) {
		r.Priority = priority
	}
}

// WithParserMatchType sets the match type for a parser rule, determining how
// the pattern is interpreted (exact, regex, glob, prefix, or suffix).
//
// Default match type is MatchExact.
func WithParserMatchType(matchType MatchType) ParserRuleOption {
	return func(r *ParserRule) {
		r.Type = matchType
	}
}

// NewParserRule creates a new parser rule with the given pattern and parser.
// By default, it uses exact matching with priority 0.
// Use functional options to customize behavior.
//
// Example:
//
//	rule := NewParserRule("example.com", parser, WithParserPriority(10))
//	rule := NewParserRule("*.example.com", parser, WithParserMatchType(MatchGlob), WithParserPriority(5))
func NewParserRule(pattern string, parser Parser, opts ...ParserRuleOption) *ParserRule {
	rule := &ParserRule{
		MatchRule: MatchRule{
			Pattern:  pattern,
			Type:     MatchExact, // default to exact matching
			Priority: 0,          // default priority
		},
		Parser: parser,
	}

	// Apply options
	for _, opt := range opts {
		opt(rule)
	}

	return rule
}

// FetcherRuleOption is a functional option for configuring FetcherRule instances.
// Use with NewFetcherRule to customize priority and match type.
type FetcherRuleOption func(*FetcherRule)

// WithFetcherPriority sets the priority for a fetcher rule. Higher priority rules
// are evaluated first when multiple rules could match a domain.
//
// Default priority is 0.
func WithFetcherPriority(priority int) FetcherRuleOption {
	return func(r *FetcherRule) {
		r.Priority = priority
	}
}

// WithFetcherMatchType sets the match type for a fetcher rule, determining how
// the pattern is interpreted (exact, regex, glob, prefix, or suffix).
//
// Default match type is MatchExact.
func WithFetcherMatchType(matchType MatchType) FetcherRuleOption {
	return func(r *FetcherRule) {
		r.Type = matchType
	}
}

// NewFetcherRule creates a new fetcher rule with the given pattern and fetcher.
// By default, it uses exact matching with priority 0.
// Use functional options to customize behavior.
//
// Example:
//
//	rule := NewFetcherRule("example.com", fetcher, WithFetcherPriority(10))
//	rule := NewFetcherRule("*.example.com", fetcher, WithFetcherMatchType(MatchGlob), WithFetcherPriority(5))
func NewFetcherRule(pattern string, fetcher fetch.Fetcher, opts ...FetcherRuleOption) *FetcherRule {
	rule := &FetcherRule{
		MatchRule: MatchRule{
			Pattern:  pattern,
			Type:     MatchExact, // default to exact matching
			Priority: 0,          // default priority
		},
		Fetcher: fetcher,
	}

	// Apply options
	for _, opt := range opts {
		opt(rule)
	}

	return rule
}
