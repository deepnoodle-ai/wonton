package humanize

import (
	"fmt"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{100, "100 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
		{-1024, "-1.0 KiB"},
	}

	for _, tt := range tests {
		result := Bytes(tt.input)
		assert.Equal(t, tt.expected, result, "Bytes(%d)", tt.input)
	}
}

func TestBytesSI(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1000, "1.0 KB"},
		{1500, "1.5 KB"},
		{1000000, "1.0 MB"},
	}

	for _, tt := range tests {
		result := BytesSI(tt.input)
		assert.Equal(t, tt.expected, result, "BytesSI(%d)", tt.input)
	}
}

func TestDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0s"},
		{time.Second, "1s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m 30s"},
		{time.Hour, "1h"},
		{90 * time.Minute, "1h 30m"},
		{25 * time.Hour, "1d 1h"},
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1s 500ms"},
		{61500 * time.Millisecond, "1m 1s"},
		{-30 * time.Second, "-30s"},
	}

	for _, tt := range tests {
		result := Duration(tt.input)
		assert.Equal(t, tt.expected, result, "Duration(%v)", tt.input)
	}
}

func TestDurationShort(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0s"},
		{45 * time.Second, "45s"},
		{90 * time.Second, "1m"},
		{3 * time.Hour, "3h"},
		{48 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		result := DurationShort(tt.input)
		assert.Equal(t, tt.expected, result, "DurationShort(%v)", tt.input)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input    time.Time
		expected string
	}{
		{now, "just now"},
		{now.Add(-30 * time.Second), "30 seconds ago"},
		{now.Add(-1 * time.Second), "1 second ago"},
		{now.Add(-5 * time.Minute), "5 minutes ago"},
		{now.Add(-1 * time.Minute), "1 minute ago"},
		{now.Add(-2 * time.Hour), "2 hours ago"},
		{now.Add(-1 * time.Hour), "1 hour ago"},
		{now.Add(-3 * 24 * time.Hour), "3 days ago"},
		{now.Add(-1 * 24 * time.Hour), "1 day ago"},
		{now.Add(5 * time.Minute), "in 5 minutes"},
		{now.Add(2 * time.Hour), "in 2 hours"},
	}

	for _, tt := range tests {
		result := RelativeTime(tt.input, now)
		assert.Equal(t, tt.expected, result, "RelativeTime(%v)", tt.input)
	}
}

func TestNumber(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{1234, "1,234"},
		{1234567, "1,234,567"},
		{-1234567, "-1,234,567"},
	}

	for _, tt := range tests {
		result := Number(tt.input)
		assert.Equal(t, tt.expected, result, "Number(%d)", tt.input)
	}
}

func TestPercentage(t *testing.T) {
	tests := []struct {
		value    float64
		total    float64
		expected string
	}{
		{0, 100, "0%"},
		{50, 100, "50%"},
		{33, 100, "33%"},
		{1, 3, "33.3%"},
		{0, 0, "0%"},
	}

	for _, tt := range tests {
		result := Percentage(tt.value, tt.total)
		assert.Equal(t, tt.expected, result, "Percentage(%v, %v)", tt.value, tt.total)
	}
}

func TestOrdinal(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{1, "1st"},
		{2, "2nd"},
		{3, "3rd"},
		{4, "4th"},
		{11, "11th"},
		{12, "12th"},
		{13, "13th"},
		{21, "21st"},
		{22, "22nd"},
		{23, "23rd"},
		{101, "101st"},
		{111, "111th"},
		{-1, "-1st"},
		{-2, "-2nd"},
		{-3, "-3rd"},
		{-11, "-11th"},
		{-21, "-21st"},
	}

	for _, tt := range tests {
		result := Ordinal(tt.input)
		assert.Equal(t, tt.expected, result, "Ordinal(%d)", tt.input)
	}
}

func TestPluralWord(t *testing.T) {
	tests := []struct {
		count    int
		singular string
		plural   string
		expected string
	}{
		{0, "item", "items", "0 items"},
		{1, "item", "items", "1 item"},
		{2, "item", "items", "2 items"},
		{-1, "degree", "degrees", "-1 degree"},
	}

	for _, tt := range tests {
		result := PluralWord(tt.count, tt.singular, tt.plural)
		assert.Equal(t, tt.expected, result, "PluralWord(%d, %q, %q)", tt.count, tt.singular, tt.plural)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 3, "..."},
	}

	for _, tt := range tests {
		result := Truncate(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, result, "Truncate(%q, %d)", tt.input, tt.maxLen)
	}
}

func TestTruncateWithSuffix(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		suffix   string
		expected string
	}{
		{"hello world", 8, "~", "hello w~"},
		{"hello", 10, "...", "hello"},
		{"lengthy", 3, "..", "l.."},
	}

	for _, tt := range tests {
		result := TruncateWithSuffix(tt.input, tt.maxLen, tt.suffix)
		assert.Equal(t, tt.expected, result)
	}
}

func TestNumberWithSeparator(t *testing.T) {
	assert.Equal(t, "1.234.567", NumberWithSeparator(1234567, "."))
	assert.Equal(t, "-9 876", NumberWithSeparator(-9876, " "))
	assert.Equal(t, "100", NumberWithSeparator(100, "_"))
}

func TestFloat(t *testing.T) {
	assert.Equal(t, "3.14", Float(3.14159, 2))
	assert.Equal(t, "2.0000", Float(2, 4))
	assert.Equal(t, "-0.50", Float(-0.5, 2))
}

func TestTime(t *testing.T) {
	now := time.Now()
	assert.Equal(t, "just now", Time(now))
	assert.Contains(t, Time(now.Add(-2*time.Minute)), "ago")
	assert.Contains(t, Time(now.Add(3*time.Minute)), "in")
}

func TestPlural(t *testing.T) {
	assert.Equal(t, "item", Plural(1, "item", "items"))
	assert.Equal(t, "items", Plural(2, "item", "items"))
	assert.Equal(t, "items", Plural(0, "item", "items"))
}

// Example functions for godoc

func ExampleBytes() {
	fmt.Println(Bytes(0))
	fmt.Println(Bytes(1024))
	fmt.Println(Bytes(1536))
	fmt.Println(Bytes(1048576))
	// Output:
	// 0 B
	// 1.0 KiB
	// 1.5 KiB
	// 1.0 MiB
}

func ExampleBytesSI() {
	fmt.Println(BytesSI(0))
	fmt.Println(BytesSI(1000))
	fmt.Println(BytesSI(1500))
	fmt.Println(BytesSI(1000000))
	// Output:
	// 0 B
	// 1.0 KB
	// 1.5 KB
	// 1.0 MB
}

func ExampleDuration() {
	fmt.Println(Duration(30 * time.Second))
	fmt.Println(Duration(90 * time.Second))
	fmt.Println(Duration(3665 * time.Second))
	fmt.Println(Duration(25 * time.Hour))
	fmt.Println(Duration(500 * time.Millisecond))
	// Output:
	// 30s
	// 1m 30s
	// 1h 1m
	// 1d 1h
	// 500ms
}

func ExampleDurationShort() {
	fmt.Println(DurationShort(45 * time.Second))
	fmt.Println(DurationShort(90 * time.Second))
	fmt.Println(DurationShort(3 * time.Hour))
	fmt.Println(DurationShort(48 * time.Hour))
	// Output:
	// 45s
	// 1m
	// 3h
	// 2d
}

func ExampleRelativeTime() {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	fmt.Println(RelativeTime(now, now))
	fmt.Println(RelativeTime(now.Add(-30*time.Second), now))
	fmt.Println(RelativeTime(now.Add(-5*time.Minute), now))
	fmt.Println(RelativeTime(now.Add(-2*time.Hour), now))
	fmt.Println(RelativeTime(now.Add(10*time.Minute), now))
	// Output:
	// just now
	// 30 seconds ago
	// 5 minutes ago
	// 2 hours ago
	// in 10 minutes
}

func ExampleNumber() {
	fmt.Println(Number(123))
	fmt.Println(Number(1234))
	fmt.Println(Number(1234567))
	fmt.Println(Number(-1000000))
	// Output:
	// 123
	// 1,234
	// 1,234,567
	// -1,000,000
}

func ExampleNumberWithSeparator() {
	fmt.Println(NumberWithSeparator(1234567, ","))
	fmt.Println(NumberWithSeparator(1234567, "."))
	fmt.Println(NumberWithSeparator(1234567, " "))
	// Output:
	// 1,234,567
	// 1.234.567
	// 1 234 567
}

func ExampleFloat() {
	fmt.Println(Float(3.14159, 2))
	fmt.Println(Float(3.14159, 4))
	fmt.Println(Float(2.0, 1))
	// Output:
	// 3.14
	// 3.1416
	// 2.0
}

func ExamplePercentage() {
	fmt.Println(Percentage(50, 100))
	fmt.Println(Percentage(75, 100))
	fmt.Println(Percentage(1, 3))
	fmt.Println(Percentage(0, 0))
	// Output:
	// 50%
	// 75%
	// 33.3%
	// 0%
}

func ExampleOrdinal() {
	fmt.Println(Ordinal(1))
	fmt.Println(Ordinal(2))
	fmt.Println(Ordinal(3))
	fmt.Println(Ordinal(11))
	fmt.Println(Ordinal(21))
	// Output:
	// 1st
	// 2nd
	// 3rd
	// 11th
	// 21st
}

func ExamplePlural() {
	fmt.Println(Plural(1, "item", "items"))
	fmt.Println(Plural(5, "item", "items"))
	fmt.Println(Plural(0, "item", "items"))
	// Output:
	// item
	// items
	// items
}

func ExamplePluralWord() {
	fmt.Println(PluralWord(1, "file", "files"))
	fmt.Println(PluralWord(5, "file", "files"))
	fmt.Println(PluralWord(0, "item", "items"))
	// Output:
	// 1 file
	// 5 files
	// 0 items
}

func ExampleTruncate() {
	fmt.Println(Truncate("hello", 10))
	fmt.Println(Truncate("hello world", 8))
	fmt.Println(Truncate("hi", 5))
	// Output:
	// hello
	// hello...
	// hi
}

func ExampleTruncateWithSuffix() {
	fmt.Println(TruncateWithSuffix("hello world", 10, "..."))
	fmt.Println(TruncateWithSuffix("hello world", 8, "~"))
	fmt.Println(TruncateWithSuffix("hello", 10, "..."))
	// Output:
	// hello w...
	// hello w~
	// hello
}
