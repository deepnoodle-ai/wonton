// Package main demonstrates the simplest form of scrollback + live region.
//
// This minimal example shows:
//   - Static content printed with tui.Print (scrollback history)
//   - Dynamic content updated with tui.LivePrinter (live region)
//   - Multiple async processes updating different parts of the live region
//   - Raw keyboard input for interactive control
//
// Run with: go run ./examples/scrollback_simple
package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/terminal"
	"github.com/deepnoodle-ai/wonton/tui"
	"golang.org/x/term"
)

// AsyncState represents the state of an async process shown in the live region.
type AsyncState struct {
	name     string
	progress int
	status   string
	color    tui.Color
	items    []string
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// === SCROLLBACK SECTION ===
	fmt.Println("=== Scrollback + Live Region Demo ===")
	fmt.Println()

	tui.Print(tui.Text("Press keys to add various views to scrollback:").Fg(tui.ColorCyan))
	fmt.Println()
	printKeyHelp()
	fmt.Println()

	// Enter raw mode for keyboard input
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Println("Requires interactive terminal")
		return
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Printf("Failed to enable raw mode: %v\n", err)
		return
	}
	defer term.Restore(fd, oldState)

	// === LIVE REGION with two async processes ===
	live := tui.NewLivePrinter(tui.PrintConfig{Width: 70})
	defer live.Stop()

	// Two independent async processes
	process1 := &AsyncState{
		name:   "Download",
		status: "Idle",
		color:  tui.ColorCyan,
		items:  []string{},
	}
	process2 := &AsyncState{
		name:   "Processing",
		status: "Idle",
		color:  tui.ColorMagenta,
		items:  []string{},
	}

	// Channels for async updates
	update1 := make(chan struct{}, 1)
	update2 := make(chan struct{}, 1)
	done := make(chan struct{})

	// Async process 1: simulates downloads
	go func() {
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				updateProcess1(process1)
				select {
				case update1 <- struct{}{}:
				default:
				}
			}
		}
	}()

	// Async process 2: simulates data processing
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				updateProcess2(process2)
				select {
				case update2 <- struct{}{}:
				default:
				}
			}
		}
	}()

	// Render initial state
	live.Update(buildLiveView(process1, process2))

	decoder := terminal.NewKeyDecoder(os.Stdin)
	running := true
	entryCount := 0

	// We need non-blocking input. Use a goroutine to read keys.
	keyChan := make(chan terminal.KeyEvent, 10)
	go func() {
		for {
			event, err := decoder.ReadKeyEvent()
			if err != nil {
				if err == io.EOF {
					close(keyChan)
					return
				}
				continue
			}
			keyChan <- event
		}
	}()

	for running {
		select {
		case <-update1:
			live.Update(buildLiveView(process1, process2))

		case <-update2:
			live.Update(buildLiveView(process1, process2))

		case event, ok := <-keyChan:
			if !ok {
				running = false
				break
			}

			switch event.Key {
			case terminal.KeyCtrlC:
				running = false
			default:
				if event.Rune != 0 {
					switch event.Rune {
					case 'q':
						running = false

					case '1': // Simple text
						live.Clear()
						printSimpleText(entryCount)
						entryCount++
						live.Update(buildLiveView(process1, process2))

					case '2': // Styled text
						live.Clear()
						printStyledText(entryCount)
						entryCount++
						live.Update(buildLiveView(process1, process2))

					case '3': // Bordered box
						live.Clear()
						printBorderedBox(entryCount)
						entryCount++
						live.Update(buildLiveView(process1, process2))

					case '4': // Nested stack
						live.Clear()
						printNestedStack(entryCount)
						entryCount++
						live.Update(buildLiveView(process1, process2))

					case '5': // Status table
						live.Clear()
						printStatusTable(entryCount)
						entryCount++
						live.Update(buildLiveView(process1, process2))

					case '6': // Random complex view
						live.Clear()
						printRandomComplex(entryCount)
						entryCount++
						live.Update(buildLiveView(process1, process2))

					case 'r': // Random view
						live.Clear()
						printRandomView(entryCount)
						entryCount++
						live.Update(buildLiveView(process1, process2))

					case 'h': // Help
						live.Clear()
						printKeyHelp()
						fmt.Println()
						live.Update(buildLiveView(process1, process2))
					}
				}
			}
		}
	}

	close(done)
	live.Clear()
	fmt.Println("Goodbye!")
}

func printKeyHelp() {
	tui.Print(tui.Stack(
		tui.Group(tui.Text("  1").Fg(tui.ColorYellow), tui.Text(" - Simple text")),
		tui.Group(tui.Text("  2").Fg(tui.ColorYellow), tui.Text(" - Styled text")),
		tui.Group(tui.Text("  3").Fg(tui.ColorYellow), tui.Text(" - Bordered box")),
		tui.Group(tui.Text("  4").Fg(tui.ColorYellow), tui.Text(" - Nested stack")),
		tui.Group(tui.Text("  5").Fg(tui.ColorYellow), tui.Text(" - Status table")),
		tui.Group(tui.Text("  6").Fg(tui.ColorYellow), tui.Text(" - Complex view")),
		tui.Group(tui.Text("  r").Fg(tui.ColorYellow), tui.Text(" - Random view")),
		tui.Group(tui.Text("  h").Fg(tui.ColorYellow), tui.Text(" - Show help")),
		tui.Group(tui.Text("  q").Fg(tui.ColorYellow), tui.Text(" - Quit")),
	))
}

// === Scrollback view generators ===

func printSimpleText(n int) {
	messages := []string{
		"The quick brown fox jumps over the lazy dog.",
		"All good things must come to an end.",
		"A journey of a thousand miles begins with a single step.",
		"To be or not to be, that is the question.",
		"In the beginning, there was nothing.",
	}
	printRaw(tui.Group(
		tui.Text("[%d] ", n).Dim(),
		tui.Text("%s", messages[rand.Intn(len(messages))]),
	))
}

func printStyledText(n int) {
	words := []string{"Important", "Notice", "Warning", "Success", "Info", "Debug"}
	word := words[rand.Intn(len(words))]

	// Generate random style combination
	styleChoice := rand.Intn(4)
	var styledWord tui.View
	switch styleChoice {
	case 0:
		styledWord = tui.Text("%s: ", word).Bold().Fg(tui.ColorRed)
	case 1:
		styledWord = tui.Text("%s: ", word).Italic().Fg(tui.ColorGreen)
	case 2:
		styledWord = tui.Text("%s: ", word).Underline().Fg(tui.ColorBlue)
	case 3:
		styledWord = tui.Text("%s: ", word).Bold().Italic().Fg(tui.ColorYellow)
	}

	printRaw(tui.Group(
		tui.Text("[%d] ", n).Dim(),
		styledWord,
		tui.Text("This is a styled message with random formatting."),
	))
}

func printBorderedBox(n int) {
	borders := []*tui.BorderStyle{&tui.RoundedBorder, &tui.SingleBorder, &tui.DoubleBorder, &tui.ThickBorder}
	colors := []tui.Color{tui.ColorCyan, tui.ColorMagenta, tui.ColorYellow, tui.ColorGreen}

	titles := []string{"Status Update", "Notification", "Alert", "Message"}
	bodies := []string{
		"Operation completed successfully.",
		"New data has been received.",
		"Please review the following items.",
		"System status is nominal.",
	}

	printRaw(
		tui.Bordered(
			tui.Stack(
				tui.Text("%s #%d", titles[rand.Intn(len(titles))], n).Bold(),
				tui.Text(""),
				tui.Text("%s", bodies[rand.Intn(len(bodies))]),
			).Padding(1),
		).Border(borders[rand.Intn(len(borders))]).BorderFg(colors[rand.Intn(len(colors))]),
		tui.PrintConfig{Width: 50},
	)
}

func printNestedStack(n int) {
	items := rand.Intn(4) + 2 // 2-5 items
	var views []tui.View
	views = append(views, tui.Text("Entry #%d - Nested Items:", n).Bold())

	for i := 0; i < items; i++ {
		subItems := rand.Intn(3) + 1 // 1-3 sub-items
		var subViews []tui.View
		subViews = append(subViews, tui.Group(
			tui.Text("  ").Fg(tui.ColorBrightBlack),
			tui.Text("Item %d:", i+1).Fg(tui.ColorCyan),
		))

		for j := 0; j < subItems; j++ {
			subViews = append(subViews, tui.Group(
				tui.Text("    - ").Fg(tui.ColorBrightBlack),
				tui.Text("Sub-item %d.%d", i+1, j+1).Dim(),
			))
		}
		views = append(views, tui.Stack(subViews...))
	}

	printRaw(tui.Stack(views...))
}

func printStatusTable(n int) {
	services := []string{"API", "Database", "Cache", "Queue", "Storage"}
	statuses := []struct {
		text  string
		color tui.Color
	}{
		{"Online", tui.ColorGreen},
		{"Degraded", tui.ColorYellow},
		{"Offline", tui.ColorRed},
		{"Starting", tui.ColorCyan},
	}

	var rows []tui.View
	rows = append(rows, tui.Text("Service Status Report #%d", n).Bold().Underline())
	rows = append(rows, tui.Text(""))

	numServices := rand.Intn(3) + 2
	for i := 0; i < numServices; i++ {
		svc := services[rand.Intn(len(services))]
		status := statuses[rand.Intn(len(statuses))]
		latency := rand.Intn(200) + 10

		rows = append(rows, tui.Group(
			tui.Text("  %-10s", svc),
			tui.Text("  "),
			tui.Text("%-10s", status.text).Fg(status.color),
			tui.Text("  "),
			tui.Text("%dms", latency).Dim(),
		))
	}

	printRaw(tui.Stack(rows...))
}

func printRandomComplex(n int) {
	// A more complex nested view with multiple elements
	header := tui.Group(
		tui.Text("[").Fg(tui.ColorBrightBlack),
		tui.Text("%s", time.Now().Format("15:04:05")).Fg(tui.ColorCyan),
		tui.Text("]").Fg(tui.ColorBrightBlack),
		tui.Text(" Complex Entry #%d", n).Bold(),
	)

	metrics := tui.Bordered(
		tui.Stack(
			tui.Text("Metrics").Bold().Fg(tui.ColorYellow),
			tui.Group(tui.Text("  CPU: "), tui.Text("%d%%", rand.Intn(100)).Fg(tui.ColorGreen)),
			tui.Group(tui.Text("  Mem: "), tui.Text("%d%%", rand.Intn(100)).Fg(tui.ColorCyan)),
			tui.Group(tui.Text("  I/O: "), tui.Text("%d MB/s", rand.Intn(500)).Fg(tui.ColorMagenta)),
		),
	).Border(&tui.RoundedBorder).BorderFg(tui.ColorBrightBlack)

	events := []string{"Request received", "Processing started", "Cache hit", "Response sent"}
	var eventViews []tui.View
	eventViews = append(eventViews, tui.Text("Recent Events:").Dim())
	for i := 0; i < rand.Intn(3)+2; i++ {
		eventViews = append(eventViews, tui.Group(
			tui.Text("  - ").Fg(tui.ColorBrightBlack),
			tui.Text("%s", events[rand.Intn(len(events))]),
		))
	}

	printRaw(tui.Stack(
		header,
		tui.Text(""),
		metrics,
		tui.Text(""),
		tui.Stack(eventViews...),
	), tui.PrintConfig{Width: 60})
}

func printRandomView(n int) {
	choice := rand.Intn(6)
	switch choice {
	case 0:
		printSimpleText(n)
	case 1:
		printStyledText(n)
	case 2:
		printBorderedBox(n)
	case 3:
		printNestedStack(n)
	case 4:
		printStatusTable(n)
	case 5:
		printRandomComplex(n)
	}
}

// === Async process updaters ===

func updateProcess1(p *AsyncState) {
	if p.progress >= 100 {
		// Reset and start new download
		p.progress = 0
		files := []string{"data.zip", "assets.tar.gz", "update.pkg", "backup.sql", "logs.txt"}
		p.status = fmt.Sprintf("Downloading %s", files[rand.Intn(len(files))])
		p.items = append(p.items, p.status)
		if len(p.items) > 3 {
			p.items = p.items[1:]
		}
	} else {
		p.progress += rand.Intn(8) + 1
		if p.progress > 100 {
			p.progress = 100
		}
	}
}

func updateProcess2(p *AsyncState) {
	if p.progress >= 100 {
		// Reset and start new task
		p.progress = 0
		tasks := []string{"Parsing JSON", "Compiling", "Optimizing", "Validating", "Indexing"}
		p.status = tasks[rand.Intn(len(tasks))]
		p.items = append(p.items, fmt.Sprintf("Completed: %s", p.status))
		if len(p.items) > 3 {
			p.items = p.items[1:]
		}
	} else {
		p.progress += rand.Intn(5) + 2
		if p.progress > 100 {
			p.progress = 100
		}
	}
}

// === Live region builder ===

func buildLiveView(p1, p2 *AsyncState) tui.View {
	return tui.Stack(
		tui.Divider().Fg(tui.ColorBrightBlack),
		tui.Text(" Live Region - Two Async Processes").Bold(),
		tui.Divider().Fg(tui.ColorBrightBlack),
		tui.Text(""),
		buildProcessView(p1),
		tui.Text(""),
		buildProcessView(p2),
		tui.Text(""),
		tui.Divider().Fg(tui.ColorBrightBlack),
		tui.Group(
			tui.Text(" Press ").Dim(),
			tui.Text("1-6").Fg(tui.ColorYellow),
			tui.Text(" to add views, ").Dim(),
			tui.Text("r").Fg(tui.ColorYellow),
			tui.Text(" random, ").Dim(),
			tui.Text("q").Fg(tui.ColorRed),
			tui.Text(" quit").Dim(),
		),
	)
}

func buildProcessView(p *AsyncState) tui.View {
	// Build progress bar
	barWidth := 20
	filled := (p.progress * barWidth) / 100
	empty := barWidth - filled

	var barParts []tui.View
	barParts = append(barParts, tui.Text("["))
	if filled > 0 {
		barParts = append(barParts, tui.Text("%s", repeatChar('█', filled)).Fg(p.color))
	}
	if empty > 0 {
		barParts = append(barParts, tui.Text("%s", repeatChar('░', empty)).Fg(tui.ColorBrightBlack))
	}
	barParts = append(barParts, tui.Text("] "))
	barParts = append(barParts, tui.Text("%3d%%", p.progress).Fg(tui.ColorYellow))

	// Build item list
	var itemViews []tui.View
	for _, item := range p.items {
		itemViews = append(itemViews, tui.Group(
			tui.Text("      - ").Fg(tui.ColorBrightBlack),
			tui.Text("%s", item).Dim(),
		))
	}

	views := []tui.View{
		tui.Group(
			tui.Text("  %s: ", p.name).Fg(p.color).Bold(),
			tui.Text("%s", p.status),
		),
		tui.Group(append([]tui.View{tui.Text("    ")}, barParts...)...),
	}
	views = append(views, itemViews...)

	return tui.Stack(views...)
}

func repeatChar(ch rune, count int) string {
	result := make([]rune, count)
	for i := range result {
		result[i] = ch
	}
	return string(result)
}

// printRaw prints a view in raw mode, converting \n to \r\n for proper display.
// In raw mode, \n only moves down without returning to column 0.
func printRaw(view tui.View, opts ...tui.PrintConfig) {
	var buf bytes.Buffer
	opts = append(opts, tui.PrintConfig{Output: &buf})
	tui.Print(view, opts...)

	// Convert \n to \r\n for raw mode
	output := bytes.ReplaceAll(buf.Bytes(), []byte("\n"), []byte("\r\n"))
	fmt.Print("\r") // Ensure we start at column 0
	os.Stdout.Write(output)
	fmt.Print("\r\n") // End with proper line termination
}
