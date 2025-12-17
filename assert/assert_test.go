package assert

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
)

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
