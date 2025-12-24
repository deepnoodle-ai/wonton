// Example: clipstack - Clipboard history manager
//
// A simple TUI showing clipboard history. Navigate with j/k, press Enter to
// copy an item back to the clipboard.
//
// Run with:
//
//	go run ./examples/clipstack
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/tui"
)

type ClipItem struct {
	Content string
	Hash    string
}

type ClipStackApp struct {
	mu          sync.Mutex
	items       []ClipItem
	hashes      map[string]bool
	selected    int
	width       int
	height      int
	lastContent string
	stopPolling chan struct{}
	statusMsg   string
}

func main() {
	app := cli.New("clipstack").
		Description("Clipboard history manager").
		Version("1.0.0")

	app.Main().Run(func(ctx *cli.Context) error {
		if !clipboard.Available() {
			return cli.Error("Clipboard not available on this system")
		}

		tuiApp := &ClipStackApp{
			hashes:      make(map[string]bool),
			stopPolling: make(chan struct{}),
			statusMsg:   "j/k navigate | Enter copy | d delete | q quit",
		}

		if content, err := clipboard.Read(); err == nil && content != "" {
			tuiApp.addItem(content)
		}

		go tuiApp.pollClipboard()
		err := tui.Run(tuiApp)
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
	ticker := time.NewTicker(250 * time.Millisecond)
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
	if strings.TrimSpace(content) == "" {
		return
	}

	hash := hashContent(content)

	// Move duplicate to top
	if app.hashes[hash] {
		for i, item := range app.items {
			if item.Hash == hash {
				app.items = append(app.items[:i], app.items[i+1:]...)
				app.items = append([]ClipItem{item}, app.items...)
				return
			}
		}
		return
	}

	app.items = append([]ClipItem{{Content: content, Hash: hash}}, app.items...)
	app.hashes[hash] = true
	app.lastContent = content

	// Keep max 100 items
	if len(app.items) > 100 {
		delete(app.hashes, app.items[len(app.items)-1].Hash)
		app.items = app.items[:100]
	}
}

func (app *ClipStackApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case tui.KeyEvent:
		if e.Rune == 'q' || e.Key == tui.KeyCtrlC || e.Key == tui.KeyEscape {
			return []tui.Cmd{tui.Quit()}
		}

		app.mu.Lock()
		defer app.mu.Unlock()

		switch e.Key {
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
			}
		case tui.KeyArrowDown:
			if app.selected < len(app.items)-1 {
				app.selected++
			}
		case tui.KeyEnter:
			if app.selected >= 0 && app.selected < len(app.items) {
				if err := clipboard.Write(app.items[app.selected].Content); err != nil {
					app.statusMsg = fmt.Sprintf("Error: %v", err)
				} else {
					app.statusMsg = "Copied to clipboard"
				}
			}
		}

		switch e.Rune {
		case 'j':
			if app.selected < len(app.items)-1 {
				app.selected++
			}
		case 'k':
			if app.selected > 0 {
				app.selected--
			}
		case 'g':
			app.selected = 0
		case 'G':
			app.selected = len(app.items) - 1
		case 'd':
			if app.selected >= 0 && app.selected < len(app.items) {
				delete(app.hashes, app.items[app.selected].Hash)
				app.items = append(app.items[:app.selected], app.items[app.selected+1:]...)
				if app.selected >= len(app.items) && app.selected > 0 {
					app.selected--
				}
				app.statusMsg = "Deleted"
			}
		}
	}
	return nil
}

func (app *ClipStackApp) View() tui.View {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.selected >= len(app.items) {
		app.selected = len(app.items) - 1
	}
	if app.selected < 0 {
		app.selected = 0
	}

	tableHeight := app.height - 4
	if tableHeight < 3 {
		tableHeight = 3
	}

	columns := []tui.TableColumn{
		{Title: "#", Width: 3},
		{Title: "Content"}, // Auto-sized, shrinks to fit
		{Title: "Size", Width: 8},
	}

	rows := make([][]string, len(app.items))
	for i, item := range app.items {
		text := strings.ReplaceAll(item.Content, "\n", " ")
		text = strings.ReplaceAll(text, "\t", " ")

		size := fmt.Sprintf("%dB", len(item.Content))
		if len(item.Content) >= 1024 {
			size = fmt.Sprintf("%.1fK", float64(len(item.Content))/1024)
		}

		rows[i] = []string{fmt.Sprintf("%d", i+1), text, size}
	}

	var content tui.View
	if len(app.items) == 0 {
		content = tui.Stack(
			tui.Spacer(),
			tui.Text("  No clipboard history - copy something to get started").Fg(tui.ColorBrightBlack),
			tui.Spacer(),
		)
	} else {
		content = tui.Table(columns, &app.selected).
			Rows(rows).
			Height(tableHeight).
			FillWidth().
			SelectedBg(tui.ColorCyan).
			SelectedFg(tui.ColorBlack)
	}

	return tui.Stack(
		tui.HeaderBar(fmt.Sprintf("Clipboard History [%d]", len(app.items))).
			Bg(tui.ColorBlue).Fg(tui.ColorWhite),
		tui.Bordered(tui.MinWidth(50, tui.MaxWidth(100, content))).
			Title("Clipboard History").
			Border(&tui.SingleBorder).
			BorderFg(tui.ColorBrightBlack),
		tui.StatusBar(app.statusMsg),
	)
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:8])
}
