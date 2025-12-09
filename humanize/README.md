# Human-friendly formatting

The `humanize` package turns numbers, durations, and timestamps into strings
people can read at a glance.

## Helpers

- `humanize.Bytes` / `BytesSI`: convert byte counts into `1.2 GiB` or `1.2 GB`.
- `humanize.Duration`, `DurationShort`: summarize `time.Duration` values.
- `humanize.Time` / `RelativeTime`: say `"5 minutes ago"` or `"in 2 hours"`.
- `humanize.Number`, `Float`, `Percentage`, `Ordinal`, and `Plural` cover common
  text presentation patterns.
- `humanize.Truncate` and `TruncateWithSuffix` keep labels within a max width.

## Example

```go
package main

import (
	"fmt"
	"time"

	"github.com/deepnoodle-ai/wonton/humanize"
)

func main() {
	fmt.Println(humanize.Bytes(1536))            // "1.5 KiB"
	fmt.Println(humanize.Duration(42 * time.Second))
	fmt.Println(humanize.Time(time.Now().Add(-15 * time.Minute)))
	fmt.Println(humanize.Percentage(42, 100))    // "42%"
	fmt.Println(humanize.Ordinal(21))            // "21st"
	fmt.Println(humanize.Truncate("long description", 8))
}
```

See `humanize/humanize_test.go` for edge cases and formatting expectations.
