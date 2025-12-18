// Package assert provides minimal test assertions with excellent diff output.
//
// This package offers a focused set of assertion functions for Go tests, built on
// google/go-cmp for deep comparisons. When assertions fail, you get colored unified
// diffs that clearly show what changed, making test failures easy to debug.
//
// # Features
//
//   - Deep equality with go-cmp: Compares structs, slices, maps, and nested types
//   - Colored diffs: Red for expected (-want), green for actual (+got)
//   - Fatal assertions: All assertions fail immediately via t.Fatalf()
//   - Unexported field support: Compares unexported fields by default
//   - Optional messages: All assertions accept optional message formatting
//
// # Basic Usage
//
//	func TestUser(t *testing.T) {
//	    user := fetchUser()
//	    assert.Equal(t, user.Name, "Alice")
//	    assert.NoError(t, err)
//	    assert.True(t, user.Active, "user should be active")
//	}
//
// # Equality Assertions
//
// Equal and NotEqual use go-cmp for deep comparison:
//
//	assert.Equal(t, got, want)                    // Deep equality
//	assert.EqualOpts(t, got, want, cmpOpts...)    // With custom cmp options
//	assert.NotEqual(t, got, want)                 // Values should differ
//
// # Error Assertions
//
// Test error conditions with various matchers:
//
//	assert.NoError(t, err)                        // err must be nil
//	assert.Error(t, err)                          // err must not be nil
//	assert.ErrorIs(t, err, target)                // errors.Is match
//	assert.ErrorAs(t, err, &target)               // errors.As match
//	assert.ErrorContains(t, err, "not found")     // Substring match
//
// # Nil and Boolean Assertions
//
//	assert.Nil(t, ptr)                            // Value must be nil
//	assert.NotNil(t, ptr)                         // Value must not be nil
//	assert.True(t, condition)                     // Boolean must be true
//	assert.False(t, condition)                    // Boolean must be false
//
// # Collection Assertions
//
//	assert.Contains(t, haystack, needle)          // String/slice/map contains
//	assert.NotContains(t, haystack, needle)       // Does not contain
//	assert.Len(t, collection, 5)                  // Length check
//	assert.Empty(t, collection)                   // Nil, zero-length, or zero value
//	assert.NotEmpty(t, collection)                // Has content
//
// # Comparison Assertions
//
// Generic comparison functions work with any ordered type:
//
//	assert.Greater(t, a, b)                       // a > b
//	assert.GreaterOrEqual(t, a, b)                // a >= b
//	assert.Less(t, a, b)                          // a < b
//	assert.LessOrEqual(t, a, b)                   // a <= b
//	assert.InDelta(t, 3.14, 3.15, 0.01)          // Float comparison with delta
//
// # Pattern and Panic Assertions
//
//	assert.Regexp(t, `\d+`, str)                  // String matches regex
//	assert.Panics(t, func() { ... })              // Function must panic
//	assert.NotPanics(t, func() { ... })           // Function must not panic
//
// # Optional Messages
//
// All assertions accept optional message arguments for context:
//
//	assert.Equal(t, got, want, "processing user ID %d", userID)
//	assert.NoError(t, err, "failed to open file")
//
// The first msgAndArgs argument can be a format string with additional arguments,
// or a single value that will be formatted with %v.
package assert

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/color"
	gocmp "github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var colorEnabled = color.IsTerminal(os.Stderr)

// SetColorEnabled enables or disables colored output in diff messages.
//
// Color is automatically enabled if stderr is a terminal. Use this function
// to explicitly control color output, for example to disable it in CI environments
// or when capturing test output to files.
func SetColorEnabled(enabled bool) { colorEnabled = enabled }

// defaultOpts are applied to all Equal comparisons.
// - EquateErrors allows comparing error types safely
// - Exporter allows comparing unexported fields (like reflect.DeepEqual)
var defaultOpts = []gocmp.Option{
	cmpopts.EquateErrors(),
	gocmp.Exporter(func(reflect.Type) bool { return true }),
}

// Equal asserts that got and want are deeply equal using go-cmp.
//
// This function performs deep comparison of any Go values including structs,
// slices, maps, and nested types. Unexported fields are compared by default.
// On failure, it displays a colored unified diff showing exactly what differs.
//
// Optional msgAndArgs provide additional context:
//   - Single string: used as-is
//   - Format string + args: passed to fmt.Sprintf
//
// Example:
//
//	user := User{Name: "Alice", Age: 30}
//	assert.Equal(t, user.Name, "Alice")
//	assert.Equal(t, user, want, "fetching user %d", userID)
func Equal(t testing.TB, got, want any, msgAndArgs ...any) {
	t.Helper()
	if diff := gocmp.Diff(want, got, defaultOpts...); diff != "" {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("\n%s\n%s", formatDiff(diff), msg)
		} else {
			t.Fatalf("\n%s", formatDiff(diff))
		}
	}
}

// EqualOpts asserts equality with custom cmp.Options.
//
// Use this when you need to customize comparison behavior beyond the defaults.
// Useful for ignoring fields, custom comparers, or transforming values.
//
// Example:
//
//	opts := cmp.Options{
//	    cmpopts.IgnoreFields(User{}, "LastLogin"),
//	    cmpopts.EquateApprox(0.01, 0),
//	}
//	assert.EqualOpts(t, got, want, opts...)
func EqualOpts(t testing.TB, got, want any, opts ...gocmp.Option) {
	t.Helper()
	if diff := gocmp.Diff(want, got, opts...); diff != "" {
		t.Fatalf("\n%s", formatDiff(diff))
	}
}

// NotEqual asserts that got and want are not deeply equal.
func NotEqual(t testing.TB, got, want any, msgAndArgs ...any) {
	t.Helper()
	if gocmp.Equal(got, want) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected values to differ, but both are:\n%#v\n%s", got, msg)
		} else {
			t.Fatalf("expected values to differ, but both are:\n%#v", got)
		}
	}
}

// NoError asserts that err is nil.
func NoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("unexpected error: %v\n%s", err, msg)
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// Error asserts that err is not nil.
func Error(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected an error but got nil\n%s", msg)
		} else {
			t.Fatalf("expected an error but got nil")
		}
	}
}

// ErrorIs asserts that errors.Is(err, target) is true.
//
// This checks if target appears anywhere in err's error chain, following
// the standard library's errors.Is semantics. Useful for wrapped errors.
//
// Example:
//
//	var ErrNotFound = errors.New("not found")
//	err := fmt.Errorf("user: %w", ErrNotFound)
//	assert.ErrorIs(t, err, ErrNotFound)
func ErrorIs(t testing.TB, err, target error, msgAndArgs ...any) {
	t.Helper()
	if !errors.Is(err, target) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("error chain does not contain target\n  got:    %v\n  target: %v\n%s", err, target, msg)
		} else {
			t.Fatalf("error chain does not contain target\n  got:    %v\n  target: %v", err, target)
		}
	}
}

// ErrorAs asserts that errors.As(err, target) is true.
//
// This checks if any error in err's chain matches target's type, and if so,
// assigns it to target. The target must be a pointer to an error type.
//
// Example:
//
//	var pathErr *os.PathError
//	assert.ErrorAs(t, err, &pathErr)
//	// Now pathErr is populated with the matched error
func ErrorAs(t testing.TB, err error, target any, msgAndArgs ...any) {
	t.Helper()
	if !errors.As(err, target) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("error chain does not contain target type %T\n  got: %v\n%s", target, err, msg)
		} else {
			t.Fatalf("error chain does not contain target type %T\n  got: %v", target, err)
		}
	}
}

// ErrorContains asserts that err's Error() string contains the substring.
//
// This performs a simple substring check on the error message. Fails if err is nil.
//
// Example:
//
//	err := processFile("/tmp/missing")
//	assert.ErrorContains(t, err, "no such file")
func ErrorContains(t testing.TB, err error, substr string, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected error containing %q but got nil\n%s", substr, msg)
		} else {
			t.Fatalf("expected error containing %q but got nil", substr)
		}
		return
	}
	if !strings.Contains(err.Error(), substr) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("error %q does not contain %q\n%s", err.Error(), substr, msg)
		} else {
			t.Fatalf("error %q does not contain %q", err.Error(), substr)
		}
	}
}

// Nil asserts that v is nil.
func Nil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if !isNil(v) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected nil, got: %#v\n%s", v, msg)
		} else {
			t.Fatalf("expected nil, got: %#v", v)
		}
	}
}

// NotNil asserts that v is not nil.
func NotNil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if isNil(v) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected non-nil value\n%s", msg)
		} else {
			t.Fatalf("expected non-nil value")
		}
	}
}

// True asserts that v is true.
func True(t testing.TB, v bool, msgAndArgs ...any) {
	t.Helper()
	if !v {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected true\n%s", msg)
		} else {
			t.Fatalf("expected true")
		}
	}
}

// False asserts that v is false.
func False(t testing.TB, v bool, msgAndArgs ...any) {
	t.Helper()
	if v {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected false\n%s", msg)
		} else {
			t.Fatalf("expected false")
		}
	}
}

// Contains asserts that haystack contains needle.
//
// Behavior depends on haystack type:
//   - string: substring check
//   - slice/array: checks if any element equals needle
//   - map: checks if needle is a key in the map
//
// Example:
//
//	assert.Contains(t, "hello world", "world")
//	assert.Contains(t, []int{1, 2, 3}, 2)
//	assert.Contains(t, map[string]int{"a": 1}, "a")
func Contains(t testing.TB, haystack, needle any, msgAndArgs ...any) {
	t.Helper()
	if !contains(haystack, needle) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%#v does not contain %#v\n%s", haystack, needle, msg)
		} else {
			t.Fatalf("%#v does not contain %#v", haystack, needle)
		}
	}
}

// NotContains asserts that haystack does not contain needle.
func NotContains(t testing.TB, haystack, needle any, msgAndArgs ...any) {
	t.Helper()
	if contains(haystack, needle) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%#v should not contain %#v\n%s", haystack, needle, msg)
		} else {
			t.Fatalf("%#v should not contain %#v", haystack, needle)
		}
	}
}

// Len asserts that v has the expected length.
//
// Works with any type that has a length: strings, slices, arrays, maps, channels.
//
// Example:
//
//	assert.Len(t, []int{1, 2, 3}, 3)
//	assert.Len(t, "hello", 5)
//	assert.Len(t, map[string]int{"a": 1, "b": 2}, 2)
func Len(t testing.TB, v any, want int, msgAndArgs ...any) {
	t.Helper()
	got := reflect.ValueOf(v).Len()
	if got != want {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected length %d, got %d\n%s", want, got, msg)
		} else {
			t.Fatalf("expected length %d, got %d", want, got)
		}
	}
}

// Empty asserts that v is empty (nil, zero length, or zero value).
//
// Checks for emptiness based on type:
//   - nil values: always empty
//   - strings, slices, maps, channels: empty if length is 0
//   - pointers: empty if nil or pointing to empty value
//   - other types: empty if zero value (via reflect.Value.IsZero)
//
// Example:
//
//	assert.Empty(t, "")
//	assert.Empty(t, []int{})
//	assert.Empty(t, nil)
func Empty(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if !isEmpty(v) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected empty, got: %#v\n%s", v, msg)
		} else {
			t.Fatalf("expected empty, got: %#v", v)
		}
	}
}

// NotEmpty asserts that v is not empty.
func NotEmpty(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if isEmpty(v) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected non-empty value\n%s", msg)
		} else {
			t.Fatalf("expected non-empty value")
		}
	}
}

// Panics asserts that f panics.
func Panics(t testing.TB, f func(), msgAndArgs ...any) {
	t.Helper()
	defer func() {
		if recover() == nil {
			msg := formatMsg(msgAndArgs...)
			if msg != "" {
				t.Fatalf("expected panic\n%s", msg)
			} else {
				t.Fatalf("expected panic")
			}
		}
	}()
	f()
}

// NotPanics asserts that f does not panic.
func NotPanics(t testing.TB, f func(), msgAndArgs ...any) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg := formatMsg(msgAndArgs...)
			if msg != "" {
				t.Fatalf("unexpected panic: %v\n%s", r, msg)
			} else {
				t.Fatalf("unexpected panic: %v", r)
			}
		}
	}()
	f()
}

// Greater asserts that a > b.
func Greater[T cmp.Ordered](t testing.TB, a, b T, msgAndArgs ...any) {
	t.Helper()
	if !(a > b) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected %v > %v\n%s", a, b, msg)
		} else {
			t.Fatalf("expected %v > %v", a, b)
		}
	}
}

// GreaterOrEqual asserts that a >= b.
func GreaterOrEqual[T cmp.Ordered](t testing.TB, a, b T, msgAndArgs ...any) {
	t.Helper()
	if !(a >= b) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected %v >= %v\n%s", a, b, msg)
		} else {
			t.Fatalf("expected %v >= %v", a, b)
		}
	}
}

// Less asserts that a < b.
func Less[T cmp.Ordered](t testing.TB, a, b T, msgAndArgs ...any) {
	t.Helper()
	if !(a < b) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected %v < %v\n%s", a, b, msg)
		} else {
			t.Fatalf("expected %v < %v", a, b)
		}
	}
}

// LessOrEqual asserts that a <= b.
func LessOrEqual[T cmp.Ordered](t testing.TB, a, b T, msgAndArgs ...any) {
	t.Helper()
	if !(a <= b) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected %v <= %v\n%s", a, b, msg)
		} else {
			t.Fatalf("expected %v <= %v", a, b)
		}
	}
}

// InDelta asserts that two numbers are within delta of each other.
//
// This is the preferred way to compare floating-point numbers, since direct
// equality comparison is unreliable due to rounding errors.
//
// Example:
//
//	result := math.Pi * 2
//	assert.InDelta(t, result, 6.283, 0.001)
func InDelta(t testing.TB, expected, actual, delta float64, msgAndArgs ...any) {
	t.Helper()
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	if diff > delta {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected %v and %v to be within %v, but difference was %v\n%s", expected, actual, delta, diff, msg)
		} else {
			t.Fatalf("expected %v and %v to be within %v, but difference was %v", expected, actual, delta, diff)
		}
	}
}

// Regexp asserts that str matches the regular expression pattern.
//
// The pattern can be either a string (which will be compiled) or a
// pre-compiled *regexp.Regexp. Uses regexp.MatchString for matching.
//
// Example:
//
//	assert.Regexp(t, `^\d{3}-\d{4}$`, phoneNumber)
//	assert.Regexp(t, regexp.MustCompile(`[A-Z][a-z]+`), "Hello")
func Regexp(t testing.TB, pattern any, str string, msgAndArgs ...any) {
	t.Helper()
	var re *regexp.Regexp
	switch p := pattern.(type) {
	case *regexp.Regexp:
		re = p
	case string:
		re = regexp.MustCompile(p)
	default:
		t.Fatalf("pattern must be string or *regexp.Regexp, got %T", pattern)
		return
	}
	if !re.MatchString(str) {
		msg := formatMsg(msgAndArgs...)
		if msg != "" {
			t.Fatalf("expected %q to match pattern %q\n%s", str, re.String(), msg)
		} else {
			t.Fatalf("expected %q to match pattern %q", str, re.String())
		}
	}
}

// --- Helpers ---

// formatDiff formats a go-cmp diff string with optional color coding.
// Lines starting with - are colored red (removed/expected).
// Lines starting with + are colored green (added/actual).
func formatDiff(diff string) string {
	if !colorEnabled {
		return "mismatch (-want +got):\n" + diff
	}
	var b strings.Builder
	b.WriteString("mismatch (-want +got):\n")
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "-"):
			b.WriteString(color.Red.Apply(line))
		case strings.HasPrefix(line, "+"):
			b.WriteString(color.Green.Apply(line))
		default:
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// formatMsg formats optional message arguments for assertion failures.
// Supports:
//   - No args: returns ""
//   - Single string: returns as-is
//   - Single non-string: formats with %v
//   - String + args: treats first arg as format string for fmt.Sprintf
//   - Non-string + args: returns "" (invalid format)
func formatMsg(msgAndArgs ...any) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	if len(msgAndArgs) == 1 {
		if s, ok := msgAndArgs[0].(string); ok {
			return s
		}
		return fmt.Sprintf("%v", msgAndArgs[0])
	}
	if s, ok := msgAndArgs[0].(string); ok {
		return fmt.Sprintf(s, msgAndArgs[1:]...)
	}
	return ""
}

// isNil checks if a value is nil, handling both interface nil and typed nil.
// Returns true for:
//   - nil interface
//   - nil pointer, chan, func, interface, map, or slice
// Returns false for non-nillable types (int, bool, struct, etc.)
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

// isEmpty checks if a value is empty according to assertion semantics.
// Returns true for:
//   - nil values
//   - zero-length strings, slices, maps, or channels
//   - nil or zero-value pointers
//   - zero values for other types (via reflect.IsZero)
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Map, reflect.Slice:
		return rv.Len() == 0
	case reflect.String:
		return rv.Len() == 0
	case reflect.Ptr:
		if rv.IsNil() {
			return true
		}
		return isEmpty(rv.Elem().Interface())
	}
	return rv.IsZero()
}

// contains checks if haystack contains needle based on the haystack type.
// For strings: substring check
// For slices/arrays: element equality check using go-cmp
// For maps: key existence check
func contains(haystack, needle any) bool {
	hv := reflect.ValueOf(haystack)
	switch hv.Kind() {
	case reflect.String:
		return strings.Contains(hv.String(), reflect.ValueOf(needle).String())
	case reflect.Slice, reflect.Array:
		for i := 0; i < hv.Len(); i++ {
			if gocmp.Equal(hv.Index(i).Interface(), needle) {
				return true
			}
		}
	case reflect.Map:
		for _, k := range hv.MapKeys() {
			if gocmp.Equal(k.Interface(), needle) {
				return true
			}
		}
	}
	return false
}
