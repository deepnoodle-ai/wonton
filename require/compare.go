package require

import (
	"cmp"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Greater asserts that e1 > e2.
func Greater[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Greater(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

// GreaterOrEqual asserts that e1 >= e2.
func GreaterOrEqual[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.GreaterOrEqual(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

// Less asserts that e1 < e2.
func Less[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Less(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

// LessOrEqual asserts that e1 <= e2.
func LessOrEqual[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.LessOrEqual(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

// Positive asserts that the specified value is positive (> 0).
func Positive[T cmp.Ordered](t TestingT, e T, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Positive(t, e, msgAndArgs...) {
		t.FailNow()
	}
}

// Negative asserts that the specified value is negative (< 0).
func Negative[T cmp.Ordered](t TestingT, e T, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Negative(t, e, msgAndArgs...) {
		t.FailNow()
	}
}

// InDelta asserts that the two numerals are within delta of each other.
func InDelta(t TestingT, expected, actual, delta float64, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.InDelta(t, expected, actual, delta, msgAndArgs...) {
		t.FailNow()
	}
}

// InEpsilon asserts that expected and actual have a relative error less than epsilon.
func InEpsilon(t TestingT, expected, actual, epsilon float64, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.InEpsilon(t, expected, actual, epsilon, msgAndArgs...) {
		t.FailNow()
	}
}

// WithinDuration asserts that the two times are within duration delta of each other.
func WithinDuration(t TestingT, expected, actual time.Time, delta time.Duration, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.WithinDuration(t, expected, actual, delta, msgAndArgs...) {
		t.FailNow()
	}
}

// WithinRange asserts that a time is within a time range (inclusive).
func WithinRange(t TestingT, actual, start, end time.Time, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.WithinRange(t, actual, start, end, msgAndArgs...) {
		t.FailNow()
	}
}
