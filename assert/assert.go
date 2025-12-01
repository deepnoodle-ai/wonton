// Package assert provides test assertions for Go.
//
// This is a modern, focused reimplementation inspired by testify/assert.
// It uses Go generics for type-safe comparisons and provides clean error messages.
//
// Example usage:
//
//	func TestSomething(t *testing.T) {
//		assert.Equal(t, expected, actual)
//		assert.NoError(t, err)
//		assert.True(t, condition)
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
}

// tHelper is the interface for t.Helper().
type tHelper interface {
	Helper()
}

// failNower is the interface for t.FailNow().
type failNower interface {
	FailNow()
}

// Fail reports a test failure with the given message.
func Fail(t TestingT, failureMessage string, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
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

// FailNow fails the test and stops execution.
func FailNow(t TestingT, failureMessage string, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	Fail(t, failureMessage, msgAndArgs...)
	if fn, ok := t.(failNower); ok {
		fn.FailNow()
	} else {
		panic("test failed and t is missing FailNow()")
	}
	return false
}

// Equal asserts that expected and actual are deeply equal.
func Equal(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if err := validateEqualArgs(expected, actual); err != nil {
		return Fail(t, fmt.Sprintf("Invalid operation: %#v == %#v (%s)", expected, actual, err), msgAndArgs...)
	}
	if !objectsAreEqual(expected, actual) {
		diff := formatDiff(expected, actual)
		expectedStr, actualStr := formatUnequalValues(expected, actual)
		return Fail(t, fmt.Sprintf("Not equal:\n\texpected: %s\n\tactual  : %s%s", expectedStr, actualStr, diff), msgAndArgs...)
	}
	return true
}

// NotEqual asserts that expected and actual are not deeply equal.
func NotEqual(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if err := validateEqualArgs(expected, actual); err != nil {
		return Fail(t, fmt.Sprintf("Invalid operation: %#v != %#v (%s)", expected, actual, err), msgAndArgs...)
	}
	if objectsAreEqual(expected, actual) {
		return Fail(t, fmt.Sprintf("Should not be: %#v", actual), msgAndArgs...)
	}
	return true
}

// Same asserts that two pointers reference the same object.
func Same(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	same, ok := samePointers(expected, actual)
	if !ok {
		return Fail(t, "Both arguments must be pointers", msgAndArgs...)
	}
	if !same {
		return Fail(t, fmt.Sprintf("Not same:\n\texpected: %p %#v\n\tactual  : %p %#v", expected, expected, actual, actual), msgAndArgs...)
	}
	return true
}

// NotSame asserts that two pointers do not reference the same object.
func NotSame(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	same, ok := samePointers(expected, actual)
	if !ok {
		return !Fail(t, "Both arguments must be pointers", msgAndArgs...)
	}
	if same {
		return Fail(t, fmt.Sprintf("Expected different pointers, but both point to: %p %#v", expected, expected), msgAndArgs...)
	}
	return true
}

// Nil asserts that the specified object is nil.
func Nil(t TestingT, object any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if isNil(object) {
		return true
	}
	return Fail(t, fmt.Sprintf("Expected nil, but got: %#v", object), msgAndArgs...)
}

// NotNil asserts that the specified object is not nil.
func NotNil(t TestingT, object any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !isNil(object) {
		return true
	}
	return Fail(t, "Expected value not to be nil", msgAndArgs...)
}

// True asserts that the specified value is true.
func True(t TestingT, value bool, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if value {
		return true
	}
	return Fail(t, "Should be true", msgAndArgs...)
}

// False asserts that the specified value is false.
func False(t TestingT, value bool, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !value {
		return true
	}
	return Fail(t, "Should be false", msgAndArgs...)
}

// Empty asserts that the specified object is empty (zero value, nil, or has zero length).
func Empty(t TestingT, object any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if isEmpty(object) {
		return true
	}
	return Fail(t, fmt.Sprintf("Should be empty, but was %v", object), msgAndArgs...)
}

// NotEmpty asserts that the specified object is not empty.
func NotEmpty(t TestingT, object any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !isEmpty(object) {
		return true
	}
	return Fail(t, fmt.Sprintf("Should NOT be empty, but was %v", object), msgAndArgs...)
}

// Len asserts that the specified object has the expected length.
func Len(t TestingT, object any, length int, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	l, ok := getLen(object)
	if !ok {
		return Fail(t, fmt.Sprintf("%v does not have a length", object), msgAndArgs...)
	}
	if l != length {
		return Fail(t, fmt.Sprintf("Should have %d item(s), but has %d", length, l), msgAndArgs...)
	}
	return true
}

// Contains asserts that the specified container contains the specified element.
// Works with strings, arrays, slices, and maps.
func Contains(t TestingT, container, element any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	ok, found := containsElement(container, element)
	if !ok {
		return Fail(t, fmt.Sprintf("%#v is not a valid container", container), msgAndArgs...)
	}
	if !found {
		return Fail(t, fmt.Sprintf("%#v does not contain %#v", container, element), msgAndArgs...)
	}
	return true
}

// NotContains asserts that the specified container does not contain the specified element.
func NotContains(t TestingT, container, element any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	ok, found := containsElement(container, element)
	if !ok {
		return Fail(t, fmt.Sprintf("%#v is not a valid container", container), msgAndArgs...)
	}
	if found {
		return Fail(t, fmt.Sprintf("%#v should not contain %#v", container, element), msgAndArgs...)
	}
	return true
}

// Zero asserts that the specified value is the zero value for its type.
func Zero(t TestingT, value any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if value == nil || reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		return true
	}
	return Fail(t, fmt.Sprintf("Should be zero, but was %v", value), msgAndArgs...)
}

// NotZero asserts that the specified value is not the zero value for its type.
func NotZero(t TestingT, value any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if value == nil || reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		return Fail(t, fmt.Sprintf("Should not be zero, but was %v", value), msgAndArgs...)
	}
	return true
}

// Panics asserts that the specified function panics.
func Panics(t TestingT, f func(), msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	didPanic, panicValue := checkPanic(f)
	if !didPanic {
		return Fail(t, fmt.Sprintf("Function should panic\n\tPanic value: %#v", panicValue), msgAndArgs...)
	}
	return true
}

// PanicsWithValue asserts that the function panics with the expected value.
func PanicsWithValue(t TestingT, expected any, f func(), msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	didPanic, panicValue := checkPanic(f)
	if !didPanic {
		return Fail(t, fmt.Sprintf("Function should panic\n\tPanic value: %#v", panicValue), msgAndArgs...)
	}
	if panicValue != expected {
		return Fail(t, fmt.Sprintf("Function should panic with value:\n\texpected: %#v\n\tactual  : %#v", expected, panicValue), msgAndArgs...)
	}
	return true
}

// PanicsWithError asserts that the function panics with an error matching the expected string.
func PanicsWithError(t TestingT, errString string, f func(), msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	didPanic, panicValue := checkPanic(f)
	if !didPanic {
		return Fail(t, fmt.Sprintf("Function should panic\n\tPanic value: %#v", panicValue), msgAndArgs...)
	}
	panicErr, ok := panicValue.(error)
	if !ok || panicErr.Error() != errString {
		return Fail(t, fmt.Sprintf("Function should panic with error message:\n\texpected: %q\n\tactual  : %#v", errString, panicValue), msgAndArgs...)
	}
	return true
}

// NotPanics asserts that the specified function does not panic.
func NotPanics(t TestingT, f func(), msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	didPanic, panicValue := checkPanic(f)
	if didPanic {
		return Fail(t, fmt.Sprintf("Function should not panic\n\tPanic value: %v", panicValue), msgAndArgs...)
	}
	return true
}

// Regexp asserts that the specified string matches the regexp.
func Regexp(t TestingT, rx any, str any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if matchRegexp(rx, str) {
		return true
	}
	return Fail(t, fmt.Sprintf("Expect %q to match %q", str, rx), msgAndArgs...)
}

// NotRegexp asserts that the specified string does not match the regexp.
func NotRegexp(t TestingT, rx any, str any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !matchRegexp(rx, str) {
		return true
	}
	return Fail(t, fmt.Sprintf("Expect %q NOT to match %q", str, rx), msgAndArgs...)
}

// IsType asserts that the specified objects are of the same type.
func IsType(t TestingT, expectedType, object any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if reflect.TypeOf(object) == reflect.TypeOf(expectedType) {
		return true
	}
	return Fail(t, fmt.Sprintf("Object expected to be of type %T, but was %T", expectedType, object), msgAndArgs...)
}

// Implements asserts that an object implements the specified interface.
func Implements(t TestingT, interfaceObject, object any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	interfaceType := reflect.TypeOf(interfaceObject).Elem()
	if object == nil {
		return Fail(t, fmt.Sprintf("Cannot check if nil implements %v", interfaceType), msgAndArgs...)
	}
	if !reflect.TypeOf(object).Implements(interfaceType) {
		return Fail(t, fmt.Sprintf("%T must implement %v", object, interfaceType), msgAndArgs...)
	}
	return true
}

// ElementsMatch asserts that two slices contain the same elements, ignoring order.
func ElementsMatch(t TestingT, listA, listB any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if isEmpty(listA) && isEmpty(listB) {
		return true
	}
	if !isList(listA) || !isList(listB) {
		return Fail(t, "Both arguments must be arrays or slices", msgAndArgs...)
	}
	extraA, extraB := diffLists(listA, listB)
	if len(extraA) == 0 && len(extraB) == 0 {
		return true
	}
	return Fail(t, formatListDiff(listA, listB, extraA, extraB), msgAndArgs...)
}

// Subset asserts that subset is a subset of list.
func Subset(t TestingT, list, subset any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if subset == nil {
		return true
	}
	listKind := reflect.TypeOf(list).Kind()
	subsetKind := reflect.TypeOf(subset).Kind()
	if listKind != reflect.Array && listKind != reflect.Slice && listKind != reflect.Map {
		return Fail(t, fmt.Sprintf("%v has unsupported type %s", list, listKind), msgAndArgs...)
	}
	if subsetKind != reflect.Array && subsetKind != reflect.Slice && subsetKind != reflect.Map {
		return Fail(t, fmt.Sprintf("%v has unsupported type %s", subset, subsetKind), msgAndArgs...)
	}
	subsetValue := reflect.ValueOf(subset)
	for i := 0; i < subsetValue.Len(); i++ {
		element := subsetValue.Index(i).Interface()
		ok, found := containsElement(list, element)
		if !ok {
			return Fail(t, fmt.Sprintf("%#v is not a valid container", list), msgAndArgs...)
		}
		if !found {
			return Fail(t, fmt.Sprintf("%#v does not contain %#v", list, element), msgAndArgs...)
		}
	}
	return true
}
