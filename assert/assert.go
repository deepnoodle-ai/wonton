// Package assert provides test assertions for Go.
//
// By default, assertions fail immediately (like testify/require).
// Use NonFatal(t) for assertions that allow the test to continue.
//
// Example usage:
//
//	func TestSomething(t *testing.T) {
//		assert.Equal(t, expected, actual)  // fails immediately
//		assert.NoError(t, err)             // fails immediately
//
//		// To continue on failure:
//		assert.NonFatal(t).Equal(expected, actual)
//	}
package assert

import (
	"fmt"
	"reflect"
	"strings"
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

// helper calls t.Helper() if available.
func helper(t TestingT) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
}

// fail reports a test failure with the given message (does not stop execution).
func fail(t TestingT, failureMessage string, msgAndArgs ...any) bool {
	helper(t)
	content := []labeledContent{
		{"Error Trace", strings.Join(callerInfo(), "\n\t\t\t")},
		{"Error", failureMessage},
	}
	if n, ok := t.(interface{ Name() string }); ok {
		content = append(content, labeledContent{"Test", n.Name()})
	}
	if message := formatMessage(msgAndArgs...); message != "" {
		content = append(content, labeledContent{"Messages", message})
	}
	t.Errorf("\n%s", formatLabeledOutput(content...))
	return false
}

// require calls FailNow if ok is false.
func require(t TestingT, ok bool) {
	if !ok {
		t.FailNow()
	}
}

// Fail reports a test failure and stops execution.
func Fail(t TestingT, failureMessage string, msgAndArgs ...any) {
	helper(t)
	fail(t, failureMessage, msgAndArgs...)
	t.FailNow()
}

// Equal asserts that expected and actual are deeply equal.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) {
	helper(t)
	require(t, checkEqual(t, expected, actual, msgAndArgs...))
}

// NotEqual asserts that expected and actual are not deeply equal.
func NotEqual(t TestingT, expected, actual any, msgAndArgs ...any) {
	helper(t)
	require(t, checkNotEqual(t, expected, actual, msgAndArgs...))
}

// Same asserts that two pointers reference the same object.
func Same(t TestingT, expected, actual any, msgAndArgs ...any) {
	helper(t)
	require(t, checkSame(t, expected, actual, msgAndArgs...))
}

// NotSame asserts that two pointers do not reference the same object.
func NotSame(t TestingT, expected, actual any, msgAndArgs ...any) {
	helper(t)
	require(t, checkNotSame(t, expected, actual, msgAndArgs...))
}

// Nil asserts that the specified object is nil.
func Nil(t TestingT, object any, msgAndArgs ...any) {
	helper(t)
	require(t, checkNil(t, object, msgAndArgs...))
}

// NotNil asserts that the specified object is not nil.
func NotNil(t TestingT, object any, msgAndArgs ...any) {
	helper(t)
	require(t, checkNotNil(t, object, msgAndArgs...))
}

// True asserts that the specified value is true.
func True(t TestingT, value bool, msgAndArgs ...any) {
	helper(t)
	require(t, checkTrue(t, value, msgAndArgs...))
}

// False asserts that the specified value is false.
func False(t TestingT, value bool, msgAndArgs ...any) {
	helper(t)
	require(t, checkFalse(t, value, msgAndArgs...))
}

// Empty asserts that the specified object is empty (zero value, nil, or has zero length).
func Empty(t TestingT, object any, msgAndArgs ...any) {
	helper(t)
	require(t, checkEmpty(t, object, msgAndArgs...))
}

// NotEmpty asserts that the specified object is not empty.
func NotEmpty(t TestingT, object any, msgAndArgs ...any) {
	helper(t)
	require(t, checkNotEmpty(t, object, msgAndArgs...))
}

// Len asserts that the specified object has the expected length.
func Len(t TestingT, object any, length int, msgAndArgs ...any) {
	helper(t)
	require(t, checkLen(t, object, length, msgAndArgs...))
}

// Contains asserts that the specified container contains the specified element.
// Works with strings, arrays, slices, and maps.
func Contains(t TestingT, container, element any, msgAndArgs ...any) {
	helper(t)
	require(t, checkContains(t, container, element, msgAndArgs...))
}

// NotContains asserts that the specified container does not contain the specified element.
func NotContains(t TestingT, container, element any, msgAndArgs ...any) {
	helper(t)
	require(t, checkNotContains(t, container, element, msgAndArgs...))
}

// Zero asserts that the specified value is the zero value for its type.
func Zero(t TestingT, value any, msgAndArgs ...any) {
	helper(t)
	require(t, checkZero(t, value, msgAndArgs...))
}

// NotZero asserts that the specified value is not the zero value for its type.
func NotZero(t TestingT, value any, msgAndArgs ...any) {
	helper(t)
	require(t, checkNotZero(t, value, msgAndArgs...))
}

// Panics asserts that the specified function panics.
func Panics(t TestingT, f func(), msgAndArgs ...any) {
	helper(t)
	require(t, checkPanics(t, f, msgAndArgs...))
}

// PanicsWithValue asserts that the function panics with the expected value.
func PanicsWithValue(t TestingT, expected any, f func(), msgAndArgs ...any) {
	helper(t)
	require(t, checkPanicsWithValue(t, expected, f, msgAndArgs...))
}

// PanicsWithError asserts that the function panics with an error matching the expected string.
func PanicsWithError(t TestingT, errString string, f func(), msgAndArgs ...any) {
	helper(t)
	require(t, checkPanicsWithError(t, errString, f, msgAndArgs...))
}

// NotPanics asserts that the specified function does not panic.
func NotPanics(t TestingT, f func(), msgAndArgs ...any) {
	helper(t)
	require(t, checkNotPanics(t, f, msgAndArgs...))
}

// Regexp asserts that the specified string matches the regexp.
func Regexp(t TestingT, rx any, str any, msgAndArgs ...any) {
	helper(t)
	require(t, checkRegexp(t, rx, str, msgAndArgs...))
}

// NotRegexp asserts that the specified string does not match the regexp.
func NotRegexp(t TestingT, rx any, str any, msgAndArgs ...any) {
	helper(t)
	require(t, checkNotRegexp(t, rx, str, msgAndArgs...))
}

// IsType asserts that the specified objects are of the same type.
func IsType(t TestingT, expectedType, object any, msgAndArgs ...any) {
	helper(t)
	require(t, checkIsType(t, expectedType, object, msgAndArgs...))
}

// Implements asserts that an object implements the specified interface.
func Implements(t TestingT, interfaceObject, object any, msgAndArgs ...any) {
	helper(t)
	require(t, checkImplements(t, interfaceObject, object, msgAndArgs...))
}

// ElementsMatch asserts that two slices contain the same elements, ignoring order.
func ElementsMatch(t TestingT, listA, listB any, msgAndArgs ...any) {
	helper(t)
	require(t, checkElementsMatch(t, listA, listB, msgAndArgs...))
}

// Subset asserts that subset is a subset of list.
func Subset(t TestingT, list, subset any, msgAndArgs ...any) {
	helper(t)
	require(t, checkSubset(t, list, subset, msgAndArgs...))
}

// --- Internal check functions (return bool, don't call FailNow) ---

func checkEqual(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	helper(t)
	if err := validateEqualArgs(expected, actual); err != nil {
		return fail(t, fmt.Sprintf("Invalid operation: %#v == %#v (%s)", expected, actual, err), msgAndArgs...)
	}
	if !objectsAreEqual(expected, actual) {
		diff := formatDiff(expected, actual)
		expectedStr, actualStr := formatUnequalValues(expected, actual)
		return fail(t, fmt.Sprintf("Not equal:\n\texpected: %s\n\tactual  : %s%s", expectedStr, actualStr, diff), msgAndArgs...)
	}
	return true
}

func checkNotEqual(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	helper(t)
	if err := validateEqualArgs(expected, actual); err != nil {
		return fail(t, fmt.Sprintf("Invalid operation: %#v != %#v (%s)", expected, actual, err), msgAndArgs...)
	}
	if objectsAreEqual(expected, actual) {
		return fail(t, fmt.Sprintf("Should not be: %#v", actual), msgAndArgs...)
	}
	return true
}

func checkSame(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	helper(t)
	same, ok := samePointers(expected, actual)
	if !ok {
		return fail(t, "Both arguments must be pointers", msgAndArgs...)
	}
	if !same {
		return fail(t, fmt.Sprintf("Not same:\n\texpected: %p %#v\n\tactual  : %p %#v", expected, expected, actual, actual), msgAndArgs...)
	}
	return true
}

func checkNotSame(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	helper(t)
	same, ok := samePointers(expected, actual)
	if !ok {
		return !fail(t, "Both arguments must be pointers", msgAndArgs...)
	}
	if same {
		return fail(t, fmt.Sprintf("Expected different pointers, but both point to: %p %#v", expected, expected), msgAndArgs...)
	}
	return true
}

func checkNil(t TestingT, object any, msgAndArgs ...any) bool {
	helper(t)
	if isNil(object) {
		return true
	}
	return fail(t, fmt.Sprintf("Expected nil, but got: %#v", object), msgAndArgs...)
}

func checkNotNil(t TestingT, object any, msgAndArgs ...any) bool {
	helper(t)
	if !isNil(object) {
		return true
	}
	return fail(t, "Expected value not to be nil", msgAndArgs...)
}

func checkTrue(t TestingT, value bool, msgAndArgs ...any) bool {
	helper(t)
	if value {
		return true
	}
	return fail(t, "Should be true", msgAndArgs...)
}

func checkFalse(t TestingT, value bool, msgAndArgs ...any) bool {
	helper(t)
	if !value {
		return true
	}
	return fail(t, "Should be false", msgAndArgs...)
}

func checkEmpty(t TestingT, object any, msgAndArgs ...any) bool {
	helper(t)
	if isEmpty(object) {
		return true
	}
	return fail(t, fmt.Sprintf("Should be empty, but was %v", object), msgAndArgs...)
}

func checkNotEmpty(t TestingT, object any, msgAndArgs ...any) bool {
	helper(t)
	if !isEmpty(object) {
		return true
	}
	return fail(t, fmt.Sprintf("Should NOT be empty, but was %v", object), msgAndArgs...)
}

func checkLen(t TestingT, object any, length int, msgAndArgs ...any) bool {
	helper(t)
	l, ok := getLen(object)
	if !ok {
		return fail(t, fmt.Sprintf("%v does not have a length", object), msgAndArgs...)
	}
	if l != length {
		return fail(t, fmt.Sprintf("Should have %d item(s), but has %d", length, l), msgAndArgs...)
	}
	return true
}

func checkContains(t TestingT, container, element any, msgAndArgs ...any) bool {
	helper(t)
	ok, found := containsElement(container, element)
	if !ok {
		return fail(t, fmt.Sprintf("%#v is not a valid container", container), msgAndArgs...)
	}
	if !found {
		return fail(t, fmt.Sprintf("%#v does not contain %#v", container, element), msgAndArgs...)
	}
	return true
}

func checkNotContains(t TestingT, container, element any, msgAndArgs ...any) bool {
	helper(t)
	ok, found := containsElement(container, element)
	if !ok {
		return fail(t, fmt.Sprintf("%#v is not a valid container", container), msgAndArgs...)
	}
	if found {
		return fail(t, fmt.Sprintf("%#v should not contain %#v", container, element), msgAndArgs...)
	}
	return true
}

func checkZero(t TestingT, value any, msgAndArgs ...any) bool {
	helper(t)
	if value == nil || reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		return true
	}
	return fail(t, fmt.Sprintf("Should be zero, but was %v", value), msgAndArgs...)
}

func checkNotZero(t TestingT, value any, msgAndArgs ...any) bool {
	helper(t)
	if value == nil || reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		return fail(t, fmt.Sprintf("Should not be zero, but was %v", value), msgAndArgs...)
	}
	return true
}

func checkPanics(t TestingT, f func(), msgAndArgs ...any) bool {
	helper(t)
	didPanic, panicValue := recoverPanic(f)
	if !didPanic {
		return fail(t, fmt.Sprintf("Function should panic\n\tPanic value: %#v", panicValue), msgAndArgs...)
	}
	return true
}

func checkPanicsWithValue(t TestingT, expected any, f func(), msgAndArgs ...any) bool {
	helper(t)
	didPanic, panicValue := recoverPanic(f)
	if !didPanic {
		return fail(t, fmt.Sprintf("Function should panic\n\tPanic value: %#v", panicValue), msgAndArgs...)
	}
	if panicValue != expected {
		return fail(t, fmt.Sprintf("Function should panic with value:\n\texpected: %#v\n\tactual  : %#v", expected, panicValue), msgAndArgs...)
	}
	return true
}

func checkPanicsWithError(t TestingT, errString string, f func(), msgAndArgs ...any) bool {
	helper(t)
	didPanic, panicValue := recoverPanic(f)
	if !didPanic {
		return fail(t, fmt.Sprintf("Function should panic\n\tPanic value: %#v", panicValue), msgAndArgs...)
	}
	panicErr, ok := panicValue.(error)
	if !ok || panicErr.Error() != errString {
		return fail(t, fmt.Sprintf("Function should panic with error message:\n\texpected: %q\n\tactual  : %#v", errString, panicValue), msgAndArgs...)
	}
	return true
}

func checkNotPanics(t TestingT, f func(), msgAndArgs ...any) bool {
	helper(t)
	didPanic, panicValue := recoverPanic(f)
	if didPanic {
		return fail(t, fmt.Sprintf("Function should not panic\n\tPanic value: %v", panicValue), msgAndArgs...)
	}
	return true
}

func checkRegexp(t TestingT, rx any, str any, msgAndArgs ...any) bool {
	helper(t)
	if matchRegexp(rx, str) {
		return true
	}
	return fail(t, fmt.Sprintf("Expect %q to match %q", str, rx), msgAndArgs...)
}

func checkNotRegexp(t TestingT, rx any, str any, msgAndArgs ...any) bool {
	helper(t)
	if !matchRegexp(rx, str) {
		return true
	}
	return fail(t, fmt.Sprintf("Expect %q NOT to match %q", str, rx), msgAndArgs...)
}

func checkIsType(t TestingT, expectedType, object any, msgAndArgs ...any) bool {
	helper(t)
	if reflect.TypeOf(object) == reflect.TypeOf(expectedType) {
		return true
	}
	return fail(t, fmt.Sprintf("Object expected to be of type %T, but was %T", expectedType, object), msgAndArgs...)
}

func checkImplements(t TestingT, interfaceObject, object any, msgAndArgs ...any) bool {
	helper(t)
	interfaceType := reflect.TypeOf(interfaceObject).Elem()
	if object == nil {
		return fail(t, fmt.Sprintf("Cannot check if nil implements %v", interfaceType), msgAndArgs...)
	}
	if !reflect.TypeOf(object).Implements(interfaceType) {
		return fail(t, fmt.Sprintf("%T must implement %v", object, interfaceType), msgAndArgs...)
	}
	return true
}

func checkElementsMatch(t TestingT, listA, listB any, msgAndArgs ...any) bool {
	helper(t)
	if isEmpty(listA) && isEmpty(listB) {
		return true
	}
	if !isList(listA) || !isList(listB) {
		return fail(t, "Both arguments must be arrays or slices", msgAndArgs...)
	}
	extraA, extraB := diffLists(listA, listB)
	if len(extraA) == 0 && len(extraB) == 0 {
		return true
	}
	return fail(t, formatListDiff(listA, listB, extraA, extraB), msgAndArgs...)
}

func checkSubset(t TestingT, list, subset any, msgAndArgs ...any) bool {
	helper(t)
	if subset == nil {
		return true
	}
	listKind := reflect.TypeOf(list).Kind()
	subsetKind := reflect.TypeOf(subset).Kind()
	if listKind != reflect.Array && listKind != reflect.Slice && listKind != reflect.Map {
		return fail(t, fmt.Sprintf("%v has unsupported type %s", list, listKind), msgAndArgs...)
	}
	if subsetKind != reflect.Array && subsetKind != reflect.Slice && subsetKind != reflect.Map {
		return fail(t, fmt.Sprintf("%v has unsupported type %s", subset, subsetKind), msgAndArgs...)
	}
	subsetValue := reflect.ValueOf(subset)
	for i := 0; i < subsetValue.Len(); i++ {
		element := subsetValue.Index(i).Interface()
		ok, found := containsElement(list, element)
		if !ok {
			return fail(t, fmt.Sprintf("%#v is not a valid container", list), msgAndArgs...)
		}
		if !found {
			return fail(t, fmt.Sprintf("%#v does not contain %#v", list, element), msgAndArgs...)
		}
	}
	return true
}
