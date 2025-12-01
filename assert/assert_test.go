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

	// Test equal values
	if !Equal(mockT, 1, 1) {
		t.Error("Equal(1, 1) should return true")
	}
	if mockT.failed {
		t.Error("Equal(1, 1) should not fail")
	}

	// Test unequal values
	mockT = &testingMock{}
	if Equal(mockT, 1, 2) {
		t.Error("Equal(1, 2) should return false")
	}
	if !mockT.failed {
		t.Error("Equal(1, 2) should fail")
	}

	// Test equal strings
	mockT = &testingMock{}
	if !Equal(mockT, "hello", "hello") {
		t.Error("Equal(hello, hello) should return true")
	}

	// Test equal slices
	mockT = &testingMock{}
	if !Equal(mockT, []int{1, 2, 3}, []int{1, 2, 3}) {
		t.Error("Equal(slice, slice) should return true")
	}

	// Test equal maps
	mockT = &testingMock{}
	if !Equal(mockT, map[string]int{"a": 1}, map[string]int{"a": 1}) {
		t.Error("Equal(map, map) should return true")
	}

	// Test equal structs
	mockT = &testingMock{}
	type person struct {
		Name string
		Age  int
	}
	if !Equal(mockT, person{"Alice", 30}, person{"Alice", 30}) {
		t.Error("Equal(struct, struct) should return true")
	}

	// Test with message
	mockT = &testingMock{}
	Equal(mockT, 1, 2, "expected %d to equal %d", 1, 2)
	if !mockT.failed {
		t.Error("Equal with message should fail")
	}
}

func TestNotEqual(t *testing.T) {
	mockT := &testingMock{}

	if !NotEqual(mockT, 1, 2) {
		t.Error("NotEqual(1, 2) should return true")
	}

	mockT = &testingMock{}
	if NotEqual(mockT, 1, 1) {
		t.Error("NotEqual(1, 1) should return false")
	}
	if !mockT.failed {
		t.Error("NotEqual(1, 1) should fail")
	}
}

func TestNil(t *testing.T) {
	mockT := &testingMock{}

	if !Nil(mockT, nil) {
		t.Error("Nil(nil) should return true")
	}

	var nilPtr *int
	mockT = &testingMock{}
	if !Nil(mockT, nilPtr) {
		t.Error("Nil(nilPtr) should return true")
	}

	mockT = &testingMock{}
	if Nil(mockT, 1) {
		t.Error("Nil(1) should return false")
	}
	if !mockT.failed {
		t.Error("Nil(1) should fail")
	}

	mockT = &testingMock{}
	ptr := new(int)
	if Nil(mockT, ptr) {
		t.Error("Nil(ptr) should return false")
	}
}

func TestNotNil(t *testing.T) {
	mockT := &testingMock{}

	if !NotNil(mockT, 1) {
		t.Error("NotNil(1) should return true")
	}

	mockT = &testingMock{}
	if NotNil(mockT, nil) {
		t.Error("NotNil(nil) should return false")
	}
	if !mockT.failed {
		t.Error("NotNil(nil) should fail")
	}
}

func TestTrue(t *testing.T) {
	mockT := &testingMock{}

	if !True(mockT, true) {
		t.Error("True(true) should return true")
	}

	mockT = &testingMock{}
	if True(mockT, false) {
		t.Error("True(false) should return false")
	}
	if !mockT.failed {
		t.Error("True(false) should fail")
	}
}

func TestFalse(t *testing.T) {
	mockT := &testingMock{}

	if !False(mockT, false) {
		t.Error("False(false) should return true")
	}

	mockT = &testingMock{}
	if False(mockT, true) {
		t.Error("False(true) should return false")
	}
	if !mockT.failed {
		t.Error("False(true) should fail")
	}
}

func TestEmpty(t *testing.T) {
	mockT := &testingMock{}

	if !Empty(mockT, "") {
		t.Error("Empty(\"\") should return true")
	}

	mockT = &testingMock{}
	if !Empty(mockT, []int{}) {
		t.Error("Empty([]int{}) should return true")
	}

	mockT = &testingMock{}
	if !Empty(mockT, map[string]int{}) {
		t.Error("Empty(map{}) should return true")
	}

	mockT = &testingMock{}
	if !Empty(mockT, 0) {
		t.Error("Empty(0) should return true")
	}

	mockT = &testingMock{}
	if Empty(mockT, "hello") {
		t.Error("Empty(\"hello\") should return false")
	}

	mockT = &testingMock{}
	if Empty(mockT, []int{1}) {
		t.Error("Empty([]int{1}) should return false")
	}
}

func TestNotEmpty(t *testing.T) {
	mockT := &testingMock{}

	if !NotEmpty(mockT, "hello") {
		t.Error("NotEmpty(\"hello\") should return true")
	}

	mockT = &testingMock{}
	if NotEmpty(mockT, "") {
		t.Error("NotEmpty(\"\") should return false")
	}
}

func TestLen(t *testing.T) {
	mockT := &testingMock{}

	if !Len(mockT, []int{1, 2, 3}, 3) {
		t.Error("Len([1,2,3], 3) should return true")
	}

	mockT = &testingMock{}
	if !Len(mockT, "hello", 5) {
		t.Error("Len(\"hello\", 5) should return true")
	}

	mockT = &testingMock{}
	if Len(mockT, []int{1, 2}, 3) {
		t.Error("Len([1,2], 3) should return false")
	}
	if !mockT.failed {
		t.Error("Len with wrong length should fail")
	}
}

func TestContains(t *testing.T) {
	mockT := &testingMock{}

	if !Contains(mockT, "hello world", "world") {
		t.Error("Contains(\"hello world\", \"world\") should return true")
	}

	mockT = &testingMock{}
	if !Contains(mockT, []int{1, 2, 3}, 2) {
		t.Error("Contains([1,2,3], 2) should return true")
	}

	mockT = &testingMock{}
	if !Contains(mockT, map[string]int{"a": 1}, "a") {
		t.Error("Contains(map, \"a\") should return true")
	}

	mockT = &testingMock{}
	if Contains(mockT, "hello", "world") {
		t.Error("Contains(\"hello\", \"world\") should return false")
	}
}

func TestNotContains(t *testing.T) {
	mockT := &testingMock{}

	if !NotContains(mockT, "hello", "world") {
		t.Error("NotContains(\"hello\", \"world\") should return true")
	}

	mockT = &testingMock{}
	if NotContains(mockT, "hello world", "world") {
		t.Error("NotContains(\"hello world\", \"world\") should return false")
	}
}

func TestZero(t *testing.T) {
	mockT := &testingMock{}

	if !Zero(mockT, 0) {
		t.Error("Zero(0) should return true")
	}

	mockT = &testingMock{}
	if !Zero(mockT, "") {
		t.Error("Zero(\"\") should return true")
	}

	mockT = &testingMock{}
	if !Zero(mockT, nil) {
		t.Error("Zero(nil) should return true")
	}

	mockT = &testingMock{}
	if Zero(mockT, 1) {
		t.Error("Zero(1) should return false")
	}
}

func TestNotZero(t *testing.T) {
	mockT := &testingMock{}

	if !NotZero(mockT, 1) {
		t.Error("NotZero(1) should return true")
	}

	mockT = &testingMock{}
	if NotZero(mockT, 0) {
		t.Error("NotZero(0) should return false")
	}
}

func TestPanics(t *testing.T) {
	mockT := &testingMock{}

	if !Panics(mockT, func() { panic("test") }) {
		t.Error("Panics should return true for panicking function")
	}

	mockT = &testingMock{}
	if Panics(mockT, func() {}) {
		t.Error("Panics should return false for non-panicking function")
	}
	if !mockT.failed {
		t.Error("Panics should fail for non-panicking function")
	}
}

func TestPanicsWithValue(t *testing.T) {
	mockT := &testingMock{}

	if !PanicsWithValue(mockT, "test", func() { panic("test") }) {
		t.Error("PanicsWithValue should return true")
	}

	mockT = &testingMock{}
	if PanicsWithValue(mockT, "test", func() { panic("other") }) {
		t.Error("PanicsWithValue should return false for wrong value")
	}
}

func TestNotPanics(t *testing.T) {
	mockT := &testingMock{}

	if !NotPanics(mockT, func() {}) {
		t.Error("NotPanics should return true for non-panicking function")
	}

	mockT = &testingMock{}
	if NotPanics(mockT, func() { panic("test") }) {
		t.Error("NotPanics should return false for panicking function")
	}
}

func TestRegexp(t *testing.T) {
	mockT := &testingMock{}

	if !Regexp(mockT, "^hello", "hello world") {
		t.Error("Regexp should return true for matching pattern")
	}

	mockT = &testingMock{}
	if !Regexp(mockT, regexp.MustCompile(`\d+`), "abc123") {
		t.Error("Regexp should return true for compiled regex")
	}

	mockT = &testingMock{}
	if Regexp(mockT, "^world", "hello world") {
		t.Error("Regexp should return false for non-matching pattern")
	}
}

func TestNotRegexp(t *testing.T) {
	mockT := &testingMock{}

	if !NotRegexp(mockT, "^world", "hello world") {
		t.Error("NotRegexp should return true for non-matching pattern")
	}

	mockT = &testingMock{}
	if NotRegexp(mockT, "^hello", "hello world") {
		t.Error("NotRegexp should return false for matching pattern")
	}
}

func TestIsType(t *testing.T) {
	mockT := &testingMock{}

	if !IsType(mockT, 1, 2) {
		t.Error("IsType should return true for same types")
	}

	mockT = &testingMock{}
	if IsType(mockT, 1, "string") {
		t.Error("IsType should return false for different types")
	}
}

func TestImplements(t *testing.T) {
	mockT := &testingMock{}

	var _ error = errors.New("test")
	if !Implements(mockT, (*error)(nil), errors.New("test")) {
		t.Error("Implements should return true")
	}
}

func TestElementsMatch(t *testing.T) {
	mockT := &testingMock{}

	if !ElementsMatch(mockT, []int{1, 2, 3}, []int{3, 2, 1}) {
		t.Error("ElementsMatch should return true for same elements in different order")
	}

	mockT = &testingMock{}
	if !ElementsMatch(mockT, []int{1, 1, 2}, []int{1, 2, 1}) {
		t.Error("ElementsMatch should return true for same elements with duplicates")
	}

	mockT = &testingMock{}
	if ElementsMatch(mockT, []int{1, 2, 3}, []int{1, 2, 4}) {
		t.Error("ElementsMatch should return false for different elements")
	}
}

func TestSubset(t *testing.T) {
	mockT := &testingMock{}

	if !Subset(mockT, []int{1, 2, 3}, []int{1, 2}) {
		t.Error("Subset should return true")
	}

	mockT = &testingMock{}
	if Subset(mockT, []int{1, 2}, []int{1, 2, 3}) {
		t.Error("Subset should return false for superset")
	}
}

func TestSame(t *testing.T) {
	mockT := &testingMock{}

	obj := new(int)
	if !Same(mockT, obj, obj) {
		t.Error("Same should return true for same pointer")
	}

	mockT = &testingMock{}
	obj1 := new(int)
	obj2 := new(int)
	if Same(mockT, obj1, obj2) {
		t.Error("Same should return false for different pointers")
	}
}

func TestNotSame(t *testing.T) {
	mockT := &testingMock{}

	obj1 := new(int)
	obj2 := new(int)
	if !NotSame(mockT, obj1, obj2) {
		t.Error("NotSame should return true for different pointers")
	}

	mockT = &testingMock{}
	obj := new(int)
	if NotSame(mockT, obj, obj) {
		t.Error("NotSame should return false for same pointer")
	}
}

func TestGreater(t *testing.T) {
	mockT := &testingMock{}

	if !Greater(mockT, 2, 1) {
		t.Error("Greater(2, 1) should return true")
	}

	mockT = &testingMock{}
	if Greater(mockT, 1, 2) {
		t.Error("Greater(1, 2) should return false")
	}

	mockT = &testingMock{}
	if Greater(mockT, 1, 1) {
		t.Error("Greater(1, 1) should return false")
	}
}

func TestGreaterOrEqual(t *testing.T) {
	mockT := &testingMock{}

	if !GreaterOrEqual(mockT, 2, 1) {
		t.Error("GreaterOrEqual(2, 1) should return true")
	}

	mockT = &testingMock{}
	if !GreaterOrEqual(mockT, 1, 1) {
		t.Error("GreaterOrEqual(1, 1) should return true")
	}

	mockT = &testingMock{}
	if GreaterOrEqual(mockT, 1, 2) {
		t.Error("GreaterOrEqual(1, 2) should return false")
	}
}

func TestLess(t *testing.T) {
	mockT := &testingMock{}

	if !Less(mockT, 1, 2) {
		t.Error("Less(1, 2) should return true")
	}

	mockT = &testingMock{}
	if Less(mockT, 2, 1) {
		t.Error("Less(2, 1) should return false")
	}
}

func TestLessOrEqual(t *testing.T) {
	mockT := &testingMock{}

	if !LessOrEqual(mockT, 1, 2) {
		t.Error("LessOrEqual(1, 2) should return true")
	}

	mockT = &testingMock{}
	if !LessOrEqual(mockT, 1, 1) {
		t.Error("LessOrEqual(1, 1) should return true")
	}

	mockT = &testingMock{}
	if LessOrEqual(mockT, 2, 1) {
		t.Error("LessOrEqual(2, 1) should return false")
	}
}

func TestPositive(t *testing.T) {
	mockT := &testingMock{}

	if !Positive(mockT, 1) {
		t.Error("Positive(1) should return true")
	}

	mockT = &testingMock{}
	if Positive(mockT, 0) {
		t.Error("Positive(0) should return false")
	}

	mockT = &testingMock{}
	if Positive(mockT, -1) {
		t.Error("Positive(-1) should return false")
	}
}

func TestNegative(t *testing.T) {
	mockT := &testingMock{}

	if !Negative(mockT, -1) {
		t.Error("Negative(-1) should return true")
	}

	mockT = &testingMock{}
	if Negative(mockT, 0) {
		t.Error("Negative(0) should return false")
	}

	mockT = &testingMock{}
	if Negative(mockT, 1) {
		t.Error("Negative(1) should return false")
	}
}

func TestInDelta(t *testing.T) {
	mockT := &testingMock{}

	if !InDelta(mockT, 1.0, 1.0, 0.0) {
		t.Error("InDelta(1.0, 1.0, 0.0) should return true")
	}

	mockT = &testingMock{}
	if !InDelta(mockT, 1.0, 1.01, 0.1) {
		t.Error("InDelta(1.0, 1.01, 0.1) should return true")
	}

	mockT = &testingMock{}
	if InDelta(mockT, 1.0, 2.0, 0.1) {
		t.Error("InDelta(1.0, 2.0, 0.1) should return false")
	}
}

func TestInEpsilon(t *testing.T) {
	mockT := &testingMock{}

	if !InEpsilon(mockT, 100.0, 101.0, 0.02) {
		t.Error("InEpsilon(100.0, 101.0, 0.02) should return true")
	}

	mockT = &testingMock{}
	if InEpsilon(mockT, 100.0, 110.0, 0.02) {
		t.Error("InEpsilon(100.0, 110.0, 0.02) should return false")
	}
}

func TestWithinDuration(t *testing.T) {
	mockT := &testingMock{}
	now := time.Now()

	if !WithinDuration(mockT, now, now, 0) {
		t.Error("WithinDuration should return true for same time")
	}

	mockT = &testingMock{}
	if !WithinDuration(mockT, now, now.Add(time.Second), 2*time.Second) {
		t.Error("WithinDuration should return true when within delta")
	}

	mockT = &testingMock{}
	if WithinDuration(mockT, now, now.Add(10*time.Second), time.Second) {
		t.Error("WithinDuration should return false when outside delta")
	}
}

func TestWithinRange(t *testing.T) {
	mockT := &testingMock{}
	now := time.Now()

	if !WithinRange(mockT, now, now.Add(-time.Second), now.Add(time.Second)) {
		t.Error("WithinRange should return true when in range")
	}

	mockT = &testingMock{}
	if WithinRange(mockT, now, now.Add(time.Second), now.Add(2*time.Second)) {
		t.Error("WithinRange should return false when before range")
	}
}

func TestNoError(t *testing.T) {
	mockT := &testingMock{}

	if !NoError(mockT, nil) {
		t.Error("NoError(nil) should return true")
	}

	mockT = &testingMock{}
	if NoError(mockT, errors.New("error")) {
		t.Error("NoError(error) should return false")
	}
}

func TestError(t *testing.T) {
	mockT := &testingMock{}

	if !Error(mockT, errors.New("error")) {
		t.Error("Error(error) should return true")
	}

	mockT = &testingMock{}
	if Error(mockT, nil) {
		t.Error("Error(nil) should return false")
	}
}

func TestEqualError(t *testing.T) {
	mockT := &testingMock{}

	if !EqualError(mockT, errors.New("test error"), "test error") {
		t.Error("EqualError should return true for matching error")
	}

	mockT = &testingMock{}
	if EqualError(mockT, errors.New("test error"), "other error") {
		t.Error("EqualError should return false for non-matching error")
	}
}

func TestErrorContains(t *testing.T) {
	mockT := &testingMock{}

	if !ErrorContains(mockT, errors.New("test error message"), "error") {
		t.Error("ErrorContains should return true for containing substring")
	}

	mockT = &testingMock{}
	if ErrorContains(mockT, errors.New("test"), "error") {
		t.Error("ErrorContains should return false for non-containing substring")
	}
}

func TestErrorIs(t *testing.T) {
	mockT := &testingMock{}
	targetErr := errors.New("target")
	wrappedErr := fmt.Errorf("wrapped: %w", targetErr)

	if !ErrorIs(mockT, wrappedErr, targetErr) {
		t.Error("ErrorIs should return true for wrapped error")
	}

	mockT = &testingMock{}
	otherErr := errors.New("other")
	if ErrorIs(mockT, wrappedErr, otherErr) {
		t.Error("ErrorIs should return false for different error")
	}
}

func TestNotErrorIs(t *testing.T) {
	mockT := &testingMock{}
	targetErr := errors.New("target")
	otherErr := errors.New("other")

	if !NotErrorIs(mockT, otherErr, targetErr) {
		t.Error("NotErrorIs should return true for different error")
	}

	mockT = &testingMock{}
	wrappedErr := fmt.Errorf("wrapped: %w", targetErr)
	if NotErrorIs(mockT, wrappedErr, targetErr) {
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
	err := &customError{msg: "custom error"}
	var target *customError

	if !ErrorAs(mockT, err, &target) {
		t.Error("ErrorAs should return true for matching type")
	}
	if target.msg != "custom error" {
		t.Error("ErrorAs should set target")
	}
}

func TestNotErrorAs(t *testing.T) {
	mockT := &testingMock{}
	err := errors.New("standard error")
	var target *customError

	if !NotErrorAs(mockT, err, &target) {
		t.Error("NotErrorAs should return true for non-matching type")
	}
}
