// Package humanize provides human-readable formatting for common values
// like byte sizes, durations, numbers, and relative times.
//
// Basic usage:
//
//	humanize.Bytes(1536)           // "1.5 KB"
//	humanize.Duration(90*time.Second) // "1m 30s"
//	humanize.Time(time.Now().Add(-2*time.Hour)) // "2 hours ago"
//	humanize.Number(1234567)       // "1,234,567"
package humanize

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Bytes formats a byte count into a human-readable string using binary units (KiB, MiB, etc.).
func Bytes(b int64) string {
	return formatBytes(b, 1024, []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"})
}

// BytesSI formats a byte count using SI units (KB, MB, etc.) with base 1000.
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

// Duration formats a duration into a human-readable string.
// For durations under a minute, shows seconds. For longer durations,
// shows the two most significant units.
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

// DurationShort formats a duration as a short string (e.g., "2h", "5m", "30s").
// Shows only the most significant unit.
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

// Time formats a time as a human-readable relative string (e.g., "2 hours ago", "in 5 minutes").
func Time(t time.Time) string {
	return RelativeTime(t, time.Now())
}

// RelativeTime formats a time relative to a reference time.
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

// Number formats a number with thousands separators.
func Number(n int64) string {
	return NumberWithSeparator(n, ",")
}

// NumberWithSeparator formats a number with a custom thousands separator.
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

// Float formats a float with the specified precision.
func Float(f float64, precision int) string {
	return fmt.Sprintf("%.*f", precision, f)
}

// Percentage formats a value as a percentage.
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

// Ordinal returns the ordinal form of a number (1st, 2nd, 3rd, etc.).
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

// Plural returns the singular or plural form based on count.
// Example: Plural(1, "item", "items") returns "item"
func Plural(count int, singular, plural string) string {
	if count == 1 || count == -1 {
		return singular
	}
	return plural
}

// PluralWord returns count followed by the appropriate word form.
// Example: PluralWord(5, "item", "items") returns "5 items"
func PluralWord(count int, singular, plural string) string {
	return fmt.Sprintf("%d %s", count, Plural(count, singular, plural))
}

// Truncate truncates a string to the specified length, adding an ellipsis if truncated.
func Truncate(s string, maxLen int) string {
	return TruncateWithSuffix(s, maxLen, "...")
}

// TruncateWithSuffix truncates a string with a custom suffix.
func TruncateWithSuffix(s string, maxLen int, suffix string) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= len(suffix) {
		return suffix[:maxLen]
	}
	return s[:maxLen-len(suffix)] + suffix
}
