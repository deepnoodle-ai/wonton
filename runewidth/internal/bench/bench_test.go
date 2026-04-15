// Package bench compares wonton/runewidth against the current upstream
// competitors:
//
//   - github.com/mattn/go-runewidth v0.0.21 (uses clipperhouse/uax29 internally)
//   - github.com/rivo/uniseg v0.4.7 (the reference grapheme segmenter)
//
// This module is isolated from the parent wonton module so it can pull in
// external dependencies without adding them to the public go.mod.
//
// Run with:
//
//	cd runewidth/internal/bench && go test -bench . -benchmem -run '^$' ./...
package bench

import (
	"strings"
	"testing"

	wonton "github.com/deepnoodle-ai/wonton/runewidth"
	gorunewidth "github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

// --- Corpus ---

var (
	asciiShort = "Hello, World!"
	asciiLong  = strings.Repeat("The quick brown fox jumps over the lazy dog. ", 30) // ~1350 bytes
	cjkShort   = "中文字テスト日本語"
	cjkLong    = strings.Repeat("中文字テスト日本語、これはテストです。", 50) // ~3500 bytes
	emoji      = "Hello 😀🎉👍🔥 World"
	zwj        = "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"
	flags      = "\U0001F1FA\U0001F1F8 \U0001F1EF\U0001F1F5 \U0001F1E9\U0001F1EA \U0001F1EB\U0001F1F7"
	combining  = strings.Repeat("e\u0301a\u0308i\u0302o\u030Bu\u0304 ", 30)
	mixed10KB  = strings.Repeat(
		"Hello, 世界! 👋🏽 ZWJ "+zwj+" flags "+flags+" combining noe\u0308l — ",
		25,
	) // ~10KB
)

// --- StringWidth benchmarks ---

func BenchmarkStringWidth_ASCIIShort_Wonton(b *testing.B) { benchStringWidth(b, asciiShort, wontonSW) }
func BenchmarkStringWidth_ASCIIShort_GoRW(b *testing.B) {
	benchStringWidth(b, asciiShort, gorunewidth.StringWidth)
}
func BenchmarkStringWidth_ASCIIShort_Uniseg(b *testing.B) {
	benchStringWidth(b, asciiShort, uniseg.StringWidth)
}

func BenchmarkStringWidth_ASCIILong_Wonton(b *testing.B) { benchStringWidth(b, asciiLong, wontonSW) }
func BenchmarkStringWidth_ASCIILong_GoRW(b *testing.B) {
	benchStringWidth(b, asciiLong, gorunewidth.StringWidth)
}
func BenchmarkStringWidth_ASCIILong_Uniseg(b *testing.B) {
	benchStringWidth(b, asciiLong, uniseg.StringWidth)
}

func BenchmarkStringWidth_CJKShort_Wonton(b *testing.B) { benchStringWidth(b, cjkShort, wontonSW) }
func BenchmarkStringWidth_CJKShort_GoRW(b *testing.B) {
	benchStringWidth(b, cjkShort, gorunewidth.StringWidth)
}
func BenchmarkStringWidth_CJKShort_Uniseg(b *testing.B) {
	benchStringWidth(b, cjkShort, uniseg.StringWidth)
}

func BenchmarkStringWidth_CJKLong_Wonton(b *testing.B) { benchStringWidth(b, cjkLong, wontonSW) }
func BenchmarkStringWidth_CJKLong_GoRW(b *testing.B) {
	benchStringWidth(b, cjkLong, gorunewidth.StringWidth)
}
func BenchmarkStringWidth_CJKLong_Uniseg(b *testing.B) {
	benchStringWidth(b, cjkLong, uniseg.StringWidth)
}

func BenchmarkStringWidth_Emoji_Wonton(b *testing.B) { benchStringWidth(b, emoji, wontonSW) }
func BenchmarkStringWidth_Emoji_GoRW(b *testing.B) {
	benchStringWidth(b, emoji, gorunewidth.StringWidth)
}
func BenchmarkStringWidth_Emoji_Uniseg(b *testing.B) { benchStringWidth(b, emoji, uniseg.StringWidth) }

func BenchmarkStringWidth_ZWJ_Wonton(b *testing.B) { benchStringWidth(b, zwj, wontonSW) }
func BenchmarkStringWidth_ZWJ_GoRW(b *testing.B)   { benchStringWidth(b, zwj, gorunewidth.StringWidth) }
func BenchmarkStringWidth_ZWJ_Uniseg(b *testing.B) { benchStringWidth(b, zwj, uniseg.StringWidth) }

func BenchmarkStringWidth_Flags_Wonton(b *testing.B) { benchStringWidth(b, flags, wontonSW) }
func BenchmarkStringWidth_Flags_GoRW(b *testing.B) {
	benchStringWidth(b, flags, gorunewidth.StringWidth)
}
func BenchmarkStringWidth_Flags_Uniseg(b *testing.B) { benchStringWidth(b, flags, uniseg.StringWidth) }

func BenchmarkStringWidth_Combining_Wonton(b *testing.B) { benchStringWidth(b, combining, wontonSW) }
func BenchmarkStringWidth_Combining_GoRW(b *testing.B) {
	benchStringWidth(b, combining, gorunewidth.StringWidth)
}
func BenchmarkStringWidth_Combining_Uniseg(b *testing.B) {
	benchStringWidth(b, combining, uniseg.StringWidth)
}

func BenchmarkStringWidth_Mixed10KB_Wonton(b *testing.B) { benchStringWidth(b, mixed10KB, wontonSW) }
func BenchmarkStringWidth_Mixed10KB_GoRW(b *testing.B) {
	benchStringWidth(b, mixed10KB, gorunewidth.StringWidth)
}
func BenchmarkStringWidth_Mixed10KB_Uniseg(b *testing.B) {
	benchStringWidth(b, mixed10KB, uniseg.StringWidth)
}

func wontonSW(s string) int { return wonton.StringWidth(s) }

func benchStringWidth(b *testing.B, s string, fn func(string) int) {
	b.ReportAllocs()
	b.SetBytes(int64(len(s)))
	for b.Loop() {
		_ = fn(s)
	}
}

// --- Truncate benchmarks ---

func BenchmarkTruncate_ASCIILong_Wonton(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = wonton.Truncate(asciiLong, 80, "…")
	}
}
func BenchmarkTruncate_ASCIILong_GoRW(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = gorunewidth.Truncate(asciiLong, 80, "…")
	}
}

func BenchmarkTruncate_Mixed_Wonton(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = wonton.Truncate(mixed10KB, 80, "…")
	}
}
func BenchmarkTruncate_Mixed_GoRW(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = gorunewidth.Truncate(mixed10KB, 80, "…")
	}
}
