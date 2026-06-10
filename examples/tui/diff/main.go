// Package main demonstrates the declarative DiffView: a unified diff is
// parsed and rendered with syntax highlighting, line numbers, and scrolling.
//
// Run with: go run ./examples/tui/diff
package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
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

// DiffDemoApp demonstrates the declarative DiffView.
type DiffDemoApp struct {
	diff    *tui.Diff
	scrollY int
}

// Init initializes the application by parsing the diff.
func (app *DiffDemoApp) Init() error {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}
	app.diff = diff
	return nil
}

// HandleEvent processes events from the runtime.
func (app *DiffDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch {
		case e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case e.Key == tui.KeyArrowUp:
			app.scrollBy(-1)
		case e.Key == tui.KeyArrowDown:
			app.scrollBy(1)
		case e.Key == tui.KeyPageUp:
			app.scrollBy(-10)
		case e.Key == tui.KeyPageDown:
			app.scrollBy(10)
		case e.Key == tui.KeyHome:
			app.scrollY = 0
		case e.Key == tui.KeyEnd:
			app.scrollY = 1 << 30 // Clamped to the bottom by the scroll view
		}
	}

	return nil
}

// scrollBy adjusts the scroll offset; the scroll view clamps the maximum.
func (app *DiffDemoApp) scrollBy(delta int) {
	app.scrollY += delta
	if app.scrollY < 0 {
		app.scrollY = 0
	}
}

// View returns the declarative view structure.
func (app *DiffDemoApp) View() tui.View {
	statusStyle := tui.NewStyle().
		WithBackground(tui.ColorBlue).
		WithForeground(tui.ColorWhite)

	return tui.Stack(
		tui.Bordered(
			tui.Scroll(
				tui.DiffView(app.diff, "go", nil).ShowLineNumbers(true),
				&app.scrollY,
			),
		).BorderFg(tui.ColorCyan).Title("Diff Viewer"),
		tui.Text(" Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump ").
			Style(statusStyle),
	)
}

func main() {
	// tui.Run calls Init automatically (DiffDemoApp implements Initializable).
	if err := tui.Run(&DiffDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
