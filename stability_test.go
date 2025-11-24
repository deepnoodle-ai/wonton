package gooey

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTerminal_PrintStyled_Concurrency(t *testing.T) {
	// This test verifies that PrintStyled is atomic and thread-safe.

	var buf bytes.Buffer
	term := NewTestTerminal(100, 100, &buf)

	var wg sync.WaitGroup
	iterations := 100
	wg.Add(2)

	// Goroutine 1: Prints "RED" in Red
	go func() {
		defer wg.Done()
		redStyle := NewStyle().WithForeground(ColorRed)
		for i := 0; i < iterations; i++ {
			// Use atomic PrintStyled
			term.PrintStyled("RED", redStyle)
			time.Sleep(time.Nanosecond)
		}
	}()

	// Goroutine 2: Prints "BLUE" in Blue
	go func() {
		defer wg.Done()
		blueStyle := NewStyle().WithForeground(ColorBlue)
		for i := 0; i < iterations; i++ {
			// Use atomic PrintStyled
			term.PrintStyled("BLUE", blueStyle)
			time.Sleep(time.Nanosecond)
		}
	}()

	wg.Wait()

	// Check the buffer for inconsistencies
	errorsFound := 0

	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			cell := term.backBuffer[y][x]
			if cell.Char == 'R' {
				if cell.Style.Foreground != ColorRed {
					t.Logf("Found 'R' with wrong color at %d,%d: %v", x, y, cell.Style.Foreground)
					errorsFound++
				}
			}
			if cell.Char == 'B' {
				if cell.Style.Foreground != ColorBlue {
					t.Logf("Found 'B' with wrong color at %d,%d: %v", x, y, cell.Style.Foreground)
					errorsFound++
				}
			}
		}
	}

	if errorsFound > 0 {
		t.Fatalf("Race condition persisted: %d cells had wrong style", errorsFound)
	}
}

func TestInvalidFPS(t *testing.T) {
	// Verify that 0 or negative FPS defaults to 30 and doesn't panic

	var buf bytes.Buffer
	term := NewTestTerminal(80, 24, &buf)

	// Test Animator
	anim := NewAnimator(term, 0)
	if anim.fps != 30 {
		t.Errorf("Expected default FPS 30 for Animator, got %d", anim.fps)
	}

	anim2 := NewAnimator(term, -10)
	if anim2.fps != 30 {
		t.Errorf("Expected default FPS 30 for Animator, got %d", anim2.fps)
	}

	// Test ScreenManager
	sm := NewScreenManager(term, 0)
	if sm.fps != 30 {
		t.Errorf("Expected default FPS 30 for ScreenManager, got %d", sm.fps)
	}

	sm2 := NewScreenManager(term, -5)
	if sm2.fps != 30 {
		t.Errorf("Expected default FPS 30 for ScreenManager, got %d", sm2.fps)
	}
}

func TestScreenManager_ConcurrentUpdates(t *testing.T) {
	// This test checks for data races or panics in ScreenManager under heavy load.

	var buf bytes.Buffer
	term := NewTestTerminal(80, 24, &buf)
	sm := NewScreenManager(term, 60)
	sm.Start()
	defer sm.Stop()

	sm.DefineRegion("header", 0, 0, 80, 1, true)
	sm.DefineRegion("content", 0, 1, 80, 20, false)

	var wg sync.WaitGroup
	workers := 10
	updates := 100
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < updates; j++ {
				sm.UpdateRegion("content", j%20, fmt.Sprintf("Worker %d Iter %d", id, j), nil)
			}
		}(i)
	}

	wg.Wait()
	// If we reached here without panic/race (detected by -race), it's a partial success.
}
