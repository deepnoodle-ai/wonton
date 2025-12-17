package assert

import (
	"time"
)

// NonFatalAssertions provides assertions that don't stop test execution on failure.
// Use NonFatal(t) to get an instance.
type NonFatalAssertions struct {
	t TestingT
}

// NonFatal returns an assertion wrapper that doesn't call FailNow on failure.
// Use this when you want to collect multiple failures before stopping.
//
//	assert.NonFatal(t).Equal(expected, actual)
//	assert.NonFatal(t).NoError(err)
func NonFatal(t TestingT) *NonFatalAssertions {
	return &NonFatalAssertions{t: t}
}

// Fail reports a test failure (does not stop execution). Returns false.
func (a *NonFatalAssertions) Fail(failureMessage string, msgAndArgs ...any) bool {
	helper(a.t)
	return fail(a.t, failureMessage, msgAndArgs...)
}

// Equal asserts that expected and actual are deeply equal.
func (a *NonFatalAssertions) Equal(expected, actual any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkEqual(a.t, expected, actual, msgAndArgs...)
}

// NotEqual asserts that expected and actual are not deeply equal.
func (a *NonFatalAssertions) NotEqual(expected, actual any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotEqual(a.t, expected, actual, msgAndArgs...)
}

// Same asserts that two pointers reference the same object.
func (a *NonFatalAssertions) Same(expected, actual any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkSame(a.t, expected, actual, msgAndArgs...)
}

// NotSame asserts that two pointers do not reference the same object.
func (a *NonFatalAssertions) NotSame(expected, actual any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotSame(a.t, expected, actual, msgAndArgs...)
}

// Nil asserts that the specified object is nil.
func (a *NonFatalAssertions) Nil(object any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNil(a.t, object, msgAndArgs...)
}

// NotNil asserts that the specified object is not nil.
func (a *NonFatalAssertions) NotNil(object any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotNil(a.t, object, msgAndArgs...)
}

// True asserts that the specified value is true.
func (a *NonFatalAssertions) True(value bool, msgAndArgs ...any) bool {
	helper(a.t)
	return checkTrue(a.t, value, msgAndArgs...)
}

// False asserts that the specified value is false.
func (a *NonFatalAssertions) False(value bool, msgAndArgs ...any) bool {
	helper(a.t)
	return checkFalse(a.t, value, msgAndArgs...)
}

// Empty asserts that the specified object is empty.
func (a *NonFatalAssertions) Empty(object any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkEmpty(a.t, object, msgAndArgs...)
}

// NotEmpty asserts that the specified object is not empty.
func (a *NonFatalAssertions) NotEmpty(object any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotEmpty(a.t, object, msgAndArgs...)
}

// Len asserts that the specified object has the expected length.
func (a *NonFatalAssertions) Len(object any, length int, msgAndArgs ...any) bool {
	helper(a.t)
	return checkLen(a.t, object, length, msgAndArgs...)
}

// Contains asserts that the specified container contains the specified element.
func (a *NonFatalAssertions) Contains(container, element any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkContains(a.t, container, element, msgAndArgs...)
}

// NotContains asserts that the specified container does not contain the specified element.
func (a *NonFatalAssertions) NotContains(container, element any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotContains(a.t, container, element, msgAndArgs...)
}

// Zero asserts that the specified value is the zero value for its type.
func (a *NonFatalAssertions) Zero(value any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkZero(a.t, value, msgAndArgs...)
}

// NotZero asserts that the specified value is not the zero value for its type.
func (a *NonFatalAssertions) NotZero(value any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotZero(a.t, value, msgAndArgs...)
}

// Panics asserts that the specified function panics.
func (a *NonFatalAssertions) Panics(f func(), msgAndArgs ...any) bool {
	helper(a.t)
	return checkPanics(a.t, f, msgAndArgs...)
}

// PanicsWithValue asserts that the function panics with the expected value.
func (a *NonFatalAssertions) PanicsWithValue(expected any, f func(), msgAndArgs ...any) bool {
	helper(a.t)
	return checkPanicsWithValue(a.t, expected, f, msgAndArgs...)
}

// PanicsWithError asserts that the function panics with an error matching the expected string.
func (a *NonFatalAssertions) PanicsWithError(errString string, f func(), msgAndArgs ...any) bool {
	helper(a.t)
	return checkPanicsWithError(a.t, errString, f, msgAndArgs...)
}

// NotPanics asserts that the specified function does not panic.
func (a *NonFatalAssertions) NotPanics(f func(), msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotPanics(a.t, f, msgAndArgs...)
}

// Regexp asserts that the specified string matches the regexp.
func (a *NonFatalAssertions) Regexp(rx any, str any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkRegexp(a.t, rx, str, msgAndArgs...)
}

// NotRegexp asserts that the specified string does not match the regexp.
func (a *NonFatalAssertions) NotRegexp(rx any, str any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotRegexp(a.t, rx, str, msgAndArgs...)
}

// IsType asserts that the specified objects are of the same type.
func (a *NonFatalAssertions) IsType(expectedType, object any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkIsType(a.t, expectedType, object, msgAndArgs...)
}

// Implements asserts that an object implements the specified interface.
func (a *NonFatalAssertions) Implements(interfaceObject, object any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkImplements(a.t, interfaceObject, object, msgAndArgs...)
}

// ElementsMatch asserts that two slices contain the same elements, ignoring order.
func (a *NonFatalAssertions) ElementsMatch(listA, listB any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkElementsMatch(a.t, listA, listB, msgAndArgs...)
}

// Subset asserts that subset is a subset of list.
func (a *NonFatalAssertions) Subset(list, subset any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkSubset(a.t, list, subset, msgAndArgs...)
}

// --- Error assertions ---

// NoError asserts that a function returned a nil error.
func (a *NonFatalAssertions) NoError(err error, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNoError(a.t, err, msgAndArgs...)
}

// Error asserts that a function returned a non-nil error.
func (a *NonFatalAssertions) Error(err error, msgAndArgs ...any) bool {
	helper(a.t)
	return checkError(a.t, err, msgAndArgs...)
}

// EqualError asserts that an error's message equals the expected string.
func (a *NonFatalAssertions) EqualError(theError error, errString string, msgAndArgs ...any) bool {
	helper(a.t)
	return checkEqualError(a.t, theError, errString, msgAndArgs...)
}

// ErrorContains asserts that an error's message contains the expected substring.
func (a *NonFatalAssertions) ErrorContains(theError error, contains string, msgAndArgs ...any) bool {
	helper(a.t)
	return checkErrorContains(a.t, theError, contains, msgAndArgs...)
}

// ErrorIs asserts that at least one of the errors in err's chain matches target.
func (a *NonFatalAssertions) ErrorIs(err, target error, msgAndArgs ...any) bool {
	helper(a.t)
	return checkErrorIs(a.t, err, target, msgAndArgs...)
}

// NotErrorIs asserts that none of the errors in err's chain matches target.
func (a *NonFatalAssertions) NotErrorIs(err, target error, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotErrorIs(a.t, err, target, msgAndArgs...)
}

// ErrorAs asserts that at least one of the errors in err's chain matches target.
func (a *NonFatalAssertions) ErrorAs(err error, target any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkErrorAs(a.t, err, target, msgAndArgs...)
}

// NotErrorAs asserts that none of the errors in err's chain matches target.
func (a *NonFatalAssertions) NotErrorAs(err error, target any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkNotErrorAs(a.t, err, target, msgAndArgs...)
}

// --- Comparison assertions ---

// Greater asserts that e1 > e2.
// Note: This uses reflection. For type safety with generics, use assert.Greater() directly.
func (a *NonFatalAssertions) Greater(e1, e2 any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkOrderedComparison(a.t, e1, e2, ">", msgAndArgs...)
}

// GreaterOrEqual asserts that e1 >= e2.
func (a *NonFatalAssertions) GreaterOrEqual(e1, e2 any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkOrderedComparison(a.t, e1, e2, ">=", msgAndArgs...)
}

// Less asserts that e1 < e2.
func (a *NonFatalAssertions) Less(e1, e2 any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkOrderedComparison(a.t, e1, e2, "<", msgAndArgs...)
}

// LessOrEqual asserts that e1 <= e2.
func (a *NonFatalAssertions) LessOrEqual(e1, e2 any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkOrderedComparison(a.t, e1, e2, "<=", msgAndArgs...)
}

// Positive asserts that the specified value is positive (> 0).
func (a *NonFatalAssertions) Positive(e any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkOrderedToZero(a.t, e, ">", msgAndArgs...)
}

// Negative asserts that the specified value is negative (< 0).
func (a *NonFatalAssertions) Negative(e any, msgAndArgs ...any) bool {
	helper(a.t)
	return checkOrderedToZero(a.t, e, "<", msgAndArgs...)
}

// InDelta asserts that the two numerals are within delta of each other.
func (a *NonFatalAssertions) InDelta(expected, actual, delta float64, msgAndArgs ...any) bool {
	helper(a.t)
	return checkInDelta(a.t, expected, actual, delta, msgAndArgs...)
}

// InEpsilon asserts that expected and actual have a relative error less than epsilon.
func (a *NonFatalAssertions) InEpsilon(expected, actual, epsilon float64, msgAndArgs ...any) bool {
	helper(a.t)
	return checkInEpsilon(a.t, expected, actual, epsilon, msgAndArgs...)
}

// WithinDuration asserts that the two times are within duration delta of each other.
func (a *NonFatalAssertions) WithinDuration(expected, actual time.Time, delta time.Duration, msgAndArgs ...any) bool {
	helper(a.t)
	return checkWithinDuration(a.t, expected, actual, delta, msgAndArgs...)
}

// WithinRange asserts that a time is within a time range (inclusive).
func (a *NonFatalAssertions) WithinRange(actual, start, end time.Time, msgAndArgs ...any) bool {
	helper(a.t)
	return checkWithinRange(a.t, actual, start, end, msgAndArgs...)
}
