package main

import (
	"fmt"
	"image"
	"log"

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

// View returns the declarative view structure.
func (app *DiffDemoApp) View() gooey.View {
	statusStyle := gooey.NewStyle().
		WithBackground(gooey.ColorBlue).
		WithForeground(gooey.ColorWhite)

	statusMsg := "Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump"
	scrollInfo := fmt.Sprintf("Line %d/%d",
		app.viewer.GetScrollPosition()+1,
		app.viewer.GetLineCount())

	return gooey.VStack(
		// Diff viewer area
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			app.viewer.Draw(frame)
		}),

		// Status line at bottom
		gooey.Height(1, gooey.Background(' ', statusStyle, gooey.HStack(
			gooey.Text(" %s ", statusMsg).Style(statusStyle),
			gooey.Spacer(),
			gooey.Text(" %s ", scrollInfo).Style(statusStyle),
		))),
	)
}

func main() {
	if err := gooey.Run(&DiffDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
