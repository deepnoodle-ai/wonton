// Example: browser - Markdown-based terminal web browser
//
// A TUI web browser that renders pages as Markdown with link navigation.
// Great for reading documentation, articles, and text-heavy sites.
//
// Navigation:
//   - Tab: Cycle focus between Content, URL bar, and Links panel
//   - Enter: Follow selected link / submit URL / open link from panel
//   - j/k or Up/Down: Scroll content (or navigate links when Links focused)
//   - b/f or Left/Right: Back/forward in history
//   - Escape: Return to content area
//
// Run with:
//
//	go run ./examples/browser https://example.com
//	go run ./examples/browser https://golang.org/doc/
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/fetch"
	"github.com/deepnoodle-ai/wonton/tui"
)

const (
	linksToShow = 8 // Show 8 links in the panel
)

// PageLink represents a link on the current page
type PageLink struct {
	URL   string
	Text  string
	Index int // 1-based link number
}

// HistoryEntry represents a visited page
type HistoryEntry struct {
	URL   string
	Title string
}

// PageMetadata holds extracted page metadata for display
type PageMetadata struct {
	Title       string
	Description string
	Author      string
	SiteName    string
	PageType    string
	Canonical   string
	WordCount   int
	LinkCount   int
}

// FocusArea represents which UI area has focus
type FocusArea int

const (
	FocusContent FocusArea = iota
	FocusURLBar
	FocusLinks
)

// BrowserApp is the TUI application state
type BrowserApp struct {
	mu sync.Mutex

	// Current page
	currentURL  string
	pageTitle   string
	markdown    string
	links       []PageLink
	metadata    PageMetadata
	loading     bool
	errorMsg    string
	statusMsg   string
	loadingTime time.Time

	// Navigation
	history      []HistoryEntry
	historyIndex int

	// Display
	scrollY int // Scroll position for markdown view
	width   int
	height  int

	// Focus and input
	focus        FocusArea
	urlInput     string // URL being edited
	urlCursor    int    // Cursor position in URL
	selectedLink int    // -1 means no link selected
	linkScroll   int    // Scroll offset in link panel

	// Fetcher
	fetcher *fetch.HTTPFetcher
}

func main() {
	app := cli.New("browser").
		Description("Markdown-based terminal web browser").
		Version("1.0.0")

	app.Main().
		Args("url").
		Flags(
			cli.Int("timeout", "t").
				Default(30).
				Help("Request timeout in seconds"),
		).
		Run(func(ctx *cli.Context) error {
			initialURL := ctx.Arg(0)
			if initialURL == "" {
				return cli.Error("URL is required").
					Hint("Usage: browser https://example.com")
			}

			// Ensure URL has scheme
			if !strings.HasPrefix(initialURL, "http://") && !strings.HasPrefix(initialURL, "https://") {
				initialURL = "https://" + initialURL
			}

			tuiApp := &BrowserApp{
				fetcher: fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
					Timeout: time.Duration(ctx.Int("timeout")) * time.Second,
					Headers: map[string]string{
						"User-Agent": "WontonBrowser/1.0 (terminal)",
					},
				}),
				historyIndex: -1,
				selectedLink: -1,
				focus:        FocusContent,
				urlInput:     initialURL,
			}

			// Start loading the initial page
			go tuiApp.loadPage(initialURL)

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

func (app *BrowserApp) loadPage(pageURL string) {
	app.mu.Lock()
	app.loading = true
	app.loadingTime = time.Now()
	app.errorMsg = ""
	app.statusMsg = "Loading..."
	app.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := app.fetcher.Fetch(ctx, &fetch.Request{
		URL:             pageURL,
		Formats:         []string{"markdown", "links"},
		OnlyMainContent: true,
	})

	app.mu.Lock()
	defer app.mu.Unlock()

	app.loading = false

	if err != nil {
		app.errorMsg = fmt.Sprintf("Error loading page: %v", err)
		app.statusMsg = "Error"
		return
	}

	// Update current page
	app.currentURL = resp.URL
	app.urlInput = resp.URL
	app.urlCursor = len(resp.URL)
	app.pageTitle = resp.Metadata.Title
	if app.pageTitle == "" {
		app.pageTitle = resp.URL
	}
	app.markdown = resp.Markdown

	// Extract metadata
	app.metadata = PageMetadata{
		Title:       resp.Metadata.Title,
		Description: resp.Metadata.Description,
		Author:      resp.Metadata.Author,
		Canonical:   resp.Metadata.Canonical,
		WordCount:   countWords(resp.Markdown),
	}
	// Extract OpenGraph metadata if available
	if resp.Metadata.OpenGraph != nil {
		if app.metadata.SiteName == "" {
			app.metadata.SiteName = resp.Metadata.OpenGraph.SiteName
		}
		if app.metadata.PageType == "" {
			app.metadata.PageType = resp.Metadata.OpenGraph.Type
		}
	}

	// Extract links for the links panel
	app.extractLinks(resp)

	app.scrollY = 0
	app.selectedLink = -1
	app.linkScroll = 0
	app.focus = FocusContent

	// Auto-select first link if available
	if len(app.links) > 0 {
		app.selectedLink = 0
	}

	app.metadata.LinkCount = len(app.links)

	// Add to history
	entry := HistoryEntry{URL: resp.URL, Title: app.pageTitle}
	if app.historyIndex < 0 || app.history[app.historyIndex].URL != resp.URL {
		// Truncate forward history if we navigated from middle
		if app.historyIndex < len(app.history)-1 {
			app.history = app.history[:app.historyIndex+1]
		}
		app.history = append(app.history, entry)
		app.historyIndex = len(app.history) - 1
	}

	elapsed := time.Since(app.loadingTime)
	app.statusMsg = fmt.Sprintf("Loaded in %dms", elapsed.Milliseconds())
}

// extractLinks extracts links from the markdown for the links panel
func (app *BrowserApp) extractLinks(resp *fetch.Response) {
	baseURL, _ := url.Parse(resp.URL)
	app.links = nil

	// Regex to find markdown links
	linkRegex := regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	linkIndex := 1

	matches := linkRegex.FindAllStringSubmatch(app.markdown, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		linkText := match[1]
		linkURL := match[2]

		// Resolve relative URL
		if baseURL != nil {
			if resolved, err := baseURL.Parse(linkURL); err == nil {
				linkURL = resolved.String()
			}
		}

		// Skip non-navigable links
		if strings.HasPrefix(linkURL, "javascript:") ||
			strings.HasPrefix(linkURL, "mailto:") ||
			strings.HasPrefix(linkURL, "#") {
			continue
		}

		app.links = append(app.links, PageLink{
			URL:   linkURL,
			Text:  linkText,
			Index: linkIndex,
		})
		linkIndex++
	}
}

func countWords(text string) int {
	count := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

func (app *BrowserApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.mu.Lock()
		app.width = e.Width
		app.height = e.Height
		app.mu.Unlock()

	case tui.KeyEvent:
		app.mu.Lock()
		defer app.mu.Unlock()

		// Global quit
		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

		// Handle input based on focus area
		switch app.focus {
		case FocusURLBar:
			return app.handleURLBarInput(e)
		case FocusLinks:
			return app.handleLinksInput(e)
		default:
			return app.handleContentInput(e)
		}
	}

	return nil
}

func (app *BrowserApp) handleContentInput(e tui.KeyEvent) []tui.Cmd {
	// Quit
	if e.Rune == 'q' || e.Rune == 'Q' {
		return []tui.Cmd{tui.Quit()}
	}

	// Tab cycles focus: URL -> Content -> Links (top to bottom)
	if e.Key == tui.KeyTab {
		if e.Shift {
			app.focus = FocusURLBar
			app.urlCursor = len(app.urlInput)
		} else {
			app.focus = FocusLinks
		}
		return nil
	}

	// Page size for scrolling
	pageSize := app.contentHeight()

	// Key handling - markdown view clamps scrollY automatically
	switch e.Key {
	case tui.KeyEnter:
		if app.selectedLink >= 0 && app.selectedLink < len(app.links) {
			linkURL := app.links[app.selectedLink].URL
			go app.loadPage(linkURL)
		}
		return nil

	case tui.KeyArrowUp:
		app.scrollY--
	case tui.KeyArrowDown:
		app.scrollY++
	case tui.KeyPageUp, tui.KeyCtrlB:
		app.scrollY -= pageSize
	case tui.KeyPageDown, tui.KeyCtrlF:
		app.scrollY += pageSize
	case tui.KeyHome:
		app.scrollY = 0
	case tui.KeyEnd:
		app.scrollY = 999999 // Will be clamped by markdown view
	case tui.KeyArrowLeft:
		app.goBack()
	case tui.KeyArrowRight:
		app.goForward()
	}

	// Rune keys
	switch e.Rune {
	case 'j':
		app.scrollY++
	case 'k':
		app.scrollY--
	case ' ':
		app.scrollY += pageSize
	case 'G':
		app.scrollY = 999999 // Will be clamped
	case 'g':
		app.scrollY = 0
	case 'b':
		app.goBack()
	case 'f':
		app.goForward()
	case 'r':
		if app.currentURL != "" {
			go app.loadPage(app.currentURL)
		}
	case 'c':
		clipboard.Write(app.currentURL)
		app.statusMsg = "URL copied"
	case 'C':
		if app.selectedLink >= 0 && app.selectedLink < len(app.links) {
			clipboard.Write(app.links[app.selectedLink].URL)
			app.statusMsg = "Link URL copied"
		}
	case 'l':
		// Quick switch to links panel
		app.focus = FocusLinks
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		linkNum := int(e.Rune - '0')
		for i, link := range app.links {
			if link.Index == linkNum {
				app.selectedLink = i
				app.updateLinkScroll()
				break
			}
		}
	}

	return nil
}

func (app *BrowserApp) handleURLBarInput(e tui.KeyEvent) []tui.Cmd {
	switch e.Key {
	case tui.KeyEscape:
		app.focus = FocusContent
		app.urlInput = app.currentURL
		return nil

	case tui.KeyTab:
		if e.Shift {
			app.focus = FocusLinks // Wrap to bottom
		} else {
			app.focus = FocusContent // Go down
		}
		return nil

	case tui.KeyEnter:
		app.focus = FocusContent
		urlToLoad := app.urlInput
		if !strings.HasPrefix(urlToLoad, "http://") && !strings.HasPrefix(urlToLoad, "https://") {
			urlToLoad = "https://" + urlToLoad
		}
		go app.loadPage(urlToLoad)
		return nil

	case tui.KeyBackspace:
		if app.urlCursor > 0 {
			app.urlInput = app.urlInput[:app.urlCursor-1] + app.urlInput[app.urlCursor:]
			app.urlCursor--
		}
		return nil

	case tui.KeyDelete:
		if app.urlCursor < len(app.urlInput) {
			app.urlInput = app.urlInput[:app.urlCursor] + app.urlInput[app.urlCursor+1:]
		}
		return nil

	case tui.KeyArrowLeft:
		if app.urlCursor > 0 {
			app.urlCursor--
		}
		return nil

	case tui.KeyArrowRight:
		if app.urlCursor < len(app.urlInput) {
			app.urlCursor++
		}
		return nil

	case tui.KeyHome, tui.KeyCtrlA:
		app.urlCursor = 0
		return nil

	case tui.KeyEnd, tui.KeyCtrlE:
		app.urlCursor = len(app.urlInput)
		return nil

	case tui.KeyCtrlU:
		app.urlInput = app.urlInput[app.urlCursor:]
		app.urlCursor = 0
		return nil

	case tui.KeyCtrlK:
		app.urlInput = app.urlInput[:app.urlCursor]
		return nil
	}

	// Handle printable characters
	if e.Rune != 0 && e.Key == tui.KeyUnknown {
		app.urlInput = app.urlInput[:app.urlCursor] + string(e.Rune) + app.urlInput[app.urlCursor:]
		app.urlCursor++
	}

	return nil
}

func (app *BrowserApp) handleLinksInput(e tui.KeyEvent) []tui.Cmd {
	switch e.Key {
	case tui.KeyEscape:
		app.focus = FocusContent
		return nil

	case tui.KeyTab:
		if e.Shift {
			app.focus = FocusContent // Go up
		} else {
			app.focus = FocusURLBar // Wrap to top
			app.urlCursor = len(app.urlInput)
		}
		return nil

	case tui.KeyEnter:
		if app.selectedLink >= 0 && app.selectedLink < len(app.links) {
			linkURL := app.links[app.selectedLink].URL
			go app.loadPage(linkURL)
		}
		return nil

	case tui.KeyArrowUp:
		app.prevLink()
		return nil

	case tui.KeyArrowDown:
		app.nextLink()
		return nil

	case tui.KeyPageUp:
		// Jump up by visible links
		for i := 0; i < linksToShow; i++ {
			app.prevLink()
		}
		return nil

	case tui.KeyPageDown:
		// Jump down by visible links
		for i := 0; i < linksToShow; i++ {
			app.nextLink()
		}
		return nil

	case tui.KeyHome:
		if len(app.links) > 0 {
			app.selectedLink = 0
			app.linkScroll = 0
		}
		return nil

	case tui.KeyEnd:
		if len(app.links) > 0 {
			app.selectedLink = len(app.links) - 1
			app.updateLinkScroll()
		}
		return nil
	}

	// Rune keys
	switch e.Rune {
	case 'j':
		app.nextLink()
	case 'k':
		app.prevLink()
	case 'q', 'Q':
		return []tui.Cmd{tui.Quit()}
	case 'c':
		if app.selectedLink >= 0 && app.selectedLink < len(app.links) {
			clipboard.Write(app.links[app.selectedLink].URL)
			app.statusMsg = "Link URL copied"
		}
	}

	return nil
}

// contentHeight returns available height for content
func (app *BrowserApp) contentHeight() int {
	// Reserve: header(3) + url bar(3) + metadata(5) + content border(2) + link panel(linksToShow+4) + footer(2)
	h := app.height - (3 + 3 + 5 + 2 + linksToShow + 4 + 2)
	if h < 5 {
		h = 5
	}
	return h
}

// sectionWidth returns the width for bordered sections based on terminal width
func (app *BrowserApp) sectionWidth() int {
	if app.width > 120 {
		return 120
	}
	if app.width > 80 {
		return 80
	}
	return app.width - 2 // Leave some margin
}

func (app *BrowserApp) nextLink() {
	if len(app.links) == 0 {
		return
	}
	app.selectedLink++
	if app.selectedLink >= len(app.links) {
		app.selectedLink = 0
	}
	app.updateLinkScroll()
}

func (app *BrowserApp) prevLink() {
	if len(app.links) == 0 {
		return
	}
	app.selectedLink--
	if app.selectedLink < 0 {
		app.selectedLink = len(app.links) - 1
	}
	app.updateLinkScroll()
}

func (app *BrowserApp) updateLinkScroll() {
	// Keep selected link visible in link panel
	if app.selectedLink < app.linkScroll {
		app.linkScroll = app.selectedLink
	}
	if app.selectedLink >= app.linkScroll+linksToShow {
		app.linkScroll = app.selectedLink - linksToShow + 1
	}
}

func (app *BrowserApp) goBack() {
	if app.historyIndex > 0 {
		app.historyIndex--
		go app.loadPage(app.history[app.historyIndex].URL)
	}
}

func (app *BrowserApp) goForward() {
	if app.historyIndex < len(app.history)-1 {
		app.historyIndex++
		go app.loadPage(app.history[app.historyIndex].URL)
	}
}

func (app *BrowserApp) View() tui.View {
	app.mu.Lock()
	defer app.mu.Unlock()

	// === HEADER ===
	header := app.buildHeader()

	// === URL BAR ===
	urlBar := app.buildURLBar()

	// === METADATA ===
	metadataView := app.buildMetadataView()

	// === CONTENT ===
	contentView := app.buildContentView()

	// === LINK PANEL ===
	linkPanel := app.buildLinkPanel()

	// === FOOTER ===
	footer := app.buildFooter()

	return tui.Stack(
		header,
		urlBar,
		metadataView,
		contentView,
		linkPanel,
		footer,
	)
}

func (app *BrowserApp) buildHeader() tui.View {
	w := app.sectionWidth()

	// Create a visually appealing header with gradient colors
	var titleParts []tui.View

	// Browser name with wave animation
	browserName := tui.Text(" WONTON BROWSER ").Animate(tui.Wave(8,
		tui.NewRGB(100, 180, 255),
		tui.NewRGB(150, 220, 255),
		tui.NewRGB(100, 180, 255),
	))

	// Navigation indicator
	var navText string
	if len(app.history) > 1 {
		navText = fmt.Sprintf(" [%d/%d] ", app.historyIndex+1, len(app.history))
	}

	// Loading indicator
	var statusIndicator tui.View
	if app.loading {
		statusIndicator = tui.Text(" Loading... ").Animate(tui.Pulse(tui.NewRGB(255, 200, 100), 8))
	} else if app.errorMsg != "" {
		statusIndicator = tui.Text(" Error ").FgRGB(255, 100, 100)
	} else {
		statusIndicator = tui.Text(" Ready ").FgRGB(100, 255, 150)
	}

	titleParts = append(titleParts,
		browserName,
		tui.Text("%s", navText).FgRGB(200, 200, 200),
		statusIndicator,
	)

	headerContent := tui.Group(titleParts...)

	return tui.Width(w, tui.Stack(
		tui.Text("").BgRGB(30, 60, 90), // Top accent line
		tui.Background(' ', tui.NewStyle().WithBgRGB(tui.NewRGB(20, 40, 60)),
			tui.Group(
				headerContent,
				tui.Spacer(),
			),
		),
		tui.Text("").BgRGB(30, 60, 90), // Bottom accent line
	))
}

func (app *BrowserApp) buildURLBar() tui.View {
	w := app.sectionWidth()

	var urlDisplay string
	var fg tui.Color

	if app.focus == FocusURLBar {
		// Show cursor in URL
		before := app.urlInput[:app.urlCursor]
		after := app.urlInput[app.urlCursor:]
		urlDisplay = before + "█" + after
		fg = tui.ColorWhite
	} else {
		urlDisplay = app.urlInput
		if urlDisplay == "" {
			urlDisplay = "(enter URL)"
		}
		fg = tui.ColorBrightBlack
	}

	title := "URL"
	if app.focus == FocusURLBar {
		title = "URL (editing - Enter to go, Esc to cancel)"
	}

	// Border color based on focus (cyan for all focused sections)
	borderColor := tui.ColorBrightBlack
	if app.focus == FocusURLBar {
		borderColor = tui.ColorCyan
	}

	return tui.Width(w, tui.Bordered(
		tui.Text(" %s", urlDisplay).Fg(fg),
	).Border(&tui.RoundedBorder).Title(title).BorderFg(borderColor))
}

func (app *BrowserApp) buildMetadataView() tui.View {
	w := app.sectionWidth()

	if app.loading {
		return tui.Width(w, tui.Bordered(
			tui.Text(" Loading %s...", app.currentURL).Fg(tui.ColorYellow),
		).Border(&tui.RoundedBorder).Title("Page").BorderFg(tui.ColorYellow))
	}

	if app.errorMsg != "" {
		return tui.Width(w, tui.Bordered(
			tui.Text(" %s", app.errorMsg).Fg(tui.ColorRed),
		).Border(&tui.RoundedBorder).Title("Error").BorderFg(tui.ColorRed))
	}

	var rows []tui.View
	labelColor := tui.NewRGB(100, 120, 140)
	valueColor := tui.NewRGB(200, 210, 220)

	// Helper to add a row
	addRow := func(label, value string, maxLen int) {
		if value == "" {
			return
		}
		if maxLen > 0 && len(value) > maxLen {
			value = value[:maxLen-3] + "..."
		}
		rows = append(rows, tui.Group(
			tui.Text(" %s: ", label).FgRGB(labelColor.R, labelColor.G, labelColor.B),
			tui.Text("%s", value).FgRGB(valueColor.R, valueColor.G, valueColor.B),
		))
	}

	maxValLen := w - 20
	if maxValLen < 30 {
		maxValLen = 30
	}

	// Title (special - bold and prominent with gradient)
	title := app.metadata.Title
	if title == "" {
		title = "(untitled)"
	}
	if len(title) > maxValLen {
		title = title[:maxValLen-3] + "..."
	}
	rows = append(rows, tui.Text(" %s", title).Bold().Animate(tui.Wave(6,
		tui.NewRGB(180, 200, 255),
		tui.NewRGB(255, 220, 180),
		tui.NewRGB(180, 200, 255),
	)))

	// Site name
	if app.metadata.SiteName != "" {
		addRow("Site", app.metadata.SiteName, maxValLen)
	}

	// Description
	desc := app.metadata.Description
	if desc != "" {
		if len(desc) > maxValLen {
			desc = desc[:maxValLen-3] + "..."
		}
		rows = append(rows, tui.Text(" %s", desc).FgRGB(150, 150, 170).Italic())
	}

	// Stats line
	var statParts []string
	statParts = append(statParts, fmt.Sprintf("%d words", app.metadata.WordCount))
	statParts = append(statParts, fmt.Sprintf("%d links", app.metadata.LinkCount))
	if app.metadata.Author != "" {
		statParts = append(statParts, fmt.Sprintf("by %s", app.metadata.Author))
	}
	if app.metadata.PageType != "" {
		statParts = append(statParts, app.metadata.PageType)
	}
	rows = append(rows, tui.Text(" %s", strings.Join(statParts, " • ")).FgRGB(120, 140, 160))

	return tui.Width(w, tui.Bordered(
		tui.Stack(rows...),
	).Border(&tui.RoundedBorder).Title("Page Info").BorderFg(tui.ColorBrightBlack))
}

func (app *BrowserApp) buildContentView() tui.View {
	w := app.sectionWidth()
	h := app.contentHeight()

	// Border color based on focus
	borderColor := tui.ColorBrightBlack
	if app.focus == FocusContent {
		borderColor = tui.ColorCyan
	}

	title := "Content"
	if app.focus == FocusContent {
		title = "Content (focused)"
	}

	var content tui.View

	if app.loading {
		content = tui.Stack(
			tui.Spacer().MinHeight(2),
			tui.Text("Loading...").Animate(tui.Pulse(tui.NewRGB(255, 200, 100), 10)),
		)
	} else if app.errorMsg != "" {
		content = tui.Stack(
			tui.Spacer().MinHeight(2),
			tui.Text("Error").Fg(tui.ColorRed).Bold(),
			tui.Text("%s", app.errorMsg).Fg(tui.ColorWhite),
		)
	} else if app.markdown == "" {
		content = tui.Text("No content").Fg(tui.ColorBrightBlack)
	} else {
		// Use the markdown view component with proper rendering
		content = tui.Markdown(app.markdown, &app.scrollY).
			MaxWidth(w - 4). // Account for border and padding
			Height(h)
	}

	return tui.Width(w, tui.Bordered(content).
		Border(&tui.RoundedBorder).
		Title(title).
		BorderFg(borderColor))
}

func (app *BrowserApp) buildLinkPanel() tui.View {
	w := app.sectionWidth()

	// Border color based on focus (cyan for all focused sections)
	borderColor := tui.ColorBrightBlack
	if app.focus == FocusLinks {
		borderColor = tui.ColorCyan
	}

	if len(app.links) == 0 {
		return tui.Width(w, tui.Bordered(
			tui.Text(" No links on this page").Fg(tui.ColorBrightBlack),
		).Border(&tui.RoundedBorder).Title("Links").BorderFg(borderColor))
	}

	var linkViews []tui.View

	// Show links around the selected one
	start := app.linkScroll
	end := start + linksToShow
	if end > len(app.links) {
		end = len(app.links)
	}

	// Calculate widths for two-column layout
	// Left column: link number and text, Right column: URL
	leftColWidth := (w - 6) / 2
	rightColWidth := w - 6 - leftColWidth - 3 // 3 for separator
	if leftColWidth < 20 {
		leftColWidth = 20
	}
	if rightColWidth < 20 {
		rightColWidth = 20
	}

	for i := start; i < end; i++ {
		link := app.links[i]

		// Prepare link text
		text := link.Text
		maxTextLen := leftColWidth - 8 // Account for " > [N] " prefix
		if maxTextLen < 10 {
			maxTextLen = 10
		}
		if len(text) > maxTextLen {
			text = text[:maxTextLen-3] + "..."
		}
		if text == "" {
			text = "(no text)"
		}

		// Prepare URL
		linkURL := link.URL
		if len(linkURL) > rightColWidth {
			linkURL = linkURL[:rightColWidth-3] + "..."
		}

		var leftView, rightView tui.View
		if i == app.selectedLink {
			// Highlighted selected link
			leftView = tui.Text(" > [%d] %s", link.Index, text).FgRGB(255, 220, 100).Bold()
			rightView = tui.Text("%s", linkURL).FgRGB(180, 160, 80)
		} else {
			leftView = tui.Text("   [%d] %s", link.Index, text).FgRGB(150, 180, 220)
			rightView = tui.Text("%s", linkURL).FgRGB(90, 100, 110)
		}

		linkViews = append(linkViews, tui.Group(
			tui.Width(leftColWidth, leftView),
			tui.Text(" → ").FgRGB(70, 80, 90),
			rightView,
		))
	}

	// Navigation hint with highlighted numbers
	linkViews = append(linkViews, tui.Group(
		tui.Text(" Link ").FgRGB(80, 100, 120),
		tui.Text("%d", app.selectedLink+1).FgRGB(255, 200, 100).Bold(),
		tui.Text(" of ").FgRGB(80, 100, 120),
		tui.Text("%d", len(app.links)).FgRGB(200, 220, 255).Bold(),
	))

	title := "Links"
	if app.focus == FocusLinks {
		title = "Links (focused - j/k or arrows to navigate, Enter to follow)"
	}

	return tui.Width(w, tui.Bordered(
		tui.Stack(linkViews...),
	).Border(&tui.RoundedBorder).Title(title).BorderFg(borderColor))
}

func (app *BrowserApp) buildFooter() tui.View {
	w := app.sectionWidth()

	// Build context-sensitive help
	var helpText string
	switch app.focus {
	case FocusURLBar:
		helpText = "Enter: Navigate | Esc: Cancel | Tab: Next area"
	case FocusLinks:
		helpText = "j/k: Navigate | Enter: Follow | c: Copy URL | Tab: Next area | Esc: Content"
	default:
		helpText = "Tab: Switch focus | j/k: Scroll | Enter: Follow link | l: Links | b/f: History | q: Quit"
	}

	// Focus indicator
	var focusIndicator string
	switch app.focus {
	case FocusURLBar:
		focusIndicator = "URL"
	case FocusLinks:
		focusIndicator = "LINKS"
	default:
		focusIndicator = "CONTENT"
	}

	// Status message
	statusText := app.statusMsg

	return tui.Width(w, tui.Stack(
		tui.Background(' ', tui.NewStyle().WithBgRGB(tui.NewRGB(25, 35, 45)),
			tui.Group(
				tui.Text(" %s ", focusIndicator).BgRGB(60, 80, 100).FgRGB(200, 220, 255).Bold(),
				tui.Text(" %s", helpText).FgRGB(140, 160, 180),
				tui.Spacer(),
				tui.Text("%s ", statusText).FgRGB(100, 120, 140),
			),
		),
	))
}
