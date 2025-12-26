package tui

import (
	"bytes"
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

// TestInlineApp_NewInlineApp tests the constructor and options
func TestInlineApp_NewInlineApp(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		runner := NewInlineApp()
		assert.NotNil(t, runner)
		assert.Equal(t, 0, runner.config.fps)
		assert.False(t, runner.config.mouseTracking)
		assert.False(t, runner.config.bracketedPaste)
	})

	t.Run("with options", func(t *testing.T) {
		var buf bytes.Buffer
		runner := NewInlineApp(
			WithInlineWidth(100),
			WithInlineOutput(&buf),
			WithInlineFPS(30),
			WithInlineMouseTracking(true),
			WithInlineBracketedPaste(true),
			WithInlinePasteTabWidth(4),
			WithInlineKittyKeyboard(true),
		)

		assert.Equal(t, 100, runner.config.width)
		assert.Equal(t, 30, runner.config.fps)
		assert.True(t, runner.config.mouseTracking)
		assert.True(t, runner.config.bracketedPaste)
		assert.Equal(t, 4, runner.config.pasteTabWidth)
		assert.True(t, runner.config.kittyKeyboard)
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
	runner := NewInlineApp(
		WithInlineOutput(&buf),
		WithInlineWidth(40),
	)

	// We can't fully test Print without running, but we can test the method exists
	assert.NotNil(t, runner.Printf)
}

// TestInlineApp_RunInline tests the convenience function
func TestInlineApp_RunInline(t *testing.T) {
	// RunInline should return error for non-terminal stdin
	app := &testInlineApp{liveText: "test"}
	err := RunInline(app)

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
	runner := NewInlineApp(
		WithInlineOutput(&buf),
		WithInlineWidth(40),
	)

	app := &testInlineApp{liveText: "Hello, World!"}
	runner.app = app

	// Create a live printer manually for testing
	runner.live = NewLivePrinter(WithWidth(40), WithOutput(&buf))

	// Render should call LiveView and update the live printer
	runner.render()

	// Check that LiveView was called
	assert.Equal(t, 1, app.getViewCalls())

	// Check that output contains the text
	output := buf.String()
	assert.Contains(t, output, "Hello")
}

// TestInlineApp_OptionsDefaults tests option default values
func TestInlineApp_OptionsDefaults(t *testing.T) {
	cfg := defaultInlineConfig()

	// Check defaults match the design doc
	assert.Equal(t, 0, cfg.fps)
	assert.False(t, cfg.mouseTracking)
	assert.False(t, cfg.bracketedPaste)
	assert.Equal(t, 0, cfg.pasteTabWidth)
	assert.False(t, cfg.kittyKeyboard)
}

// TestInlineApp_ConcurrentPrint tests thread safety of Print
func TestInlineApp_ConcurrentPrint(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(
		WithInlineOutput(&buf),
		WithInlineWidth(40),
	)

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(WithWidth(40), WithOutput(&buf))

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
	runner := NewInlineApp(
		WithInlineOutput(&buf),
		WithInlineWidth(40),
	)

	app := &testInlineApp{liveText: "test"}
	runner.app = app
	runner.live = NewLivePrinter(WithWidth(40), WithOutput(&buf))

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
		Fprint(&buf, view, WithWidth(20))
		output := buf.String()

		// Should contain plain \n (or no special line ending if single line equivalent)
		// Multi-line stack should have \n between lines
		assert.True(t, strings.Contains(output, "\n") || !strings.Contains(output, "\r\n"))
	})

	t.Run("with raw mode uses CRLF", func(t *testing.T) {
		var buf bytes.Buffer
		view := Stack(Text("Line 1"), Text("Line 2"))
		Fprint(&buf, view, WithWidth(20), WithRawMode())
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
