package assert

import (
	"cmp"
	"fmt"
	"math"
	"time"
)

// Greater asserts that e1 > e2.
func Greater[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) {
	helper(t)
	require(t, checkGreater(t, e1, e2, msgAndArgs...))
}

// GreaterOrEqual asserts that e1 >= e2.
func GreaterOrEqual[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) {
	helper(t)
	require(t, checkGreaterOrEqual(t, e1, e2, msgAndArgs...))
}

// Less asserts that e1 < e2.
func Less[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) {
	helper(t)
	require(t, checkLess(t, e1, e2, msgAndArgs...))
}

// LessOrEqual asserts that e1 <= e2.
func LessOrEqual[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) {
	helper(t)
	require(t, checkLessOrEqual(t, e1, e2, msgAndArgs...))
}

// Positive asserts that the specified value is positive (> 0).
func Positive[T cmp.Ordered](t TestingT, e T, msgAndArgs ...any) {
	helper(t)
	require(t, checkPositive(t, e, msgAndArgs...))
}

// Negative asserts that the specified value is negative (< 0).
func Negative[T cmp.Ordered](t TestingT, e T, msgAndArgs ...any) {
	helper(t)
	require(t, checkNegative(t, e, msgAndArgs...))
}

// InDelta asserts that the two numerals are within delta of each other.
func InDelta(t TestingT, expected, actual, delta float64, msgAndArgs ...any) {
	helper(t)
	require(t, checkInDelta(t, expected, actual, delta, msgAndArgs...))
}

// InEpsilon asserts that expected and actual have a relative error less than epsilon.
func InEpsilon(t TestingT, expected, actual, epsilon float64, msgAndArgs ...any) {
	helper(t)
	require(t, checkInEpsilon(t, expected, actual, epsilon, msgAndArgs...))
}

// WithinDuration asserts that the two times are within duration delta of each other.
func WithinDuration(t TestingT, expected, actual time.Time, delta time.Duration, msgAndArgs ...any) {
	helper(t)
	require(t, checkWithinDuration(t, expected, actual, delta, msgAndArgs...))
}

// WithinRange asserts that a time is within a time range (inclusive).
func WithinRange(t TestingT, actual, start, end time.Time, msgAndArgs ...any) {
	helper(t)
	require(t, checkWithinRange(t, actual, start, end, msgAndArgs...))
}

// --- Internal check functions ---

func checkGreater[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) bool {
	helper(t)
	if e1 > e2 {
		return true
	}
	return fail(t, fmt.Sprintf("%v is not greater than %v", e1, e2), msgAndArgs...)
}

func checkGreaterOrEqual[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) bool {
	helper(t)
	if e1 >= e2 {
		return true
	}
	return fail(t, fmt.Sprintf("%v is not greater than or equal to %v", e1, e2), msgAndArgs...)
}

func checkLess[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) bool {
	helper(t)
	if e1 < e2 {
		return true
	}
	return fail(t, fmt.Sprintf("%v is not less than %v", e1, e2), msgAndArgs...)
}

func checkLessOrEqual[T cmp.Ordered](t TestingT, e1, e2 T, msgAndArgs ...any) bool {
	helper(t)
	if e1 <= e2 {
		return true
	}
	return fail(t, fmt.Sprintf("%v is not less than or equal to %v", e1, e2), msgAndArgs...)
}

func checkPositive[T cmp.Ordered](t TestingT, e T, msgAndArgs ...any) bool {
	helper(t)
	var zero T
	if e > zero {
		return true
	}
	return fail(t, fmt.Sprintf("%v is not positive", e), msgAndArgs...)
}

func checkNegative[T cmp.Ordered](t TestingT, e T, msgAndArgs ...any) bool {
	helper(t)
	var zero T
	if e < zero {
		return true
	}
	return fail(t, fmt.Sprintf("%v is not negative", e), msgAndArgs...)
}

func checkInDelta(t TestingT, expected, actual, delta float64, msgAndArgs ...any) bool {
	helper(t)
	if math.IsNaN(expected) && math.IsNaN(actual) {
		return true
	}
	if math.IsNaN(expected) {
		return fail(t, "Expected must not be NaN", msgAndArgs...)
	}
	if math.IsNaN(actual) {
		return fail(t, fmt.Sprintf("Expected %v with delta %v, but was NaN", expected, delta), msgAndArgs...)
	}
	diff := expected - actual
	if diff < -delta || diff > delta {
		return fail(t, fmt.Sprintf("Max difference between %v and %v allowed is %v, but difference was %v", expected, actual, delta, diff), msgAndArgs...)
	}
	return true
}

func checkInEpsilon(t TestingT, expected, actual, epsilon float64, msgAndArgs ...any) bool {
	helper(t)
	if math.IsNaN(epsilon) {
		return fail(t, "Epsilon must not be NaN", msgAndArgs...)
	}
	if math.IsNaN(expected) && math.IsNaN(actual) {
		return true
	}
	if math.IsNaN(expected) {
		return fail(t, "Expected must not be NaN", msgAndArgs...)
	}
	if expected == 0 {
		return fail(t, "Expected must have a value other than zero to calculate relative error", msgAndArgs...)
	}
	if math.IsNaN(actual) {
		return fail(t, "Actual must not be NaN", msgAndArgs...)
	}
	relativeError := math.Abs(expected-actual) / math.Abs(expected)
	if relativeError > epsilon {
		return fail(t, fmt.Sprintf("Relative error is too high: %v (expected) < %v (actual)", epsilon, relativeError), msgAndArgs...)
	}
	return true
}

func checkWithinDuration(t TestingT, expected, actual time.Time, delta time.Duration, msgAndArgs ...any) bool {
	helper(t)
	dt := expected.Sub(actual)
	if dt < -delta || dt > delta {
		return fail(t, fmt.Sprintf("Max difference between %v and %v allowed is %v, but difference was %v", expected, actual, delta, dt), msgAndArgs...)
	}
	return true
}

func checkWithinRange(t TestingT, actual, start, end time.Time, msgAndArgs ...any) bool {
	helper(t)
	if end.Before(start) {
		return fail(t, "Start should be before end", msgAndArgs...)
	}
	if actual.Before(start) {
		return fail(t, fmt.Sprintf("Time %v expected to be in range %v to %v, but is before the range", actual, start, end), msgAndArgs...)
	}
	if actual.After(end) {
		return fail(t, fmt.Sprintf("Time %v expected to be in range %v to %v, but is after the range", actual, start, end), msgAndArgs...)
	}
	return true
}
