package assert

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestEqual_TimeEquality(t *testing.T) {
	// reflect.DeepEqual reports these as unequal (monotonic clock reading vs
	// not), but go-cmp considers them equal via time.Time's Equal method.
	// Equal should pass rather than failing with an empty diff.
	now := time.Now()
	m := &mockT{}
	Equal(m, now, now.UTC())
	if m.failed {
		t.Errorf("Equal should pass for equivalent times, got: %s", m.fatalMsg)
	}
}

func TestNotEqual_UnexportedFields(t *testing.T) {
	type private struct {
		name string
		age  int
	}

	t.Run("different values pass without panic", func(t *testing.T) {
		m := &mockT{}
		NotEqual(m, private{name: "a", age: 1}, private{name: "b", age: 2})
		if m.failed {
			t.Error("NotEqual should pass for different structs with unexported fields")
		}
	})

	t.Run("equal values fail without panic", func(t *testing.T) {
		m := &mockT{}
		NotEqual(m, private{name: "a", age: 1}, private{name: "a", age: 1})
		if !m.failed {
			t.Error("NotEqual should fail for equal structs with unexported fields")
		}
	})
}

func TestInDelta_NaN(t *testing.T) {
	cases := []struct {
		name                    string
		expected, actual, delta float64
	}{
		{"NaN expected", math.NaN(), 1.0, 0.1},
		{"NaN actual", 1.0, math.NaN(), 0.1},
		{"both NaN", math.NaN(), math.NaN(), 0.1},
		{"NaN delta", 1.0, 1.0, math.NaN()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &mockT{}
			InDelta(m, tc.expected, tc.actual, tc.delta)
			if !m.failed {
				t.Error("InDelta should fail for NaN arguments")
			}
		})
	}
}

func TestRegexp_InvalidPattern(t *testing.T) {
	m := &mockT{}
	NotPanics(t, func() {
		Regexp(m, `[unclosed`, "anything")
	})
	if !m.failed {
		t.Error("Regexp should fail for an invalid pattern")
	}
	if !strings.Contains(m.fatalMsg, "invalid regexp") {
		t.Errorf("expected invalid pattern message, got: %s", m.fatalMsg)
	}
}

func TestLen_UnsupportedType(t *testing.T) {
	m := &mockT{}
	NotPanics(t, func() {
		Len(m, 42, 2)
	})
	if !m.failed {
		t.Error("Len should fail for a type without a length")
	}
	if !strings.Contains(m.fatalMsg, "cannot get length") {
		t.Errorf("expected length error message, got: %s", m.fatalMsg)
	}
}

func TestContains_NonStringNeedleInString(t *testing.T) {
	// A non-string needle can never be a substring; previously this compared
	// against reflect's "<int Value>" placeholder text.
	m := &mockT{}
	Contains(m, "value: <int Value>", 42)
	if !m.failed {
		t.Error("Contains should fail for a non-string needle in a string haystack")
	}
}
