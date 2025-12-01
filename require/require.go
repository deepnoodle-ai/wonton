// Package require provides test assertions that fail immediately.
//
// This package wraps the assert package but calls t.FailNow() on failure,
// stopping the test execution immediately rather than continuing.
//
// Example usage:
//
//	func TestSomething(t *testing.T) {
//		require.NoError(t, err)
//		require.Equal(t, expected, actual)
//	}
package require

import (
	"github.com/deepnoodle-ai/gooey/assert"
)

// TestingT is the interface that testing.T implements.
type TestingT interface {
	Errorf(format string, args ...any)
	FailNow()
}

// tHelper is the interface for t.Helper().
type tHelper interface {
	Helper()
}

// Fail reports a test failure with the given message and stops execution.
func Fail(t TestingT, failureMessage string, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Fail(t, failureMessage, msgAndArgs...) {
		t.FailNow()
	}
}

// FailNow fails the test and stops execution.
func FailNow(t TestingT, failureMessage string, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	assert.FailNow(t, failureMessage, msgAndArgs...)
}

// Equal asserts that expected and actual are deeply equal.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Equal(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// NotEqual asserts that expected and actual are not deeply equal.
func NotEqual(t TestingT, expected, actual any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotEqual(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// Same asserts that two pointers reference the same object.
func Same(t TestingT, expected, actual any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Same(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// NotSame asserts that two pointers do not reference the same object.
func NotSame(t TestingT, expected, actual any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotSame(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// Nil asserts that the specified object is nil.
func Nil(t TestingT, object any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Nil(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// NotNil asserts that the specified object is not nil.
func NotNil(t TestingT, object any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotNil(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// True asserts that the specified value is true.
func True(t TestingT, value bool, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.True(t, value, msgAndArgs...) {
		t.FailNow()
	}
}

// False asserts that the specified value is false.
func False(t TestingT, value bool, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.False(t, value, msgAndArgs...) {
		t.FailNow()
	}
}

// Empty asserts that the specified object is empty.
func Empty(t TestingT, object any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Empty(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// NotEmpty asserts that the specified object is not empty.
func NotEmpty(t TestingT, object any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotEmpty(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

// Len asserts that the specified object has the expected length.
func Len(t TestingT, object any, length int, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Len(t, object, length, msgAndArgs...) {
		t.FailNow()
	}
}

// Contains asserts that the specified container contains the specified element.
func Contains(t TestingT, container, element any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Contains(t, container, element, msgAndArgs...) {
		t.FailNow()
	}
}

// NotContains asserts that the specified container does not contain the specified element.
func NotContains(t TestingT, container, element any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotContains(t, container, element, msgAndArgs...) {
		t.FailNow()
	}
}

// Zero asserts that the specified value is the zero value for its type.
func Zero(t TestingT, value any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Zero(t, value, msgAndArgs...) {
		t.FailNow()
	}
}

// NotZero asserts that the specified value is not the zero value for its type.
func NotZero(t TestingT, value any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotZero(t, value, msgAndArgs...) {
		t.FailNow()
	}
}

// Panics asserts that the specified function panics.
func Panics(t TestingT, f func(), msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Panics(t, f, msgAndArgs...) {
		t.FailNow()
	}
}

// PanicsWithValue asserts that the function panics with the expected value.
func PanicsWithValue(t TestingT, expected any, f func(), msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.PanicsWithValue(t, expected, f, msgAndArgs...) {
		t.FailNow()
	}
}

// PanicsWithError asserts that the function panics with an error matching the expected string.
func PanicsWithError(t TestingT, errString string, f func(), msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.PanicsWithError(t, errString, f, msgAndArgs...) {
		t.FailNow()
	}
}

// NotPanics asserts that the specified function does not panic.
func NotPanics(t TestingT, f func(), msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotPanics(t, f, msgAndArgs...) {
		t.FailNow()
	}
}

// Regexp asserts that the specified string matches the regexp.
func Regexp(t TestingT, rx any, str any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Regexp(t, rx, str, msgAndArgs...) {
		t.FailNow()
	}
}

// NotRegexp asserts that the specified string does not match the regexp.
func NotRegexp(t TestingT, rx any, str any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotRegexp(t, rx, str, msgAndArgs...) {
		t.FailNow()
	}
}

// IsType asserts that the specified objects are of the same type.
func IsType(t TestingT, expectedType, object any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.IsType(t, expectedType, object, msgAndArgs...) {
		t.FailNow()
	}
}

// Implements asserts that an object implements the specified interface.
func Implements(t TestingT, interfaceObject, object any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Implements(t, interfaceObject, object, msgAndArgs...) {
		t.FailNow()
	}
}

// ElementsMatch asserts that two slices contain the same elements, ignoring order.
func ElementsMatch(t TestingT, listA, listB any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.ElementsMatch(t, listA, listB, msgAndArgs...) {
		t.FailNow()
	}
}

// Subset asserts that subset is a subset of list.
func Subset(t TestingT, list, subset any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Subset(t, list, subset, msgAndArgs...) {
		t.FailNow()
	}
}
