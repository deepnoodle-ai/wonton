package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

const sampleDiff = `diff --git a/server.go b/server.go
--- a/server.go
+++ b/server.go
@@ -1,15 +1,18 @@
 package main

 import (
-	"fmt"
+	"log"
 	"net/http"
+	"os"
 )

-func handler(w http.ResponseWriter, r *http.Request) {
-	fmt.Fprintf(w, "Hello, World!")
+func handlerFunc(w http.ResponseWriter, r *http.Request) {
+	log.Printf("Request from %s", r.RemoteAddr)
+	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
 }

 func main() {
-	http.HandleFunc("/", handler)
-	http.ListenAndServe(":8080", nil)
+	port := os.Getenv("PORT")
+	if port == "" {
+		port = "8080"
+	}
+	log.Printf("Starting server on port %s", port)
+	http.HandleFunc("/", handlerFunc)
+	if err := http.ListenAndServe(":"+port, nil); err != nil {
+		log.Fatal(err)
+	}
 }`

// DiffDemoApp demonstrates diff viewing using the Runtime architecture.
// It shows how to use the DiffViewer widget with syntax highlighting and scrolling.
type DiffDemoApp struct {
	viewer *gooey.DiffViewer
	width  int
	height int
}

// Init initializes the application by creating the diff viewer.
func (app *DiffDemoApp) Init() error {
	viewer, err := gooey.NewDiffViewer(sampleDiff, "go")
	if err != nil {
		return fmt.Errorf("failed to create diff viewer: %w", err)
	}
	app.viewer = viewer

	// Set initial bounds (will be updated on first resize event)
	app.viewer.SetBounds(image.Rect(0, 0, app.width, app.height-2))
	app.viewer.Init()

	return nil
}

// HandleEvent processes events from the runtime.
func (app *DiffDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		app.viewer.HandleKey(e)

	case gooey.ResizeEvent:
		// Update dimensions and viewer bounds on resize
		app.width = e.Width
		app.height = e.Height
		app.viewer.SetBounds(image.Rect(0, 0, e.Width, e.Height-2))
	}

	return nil
}

// Render draws the current application state.
func (app *DiffDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Draw the diff viewer
	app.viewer.Draw(frame)

	// Draw status line at bottom
	statusStyle := gooey.NewStyle().
		WithBackground(gooey.ColorBlue).
		WithForeground(gooey.ColorWhite)

	statusMsg := "Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump"
	statusLine := fmt.Sprintf(" %s ", statusMsg)

	// Pad to full width
	for len(statusLine) < width {
		statusLine += " "
	}
	if len(statusLine) > width {
		statusLine = statusLine[:width]
	}

	frame.PrintStyled(0, height-1, statusLine, statusStyle)

	// Draw scroll indicator
	scrollInfo := fmt.Sprintf(" Line %d/%d ",
		app.viewer.GetScrollPosition()+1,
		app.viewer.GetLineCount())
	scrollX := width - len(scrollInfo) - 1
	if scrollX > 0 && scrollX+len(scrollInfo) <= width {
		frame.PrintStyled(scrollX, height-1, scrollInfo, statusStyle)
	}
}

func main() {
	// Create and initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Get initial terminal size
	width, height := terminal.Size()

	// Create the application
	app := &DiffDemoApp{
		width:  width,
		height: height,
	}

	// Create and run the runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
