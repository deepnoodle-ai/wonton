// Example: shapes demonstrates basic GIF creation with geometric shapes.
//
// Run with: go run ./examples/gif/shapes
package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/deepnoodle-ai/wonton/gif"
)

func main() {
	// Create a 200x200 GIF
	g := gif.New(200, 200)
	g.SetLoopCount(0) // Loop forever

	// Create 60 frames (about 3 seconds at 50ms/frame)
	for i := 0; i < 60; i++ {
		angle := float64(i) * math.Pi / 30 // Rotate over time

		g.AddFrameWithDelay(func(f *gif.Frame) {
			// Dark blue background
			f.Fill(gif.RGB(20, 30, 50))

			// Rotating squares
			cx, cy := 100, 100
			size := 40

			for j := 0; j < 4; j++ {
				a := angle + float64(j)*math.Pi/2
				x := cx + int(60*math.Cos(a))
				y := cy + int(60*math.Sin(a))

				// Draw filled squares with different colors
				colors := []color.RGBA{
					gif.RGB(255, 100, 100), // Red
					gif.RGB(100, 255, 100), // Green
					gif.RGB(100, 100, 255), // Blue
					gif.RGB(255, 255, 100), // Yellow
				}
				f.FillRect(x-size/2, y-size/2, size, size, colors[j])
				f.DrawRect(x-size/2, y-size/2, size, size, gif.White)
			}

			// Center circle
			f.FillCircle(cx, cy, 20, gif.RGB(255, 200, 100))
			f.DrawCircle(cx, cy, 20, gif.White)

			// Pulsating outer ring
			ringRadius := 80 + int(10*math.Sin(angle*2))
			f.DrawCircle(cx, cy, ringRadius, gif.Cyan)
		}, 5) // 50ms per frame
	}

	// Save the GIF
	if err := g.Save("shapes.gif"); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Created shapes.gif")
}
