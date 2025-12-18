# humanize

Format values into human-readable strings. Converts byte sizes, durations, numbers, times, and percentages into formats suitable for display in user interfaces and logs.

## Usage Examples

### Byte Sizes

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/humanize"
)

func main() {
    // Binary units (1024-based)
    fmt.Println(humanize.Bytes(1536))        // "1.5 KiB"
    fmt.Println(humanize.Bytes(1048576))     // "1.0 MiB"
    fmt.Println(humanize.Bytes(1073741824))  // "1.0 GiB"

    // SI units (1000-based)
    fmt.Println(humanize.BytesSI(1500))      // "1.5 KB"
    fmt.Println(humanize.BytesSI(1000000))   // "1.0 MB"
}
```

### Durations

```go
import "time"

// Multiple units (shows two most significant)
fmt.Println(humanize.Duration(90 * time.Second))           // "1m 30s"
fmt.Println(humanize.Duration(3665 * time.Second))         // "1h 1m"
fmt.Println(humanize.Duration(25 * time.Hour))             // "1d 1h"
fmt.Println(humanize.Duration(500 * time.Millisecond))     // "500ms"

// Short format (single unit only)
fmt.Println(humanize.DurationShort(90 * time.Second))      // "1m"
fmt.Println(humanize.DurationShort(3665 * time.Second))    // "1h"
fmt.Println(humanize.DurationShort(25 * time.Hour))        // "1d"
```

### Relative Times

```go
now := time.Now()

fmt.Println(humanize.Time(now.Add(-2 * time.Hour)))        // "2 hours ago"
fmt.Println(humanize.Time(now.Add(-30 * time.Minute)))     // "30 minutes ago"
fmt.Println(humanize.Time(now.Add(-5 * time.Second)))      // "just now"
fmt.Println(humanize.Time(now.Add(10 * time.Minute)))      // "in 10 minutes"

// Custom reference time
past := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
ref := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
fmt.Println(humanize.RelativeTime(past, ref))              // "2 weeks ago"
```

### Numbers

```go
fmt.Println(humanize.Number(1234567))                      // "1,234,567"
fmt.Println(humanize.Number(-1000000))                     // "-1,000,000"

// Custom separator
fmt.Println(humanize.NumberWithSeparator(1234567, " "))    // "1 234 567"
fmt.Println(humanize.NumberWithSeparator(1234567, "."))    // "1.234.567"
```

### Floating Point Numbers

```go
fmt.Println(humanize.Float(3.14159265, 2))                 // "3.14"
fmt.Println(humanize.Float(3.14159265, 4))                 // "3.1416"
```

### Percentages

```go
fmt.Println(humanize.Percentage(75, 100))                  // "75%"
fmt.Println(humanize.Percentage(33, 100))                  // "33.0%"
fmt.Println(humanize.Percentage(3, 7))                     // "42.9%"
fmt.Println(humanize.Percentage(0, 0))                     // "0%"
```

### Ordinals

```go
fmt.Println(humanize.Ordinal(1))                           // "1st"
fmt.Println(humanize.Ordinal(2))                           // "2nd"
fmt.Println(humanize.Ordinal(3))                           // "3rd"
fmt.Println(humanize.Ordinal(11))                          // "11th"
fmt.Println(humanize.Ordinal(21))                          // "21st"
fmt.Println(humanize.Ordinal(100))                         // "100th"
```

### Pluralization

```go
// Choose singular or plural
fmt.Println(humanize.Plural(1, "item", "items"))           // "item"
fmt.Println(humanize.Plural(5, "item", "items"))           // "items"

// Include count
fmt.Println(humanize.PluralWord(1, "file", "files"))       // "1 file"
fmt.Println(humanize.PluralWord(5, "file", "files"))       // "5 files"
```

### String Truncation

```go
fmt.Println(humanize.Truncate("Hello, World!", 8))         // "Hello..."

// Custom suffix
fmt.Println(humanize.TruncateWithSuffix("Hello", 10, "…")) // "Hello"
fmt.Println(humanize.TruncateWithSuffix("Hello, World!", 8, "…")) // "Hello, …"
```

## API Reference

### Byte Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Bytes` | Formats bytes using binary units (KiB, MiB) | `b int64` | `string` |
| `BytesSI` | Formats bytes using SI units (KB, MB) | `b int64` | `string` |

### Duration Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Duration` | Formats duration with up to 2 significant units | `d time.Duration` | `string` |
| `DurationShort` | Formats duration with only the most significant unit | `d time.Duration` | `string` |

### Time Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Time` | Formats time relative to now | `t time.Time` | `string` |
| `RelativeTime` | Formats time relative to reference time | `t, ref time.Time` | `string` |

### Number Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Number` | Formats number with comma separators | `n int64` | `string` |
| `NumberWithSeparator` | Formats number with custom separator | `n int64, sep string` | `string` |
| `Float` | Formats float with precision | `f float64, precision int` | `string` |
| `Percentage` | Formats ratio as percentage | `value, total float64` | `string` |

### Word Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Ordinal` | Returns ordinal form (1st, 2nd, 3rd) | `n int` | `string` |
| `Plural` | Returns singular or plural form | `count int, singular, plural string` | `string` |
| `PluralWord` | Returns count with appropriate word form | `count int, singular, plural string` | `string` |

### String Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Truncate` | Truncates string with "..." suffix | `s string, maxLen int` | `string` |
| `TruncateWithSuffix` | Truncates string with custom suffix | `s string, maxLen int, suffix string` | `string` |

## Related Packages

- [terminal](../terminal) - Display humanized values in terminal UIs
- [tui](../tui) - Build UIs that display human-readable metrics

## Design Notes

All functions handle negative values appropriately. Zero values are formatted as "0" with appropriate units. Duration formatting prioritizes readability over precision, showing only the two most significant units.

Relative time uses approximate month (30 days) and year (365 days) calculations rather than calendar-aware logic. This keeps the API simple and avoids timezone complexities.
