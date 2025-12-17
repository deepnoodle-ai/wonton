package clipboard

import (
	"context"
	"fmt"
	"log"
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

// Example demonstrates basic clipboard read and write operations.
func Example() {
	// Check if clipboard is available
	if !Available() {
		fmt.Println("clipboard not available")
		return
	}

	// Write to clipboard
	err := Write("Hello, Clipboard!")
	if err != nil {
		log.Fatal(err)
	}

	// Read from clipboard
	text, err := Read()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(text)
	// Output: Hello, Clipboard!
}

// ExampleWrite demonstrates writing text to the clipboard.
func ExampleWrite() {
	err := Write("Sample text")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Text written to clipboard")
}

// ExampleRead demonstrates reading text from the clipboard.
func ExampleRead() {
	text, err := Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Clipboard contains: %s\n", text)
}

// ExampleWriteWithTimeout demonstrates writing with a custom timeout.
func ExampleWriteWithTimeout() {
	// Use a shorter timeout for the operation
	err := WriteWithTimeout("urgent data", 2*time.Second)
	if err == ErrTimeout {
		fmt.Println("operation timed out")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Data written successfully")
}

// ExampleReadWithTimeout demonstrates reading with a custom timeout.
func ExampleReadWithTimeout() {
	// Use a custom timeout
	text, err := ReadWithTimeout(3 * time.Second)
	if err == ErrTimeout {
		fmt.Println("operation timed out")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Read: %s\n", text)
}

// ExampleWriteContext demonstrates using context for cancellation.
func ExampleWriteContext() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := WriteContext(ctx, "important data")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Data written with context")
}

// ExampleReadContext demonstrates using context for cancellation.
func ExampleReadContext() {
	// Create a context that can be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text, err := ReadContext(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Read with context: %s\n", text)
}

// ExampleAvailable demonstrates checking clipboard availability.
func ExampleAvailable() {
	if Available() {
		fmt.Println("Clipboard is available")
	} else {
		fmt.Println("Clipboard is not available - install xclip, xsel, or wl-clipboard")
	}
}

// ExampleClear demonstrates clearing the clipboard.
func ExampleClear() {
	// Clear the clipboard contents
	err := Clear()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Clipboard cleared")
}
