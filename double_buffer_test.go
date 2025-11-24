package gooey

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Helper to create a test terminal
func createTestTerminal(width, height int) (*Terminal, *bytes.Buffer) {
	out := new(bytes.Buffer)
	term := NewTestTerminal(width, height, out)
	return term, out
}

func TestRaceCondition_UpdateAndDraw(t *testing.T) {
	term, _ := createTestTerminal(80, 24)
	sm := NewScreenManager(term, 60)
	sm.DefineRegion("test", 0, 0, 20, 1, false)
	sm.Start()
	defer sm.Stop()

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Update Region
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			sm.UpdateRegion("test", 0, fmt.Sprintf("Update %d", i), nil)
			time.Sleep(time.Millisecond)
		}
	}()

	// Goroutine 2: Redraw (implicit via Start, but let's force some via updates or just wait)
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
	}()

	wg.Wait()
	// If no panic, we passed (mostly). Use -race to really check.
}

func TestResizeBuffer(t *testing.T) {
	term, _ := createTestTerminal(10, 10)
	term.PrintAt(0, 0, "Hello")
	term.Flush()

	require.Equal(t, 'H', term.backBuffer[0][0].Char)

	// Resize smaller
	term.resizeBuffers(5, 5)
	term.width = 5
	term.height = 5

	require.Equal(t, 5, len(term.backBuffer))
	require.Equal(t, 5, len(term.backBuffer[0]))
	require.Equal(t, 'H', term.backBuffer[0][0].Char)

	// Resize larger
	term.resizeBuffers(10, 10)
	term.width = 10
	term.height = 10

	require.Equal(t, 10, len(term.backBuffer))
	require.Equal(t, 'H', term.backBuffer[0][0].Char) // Should persist
	require.Equal(t, ' ', term.backBuffer[9][9].Char) // New space

	// Check virtual cursor
	term.MoveCursor(8, 8)
	term.resizeBuffers(5, 5) // Cursor now OOB
	term.width = 5
	term.height = 5
	// Write should be safe (ignored) or handled
	term.Print("X")
	// virtualX is 8, width is 5.
	// Expectation: Should not panic.
}

func TestLineClearing(t *testing.T) {
	term, _ := createTestTerminal(20, 5)
	sm := NewScreenManager(term, 0) // 0 FPS, manual draw
	sm.DefineRegion("content", 0, 0, 20, 1, false)

	// 1. Draw text
	sm.UpdateRegion("content", 0, "Hello World", nil)
	sm.draw()
	require.Equal(t, 'H', term.backBuffer[0][0].Char)

	// 2. Clear text (empty string)
	sm.UpdateRegion("content", 0, "", nil)
	sm.draw()

	// BUG EXPECTATION: The text is NOT cleared because drawRegion skips empty strings
	// We assert what currently happens (bug presence) or what SHOULD happen?
	// Instructions say "Identify ... issues ... Create unit test cases ... to cover these most important cases".
	// Usually we write tests that FAIL if the bug exists, to prove the bug.
	// But if I write `require.Equal(t, ' ', ...)` it will fail.
	// I will write the assertion for CORRECT behavior, so the test fails, highlighting the bug.
	require.Equal(t, ' ', term.backBuffer[0][0].Char, "Line should be cleared when content is empty")
}

func TestMultiWidthChar(t *testing.T) {
	term, _ := createTestTerminal(20, 5)

	// Wide character 'Ａ' (U+FF21) - should take 2 cells
	term.Print("Ａ")

	// Check buffer - wide char should be at [0][0] with width 2
	require.Equal(t, 'Ａ', term.backBuffer[0][0].Char)
	require.Equal(t, 2, term.backBuffer[0][0].Width, "Wide character should have width 2")

	// The second cell should be a continuation cell
	require.Equal(t, true, term.backBuffer[0][1].Continuation, "Second cell should be continuation")

	// Next character 'B' should be at position [0][2] since wide char takes 2 cells
	term.Print("B")
	require.Equal(t, 'B', term.backBuffer[0][2].Char, "Next char should be after wide char (at position 2)")
}

func TestOverlappingRegions(t *testing.T) {
	term, _ := createTestTerminal(20, 5)
	sm := NewScreenManager(term, 0)

	sm.DefineRegion("layer1", 0, 0, 10, 1, false)
	sm.DefineRegion("layer2", 0, 0, 10, 1, false)

	sm.UpdateRegion("layer1", 0, "AAAAA", nil)
	sm.UpdateRegion("layer2", 0, "BBBBB", nil)

	sm.draw()

	// Non-deterministic which one wins.
	// We just ensure one of them is there.
	char := term.backBuffer[0][0].Char
	require.True(t, char == 'A' || char == 'B')
}

func TestScrollUp(t *testing.T) {
	term, _ := createTestTerminal(10, 3)

	term.Println("Line 1")
	term.Println("Line 2")
	term.Println("Line 3") // Cursor at (0, 3) -> Scroll -> (0, 2)

	// Buffer should contain:
	// Line 2
	// Line 3
	// (empty) -> Wait, Println does: Print("Line 3"), then \n (scrolls if at bottom)

	// Check line 0
	// "Line 2 " (padded)
	require.Equal(t, 'L', term.backBuffer[0][0].Char)
	require.Equal(t, 'i', term.backBuffer[0][1].Char)
	require.Equal(t, '2', term.backBuffer[0][5].Char)

	// Check line 1
	// "Line 3 "
	require.Equal(t, '3', term.backBuffer[1][5].Char)

	// Check line 2 (cursor is here now)
	require.Equal(t, ' ', term.backBuffer[2][0].Char)
}

func TestColorsAndStyles(t *testing.T) {
	term, out := createTestTerminal(20, 5)

	style := NewStyle().WithForeground(ColorRed)
	term.SetStyle(style)
	term.Print("Red")
	term.Reset()
	term.Print("Default")

	term.Flush()

	output := out.String()
	require.Contains(t, output, "31m") // Red code
	require.Contains(t, output, "Red")
	require.Contains(t, output, "Default")
	// Should contain reset code \033[0m
	require.Contains(t, output, "\033[0m")
}

func TestClearLine_Background(t *testing.T) {
	term, _ := createTestTerminal(20, 5)

	// Set Blue BG
	bg := NewStyle().WithBackground(ColorBlue)
	term.SetStyle(bg)
	term.Print("Test") // Sets "Test" with Blue BG

	// Clear line
	term.ClearLine()

	// Check first cell style.
	// ClearLine now uses current style (Blue BG), so it should be Blue BG.
	cell := term.backBuffer[0][0]
	require.Equal(t, ColorBlue, cell.Style.Background)

	// Verify current style is still Blue BG?
	// ClearLine doesn't change current style.
	// But ClearToEndOfLine MIGHT use it?

	term.SetStyle(bg)
	term.PrintAt(0, 1, "Test2")
	term.MoveCursor(0, 1)
	term.ClearToEndOfLine()

	// ClearToEndOfLine uses current style if BG is set
	cell2 := term.backBuffer[1][0]
	require.Equal(t, ColorBlue, cell2.Style.Background)
}

func TestOutOfBounds(t *testing.T) {
	term, _ := createTestTerminal(10, 10)

	require.NotPanics(t, func() {
		term.PrintAt(-1, -1, "OutOfBounds")
		term.PrintAt(100, 100, "WayOut")
	})

	// Verify nothing written to (0,0)
	require.Equal(t, ' ', term.backBuffer[0][0].Char)
}

func TestNilAnimation(t *testing.T) {

	term, _ := createTestTerminal(10, 10)

	sm := NewScreenManager(term, 0)

	sm.DefineRegion("test", 0, 0, 10, 1, false)

	// Update with nil animation (normal usage)

	sm.UpdateRegion("test", 0, "Hello", nil)

	require.NotPanics(t, func() {

		sm.draw()

	})

	require.Equal(t, 'H', term.backBuffer[0][0].Char)

}

func TestRegionInterference(t *testing.T) {

	term, _ := createTestTerminal(20, 5)

	sm := NewScreenManager(term, 0)

	// Region on left

	sm.DefineRegion("left", 0, 0, 5, 1, false)

	// Text on right (manually placed or another region)

	term.PrintAt(10, 0, "Right")

	// Update left

	sm.UpdateRegion("left", 0, "Left", nil)

	sm.draw()

	// Check Right text

	require.Equal(t, 'R', term.backBuffer[0][10].Char, "Drawing left region should not wipe right side")

}
