package assert

import (
	"reflect"
	"regexp"
	"testing"
)

func TestObjectsAreEqual(t *testing.T) {
	tests := []struct {
		name     string
		expected any
		actual   any
		want     bool
	}{
		{
			name:     "both nil",
			expected: nil,
			actual:   nil,
			want:     true,
		},
		{
			name:     "expected nil actual not",
			expected: nil,
			actual:   42,
			want:     false,
		},
		{
			name:     "expected not nil actual nil",
			expected: 42,
			actual:   nil,
			want:     false,
		},
		{
			name:     "equal ints",
			expected: 42,
			actual:   42,
			want:     true,
		},
		{
			name:     "unequal ints",
			expected: 42,
			actual:   43,
			want:     false,
		},
		{
			name:     "equal strings",
			expected: "hello",
			actual:   "hello",
			want:     true,
		},
		{
			name:     "unequal strings",
			expected: "hello",
			actual:   "world",
			want:     false,
		},
		{
			name:     "equal byte slices",
			expected: []byte{1, 2, 3},
			actual:   []byte{1, 2, 3},
			want:     true,
		},
		{
			name:     "unequal byte slices",
			expected: []byte{1, 2, 3},
			actual:   []byte{1, 2, 4},
			want:     false,
		},
		{
			name:     "byte slice vs non-byte",
			expected: []byte{1, 2, 3},
			actual:   "hello",
			want:     false,
		},
		{
			name:     "nil byte slice vs nil byte slice",
			expected: []byte(nil),
			actual:   []byte(nil),
			want:     true,
		},
		{
			name:     "nil byte slice vs empty byte slice",
			expected: []byte(nil),
			actual:   []byte{},
			want:     false,
		},
		{
			name:     "equal slices",
			expected: []int{1, 2, 3},
			actual:   []int{1, 2, 3},
			want:     true,
		},
		{
			name:     "equal maps",
			expected: map[string]int{"a": 1},
			actual:   map[string]int{"a": 1},
			want:     true,
		},
		{
			name:     "equal structs",
			expected: struct{ A int }{A: 1},
			actual:   struct{ A int }{A: 1},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := objectsAreEqual(tt.expected, tt.actual)
			if got != tt.want {
				t.Errorf("objectsAreEqual(%v, %v) = %v, want %v", tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

func TestValidateEqualArgs(t *testing.T) {
	tests := []struct {
		name     string
		expected any
		actual   any
		wantErr  bool
	}{
		{
			name:     "both nil",
			expected: nil,
			actual:   nil,
			wantErr:  false,
		},
		{
			name:     "normal values",
			expected: 42,
			actual:   43,
			wantErr:  false,
		},
		{
			name:     "expected is function",
			expected: func() {},
			actual:   42,
			wantErr:  true,
		},
		{
			name:     "actual is function",
			expected: 42,
			actual:   func() {},
			wantErr:  true,
		},
		{
			name:     "both functions",
			expected: func() {},
			actual:   func() {},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEqualArgs(tt.expected, tt.actual)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEqualArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsFunction(t *testing.T) {
	tests := []struct {
		name string
		arg  any
		want bool
	}{
		{
			name: "nil",
			arg:  nil,
			want: false,
		},
		{
			name: "function",
			arg:  func() {},
			want: true,
		},
		{
			name: "function with args",
			arg:  func(a int) int { return a },
			want: true,
		},
		{
			name: "int",
			arg:  42,
			want: false,
		},
		{
			name: "string",
			arg:  "hello",
			want: false,
		},
		{
			name: "slice",
			arg:  []int{1, 2, 3},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFunction(tt.arg)
			if got != tt.want {
				t.Errorf("isFunction(%v) = %v, want %v", tt.arg, got, tt.want)
			}
		})
	}
}

func TestIsNil(t *testing.T) {
	var nilPtr *int
	var nilSlice []int
	var nilMap map[string]int
	var nilChan chan int
	var nilFunc func()
	var nilInterface any

	tests := []struct {
		name   string
		object any
		want   bool
	}{
		{
			name:   "nil interface",
			object: nil,
			want:   true,
		},
		{
			name:   "nil pointer",
			object: nilPtr,
			want:   true,
		},
		{
			name:   "nil slice",
			object: nilSlice,
			want:   true,
		},
		{
			name:   "nil map",
			object: nilMap,
			want:   true,
		},
		{
			name:   "nil chan",
			object: nilChan,
			want:   true,
		},
		{
			name:   "nil func",
			object: nilFunc,
			want:   true,
		},
		{
			name:   "nil interface value",
			object: nilInterface,
			want:   true,
		},
		{
			name:   "non-nil int",
			object: 42,
			want:   false,
		},
		{
			name:   "non-nil string",
			object: "hello",
			want:   false,
		},
		{
			name:   "non-nil pointer",
			object: new(int),
			want:   false,
		},
		{
			name:   "non-nil slice",
			object: []int{1, 2, 3},
			want:   false,
		},
		{
			name:   "empty but non-nil slice",
			object: []int{},
			want:   false,
		},
		{
			name:   "non-nil map",
			object: map[string]int{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNil(tt.object)
			if got != tt.want {
				t.Errorf("isNil(%v) = %v, want %v", tt.object, got, tt.want)
			}
		})
	}
}

func TestSamePointers(t *testing.T) {
	a := new(int)
	b := new(int)
	*a = 42
	*b = 42

	tests := []struct {
		name     string
		first    any
		second   any
		wantSame bool
		wantOk   bool
	}{
		{
			name:     "same pointer",
			first:    a,
			second:   a,
			wantSame: true,
			wantOk:   true,
		},
		{
			name:     "different pointers same value",
			first:    a,
			second:   b,
			wantSame: false,
			wantOk:   true,
		},
		{
			name:     "non-pointer first",
			first:    42,
			second:   a,
			wantSame: false,
			wantOk:   false,
		},
		{
			name:     "non-pointer second",
			first:    a,
			second:   42,
			wantSame: false,
			wantOk:   false,
		},
		{
			name:     "different types",
			first:    new(int),
			second:   new(string),
			wantSame: false,
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSame, gotOk := samePointers(tt.first, tt.second)
			if gotSame != tt.wantSame {
				t.Errorf("samePointers() same = %v, want %v", gotSame, tt.wantSame)
			}
			if gotOk != tt.wantOk {
				t.Errorf("samePointers() ok = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	var nilPtr *int
	nonNilPtr := new(int)
	*nonNilPtr = 42

	tests := []struct {
		name   string
		object any
		want   bool
	}{
		{
			name:   "nil",
			object: nil,
			want:   true,
		},
		{
			name:   "zero int",
			object: 0,
			want:   true,
		},
		{
			name:   "non-zero int",
			object: 42,
			want:   false,
		},
		{
			name:   "empty string",
			object: "",
			want:   true,
		},
		{
			name:   "non-empty string",
			object: "hello",
			want:   false,
		},
		{
			name:   "nil slice",
			object: []int(nil),
			want:   true,
		},
		{
			name:   "empty slice",
			object: []int{},
			want:   true,
		},
		{
			name:   "non-empty slice",
			object: []int{1},
			want:   false,
		},
		{
			name:   "nil map",
			object: map[string]int(nil),
			want:   true,
		},
		{
			name:   "empty map",
			object: map[string]int{},
			want:   true,
		},
		{
			name:   "non-empty map",
			object: map[string]int{"a": 1},
			want:   false,
		},
		{
			name:   "nil pointer",
			object: nilPtr,
			want:   true,
		},
		{
			name:   "pointer to zero",
			object: new(int),
			want:   true,
		},
		{
			name:   "pointer to non-zero",
			object: nonNilPtr,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmpty(tt.object)
			if got != tt.want {
				t.Errorf("isEmpty(%v) = %v, want %v", tt.object, got, tt.want)
			}
		})
	}
}

func TestGetLen(t *testing.T) {
	tests := []struct {
		name    string
		x       any
		wantLen int
		wantOk  bool
	}{
		{
			name:    "slice",
			x:       []int{1, 2, 3},
			wantLen: 3,
			wantOk:  true,
		},
		{
			name:    "empty slice",
			x:       []int{},
			wantLen: 0,
			wantOk:  true,
		},
		{
			name:    "string",
			x:       "hello",
			wantLen: 5,
			wantOk:  true,
		},
		{
			name:    "map",
			x:       map[string]int{"a": 1, "b": 2},
			wantLen: 2,
			wantOk:  true,
		},
		{
			name:    "array",
			x:       [3]int{1, 2, 3},
			wantLen: 3,
			wantOk:  true,
		},
		{
			name:    "int has no length",
			x:       42,
			wantLen: 0,
			wantOk:  false, // panics and returns zero values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLen, gotOk := getLen(tt.x)
			if gotOk != tt.wantOk {
				t.Errorf("getLen() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotOk && gotLen != tt.wantLen {
				t.Errorf("getLen() len = %v, want %v", gotLen, tt.wantLen)
			}
		})
	}
}

func TestContainsElement(t *testing.T) {
	tests := []struct {
		name      string
		container any
		element   any
		wantOk    bool
		wantFound bool
	}{
		{
			name:      "slice contains element",
			container: []int{1, 2, 3},
			element:   2,
			wantOk:    true,
			wantFound: true,
		},
		{
			name:      "slice does not contain element",
			container: []int{1, 2, 3},
			element:   4,
			wantOk:    true,
			wantFound: false,
		},
		{
			name:      "string contains substring",
			container: "hello world",
			element:   "world",
			wantOk:    true,
			wantFound: true,
		},
		{
			name:      "string does not contain substring",
			container: "hello world",
			element:   "foo",
			wantOk:    true,
			wantFound: false,
		},
		{
			name:      "map contains key",
			container: map[string]int{"a": 1, "b": 2},
			element:   "a",
			wantOk:    true,
			wantFound: true,
		},
		{
			name:      "map does not contain key",
			container: map[string]int{"a": 1, "b": 2},
			element:   "c",
			wantOk:    true,
			wantFound: false,
		},
		{
			name:      "array contains element",
			container: [3]int{1, 2, 3},
			element:   2,
			wantOk:    true,
			wantFound: true,
		},
		{
			name:      "nil container",
			container: nil,
			element:   1,
			wantOk:    false,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, gotFound := containsElement(tt.container, tt.element)
			if gotOk != tt.wantOk {
				t.Errorf("containsElement() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotFound != tt.wantFound {
				t.Errorf("containsElement() found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestIsList(t *testing.T) {
	tests := []struct {
		name string
		list any
		want bool
	}{
		{
			name: "slice",
			list: []int{1, 2, 3},
			want: true,
		},
		{
			name: "array",
			list: [3]int{1, 2, 3},
			want: true,
		},
		{
			name: "string",
			list: "hello",
			want: false,
		},
		{
			name: "map",
			list: map[string]int{},
			want: false,
		},
		{
			name: "int",
			list: 42,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isList(tt.list)
			if got != tt.want {
				t.Errorf("isList(%v) = %v, want %v", tt.list, got, tt.want)
			}
		})
	}
}

func TestDiffLists(t *testing.T) {
	tests := []struct {
		name       string
		listA      any
		listB      any
		wantExtraA []any
		wantExtraB []any
	}{
		{
			name:       "equal lists",
			listA:      []int{1, 2, 3},
			listB:      []int{1, 2, 3},
			wantExtraA: nil,
			wantExtraB: nil,
		},
		{
			name:       "extra in A",
			listA:      []int{1, 2, 3, 4},
			listB:      []int{1, 2, 3},
			wantExtraA: []any{4},
			wantExtraB: nil,
		},
		{
			name:       "extra in B",
			listA:      []int{1, 2, 3},
			listB:      []int{1, 2, 3, 4},
			wantExtraA: nil,
			wantExtraB: []any{4},
		},
		{
			name:       "different elements",
			listA:      []int{1, 2, 3},
			listB:      []int{4, 5, 6},
			wantExtraA: []any{1, 2, 3},
			wantExtraB: []any{4, 5, 6},
		},
		{
			name:       "same elements different order",
			listA:      []int{1, 2, 3},
			listB:      []int{3, 2, 1},
			wantExtraA: nil,
			wantExtraB: nil,
		},
		{
			name:       "duplicates in A",
			listA:      []int{1, 1, 2},
			listB:      []int{1, 2},
			wantExtraA: []any{1},
			wantExtraB: nil,
		},
		{
			name:       "duplicates in B",
			listA:      []int{1, 2},
			listB:      []int{1, 1, 2},
			wantExtraA: nil,
			wantExtraB: []any{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExtraA, gotExtraB := diffLists(tt.listA, tt.listB)
			if !reflect.DeepEqual(gotExtraA, tt.wantExtraA) {
				t.Errorf("diffLists() extraA = %v, want %v", gotExtraA, tt.wantExtraA)
			}
			if !reflect.DeepEqual(gotExtraB, tt.wantExtraB) {
				t.Errorf("diffLists() extraB = %v, want %v", gotExtraB, tt.wantExtraB)
			}
		})
	}
}

func TestTypeAndKind(t *testing.T) {
	tests := []struct {
		name     string
		v        any
		wantKind reflect.Kind
	}{
		{
			name:     "int",
			v:        42,
			wantKind: reflect.Int,
		},
		{
			name:     "string",
			v:        "hello",
			wantKind: reflect.String,
		},
		{
			name:     "slice",
			v:        []int{1, 2, 3},
			wantKind: reflect.Slice,
		},
		{
			name:     "pointer to int",
			v:        new(int),
			wantKind: reflect.Int,
		},
		{
			name:     "pointer to string",
			v:        new(string),
			wantKind: reflect.String,
		},
		{
			name:     "pointer to slice",
			v:        new([]int),
			wantKind: reflect.Slice,
		},
		{
			name:     "struct",
			v:        struct{ A int }{},
			wantKind: reflect.Struct,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotKind := typeAndKind(tt.v)
			if gotKind != tt.wantKind {
				t.Errorf("typeAndKind() kind = %v, want %v", gotKind, tt.wantKind)
			}
		})
	}
}

func TestRecoverPanic(t *testing.T) {
	tests := []struct {
		name           string
		f              func()
		wantDidPanic   bool
		wantPanicValue any
	}{
		{
			name:           "no panic",
			f:              func() {},
			wantDidPanic:   false,
			wantPanicValue: nil,
		},
		{
			name:           "panic with string",
			f:              func() { panic("oops") },
			wantDidPanic:   true,
			wantPanicValue: "oops",
		},
		{
			name:           "panic with int",
			f:              func() { panic(42) },
			wantDidPanic:   true,
			wantPanicValue: 42,
		},
		// Note: In Go 1.21+, panic(nil) actually causes a real panic with a
		// *runtime.PanicNilError, so we don't test that case here.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDidPanic, gotPanicValue := recoverPanic(tt.f)
			if gotDidPanic != tt.wantDidPanic {
				t.Errorf("recoverPanic() didPanic = %v, want %v", gotDidPanic, tt.wantDidPanic)
			}
			if gotPanicValue != tt.wantPanicValue {
				t.Errorf("recoverPanic() panicValue = %v, want %v", gotPanicValue, tt.wantPanicValue)
			}
		})
	}
}

func TestMatchRegexp(t *testing.T) {
	tests := []struct {
		name string
		rx   any
		str  any
		want bool
	}{
		{
			name: "string pattern matches string",
			rx:   "hello",
			str:  "hello world",
			want: true,
		},
		{
			name: "string pattern no match",
			rx:   "foo",
			str:  "hello world",
			want: false,
		},
		{
			name: "regex pattern matches",
			rx:   "h.llo",
			str:  "hello",
			want: true,
		},
		{
			name: "compiled regex matches",
			rx:   regexp.MustCompile(`\d+`),
			str:  "abc123def",
			want: true,
		},
		{
			name: "compiled regex no match",
			rx:   regexp.MustCompile(`\d+`),
			str:  "abcdef",
			want: false,
		},
		{
			name: "byte slice input",
			rx:   "test",
			str:  []byte("this is a test"),
			want: true,
		},
		{
			name: "non-string input",
			rx:   "42",
			str:  42,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchRegexp(tt.rx, tt.str)
			if got != tt.want {
				t.Errorf("matchRegexp(%v, %v) = %v, want %v", tt.rx, tt.str, got, tt.want)
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name    string
		e1      any
		e2      any
		op      string
		want    bool
		wantErr bool
	}{
		// int tests
		{name: "int less than", e1: 1, e2: 2, op: "<", want: true},
		{name: "int not less than", e1: 2, e2: 1, op: "<", want: false},
		{name: "int greater than", e1: 2, e2: 1, op: ">", want: true},
		{name: "int less or equal", e1: 1, e2: 1, op: "<=", want: true},
		{name: "int greater or equal", e1: 2, e2: 2, op: ">=", want: true},

		// float64 tests
		{name: "float less than", e1: 1.5, e2: 2.5, op: "<", want: true},
		{name: "float greater than", e1: 2.5, e2: 1.5, op: ">", want: true},

		// string tests
		{name: "string less than", e1: "a", e2: "b", op: "<", want: true},
		{name: "string greater than", e1: "b", e2: "a", op: ">", want: true},

		// type mismatch
		{name: "type mismatch int/string", e1: 1, e2: "1", op: "<", want: false, wantErr: true},
		{name: "type mismatch int/float", e1: 1, e2: 1.0, op: "<", want: false, wantErr: true},

		// unsupported type
		{name: "unsupported type slice", e1: []int{1}, e2: []int{2}, op: "<", want: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compareValues(tt.e1, tt.e2, tt.op)
			if (err != nil) != tt.wantErr {
				t.Errorf("compareValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("compareValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareToZero(t *testing.T) {
	tests := []struct {
		name    string
		e       any
		op      string
		want    bool
		wantErr bool
	}{
		{name: "positive int > 0", e: 1, op: ">", want: true},
		{name: "zero int > 0", e: 0, op: ">", want: false},
		{name: "negative int > 0", e: -1, op: ">", want: false},
		{name: "negative int < 0", e: -1, op: "<", want: true},
		{name: "positive int < 0", e: 1, op: "<", want: false},
		{name: "positive float > 0", e: 1.5, op: ">", want: true},
		{name: "negative float < 0", e: -1.5, op: "<", want: true},
		{name: "unsupported type", e: "hello", op: ">", want: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compareToZero(tt.e, tt.op)
			if (err != nil) != tt.wantErr {
				t.Errorf("compareToZero() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("compareToZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalOp(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		op   string
		want bool
	}{
		{name: "greater than true", a: 2, b: 1, op: ">", want: true},
		{name: "greater than false", a: 1, b: 2, op: ">", want: false},
		{name: "greater or equal true (greater)", a: 2, b: 1, op: ">=", want: true},
		{name: "greater or equal true (equal)", a: 1, b: 1, op: ">=", want: true},
		{name: "greater or equal false", a: 1, b: 2, op: ">=", want: false},
		{name: "less than true", a: 1, b: 2, op: "<", want: true},
		{name: "less than false", a: 2, b: 1, op: "<", want: false},
		{name: "less or equal true (less)", a: 1, b: 2, op: "<=", want: true},
		{name: "less or equal true (equal)", a: 1, b: 1, op: "<=", want: true},
		{name: "less or equal false", a: 2, b: 1, op: "<=", want: false},
		{name: "unknown op", a: 1, b: 2, op: "==", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalOp(tt.a, tt.b, tt.op)
			if got != tt.want {
				t.Errorf("evalOp(%d, %d, %q) = %v, want %v", tt.a, tt.b, tt.op, got, tt.want)
			}
		})
	}
}
