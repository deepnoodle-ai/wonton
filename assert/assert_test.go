package assert

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
)

// Example demonstrates basic assertion usage.
func Example() {
	// In a real test, you would use t *testing.T
	// This example uses a mock for demonstration

	// Equal compares values deeply
	type User struct {
		Name string
		Age  int
	}
	user := User{Name: "Alice", Age: 30}
	_ = user // In real test: assert.Equal(t, user.Name, "Alice")

	// NoError checks for nil errors
	var err error = nil
	_ = err // In real test: assert.NoError(t, err)

	// True/False check boolean conditions
	active := true
	_ = active // In real test: assert.True(t, active)
}

// ExampleEqual demonstrates deep equality assertions.
func ExampleEqual() {
	// Mock for demonstration - use *testing.T in real tests
	t := &mockT{}

	// Compare simple values
	Equal(t, 42, 42)

	// Compare structs
	type Point struct {
		X, Y int
	}
	Equal(t, Point{X: 1, Y: 2}, Point{X: 1, Y: 2})

	// With optional message
	Equal(t, "hello", "hello", "greeting should match")

	// Output:
}

// ExampleNoError demonstrates error assertions.
func ExampleNoError() {
	t := &mockT{}

	// Assert that operation succeeded
	err := performOperation()
	NoError(t, err)

	// With custom message
	NoError(t, err, "operation should succeed")

	// Output:
}

func performOperation() error {
	return nil // Simulated success
}

// ExampleErrorIs demonstrates error chain checking.
func ExampleErrorIs() {
	t := &mockT{}

	// Define a sentinel error
	var ErrNotFound = errors.New("not found")

	// Create a wrapped error
	err := fmt.Errorf("user database: %w", ErrNotFound)

	// Check if ErrNotFound is in the error chain
	ErrorIs(t, err, ErrNotFound)

	// Output:
}

// ExampleErrorAs demonstrates error type checking.
func ExampleErrorAs() {
	t := &mockT{}

	// Create a PathError
	err := &os.PathError{
		Op:   "open",
		Path: "/tmp/missing",
		Err:  errors.New("no such file"),
	}

	// Check if error is of a specific type
	var pathErr *os.PathError
	ErrorAs(t, err, &pathErr)

	// Now pathErr is populated and can be used
	_ = pathErr.Path // "/tmp/missing"

	// Output:
}

// ExampleContains demonstrates containment checks.
func ExampleContains() {
	t := &mockT{}

	// String contains substring
	Contains(t, "hello world", "world")

	// Slice contains element
	Contains(t, []int{1, 2, 3}, 2)

	// Map contains key
	Contains(t, map[string]int{"foo": 1}, "foo")

	// Output:
}

// ExampleLen demonstrates length assertions.
func ExampleLen() {
	t := &mockT{}

	// Check slice length
	Len(t, []int{1, 2, 3}, 3)

	// Check string length
	Len(t, "hello", 5)

	// Check map length
	Len(t, map[string]int{"a": 1, "b": 2}, 2)

	// Output:
}

// ExampleEmpty demonstrates empty value checking.
func ExampleEmpty() {
	t := &mockT{}

	// Empty slice
	Empty(t, []int{})

	// Empty string
	Empty(t, "")

	// Nil value
	Empty(t, nil)

	// Output:
}

// ExampleGreater demonstrates comparison assertions.
func ExampleGreater() {
	t := &mockT{}

	// Compare integers
	Greater(t, 10, 5)

	// Compare floats
	Greater(t, 3.14, 2.71)

	// Compare strings (lexicographic)
	Greater(t, "banana", "apple")

	// Output:
}

// ExampleInDelta demonstrates floating-point comparison.
func ExampleInDelta() {
	t := &mockT{}

	// Compare floats with tolerance
	result := 3.14159
	InDelta(t, result, 3.14, 0.01)

	// Useful for calculations that might have rounding errors
	calculated := 1.0 / 3.0 * 3.0
	InDelta(t, calculated, 1.0, 0.000001)

	// Output:
}

// ExampleRegexp demonstrates pattern matching.
func ExampleRegexp() {
	t := &mockT{}

	// Match with string pattern
	Regexp(t, `^\d{3}-\d{4}$`, "555-1234")

	// Match with compiled pattern
	pattern := regexp.MustCompile(`[A-Z][a-z]+`)
	Regexp(t, pattern, "Hello")

	// Output:
}

// ExamplePanics demonstrates panic assertions.
func ExamplePanics() {
	t := &mockT{}

	// Assert that function panics
	Panics(t, func() {
		panic("oops!")
	})

	// Assert that function doesn't panic
	NotPanics(t, func() {
		// Normal execution
	})

	// Output:
}

// mockT captures fatal calls for testing assertions.
type mockT struct {
	testing.TB
	failed   bool
	fatalMsg string
}

func (m *mockT) Helper()                           {}
func (m *mockT) Fatalf(format string, args ...any) { m.failed = true; m.fatalMsg = fmt.Sprintf(format, args...) }
func (m *mockT) Errorf(format string, args ...any) { m.failed = true }
func (m *mockT) FailNow()                          { m.failed = true }
func (m *mockT) Failed() bool                      { return m.failed }
func (m *mockT) Name() string                      { return "mockT" }

func TestEqual(t *testing.T) {
	t.Run("equal values pass", func(t *testing.T) {
		m := &mockT{}
		Equal(m, 42, 42)
		if m.failed {
			t.Error("Equal should pass for equal values")
		}
	})

	t.Run("unequal values fail", func(t *testing.T) {
		m := &mockT{}
		Equal(m, 42, 43)
		if !m.failed {
			t.Error("Equal should fail for unequal values")
		}
		if !strings.Contains(m.fatalMsg, "mismatch") {
			t.Errorf("expected diff message, got: %s", m.fatalMsg)
		}
	})

	t.Run("structs with diff", func(t *testing.T) {
		type User struct {
			Name string
			Age  int
		}
		m := &mockT{}
		SetColorEnabled(false)
		defer SetColorEnabled(true)
		Equal(m, User{Name: "Alice", Age: 30}, User{Name: "Alice", Age: 31})
		if !m.failed {
			t.Error("Equal should fail for different structs")
		}
		if !strings.Contains(m.fatalMsg, "Age") {
			t.Errorf("expected Age in diff, got: %s", m.fatalMsg)
		}
	})

	t.Run("with cmp options", func(t *testing.T) {
		type User struct {
			Name     string
			internal int
		}
		m := &mockT{}
		EqualOpts(m, User{Name: "A", internal: 1}, User{Name: "A", internal: 1}, gocmp.AllowUnexported(User{}))
		if m.failed {
			t.Error("EqualOpts should pass with cmp options")
		}
	})
}

func TestNotEqual(t *testing.T) {
	t.Run("different values pass", func(t *testing.T) {
		m := &mockT{}
		NotEqual(m, 42, 43)
		if m.failed {
			t.Error("NotEqual should pass for different values")
		}
	})

	t.Run("equal values fail", func(t *testing.T) {
		m := &mockT{}
		NotEqual(m, 42, 42)
		if !m.failed {
			t.Error("NotEqual should fail for equal values")
		}
	})
}

func TestNoError(t *testing.T) {
	t.Run("nil error passes", func(t *testing.T) {
		m := &mockT{}
		NoError(m, nil)
		if m.failed {
			t.Error("NoError should pass for nil error")
		}
	})

	t.Run("non-nil error fails", func(t *testing.T) {
		m := &mockT{}
		NoError(m, errors.New("oops"))
		if !m.failed {
			t.Error("NoError should fail for non-nil error")
		}
		if !strings.Contains(m.fatalMsg, "oops") {
			t.Errorf("expected error message, got: %s", m.fatalMsg)
		}
	})

	t.Run("with message", func(t *testing.T) {
		m := &mockT{}
		NoError(m, errors.New("oops"), "doing %s", "something")
		if !strings.Contains(m.fatalMsg, "doing something") {
			t.Errorf("expected custom message, got: %s", m.fatalMsg)
		}
	})
}

func TestError(t *testing.T) {
	t.Run("non-nil error passes", func(t *testing.T) {
		m := &mockT{}
		Error(m, errors.New("expected"))
		if m.failed {
			t.Error("Error should pass for non-nil error")
		}
	})

	t.Run("nil error fails", func(t *testing.T) {
		m := &mockT{}
		Error(m, nil)
		if !m.failed {
			t.Error("Error should fail for nil error")
		}
	})
}

func TestErrorIs(t *testing.T) {
	baseErr := errors.New("base")
	wrappedErr := fmt.Errorf("wrapped: %w", baseErr)

	t.Run("matching error passes", func(t *testing.T) {
		m := &mockT{}
		ErrorIs(m, wrappedErr, baseErr)
		if m.failed {
			t.Error("ErrorIs should pass when target is in chain")
		}
	})

	t.Run("non-matching error fails", func(t *testing.T) {
		m := &mockT{}
		ErrorIs(m, errors.New("other"), baseErr)
		if !m.failed {
			t.Error("ErrorIs should fail when target is not in chain")
		}
	})
}

func TestErrorContains(t *testing.T) {
	t.Run("contains substring passes", func(t *testing.T) {
		m := &mockT{}
		ErrorContains(m, errors.New("file not found"), "not found")
		if m.failed {
			t.Error("ErrorContains should pass when substring present")
		}
	})

	t.Run("missing substring fails", func(t *testing.T) {
		m := &mockT{}
		ErrorContains(m, errors.New("file not found"), "permission")
		if !m.failed {
			t.Error("ErrorContains should fail when substring missing")
		}
	})

	t.Run("nil error fails", func(t *testing.T) {
		m := &mockT{}
		ErrorContains(m, nil, "anything")
		if !m.failed {
			t.Error("ErrorContains should fail for nil error")
		}
	})
}

func TestNil(t *testing.T) {
	t.Run("nil passes", func(t *testing.T) {
		m := &mockT{}
		Nil(m, nil)
		if m.failed {
			t.Error("Nil should pass for nil")
		}
	})

	t.Run("nil pointer passes", func(t *testing.T) {
		m := &mockT{}
		var p *int
		Nil(m, p)
		if m.failed {
			t.Error("Nil should pass for nil pointer")
		}
	})

	t.Run("non-nil fails", func(t *testing.T) {
		m := &mockT{}
		Nil(m, 42)
		if !m.failed {
			t.Error("Nil should fail for non-nil value")
		}
	})
}

func TestNotNil(t *testing.T) {
	t.Run("non-nil passes", func(t *testing.T) {
		m := &mockT{}
		NotNil(m, 42)
		if m.failed {
			t.Error("NotNil should pass for non-nil value")
		}
	})

	t.Run("nil fails", func(t *testing.T) {
		m := &mockT{}
		NotNil(m, nil)
		if !m.failed {
			t.Error("NotNil should fail for nil")
		}
	})
}

func TestTrue(t *testing.T) {
	t.Run("true passes", func(t *testing.T) {
		m := &mockT{}
		True(m, true)
		if m.failed {
			t.Error("True should pass for true")
		}
	})

	t.Run("false fails", func(t *testing.T) {
		m := &mockT{}
		True(m, false)
		if !m.failed {
			t.Error("True should fail for false")
		}
	})

	t.Run("with message", func(t *testing.T) {
		m := &mockT{}
		True(m, false, "value should be %d", 42)
		if !strings.Contains(m.fatalMsg, "value should be 42") {
			t.Errorf("expected custom message, got: %s", m.fatalMsg)
		}
	})
}

func TestFalse(t *testing.T) {
	t.Run("false passes", func(t *testing.T) {
		m := &mockT{}
		False(m, false)
		if m.failed {
			t.Error("False should pass for false")
		}
	})

	t.Run("true fails", func(t *testing.T) {
		m := &mockT{}
		False(m, true)
		if !m.failed {
			t.Error("False should fail for true")
		}
	})
}

func TestContains(t *testing.T) {
	t.Run("string contains", func(t *testing.T) {
		m := &mockT{}
		Contains(m, "hello world", "world")
		if m.failed {
			t.Error("Contains should pass for substring")
		}
	})

	t.Run("string not contains", func(t *testing.T) {
		m := &mockT{}
		Contains(m, "hello world", "foo")
		if !m.failed {
			t.Error("Contains should fail for missing substring")
		}
	})

	t.Run("slice contains", func(t *testing.T) {
		m := &mockT{}
		Contains(m, []int{1, 2, 3}, 2)
		if m.failed {
			t.Error("Contains should pass for element in slice")
		}
	})

	t.Run("slice not contains", func(t *testing.T) {
		m := &mockT{}
		Contains(m, []int{1, 2, 3}, 4)
		if !m.failed {
			t.Error("Contains should fail for missing element")
		}
	})

	t.Run("map contains key", func(t *testing.T) {
		m := &mockT{}
		Contains(m, map[string]int{"a": 1}, "a")
		if m.failed {
			t.Error("Contains should pass for key in map")
		}
	})
}

func TestNotContains(t *testing.T) {
	t.Run("string not contains passes", func(t *testing.T) {
		m := &mockT{}
		NotContains(m, "hello world", "foo")
		if m.failed {
			t.Error("NotContains should pass for missing substring")
		}
	})

	t.Run("string contains fails", func(t *testing.T) {
		m := &mockT{}
		NotContains(m, "hello world", "world")
		if !m.failed {
			t.Error("NotContains should fail for present substring")
		}
	})
}

func TestLen(t *testing.T) {
	t.Run("correct length passes", func(t *testing.T) {
		m := &mockT{}
		Len(m, []int{1, 2, 3}, 3)
		if m.failed {
			t.Error("Len should pass for correct length")
		}
	})

	t.Run("incorrect length fails", func(t *testing.T) {
		m := &mockT{}
		Len(m, []int{1, 2, 3}, 2)
		if !m.failed {
			t.Error("Len should fail for incorrect length")
		}
	})

	t.Run("string length", func(t *testing.T) {
		m := &mockT{}
		Len(m, "hello", 5)
		if m.failed {
			t.Error("Len should work with strings")
		}
	})

	t.Run("map length", func(t *testing.T) {
		m := &mockT{}
		Len(m, map[string]int{"a": 1, "b": 2}, 2)
		if m.failed {
			t.Error("Len should work with maps")
		}
	})
}

func TestEmpty(t *testing.T) {
	t.Run("nil is empty", func(t *testing.T) {
		m := &mockT{}
		Empty(m, nil)
		if m.failed {
			t.Error("Empty should pass for nil")
		}
	})

	t.Run("empty slice is empty", func(t *testing.T) {
		m := &mockT{}
		Empty(m, []int{})
		if m.failed {
			t.Error("Empty should pass for empty slice")
		}
	})

	t.Run("empty string is empty", func(t *testing.T) {
		m := &mockT{}
		Empty(m, "")
		if m.failed {
			t.Error("Empty should pass for empty string")
		}
	})

	t.Run("non-empty fails", func(t *testing.T) {
		m := &mockT{}
		Empty(m, []int{1})
		if !m.failed {
			t.Error("Empty should fail for non-empty slice")
		}
	})
}

func TestNotEmpty(t *testing.T) {
	t.Run("non-empty passes", func(t *testing.T) {
		m := &mockT{}
		NotEmpty(m, []int{1})
		if m.failed {
			t.Error("NotEmpty should pass for non-empty slice")
		}
	})

	t.Run("empty fails", func(t *testing.T) {
		m := &mockT{}
		NotEmpty(m, []int{})
		if !m.failed {
			t.Error("NotEmpty should fail for empty slice")
		}
	})
}

func TestPanics(t *testing.T) {
	t.Run("panic passes", func(t *testing.T) {
		m := &mockT{}
		Panics(m, func() { panic("oops") })
		if m.failed {
			t.Error("Panics should pass when function panics")
		}
	})

	t.Run("no panic fails", func(t *testing.T) {
		m := &mockT{}
		Panics(m, func() {})
		if !m.failed {
			t.Error("Panics should fail when function doesn't panic")
		}
	})
}

func TestNotPanics(t *testing.T) {
	t.Run("no panic passes", func(t *testing.T) {
		m := &mockT{}
		NotPanics(m, func() {})
		if m.failed {
			t.Error("NotPanics should pass when function doesn't panic")
		}
	})

	t.Run("panic fails", func(t *testing.T) {
		m := &mockT{}
		NotPanics(m, func() { panic("oops") })
		if !m.failed {
			t.Error("NotPanics should fail when function panics")
		}
	})
}

func TestGreater(t *testing.T) {
	t.Run("greater passes", func(t *testing.T) {
		m := &mockT{}
		Greater(m, 5, 3)
		if m.failed {
			t.Error("Greater should pass for 5 > 3")
		}
	})

	t.Run("equal fails", func(t *testing.T) {
		m := &mockT{}
		Greater(m, 3, 3)
		if !m.failed {
			t.Error("Greater should fail for 3 > 3")
		}
	})

	t.Run("less fails", func(t *testing.T) {
		m := &mockT{}
		Greater(m, 2, 3)
		if !m.failed {
			t.Error("Greater should fail for 2 > 3")
		}
	})

	t.Run("floats", func(t *testing.T) {
		m := &mockT{}
		Greater(m, 3.14, 2.71)
		if m.failed {
			t.Error("Greater should work with floats")
		}
	})

	t.Run("strings", func(t *testing.T) {
		m := &mockT{}
		Greater(m, "b", "a")
		if m.failed {
			t.Error("Greater should work with strings")
		}
	})
}

func TestGreaterOrEqual(t *testing.T) {
	t.Run("greater passes", func(t *testing.T) {
		m := &mockT{}
		GreaterOrEqual(m, 5, 3)
		if m.failed {
			t.Error("GreaterOrEqual should pass for 5 >= 3")
		}
	})

	t.Run("equal passes", func(t *testing.T) {
		m := &mockT{}
		GreaterOrEqual(m, 3, 3)
		if m.failed {
			t.Error("GreaterOrEqual should pass for 3 >= 3")
		}
	})

	t.Run("less fails", func(t *testing.T) {
		m := &mockT{}
		GreaterOrEqual(m, 2, 3)
		if !m.failed {
			t.Error("GreaterOrEqual should fail for 2 >= 3")
		}
	})
}

func TestLess(t *testing.T) {
	t.Run("less passes", func(t *testing.T) {
		m := &mockT{}
		Less(m, 2, 3)
		if m.failed {
			t.Error("Less should pass for 2 < 3")
		}
	})

	t.Run("equal fails", func(t *testing.T) {
		m := &mockT{}
		Less(m, 3, 3)
		if !m.failed {
			t.Error("Less should fail for 3 < 3")
		}
	})

	t.Run("greater fails", func(t *testing.T) {
		m := &mockT{}
		Less(m, 5, 3)
		if !m.failed {
			t.Error("Less should fail for 5 < 3")
		}
	})
}

func TestLessOrEqual(t *testing.T) {
	t.Run("less passes", func(t *testing.T) {
		m := &mockT{}
		LessOrEqual(m, 2, 3)
		if m.failed {
			t.Error("LessOrEqual should pass for 2 <= 3")
		}
	})

	t.Run("equal passes", func(t *testing.T) {
		m := &mockT{}
		LessOrEqual(m, 3, 3)
		if m.failed {
			t.Error("LessOrEqual should pass for 3 <= 3")
		}
	})

	t.Run("greater fails", func(t *testing.T) {
		m := &mockT{}
		LessOrEqual(m, 5, 3)
		if !m.failed {
			t.Error("LessOrEqual should fail for 5 <= 3")
		}
	})
}

func TestFormatDiff(t *testing.T) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)

	diff := "-old\n+new\n context"
	result := formatDiff(diff)

	if !strings.Contains(result, "mismatch (-want +got):") {
		t.Error("formatDiff should include header")
	}
	if !strings.Contains(result, "-old") {
		t.Error("formatDiff should include removed lines")
	}
	if !strings.Contains(result, "+new") {
		t.Error("formatDiff should include added lines")
	}
}

func TestIsNil(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"nil interface", nil, true},
		{"nil pointer", (*int)(nil), true},
		{"nil slice", ([]int)(nil), true},
		{"nil map", (map[string]int)(nil), true},
		{"nil chan", (chan int)(nil), true},
		{"nil func", (func())(nil), true},
		{"non-nil int", 42, false},
		{"non-nil pointer", new(int), false},
		{"empty slice", []int{}, false},
		{"empty map", map[string]int{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNil(tt.v); got != tt.want {
				t.Errorf("isNil(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"nil", nil, true},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"nil slice", ([]int)(nil), true},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1}, false},
		{"nil map", (map[string]int)(nil), true},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmpty(tt.v); got != tt.want {
				t.Errorf("isEmpty(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestRegexp(t *testing.T) {
	t.Run("string pattern matches", func(t *testing.T) {
		m := &mockT{}
		Regexp(m, `\d+`, "abc123def")
		if m.failed {
			t.Error("Regexp should pass when pattern matches")
		}
	})

	t.Run("string pattern does not match", func(t *testing.T) {
		m := &mockT{}
		Regexp(m, `\d+`, "abcdef")
		if !m.failed {
			t.Error("Regexp should fail when pattern does not match")
		}
		if !strings.Contains(m.fatalMsg, "match pattern") {
			t.Errorf("expected match pattern message, got: %s", m.fatalMsg)
		}
	})

	t.Run("compiled regexp matches", func(t *testing.T) {
		m := &mockT{}
		re := regexp.MustCompile(`[A-Z][a-z]+`)
		Regexp(m, re, "Hello World")
		if m.failed {
			t.Error("Regexp should pass when compiled pattern matches")
		}
	})

	t.Run("compiled regexp does not match", func(t *testing.T) {
		m := &mockT{}
		re := regexp.MustCompile(`^[0-9]+$`)
		Regexp(m, re, "abc")
		if !m.failed {
			t.Error("Regexp should fail when compiled pattern does not match")
		}
	})

	t.Run("with custom message", func(t *testing.T) {
		m := &mockT{}
		Regexp(m, `^test$`, "nottest", "checking %s", "value")
		if !strings.Contains(m.fatalMsg, "checking value") {
			t.Errorf("expected custom message, got: %s", m.fatalMsg)
		}
	})
}

func TestInDeltaEdgeCases(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		m := &mockT{}
		InDelta(m, 3.14, 3.14, 0.0)
		if m.failed {
			t.Error("InDelta should pass for exact match")
		}
	})

	t.Run("negative delta", func(t *testing.T) {
		m := &mockT{}
		InDelta(m, 5.0, 4.0, 1.0)
		if m.failed {
			t.Error("InDelta should pass when diff is within delta")
		}
	})

	t.Run("fails outside delta", func(t *testing.T) {
		m := &mockT{}
		InDelta(m, 5.0, 3.0, 1.0)
		if !m.failed {
			t.Error("InDelta should fail when diff exceeds delta")
		}
	})

	t.Run("with custom message", func(t *testing.T) {
		m := &mockT{}
		InDelta(m, 10.0, 1.0, 0.1, "comparing %s", "values")
		if !strings.Contains(m.fatalMsg, "comparing values") {
			t.Errorf("expected custom message, got: %s", m.fatalMsg)
		}
	})
}

func TestFormatDiffWithColors(t *testing.T) {
	// Test with colors enabled
	SetColorEnabled(true)
	defer SetColorEnabled(false)

	diff := "-old line\n+new line\n context"
	result := formatDiff(diff)

	if !strings.Contains(result, "mismatch (-want +got):") {
		t.Error("formatDiff should include header")
	}
	// With colors enabled, ANSI codes should be present
	if !strings.Contains(result, "\033[") {
		t.Error("formatDiff should include ANSI codes when colors are enabled")
	}
}

func TestFormatMsgEdgeCases(t *testing.T) {
	t.Run("empty args returns empty", func(t *testing.T) {
		result := formatMsg()
		if result != "" {
			t.Errorf("formatMsg() should return empty string, got: %s", result)
		}
	})

	t.Run("single non-string arg", func(t *testing.T) {
		result := formatMsg(42)
		if result != "42" {
			t.Errorf("formatMsg(42) should return '42', got: %s", result)
		}
	})

	t.Run("non-string first arg with more args returns empty", func(t *testing.T) {
		result := formatMsg(42, "ignored")
		if result != "" {
			t.Errorf("formatMsg(42, 'ignored') should return empty, got: %s", result)
		}
	})
}

// customTestError is used for ErrorAs tests
type customTestError struct {
	Code int
	Msg  string
}

func (e *customTestError) Error() string {
	return e.Msg
}

func TestErrorAsWithCustomType(t *testing.T) {
	baseErr := &customTestError{Code: 404, Msg: "not found"}
	wrappedErr := fmt.Errorf("wrapped: %w", baseErr)

	t.Run("matching type passes", func(t *testing.T) {
		m := &mockT{}
		var target *customTestError
		ErrorAs(m, wrappedErr, &target)
		if m.failed {
			t.Error("ErrorAs should pass when target type is in chain")
		}
	})

	t.Run("non-matching type fails", func(t *testing.T) {
		m := &mockT{}
		var target *os.PathError
		ErrorAs(m, baseErr, &target)
		if !m.failed {
			t.Error("ErrorAs should fail when target type is not in chain")
		}
	})

	t.Run("with custom message", func(t *testing.T) {
		m := &mockT{}
		var target *os.PathError
		ErrorAs(m, baseErr, &target, "checking error type")
		if !strings.Contains(m.fatalMsg, "checking error type") {
			t.Errorf("expected custom message, got: %s", m.fatalMsg)
		}
	})
}

func TestContainsWithArray(t *testing.T) {
	t.Run("array contains element", func(t *testing.T) {
		m := &mockT{}
		arr := [3]int{1, 2, 3}
		Contains(m, arr, 2)
		if m.failed {
			t.Error("Contains should pass for element in array")
		}
	})

	t.Run("array does not contain element", func(t *testing.T) {
		m := &mockT{}
		arr := [3]int{1, 2, 3}
		Contains(m, arr, 4)
		if !m.failed {
			t.Error("Contains should fail for missing element in array")
		}
	})
}

func TestNotContainsWithSliceAndMap(t *testing.T) {
	t.Run("slice not contains", func(t *testing.T) {
		m := &mockT{}
		NotContains(m, []int{1, 2, 3}, 4)
		if m.failed {
			t.Error("NotContains should pass for missing element")
		}
	})

	t.Run("map key not contains", func(t *testing.T) {
		m := &mockT{}
		NotContains(m, map[string]int{"a": 1}, "b")
		if m.failed {
			t.Error("NotContains should pass for missing key")
		}
	})

	t.Run("map key contains fails", func(t *testing.T) {
		m := &mockT{}
		NotContains(m, map[string]int{"a": 1}, "a")
		if !m.failed {
			t.Error("NotContains should fail for present key")
		}
	})
}

func TestIsEmptyWithPointer(t *testing.T) {
	t.Run("nil pointer is empty", func(t *testing.T) {
		var p *int
		if !isEmpty(p) {
			t.Error("nil pointer should be empty")
		}
	})

	t.Run("pointer to zero value is empty", func(t *testing.T) {
		zero := 0
		if !isEmpty(&zero) {
			t.Error("pointer to zero value should be empty")
		}
	})

	t.Run("pointer to non-zero value is not empty", func(t *testing.T) {
		val := 42
		if isEmpty(&val) {
			t.Error("pointer to non-zero value should not be empty")
		}
	})

	t.Run("empty chan is empty", func(t *testing.T) {
		ch := make(chan int)
		if !isEmpty(ch) {
			t.Error("empty channel should be empty")
		}
	})
}

func TestEqualWithMessage(t *testing.T) {
	t.Run("single string message", func(t *testing.T) {
		m := &mockT{}
		SetColorEnabled(false)
		defer SetColorEnabled(true)
		Equal(m, 1, 2, "custom message")
		if !strings.Contains(m.fatalMsg, "custom message") {
			t.Errorf("expected custom message, got: %s", m.fatalMsg)
		}
	})

	t.Run("formatted message", func(t *testing.T) {
		m := &mockT{}
		SetColorEnabled(false)
		defer SetColorEnabled(true)
		Equal(m, 1, 2, "value was %d", 42)
		if !strings.Contains(m.fatalMsg, "value was 42") {
			t.Errorf("expected formatted message, got: %s", m.fatalMsg)
		}
	})
}

func TestNotEqualWithMessage(t *testing.T) {
	m := &mockT{}
	NotEqual(m, 42, 42, "values should differ")
	if !strings.Contains(m.fatalMsg, "values should differ") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestNilWithMessage(t *testing.T) {
	m := &mockT{}
	Nil(m, 42, "expected nil")
	if !strings.Contains(m.fatalMsg, "expected nil") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestNotNilWithMessage(t *testing.T) {
	m := &mockT{}
	NotNil(m, nil, "expected non-nil")
	if !strings.Contains(m.fatalMsg, "expected non-nil") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestFalseWithMessage(t *testing.T) {
	m := &mockT{}
	False(m, true, "should be false")
	if !strings.Contains(m.fatalMsg, "should be false") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestEmptyWithMessage(t *testing.T) {
	m := &mockT{}
	Empty(m, []int{1}, "should be empty")
	if !strings.Contains(m.fatalMsg, "should be empty") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestNotEmptyWithMessage(t *testing.T) {
	m := &mockT{}
	NotEmpty(m, []int{}, "should not be empty")
	if !strings.Contains(m.fatalMsg, "should not be empty") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestContainsWithMessage(t *testing.T) {
	m := &mockT{}
	Contains(m, "hello", "xyz", "substring check")
	if !strings.Contains(m.fatalMsg, "substring check") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestNotContainsWithMessage(t *testing.T) {
	m := &mockT{}
	NotContains(m, "hello world", "world", "exclusion check")
	if !strings.Contains(m.fatalMsg, "exclusion check") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestLenWithMessage(t *testing.T) {
	m := &mockT{}
	Len(m, []int{1, 2}, 3, "length mismatch")
	if !strings.Contains(m.fatalMsg, "length mismatch") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestPanicsWithMessage(t *testing.T) {
	m := &mockT{}
	Panics(m, func() {}, "should panic")
	if !strings.Contains(m.fatalMsg, "should panic") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestNotPanicsWithMessage(t *testing.T) {
	m := &mockT{}
	NotPanics(m, func() { panic("oops") }, "should not panic")
	if !strings.Contains(m.fatalMsg, "should not panic") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestGreaterWithMessage(t *testing.T) {
	m := &mockT{}
	Greater(m, 1, 2, "comparison check")
	if !strings.Contains(m.fatalMsg, "comparison check") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestGreaterOrEqualWithMessage(t *testing.T) {
	m := &mockT{}
	GreaterOrEqual(m, 1, 2, "comparison check")
	if !strings.Contains(m.fatalMsg, "comparison check") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestLessWithMessage(t *testing.T) {
	m := &mockT{}
	Less(m, 2, 1, "comparison check")
	if !strings.Contains(m.fatalMsg, "comparison check") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestLessOrEqualWithMessage(t *testing.T) {
	m := &mockT{}
	LessOrEqual(m, 2, 1, "comparison check")
	if !strings.Contains(m.fatalMsg, "comparison check") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestErrorWithMessage(t *testing.T) {
	m := &mockT{}
	Error(m, nil, "expected error")
	if !strings.Contains(m.fatalMsg, "expected error") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestErrorIsWithMessage(t *testing.T) {
	m := &mockT{}
	ErrorIs(m, errors.New("other"), errors.New("target"), "error chain check")
	if !strings.Contains(m.fatalMsg, "error chain check") {
		t.Errorf("expected custom message, got: %s", m.fatalMsg)
	}
}

func TestErrorContainsWithMessage(t *testing.T) {
	t.Run("nil error with message", func(t *testing.T) {
		m := &mockT{}
		ErrorContains(m, nil, "anything", "nil error check")
		if !strings.Contains(m.fatalMsg, "nil error check") {
			t.Errorf("expected custom message, got: %s", m.fatalMsg)
		}
	})

	t.Run("missing substring with message", func(t *testing.T) {
		m := &mockT{}
		ErrorContains(m, errors.New("actual error"), "expected", "substring check")
		if !strings.Contains(m.fatalMsg, "substring check") {
			t.Errorf("expected custom message, got: %s", m.fatalMsg)
		}
	})
}
