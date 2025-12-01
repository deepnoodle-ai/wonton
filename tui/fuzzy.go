package tui

import (
	"sort"
	"strings"
)

// FuzzyMatch checks if pattern matches text (subsequence match)
func FuzzyMatch(pattern, text string) bool {
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	pIdx := 0
	for i := 0; i < len(text); i++ {
		if pIdx >= len(pattern) {
			return true
		}
		if text[i] == pattern[pIdx] {
			pIdx++
		}
	}
	return pIdx == len(pattern)
}

// FuzzySearch returns the top matches for pattern in candidates
func FuzzySearch(pattern string, candidates []string) []string {
	if pattern == "" {
		return candidates
	}

	var matches []string
	for _, c := range candidates {
		if FuzzyMatch(pattern, c) {
			matches = append(matches, c)
		}
	}

	// Simple sort by length for now (shorter matches are usually better)
	// A better scoring system would be ideal but this is a start
	sort.Slice(matches, func(i, j int) bool {
		return len(matches[i]) < len(matches[j])
	})

	return matches
}
