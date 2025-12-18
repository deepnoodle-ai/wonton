package clipboard

import (
	"context"
	"fmt"
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

// Example_usage demonstrates basic clipboard read and write operations.
// This example is not runnable in tests because clipboard access may not
// be available in CI/headless environments.
func Example_usage() {
	// Check if clipboard is available
	if !Available() {
		fmt.Println("clipboard not available")
		return
	}

	// Write to clipboard
	err := Write("Hello, Clipboard!")
	if err != nil {
		fmt.Println("write error:", err)
		return
	}

	// Read from clipboard
	text, err := Read()
	if err != nil {
		fmt.Println("read error:", err)
		return
	}

	fmt.Println("clipboard contains:", text)
}

// Example_write demonstrates writing text to the clipboard.
func Example_write() {
	err := Write("Sample text")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("Text written to clipboard")
}

// Example_read demonstrates reading text from the clipboard.
func Example_read() {
	text, err := Read()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("Clipboard contains: %s\n", text)
}

// Example_writeWithTimeout demonstrates writing with a custom timeout.
func Example_writeWithTimeout() {
	// Use a shorter timeout for the operation
	err := WriteWithTimeout("urgent data", 2*time.Second)
	if err == ErrTimeout {
		fmt.Println("operation timed out")
		return
	}
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("Data written successfully")
}

// Example_readWithTimeout demonstrates reading with a custom timeout.
func Example_readWithTimeout() {
	// Use a custom timeout
	text, err := ReadWithTimeout(3 * time.Second)
	if err == ErrTimeout {
		fmt.Println("operation timed out")
		return
	}
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("Read: %s\n", text)
}

// Example_writeContext demonstrates using context for cancellation.
func Example_writeContext() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := WriteContext(ctx, "important data")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("Data written with context")
}

// Example_readContext demonstrates using context for cancellation.
func Example_readContext() {
	// Create a context that can be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text, err := ReadContext(ctx)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("Read with context: %s\n", text)
}

// Example_available demonstrates checking clipboard availability.
func Example_available() {
	if Available() {
		fmt.Println("Clipboard is available")
	} else {
		fmt.Println("Clipboard is not available - install xclip, xsel, or wl-clipboard")
	}
}

// Example_clear demonstrates clearing the clipboard.
func Example_clear() {
	// Clear the clipboard contents
	err := Clear()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("Clipboard cleared")
}
