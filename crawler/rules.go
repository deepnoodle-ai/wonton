package crawler

import (
	"context"
	"regexp"
	"strings"

	"github.com/deepnoodle-ai/wonton/fetch"
)

// MatchType defines the type of pattern matching for rules
type MatchType string

const (
	MatchExact  MatchType = "exact"  // Exact domain match
	MatchRegex  MatchType = "regex"  // Regular expression match
	MatchSuffix MatchType = "suffix" // Domain suffix match (e.g., ".com")
	MatchPrefix MatchType = "prefix" // Domain prefix match (e.g., "blog.")
	MatchGlob   MatchType = "glob"   // Glob pattern match (e.g., "*.example.com")
)

// MatchRule defines the core matching logic that can be used by different rule types
type MatchRule struct {
	Pattern  string         // The pattern to match against
	Type     MatchType      // The type of matching to perform
	Priority int            // Priority for rule evaluation (higher = first)
	compiled *regexp.Regexp // Compiled regex for performance (internal use)
}

// ParserRule defines a rule for matching domains to parsers
type ParserRule struct {
	MatchRule
	Parser Parser // The parser to use for matching domains
}

// FetcherRule defines a rule for matching domains to fetchers
type FetcherRule struct {
	MatchRule
	Fetcher fetch.Fetcher // The fetcher to use for matching domains
}

// Parser is an interface describing a webpage parser. It accepts the fetched
// page and returns a parsed object.
type Parser interface {
	Parse(ctx context.Context, page *fetch.Response) (any, error)
}

// Compile compiles regex patterns for the match rule if needed
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

// Matches checks if the given value matches the rule
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

// ParserRuleOption defines a function that modifies a ParserRule
type ParserRuleOption func(*ParserRule)

// WithParserPriority sets the priority for a parser rule
func WithParserPriority(priority int) ParserRuleOption {
	return func(r *ParserRule) {
		r.Priority = priority
	}
}

// WithParserMatchType sets the match type for a parser rule
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

// FetcherRuleOption defines a function that modifies a FetcherRule
type FetcherRuleOption func(*FetcherRule)

// WithFetcherPriority sets the priority for a fetcher rule
func WithFetcherPriority(priority int) FetcherRuleOption {
	return func(r *FetcherRule) {
		r.Priority = priority
	}
}

// WithFetcherMatchType sets the match type for a fetcher rule
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
