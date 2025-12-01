package require

import (
	"errors"
	"fmt"
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
	panic("FailNow called") // Simulate stopping test execution
}

func (m *testingMock) Name() string {
	return "testingMock"
}

// runRequire runs a require function and captures whether it called FailNow.
func runRequire(f func()) (failed bool) {
	defer func() {
		if r := recover(); r != nil {
			if r == "FailNow called" {
				failed = true
			} else {
				panic(r)
			}
		}
	}()
	f()
	return false
}

func TestEqual(t *testing.T) {
	mockT := &testingMock{}

	// Test equal values - should not call FailNow
	failed := runRequire(func() { Equal(mockT, 1, 1) })
	if failed {
		t.Error("Equal(1, 1) should not call FailNow")
	}

	// Test unequal values - should call FailNow
	mockT = &testingMock{}
	failed = runRequire(func() { Equal(mockT, 1, 2) })
	if !failed {
		t.Error("Equal(1, 2) should call FailNow")
	}
}

func TestNotEqual(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { NotEqual(mockT, 1, 2) })
	if failed {
		t.Error("NotEqual(1, 2) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { NotEqual(mockT, 1, 1) })
	if !failed {
		t.Error("NotEqual(1, 1) should call FailNow")
	}
}

func TestNil(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { Nil(mockT, nil) })
	if failed {
		t.Error("Nil(nil) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { Nil(mockT, 1) })
	if !failed {
		t.Error("Nil(1) should call FailNow")
	}
}

func TestNotNil(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { NotNil(mockT, 1) })
	if failed {
		t.Error("NotNil(1) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { NotNil(mockT, nil) })
	if !failed {
		t.Error("NotNil(nil) should call FailNow")
	}
}

func TestTrue(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { True(mockT, true) })
	if failed {
		t.Error("True(true) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { True(mockT, false) })
	if !failed {
		t.Error("True(false) should call FailNow")
	}
}

func TestFalse(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { False(mockT, false) })
	if failed {
		t.Error("False(false) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { False(mockT, true) })
	if !failed {
		t.Error("False(true) should call FailNow")
	}
}

func TestNoError(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { NoError(mockT, nil) })
	if failed {
		t.Error("NoError(nil) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { NoError(mockT, errors.New("error")) })
	if !failed {
		t.Error("NoError(error) should call FailNow")
	}
}

func TestError(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { Error(mockT, errors.New("error")) })
	if failed {
		t.Error("Error(error) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { Error(mockT, nil) })
	if !failed {
		t.Error("Error(nil) should call FailNow")
	}
}

func TestContains(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { Contains(mockT, "hello world", "world") })
	if failed {
		t.Error("Contains should not call FailNow when element is found")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { Contains(mockT, "hello", "world") })
	if !failed {
		t.Error("Contains should call FailNow when element is not found")
	}
}

func TestLen(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { Len(mockT, []int{1, 2, 3}, 3) })
	if failed {
		t.Error("Len should not call FailNow when length matches")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { Len(mockT, []int{1, 2}, 3) })
	if !failed {
		t.Error("Len should call FailNow when length does not match")
	}
}

func TestGreater(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { Greater(mockT, 2, 1) })
	if failed {
		t.Error("Greater(2, 1) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { Greater(mockT, 1, 2) })
	if !failed {
		t.Error("Greater(1, 2) should call FailNow")
	}
}

func TestLess(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { Less(mockT, 1, 2) })
	if failed {
		t.Error("Less(1, 2) should not call FailNow")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { Less(mockT, 2, 1) })
	if !failed {
		t.Error("Less(2, 1) should call FailNow")
	}
}

func TestInDelta(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { InDelta(mockT, 1.0, 1.01, 0.1) })
	if failed {
		t.Error("InDelta should not call FailNow when within delta")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { InDelta(mockT, 1.0, 2.0, 0.1) })
	if !failed {
		t.Error("InDelta should call FailNow when outside delta")
	}
}

func TestWithinDuration(t *testing.T) {
	mockT := &testingMock{}
	now := time.Now()

	failed := runRequire(func() { WithinDuration(mockT, now, now.Add(time.Second), 2*time.Second) })
	if failed {
		t.Error("WithinDuration should not call FailNow when within duration")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { WithinDuration(mockT, now, now.Add(10*time.Second), time.Second) })
	if !failed {
		t.Error("WithinDuration should call FailNow when outside duration")
	}
}

func TestElementsMatch(t *testing.T) {
	mockT := &testingMock{}

	failed := runRequire(func() { ElementsMatch(mockT, []int{1, 2, 3}, []int{3, 2, 1}) })
	if failed {
		t.Error("ElementsMatch should not call FailNow when elements match")
	}

	mockT = &testingMock{}
	failed = runRequire(func() { ElementsMatch(mockT, []int{1, 2, 3}, []int{1, 2, 4}) })
	if !failed {
		t.Error("ElementsMatch should call FailNow when elements don't match")
	}
}
