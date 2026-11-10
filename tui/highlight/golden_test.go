package highlight

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/termtest"
	"github.com/deepnoodle-ai/wonton/tui"
	"github.com/deepnoodle-ai/wonton/tui/tuitest"
)

// Golden tests compare rendered output against saved snapshot files in
// testdata/snapshots. Run with -update to create or update them.

func TestGolden_Code_Go(t *testing.T) {
	code := `func main() {
	fmt.Println("Hello")
}`
	view := Code(code, "go").LineNumbers(true)
	screen := tuitest.Screen(view, tui.WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Code_NoLineNumbers(t *testing.T) {
	code := `print("Hello")`
	view := Code(code, "python").LineNumbers(false)
	screen := tuitest.Screen(view, tui.WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Code_PlainText(t *testing.T) {
	code := "Just some plain text\nWith multiple lines"
	view := Code(code, "text").LineNumbers(true)
	screen := tuitest.Screen(view, tui.WithWidth(35))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Code_InPanel(t *testing.T) {
	// Code snippet within a bordered panel
	code := `func hello() {
    fmt.Println("Hi")
}`
	view := tui.Bordered(
		tui.Stack(
			tui.Text("main.go").Dim(),
			tui.Divider(),
			Code(code, "go").LineNumbers(true),
		),
	).Border(&tui.RoundedBorder).Title("Source")
	screen := tuitest.Screen(view, tui.WithWidth(35), tui.WithHeight(10))
	termtest.AssertScreen(t, screen)
}

func TestGolden_DiffView_Basic(t *testing.T) {
	// Basic diff display
	scrollY := 0
	diffText := `--- a/test.go
+++ b/test.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
 func main() {
 }`
	view, _ := DiffViewFromText(diffText, "go", &scrollY)
	view.ShowLineNumbers(true).Height(8)
	screen := tuitest.Screen(view, tui.WithWidth(40), tui.WithHeight(8))
	termtest.AssertScreen(t, screen)
}
