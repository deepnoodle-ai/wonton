package assert

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// NoError asserts that a function returned a nil error.
func NoError(t TestingT, err error, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if err == nil {
		return true
	}
	return Fail(t, fmt.Sprintf("Received unexpected error:\n%+v", err), msgAndArgs...)
}

// Error asserts that a function returned a non-nil error.
func Error(t TestingT, err error, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if err != nil {
		return true
	}
	return Fail(t, "An error is expected but got nil", msgAndArgs...)
}

// EqualError asserts that an error's message equals the expected string.
func EqualError(t TestingT, theError error, errString string, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !Error(t, theError, msgAndArgs...) {
		return false
	}
	if theError.Error() != errString {
		return Fail(t, fmt.Sprintf("Error message not equal:\n\texpected: %q\n\tactual  : %q", errString, theError.Error()), msgAndArgs...)
	}
	return true
}

// ErrorContains asserts that an error's message contains the expected substring.
func ErrorContains(t TestingT, theError error, contains string, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !Error(t, theError, msgAndArgs...) {
		return false
	}
	if !strings.Contains(theError.Error(), contains) {
		return Fail(t, fmt.Sprintf("Error %q does not contain %q", theError.Error(), contains), msgAndArgs...)
	}
	return true
}

// ErrorIs asserts that at least one of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func ErrorIs(t TestingT, err, target error, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if errors.Is(err, target) {
		return true
	}
	var expectedText string
	if target != nil {
		expectedText = target.Error()
	}
	if err == nil {
		return Fail(t, fmt.Sprintf("Expected error with %q in chain but got nil", expectedText), msgAndArgs...)
	}
	chain := buildErrorChainString(err)
	return Fail(t, fmt.Sprintf("Target error should be in err chain:\n\texpected: %q\n\tin chain: %s", expectedText, chain), msgAndArgs...)
}

// NotErrorIs asserts that none of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func NotErrorIs(t TestingT, err, target error, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !errors.Is(err, target) {
		return true
	}
	var expectedText string
	if target != nil {
		expectedText = target.Error()
	}
	chain := buildErrorChainString(err)
	return Fail(t, fmt.Sprintf("Target error should NOT be in err chain:\n\tfound: %q\n\tin chain: %s", expectedText, chain), msgAndArgs...)
}

// ErrorAs asserts that at least one of the errors in err's chain matches target,
// and if so, sets target to that error value.
// This is a wrapper for errors.As.
func ErrorAs(t TestingT, err error, target any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if errors.As(err, target) {
		return true
	}
	expectedType := reflect.TypeOf(target).Elem().String()
	if err == nil {
		return Fail(t, fmt.Sprintf("An error is expected but got nil.\n\texpected: %s", expectedType), msgAndArgs...)
	}
	chain := buildErrorChainString(err)
	return Fail(t, fmt.Sprintf("Should be in error chain:\n\texpected: %s\n\tin chain: %s", expectedType, chain), msgAndArgs...)
}

// NotErrorAs asserts that none of the errors in err's chain matches target.
func NotErrorAs(t TestingT, err error, target any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !errors.As(err, target) {
		return true
	}
	chain := buildErrorChainString(err)
	return Fail(t, fmt.Sprintf("Target error should NOT be in err chain:\n\tfound: %s\n\tin chain: %s", reflect.TypeOf(target).Elem().String(), chain), msgAndArgs...)
}

// buildErrorChainString returns a string representation of the error chain.
func buildErrorChainString(err error) string {
	if err == nil {
		return ""
	}
	var chain string
	for e := err; e != nil; e = errors.Unwrap(e) {
		if chain != "" {
			chain += "\n\t       "
		}
		chain += fmt.Sprintf("%q (%T)", e.Error(), e)
	}
	return chain
}
