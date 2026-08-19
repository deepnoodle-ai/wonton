# strs

Small helpers for strings and string slices: pick the first non-empty value from a set of fallbacks, or remove duplicates while preserving order.

## Summary

These are the one-line helpers that otherwise get re-implemented in every package that assembles a value from layered configuration or collects a list of tags, hosts, or paths. Nothing here allocates unless it has to, and every function is safe to call with a nil or empty input.

## Usage Examples

### Layered Fallbacks

```go
package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/deepnoodle-ai/wonton/strs"
)

func main() {
    endpointFlag := flag.String("endpoint", "", "API endpoint")
    flag.Parse()

    // Take the flag if set, then the environment, then a default.
    endpoint := strs.FirstNonEmpty(
        *endpointFlag,
        os.Getenv("API_ENDPOINT"),
        "https://api.example.com",
    )
    fmt.Println(endpoint)
}
```

`FirstNonEmpty` treats a whitespace-only string as a real value. When the input
comes from a config file or user typing, use `FirstNonBlank` instead:

```go
// Returns "  from-config  " — the original, with whitespace intact.
value := strs.FirstNonBlank("", "   ", "  from-config  ")

// Returns "from-config" — trimmed.
value = strs.FirstNonBlankTrim("", "   ", "  from-config  ")
```

### Deduplicating While Preserving Order

```go
hosts := strs.Dedupe([]string{"b.example.com", "a.example.com", "b.example.com"})
// [b.example.com a.example.com]

// Trims each value, drops blanks, and dedupes the trimmed form.
tags := strs.DedupeNonBlank([]string{" go ", "go", "", "  ", "cli"})
// [go cli]
```

`Dedupe` keeps empty strings; `DedupeNonBlank` drops them. Both preserve the
order of first appearance and return nil rather than an empty slice when
nothing survives.

## API Reference

| Function                       | Description                                                        | Returns    |
| ------------------------------ | ------------------------------------------------------------------ | ---------- |
| `FirstNonEmpty(values...)`     | First value that is not `""`                                       | `string`   |
| `FirstNonBlank(values...)`     | First value that is non-empty after trimming; returns the original | `string`   |
| `FirstNonBlankTrim(values...)` | Same, but returns the trimmed value                                | `string`   |
| `Dedupe(values)`               | Duplicates removed, order preserved, empty strings kept            | `[]string` |
| `DedupeNonBlank(values)`       | Duplicates and blanks removed, values trimmed                      | `[]string` |

## Related Packages

- **[ptr](../ptr/)** — the same kind of small helper, for pointers
- **[humanize](../humanize/)** — formatting values for display
- **[web](../web/)** — URL canonicalization and text normalization
