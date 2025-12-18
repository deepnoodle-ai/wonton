# assert

Test assertions with excellent diff output for Go tests. Built on go-cmp with
colored unified diffs on failure.

## Summary

The assert package provides minimal, fatal test assertions optimized for
readability. All assertions fail immediately (t.Fatal) and show clear, colored
diffs when values don't match. Built on google/go-cmp for comparisons, it
compares unexported fields by default and provides specialized assertions for
errors, nil checks, collections, and numeric comparisons.

## Usage Examples

### Basic Equality

```go
package mypackage

import (
    "testing"
    "github.com/deepnoodle-ai/wonton/assert"
)

func TestUser(t *testing.T) {
    got := User{Name: "Alice", Age: 30}
    want := User{Name: "Alice", Age: 30}
    assert.Equal(t, got, want)
}

func TestNotEqual(t *testing.T) {
    got := fetchData()
    forbidden := corruptedData()
    assert.NotEqual(t, got, forbidden)
}
```

### Error Assertions

```go
func TestErrors(t *testing.T) {
    // Assert no error occurred
    err := DoSomething()
    assert.NoError(t, err)

    // Assert an error occurred
    err = DoInvalidThing()
    assert.Error(t, err)

    // Assert error contains specific error in chain
    err = wrapped.Error()
    assert.ErrorIs(t, err, io.EOF)

    // Assert error is of specific type
    var pathErr *os.PathError
    assert.ErrorAs(t, err, &pathErr)

    // Assert error message contains substring
    err = DoThing()
    assert.ErrorContains(t, err, "connection refused")
}
```

### Nil Checks

```go
func TestNil(t *testing.T) {
    var ptr *User
    assert.Nil(t, ptr)

    user := &User{}
    assert.NotNil(t, user)
}
```

### Boolean Assertions

```go
func TestBooleans(t *testing.T) {
    assert.True(t, isValid())
    assert.False(t, isEmpty())
}
```

### Collections

```go
func TestCollections(t *testing.T) {
    slice := []string{"apple", "banana", "cherry"}

    // Check length
    assert.Len(t, slice, 3)

    // Check contains
    assert.Contains(t, slice, "banana")
    assert.NotContains(t, slice, "grape")

    // Check empty/non-empty
    assert.NotEmpty(t, slice)
    assert.Empty(t, []string{})
}
```

### Numeric Comparisons

```go
func TestNumeric(t *testing.T) {
    assert.Greater(t, 10, 5)
    assert.GreaterOrEqual(t, 10, 10)
    assert.Less(t, 5, 10)
    assert.LessOrEqual(t, 5, 5)

    // Floating point comparison with delta
    assert.InDelta(t, 3.14159, 3.14, 0.01)
}
```

### Pattern Matching

```go
func TestPatterns(t *testing.T) {
    email := "user@example.com"
    assert.Regexp(t, `^[\w.]+@[\w.]+$`, email)

    // Can also use compiled regex
    pattern := regexp.MustCompile(`\d{3}-\d{4}`)
    assert.Regexp(t, pattern, "555-1234")
}
```

### Panic Assertions

```go
func TestPanics(t *testing.T) {
    assert.Panics(t, func() {
        panic("boom")
    })

    assert.NotPanics(t, func() {
        DoSafeThing()
    })
}
```

### Custom Options

```go
func TestCustomComparison(t *testing.T) {
    got := []User{{Name: "Alice", Age: 30}}
    want := []User{{Name: "Alice", Age: 31}}

    // Use custom cmp.Options for specialized comparisons
    assert.EqualOpts(t, got, want,
        cmpopts.IgnoreFields(User{}, "Age"))
}
```

### Custom Messages

```go
func TestWithMessages(t *testing.T) {
    assert.Equal(t, got, want, "user data mismatch")
    assert.Equal(t, got, want, "expected %d users, got %d", 5, len(got))
}
```

### Disabling Colors

```go
func TestMain(m *testing.M) {
    // Disable colors for CI environments
    assert.SetColorEnabled(false)
    os.Exit(m.Run())
}
```

## API Reference

### Core Assertions

| Function                           | Description                              | Parameters                                          |
| ---------------------------------- | ---------------------------------------- | --------------------------------------------------- |
| `Equal(t, got, want, msg...)`      | Asserts deep equality using go-cmp       | `t testing.TB`, values to compare, optional message |
| `EqualOpts(t, got, want, opts...)` | Asserts equality with custom cmp.Options | `t testing.TB`, values, `cmp.Option` variadic       |
| `NotEqual(t, got, want, msg...)`   | Asserts values are not equal             | `t testing.TB`, values, optional message            |

### Error Assertions

| Function                                | Description                              | Parameters                                                |
| --------------------------------------- | ---------------------------------------- | --------------------------------------------------------- |
| `NoError(t, err, msg...)`               | Asserts error is nil                     | `t testing.TB`, `error`, optional message                 |
| `Error(t, err, msg...)`                 | Asserts error is not nil                 | `t testing.TB`, `error`, optional message                 |
| `ErrorIs(t, err, target, msg...)`       | Asserts errors.Is(err, target)           | `t testing.TB`, `error`, target `error`, optional message |
| `ErrorAs(t, err, target, msg...)`       | Asserts errors.As(err, target)           | `t testing.TB`, `error`, target pointer, optional message |
| `ErrorContains(t, err, substr, msg...)` | Asserts error message contains substring | `t testing.TB`, `error`, `string`, optional message       |

### Nil Checks

| Function               | Description              | Parameters                              |
| ---------------------- | ------------------------ | --------------------------------------- |
| `Nil(t, v, msg...)`    | Asserts value is nil     | `t testing.TB`, `any`, optional message |
| `NotNil(t, v, msg...)` | Asserts value is not nil | `t testing.TB`, `any`, optional message |

### Boolean Assertions

| Function              | Description              | Parameters                               |
| --------------------- | ------------------------ | ---------------------------------------- |
| `True(t, v, msg...)`  | Asserts boolean is true  | `t testing.TB`, `bool`, optional message |
| `False(t, v, msg...)` | Asserts boolean is false | `t testing.TB`, `bool`, optional message |

### Collection Assertions

| Function                                   | Description                              | Parameters                                                 |
| ------------------------------------------ | ---------------------------------------- | ---------------------------------------------------------- |
| `Contains(t, haystack, needle, msg...)`    | Asserts collection contains value        | `t testing.TB`, collection, value, optional message        |
| `NotContains(t, haystack, needle, msg...)` | Asserts collection doesn't contain value | `t testing.TB`, collection, value, optional message        |
| `Len(t, v, want, msg...)`                  | Asserts length equals expected           | `t testing.TB`, collection, `int` length, optional message |
| `Empty(t, v, msg...)`                      | Asserts value is empty                   | `t testing.TB`, `any`, optional message                    |
| `NotEmpty(t, v, msg...)`                   | Asserts value is not empty               | `t testing.TB`, `any`, optional message                    |

### Panic Assertions

| Function                  | Description                    | Parameters                                 |
| ------------------------- | ------------------------------ | ------------------------------------------ |
| `Panics(t, f, msg...)`    | Asserts function panics        | `t testing.TB`, `func()`, optional message |
| `NotPanics(t, f, msg...)` | Asserts function doesn't panic | `t testing.TB`, `func()`, optional message |

### Numeric Comparisons

| Function                                      | Description                     | Parameters                                           |
| --------------------------------------------- | ------------------------------- | ---------------------------------------------------- |
| `Greater[T](t, a, b, msg...)`                 | Asserts a > b                   | `t testing.TB`, `T` (ordered), `T`, optional message |
| `GreaterOrEqual[T](t, a, b, msg...)`          | Asserts a >= b                  | `t testing.TB`, `T` (ordered), `T`, optional message |
| `Less[T](t, a, b, msg...)`                    | Asserts a < b                   | `t testing.TB`, `T` (ordered), `T`, optional message |
| `LessOrEqual[T](t, a, b, msg...)`             | Asserts a <= b                  | `t testing.TB`, `T` (ordered), `T`, optional message |
| `InDelta(t, expected, actual, delta, msg...)` | Asserts floats are within delta | `t testing.TB`, three `float64`, optional message    |

### Pattern Matching

| Function                          | Description                  | Parameters                                                                         |
| --------------------------------- | ---------------------------- | ---------------------------------------------------------------------------------- |
| `Regexp(t, pattern, str, msg...)` | Asserts string matches regex | `t testing.TB`, pattern (`string` or `*regexp.Regexp`), `string`, optional message |

### Configuration

| Function                   | Description                        | Parameters |
| -------------------------- | ---------------------------------- | ---------- |
| `SetColorEnabled(enabled)` | Enables or disables colored output | `bool`     |

## Related Packages

- **[tui](../tui/)** - Terminal UI library that also uses colored output
- **[color](../color/)** - Color utilities used for diff rendering
- **[termtest](../termtest/)** - Terminal output testing with snapshot support
