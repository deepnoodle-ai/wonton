// Package assert provides minimal test assertions with excellent diff output.
//
// Built on go-cmp for comparisons, with colored unified diffs on failure.
// All assertions fail immediately (fatal).
//
// Example:
//
//	func TestUser(t *testing.T) {
//	    got := fetchUser()
//	    assert.Equal(t, got, want)
//	    assert.NoError(t, err)
//	}
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

// SetColorEnabled enables or disables colored output.
func SetColorEnabled(enabled bool) { colorEnabled = enabled }

// defaultOpts are applied to all Equal comparisons.
// - EquateErrors allows comparing error types safely
// - Exporter allows comparing unexported fields (like reflect.DeepEqual)
var defaultOpts = []gocmp.Option{
	cmpopts.EquateErrors(),
	gocmp.Exporter(func(reflect.Type) bool { return true }),
}

// Equal asserts that got and want are deeply equal using go-cmp.
// Shows a colored unified diff on failure.
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

// ErrorContains asserts that err contains the substring.
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
// Works with strings, slices, arrays, and maps (checks keys).
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
// pattern can be a string or *regexp.Regexp.
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
