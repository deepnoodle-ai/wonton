// Package strs provides small helpers for working with strings and string
// slices: picking the first non-empty value from a set of fallbacks, and
// removing duplicates while preserving order.
//
// These are the one-line helpers that otherwise get re-implemented in every
// package that assembles a value from layered configuration or collects a
// list of tags, hosts, or paths.
package strs

import "strings"

// FirstNonEmpty returns the first value that is not the empty string, or ""
// if every value is empty. Use it for layered fallbacks, such as a flag value
// falling back to an environment variable and then to a default.
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// FirstNonBlank returns the first value whose whitespace-trimmed form is
// non-empty, or "" if every value is blank. The returned string is the
// original value, with its surrounding whitespace intact.
func FirstNonBlank(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// FirstNonBlankTrim is like [FirstNonBlank], but returns the trimmed value
// rather than the original.
func FirstNonBlankTrim(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// Dedupe returns values with duplicates removed, preserving the order of
// first appearance. Empty strings are preserved; use [DedupeNonBlank] to drop
// them. A nil or empty input returns nil.
func Dedupe(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// DedupeNonBlank returns values with duplicates and blank entries removed,
// preserving the order of first appearance. Each value is trimmed before the
// comparison, and the trimmed form is what the caller receives. The result is
// nil when nothing survives, so an all-blank input is indistinguishable from
// an empty one.
func DedupeNonBlank(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
