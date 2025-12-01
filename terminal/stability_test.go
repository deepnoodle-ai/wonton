package terminal

import (
	"bytes"
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
