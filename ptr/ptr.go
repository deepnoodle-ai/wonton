// Package ptr provides small generic helpers for working with pointers.
//
// Code that speaks JSON or talks to generated API clients constantly needs to
// box a scalar into a pointer (so an optional field can be omitted) or unbox a
// pointer to its zero value when nil. Centralizing those one-liners keeps them
// out of every package that needs them.
package ptr

// To returns a pointer to v. It is the expression form of taking the address
// of a variable, so a literal can be passed to an optional field directly:
//
//	req.Limit = ptr.To(50)
func To[T any](v T) *T { return &v }

// Deref returns *p, or the zero value of T when p is nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// Or returns *p, or fallback when p is nil.
func Or[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

// IfNotZero returns a pointer to v when v is not the zero value of T, and nil
// otherwise. Useful for optional fields that should be omitted when empty.
func IfNotZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// DerefSlice returns *p, or nil when p is nil. Useful when generated code
// boxes a slice field as *[]T and the caller wants to range over it safely.
func DerefSlice[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
}

// DerefMap returns *p, or nil when p is nil.
func DerefMap[K comparable, V any](p *map[K]V) map[K]V {
	if p == nil {
		return nil
	}
	return *p
}

// MapIfNotEmpty returns &m when m has entries, and nil otherwise. Mirrors the
// shape of an optional JSON object field.
func MapIfNotEmpty[K comparable, V any](m map[K]V) *map[K]V {
	if len(m) == 0 {
		return nil
	}
	return &m
}

// SliceIfNotEmpty returns a pointer to a copy of values when values is
// non-empty, and nil otherwise. The copy means later mutation of the source
// slice does not affect the returned pointer's backing array.
func SliceIfNotEmpty[T any](values []T) *[]T {
	if len(values) == 0 {
		return nil
	}
	out := append([]T(nil), values...)
	return &out
}
