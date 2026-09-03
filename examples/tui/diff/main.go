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
	diff          *tui.Diff
	scrollY       int
	width, height int
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
	case tui.ResizeEvent:
		app.width, app.height = e.Width, e.Height
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		page := max(app.viewHeight()-1, 1)
		switch e.Key {
		case tui.KeyArrowUp:
			app.scrollBy(-1)
		case tui.KeyArrowDown:
			app.scrollBy(1)
		case tui.KeyPageUp:
			app.scrollBy(-page)
		case tui.KeyPageDown:
			app.scrollBy(page)
		case tui.KeyHome:
			app.scrollY = 0
		case tui.KeyEnd:
			app.scrollY = app.maxScroll()
		}
	}

	return nil
}

// viewHeight is what the border and the status line leave for the diff.
func (app *DiffDemoApp) viewHeight() int { return max(app.height-3, 1) }

// maxScroll is roughly how far the content can move before its last line
// reaches the bottom. Measure reports the height the diff would render at,
// without a terminal to render it to. It measures the content alone, so the
// answer is an upper bound — ScrollView clamps scrollY to the exact bottom
// when it renders, and writes the clamped value back.
func (app *DiffDemoApp) maxScroll() int {
	if app.width == 0 {
		return 0
	}
	_, contentHeight := tui.Measure(app.diffView(), app.width-2, 0)
	return max(contentHeight-app.viewHeight(), 0)
}

func (app *DiffDemoApp) scrollBy(lines int) {
	app.scrollY = min(max(app.scrollY+lines, 0), app.maxScroll())
}

func (app *DiffDemoApp) diffView() tui.View {
	return tui.DiffView(app.diff, "go", nil).ShowLineNumbers(true)
}

// View returns the declarative view structure.
func (app *DiffDemoApp) View() tui.View {
	statusStyle := tui.NewStyle().
		WithBackground(tui.ColorBlue).
		WithForeground(tui.ColorWhite)

	return tui.Stack(
		tui.Bordered(
			tui.Scroll(app.diffView(), &app.scrollY),
		).BorderFg(tui.ColorCyan).Title("Diff Viewer"),
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
