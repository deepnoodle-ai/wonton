package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	// This demo shows how ScreenManager prevents flickering when updating
	// multiple regions at high frequency.

	terminal, err := gooey.NewTerminal()
	if err != nil {
		panic(err)
	}

	// Enable alternate screen for full immersion
	terminal.EnableAlternateScreen()
	terminal.HideCursor()
	defer func() {
		terminal.Close()
	}()

	// Create ScreenManager at 60 FPS
	sm := gooey.NewScreenManager(terminal, 60)

	width, height := terminal.Size()
	var quadrantHeight int

	// Function to set up regions based on current terminal size
	setupRegions := func(w, h int) {
		// Define a header
		sm.DefineRegion("header", 0, 0, w, 3, false)
		sm.UpdateRegion("header", 0, "⚡ Flicker-Free Update Demo ⚡", gooey.CreateRainbowText("⚡ Flicker-Free Update Demo ⚡", 10))
		sm.UpdateRegion("header", 1, "Updates occur every 10ms. Press Ctrl+C to exit.", nil)
		sm.UpdateRegion("header", 2, "--------------------------------------------------", nil)

		// Define 4 data quadrants
		midX := w / 2
		midY := (h-3)/2 + 3
		quadrantHeight = midY - 4

		sm.DefineRegion("q1", 2, 4, midX-4, quadrantHeight, false)
		sm.DefineRegion("q2", midX+2, 4, midX-4, quadrantHeight, false)
		sm.DefineRegion("q3", 2, midY+2, midX-4, quadrantHeight, false)
		sm.DefineRegion("q4", midX+2, midY+2, midX-4, quadrantHeight, false)
	}

	// Initial setup
	setupRegions(width, height)

	// Enable automatic resize handling
	terminal.WatchResize()
	defer terminal.StopWatchResize()

	terminal.OnResize(func(w, h int) {
		width, height = w, h
		setupRegions(w, h)
	})

	sm.Start()
	defer sm.Stop()

	// Handle Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		// Fast update loop
		chars := []string{"@", "#", "$", "%", "&", "*", "+", "=", "?", "!"}
		colors := []gooey.RGB{
			gooey.NewRGB(255, 0, 0),
			gooey.NewRGB(0, 255, 0),
			gooey.NewRGB(0, 0, 255),
			gooey.NewRGB(255, 255, 0),
			gooey.NewRGB(0, 255, 255),
			gooey.NewRGB(255, 0, 255),
		}

		for {
			select {
			case <-c:
				return // Will be caught by main channel
			default:
				// Randomly update lines in quadrants
				updateQuadrant(sm, "q1", chars, colors, quadrantHeight)
				updateQuadrant(sm, "q2", chars, colors, quadrantHeight)
				updateQuadrant(sm, "q3", chars, colors, quadrantHeight)
				updateQuadrant(sm, "q4", chars, colors, quadrantHeight)

				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	<-c // Wait for interrupt
}

func updateQuadrant(sm *gooey.ScreenManager, region string, chars []string, colors []gooey.RGB, height int) {
	// Update a random line
	line := rand.Intn(height)

	// Build a random string
	str := ""
	for i := 0; i < 20; i++ {
		str += chars[rand.Intn(len(chars))]
	}

	// Add a timestamp
	str += fmt.Sprintf(" %d", time.Now().UnixNano())

	// Create a pulse animation for this update
	color := colors[rand.Intn(len(colors))]
	anim := gooey.CreatePulseText(color, 30+rand.Intn(20))

	sm.UpdateRegion(region, line, str, anim)
}
