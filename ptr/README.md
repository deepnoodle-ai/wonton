# ptr

Generic helpers for working with pointers: box a value, unbox one safely, and express "omit this field when empty" without a temporary variable.

## Summary

Code that speaks JSON or talks to generated API clients constantly needs a pointer to a literal (so an optional field can be omitted) or the zero value of a nil pointer (so a read is safe). Go has no address-of operator for expressions, so each of those becomes a temporary variable or a hand-rolled helper. This package supplies the helpers once.

## Usage Examples

### Boxing Values

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/deepnoodle-ai/wonton/ptr"
)

type SearchRequest struct {
    Query string  `json:"query"`
    Limit *int    `json:"limit,omitempty"`
    Cursor *string `json:"cursor,omitempty"`
}

func main() {
    req := SearchRequest{
        Query: "wonton",
        Limit: ptr.To(50), // no temporary variable needed
    }
    out, _ := json.Marshal(req)
    fmt.Println(string(out)) // {"query":"wonton","limit":50}
}
```

### Reading Optional Fields

```go
// Deref returns the zero value when the pointer is nil.
limit := ptr.Deref(req.Limit)        // 0 when unset

// Or supplies a fallback instead.
limit = ptr.Or(req.Limit, 25)        // 25 when unset

// A pointer to the zero value is still a value: Or returns 0, not 25.
zero := ptr.Or(ptr.To(0), 25)        // 0
```

### Omitting Empty Values

```go
// nil when the string is empty, so the field is omitted from the JSON.
req.Cursor = ptr.IfNotZero(cursor)

// nil when the map or slice has no entries.
body.Labels = ptr.MapIfNotEmpty(labels)
body.Tags = ptr.SliceIfNotEmpty(tags)
```

`SliceIfNotEmpty` boxes a *copy*, so mutating the source slice afterward does
not change what the caller sends.

### Ranging Over Boxed Collections

Generated clients often box collection fields as `*[]T` or `*map[K]V`.
`DerefSlice` and `DerefMap` make those safe to range over:

```go
for _, item := range ptr.DerefSlice(resp.Items) { // no nil check needed
    fmt.Println(item.ID)
}
```

## API Reference

| Function                   | Description                                              | Returns    |
| -------------------------- | -------------------------------------------------------- | ---------- |
| `To(v)`                    | Pointer to `v`                                           | `*T`       |
| `Deref(p)`                 | `*p`, or the zero value of `T` when `p` is nil           | `T`        |
| `Or(p, fallback)`          | `*p`, or `fallback` when `p` is nil                      | `T`        |
| `IfNotZero(v)`             | Pointer to `v`, or nil when `v` is the zero value        | `*T`       |
| `DerefSlice(p)`            | `*p`, or nil when `p` is nil                             | `[]T`      |
| `DerefMap(p)`              | `*p`, or nil when `p` is nil                             | `map[K]V`  |
| `MapIfNotEmpty(m)`         | `&m`, or nil when `m` has no entries                     | `*map[K]V` |
| `SliceIfNotEmpty(values)`  | Pointer to a copy of `values`, or nil when empty         | `*[]T`     |

`IfNotZero` requires `T` to be comparable; the rest accept any type.

## Related Packages

- **[strs](../strs/)** — the same kind of small helper, for strings and string slices
- **[schema](../schema/)** — JSON Schema generation, where optional fields show up constantly
