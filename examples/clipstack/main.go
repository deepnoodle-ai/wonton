// Example: clipstack - Clipboard history manager
//
// A TUI for managing clipboard history. View, search, and paste from your
// clipboard history. Perfect for developers who copy lots of snippets.
//
// Run with:
//
//	go run ./examples/clipstack
//	go run ./examples/clipstack --max 50       # Keep 50 items max
//	go run ./examples/clipstack --poll 500ms   # Check every 500ms
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/humanize"
	"github.com/deepnoodle-ai/wonton/tui"
)

// ClipItem represents a clipboard history entry
type ClipItem struct {
	Content   string
	Hash      string
	Timestamp time.Time
	Pinned    bool
	Type      string // text, code, url, etc.
}

// ClipStackApp is the TUI application
type ClipStackApp struct {
	mu sync.Mutex

	// History
	items    []ClipItem
	maxItems int
	hashes   map[string]bool // For deduplication

	// Selection and display
	selected     int
	scrollOffset int
	filterText   string
	filtering    bool
	width        int
	height       int

	// Polling
	pollInterval time.Duration
	lastContent  string
	stopPolling  chan struct{}

	// Status
	statusMsg string
}

func main() {
	app := cli.New("clipstack").
		Description("Clipboard history manager with TUI").
		Version("1.0.0")

	app.Main().
		Flags(
			cli.Int("max", "m").
				Default(100).
				Help("Maximum history items to keep"),
			cli.Int("poll", "p").
				Default(250).
				Help("Clipboard poll interval in milliseconds"),
		).
		Run(func(ctx *cli.Context) error {
			if !clipboard.Available() {
				return cli.Error("Clipboard not available on this system")
			}

			tuiApp := &ClipStackApp{
				maxItems:     ctx.Int("max"),
				pollInterval: time.Duration(ctx.Int("poll")) * time.Millisecond,
				hashes:       make(map[string]bool),
				stopPolling:  make(chan struct{}),
				statusMsg:    "↑↓ navigate | Enter paste | p pin | d delete | / search | q quit",
			}

			// Get initial clipboard content
			if content, err := clipboard.Read(); err == nil && content != "" {
				tuiApp.addItem(content)
			}

			// Start polling
			go tuiApp.pollClipboard()

			// Run TUI
			err := tui.Run(tuiApp)

			// Stop polling
			close(tuiApp.stopPolling)

			return err
		})

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func (app *ClipStackApp) pollClipboard() {
	ticker := time.NewTicker(app.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-app.stopPolling:
			return
		case <-ticker.C:
			content, err := clipboard.Read()
			if err != nil || content == "" {
				continue
			}

			app.mu.Lock()
			if content != app.lastContent {
				app.lastContent = content
				app.addItemLocked(content)
			}
			app.mu.Unlock()
		}
	}
}

func (app *ClipStackApp) addItem(content string) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.addItemLocked(content)
}

func (app *ClipStackApp) addItemLocked(content string) {
	// Skip empty or whitespace-only
	if strings.TrimSpace(content) == "" {
		return
	}

	// Calculate hash for deduplication
	hash := hashContent(content)

	// Check if already exists
	if app.hashes[hash] {
		// Move existing item to top
		for i, item := range app.items {
			if item.Hash == hash {
				app.items = append(app.items[:i], app.items[i+1:]...)
				item.Timestamp = time.Now()
				app.items = append([]ClipItem{item}, app.items...)
				return
			}
		}
		return
	}

	// Detect content type
	contentType := detectContentType(content)

	// Add new item at the top
	item := ClipItem{
		Content:   content,
		Hash:      hash,
		Timestamp: time.Now(),
		Type:      contentType,
	}

	app.items = append([]ClipItem{item}, app.items...)
	app.hashes[hash] = true

	// Trim to max size (keeping pinned items)
	app.trimHistory()

	app.lastContent = content
}

func (app *ClipStackApp) trimHistory() {
	if len(app.items) <= app.maxItems {
		return
	}

	// Keep all pinned items + most recent unpinned up to maxItems
	var pinned []ClipItem
	var unpinned []ClipItem

	for _, item := range app.items {
		if item.Pinned {
			pinned = append(pinned, item)
		} else {
			unpinned = append(unpinned, item)
		}
	}

	// Remove oldest unpinned items
	maxUnpinned := app.maxItems - len(pinned)
	if maxUnpinned < 0 {
		maxUnpinned = 0
	}
	if len(unpinned) > maxUnpinned {
		// Remove hashes of removed items
		for i := maxUnpinned; i < len(unpinned); i++ {
			delete(app.hashes, unpinned[i].Hash)
		}
		unpinned = unpinned[:maxUnpinned]
	}

	// Rebuild items list
	app.items = append(pinned, unpinned...)
}

func (app *ClipStackApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case tui.KeyEvent:
		// Handle filter mode
		if app.filtering {
			return app.handleFilterKey(e)
		}

		// Quit
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyCtrlC || e.Key == tui.KeyEscape {
			return []tui.Cmd{tui.Quit()}
		}

		app.mu.Lock()
		defer app.mu.Unlock()

		filteredItems := app.getFilteredItems()

		// Calculate page size for navigation
		listHeight := app.height - 8
		if listHeight < 5 {
			listHeight = 5
		}

		// Navigation (less-compatible)
		switch e.Key {
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
			}
		case tui.KeyArrowDown:
			if app.selected < len(filteredItems)-1 {
				app.selected++
			}
		case tui.KeyHome:
			app.selected = 0
		case tui.KeyEnd:
			app.selected = len(filteredItems) - 1
		case tui.KeyPageUp, tui.KeyCtrlB:
			app.selected -= listHeight
			if app.selected < 0 {
				app.selected = 0
			}
		case tui.KeyPageDown, tui.KeyCtrlF:
			app.selected += listHeight
			if app.selected >= len(filteredItems) {
				app.selected = len(filteredItems) - 1
			}
		case tui.KeyCtrlD:
			// Half page down
			app.selected += listHeight / 2
			if app.selected >= len(filteredItems) {
				app.selected = len(filteredItems) - 1
			}
		case tui.KeyCtrlU:
			// Half page up
			app.selected -= listHeight / 2
			if app.selected < 0 {
				app.selected = 0
			}
		case tui.KeyEnter:
			if app.selected >= 0 && app.selected < len(filteredItems) {
				item := filteredItems[app.selected]
				if err := clipboard.Write(item.Content); err != nil {
					app.statusMsg = fmt.Sprintf("Error: %v", err)
				} else {
					app.statusMsg = "✓ Copied to clipboard"
				}
			}
		}

		// Less-compatible rune keys for navigation
		switch e.Rune {
		case 'j':
			// Down one line
			if app.selected < len(filteredItems)-1 {
				app.selected++
			}
		case 'k':
			// Up one line
			if app.selected > 0 {
				app.selected--
			}
		case ' ', 'f':
			// Page down
			app.selected += listHeight
			if app.selected >= len(filteredItems) {
				app.selected = len(filteredItems) - 1
			}
		case 'b':
			// Page up
			app.selected -= listHeight
			if app.selected < 0 {
				app.selected = 0
			}
		case 'g':
			// Go to top
			app.selected = 0
		case 'G':
			// Go to bottom
			app.selected = len(filteredItems) - 1
		}

		// Commands (note: 'd' is delete, 'c' is clear - not less navigation)
		switch e.Rune {
		case '/':
			app.filtering = true
			app.filterText = ""
			app.statusMsg = "Type to filter, Enter to confirm, Esc to cancel"
		case 'p', 'P':
			if app.selected >= 0 && app.selected < len(filteredItems) {
				// Find and toggle pin on actual item
				item := filteredItems[app.selected]
				for i := range app.items {
					if app.items[i].Hash == item.Hash {
						app.items[i].Pinned = !app.items[i].Pinned
						if app.items[i].Pinned {
							app.statusMsg = "📌 Item pinned"
						} else {
							app.statusMsg = "📌 Item unpinned"
						}
						break
					}
				}
			}
		case 'd', 'D':
			if app.selected >= 0 && app.selected < len(filteredItems) {
				item := filteredItems[app.selected]
				for i := range app.items {
					if app.items[i].Hash == item.Hash {
						delete(app.hashes, item.Hash)
						app.items = append(app.items[:i], app.items[i+1:]...)
						if app.selected >= len(app.getFilteredItems()) {
							app.selected = len(app.getFilteredItems()) - 1
						}
						app.statusMsg = "🗑 Item deleted"
						break
					}
				}
			}
		case 'c', 'C':
			// Clear unpinned items
			var pinned []ClipItem
			for _, item := range app.items {
				if item.Pinned {
					pinned = append(pinned, item)
				} else {
					delete(app.hashes, item.Hash)
				}
			}
			app.items = pinned
			app.selected = 0
			app.statusMsg = "🗑 Cleared unpinned items"
		}
	}

	return nil
}

func (app *ClipStackApp) handleFilterKey(e tui.KeyEvent) []tui.Cmd {
	app.mu.Lock()
	defer app.mu.Unlock()

	switch e.Key {
	case tui.KeyEscape:
		app.filtering = false
		app.filterText = ""
		app.statusMsg = "↑↓ navigate | Enter paste | p pin | d delete | / search | q quit"
	case tui.KeyEnter:
		app.filtering = false
		app.selected = 0
		if app.filterText != "" {
			app.statusMsg = fmt.Sprintf("Filtered by: %s (Esc to clear)", app.filterText)
		} else {
			app.statusMsg = "↑↓ navigate | Enter paste | p pin | d delete | / search | q quit"
		}
	case tui.KeyBackspace:
		if len(app.filterText) > 0 {
			app.filterText = app.filterText[:len(app.filterText)-1]
		}
	default:
		if e.Rune != 0 && unicode.IsPrint(e.Rune) {
			app.filterText += string(e.Rune)
			app.selected = 0
		}
	}

	return nil
}

func (app *ClipStackApp) getFilteredItems() []ClipItem {
	if app.filterText == "" {
		return app.items
	}

	filter := strings.ToLower(app.filterText)
	var filtered []ClipItem
	for _, item := range app.items {
		if strings.Contains(strings.ToLower(item.Content), filter) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (app *ClipStackApp) View() tui.View {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Header
	header := tui.HeaderBar(fmt.Sprintf("📋 Clipboard History  [%d items, max: %d]",
		len(app.items), app.maxItems)).
		Bg(tui.ColorBlue).
		Fg(tui.ColorWhite)

	// Filter bar
	var filterBar tui.View
	if app.filtering {
		filterBar = tui.Group(
			tui.Text(" / ").Fg(tui.ColorYellow).Bold(),
			tui.Text("%s", app.filterText).Fg(tui.ColorWhite),
			tui.Text("█").Fg(tui.ColorWhite), // Cursor
		)
	} else if app.filterText != "" {
		filterBar = tui.Group(
			tui.Text(" Filter: ").Fg(tui.ColorBrightBlack),
			tui.Text("%s", app.filterText).Fg(tui.ColorYellow),
			tui.Spacer(),
			tui.Text("(Esc to clear)").Fg(tui.ColorBrightBlack),
		)
	} else {
		filterBar = tui.Spacer().MinHeight(1)
	}

	// Get filtered items
	filteredItems := app.getFilteredItems()

	// Ensure selected is in bounds
	if app.selected >= len(filteredItems) {
		app.selected = len(filteredItems) - 1
	}
	if app.selected < 0 {
		app.selected = 0
	}

	// Item list
	var listViews []tui.View

	if len(filteredItems) == 0 {
		if len(app.items) == 0 {
			listViews = append(listViews,
				tui.Spacer(),
				tui.Text("No clipboard history yet").Fg(tui.ColorBrightBlack),
				tui.Spacer().MinHeight(1),
				tui.Text("Copy something to get started!").Fg(tui.ColorBrightBlack),
				tui.Spacer())
		} else {
			listViews = append(listViews,
				tui.Spacer(),
				tui.Text("No items match filter").Fg(tui.ColorYellow),
				tui.Spacer())
		}
	} else {
		// Calculate visible range
		listHeight := app.height - 8
		if listHeight < 5 {
			listHeight = 5
		}

		start := app.scrollOffset
		end := start + listHeight
		if end > len(filteredItems) {
			end = len(filteredItems)
		}

		// Adjust scroll to keep selected visible
		if app.selected < start {
			app.scrollOffset = app.selected
			start = app.scrollOffset
			end = start + listHeight
			if end > len(filteredItems) {
				end = len(filteredItems)
			}
		} else if app.selected >= end {
			app.scrollOffset = app.selected - listHeight + 1
			if app.scrollOffset < 0 {
				app.scrollOffset = 0
			}
			start = app.scrollOffset
			end = start + listHeight
			if end > len(filteredItems) {
				end = len(filteredItems)
			}
		}

		for i := start; i < end; i++ {
			listViews = append(listViews, app.formatItem(filteredItems[i], i == app.selected))
		}
	}

	// Detail panel
	var detailViews []tui.View
	if app.selected >= 0 && app.selected < len(filteredItems) {
		item := filteredItems[app.selected]
		detailViews = app.formatDetail(item)
	} else {
		detailViews = []tui.View{
			tui.Text("Select an item to view").Fg(tui.ColorBrightBlack),
		}
	}

	// Main layout
	return tui.Stack(
		header,
		filterBar,
		tui.Group(
			// List
			tui.Stack(
				tui.Bordered(
					tui.Stack(listViews...),
				).Title("History").BorderFg(tui.ColorCyan),
			),
			// Detail
			tui.Stack(
				tui.Bordered(
					tui.Stack(detailViews...).Padding(1),
				).Title("Preview").BorderFg(tui.ColorYellow),
			),
		),
		tui.StatusBar(app.statusMsg),
	)
}

func (app *ClipStackApp) formatItem(item ClipItem, selected bool) tui.View {
	// Icon based on type
	icon := "📄"
	switch item.Type {
	case "url":
		icon = "🔗"
	case "code":
		icon = "💻"
	case "json":
		icon = "📊"
	case "path":
		icon = "📁"
	}

	if item.Pinned {
		icon = "📌"
	}

	// Preview text
	preview := item.Content
	preview = strings.ReplaceAll(preview, "\n", "↵")
	preview = strings.ReplaceAll(preview, "\t", "→")
	maxLen := app.width/2 - 20
	if maxLen < 20 {
		maxLen = 20
	}
	if len(preview) > maxLen {
		preview = preview[:maxLen-3] + "..."
	}

	// Time ago
	timeAgo := humanize.Duration(time.Since(item.Timestamp))

	var bg tui.Color
	var fg tui.Color
	if selected {
		bg = tui.ColorCyan
		fg = tui.ColorBlack
	} else {
		bg = tui.ColorDefault
		fg = tui.ColorWhite
	}

	return tui.Group(
		tui.Text(" %s ", icon).Bg(bg),
		tui.Text("%s", preview).Fg(fg).Bg(bg),
		tui.Spacer(),
		tui.Text("%s", timeAgo).Fg(tui.ColorBrightBlack).Bg(bg),
		tui.Text(" ").Bg(bg),
	)
}

func (app *ClipStackApp) formatDetail(item ClipItem) []tui.View {
	var views []tui.View

	// Type and time
	views = append(views,
		tui.Group(
			tui.Text("Type: %s", item.Type).Fg(tui.ColorYellow),
			tui.Spacer(),
			tui.Text("%s", item.Timestamp.Format("15:04:05")).Fg(tui.ColorBrightBlack),
		),
	)

	if item.Pinned {
		views = append(views, tui.Text("📌 Pinned").Fg(tui.ColorMagenta))
	}

	// Size
	views = append(views,
		tui.Text("Size: %s", humanize.Bytes(int64(len(item.Content)))).Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
	)

	// Content preview
	lines := strings.Split(item.Content, "\n")
	maxLines := 15
	if len(lines) > maxLines {
		for i := 0; i < maxLines-1; i++ {
			line := lines[i]
			if len(line) > 60 {
				line = line[:57] + "..."
			}
			views = append(views, tui.Text("%s", line).Fg(tui.ColorWhite))
		}
		views = append(views, tui.Text("... (%d more lines)", len(lines)-maxLines+1).Fg(tui.ColorBrightBlack))
	} else {
		for _, line := range lines {
			if len(line) > 60 {
				line = line[:57] + "..."
			}
			views = append(views, tui.Text("%s", line).Fg(tui.ColorWhite))
		}
	}

	return views
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:8]) // First 8 bytes
}

func detectContentType(content string) string {
	content = strings.TrimSpace(content)

	// URL
	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		return "url"
	}

	// File path
	if strings.HasPrefix(content, "/") || strings.HasPrefix(content, "~/") ||
		strings.HasPrefix(content, "./") || strings.Contains(content, "\\") {
		if !strings.Contains(content, "\n") {
			return "path"
		}
	}

	// JSON
	trimmed := strings.TrimSpace(content)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		return "json"
	}

	// Code-like (has programming syntax)
	codeIndicators := []string{
		"func ", "function ", "def ", "class ", "import ", "package ",
		"const ", "let ", "var ", "return ", "if ", "for ", "while ",
		"=>", "->", "::", "===", "!==", "&&", "||",
	}
	for _, indicator := range codeIndicators {
		if strings.Contains(content, indicator) {
			return "code"
		}
	}

	return "text"
}
