package assert

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// objectsAreEqual determines if two objects are considered equal.
func objectsAreEqual(expected, actual any) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}
	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}
	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

// validateEqualArgs checks whether provided arguments can be safely compared.
func validateEqualArgs(expected, actual any) error {
	if expected == nil && actual == nil {
		return nil
	}
	if isFunction(expected) || isFunction(actual) {
		return fmt.Errorf("cannot compare functions")
	}
	return nil
}

// isFunction returns true if the argument is a function.
func isFunction(arg any) bool {
	if arg == nil {
		return false
	}
	return reflect.TypeOf(arg).Kind() == reflect.Func
}

// isNil checks if a specified object is nil.
func isNil(object any) bool {
	if object == nil {
		return true
	}
	value := reflect.ValueOf(object)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	}
	return false
}

// samePointers checks if two pointers point to the same object.
func samePointers(first, second any) (same bool, ok bool) {
	firstPtr, secondPtr := reflect.ValueOf(first), reflect.ValueOf(second)
	if firstPtr.Kind() != reflect.Ptr || secondPtr.Kind() != reflect.Ptr {
		return false, false
	}
	firstType, secondType := reflect.TypeOf(first), reflect.TypeOf(second)
	if firstType != secondType {
		return false, true
	}
	return first == second, true
}

// isEmpty checks if a specified object is considered empty.
func isEmpty(object any) bool {
	if object == nil {
		return true
	}
	objValue := reflect.ValueOf(object)
	if objValue.IsZero() {
		return true
	}
	switch objValue.Kind() {
	case reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
	case reflect.Ptr:
		return isEmpty(objValue.Elem().Interface())
	}
	return false
}

// getLen tries to get the length of an object.
func getLen(x any) (int, bool) {
	v := reflect.ValueOf(x)
	defer func() {
		recover()
	}()
	return v.Len(), true
}

// containsElement checks if a container contains an element.
func containsElement(container, element any) (ok, found bool) {
	containerValue := reflect.ValueOf(container)
	containerType := reflect.TypeOf(container)
	if containerType == nil {
		return false, false
	}
	containerKind := containerType.Kind()
	defer func() {
		if e := recover(); e != nil {
			ok = false
			found = false
		}
	}()
	if containerKind == reflect.String {
		elementValue := reflect.ValueOf(element)
		return true, strings.Contains(containerValue.String(), elementValue.String())
	}
	if containerKind == reflect.Map {
		mapKeys := containerValue.MapKeys()
		for i := 0; i < len(mapKeys); i++ {
			if objectsAreEqual(mapKeys[i].Interface(), element) {
				return true, true
			}
		}
		return true, false
	}
	for i := 0; i < containerValue.Len(); i++ {
		if objectsAreEqual(containerValue.Index(i).Interface(), element) {
			return true, true
		}
	}
	return true, false
}

// isList checks if the provided value is an array or slice.
func isList(list any) bool {
	kind := reflect.TypeOf(list).Kind()
	return kind == reflect.Array || kind == reflect.Slice
}

// diffLists returns elements that are only in A and only in B.
func diffLists(listA, listB any) (extraA, extraB []any) {
	aValue := reflect.ValueOf(listA)
	bValue := reflect.ValueOf(listB)
	aLen := aValue.Len()
	bLen := bValue.Len()
	visited := make([]bool, bLen)
	for i := 0; i < aLen; i++ {
		element := aValue.Index(i).Interface()
		found := false
		for j := 0; j < bLen; j++ {
			if visited[j] {
				continue
			}
			if objectsAreEqual(bValue.Index(j).Interface(), element) {
				visited[j] = true
				found = true
				break
			}
		}
		if !found {
			extraA = append(extraA, element)
		}
	}
	for j := 0; j < bLen; j++ {
		if !visited[j] {
			extraB = append(extraB, bValue.Index(j).Interface())
		}
	}
	return
}

// typeAndKind returns the type and kind of a value, dereferencing pointers.
func typeAndKind(v any) (reflect.Type, reflect.Kind) {
	t := reflect.TypeOf(v)
	k := t.Kind()
	if k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	return t, k
}

// recoverPanic checks if a function panics and returns the panic value.
func recoverPanic(f func()) (didPanic bool, panicValue any) {
	didPanic = true
	defer func() {
		panicValue = recover()
		if panicValue == nil {
			didPanic = false
		}
	}()
	f()
	didPanic = false
	return
}

// matchRegexp returns true if a specified regexp matches a string.
func matchRegexp(rx any, str any) bool {
	var r *regexp.Regexp
	if rr, ok := rx.(*regexp.Regexp); ok {
		r = rr
	} else {
		r = regexp.MustCompile(fmt.Sprint(rx))
	}
	switch v := str.(type) {
	case []byte:
		return r.Match(v)
	case string:
		return r.MatchString(v)
	default:
		return r.MatchString(fmt.Sprint(v))
	}
}

// checkOrderedComparison compares two values using reflection.
// This is used by NonFatalAssertions since Go doesn't allow generic methods.
func checkOrderedComparison(t TestingT, e1, e2 any, op string, msgAndArgs ...any) bool {
	helper(t)
	result, err := compareValues(e1, e2, op)
	if err != nil {
		return fail(t, err.Error(), msgAndArgs...)
	}
	if !result {
		return fail(t, fmt.Sprintf("%v is not %s %v", e1, op, e2), msgAndArgs...)
	}
	return true
}

// checkOrderedToZero compares a value to zero using reflection.
func checkOrderedToZero(t TestingT, e any, op string, msgAndArgs ...any) bool {
	helper(t)
	result, err := compareToZero(e, op)
	if err != nil {
		return fail(t, err.Error(), msgAndArgs...)
	}
	if !result {
		desc := "positive"
		if op == "<" {
			desc = "negative"
		}
		return fail(t, fmt.Sprintf("%v is not %s", e, desc), msgAndArgs...)
	}
	return true
}

// compareValues compares two values and returns whether the comparison holds.
func compareValues(e1, e2 any, op string) (bool, error) {
	switch v1 := e1.(type) {
	case int:
		v2, ok := e2.(int)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case int8:
		v2, ok := e2.(int8)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case int16:
		v2, ok := e2.(int16)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case int32:
		v2, ok := e2.(int32)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case int64:
		v2, ok := e2.(int64)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case uint:
		v2, ok := e2.(uint)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case uint8:
		v2, ok := e2.(uint8)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case uint16:
		v2, ok := e2.(uint16)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case uint32:
		v2, ok := e2.(uint32)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case uint64:
		v2, ok := e2.(uint64)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case float32:
		v2, ok := e2.(float32)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case float64:
		v2, ok := e2.(float64)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	case string:
		v2, ok := e2.(string)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T vs %T", e1, e2)
		}
		return evalOp(v1, v2, op), nil
	default:
		return false, fmt.Errorf("unsupported type for comparison: %T", e1)
	}
}

// compareToZero compares a value to zero.
func compareToZero(e any, op string) (bool, error) {
	switch v := e.(type) {
	case int:
		return evalOp(v, 0, op), nil
	case int8:
		return evalOp(v, 0, op), nil
	case int16:
		return evalOp(v, 0, op), nil
	case int32:
		return evalOp(v, 0, op), nil
	case int64:
		return evalOp(v, 0, op), nil
	case uint:
		return evalOp(v, 0, op), nil
	case uint8:
		return evalOp(v, 0, op), nil
	case uint16:
		return evalOp(v, 0, op), nil
	case uint32:
		return evalOp(v, 0, op), nil
	case uint64:
		return evalOp(v, 0, op), nil
	case float32:
		return evalOp(v, 0, op), nil
	case float64:
		return evalOp(v, 0, op), nil
	default:
		return false, fmt.Errorf("unsupported type for comparison: %T", e)
	}
}

// evalOp evaluates a comparison operation on ordered types.
func evalOp[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 | ~string](a, b T, op string) bool {
	switch op {
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	}
	return false
}
