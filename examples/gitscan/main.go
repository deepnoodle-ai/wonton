// Example: gitscan - Interactive Git browser
//
// A TUI for browsing git commit history and viewing diffs. Similar to tig but
// simpler and focused on quick navigation and diff viewing.
//
// Run with:
//
//	go run ./examples/gitscan                    # Browse current repo
//	go run ./examples/gitscan /path/to/repo     # Browse specific repo
//	go run ./examples/gitscan --limit 50        # Show last 50 commits
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/git"
	"github.com/deepnoodle-ai/wonton/humanize"
	"github.com/deepnoodle-ai/wonton/tui"
	"github.com/deepnoodle-ai/wonton/unidiff"
)

// ViewMode determines what the main panel shows
type ViewMode int

const (
	ViewCommits ViewMode = iota
	ViewDiff
	ViewFiles
)

// GitScanApp is the TUI application
type GitScanApp struct {
	mu sync.Mutex

	// Repository
	repo       *git.Repository
	repoPath   string
	branch     string
	status     *git.Status

	// Commits
	commits        []git.Commit
	selectedCommit int
	commitScroll   int

	// Diff view
	diff        *git.Diff
	parsedDiff  *unidiff.Diff
	diffLines   []diffLine
	selectedLine int
	diffScroll   int

	// Files view
	files         []git.DiffFile
	selectedFile  int
	fileScroll    int

	// View state
	mode      ViewMode
	width     int
	height    int
	statusMsg string
}

// diffLine represents a single line in the diff view
type diffLine struct {
	Text     string
	Type     string // header, hunk, add, remove, context
	File     string
	LineNum  int
}

func main() {
	app := cli.New("gitscan").
		Description("Interactive Git browser").
		Version("1.0.0")

	app.Main().
		Args("path?").
		Flags(
			cli.Int("limit", "n").
				Default(100).
				Help("Maximum commits to load"),
			cli.String("author", "a").
				Help("Filter by author"),
			cli.String("grep", "g").
				Help("Filter by commit message"),
		).
		Run(func(ctx *cli.Context) error {
			path := ctx.Arg(0)
			if path == "" {
				path = "."
			}

			repo, err := git.Open(path)
			if err != nil {
				return cli.Error("Not a git repository").
					Hint("Run from within a git repository or specify a path")
			}

			tuiApp := &GitScanApp{
				repo:      repo,
				repoPath:  repo.Path,
				statusMsg: "↑↓/jk navigate | Space/b page | Enter diff | f files | c copy | q quit",
			}

			// Load initial data
			bgCtx := context.Background()

			// Get branch
			branch, _ := repo.CurrentBranch(bgCtx)
			tuiApp.branch = branch

			// Get status
			status, _ := repo.Status(bgCtx)
			tuiApp.status = status

			// Load commits
			opts := git.LogOptions{
				Limit:  ctx.Int("limit"),
				Author: ctx.String("author"),
				Grep:   ctx.String("grep"),
			}
			commits, err := repo.Log(bgCtx, opts)
			if err != nil {
				return fmt.Errorf("failed to load commits: %w", err)
			}
			tuiApp.commits = commits

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

func (app *GitScanApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case tui.KeyEvent:
		// Global quit
		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

		app.mu.Lock()
		defer app.mu.Unlock()

		// Mode-specific handling
		switch app.mode {
		case ViewCommits:
			return app.handleCommitsKey(e)
		case ViewDiff:
			return app.handleDiffKey(e)
		case ViewFiles:
			return app.handleFilesKey(e)
		}
	}

	return nil
}

func (app *GitScanApp) handleCommitsKey(e tui.KeyEvent) []tui.Cmd {
	// Quit
	if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape {
		return []tui.Cmd{tui.Quit()}
	}

	// Calculate page size
	listHeight := app.height - 6
	if listHeight < 5 {
		listHeight = 5
	}

	switch e.Key {
	case tui.KeyArrowUp:
		if app.selectedCommit > 0 {
			app.selectedCommit--
		}
	case tui.KeyArrowDown:
		if app.selectedCommit < len(app.commits)-1 {
			app.selectedCommit++
		}
	case tui.KeyHome:
		app.selectedCommit = 0
	case tui.KeyEnd:
		app.selectedCommit = len(app.commits) - 1
	case tui.KeyPageUp, tui.KeyCtrlB:
		app.selectedCommit -= listHeight
		if app.selectedCommit < 0 {
			app.selectedCommit = 0
		}
	case tui.KeyPageDown, tui.KeyCtrlF:
		app.selectedCommit += listHeight
		if app.selectedCommit >= len(app.commits) {
			app.selectedCommit = len(app.commits) - 1
		}
	case tui.KeyCtrlD:
		// Half page down
		app.selectedCommit += listHeight / 2
		if app.selectedCommit >= len(app.commits) {
			app.selectedCommit = len(app.commits) - 1
		}
	case tui.KeyCtrlU:
		// Half page up
		app.selectedCommit -= listHeight / 2
		if app.selectedCommit < 0 {
			app.selectedCommit = 0
		}
	case tui.KeyEnter:
		app.loadDiff()
		app.mode = ViewDiff
	}

	// Less-compatible rune keys
	switch e.Rune {
	case 'j':
		if app.selectedCommit < len(app.commits)-1 {
			app.selectedCommit++
		}
	case 'k':
		if app.selectedCommit > 0 {
			app.selectedCommit--
		}
	case ' ':
		// Page down (note: 'f' is used for files view)
		app.selectedCommit += listHeight
		if app.selectedCommit >= len(app.commits) {
			app.selectedCommit = len(app.commits) - 1
		}
	case 'b':
		// Page up
		app.selectedCommit -= listHeight
		if app.selectedCommit < 0 {
			app.selectedCommit = 0
		}
	case 'g':
		app.selectedCommit = 0
	case 'G':
		app.selectedCommit = len(app.commits) - 1
	}

	switch e.Rune {
	case 'f', 'F':
		app.loadDiff()
		app.mode = ViewFiles
	case 'c', 'C':
		if app.selectedCommit >= 0 && app.selectedCommit < len(app.commits) {
			hash := app.commits[app.selectedCommit].Hash
			if err := clipboard.Write(hash); err == nil {
				app.statusMsg = fmt.Sprintf("✓ Copied %s", hash[:8])
			} else {
				app.statusMsg = fmt.Sprintf("Error: %v", err)
			}
		}
	case 's', 'S':
		if app.selectedCommit >= 0 && app.selectedCommit < len(app.commits) {
			hash := app.commits[app.selectedCommit].ShortHash
			if err := clipboard.Write(hash); err == nil {
				app.statusMsg = fmt.Sprintf("✓ Copied %s", hash)
			}
		}
	}

	return nil
}

func (app *GitScanApp) handleDiffKey(e tui.KeyEvent) []tui.Cmd {
	// Back to commits
	if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape {
		app.mode = ViewCommits
		app.statusMsg = "↑↓/jk navigate | Space/b page | f files | c copy hash | q quit"
		return nil
	}

	// Calculate page size
	viewHeight := app.height - 6
	if viewHeight < 5 {
		viewHeight = 5
	}

	switch e.Key {
	case tui.KeyArrowUp:
		if app.selectedLine > 0 {
			app.selectedLine--
		}
	case tui.KeyArrowDown:
		if app.selectedLine < len(app.diffLines)-1 {
			app.selectedLine++
		}
	case tui.KeyHome:
		app.selectedLine = 0
	case tui.KeyEnd:
		app.selectedLine = len(app.diffLines) - 1
	case tui.KeyPageUp, tui.KeyCtrlB:
		app.selectedLine -= viewHeight
		if app.selectedLine < 0 {
			app.selectedLine = 0
		}
	case tui.KeyPageDown, tui.KeyCtrlF:
		app.selectedLine += viewHeight
		if app.selectedLine >= len(app.diffLines) {
			app.selectedLine = len(app.diffLines) - 1
		}
	case tui.KeyCtrlD:
		// Half page down
		app.selectedLine += viewHeight / 2
		if app.selectedLine >= len(app.diffLines) {
			app.selectedLine = len(app.diffLines) - 1
		}
	case tui.KeyCtrlU:
		// Half page up
		app.selectedLine -= viewHeight / 2
		if app.selectedLine < 0 {
			app.selectedLine = 0
		}
	}

	// Less-compatible rune keys
	switch e.Rune {
	case 'j':
		if app.selectedLine < len(app.diffLines)-1 {
			app.selectedLine++
		}
	case 'k':
		if app.selectedLine > 0 {
			app.selectedLine--
		}
	case ' ':
		// Page down
		app.selectedLine += viewHeight
		if app.selectedLine >= len(app.diffLines) {
			app.selectedLine = len(app.diffLines) - 1
		}
	case 'b':
		// Page up
		app.selectedLine -= viewHeight
		if app.selectedLine < 0 {
			app.selectedLine = 0
		}
	case 'g':
		app.selectedLine = 0
	case 'G':
		app.selectedLine = len(app.diffLines) - 1
	}

	switch e.Rune {
	case 'f', 'F':
		app.mode = ViewFiles
	case 'c', 'C':
		// Copy selected line
		if app.selectedLine >= 0 && app.selectedLine < len(app.diffLines) {
			line := app.diffLines[app.selectedLine]
			if err := clipboard.Write(line.Text); err == nil {
				app.statusMsg = "✓ Copied line"
			}
		}
	}

	return nil
}

func (app *GitScanApp) handleFilesKey(e tui.KeyEvent) []tui.Cmd {
	// Back
	if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape {
		app.mode = ViewCommits
		app.statusMsg = "↑↓/jk navigate | Space/b page | f files | c copy hash | q quit"
		return nil
	}

	// Calculate page size
	listHeight := app.height - 8
	if listHeight < 5 {
		listHeight = 5
	}

	switch e.Key {
	case tui.KeyArrowUp:
		if app.selectedFile > 0 {
			app.selectedFile--
		}
	case tui.KeyArrowDown:
		if app.selectedFile < len(app.files)-1 {
			app.selectedFile++
		}
	case tui.KeyHome:
		app.selectedFile = 0
	case tui.KeyEnd:
		app.selectedFile = len(app.files) - 1
	case tui.KeyPageUp, tui.KeyCtrlB:
		app.selectedFile -= listHeight
		if app.selectedFile < 0 {
			app.selectedFile = 0
		}
	case tui.KeyPageDown, tui.KeyCtrlF:
		app.selectedFile += listHeight
		if app.selectedFile >= len(app.files) {
			app.selectedFile = len(app.files) - 1
		}
	case tui.KeyCtrlD:
		app.selectedFile += listHeight / 2
		if app.selectedFile >= len(app.files) {
			app.selectedFile = len(app.files) - 1
		}
	case tui.KeyCtrlU:
		app.selectedFile -= listHeight / 2
		if app.selectedFile < 0 {
			app.selectedFile = 0
		}
	case tui.KeyEnter:
		// Show diff for this file
		app.mode = ViewDiff
		// Jump to this file in diff
		if app.selectedFile >= 0 && app.selectedFile < len(app.files) {
			targetFile := app.files[app.selectedFile].Path
			for i, line := range app.diffLines {
				if line.File == targetFile && line.Type == "header" {
					app.selectedLine = i
					break
				}
			}
		}
	}

	// Less-compatible rune keys
	switch e.Rune {
	case 'j':
		if app.selectedFile < len(app.files)-1 {
			app.selectedFile++
		}
	case 'k':
		if app.selectedFile > 0 {
			app.selectedFile--
		}
	case ' ':
		// Page down
		app.selectedFile += listHeight
		if app.selectedFile >= len(app.files) {
			app.selectedFile = len(app.files) - 1
		}
	case 'b':
		// Page up
		app.selectedFile -= listHeight
		if app.selectedFile < 0 {
			app.selectedFile = 0
		}
	case 'g':
		app.selectedFile = 0
	case 'G':
		app.selectedFile = len(app.files) - 1
	}

	switch e.Rune {
	case 'c', 'C':
		// Copy file path
		if app.selectedFile >= 0 && app.selectedFile < len(app.files) {
			path := app.files[app.selectedFile].Path
			if err := clipboard.Write(path); err == nil {
				app.statusMsg = fmt.Sprintf("✓ Copied %s", path)
			}
		}
	case 'd', 'D':
		app.mode = ViewDiff
	}

	return nil
}

func (app *GitScanApp) loadDiff() {
	if app.selectedCommit < 0 || app.selectedCommit >= len(app.commits) {
		return
	}

	commit := app.commits[app.selectedCommit]
	ctx := context.Background()

	// Determine the comparison range
	var opts git.DiffOptions
	if len(commit.ParentHashes) > 0 {
		opts = git.DiffOptions{
			From:         commit.ParentHashes[0],
			To:           commit.Hash,
			IncludePatch: true,
		}
	} else {
		// Root commit - compare against empty tree
		opts = git.DiffOptions{
			Ref:          commit.Hash,
			IncludePatch: true,
		}
	}

	diff, err := app.repo.Diff(ctx, opts)
	if err != nil {
		app.statusMsg = fmt.Sprintf("Error loading diff: %v", err)
		return
	}

	app.diff = diff
	app.files = diff.Files
	app.selectedFile = 0
	app.selectedLine = 0
	app.diffScroll = 0

	// Build diff lines for display
	app.diffLines = nil
	for _, file := range diff.Files {
		// File header
		app.diffLines = append(app.diffLines, diffLine{
			Text: fmt.Sprintf("─── %s ───", file.Path),
			Type: "header",
			File: file.Path,
		})

		if file.Patch == "" {
			app.diffLines = append(app.diffLines, diffLine{
				Text: "(binary file or no changes)",
				Type: "context",
				File: file.Path,
			})
			continue
		}

		// Parse the patch
		parsed, err := unidiff.Parse(file.Patch)
		if err != nil {
			app.diffLines = append(app.diffLines, diffLine{
				Text: fmt.Sprintf("(error parsing diff: %v)", err),
				Type: "context",
				File: file.Path,
			})
			continue
		}

		for _, pf := range parsed.Files {
			for _, hunk := range pf.Hunks {
				// Hunk header
				app.diffLines = append(app.diffLines, diffLine{
					Text: hunk.Header,
					Type: "hunk",
					File: file.Path,
				})

				for _, line := range hunk.Lines {
					lineType := "context"
					switch line.Type {
					case unidiff.LineAdded:
						lineType = "add"
					case unidiff.LineRemoved:
						lineType = "remove"
					}
					app.diffLines = append(app.diffLines, diffLine{
						Text:    line.Content,
						Type:    lineType,
						File:    file.Path,
						LineNum: line.NewLineNum,
					})
				}
			}
		}
	}

	app.statusMsg = "↑↓ scroll | f files | c copy | q back"
}

func (app *GitScanApp) View() tui.View {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Header
	branchDisplay := app.branch
	if branchDisplay == "" {
		branchDisplay = "detached"
	}

	statusMark := ""
	if app.status != nil && !app.status.IsClean {
		statusMark = "*"
	}

	header := tui.HeaderBar(fmt.Sprintf("gitscan  ⎇ %s%s  [%d commits]",
		branchDisplay, statusMark, len(app.commits))).
		Bg(tui.ColorBlue).
		Fg(tui.ColorWhite)

	// Main content based on mode
	var mainContent tui.View
	switch app.mode {
	case ViewCommits:
		mainContent = app.viewCommits()
	case ViewDiff:
		mainContent = app.viewDiff()
	case ViewFiles:
		mainContent = app.viewFiles()
	}

	return tui.Stack(
		header,
		mainContent,
		tui.StatusBar(app.statusMsg),
	)
}

func (app *GitScanApp) viewCommits() tui.View {
	if len(app.commits) == 0 {
		return tui.Stack(
			tui.Spacer(),
			tui.Text("No commits found").Fg(tui.ColorYellow),
			tui.Spacer(),
		)
	}

	// Calculate visible range
	listHeight := app.height - 6
	if listHeight < 5 {
		listHeight = 5
	}

	start := app.commitScroll
	end := start + listHeight
	if end > len(app.commits) {
		end = len(app.commits)
	}

	// Adjust scroll
	if app.selectedCommit < start {
		app.commitScroll = app.selectedCommit
		start = app.commitScroll
		end = start + listHeight
		if end > len(app.commits) {
			end = len(app.commits)
		}
	} else if app.selectedCommit >= end {
		app.commitScroll = app.selectedCommit - listHeight + 1
		if app.commitScroll < 0 {
			app.commitScroll = 0
		}
		start = app.commitScroll
		end = start + listHeight
		if end > len(app.commits) {
			end = len(app.commits)
		}
	}

	var commitViews []tui.View
	for i := start; i < end; i++ {
		commitViews = append(commitViews, app.formatCommit(app.commits[i], i == app.selectedCommit))
	}

	// Detail panel for selected commit
	var detailViews []tui.View
	if app.selectedCommit >= 0 && app.selectedCommit < len(app.commits) {
		commit := app.commits[app.selectedCommit]
		detailViews = []tui.View{
			tui.Text("Commit: %s", commit.ShortHash).Fg(tui.ColorYellow).Bold(),
			tui.Spacer().MinHeight(1),
			tui.Text("Author: %s", commit.Author.Name).Fg(tui.ColorCyan),
			tui.Text("<%s>", commit.Author.Email).Fg(tui.ColorBrightBlack),
			tui.Spacer().MinHeight(1),
			tui.Text("Date: %s", commit.Timestamp.Format("2006-01-02 15:04:05")),
			tui.Text("(%s)", humanize.Duration(time.Since(commit.Timestamp))).Fg(tui.ColorBrightBlack),
			tui.Spacer().MinHeight(1),
			tui.Divider(),
			tui.Spacer().MinHeight(1),
		}

		// Subject and body
		detailViews = append(detailViews, tui.Text("%s", commit.Subject).Bold())
		if commit.Body != "" {
			detailViews = append(detailViews, tui.Spacer().MinHeight(1))
			lines := strings.Split(commit.Body, "\n")
			for _, line := range lines {
				if len(line) > 50 {
					line = line[:47] + "..."
				}
				detailViews = append(detailViews, tui.Text("%s", line).Fg(tui.ColorBrightBlack))
			}
		}
	}

	return tui.Group(
		// Commit list
		tui.Stack(
			tui.Bordered(
				tui.Stack(commitViews...),
			).Title("Commits").BorderFg(tui.ColorCyan),
		),
		// Detail panel
		tui.Stack(
			tui.Bordered(
				tui.Stack(detailViews...).Padding(1),
			).Title("Details").BorderFg(tui.ColorYellow),
		),
	)
}

func (app *GitScanApp) formatCommit(commit git.Commit, selected bool) tui.View {
	// Time ago
	timeAgo := humanize.Duration(time.Since(commit.Timestamp))
	if len(timeAgo) > 10 {
		timeAgo = timeAgo[:10]
	}

	// Subject (truncated)
	subject := commit.Subject
	maxLen := app.width/2 - 25
	if maxLen < 20 {
		maxLen = 20
	}
	if len(subject) > maxLen {
		subject = subject[:maxLen-3] + "..."
	}

	var bg tui.Color
	if selected {
		bg = tui.ColorCyan
	} else {
		bg = tui.ColorDefault
	}

	hashFg := tui.ColorYellow
	subjectFg := tui.ColorWhite
	if selected {
		hashFg = tui.ColorBlack
		subjectFg = tui.ColorBlack
	}

	return tui.Group(
		tui.Text(" %s ", commit.ShortHash).Fg(hashFg).Bg(bg).Bold(),
		tui.Text("%s", subject).Fg(subjectFg).Bg(bg),
		tui.Spacer(),
		tui.Text("%s ", timeAgo).Fg(tui.ColorBrightBlack).Bg(bg),
	)
}

func (app *GitScanApp) viewDiff() tui.View {
	if len(app.diffLines) == 0 {
		return tui.Stack(
			tui.Spacer(),
			tui.Text("No changes").Fg(tui.ColorBrightBlack),
			tui.Spacer(),
		)
	}

	// Calculate visible range
	viewHeight := app.height - 6
	if viewHeight < 5 {
		viewHeight = 5
	}

	start := app.diffScroll
	end := start + viewHeight
	if end > len(app.diffLines) {
		end = len(app.diffLines)
	}

	// Adjust scroll
	if app.selectedLine < start {
		app.diffScroll = app.selectedLine
		start = app.diffScroll
		end = start + viewHeight
		if end > len(app.diffLines) {
			end = len(app.diffLines)
		}
	} else if app.selectedLine >= end {
		app.diffScroll = app.selectedLine - viewHeight + 1
		if app.diffScroll < 0 {
			app.diffScroll = 0
		}
		start = app.diffScroll
		end = start + viewHeight
		if end > len(app.diffLines) {
			end = len(app.diffLines)
		}
	}

	var diffViews []tui.View
	for i := start; i < end; i++ {
		line := app.diffLines[i]
		selected := i == app.selectedLine

		var fg tui.Color
		var bg tui.Color
		prefix := " "

		switch line.Type {
		case "header":
			fg = tui.ColorCyan
			if selected {
				bg = tui.ColorCyan
				fg = tui.ColorBlack
			}
		case "hunk":
			fg = tui.ColorMagenta
			if selected {
				bg = tui.ColorMagenta
				fg = tui.ColorWhite
			}
		case "add":
			fg = tui.ColorGreen
			prefix = "+"
			if selected {
				bg = tui.ColorGreen
				fg = tui.ColorBlack
			}
		case "remove":
			fg = tui.ColorRed
			prefix = "-"
			if selected {
				bg = tui.ColorRed
				fg = tui.ColorWhite
			}
		default:
			fg = tui.ColorWhite
			if selected {
				bg = tui.ColorBrightBlack
			}
		}

		text := line.Text
		maxLen := app.width - 4
		if len(text) > maxLen {
			text = text[:maxLen-3] + "..."
		}

		if line.Type == "header" || line.Type == "hunk" {
			diffViews = append(diffViews, tui.Text(" %s", text).Fg(fg).Bg(bg))
		} else {
			diffViews = append(diffViews, tui.Text(" %s%s", prefix, text).Fg(fg).Bg(bg))
		}
	}

	// Commit info in header
	var commitInfo string
	if app.selectedCommit >= 0 && app.selectedCommit < len(app.commits) {
		c := app.commits[app.selectedCommit]
		commitInfo = fmt.Sprintf("%s - %s", c.ShortHash, c.Subject)
		if len(commitInfo) > 60 {
			commitInfo = commitInfo[:57] + "..."
		}
	}

	return tui.Stack(
		tui.Text(" %s", commitInfo).Fg(tui.ColorYellow).Bold(),
		tui.Bordered(
			tui.Stack(diffViews...),
		).Title(fmt.Sprintf("Diff (%d/%d)", app.selectedLine+1, len(app.diffLines))).BorderFg(tui.ColorGreen),
	)
}

func (app *GitScanApp) viewFiles() tui.View {
	if len(app.files) == 0 {
		return tui.Stack(
			tui.Spacer(),
			tui.Text("No files changed").Fg(tui.ColorBrightBlack),
			tui.Spacer(),
		)
	}

	// File list
	listHeight := app.height - 8
	if listHeight < 5 {
		listHeight = 5
	}

	start := app.fileScroll
	end := start + listHeight
	if end > len(app.files) {
		end = len(app.files)
	}

	// Adjust scroll
	if app.selectedFile < start {
		app.fileScroll = app.selectedFile
		start = app.fileScroll
		end = start + listHeight
		if end > len(app.files) {
			end = len(app.files)
		}
	} else if app.selectedFile >= end {
		app.fileScroll = app.selectedFile - listHeight + 1
		if app.fileScroll < 0 {
			app.fileScroll = 0
		}
		start = app.fileScroll
		end = start + listHeight
		if end > len(app.files) {
			end = len(app.files)
		}
	}

	var fileViews []tui.View
	for i := start; i < end; i++ {
		file := app.files[i]
		selected := i == app.selectedFile

		// Status icon
		var icon string
		var iconColor tui.Color
		switch file.Status {
		case "added":
			icon = "A"
			iconColor = tui.ColorGreen
		case "deleted":
			icon = "D"
			iconColor = tui.ColorRed
		case "modified":
			icon = "M"
			iconColor = tui.ColorYellow
		case "renamed":
			icon = "R"
			iconColor = tui.ColorCyan
		default:
			icon = "?"
			iconColor = tui.ColorBrightBlack
		}

		// Stats
		stats := fmt.Sprintf("+%d -%d", file.Additions, file.Deletions)

		var bg tui.Color
		if selected {
			bg = tui.ColorCyan
			iconColor = tui.ColorBlack
		}

		path := file.Path
		maxLen := app.width - 20
		if len(path) > maxLen {
			path = "..." + path[len(path)-maxLen+3:]
		}

		fileViews = append(fileViews, tui.Group(
			tui.Text(" %s ", icon).Fg(iconColor).Bg(bg).Bold(),
			tui.Text("%s", path).Fg(tui.ColorWhite).Bg(bg),
			tui.Spacer(),
			tui.Text("%s ", stats).Fg(tui.ColorBrightBlack).Bg(bg),
		))
	}

	// Summary
	var totalAdd, totalDel int
	if app.diff != nil {
		totalAdd = app.diff.TotalAdded
		totalDel = app.diff.TotalRemoved
	}

	summary := tui.Group(
		tui.Text(" %d files changed", len(app.files)).Fg(tui.ColorWhite),
		tui.Spacer().MinWidth(2),
		tui.Text("+%d", totalAdd).Fg(tui.ColorGreen),
		tui.Spacer().MinWidth(1),
		tui.Text("-%d", totalDel).Fg(tui.ColorRed),
	)

	return tui.Stack(
		summary,
		tui.Bordered(
			tui.Stack(fileViews...),
		).Title("Changed Files").BorderFg(tui.ColorYellow),
	)
}
