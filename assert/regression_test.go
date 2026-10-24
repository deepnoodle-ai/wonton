package assert

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

type withPrivate struct {
	Name string
	id   int
}

// BUG-033: Contains and NotContains compare elements the way Equal does, so
// unexported fields and errors do not panic.
func TestContainsUnexportedFieldsAndErrors(t *testing.T) {
	err := errors.New("x")
	NotPanics(t, func() {
		m := &mockT{}
		Contains(m, []error{err}, err)
		if m.failed {
			t.Errorf("Contains([]error{err}, err) failed: %s", m.fatalMsg)
		}
	})
	NotPanics(t, func() {
		m := &mockT{}
		Contains(m, []withPrivate{{Name: "a", id: 1}}, withPrivate{Name: "a", id: 1})
		if m.failed {
			t.Errorf("Contains with unexported fields failed: %s", m.fatalMsg)
		}
		m = &mockT{}
		NotContains(m, []withPrivate{{Name: "a", id: 1}}, withPrivate{Name: "a", id: 2})
		if m.failed {
			t.Errorf("NotContains with unexported fields failed: %s", m.fatalMsg)
		}
		m = &mockT{}
		Contains(m, map[withPrivate]bool{{Name: "k", id: 3}: true}, withPrivate{Name: "k", id: 3})
		if m.failed {
			t.Errorf("Contains on map keys failed: %s", m.fatalMsg)
		}
	})
}

// BUG-033: a nil *regexp.Regexp fails the assertion instead of panicking.
func TestRegexpNilPattern(t *testing.T) {
	m := &mockT{}
	NotPanics(t, func() {
		Regexp(m, (*regexp.Regexp)(nil), "abc")
	})
	if !m.failed || !strings.Contains(m.fatalMsg, "nil") {
		t.Errorf("expected nil pattern failure, got failed=%v msg=%q", m.failed, m.fatalMsg)
	}
}
