// Package humanize provides human-readable formatting for common values
// like byte sizes, durations, numbers, and relative times.
//
// This package is designed for displaying data in user interfaces, logs,
// and command-line tools where human readability is more important than
// machine precision. All functions handle negative values appropriately
// and use sensible defaults for zero values.
//
// # Byte Formatting
//
// Format byte counts with binary (1024-based) or SI (1000-based) units:
//
//	humanize.Bytes(1536)           // "1.5 KiB"
//	humanize.BytesSI(1500)         // "1.5 KB"
//
// # Duration Formatting
//
// Format durations with multiple units or in short form:
//
//	humanize.Duration(90*time.Second)      // "1m 30s"
//	humanize.DurationShort(90*time.Second) // "1m"
//
// # Relative Time
//
// Display times relative to now or a reference point:
//
//	humanize.Time(time.Now().Add(-2*time.Hour)) // "2 hours ago"
//	humanize.RelativeTime(past, ref)             // custom reference
//
// # Number Formatting
//
// Format numbers with thousands separators and percentages:
//
//	humanize.Number(1234567)           // "1,234,567"
//	humanize.Percentage(75, 100)       // "75%"
//	humanize.Float(3.14159, 2)         // "3.14"
//
// # Text Utilities
//
// Ordinals, pluralization, and truncation:
//
//	humanize.Ordinal(3)                    // "3rd"
//	humanize.PluralWord(5, "item", "items") // "5 items"
//	humanize.Truncate("Hello, World!", 8)   // "Hello..."
package humanize

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Bytes formats a byte count into a human-readable string using binary units
// (base 1024). Units are KiB, MiB, GiB, TiB, PiB, EiB.
//
// The function automatically selects the most appropriate unit to keep the
// numeric value between 0 and 1024. Negative values are supported and prefixed
// with a minus sign. Zero returns "0 B".
//
// Examples:
//
//	Bytes(0)           // "0 B"
//	Bytes(1024)        // "1.0 KiB"
//	Bytes(1536)        // "1.5 KiB"
//	Bytes(1048576)     // "1.0 MiB"
//	Bytes(-2048)       // "-2.0 KiB"
func Bytes(b int64) string {
	return formatBytes(b, 1024, []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"})
}

// BytesSI formats a byte count using SI (metric) units with base 1000.
// Units are KB, MB, GB, TB, PB, EB.
//
// This function is useful when working with storage devices or networks that
// use decimal units. The function automatically selects the most appropriate
// unit to keep the numeric value between 0 and 1000.
//
// Examples:
//
//	BytesSI(0)         // "0 B"
//	BytesSI(1000)      // "1.0 KB"
//	BytesSI(1500)      // "1.5 KB"
//	BytesSI(1000000)   // "1.0 MB"
func BytesSI(b int64) string {
	return formatBytes(b, 1000, []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"})
}

func formatBytes(b int64, base float64, units []string) string {
	if b < 0 {
		return "-" + formatBytes(-b, base, units)
	}
	if b == 0 {
		return "0 B"
	}

	bf := float64(b)
	for _, unit := range units[:len(units)-1] {
		if bf < base {
			if unit == "B" {
				return fmt.Sprintf("%d %s", b, unit)
			}
			return fmt.Sprintf("%.1f %s", bf, unit)
		}
		bf /= base
	}
	return fmt.Sprintf("%.1f %s", bf, units[len(units)-1])
}

// Duration formats a duration into a human-readable string with up to two
// significant time units (days, hours, minutes, seconds, milliseconds).
//
// The function prioritizes readability over precision by limiting output to
// the two most significant units. For sub-second durations, it displays
// milliseconds. Negative durations are prefixed with a minus sign.
//
// Examples:
//
//	Duration(0)                       // "0s"
//	Duration(30 * time.Second)        // "30s"
//	Duration(90 * time.Second)        // "1m 30s"
//	Duration(3665 * time.Second)      // "1h 1m"
//	Duration(25 * time.Hour)          // "1d 1h"
//	Duration(500 * time.Millisecond)  // "500ms"
//	Duration(-30 * time.Second)       // "-30s"
func Duration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	neg := d < 0
	if neg {
		d = -d
	}

	var parts []string

	days := d / (24 * time.Hour)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
		d -= days * 24 * time.Hour
	}

	hours := d / time.Hour
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		d -= hours * time.Hour
	}

	mins := d / time.Minute
	if mins > 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
		d -= mins * time.Minute
	}

	secs := d / time.Second
	if secs > 0 || len(parts) == 0 {
		if d%time.Second == 0 {
			parts = append(parts, fmt.Sprintf("%ds", secs))
		} else {
			// Show milliseconds for sub-second precision
			ms := float64(d) / float64(time.Millisecond)
			parts = append(parts, fmt.Sprintf("%.0fms", ms))
		}
	}

	// Show at most 2 significant units
	if len(parts) > 2 {
		parts = parts[:2]
	}

	result := strings.Join(parts, " ")
	if neg {
		result = "-" + result
	}
	return result
}

// DurationShort formats a duration as a compact string showing only the most
// significant time unit. This is useful for space-constrained displays.
//
// The function rounds down to the largest whole unit that fits the duration.
// Units include days (d), hours (h), minutes (m), seconds (s), milliseconds
// (ms), and microseconds (µs).
//
// Examples:
//
//	DurationShort(0)                      // "0s"
//	DurationShort(45 * time.Second)       // "45s"
//	DurationShort(90 * time.Second)       // "1m"
//	DurationShort(3 * time.Hour)          // "3h"
//	DurationShort(48 * time.Hour)         // "2d"
//	DurationShort(500 * time.Millisecond) // "500ms"
func DurationShort(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	neg := d < 0
	if neg {
		d = -d
	}

	var result string
	switch {
	case d >= 24*time.Hour:
		result = fmt.Sprintf("%dd", d/(24*time.Hour))
	case d >= time.Hour:
		result = fmt.Sprintf("%dh", d/time.Hour)
	case d >= time.Minute:
		result = fmt.Sprintf("%dm", d/time.Minute)
	case d >= time.Second:
		result = fmt.Sprintf("%ds", d/time.Second)
	case d >= time.Millisecond:
		result = fmt.Sprintf("%dms", d/time.Millisecond)
	default:
		result = fmt.Sprintf("%dµs", d/time.Microsecond)
	}

	if neg {
		return "-" + result
	}
	return result
}

// Time formats a time as a human-readable relative string compared to the
// current time. This is a convenience wrapper around RelativeTime.
//
// The function uses the current time (time.Now()) as the reference point.
// Times in the past are formatted as "X ago", while future times are
// formatted as "in X". Very recent times (less than 1 second) return "just now".
//
// Examples:
//
//	Time(time.Now())                         // "just now"
//	Time(time.Now().Add(-30 * time.Minute))  // "30 minutes ago"
//	Time(time.Now().Add(-2 * time.Hour))     // "2 hours ago"
//	Time(time.Now().Add(10 * time.Minute))   // "in 10 minutes"
func Time(t time.Time) string {
	return RelativeTime(t, time.Now())
}

// RelativeTime formats a time relative to a reference time, producing
// human-readable strings like "2 hours ago" or "in 5 minutes".
//
// The function uses approximate calculations for months (30 days) and years
// (365 days) rather than calendar-aware logic, which keeps the implementation
// simple and avoids timezone complexities. Times within 1 second of the
// reference return "just now".
//
// Examples:
//
//	now := time.Now()
//	past := now.Add(-2 * time.Hour)
//	RelativeTime(past, now)        // "2 hours ago"
//
//	future := now.Add(5 * time.Minute)
//	RelativeTime(future, now)      // "in 5 minutes"
func RelativeTime(t, ref time.Time) string {
	d := ref.Sub(t)
	past := d >= 0
	if !past {
		d = -d
	}

	var value int
	var unit string

	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		value = int(d / time.Second)
		unit = "second"
	case d < time.Hour:
		value = int(d / time.Minute)
		unit = "minute"
	case d < 24*time.Hour:
		value = int(d / time.Hour)
		unit = "hour"
	case d < 7*24*time.Hour:
		value = int(d / (24 * time.Hour))
		unit = "day"
	case d < 30*24*time.Hour:
		value = int(d / (7 * 24 * time.Hour))
		unit = "week"
	case d < 365*24*time.Hour:
		value = int(d / (30 * 24 * time.Hour))
		unit = "month"
	default:
		value = int(d / (365 * 24 * time.Hour))
		unit = "year"
	}

	if value != 1 {
		unit += "s"
	}

	if past {
		return fmt.Sprintf("%d %s ago", value, unit)
	}
	return fmt.Sprintf("in %d %s", value, unit)
}

// Number formats an integer with comma thousands separators for improved
// readability. This is a convenience wrapper around NumberWithSeparator.
//
// Negative numbers are supported and the minus sign is placed before any
// separators. Numbers with 3 or fewer digits are returned unchanged.
//
// Examples:
//
//	Number(0)         // "0"
//	Number(123)       // "123"
//	Number(1234)      // "1,234"
//	Number(1234567)   // "1,234,567"
//	Number(-1000000)  // "-1,000,000"
func Number(n int64) string {
	return NumberWithSeparator(n, ",")
}

// NumberWithSeparator formats an integer with a custom thousands separator.
// This is useful for localization or different formatting conventions.
//
// The separator can be any string. Common choices include "," (English),
// "." (European), " " (French), or "_" (programming contexts).
//
// Examples:
//
//	NumberWithSeparator(1234567, ",")  // "1,234,567"
//	NumberWithSeparator(1234567, ".")  // "1.234.567"
//	NumberWithSeparator(1234567, " ")  // "1 234 567"
//	NumberWithSeparator(1234567, "_")  // "1_234_567"
func NumberWithSeparator(n int64, sep string) string {
	neg := n < 0
	if neg {
		n = -n
	}

	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}

	// Insert separators from right to left
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteString(sep)
		}
		result.WriteRune(c)
	}

	if neg {
		return "-" + result.String()
	}
	return result.String()
}

// Float formats a floating-point number with the specified number of decimal
// places. This is a convenience wrapper around fmt.Sprintf for fixed-precision
// formatting.
//
// The precision parameter controls how many digits appear after the decimal
// point. The number is rounded to fit the specified precision.
//
// Examples:
//
//	Float(3.14159, 2)    // "3.14"
//	Float(3.14159, 4)    // "3.1416"
//	Float(2.0, 4)        // "2.0000"
//	Float(-0.5, 2)       // "-0.50"
func Float(f float64, precision int) string {
	return fmt.Sprintf("%.*f", precision, f)
}

// Percentage formats a ratio as a percentage string. The result shows whole
// percentages without decimals, or one decimal place for non-whole values.
//
// If the total is zero, the function returns "0%" to avoid division by zero.
// The value and total can be in any units as long as they're consistent.
//
// Examples:
//
//	Percentage(0, 100)    // "0%"
//	Percentage(50, 100)   // "50%"
//	Percentage(75, 100)   // "75%"
//	Percentage(1, 3)      // "33.3%"
//	Percentage(2, 3)      // "66.7%"
//	Percentage(0, 0)      // "0%"
func Percentage(value, total float64) string {
	if total == 0 {
		return "0%"
	}
	pct := (value / total) * 100
	if pct == math.Trunc(pct) {
		return fmt.Sprintf("%.0f%%", pct)
	}
	return fmt.Sprintf("%.1f%%", pct)
}

// Ordinal returns the ordinal form of a number with the appropriate English
// suffix (1st, 2nd, 3rd, 4th, etc.).
//
// The function correctly handles special cases like 11th, 12th, and 13th,
// which use "th" instead of following the normal pattern.
//
// Examples:
//
//	Ordinal(1)    // "1st"
//	Ordinal(2)    // "2nd"
//	Ordinal(3)    // "3rd"
//	Ordinal(4)    // "4th"
//	Ordinal(11)   // "11th"
//	Ordinal(21)   // "21st"
//	Ordinal(100)  // "100th"
//	Ordinal(101)  // "101st"
func Ordinal(n int) string {
	suffix := "th"
	switch n % 100 {
	case 11, 12, 13:
		// Special case for 11th, 12th, 13th
	default:
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

// Plural returns the singular or plural form of a word based on a count.
// This is useful for constructing grammatically correct messages.
//
// The function treats 1 and -1 as singular, everything else as plural.
// This handles the common case of negative counts in contexts like
// temperature or balance differences.
//
// Examples:
//
//	Plural(0, "item", "items")    // "items"
//	Plural(1, "item", "items")    // "item"
//	Plural(2, "item", "items")    // "items"
//	Plural(-1, "file", "files")   // "file"
//	Plural(-5, "file", "files")   // "files"
func Plural(count int, singular, plural string) string {
	if count == 1 || count == -1 {
		return singular
	}
	return plural
}

// PluralWord returns a count followed by the appropriate singular or plural
// word form. This is a convenience wrapper that combines the count with the
// result from Plural.
//
// Examples:
//
//	PluralWord(0, "item", "items")    // "0 items"
//	PluralWord(1, "item", "items")    // "1 item"
//	PluralWord(5, "item", "items")    // "5 items"
//	PluralWord(-1, "degree", "degrees") // "-1 degree"
func PluralWord(count int, singular, plural string) string {
	return fmt.Sprintf("%d %s", count, Plural(count, singular, plural))
}

// Truncate shortens a string to the specified maximum length, appending "..."
// if the string is truncated. This is useful for displaying long strings in
// constrained spaces. This is a convenience wrapper around TruncateWithSuffix.
//
// If the string is already shorter than or equal to maxLen, it's returned
// unchanged. The suffix is included in the length calculation, so the result
// never exceeds maxLen characters.
//
// Examples:
//
//	Truncate("hello", 10)         // "hello"
//	Truncate("hello world", 8)    // "hello..."
//	Truncate("hi", 2)             // "hi"
//	Truncate("hello", 3)          // "..."
func Truncate(s string, maxLen int) string {
	return TruncateWithSuffix(s, maxLen, "...")
}

// TruncateWithSuffix shortens a string to the specified maximum length with
// a custom suffix. This allows for localization or different visual styles.
//
// If maxLen is less than or equal to the suffix length, only the suffix
// (truncated to maxLen) is returned. The suffix can be any string, such as
// "...", "…", "~", or " [more]".
//
// Examples:
//
//	TruncateWithSuffix("hello world", 10, "...")  // "hello w..."
//	TruncateWithSuffix("hello world", 8, "~")     // "hello w~"
//	TruncateWithSuffix("hello", 10, "...")        // "hello"
//	TruncateWithSuffix("lengthy", 3, "..")        // "l.."
func TruncateWithSuffix(s string, maxLen int, suffix string) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= len(suffix) {
		return suffix[:maxLen]
	}
	return s[:maxLen-len(suffix)] + suffix
}
