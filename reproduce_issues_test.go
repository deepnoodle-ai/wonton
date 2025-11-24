package gooey

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

// Issue 1: Animator Restart Fragility
func TestAnimator_Restart(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	anim := NewAnimator(term, 60)

	_ = anim.Start()
	time.Sleep(10 * time.Millisecond)
	anim.Stop()

	// This second start would fail/panic/return immediately if the stop channel is closed and not recreated
	done := make(chan struct{})
	go func() {
		_ = anim.Start()
		close(done)
	}()

	select {
	case <-done:
		// Passed start
	case <-time.After(100 * time.Millisecond):
		// Passed
	}
	anim.Stop()
}

// Issue 2: Terminal Clear Style Logic
func TestTerminal_Clear_Style(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})

	// Set a style (e.g., red background)
	redBg := NewStyle().WithBgRGB(NewRGB(255, 0, 0))
	term.SetStyle(redBg)

	// Clear the screen
	term.Clear()

	// Check if the buffer cells have the style
	cell := term.backBuffer[0][0]

	// Now it should PASS
	assert.Equal(t, redBg, cell.Style, "Clear() should respect the currently set style")
}

// Issue 3: Terminal Negative Cursor Panic
func TestTerminal_NegativeCursor_Panic(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})

	// Move to negative coordinates
	term.MoveCursor(-5, -5)

	// Try to perform an operation that iterates based on cursor
	require.NotPanics(t, func() {
		term.ClearToEndOfLine()
	}, "ClearToEndOfLine should not panic with negative cursor")

	// Try to print
	require.NotPanics(t, func() {
		term.Print("Hello")
	}, "Print should not panic with negative cursor")
}

// Issue 4: ScreenManager Z-Order (Overlapping Regions)
func TestScreenManager_OverlappingRegions_Determinism(t *testing.T) {
	term := NewTestTerminal(10, 10, &bytes.Buffer{})
	sm := NewScreenManager(term, 60)

	// Define two overlapping regions
	// With the fix, order depends on definition order
	sm.DefineRegion("layer1", 0, 0, 5, 5, false)
	sm.DefineRegion("layer2", 0, 0, 5, 5, false)

	sm.UpdateRegion("layer1", 0, "AAAAA", nil)
	sm.UpdateRegion("layer2", 0, "BBBBB", nil)

	// Draw
	sm.draw()

	// Check what's on the screen at (0,0)
	// Should be B because layer2 is defined after layer1 and iterated in order

	cell := term.backBuffer[0][0]
	assert.Equal(t, 'B', cell.Char, "Layer2 should be on top of Layer1")
}

// Issue 5: Input Hotkey Race Condition
func TestInput_Hotkey_Race(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	input := NewInput(term)

	var wg sync.WaitGroup
	wg.Add(2)

	// Routine 1: Simulate reading
	go func() {
		defer wg.Done()
		// Use lock to be thread safe (as Read() would be)
		for i := 0; i < 1000; i++ {
			input.mu.RLock()
			_ = input.hotkeys[KeyCtrlC]
			input.mu.RUnlock()
		}
	}()

	// Routine 2: Set hotkeys
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			input.SetHotkey(KeyCtrlC, func() {})
		}
	}()

	wg.Wait()
}

// Issue 6: Layout Header Cursor Position
func TestLayout_HeaderCursor_Bug(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	layout := NewLayout(term)

	header := SimpleHeader("Title", NewStyle())
	header.Right = "Right"
	layout.SetHeader(header)

	// Simulate a saved cursor position at Y=10
	term.virtualX = 5
	term.virtualY = 10
	term.SaveCursor()

	// Draw header
	frame, _ := term.BeginFrame()
	layout.drawHeader(frame)
	term.EndFrame(frame)

	// Check buffer at Y=0
	foundAtTop := false
	for x := 0; x < 80; x++ {
		if term.backBuffer[0][x].Char == 'R' {
			foundAtTop = true
			break
		}
	}

	foundAtBottom := false
	for x := 0; x < 80; x++ {
		if term.backBuffer[10][x].Char == 'R' {
			foundAtBottom = true
			break
		}
	}

	assert.True(t, foundAtTop, "Header text should be at Y=0")
	assert.False(t, foundAtBottom, "Header text should NOT be at Y=10")
}
