package terminal_test

import (
	"fmt"
	"strings"

	"github.com/deepnoodle-ai/wonton/terminal"
)

// Example demonstrates basic terminal rendering with the frame-based API.
func Example() {
	// Create a test terminal (in real code, use terminal.NewTerminal())
	var output strings.Builder
	term := terminal.NewTestTerminal(40, 10, &output)

	// Render a frame
	frame, _ := term.BeginFrame()
	style := terminal.NewStyle().WithForeground(terminal.ColorBlue).WithBold()
	frame.PrintStyled(0, 0, "Hello, World!", style)
	term.EndFrame(frame)

	// The output contains ANSI escape codes for positioning and styling
	fmt.Println("Frame rendered successfully")
	// Output: Frame rendered successfully
}

// ExampleNewStyle demonstrates creating and using text styles.
func ExampleNewStyle() {
	// Create a style with multiple attributes
	style := terminal.NewStyle().
		WithForeground(terminal.ColorRed).
		WithBackground(terminal.ColorWhite).
		WithBold().
		WithUnderline()

	// Apply the style to text
	styledText := style.Apply("Important")

	// The text is wrapped with ANSI escape codes
	fmt.Printf("Styled text has ANSI codes: %v\n", len(styledText) > len("Important"))
	// Output: Styled text has ANSI codes: true
}

// ExampleStyle_Apply demonstrates applying a style to text.
func ExampleStyle_Apply() {
	style := terminal.NewStyle().WithBold()
	text := style.Apply("Bold text")

	// The text is wrapped with ANSI codes for bold and reset
	fmt.Printf("Text contains escape codes: %v\n", strings.Contains(text, "\033["))
	// Output: Text contains escape codes: true
}

// ExampleStyle_Merge demonstrates merging two styles.
func ExampleStyle_Merge() {
	// Base style with foreground color and bold
	base := terminal.NewStyle().
		WithForeground(terminal.ColorBlue).
		WithBold()

	// Override style with background color
	highlight := terminal.NewStyle().
		WithBackground(terminal.ColorYellow)

	// Merge: result has all attributes from both styles
	combined := base.Merge(highlight)

	// The combined style has both bold and colors
	text := combined.Apply("Highlighted")
	// Check for bold (code 1) and color codes
	hasBold := strings.Contains(text, ";1;")
	hasColors := strings.Contains(text, "34") || strings.Contains(text, "43") // Blue FG or Yellow BG
	fmt.Printf("Has styles: %v\n", hasBold || hasColors)
	// Output: Has styles: true
}

// ExampleNewHyperlink demonstrates creating clickable hyperlinks.
func ExampleNewHyperlink() {
	// Create a hyperlink with default styling (blue, underlined)
	link := terminal.NewHyperlink("https://golang.org", "Go Website")

	// Hyperlinks can be printed using RenderFrame.PrintHyperlink
	// or formatted with Format() for direct printing
	formatted := link.Format()

	// The formatted string contains OSC 8 escape codes
	fmt.Printf("Contains OSC 8: %v\n", strings.Contains(formatted, "\033]8;"))
	// Output: Contains OSC 8: true
}

// ExampleHyperlink_FormatFallback demonstrates the fallback format for hyperlinks.
func ExampleHyperlink_FormatFallback() {
	link := terminal.NewHyperlink("https://example.com", "Example")

	// Fallback format shows URL in parentheses
	fallback := link.FormatFallback()

	fmt.Println(strings.Contains(fallback, "Example"))
	fmt.Println(strings.Contains(fallback, "https://example.com"))
	// Output:
	// true
	// true
}

// ExampleKeyDecoder demonstrates decoding keyboard input.
func ExampleKeyDecoder() {
	// Create a decoder from a test input
	input := strings.NewReader("a\x1b[A") // 'a' key followed by up arrow
	decoder := terminal.NewKeyDecoder(input)

	// Read the first event (character 'a')
	event1, _ := decoder.ReadEvent()
	if keyEvent, ok := event1.(terminal.KeyEvent); ok {
		fmt.Printf("Character: %c\n", keyEvent.Rune)
	}

	// Read the second event (up arrow)
	event2, _ := decoder.ReadEvent()
	if keyEvent, ok := event2.(terminal.KeyEvent); ok {
		fmt.Printf("Special key: %v\n", keyEvent.Key == terminal.KeyArrowUp)
	}

	// Output:
	// Character: a
	// Special key: true
}

// ExampleParseMouseEvent demonstrates parsing mouse events.
func ExampleParseMouseEvent() {
	// SGR format mouse event: left button press at (10, 5)
	// Format: ESC [ < button ; x ; y M
	sequence := []byte("<0;11;6M") // Note: coordinates are 1-based in the protocol

	event, err := terminal.ParseMouseEvent(sequence)
	if err != nil {
		fmt.Println("Parse error")
		return
	}

	fmt.Printf("Button: Left\n")
	fmt.Printf("Position: %d,%d\n", event.X, event.Y)
	fmt.Printf("Type: Press\n")

	// Output:
	// Button: Left
	// Position: 10,5
	// Type: Press
}

// ExampleMouseHandler demonstrates managing clickable regions.
func ExampleMouseHandler() {
	handler := terminal.NewMouseHandler()

	// Track click events
	clicked := false

	// Add a clickable region
	region := &terminal.MouseRegion{
		X:      10,
		Y:      5,
		Width:  15,
		Height: 1,
		Label:  "Submit Button",
		OnClick: func(event *terminal.MouseEvent) {
			clicked = true
		},
	}
	handler.AddRegion(region)

	// Simulate a click at (12, 5) - inside the region
	pressEvent := &terminal.MouseEvent{
		X:      12,
		Y:      5,
		Button: terminal.MouseButtonLeft,
		Type:   terminal.MousePress,
	}
	handler.HandleEvent(pressEvent)

	releaseEvent := &terminal.MouseEvent{
		X:      12,
		Y:      5,
		Button: terminal.MouseButtonNone,
		Type:   terminal.MouseRelease,
	}
	handler.HandleEvent(releaseEvent)

	fmt.Printf("Button was clicked: %v\n", clicked)
	// Output: Button was clicked: true
}

// ExampleRenderFrame demonstrates using RenderFrame for drawing.
func ExampleRenderFrame() {
	var output strings.Builder
	term := terminal.NewTestTerminal(40, 10, &output)

	// Begin a frame
	frame, _ := term.BeginFrame()

	// Get frame dimensions
	width, height := frame.Size()
	fmt.Printf("Frame size: %dx%d\n", width, height)

	// Draw at different positions
	style := terminal.NewStyle()
	frame.PrintStyled(0, 0, "Top left", style)
	frame.PrintStyled(0, 1, "Second line", style)

	// Fill a rectangular area
	frame.FillStyled(10, 3, 5, 2, '*', style)

	// End the frame (flushes to terminal)
	term.EndFrame(frame)

	// Output: Frame size: 40x10
}

// ExampleTerminal_EnableMetrics demonstrates performance metrics.
func ExampleTerminal_EnableMetrics() {
	var output strings.Builder
	term := terminal.NewTestTerminal(40, 10, &output)

	// Enable metrics collection
	term.EnableMetrics()

	// Render some frames
	for i := 0; i < 5; i++ {
		frame, _ := term.BeginFrame()
		style := terminal.NewStyle()
		frame.PrintStyled(0, i, fmt.Sprintf("Line %d", i), style)
		term.EndFrame(frame)
	}

	// Get metrics
	snapshot := term.GetMetrics()
	fmt.Printf("Frames rendered: %d\n", snapshot.TotalFrames)
	fmt.Printf("Cells updated: %d\n", snapshot.CellsUpdated)

	// Output:
	// Frames rendered: 5
	// Cells updated: 30
}

// ExampleRenderFrame_SubFrame demonstrates using SubFrames for clipping.
func ExampleRenderFrame_SubFrame() {
	var output strings.Builder
	term := terminal.NewTestTerminal(40, 10, &output)

	frame, _ := term.BeginFrame()

	// Create a sub-frame that only covers part of the terminal
	// SubFrames use image.Rectangle for bounds
	// This creates a 20x5 region starting at (5, 2)
	bounds := frame.GetBounds()
	subFrame := frame.SubFrame(bounds)

	// Drawing on the sub-frame is automatically clipped
	width, height := subFrame.Size()
	fmt.Printf("SubFrame size: %dx%d\n", width, height)

	term.EndFrame(frame)

	// Output: SubFrame size: 40x10
}
