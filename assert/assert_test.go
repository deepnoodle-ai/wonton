package assert

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"
)

// testingMock is a mock testing.T for capturing test failures.
type testingMock struct {
	failed  bool
	message string
}

func (m *testingMock) Errorf(format string, args ...any) {
	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func (m *testingMock) Helper() {}

func (m *testingMock) FailNow() {
	m.failed = true
}

func (m *testingMock) Name() string {
	return "testingMock"
}

func TestEqual(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	// Test equal values
	if !a.Equal(1, 1) {
		t.Error("Equal(1, 1) should return true")
	}
	if mockT.failed {
		t.Error("Equal(1, 1) should not fail")
	}

	// Test unequal values
	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Equal(1, 2) {
		t.Error("Equal(1, 2) should return false")
	}
	if !mockT.failed {
		t.Error("Equal(1, 2) should fail")
	}

	// Test equal strings
	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Equal("hello", "hello") {
		t.Error("Equal(hello, hello) should return true")
	}

	// Test equal slices
	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Equal([]int{1, 2, 3}, []int{1, 2, 3}) {
		t.Error("Equal(slice, slice) should return true")
	}

	// Test equal maps
	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Equal(map[string]int{"a": 1}, map[string]int{"a": 1}) {
		t.Error("Equal(map, map) should return true")
	}

	// Test equal structs
	mockT = &testingMock{}
	a = NonFatal(mockT)
	type person struct {
		Name string
		Age  int
	}
	if !a.Equal(person{"Alice", 30}, person{"Alice", 30}) {
		t.Error("Equal(struct, struct) should return true")
	}

	// Test with message
	mockT = &testingMock{}
	a = NonFatal(mockT)
	a.Equal(1, 2, "expected %d to equal %d", 1, 2)
	if !mockT.failed {
		t.Error("Equal with message should fail")
	}
}

func TestNotEqual(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.NotEqual(1, 2) {
		t.Error("NotEqual(1, 2) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.NotEqual(1, 1) {
		t.Error("NotEqual(1, 1) should return false")
	}
	if !mockT.failed {
		t.Error("NotEqual(1, 1) should fail")
	}
}

func TestNil(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Nil(nil) {
		t.Error("Nil(nil) should return true")
	}

	var nilPtr *int
	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Nil(nilPtr) {
		t.Error("Nil(nilPtr) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Nil(1) {
		t.Error("Nil(1) should return false")
	}
	if !mockT.failed {
		t.Error("Nil(1) should fail")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	ptr := new(int)
	if a.Nil(ptr) {
		t.Error("Nil(ptr) should return false")
	}
}

func TestNotNil(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.NotNil(1) {
		t.Error("NotNil(1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.NotNil(nil) {
		t.Error("NotNil(nil) should return false")
	}
	if !mockT.failed {
		t.Error("NotNil(nil) should fail")
	}
}

func TestTrue(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.True(true) {
		t.Error("True(true) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.True(false) {
		t.Error("True(false) should return false")
	}
	if !mockT.failed {
		t.Error("True(false) should fail")
	}
}

func TestFalse(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.False(false) {
		t.Error("False(false) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.False(true) {
		t.Error("False(true) should return false")
	}
	if !mockT.failed {
		t.Error("False(true) should fail")
	}
}

func TestEmpty(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Empty("") {
		t.Error("Empty(\"\") should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Empty([]int{}) {
		t.Error("Empty([]int{}) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Empty(map[string]int{}) {
		t.Error("Empty(map{}) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Empty(0) {
		t.Error("Empty(0) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Empty("hello") {
		t.Error("Empty(\"hello\") should return false")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Empty([]int{1}) {
		t.Error("Empty([]int{1}) should return false")
	}
}

func TestNotEmpty(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.NotEmpty("hello") {
		t.Error("NotEmpty(\"hello\") should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.NotEmpty("") {
		t.Error("NotEmpty(\"\") should return false")
	}
}

func TestLen(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Len([]int{1, 2, 3}, 3) {
		t.Error("Len([1,2,3], 3) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Len("hello", 5) {
		t.Error("Len(\"hello\", 5) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Len([]int{1, 2}, 3) {
		t.Error("Len([1,2], 3) should return false")
	}
	if !mockT.failed {
		t.Error("Len with wrong length should fail")
	}
}

func TestContains(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Contains("hello world", "world") {
		t.Error("Contains(\"hello world\", \"world\") should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Contains([]int{1, 2, 3}, 2) {
		t.Error("Contains([1,2,3], 2) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Contains(map[string]int{"a": 1}, "a") {
		t.Error("Contains(map, \"a\") should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Contains("hello", "world") {
		t.Error("Contains(\"hello\", \"world\") should return false")
	}
}

func TestNotContains(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.NotContains("hello", "world") {
		t.Error("NotContains(\"hello\", \"world\") should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.NotContains("hello world", "world") {
		t.Error("NotContains(\"hello world\", \"world\") should return false")
	}
}

func TestZero(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Zero(0) {
		t.Error("Zero(0) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Zero("") {
		t.Error("Zero(\"\") should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Zero(nil) {
		t.Error("Zero(nil) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Zero(1) {
		t.Error("Zero(1) should return false")
	}
}

func TestNotZero(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.NotZero(1) {
		t.Error("NotZero(1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.NotZero(0) {
		t.Error("NotZero(0) should return false")
	}
}

func TestPanics(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Panics(func() { panic("test") }) {
		t.Error("Panics should return true for panicking function")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Panics(func() {}) {
		t.Error("Panics should return false for non-panicking function")
	}
	if !mockT.failed {
		t.Error("Panics should fail for non-panicking function")
	}
}

func TestPanicsWithValue(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.PanicsWithValue("test", func() { panic("test") }) {
		t.Error("PanicsWithValue should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.PanicsWithValue("test", func() { panic("other") }) {
		t.Error("PanicsWithValue should return false for wrong value")
	}
}

func TestNotPanics(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.NotPanics(func() {}) {
		t.Error("NotPanics should return true for non-panicking function")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.NotPanics(func() { panic("test") }) {
		t.Error("NotPanics should return false for panicking function")
	}
}

func TestRegexp(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Regexp("^hello", "hello world") {
		t.Error("Regexp should return true for matching pattern")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.Regexp(regexp.MustCompile(`\d+`), "abc123") {
		t.Error("Regexp should return true for compiled regex")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Regexp("^world", "hello world") {
		t.Error("Regexp should return false for non-matching pattern")
	}
}

func TestNotRegexp(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.NotRegexp("^world", "hello world") {
		t.Error("NotRegexp should return true for non-matching pattern")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.NotRegexp("^hello", "hello world") {
		t.Error("NotRegexp should return false for matching pattern")
	}
}

func TestIsType(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.IsType(1, 2) {
		t.Error("IsType should return true for same types")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.IsType(1, "string") {
		t.Error("IsType should return false for different types")
	}
}

func TestImplements(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	var _ error = errors.New("test")
	if !a.Implements((*error)(nil), errors.New("test")) {
		t.Error("Implements should return true")
	}
}

func TestElementsMatch(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.ElementsMatch([]int{1, 2, 3}, []int{3, 2, 1}) {
		t.Error("ElementsMatch should return true for same elements in different order")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.ElementsMatch([]int{1, 1, 2}, []int{1, 2, 1}) {
		t.Error("ElementsMatch should return true for same elements with duplicates")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.ElementsMatch([]int{1, 2, 3}, []int{1, 2, 4}) {
		t.Error("ElementsMatch should return false for different elements")
	}
}

func TestSubset(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Subset([]int{1, 2, 3}, []int{1, 2}) {
		t.Error("Subset should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Subset([]int{1, 2}, []int{1, 2, 3}) {
		t.Error("Subset should return false for superset")
	}
}

func TestSame(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	obj := new(int)
	if !a.Same(obj, obj) {
		t.Error("Same should return true for same pointer")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	obj1 := new(int)
	obj2 := new(int)
	if a.Same(obj1, obj2) {
		t.Error("Same should return false for different pointers")
	}
}

func TestNotSame(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	obj1 := new(int)
	obj2 := new(int)
	if !a.NotSame(obj1, obj2) {
		t.Error("NotSame should return true for different pointers")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	obj := new(int)
	if a.NotSame(obj, obj) {
		t.Error("NotSame should return false for same pointer")
	}
}

func TestGreater(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Greater(2, 1) {
		t.Error("Greater(2, 1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Greater(1, 2) {
		t.Error("Greater(1, 2) should return false")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Greater(1, 1) {
		t.Error("Greater(1, 1) should return false")
	}
}

func TestGreaterOrEqual(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.GreaterOrEqual(2, 1) {
		t.Error("GreaterOrEqual(2, 1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.GreaterOrEqual(1, 1) {
		t.Error("GreaterOrEqual(1, 1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.GreaterOrEqual(1, 2) {
		t.Error("GreaterOrEqual(1, 2) should return false")
	}
}

func TestLess(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Less(1, 2) {
		t.Error("Less(1, 2) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Less(2, 1) {
		t.Error("Less(2, 1) should return false")
	}
}

func TestLessOrEqual(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.LessOrEqual(1, 2) {
		t.Error("LessOrEqual(1, 2) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.LessOrEqual(1, 1) {
		t.Error("LessOrEqual(1, 1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.LessOrEqual(2, 1) {
		t.Error("LessOrEqual(2, 1) should return false")
	}
}

func TestPositive(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Positive(1) {
		t.Error("Positive(1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Positive(0) {
		t.Error("Positive(0) should return false")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Positive(-1) {
		t.Error("Positive(-1) should return false")
	}
}

func TestNegative(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Negative(-1) {
		t.Error("Negative(-1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Negative(0) {
		t.Error("Negative(0) should return false")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Negative(1) {
		t.Error("Negative(1) should return false")
	}
}

func TestInDelta(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.InDelta(1.0, 1.0, 0.0) {
		t.Error("InDelta(1.0, 1.0, 0.0) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.InDelta(1.0, 1.01, 0.1) {
		t.Error("InDelta(1.0, 1.01, 0.1) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.InDelta(1.0, 2.0, 0.1) {
		t.Error("InDelta(1.0, 2.0, 0.1) should return false")
	}
}

func TestInEpsilon(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.InEpsilon(100.0, 101.0, 0.02) {
		t.Error("InEpsilon(100.0, 101.0, 0.02) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.InEpsilon(100.0, 110.0, 0.02) {
		t.Error("InEpsilon(100.0, 110.0, 0.02) should return false")
	}
}

func TestWithinDuration(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)
	now := time.Now()

	if !a.WithinDuration(now, now, 0) {
		t.Error("WithinDuration should return true for same time")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if !a.WithinDuration(now, now.Add(time.Second), 2*time.Second) {
		t.Error("WithinDuration should return true when within delta")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.WithinDuration(now, now.Add(10*time.Second), time.Second) {
		t.Error("WithinDuration should return false when outside delta")
	}
}

func TestWithinRange(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)
	now := time.Now()

	if !a.WithinRange(now, now.Add(-time.Second), now.Add(time.Second)) {
		t.Error("WithinRange should return true when in range")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.WithinRange(now, now.Add(time.Second), now.Add(2*time.Second)) {
		t.Error("WithinRange should return false when before range")
	}
}

func TestNoError(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.NoError(nil) {
		t.Error("NoError(nil) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.NoError(errors.New("error")) {
		t.Error("NoError(error) should return false")
	}
}

func TestError(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.Error(errors.New("error")) {
		t.Error("Error(error) should return true")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.Error(nil) {
		t.Error("Error(nil) should return false")
	}
}

func TestEqualError(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.EqualError(errors.New("test error"), "test error") {
		t.Error("EqualError should return true for matching error")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.EqualError(errors.New("test error"), "other error") {
		t.Error("EqualError should return false for non-matching error")
	}
}

func TestErrorContains(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)

	if !a.ErrorContains(errors.New("test error message"), "error") {
		t.Error("ErrorContains should return true for containing substring")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	if a.ErrorContains(errors.New("test"), "error") {
		t.Error("ErrorContains should return false for non-containing substring")
	}
}

func TestErrorIs(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)
	targetErr := errors.New("target")
	wrappedErr := fmt.Errorf("wrapped: %w", targetErr)

	if !a.ErrorIs(wrappedErr, targetErr) {
		t.Error("ErrorIs should return true for wrapped error")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	otherErr := errors.New("other")
	if a.ErrorIs(wrappedErr, otherErr) {
		t.Error("ErrorIs should return false for different error")
	}
}

func TestNotErrorIs(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)
	targetErr := errors.New("target")
	otherErr := errors.New("other")

	if !a.NotErrorIs(otherErr, targetErr) {
		t.Error("NotErrorIs should return true for different error")
	}

	mockT = &testingMock{}
	a = NonFatal(mockT)
	wrappedErr := fmt.Errorf("wrapped: %w", targetErr)
	if a.NotErrorIs(wrappedErr, targetErr) {
		t.Error("NotErrorIs should return false for wrapped error")
	}
}

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func TestErrorAs(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)
	err := &customError{msg: "custom error"}
	var target *customError

	if !a.ErrorAs(err, &target) {
		t.Error("ErrorAs should return true for matching type")
	}
	if target.msg != "custom error" {
		t.Error("ErrorAs should set target")
	}
}

func TestNotErrorAs(t *testing.T) {
	mockT := &testingMock{}
	a := NonFatal(mockT)
	err := errors.New("standard error")
	var target *customError

	if !a.NotErrorAs(err, &target) {
		t.Error("NotErrorAs should return true for non-matching type")
	}
}

// TestFatalBehavior tests that the default functions call FailNow
func TestFatalBehavior(t *testing.T) {
	mockT := &testingMock{}

	// This should call FailNow
	Equal(mockT, 1, 2)
	if !mockT.failed {
		t.Error("Equal should have called FailNow")
	}
}
