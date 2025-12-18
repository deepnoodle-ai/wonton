// Example: envview - Interactive environment variable browser
//
// A TUI for browsing and editing environment variables and .env files.
// Perfect for debugging configuration issues and managing environment settings.
//
// Run with:
//
//	go run ./examples/envview
//	go run ./examples/envview .env .env.local
//	go run ./examples/envview --prefix MYAPP
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/env"
	"github.com/deepnoodle-ai/wonton/tui"
)

// EnvVar represents an environment variable with metadata
type EnvVar struct {
	Key    string
	Value  string
	Source string // "env", "file:.env", etc.
}

// EnvViewApp is the TUI application state
type EnvViewApp struct {
	// All variables
	allVars []EnvVar

	// Filtered variables (based on search)
	filteredVars []EnvVar

	// UI state
	selected     int
	scrollOffset int
	searchQuery  string
	searchMode   bool
	showValues   bool
	width        int
	height       int

	// Filter options
	prefix     string
	sourceFile string
}

func main() {
	app := cli.New("envview").
		Description("Interactive environment variable browser").
		Version("1.0.0")

	app.Main().
		Flags(
			cli.String("prefix", "p").
				Help("Filter variables by prefix"),
			cli.Bool("show-values", "v").
				Help("Show values by default (they are hidden for security)"),
		).
		Run(func(ctx *cli.Context) error {
			tuiApp := &EnvViewApp{
				showValues: ctx.Bool("show-values"),
				prefix:     ctx.String("prefix"),
			}

			// Load environment variables
			tuiApp.loadEnvVars()

			// Load .env files if specified
			files := ctx.Args()
			for _, file := range files {
				if err := tuiApp.loadEnvFile(file); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not load %s: %v\n", file, err)
				}
			}

			// Apply initial filter
			tuiApp.applyFilter()

			// Run TUI
			return tui.Run(tuiApp)
		})

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func (app *EnvViewApp) loadEnvVars() {
	// Get all environment variables
	for _, e := range os.Environ() {
		if k, v, ok := strings.Cut(e, "="); ok {
			app.allVars = append(app.allVars, EnvVar{
				Key:    k,
				Value:  v,
				Source: "environment",
			})
		}
	}

	// Sort by key
	sort.Slice(app.allVars, func(i, j int) bool {
		return app.allVars[i].Key < app.allVars[j].Key
	})
}

func (app *EnvViewApp) loadEnvFile(filename string) error {
	envMap, err := env.ReadEnvFile(filename)
	if err != nil {
		return err
	}

	source := "file:" + filename

	// Add or update variables from file
	for k, v := range envMap {
		found := false
		for i := range app.allVars {
			if app.allVars[i].Key == k {
				// Update existing - file value takes precedence for display
				app.allVars[i].Value = v
				app.allVars[i].Source = source
				found = true
				break
			}
		}
		if !found {
			app.allVars = append(app.allVars, EnvVar{
				Key:    k,
				Value:  v,
				Source: source,
			})
		}
	}

	// Re-sort
	sort.Slice(app.allVars, func(i, j int) bool {
		return app.allVars[i].Key < app.allVars[j].Key
	})

	return nil
}

func (app *EnvViewApp) applyFilter() {
	app.filteredVars = nil

	query := strings.ToLower(app.searchQuery)

	for _, v := range app.allVars {
		// Check prefix filter
		if app.prefix != "" && !strings.HasPrefix(v.Key, app.prefix) {
			continue
		}

		// Check search query
		if query != "" {
			keyLower := strings.ToLower(v.Key)
			valueLower := strings.ToLower(v.Value)
			if !strings.Contains(keyLower, query) && !strings.Contains(valueLower, query) {
				continue
			}
		}

		app.filteredVars = append(app.filteredVars, v)
	}

	// Reset selection if out of bounds
	if app.selected >= len(app.filteredVars) {
		app.selected = len(app.filteredVars) - 1
	}
	if app.selected < 0 {
		app.selected = 0
	}
}

func (app *EnvViewApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case tui.KeyEvent:
		// Handle search mode input
		if app.searchMode {
			switch e.Key {
			case tui.KeyEscape:
				app.searchMode = false
				app.searchQuery = ""
				app.applyFilter()
			case tui.KeyEnter:
				app.searchMode = false
			case tui.KeyBackspace:
				if len(app.searchQuery) > 0 {
					app.searchQuery = app.searchQuery[:len(app.searchQuery)-1]
					app.applyFilter()
				}
			default:
				if e.Rune != 0 {
					app.searchQuery += string(e.Rune)
					app.applyFilter()
				}
			}
			return nil
		}

		// Calculate page size
		listHeight := app.height - 12
		if listHeight < 5 {
			listHeight = 5
		}

		// Normal mode - navigation (less-compatible)
		switch e.Key {
		case tui.KeyEscape, tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
				app.adjustScroll()
			}
		case tui.KeyArrowDown:
			if app.selected < len(app.filteredVars)-1 {
				app.selected++
				app.adjustScroll()
			}
		case tui.KeyPageUp, tui.KeyCtrlB:
			app.selected -= listHeight
			if app.selected < 0 {
				app.selected = 0
			}
			app.adjustScroll()
		case tui.KeyPageDown, tui.KeyCtrlF:
			app.selected += listHeight
			if app.selected >= len(app.filteredVars) {
				app.selected = len(app.filteredVars) - 1
			}
			app.adjustScroll()
		case tui.KeyCtrlD:
			// Half page down
			app.selected += listHeight / 2
			if app.selected >= len(app.filteredVars) {
				app.selected = len(app.filteredVars) - 1
			}
			app.adjustScroll()
		case tui.KeyCtrlU:
			// Half page up
			app.selected -= listHeight / 2
			if app.selected < 0 {
				app.selected = 0
			}
			app.adjustScroll()
		case tui.KeyHome:
			app.selected = 0
			app.adjustScroll()
		case tui.KeyEnd:
			app.selected = len(app.filteredVars) - 1
			app.adjustScroll()
		}

		// Less-compatible rune keys for navigation
		switch e.Rune {
		case 'j':
			// Down one line
			if app.selected < len(app.filteredVars)-1 {
				app.selected++
				app.adjustScroll()
			}
		case 'k':
			// Up one line
			if app.selected > 0 {
				app.selected--
				app.adjustScroll()
			}
		case ' ', 'f':
			// Page down
			app.selected += listHeight
			if app.selected >= len(app.filteredVars) {
				app.selected = len(app.filteredVars) - 1
			}
			app.adjustScroll()
		case 'b':
			// Page up
			app.selected -= listHeight
			if app.selected < 0 {
				app.selected = 0
			}
			app.adjustScroll()
		case 'd':
			// Half page down
			app.selected += listHeight / 2
			if app.selected >= len(app.filteredVars) {
				app.selected = len(app.filteredVars) - 1
			}
			app.adjustScroll()
		case 'u':
			// Half page up
			app.selected -= listHeight / 2
			if app.selected < 0 {
				app.selected = 0
			}
			app.adjustScroll()
		case 'g':
			// Go to top
			app.selected = 0
			app.adjustScroll()
		case 'G':
			// Go to bottom
			app.selected = len(app.filteredVars) - 1
			app.adjustScroll()
		}

		// Commands
		switch e.Rune {
		case 'q', 'Q':
			return []tui.Cmd{tui.Quit()}
		case '/':
			app.searchMode = true
			app.searchQuery = ""
		case 'v':
			app.showValues = !app.showValues
		case 'c':
			// Copy selected key to clipboard
			if app.selected >= 0 && app.selected < len(app.filteredVars) {
				clipboard.Write(app.filteredVars[app.selected].Key)
			}
		case 'C':
			// Copy selected value to clipboard
			if app.selected >= 0 && app.selected < len(app.filteredVars) {
				clipboard.Write(app.filteredVars[app.selected].Value)
			}
		case 'e':
			// Export selected as KEY=VALUE
			if app.selected >= 0 && app.selected < len(app.filteredVars) {
				v := app.filteredVars[app.selected]
				clipboard.Write(fmt.Sprintf("%s=%s", v.Key, v.Value))
			}
		}
	}

	return nil
}

func (app *EnvViewApp) adjustScroll() {
	listHeight := app.height - 12 // Account for header, footer, detail panel
	if listHeight < 5 {
		listHeight = 5
	}

	// Adjust scroll to keep selected visible
	if app.selected < app.scrollOffset {
		app.scrollOffset = app.selected
	} else if app.selected >= app.scrollOffset+listHeight {
		app.scrollOffset = app.selected - listHeight + 1
	}
}

func (app *EnvViewApp) View() tui.View {
	// Header
	var headerText string
	if app.searchMode {
		headerText = fmt.Sprintf("Environment Variables  Search: %s_", app.searchQuery)
	} else {
		headerText = fmt.Sprintf("Environment Variables  [%d/%d]", len(app.filteredVars), len(app.allVars))
	}

	header := tui.HeaderBar(headerText).
		Bg(tui.ColorBlue).
		Fg(tui.ColorWhite)

	// Filter info
	var filterParts []string
	if app.prefix != "" {
		filterParts = append(filterParts, fmt.Sprintf("prefix: %s", app.prefix))
	}
	if app.searchQuery != "" && !app.searchMode {
		filterParts = append(filterParts, fmt.Sprintf("search: %s", app.searchQuery))
	}

	var filterBar tui.View
	if len(filterParts) > 0 {
		filterBar = tui.Text(" Filters: %s", strings.Join(filterParts, ", ")).Fg(tui.ColorYellow)
	} else {
		filterBar = tui.Text(" All variables").Fg(tui.ColorBrightBlack)
	}

	// Variable list
	listHeight := app.height - 12
	if listHeight < 5 {
		listHeight = 5
	}

	var listViews []tui.View
	if len(app.filteredVars) == 0 {
		listViews = append(listViews, tui.Text("No variables match the filter").Fg(tui.ColorBrightBlack))
	} else {
		end := app.scrollOffset + listHeight
		if end > len(app.filteredVars) {
			end = len(app.filteredVars)
		}

		for i := app.scrollOffset; i < end; i++ {
			v := app.filteredVars[i]
			listViews = append(listViews, app.formatVar(v, i == app.selected))
		}
	}

	// Detail panel for selected variable
	var detailViews []tui.View
	if app.selected >= 0 && app.selected < len(app.filteredVars) {
		v := app.filteredVars[app.selected]
		detailViews = app.formatDetail(v)
	} else {
		detailViews = []tui.View{
			tui.Text("No variable selected").Fg(tui.ColorBrightBlack),
		}
	}

	// Stats
	var valueStatus string
	if app.showValues {
		valueStatus = "shown"
	} else {
		valueStatus = "hidden"
	}
	statsBar := tui.Text(" Values: %s | Sources: %d env, %d file",
		valueStatus,
		app.countBySource("environment"),
		len(app.filteredVars)-app.countBySource("environment")).
		Fg(tui.ColorBrightBlack)

	// Help
	helpText := "jk/↑↓ nav | Space/b page | / search | v values | c/C copy | e export | q quit"

	return tui.Stack(
		header,
		filterBar,
		tui.Spacer().MinHeight(1),
		tui.Group(
			tui.Stack(
				tui.Bordered(
					tui.Stack(listViews...),
				).Title("Variables").BorderFg(tui.ColorCyan),
			),
			tui.Stack(
				tui.Bordered(
					tui.Stack(detailViews...).Padding(1),
				).Title("Detail").BorderFg(tui.ColorYellow),
			),
		),
		statsBar,
		tui.StatusBar(helpText),
	)
}

func (app *EnvViewApp) formatVar(v EnvVar, selected bool) tui.View {
	var bg, fg tui.Color
	if selected {
		bg = tui.ColorCyan
		fg = tui.ColorBlack
	} else {
		bg = tui.ColorDefault
		fg = tui.ColorWhite
	}

	// Truncate key if needed
	key := v.Key
	maxKeyLen := 30
	if len(key) > maxKeyLen {
		key = key[:maxKeyLen-3] + "..."
	}

	// Format value
	var valueDisplay string
	if app.showValues {
		valueDisplay = v.Value
		if len(valueDisplay) > 50 {
			valueDisplay = valueDisplay[:47] + "..."
		}
	} else {
		valueDisplay = strings.Repeat("*", min(len(v.Value), 20))
	}

	// Source indicator
	var sourceIcon string
	if strings.HasPrefix(v.Source, "file:") {
		sourceIcon = "F"
	} else {
		sourceIcon = "E"
	}

	return tui.Group(
		tui.Text(" %s ", sourceIcon).Fg(tui.ColorYellow).Bg(bg),
		tui.Text(" %-30s ", key).Fg(fg).Bg(bg).Bold(),
		tui.Text(" %s", valueDisplay).Fg(tui.ColorBrightBlack).Bg(bg),
	)
}

func (app *EnvViewApp) formatDetail(v EnvVar) []tui.View {
	views := []tui.View{
		tui.Text("Key:").Bold().Fg(tui.ColorCyan),
		tui.Text("  %s", v.Key),
		tui.Spacer().MinHeight(1),
		tui.Text("Value:").Bold().Fg(tui.ColorCyan),
	}

	// Show value (possibly truncated)
	if app.showValues {
		value := v.Value
		lines := strings.Split(value, "\n")
		for i, line := range lines {
			if i >= 5 {
				views = append(views, tui.Text("  ... (%d more lines)", len(lines)-5).Fg(tui.ColorBrightBlack))
				break
			}
			if len(line) > 60 {
				line = line[:57] + "..."
			}
			views = append(views, tui.Text("  %s", line))
		}
	} else {
		views = append(views, tui.Text("  [hidden - press 'v' to show]").Fg(tui.ColorBrightBlack))
	}

	views = append(views,
		tui.Spacer().MinHeight(1),
		tui.Text("Source:").Bold().Fg(tui.ColorCyan),
		tui.Text("  %s", v.Source),
		tui.Spacer().MinHeight(1),
		tui.Text("Length: %d characters", len(v.Value)).Fg(tui.ColorBrightBlack),
	)

	return views
}

func (app *EnvViewApp) countBySource(source string) int {
	count := 0
	for _, v := range app.filteredVars {
		if v.Source == source {
			count++
		}
	}
	return count
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
