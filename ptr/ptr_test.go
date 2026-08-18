package ptr

import (
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTo(t *testing.T) {
	p := To(42)
	assert.NotNil(t, p)
	assert.Equal(t, *p, 42)

	s := To("hello")
	assert.Equal(t, *s, "hello")

	// Each call boxes an independent copy.
	a, b := To(1), To(1)
	*a = 2
	assert.Equal(t, *b, 1)
}

func TestDeref(t *testing.T) {
	assert.Equal(t, Deref(To(7)), 7)
	assert.Equal(t, Deref[int](nil), 0)
	assert.Equal(t, Deref[string](nil), "")
	assert.Equal(t, Deref[time.Duration](nil), time.Duration(0))
}

func TestOr(t *testing.T) {
	assert.Equal(t, Or(To(7), 99), 7)
	assert.Equal(t, Or(nil, 99), 99)
	assert.Equal(t, Or(To(0), 99), 0, "a pointer to the zero value is still a value")
}

func TestIfNotZero(t *testing.T) {
	assert.Equal(t, *IfNotZero("x"), "x")
	assert.Nil(t, IfNotZero(""))
	assert.Nil(t, IfNotZero(0))
	assert.Nil(t, IfNotZero(time.Duration(0)))
	d := IfNotZero(3 * time.Second)
	assert.Equal(t, *d, 3*time.Second)
}

func TestDerefSlice(t *testing.T) {
	values := []int{1, 2, 3}
	assert.Equal(t, DerefSlice(&values), values)
	assert.Nil(t, DerefSlice[int](nil))

	var empty []string
	assert.Nil(t, DerefSlice(&empty))
}

func TestDerefMap(t *testing.T) {
	m := map[string]int{"a": 1}
	assert.Equal(t, DerefMap(&m), m)
	assert.Nil(t, DerefMap[string, int](nil))
}

func TestMapIfNotEmpty(t *testing.T) {
	m := map[string]int{"a": 1}
	assert.Equal(t, *MapIfNotEmpty(m), m)
	assert.Nil(t, MapIfNotEmpty(map[string]int{}))
	assert.Nil(t, MapIfNotEmpty[string, int](nil))
}

func TestSliceIfNotEmpty(t *testing.T) {
	values := []int{1, 2, 3}
	boxed := SliceIfNotEmpty(values)
	assert.Equal(t, *boxed, values)

	// The copy insulates the boxed slice from later mutation of the source.
	values[0] = 99
	assert.Equal(t, (*boxed)[0], 1)

	assert.Nil(t, SliceIfNotEmpty([]int{}))
	assert.Nil(t, SliceIfNotEmpty[int](nil))
}
