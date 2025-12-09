package humanize

import (
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/require"
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
		require.Equal(t, tt.expected, result, "Bytes(%d)", tt.input)
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
		require.Equal(t, tt.expected, result, "BytesSI(%d)", tt.input)
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
		{-30 * time.Second, "-30s"},
	}

	for _, tt := range tests {
		result := Duration(tt.input)
		require.Equal(t, tt.expected, result, "Duration(%v)", tt.input)
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
		require.Equal(t, tt.expected, result, "DurationShort(%v)", tt.input)
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
		require.Equal(t, tt.expected, result, "RelativeTime(%v)", tt.input)
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
		require.Equal(t, tt.expected, result, "Number(%d)", tt.input)
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
		require.Equal(t, tt.expected, result, "Percentage(%v, %v)", tt.value, tt.total)
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
	}

	for _, tt := range tests {
		result := Ordinal(tt.input)
		require.Equal(t, tt.expected, result, "Ordinal(%d)", tt.input)
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
		require.Equal(t, tt.expected, result, "PluralWord(%d, %q, %q)", tt.count, tt.singular, tt.plural)
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
		require.Equal(t, tt.expected, result, "Truncate(%q, %d)", tt.input, tt.maxLen)
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
		require.Equal(t, tt.expected, result)
	}
}

func TestNumberWithSeparator(t *testing.T) {
	require.Equal(t, "1.234.567", NumberWithSeparator(1234567, "."))
	require.Equal(t, "-9 876", NumberWithSeparator(-9876, " "))
	require.Equal(t, "100", NumberWithSeparator(100, "_"))
}

func TestFloat(t *testing.T) {
	require.Equal(t, "3.14", Float(3.14159, 2))
	require.Equal(t, "2.0000", Float(2, 4))
	require.Equal(t, "-0.50", Float(-0.5, 2))
}

func TestTime(t *testing.T) {
	now := time.Now()
	require.Equal(t, "just now", Time(now))
	require.Contains(t, Time(now.Add(-2*time.Minute)), "ago")
	require.Contains(t, Time(now.Add(3*time.Minute)), "in")
}

func TestPlural(t *testing.T) {
	require.Equal(t, "item", Plural(1, "item", "items"))
	require.Equal(t, "items", Plural(2, "item", "items"))
	require.Equal(t, "items", Plural(0, "item", "items"))
}
