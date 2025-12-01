package require

import (
	"github.com/deepnoodle-ai/wonton/assert"
)

// NoError asserts that a function returned a nil error.
func NoError(t TestingT, err error, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NoError(t, err, msgAndArgs...) {
		t.FailNow()
	}
}

// Error asserts that a function returned a non-nil error.
func Error(t TestingT, err error, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.Error(t, err, msgAndArgs...) {
		t.FailNow()
	}
}

// EqualError asserts that an error's message equals the expected string.
func EqualError(t TestingT, theError error, errString string, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.EqualError(t, theError, errString, msgAndArgs...) {
		t.FailNow()
	}
}

// ErrorContains asserts that an error's message contains the expected substring.
func ErrorContains(t TestingT, theError error, contains string, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.ErrorContains(t, theError, contains, msgAndArgs...) {
		t.FailNow()
	}
}

// ErrorIs asserts that at least one of the errors in err's chain matches target.
func ErrorIs(t TestingT, err, target error, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.ErrorIs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}

// NotErrorIs asserts that none of the errors in err's chain matches target.
func NotErrorIs(t TestingT, err, target error, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotErrorIs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}

// ErrorAs asserts that at least one of the errors in err's chain matches target.
func ErrorAs(t TestingT, err error, target any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.ErrorAs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}

// NotErrorAs asserts that none of the errors in err's chain matches target.
func NotErrorAs(t TestingT, err error, target any, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !assert.NotErrorAs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}
