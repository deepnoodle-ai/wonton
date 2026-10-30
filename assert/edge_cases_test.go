package assert

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
)

type userWithSecret struct {
	Name      string
	LastLogin int
	secret    int
}

// H21: EqualOpts compares unexported fields like Equal instead of panicking.
func TestEqualOptsUnexportedFields(t *testing.T) {
	ignore := cmpopts.IgnoreFields(userWithSecret{}, "LastLogin")
	EqualOpts(t, userWithSecret{Name: "a", LastLogin: 1}, userWithSecret{Name: "a", LastLogin: 2}, ignore)

	m := &mockT{}
	EqualOpts(m, userWithSecret{Name: "a", secret: 1}, userWithSecret{Name: "a", secret: 2}, ignore)
	True(t, m.failed, "EqualOpts must see a difference in an unexported field")

	// A caller's own EquateErrors does not clash with the defaults.
	err := errors.New("x")
	EqualOpts(t, err, err, cmpopts.EquateErrors(), ignore)
}

// H22: Empty and NotEmpty stop at a pointer cycle instead of overflowing.
func TestEmptyCyclicPointer(t *testing.T) {
	type P *P
	var p P
	p = &p
	NotEmpty(t, p)

	var x any
	x = &x
	NotEmpty(t, &x)

	var n *int
	Empty(t, &n)
	zero := 0
	ptr := &zero
	Empty(t, &ptr)
}

// M35: Contains uses the same equality rule as Equal.
func TestContainsMatchesEqual(t *testing.T) {
	Equal(t, errors.New("x"), errors.New("x"))
	Contains(t, []error{errors.New("x")}, errors.New("x"))

	m := &mockT{}
	NotContains(m, []error{errors.New("x")}, errors.New("x"))
	True(t, m.failed, "NotContains must fail when Equal would pass")
}
