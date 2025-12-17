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
	width   int
	height  int
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
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

		// Handle scrolling
		pageSize := app.height - 3
		if pageSize < 1 {
			pageSize = 1
		}

		switch e.Key {
		case tui.KeyArrowUp:
			if app.scrollY > 0 {
				app.scrollY--
			}
		case tui.KeyArrowDown:
			app.scrollY++
		case tui.KeyPageUp:
			app.scrollY -= pageSize
			if app.scrollY < 0 {
				app.scrollY = 0
			}
		case tui.KeyPageDown:
			app.scrollY += pageSize
		case tui.KeyHome:
			app.scrollY = 0
		case tui.KeyEnd:
			app.scrollY = 1000 // will be clamped
		}

	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// View returns the declarative view structure.
func (app *DiffDemoApp) View() tui.View {
	diffHeight := app.height - 2
	if diffHeight < 1 {
		diffHeight = 1
	}

	statusStyle := tui.NewStyle().
		WithBackground(tui.ColorBlue).
		WithForeground(tui.ColorWhite)

	return tui.Stack(
		tui.DiffView(app.diff, "go", &app.scrollY).
			Height(diffHeight).
			ShowLineNumbers(true),
		tui.Text(" Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump ").
			Style(statusStyle),
	)
}

func main() {
	app := &DiffDemoApp{}
	if err := app.Init(); err != nil {
		log.Fatal(err)
	}
	if err := tui.Run(app); err != nil {
		log.Fatal(err)
	}
}
