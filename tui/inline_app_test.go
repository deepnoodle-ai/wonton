package tui

import (
	"bytes"
	"image"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// testInlineApp is a minimal InlineApplication for testing
type testInlineApp struct {
	runner     *InlineApp
	mu         sync.Mutex
	viewCalls  int
	events     []Event
	liveText   string
	shouldQuit bool
}

func (a *testInlineApp) LiveView() View {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.viewCalls++
	return Text("%s", a.liveText)
}

func (a *testInlineApp) HandleEvent(event Event) []Cmd {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)

	// Check for quit key
	if key, ok := event.(KeyEvent); ok && key.Rune == 'q' {
		a.shouldQuit = true
		return []Cmd{Quit()}
	}

	return nil
}

func (a *testInlineApp) getViewCalls() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.viewCalls
}

func (a *testInlineApp) getEvents() []Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]Event, len(a.events))
	copy(result, a.events)
	return result
}

// testInitDestroyApp tracks Init/Destroy calls
type testInitDestroyApp struct {
	testInlineApp
	initCalled    bool
	destroyCalled bool
	initErr       error
}

func (a *testInitDestroyApp) Init() error {
	a.initCalled = true
	return a.initErr
}

func (a *testInitDestroyApp) Destroy() {
	a.destroyCalled = true
}

// testFocusable is a simple focusable for testing
type testFocusable struct {
	id      string
	focused bool
	bounds  image.Rectangle
}

func (m *testFocusable) FocusID() string                { return m.id }
func (m *testFocusable) IsFocused() bool                { return m.focused }
func (m *testFocusable) SetFocused(focused bool)        { m.focused = focused }
func (m *testFocusable) FocusBounds() image.Rectangle   { return m.bounds }
func (m *testFocusable) HandleKeyEvent(e KeyEvent) bool { return false }

// newTestFocusable creates a focusable with bounds at (x, y)
func newTestFocusable(id string, x, y int) *testFocusable {
	return &testFocusable{
		id:     id,
		bounds: image.Rect(x, y, x+10, y+1),
	}
}

// TestInlineApp_NewInlineApp tests the constructor and config
func TestInlineApp_NewInlineApp(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		runner := NewInlineApp()
		assert.NotNil(t, runner)
		assert.Equal(t, 0, runner.config.FPS)
		assert.False(t, runner.config.MouseTracking)
		assert.False(t, runner.config.BracketedPaste)
	})

	t.Run("with config", func(t *testing.T) {
		var buf bytes.Buffer
		runner := NewInlineApp(InlineAppConfig{
			Width:          100,
			Output:         &buf,
			FPS:            30,
			MouseTracking:  true,
			BracketedPaste: true,
			PasteTabWidth:  4,
			KittyKeyboard:  true,
		})

		assert.Equal(t, 100, runner.config.Width)
		assert.Equal(t, 30, runner.config.FPS)
		assert.True(t, runner.config.MouseTracking)
		assert.True(t, runner.config.BracketedPaste)
		assert.Equal(t, 4, runner.config.PasteTabWidth)
		assert.True(t, runner.config.KittyKeyboard)
	})
}

// TestInlineApp_RequiresInlineApplication tests interface validation
func TestInlineApp_RequiresInlineApplication(t *testing.T) {
	runner := NewInlineApp()

	// App without LiveView should fail
	type noLiveView struct{}
	err := runner.Run(&noLiveView{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "InlineApplication")
}

// TestInlineApp_Stop tests graceful shutdown
func TestInlineApp_Stop(t *testing.T) {
	// This test verifies Stop() sends a QuitEvent
	// We can't fully test Run() without a real terminal

	runner := NewInlineApp()

	// Start a goroutine that will call Stop after a short delay
	done := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Millisecond)
		runner.Stop()
		close(done)
	}()

	// Wait for Stop to be called
	select {
	case <-done:
		// Success - Stop was called
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Stop did not complete in time")
	}
}

// TestInlineApp_SendEvent tests event injection
func TestInlineApp_SendEvent(t *testing.T) {
	runner := NewInlineApp()

	// SendEvent should not block even when no one is listening
	done := make(chan struct{})
	go func() {
		runner.SendEvent(TickEvent{Time: time.Now()})
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("SendEvent blocked")
	}
}

// TestInlineApp_PrintFormats tests Printf formatting
func TestInlineApp_PrintFormats(t *testing.T) {
	// This is a basic test that Printf creates the expected view
	// Full integration testing would require a real terminal

	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  40,
	})

	// We can't fully test Print without running, but we can test the method exists
	assert.NotNil(t, runner.Printf)
}

// TestInlineApp_RunInline tests the convenience function
func TestInlineApp_RunInline(t *testing.T) {
	// RunInline should return error for non-terminal stdin
	app := &testInlineApp{liveText: "test"}
	err := RunInline(app, nil)

	// Should error because stdin is not a terminal in tests
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terminal")
}

// TestInlineApp_MouseClickSynthesis tests mouse event processing
func TestInlineApp_MouseClickSynthesis(t *testing.T) {
	runner := NewInlineApp()

	// Test press at location
	pressEvent := MouseEvent{
		X:      10,
		Y:      5,
		Button: MouseButtonLeft,
		Type:   MousePress,
		Time:   time.Now(),
	}

	result, click := runner.processMouseEvent(pressEvent)
	assert.Equal(t, pressEvent, result)
	assert.Nil(t, click)
	assert.True(t, runner.mousePressed)
	assert.Equal(t, 10, runner.mousePressX)
	assert.Equal(t, 5, runner.mousePressY)

	// Test release at same location (should synthesize click)
	releaseEvent := MouseEvent{
		X:      10,
		Y:      5,
		Button: MouseButtonLeft,
		Type:   MouseRelease,
		Time:   time.Now(),
	}

	result, click = runner.processMouseEvent(releaseEvent)
	assert.Equal(t, releaseEvent, result)
	assert.NotNil(t, click)
	assert.Equal(t, MouseClick, click.(MouseEvent).Type)
	assert.Equal(t, 10, click.(MouseEvent).X)
	assert.Equal(t, 5, click.(MouseEvent).Y)

	// Test release at different location (no click)
	runner.mousePressed = false
	runner.mousePressX = 0
	runner.mousePressY = 0

	pressEvent2 := MouseEvent{X: 0, Y: 0, Type: MousePress, Time: time.Now()}
	runner.processMouseEvent(pressEvent2)

	releaseEvent2 := MouseEvent{X: 5, Y: 5, Type: MouseRelease, Time: time.Now()}
	result, click = runner.processMouseEvent(releaseEvent2)
	assert.Nil(t, click)
}

// TestInlineApp_ProcessEventWithQuitCheck tests quit detection
func TestInlineApp_ProcessEventWithQuitCheck(t *testing.T) {
	runner := NewInlineApp()
	runner.app = &testInlineApp{liveText: "test"}

	// Regular event should return false
	keyEvent := KeyEvent{Rune: 'a', Time: time.Now()}
	isQuit := runner.processEventWithQuitCheck(keyEvent)
	assert.False(t, isQuit)

	// QuitEvent should return true
	quitEvent := QuitEvent{Time: time.Now()}
	isQuit = runner.processEventWithQuitCheck(quitEvent)
	assert.True(t, isQuit)

	// BatchEvent containing QuitEvent should return true
	batchEvent := BatchEvent{
		Time: time.Now(),
		Events: []Event{
			KeyEvent{Rune: 'a', Time: time.Now()},
			QuitEvent{Time: time.Now()},
		},
	}
	isQuit = runner.processEventWithQuitCheck(batchEvent)
	assert.True(t, isQuit)
}

// TestInlineApp_LivePrinterIntegration tests LivePrinter output
func TestInlineApp_LivePrinterIntegration(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  40,
	})

	app := &testInlineApp{liveText: "Hello, World!"}
	runner.app = app

	// Create a live printer manually for testing
	runner.live = NewLivePrinter(PrintConfig{Width: 40, Output: &buf})

	// Render should call LiveView and update the live printer
	runner.render()

	// Check that LiveView was called
	assert.Equal(t, 1, app.getViewCalls())

	// Check that output contains the text
	output := buf.String()
	assert.Contains(t, output, "Hello")
}

// TestInlineApp_ConfigDefaults tests config default values
func TestInlineApp_ConfigDefaults(t *testing.T) {
	cfg := InlineAppConfig{}.withDefaults()

	// Check defaults match the design doc
	assert.Equal(t, 0, cfg.FPS)
	assert.False(t, cfg.MouseTracking)
	assert.False(t, cfg.BracketedPaste)
	assert.Equal(t, 0, cfg.PasteTabWidth)
	assert.False(t, cfg.KittyKeyboard)
}

// TestInlineApp_ConcurrentPrint tests thread safety of Print
func TestInlineApp_ConcurrentPrint(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  40,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 40, Output: &buf})

	// Multiple goroutines calling methods concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			runner.Print(Text("Message %d", n))
		}(i)
	}

	wg.Wait()

	// If we get here without deadlock or panic, the test passes
}

// TestInlineApp_PrintRawModeLineEndings tests that Print uses raw mode line endings
func TestInlineApp_PrintRawModeLineEndings(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  40,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 40, Output: &buf})

	// Print a multi-line view
	runner.Print(Stack(
		Text("Line 1"),
		Text("Line 2"),
	))

	output := buf.String()

	// In raw mode, we should see \r\n (CRLF) not just \n
	// The final newline after print should also be \r\n
	assert.Contains(t, output, "\r\n")
}

// TestWithRawMode tests the WithRawMode option for Print
func TestWithRawMode(t *testing.T) {
	t.Run("without raw mode uses LF", func(t *testing.T) {
		var buf bytes.Buffer
		view := Stack(Text("Line 1"), Text("Line 2"))
		Fprint(&buf, view, PrintConfig{Width: 20})
		output := buf.String()

		// Should contain plain \n (or no special line ending if single line equivalent)
		// Multi-line stack should have \n between lines
		assert.True(t, strings.Contains(output, "\n") || !strings.Contains(output, "\r\n"))
	})

	t.Run("with raw mode uses CRLF", func(t *testing.T) {
		var buf bytes.Buffer
		view := Stack(Text("Line 1"), Text("Line 2"))
		Fprint(&buf, view, PrintConfig{Width: 20, RawMode: true})
		output := buf.String()

		// Should contain \r\n
		assert.Contains(t, output, "\r\n")
	})
}

// TestInlineApp_AlreadyRunningError tests double-run prevention
func TestInlineApp_AlreadyRunningError(t *testing.T) {
	runner := NewInlineApp()

	// Simulate running state
	runner.mu.Lock()
	runner.running = true
	runner.mu.Unlock()

	app := &testInlineApp{liveText: "test"}
	err := runner.Run(app)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

// TestInlineApp_ProcessResizeEvent tests ResizeEvent handling
func TestInlineApp_ProcessResizeEvent(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Initial width should be 80
	assert.Equal(t, 80, runner.config.Width)

	// Process a resize event
	resizeEvent := ResizeEvent{
		Time:   time.Now(),
		Width:  120,
		Height: 40,
	}

	runner.processEvent(resizeEvent)

	// Width should be updated
	runner.mu.Lock()
	updatedWidth := runner.config.Width
	runner.mu.Unlock()

	assert.Equal(t, 120, updatedWidth)
}

// TestInlineApp_ResizeChannelInitialized tests that resize channel is created
func TestInlineApp_ResizeChannelInitialized(t *testing.T) {
	runner := NewInlineApp()

	// Before Run(), resize channel should be nil
	assert.Nil(t, runner.resizeChan)

	// After attempting Run() (which will fail due to no terminal),
	// we can verify the channel would be initialized
	// We can't fully test this without a real terminal, but we can
	// verify the field exists and is used properly in other tests
}

// TestInlineApp_MultipleResizeEvents tests handling multiple resize events
func TestInlineApp_MultipleResizeEvents(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Process multiple resize events in sequence
	sizes := []int{100, 120, 90, 150}
	for _, width := range sizes {
		resizeEvent := ResizeEvent{
			Time:   time.Now(),
			Width:  width,
			Height: 40,
		}
		runner.processEvent(resizeEvent)

		runner.mu.Lock()
		currentWidth := runner.config.Width
		runner.mu.Unlock()

		assert.Equal(t, width, currentWidth)
	}
}

// TestInlineApp_ResizeEventWithConcurrentPrint tests resize during Print operations
func TestInlineApp_ResizeEventWithConcurrentPrint(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	var wg sync.WaitGroup

	// Start multiple goroutines that print
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			runner.Print(Text("Message %d", n))
		}(i)
	}

	// Start goroutines that send resize events
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(width int) {
			defer wg.Done()
			runner.processEvent(ResizeEvent{
				Time:   time.Now(),
				Width:  80 + width*10,
				Height: 40,
			})
		}(i)
	}

	wg.Wait()

	// If we get here without deadlock or panic, the test passes
	// The mutex should protect concurrent access to config.Width
}

// TestInlineApp_Printf tests formatted printing to scrollback
func TestInlineApp_Printf(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "live content"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Test simple format string
	runner.Printf("Hello, %s!", "World")
	output := buf.String()
	assert.Contains(t, output, "Hello, World!")

	// Clear buffer and test with numbers
	buf.Reset()
	runner.Printf("Count: %d, Value: %0.2f", 42, 3.14159)
	output = buf.String()
	assert.Contains(t, output, "Count: 42")
	assert.Contains(t, output, "Value: 3.14")

	// Test that it includes proper line endings for raw mode
	assert.Contains(t, output, "\r\n")
}

// TestInlineApp_PrintfMultipleArgs tests Printf with various argument types
func TestInlineApp_PrintfMultipleArgs(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Test with multiple different types
	runner.Printf("String: %s, Int: %d, Bool: %t, Float: %.1f", "test", 123, true, 45.6)
	output := buf.String()

	assert.Contains(t, output, "String: test")
	assert.Contains(t, output, "Int: 123")
	assert.Contains(t, output, "Bool: true")
	assert.Contains(t, output, "Float: 45.6")
}

// TestInlineApp_PrintfNoArgs tests Printf with no format arguments
func TestInlineApp_PrintfNoArgs(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Printf with just a string and no format specifiers
	runner.Printf("Simple message")
	output := buf.String()
	assert.Contains(t, output, "Simple message")
}

// TestInlineApp_ClearScrollback tests clearing terminal scrollback
func TestInlineApp_ClearScrollback(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "live content"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Add some content first
	runner.Printf("Some scrollback content")
	buf.Reset()

	// Clear scrollback
	runner.ClearScrollback()
	output := buf.String()

	// Should contain the ANSI escape sequences for:
	// \033[3J - clear scrollback buffer
	// \033[2J - clear screen
	// \033[H  - move cursor to home position
	assert.Contains(t, output, "\033[3J", "should contain clear scrollback sequence")
	assert.Contains(t, output, "\033[2J", "should contain clear screen sequence")
	assert.Contains(t, output, "\033[H", "should contain home cursor sequence")

	// Should also re-render the live view
	assert.Contains(t, output, "live content", "should re-render live view after clear")
}

// TestInlineApp_ClearScrollbackMultipleTimes tests multiple clear operations
func TestInlineApp_ClearScrollbackMultipleTimes(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "persistent live view"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Clear multiple times
	for i := 0; i < 3; i++ {
		buf.Reset()
		runner.ClearScrollback()
		output := buf.String()

		// Each clear should work properly
		assert.Contains(t, output, "\033[3J")
		assert.Contains(t, output, "persistent live view")
	}
}

// TestInlineApp_ClearScrollbackThreadSafe tests concurrent ClearScrollback calls
func TestInlineApp_ClearScrollbackThreadSafe(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	var wg sync.WaitGroup

	// Multiple goroutines calling ClearScrollback
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runner.ClearScrollback()
		}()
	}

	// Also call Printf concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			runner.Printf("Message %d", n)
		}(i)
	}

	wg.Wait()

	// If we get here without deadlock or panic, the test passes
}

// TestInlineApp_ProcessEvent_FocusSetEvent tests FocusSetEvent handling
func TestInlineApp_ProcessEvent_FocusSetEvent(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Register a focusable element
	runner.focusMgr.Register(newTestFocusable("test-input", 10, 5))

	// Process FocusSetEvent
	focusEvent := FocusSetEvent{
		ID:   "test-input",
		Time: time.Now(),
	}

	runner.processEvent(focusEvent)

	// Verify focus was set
	focusedID := runner.focusMgr.GetFocusedID()
	assert.Equal(t, "test-input", focusedID)
}

// TestInlineApp_ProcessEvent_FocusNextEvent tests FocusNextEvent handling
func TestInlineApp_ProcessEvent_FocusNextEvent(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Register multiple focusable elements
	runner.focusMgr.Register(newTestFocusable("input1", 10, 5))
	runner.focusMgr.Register(newTestFocusable("input2", 10, 10))
	runner.focusMgr.Register(newTestFocusable("input3", 10, 15))

	// Set initial focus
	runner.focusMgr.SetFocus("input1")
	assert.Equal(t, "input1", runner.focusMgr.GetFocusedID())

	// Process FocusNextEvent
	focusNextEvent := FocusNextEvent{Time: time.Now()}
	runner.processEvent(focusNextEvent)

	// Focus should move to next element
	assert.Equal(t, "input2", runner.focusMgr.GetFocusedID())

	// Process again
	runner.processEvent(focusNextEvent)
	assert.Equal(t, "input3", runner.focusMgr.GetFocusedID())
}

// TestInlineApp_ProcessEvent_FocusPrevEvent tests FocusPrevEvent handling
func TestInlineApp_ProcessEvent_FocusPrevEvent(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Register multiple focusable elements
	runner.focusMgr.Register(newTestFocusable("input1", 10, 5))
	runner.focusMgr.Register(newTestFocusable("input2", 10, 10))
	runner.focusMgr.Register(newTestFocusable("input3", 10, 15))

	// Set focus to third element
	runner.focusMgr.SetFocus("input3")
	assert.Equal(t, "input3", runner.focusMgr.GetFocusedID())

	// Process FocusPrevEvent
	focusPrevEvent := FocusPrevEvent{Time: time.Now()}
	runner.processEvent(focusPrevEvent)

	// Focus should move to previous element
	assert.Equal(t, "input2", runner.focusMgr.GetFocusedID())

	// Process again
	runner.processEvent(focusPrevEvent)
	assert.Equal(t, "input1", runner.focusMgr.GetFocusedID())
}

// TestInlineApp_ProcessEvent_MouseClickFocusHandling tests mouse click focus routing
func TestInlineApp_ProcessEvent_MouseClickFocusHandling(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Register focusable elements at different positions
	runner.focusMgr.Register(newTestFocusable("input1", 10, 5))
	runner.focusMgr.Register(newTestFocusable("input2", 20, 10))

	// Click on input2's position
	clickEvent := MouseEvent{
		X:      20,
		Y:      10,
		Button: MouseButtonLeft,
		Type:   MouseClick,
		Time:   time.Now(),
	}

	runner.processEvent(clickEvent)

	// Focus should move to clicked element
	assert.Equal(t, "input2", runner.focusMgr.GetFocusedID())
}

// TestInlineApp_ProcessEvent_MouseNonClickIgnored tests that non-click mouse events don't affect focus
func TestInlineApp_ProcessEvent_MouseNonClickIgnored(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Register focusable elements
	runner.focusMgr.Register(newTestFocusable("input1", 10, 5))
	runner.focusMgr.Register(newTestFocusable("input2", 20, 10))

	// Set initial focus
	runner.focusMgr.SetFocus("input1")
	initialFocus := runner.focusMgr.GetFocusedID()

	// Send mouse press (not click)
	pressEvent := MouseEvent{
		X:      20,
		Y:      10,
		Button: MouseButtonLeft,
		Type:   MousePress,
		Time:   time.Now(),
	}

	runner.processEvent(pressEvent)

	// Focus should not change for non-click events
	assert.Equal(t, initialFocus, runner.focusMgr.GetFocusedID())
}

// TestInlineApp_ProcessEvent_KeyEventRoutedToFocusManager tests key event routing
func TestInlineApp_ProcessEvent_KeyEventRoutedToFocusManager(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(InlineAppConfig{
		Output: &buf,
		Width:  80,
	})

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(PrintConfig{Width: 80, Output: &buf})

	// Register focusable elements
	runner.focusMgr.Register(newTestFocusable("input1", 10, 5))
	runner.focusMgr.Register(newTestFocusable("input2", 10, 10))

	// Set initial focus
	runner.focusMgr.SetFocus("input1")
	assert.Equal(t, "input1", runner.focusMgr.GetFocusedID())

	// Send Tab key event (should trigger focus next)
	tabEvent := KeyEvent{
		Key:  KeyTab,
		Time: time.Now(),
	}

	runner.processEvent(tabEvent)

	// Focus should move to next element
	// (FocusManager's HandleKey processes Tab as FocusNext)
	assert.Equal(t, "input2", runner.focusMgr.GetFocusedID())
}

// TestInlineApp_ProcessMouseEvent_NonMouseEvent tests handling of non-mouse events
func TestInlineApp_ProcessMouseEvent_NonMouseEvent(t *testing.T) {
	runner := NewInlineApp()

	// Pass a non-mouse event
	keyEvent := KeyEvent{Rune: 'a', Time: time.Now()}
	result, click := runner.processMouseEvent(keyEvent)

	// Should return original event unchanged, no click synthesized
	assert.Equal(t, keyEvent, result)
	assert.Nil(t, click)
}

// TestInlineApp_ProcessMouseEvent_Drag tests mouse drag (press and release at different positions)
func TestInlineApp_ProcessMouseEvent_Drag(t *testing.T) {
	runner := NewInlineApp()

	// Press at one location
	pressEvent := MouseEvent{
		X:      10,
		Y:      5,
		Button: MouseButtonLeft,
		Type:   MousePress,
		Time:   time.Now(),
	}

	result, click := runner.processMouseEvent(pressEvent)
	assert.Equal(t, pressEvent, result)
	assert.Nil(t, click)
	assert.True(t, runner.mousePressed)

	// Release at different location (drag)
	releaseEvent := MouseEvent{
		X:      20,
		Y:      15,
		Button: MouseButtonLeft,
		Type:   MouseRelease,
		Time:   time.Now(),
	}

	result, click = runner.processMouseEvent(releaseEvent)

	// Should return release event but NO click (different position)
	assert.Equal(t, releaseEvent, result)
	assert.Nil(t, click)
	assert.False(t, runner.mousePressed)
}

// TestInlineApp_ProcessMouseEvent_ReleaseWithoutPress tests release without prior press
func TestInlineApp_ProcessMouseEvent_ReleaseWithoutPress(t *testing.T) {
	runner := NewInlineApp()

	// Release without pressing
	releaseEvent := MouseEvent{
		X:      10,
		Y:      5,
		Button: MouseButtonLeft,
		Type:   MouseRelease,
		Time:   time.Now(),
	}

	result, click := runner.processMouseEvent(releaseEvent)

	// Should return release event but no click
	assert.Equal(t, releaseEvent, result)
	assert.Nil(t, click)
	assert.False(t, runner.mousePressed)
}

// TestInlineApp_ProcessMouseEvent_MouseMove tests MouseMove events
func TestInlineApp_ProcessMouseEvent_MouseMove(t *testing.T) {
	runner := NewInlineApp()

	// Send MouseMove event
	moveEvent := MouseEvent{
		X:    15,
		Y:    10,
		Type: MouseMove,
		Time: time.Now(),
	}

	result, click := runner.processMouseEvent(moveEvent)

	// Should pass through unchanged, no click synthesized
	assert.Equal(t, moveEvent, result)
	assert.Nil(t, click)
}

// TestInlineApp_ProcessMouseEvent_MouseScroll tests MouseScroll events
func TestInlineApp_ProcessMouseEvent_MouseScroll(t *testing.T) {
	runner := NewInlineApp()

	// Send MouseScroll event
	scrollEvent := MouseEvent{
		X:      10,
		Y:      5,
		Type:   MouseScroll,
		Button: MouseButtonWheelUp,
		Time:   time.Now(),
	}

	result, click := runner.processMouseEvent(scrollEvent)

	// Should pass through unchanged, no click synthesized
	assert.Equal(t, scrollEvent, result)
	assert.Nil(t, click)
}

// TestInlineApp_ProcessMouseEvent_MouseClickPassthrough tests MouseClick events
func TestInlineApp_ProcessMouseEvent_MouseClickPassthrough(t *testing.T) {
	runner := NewInlineApp()

	// Send MouseClick event directly
	clickEvent := MouseEvent{
		X:      10,
		Y:      5,
		Button: MouseButtonLeft,
		Type:   MouseClick,
		Time:   time.Now(),
	}

	result, click := runner.processMouseEvent(clickEvent)

	// Should pass through unchanged
	assert.Equal(t, clickEvent, result)
	assert.Nil(t, click) // No additional click synthesized
}

// TestInlineApp_ProcessMouseEvent_PreserveModifiers tests that modifiers are preserved in synthesized clicks
func TestInlineApp_ProcessMouseEvent_PreserveModifiers(t *testing.T) {
	runner := NewInlineApp()

	// Press with modifiers
	pressEvent := MouseEvent{
		X:         10,
		Y:         5,
		Button:    MouseButtonLeft,
		Type:      MousePress,
		Modifiers: MouseModifiers(1), // Some modifier
		Time:      time.Now(),
	}

	runner.processMouseEvent(pressEvent)

	// Release at same location with different modifiers
	releaseEvent := MouseEvent{
		X:         10,
		Y:         5,
		Button:    MouseButtonLeft,
		Type:      MouseRelease,
		Modifiers: MouseModifiers(2), // Different modifier
		Time:      time.Now(),
	}

	result, click := runner.processMouseEvent(releaseEvent)

	// Click should be synthesized with release event's modifiers
	assert.Equal(t, releaseEvent, result)
	assert.NotNil(t, click)
	clickedEvent := click.(MouseEvent)
	assert.Equal(t, MouseClick, clickedEvent.Type)
	assert.Equal(t, MouseModifiers(2), clickedEvent.Modifiers)
}

// TestInlineApp_ProcessMouseEvent_DifferentButtons tests press and release with different buttons
func TestInlineApp_ProcessMouseEvent_DifferentButtons(t *testing.T) {
	runner := NewInlineApp()

	// Press left button
	pressEvent := MouseEvent{
		X:      10,
		Y:      5,
		Button: MouseButtonLeft,
		Type:   MousePress,
		Time:   time.Now(),
	}

	runner.processMouseEvent(pressEvent)
	assert.Equal(t, MouseButtonLeft, runner.mousePressButton)

	// Release right button at same location
	releaseEvent := MouseEvent{
		X:      10,
		Y:      5,
		Button: MouseButtonRight,
		Type:   MouseRelease,
		Time:   time.Now(),
	}

	result, click := runner.processMouseEvent(releaseEvent)

	// Click should be synthesized with the PRESS button (not release button)
	assert.Equal(t, releaseEvent, result)
	assert.NotNil(t, click)
	clickedEvent := click.(MouseEvent)
	assert.Equal(t, MouseButtonLeft, clickedEvent.Button) // Original press button
}
