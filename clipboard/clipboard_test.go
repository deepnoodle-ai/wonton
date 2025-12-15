package clipboard

import (
	"context"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestAvailable(t *testing.T) {
	// This just tests that Available doesn't panic
	_ = Available()
}

func TestReadWrite(t *testing.T) {
	if !Available() {
		t.Skip("clipboard not available")
	}

	// Save original clipboard content
	original, _ := Read()

	// Test write and read
	testText := "wonton clipboard test " + time.Now().String()
	err := Write(testText)
	assert.NoError(t, err)

	result, err := Read()
	assert.NoError(t, err)
	assert.Equal(t, testText, result)

	// Restore original content
	if original != "" {
		_ = Write(original)
	}
}

func TestWriteContext(t *testing.T) {
	if !Available() {
		t.Skip("clipboard not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := WriteContext(ctx, "test")
	assert.NoError(t, err)
}

func TestReadContext(t *testing.T) {
	if !Available() {
		t.Skip("clipboard not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := ReadContext(ctx)
	assert.NoError(t, err)
}

func TestClear(t *testing.T) {
	if !Available() {
		t.Skip("clipboard not available")
	}

	// Write some content first
	_ = Write("content to clear")

	err := Clear()
	assert.NoError(t, err)

	result, err := Read()
	assert.NoError(t, err)
	assert.Equal(t, "", result)
}
