# Testing assertions

The `assert` package is a focused assertion helper for Go tests. It borrows the
ergonomics of `testify/assert` but trims dependencies, embraces generics where
useful, and surfaces high-signal diffs for failures.

## Supported helpers

- Equality: `assert.Equal`, `assert.NotEqual`, `assert.Same`, `assert.NotSame`.
- Truthiness: `assert.True`, `assert.False`, `assert.Nil`, `assert.NotNil`,
  `assert.Zero`, `assert.NotZero`.
- Collections: `assert.Len`, `assert.Contains`, `assert.NotContains`,
  `assert.ElementsMatch`, `assert.Subset`.
- Errors and panics: `assert.NoError`, `assert.Error`, `assert.Panics`,
  `assert.PanicsWithValue`, `assert.PanicsWithError`.

See `assert/assert.go` for the full list.

## Example

```go
package widgets_test

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestBuilder(t *testing.T) {
	result, err := buildWidget("demo")
	assert.NoError(t, err)

	assert.Equal(t, "demo", result.Name)
	assert.ElementsMatch(t, []string{"red", "blue"}, result.Tags)

	assert.PanicsWithError(t, "invalid mode", func() {
		buildWidget("panic")
	})
}
```

`assert.FailNow` integrates with `t.FailNow`, so your tests stop immediately when
critical conditions are not met.
