package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
	"golang.org/x/term"
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

func main() {
	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Enable raw mode for key reading
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Enable alternate screen
	terminal.EnableAlternateScreen()

	// Get terminal size
	width, height := terminal.Size()

	// Create diff viewer
	viewer, err := gooey.NewDiffViewer(sampleDiff, "go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating diff viewer: %v\n", err)
		os.Exit(1)
	}

	viewer.SetBounds(image.Rect(0, 0, width, height-2)) // Leave room for status line
	viewer.Init()

	// Create status message
	statusMsg := "Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump"

	// Create key decoder
	decoder := gooey.NewKeyDecoder(os.Stdin)

	// Main render loop
	for {
		// Begin frame
		frame, err := terminal.BeginFrame()
		if err != nil {
			break
		}

		// Clear screen
		frame.Fill(' ', gooey.NewStyle())

		// Draw the diff viewer
		viewer.Draw(frame)

		// Draw status line at bottom
		statusStyle := gooey.NewStyle().
			WithBackground(gooey.ColorBlue).
			WithForeground(gooey.ColorWhite)

		statusLine := fmt.Sprintf(" %s ", statusMsg)
		// Pad to full width
		for len(statusLine) < width {
			statusLine += " "
		}
		frame.PrintStyled(0, height-1, statusLine[:width], statusStyle)

		// Draw scroll indicator
		scrollInfo := fmt.Sprintf(" Line %d/%d ",
			viewer.GetScrollPosition()+1,
			viewer.GetLineCount())
		scrollX := width - len(scrollInfo) - 1
		if scrollX > 0 {
			frame.PrintStyled(scrollX, height-1, scrollInfo, statusStyle)
		}

		// End frame
		terminal.EndFrame(frame)

		// Handle input
		key, err := decoder.ReadKeyEvent()
		if err != nil {
			break
		}

		// Handle quit
		if key.Rune == 'q' || key.Rune == 'Q' {
			return
		}

		// Let the viewer handle the key
		viewer.HandleKey(key)
	}
}
