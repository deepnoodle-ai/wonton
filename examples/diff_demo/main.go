package main

import (
	"fmt"
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

// DiffDemoApp demonstrates the declarative DiffView.
type DiffDemoApp struct {
	diff    *gooey.Diff
	scrollY int
	width   int
	height  int
}

// Init initializes the application by parsing the diff.
func (app *DiffDemoApp) Init() error {
	diff, err := gooey.ParseUnifiedDiff(sampleDiff)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}
	app.diff = diff
	return nil
}

// HandleEvent processes events from the runtime.
func (app *DiffDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

		// Handle scrolling
		pageSize := app.height - 3
		if pageSize < 1 {
			pageSize = 1
		}

		switch e.Key {
		case gooey.KeyArrowUp:
			if app.scrollY > 0 {
				app.scrollY--
			}
		case gooey.KeyArrowDown:
			app.scrollY++
		case gooey.KeyPageUp:
			app.scrollY -= pageSize
			if app.scrollY < 0 {
				app.scrollY = 0
			}
		case gooey.KeyPageDown:
			app.scrollY += pageSize
		case gooey.KeyHome:
			app.scrollY = 0
		case gooey.KeyEnd:
			app.scrollY = 1000 // will be clamped
		}

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// View returns the declarative view structure.
func (app *DiffDemoApp) View() gooey.View {
	diffHeight := app.height - 2
	if diffHeight < 1 {
		diffHeight = 1
	}

	statusStyle := gooey.NewStyle().
		WithBackground(gooey.ColorBlue).
		WithForeground(gooey.ColorWhite)

	return gooey.VStack(
		gooey.DiffView(app.diff, "go", &app.scrollY).
			Height(diffHeight).
			ShowLineNumbers(true),
		gooey.Text(" Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump ").
			Style(statusStyle),
	)
}

func main() {
	app := &DiffDemoApp{}
	if err := app.Init(); err != nil {
		log.Fatal(err)
	}
	if err := gooey.Run(app); err != nil {
		log.Fatal(err)
	}
}
