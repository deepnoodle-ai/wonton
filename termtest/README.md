# termtest

Snapshot testing for terminal applications. Captures ANSI sequences, simulates terminal screen state, and compares output against golden files. Ideal for testing CLI tools, TUI applications, and any code that produces terminal output.

## Usage Examples

### Basic Screen Testing

```go
package main

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/termtest"
)

func TestSimpleOutput(t *testing.T) {
	// Create a virtual terminal screen
	screen := termtest.NewScreen(80, 24)

	// Write ANSI output to the screen
	screen.Write([]byte("Hello, \x1b[32mWorld\x1b[0m!"))

	// Assert screen content matches snapshot
	// Run with -update flag to create/update snapshots
	termtest.AssertScreen(t, screen)
}

func TestColoredOutput(t *testing.T) {
	screen := termtest.NewScreen(80, 24)

	// Test ANSI color codes
	screen.Write([]byte("\x1b[31mRed\x1b[0m "))
	screen.Write([]byte("\x1b[32mGreen\x1b[0m "))
	screen.Write([]byte("\x1b[34mBlue\x1b[0m"))

	// Snapshot comparison
	termtest.AssertScreen(t, screen)
}
```

### Content Assertions

```go
func TestMenuDisplay(t *testing.T) {
	screen := termtest.NewScreen(80, 24)

	// Simulate menu output
	screen.Write([]byte("Main Menu\r\n"))
	screen.Write([]byte("1. Start\r\n"))
	screen.Write([]byte("2. Options\r\n"))
	screen.Write([]byte("3. Exit\r\n"))

	// Assert content is present
	termtest.AssertContains(t, screen, "Main Menu")
	termtest.AssertContains(t, screen, "Start")
	termtest.AssertNotContains(t, screen, "Hidden")

	// Assert specific row content
	termtest.AssertRow(t, screen, 0, "Main Menu")
	termtest.AssertRow(t, screen, 1, "1. Start")

	// Assert row contains text
	termtest.AssertRowContains(t, screen, 2, "Options")

	// Assert row starts with prefix
	termtest.AssertRowPrefix(t, screen, 3, "3. ")
}
```

### Cursor Position Testing

```go
func TestCursorMovement(t *testing.T) {
	screen := termtest.NewScreen(80, 24)

	// Initial cursor position
	termtest.AssertCursor(t, screen, 0, 0)

	// Write text and check cursor moved
	screen.Write([]byte("Hello"))
	termtest.AssertCursor(t, screen, 5, 0)

	// Test cursor movement sequences
	screen.Write([]byte("\x1b[5;10H")) // Move to row 5, col 10
	termtest.AssertCursor(t, screen, 9, 4) // 0-indexed

	// Write at new position
	screen.Write([]byte("Here"))
	termtest.AssertRow(t, screen, 4, "         Here")
}
```

### Style and Cell Testing

```go
func TestStyledText(t *testing.T) {
	screen := termtest.NewScreen(80, 24)

	// Write bold text
	screen.Write([]byte("\x1b[1mBold\x1b[0m"))

	// Check specific cell content
	termtest.AssertCell(t, screen, 0, 0, 'B')
	termtest.AssertCell(t, screen, 1, 0, 'o')

	// Check cell style
	termtest.AssertCellBold(t, screen, 0, 0, true)
	termtest.AssertCellBold(t, screen, 5, 0, false)
}
```

### Capture Output from Functions

```go
func TestFunctionOutput(t *testing.T) {
	// Capture output
	capture := termtest.NewCapture(nil)

	// Write to capture
	myFunction(capture)

	// Convert to screen for assertions
	screen := capture.Screen(80, 24)
	termtest.AssertContains(t, screen, "Expected output")
}

func myFunction(w io.Writer) {
	fmt.Fprintf(w, "Expected output\n")
	fmt.Fprintf(w, "\x1b[32mGreen text\x1b[0m\n")
}
```

### Recording and Replay

```go
func TestWithRecorder(t *testing.T) {
	// Record all writes with timing
	recorder := termtest.NewRecorder(80, 24)

	// Write output
	recorder.Write([]byte("Frame 1\r\n"))
	recorder.Write([]byte("Frame 2\r\n"))

	// Get current screen state
	screen := recorder.Screen()
	termtest.AssertContains(t, screen, "Frame 1")
	termtest.AssertContains(t, screen, "Frame 2")

	// Replay to different size screen
	replayScreen := recorder.Replay(100, 30)
	termtest.AssertContains(t, replayScreen, "Frame 1")
}
```

### Custom Snapshot Names

```go
func TestMultipleSnapshots(t *testing.T) {
	screen1 := termtest.NewScreen(80, 24)
	screen1.Write([]byte("First state"))
	termtest.AssertScreenNamed(t, "initial_state", screen1)

	screen2 := termtest.NewScreen(80, 24)
	screen2.Write([]byte("Second state"))
	termtest.AssertScreenNamed(t, "final_state", screen2)
}
```

### Testing Complex ANSI Sequences

```go
func TestANSISequences(t *testing.T) {
	screen := termtest.NewScreen(80, 24)

	// Test cursor positioning
	screen.Write([]byte("\x1b[1;1HTop-left"))
	screen.Write([]byte("\x1b[24;1HBottom-left"))
	screen.Write([]byte("\x1b[1;71HTop-right"))

	termtest.AssertRow(t, screen, 0, "Top-left")
	termtest.AssertRow(t, screen, 23, "Bottom-left")

	// Test erase sequences
	screen.Write([]byte("\x1b[2J"))      // Clear screen
	screen.Write([]byte("\x1b[1;1HNew"))  // Write new content
	termtest.AssertRow(t, screen, 0, "New")
	termtest.AssertRow(t, screen, 23, "") // Bottom should be empty

	// Test colors
	screen.Write([]byte("\x1b[31mRed "))      // Red foreground
	screen.Write([]byte("\x1b[42mGreen-bg "))  // Green background
	screen.Write([]byte("\x1b[1;34mBold-Blue")) // Bold blue
	termtest.AssertRow(t, screen, 0, "NewRed Green-bg Bold-Blue")
}
```

### Screen Comparison

```go
func TestScreenEquality(t *testing.T) {
	screen1 := termtest.NewScreen(80, 24)
	screen1.Write([]byte("Same content"))

	screen2 := termtest.NewScreen(80, 24)
	screen2.Write([]byte("Same content"))

	// Compare text content
	termtest.AssertEqual(t, screen1, screen2)

	// Check text equality without assertion
	if !termtest.Equal(screen1, screen2) {
		t.Error("Screens should be equal")
	}

	// Compare including styles
	if !termtest.EqualStyled(screen1, screen2) {
		t.Error("Styled screens should be equal")
	}
}
```

### Testing Screen Clearing

```go
func TestScreenClear(t *testing.T) {
	screen := termtest.NewScreen(80, 24)

	// Write content
	screen.Write([]byte("Line 1\r\nLine 2\r\nLine 3"))

	// Clear screen
	screen.Clear()

	// Assert screen is empty
	termtest.AssertEmpty(t, screen)

	// Test partial clearing
	screen.Write([]byte("Before\x1b[Kafter"))
	// \x1b[K clears from cursor to end of line
}
```

## API Reference

### Screen Types

| Type | Description |
|------|-------------|
| `Screen` | Virtual terminal screen buffer with ANSI support |
| `Cell` | Single character cell with style information |
| `Style` | Text styling (colors, bold, italic, etc.) |
| `Color` | Terminal color (basic, 256-color, or RGB) |
| `ColorType` | Color type enumeration |
| `Capture` | Output capture with forwarding |
| `Recorder` | Output recorder with timing |
| `Buffer` | Simple buffer with screen conversion |

### Screen Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NewScreen` | Creates virtual terminal screen | `width, height int` | `*Screen` |
| `Screen.Write` | Writes ANSI output to screen | `p []byte` | `n int, err error` |
| `Screen.WriteString` | Writes string at cursor | `str string` | none |
| `Screen.WriteRune` | Writes single rune | `r rune` | none |
| `Screen.Text` | Gets screen as plain text | none | `string` |
| `Screen.Row` | Gets single row text | `y int` | `string` |
| `Screen.Contains` | Checks if text is present | `text string` | `bool` |
| `Screen.Size` | Returns dimensions | none | `width, height int` |
| `Screen.Cursor` | Returns cursor position | none | `x, y int` |
| `Screen.SetCursor` | Sets cursor position | `x, y int` | none |
| `Screen.Cell` | Returns cell at position | `x, y int` | `Cell` |
| `Screen.SetCell` | Sets cell content and style | `x, y int, char rune, style Style` | none |
| `Screen.Clear` | Clears entire screen | none | none |
| `Screen.ClearLine` | Clears current line | none | none |
| `Screen.ClearToEndOfLine` | Clears cursor to line end | none | none |
| `Screen.ClearToStartOfLine` | Clears line start to cursor | none | none |
| `Screen.ClearToEndOfScreen` | Clears cursor to screen end | none | none |
| `Screen.ClearToStartOfScreen` | Clears screen start to cursor | none | none |

### Assertion Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `AssertScreen` | Compares screen to snapshot | `t *testing.T, screen *Screen` | none |
| `AssertScreenNamed` | Compares to named snapshot | `t *testing.T, name string, screen *Screen` | none |
| `AssertText` | Compares text to snapshot | `t *testing.T, actual string` | none |
| `AssertTextNamed` | Compares to named text snapshot | `t *testing.T, name string, actual string` | none |
| `AssertContains` | Asserts text is present | `t *testing.T, screen *Screen, text string` | none |
| `AssertNotContains` | Asserts text is absent | `t *testing.T, screen *Screen, text string` | none |
| `AssertRow` | Asserts row matches exactly | `t *testing.T, screen *Screen, row int, expected string` | none |
| `AssertRowContains` | Asserts row contains text | `t *testing.T, screen *Screen, row int, text string` | none |
| `AssertRowPrefix` | Asserts row starts with text | `t *testing.T, screen *Screen, row int, prefix string` | none |
| `AssertCursor` | Asserts cursor position | `t *testing.T, screen *Screen, x, y int` | none |
| `AssertTextEqual` | Asserts exact text match | `t *testing.T, screen *Screen, expected string` | none |
| `AssertCell` | Asserts cell character | `t *testing.T, screen *Screen, x, y int, expected rune` | none |
| `AssertCellStyle` | Asserts cell style | `t *testing.T, screen *Screen, x, y int, style Style` | none |
| `AssertCellBold` | Asserts cell bold state | `t *testing.T, screen *Screen, x, y int, bold bool` | none |
| `AssertEmpty` | Asserts screen is empty | `t *testing.T, screen *Screen` | none |
| `AssertEqual` | Asserts screens have same text | `t *testing.T, expected, actual *Screen` | none |
| `RequireContains` | Like AssertContains but fails immediately | `t *testing.T, screen *Screen, text string` | none |
| `RequireRow` | Like AssertRow but fails immediately | `t *testing.T, screen *Screen, row int, expected string` | none |

### Capture Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NewCapture` | Creates output capture | `writer io.Writer` | `*Capture` |
| `Capture.Write` | Writes and captures | `p []byte` | `n int, err error` |
| `Capture.Bytes` | Returns captured bytes | none | `[]byte` |
| `Capture.String` | Returns captured string | none | `string` |
| `Capture.Reset` | Clears captured output | none | none |
| `Capture.Screen` | Converts to screen | `width, height int` | `*Screen` |

### Recorder Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NewRecorder` | Creates recorder | `width, height int` | `*Recorder` |
| `Recorder.Write` | Records write event | `p []byte` | `n int, err error` |
| `Recorder.Screen` | Gets current screen | none | `*Screen` |
| `Recorder.Events` | Gets all recorded events | none | `[]RecordedEvent` |
| `Recorder.Reset` | Clears recorder | none | none |
| `Recorder.Replay` | Replays to new screen | `width, height int` | `*Screen` |

### Comparison Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Equal` | Checks text equality | `a, b *Screen` | `bool` |
| `EqualStyled` | Checks full equality with styles | `a, b *Screen` | `bool` |
| `Diff` | Generates unified diff | `expected, actual string` | `string` |

## Testing Workflow

To update snapshots, run tests with the `-update` flag:

```bash
go test -update ./...
```

Or set environment variable:

```bash
TERMTEST_UPDATE=1 go test ./...
```

Snapshots are stored in `testdata/snapshots/` directory as `.snap` files.

## Related Packages

- [terminal](../terminal) - Low-level terminal control and ANSI sequences
- [termsession](../termsession) - Terminal session recording and playback
- [tui](../tui) - Declarative terminal UI framework
- [assert](../assert) - General-purpose test assertions with diffs
