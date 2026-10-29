package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/runewidth"
	"github.com/deepnoodle-ai/wonton/tui"
)

var (
	accent = color.Palette(75)  // focus, selection, buttons
	dim    = color.Palette(244) // secondary text
	faint  = color.Palette(238) // borders of panes without focus
	rowBg  = color.Palette(236) // the selected row in the sidebar
	busy   = color.Palette(214) // a pane that is writing output
	edge   = color.Palette(235) // the line between the sidebar and the panes
	handle = color.Palette(250) // an edge between panes under the pointer: not focus
	selBg  = color.Palette(24)  // selected text in a pane
)

func (a *app) View() tui.View {
	a.hits.Reset()
	main := tui.Stack(
		a.topBar(),
		a.hits.Area("main", a.mainArea()),
	).Flex(1)
	root := tui.Group(
		tui.Width(a.navWidth, a.sidebar()),
		a.hits.Area("nav-gutter", tui.Width(1, tui.Fill('│').Fg(edge))),
		tui.Width(1, tui.Fill(' ')),
		main,
	)
	return tui.ZStack(
		a.hits.Area("root", root),
		a.dialogView(),
		tui.Menu(&a.menu).
			BorderStyle(tui.NewStyle().WithForeground(accent)).
			SelectedStyle(tui.NewStyle().WithBackground(accent).WithForeground(color.Black)).
			HintStyle(tui.NewStyle().WithForeground(dim)),
	)
}

// --- sidebar ---

func (a *app) sidebar() tui.View {
	w := a.navWidth
	rows := []tui.View{
		tui.Group(
			a.hits.Area("spaces", tui.Text("%s", fit(" spaces", w-3)).Fg(dim)),
			a.hits.Area("btn:new-space", tui.Text(" + ").Fg(accent)),
		),
	}
	for i, sp := range a.spaces {
		rows = append(rows, a.spaceRow(sp, i == a.current, w))
	}
	rows = append(rows, tui.Text(""), tui.Text("%s", fit(" panes", w)).Fg(dim))
	for _, sp := range a.spaces {
		for _, p := range a.panesIn(sp) {
			rows = append(rows, a.paneRow(p, w))
		}
	}
	rows = append(rows, tui.Spacer())
	rows = append(rows, a.sidebarFooter(w)...)
	return tui.Stack(rows...)
}

// sidebarFooter is the bottom of the sidebar: what corral is waiting for or
// has to say, then the keys to know. Clicking the keys lists them all.
func (a *app) sidebarFooter(w int) []tui.View {
	var rows []tui.View
	switch {
	case a.prefix:
		rows = append(rows, tui.Group(
			tui.Text(" "),
			tui.Text(" ^B ").Bg(accent).Fg(color.Black).Bold(),
			tui.Text("%s", fit(" press a key", w-5)).Fg(accent),
		), tui.Text(""))
	case a.status != "" && time.Since(a.statusAt) < 4*time.Second:
		lines := strings.Split(tui.WrapText(a.status, w-2), "\n")
		for _, line := range lines[:min(len(lines), 3)] {
			rows = append(rows, tui.Text("%s", fit(" "+line, w)).Fg(busy))
		}
		rows = append(rows, tui.Text(""))
	}
	help := tui.Stack(
		a.helpRow("^B ?", "all keys", w),
		a.helpRow("^B d", "quit", w),
		a.helpRow("right-click", "menus", w),
	)
	return append(rows, a.hits.Area("btn:help", help))
}

func (a *app) helpRow(key, does string, w int) tui.View {
	return tui.Group(
		tui.Text(" %-12s", key).Fg(accent),
		tui.Text("%s", fit(does, w-13)).Fg(dim),
	)
}

func (a *app) spaceRow(sp *space, current bool, w int) tui.View {
	dot := "○"
	dotColor := dim
	for _, p := range a.panesIn(sp) {
		if p.busy() {
			dot, dotColor = "●", busy
			break
		}
	}
	name := tui.Text("%s", fit(fmt.Sprintf(" %s %s", dot, sp.name), w))
	detail := tui.Text("%s", fit("   "+a.describeDir(sp.dir), w)).Fg(dim)
	if current {
		name.Bg(rowBg).Bold()
		detail.Bg(rowBg)
	}
	name.Fg(dotColor)
	if current {
		name.Fg(accent)
	}
	return a.hits.Area("space:"+sp.id, tui.Stack(name, detail))
}

func (a *app) paneRow(p *pane, w int) tui.View {
	focused := p.space == a.space() && p.space.focus == p.id
	dot, dotColor := "○", dim
	if p.busy() {
		dot, dotColor = "●", busy
	}
	label := fmt.Sprintf(" %s %s · %s", dot, p.label(), p.space.name)
	top := tui.Text("%s", fit(label, w)).Fg(dotColor)
	bottom := tui.Text("%s", fit("   "+a.describeDir(p.term.Screen().Path), w)).Fg(dim)
	if focused {
		top.Bg(rowBg).Fg(accent).Bold()
		bottom.Bg(rowBg)
	}
	return a.hits.Area("item:"+p.id, tui.Stack(top, bottom))
}

// describeDir shortens a directory for display and adds its git branch.
func (a *app) describeDir(dir string) string {
	if dir == "" {
		return ""
	}
	short := dir
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(dir, home) {
		short = "~" + strings.TrimPrefix(dir, home)
	}
	if branch := a.branches[dir]; branch != "" {
		return filepath.Base(short) + " (" + branch + ")"
	}
	return short
}

// fit truncates or pads s to exactly w columns, so a row's background spans
// the sidebar.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = runewidth.Truncate(s, w, "…")
	return s + strings.Repeat(" ", max(0, w-runewidth.StringWidth(s)))
}

// --- top bar ---

func (a *app) topBar() tui.View {
	sp := a.space()
	n := len(sp.split.Leaves())
	items := []tui.View{
		a.hits.Area("topbar", tui.Group(
			tui.Text(" %s", sp.name).Bold(),
			tui.Text("  %d pane%s", n, plural(n)).Fg(dim),
		)),
		tui.Spacer(),
	}
	if sp.zoom != "" {
		items = append(items, a.button("unzoom", "⤡ unzoom"))
	}
	items = append(items,
		a.button("split-right", "▥ split right"),
		a.button("split-down", "▤ split down"),
		a.button("column", "+ column"),
	)
	return tui.Group(items...)
}

// button is a clickable label; click handles "btn:"+id.
func (a *app) button(id, label string) tui.View {
	return a.hits.Area("btn:"+id, tui.Text(" %s ", label).Fg(accent))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// --- panes ---

func (a *app) mainArea() tui.View {
	sp := a.space()
	if sp.zoom != "" {
		// A one-pane layout of its own fills the area, the same as the
		// space's layout would.
		zoomed := &tui.SplitState{Root: &tui.SplitNode{ID: sp.zoom}}
		return tui.Split(zoomed, a.paneView)
	}
	if len(sp.split.Leaves()) == 0 {
		return tui.Stack(
			tui.Spacer(),
			tui.Text("No panes in %s", sp.name).Fg(dim).Center(),
			tui.Text(""),
			tui.Group(tui.Spacer(), a.hits.Area("btn:column", tui.Text("[ + new pane ]").Fg(accent)), tui.Spacer()),
			tui.Spacer(),
		).Flex(1)
	}
	return tui.Split(&sp.split, a.paneView).ActiveGutterStyle(tui.NewStyle().WithForeground(handle))
}

func (a *app) paneView(id string) tui.View {
	p := a.panes[id]
	if p == nil {
		return tui.Empty()
	}
	sp := p.space
	focused := sp.focus == id
	screen := p.term.Screen()

	cells := tui.Cells(screen.Cells)
	if a.selPane == id {
		cells.Select(a.sel).SelectStyle(tui.NewStyle().WithBackground(selBg).WithForeground(color.Palette(255)))
	}
	if focused && screen.CursorVisible {
		cells.Cursor(screen.CursorX, screen.CursorY)
	}

	dot := "○"
	if p.busy() {
		dot = "●"
	}
	title := fmt.Sprintf(" %s %s", dot, p.label())
	if where := a.describeDir(screen.Path); where != "" {
		title += " · " + where
	}
	if screen.InMode {
		title += fmt.Sprintf(" · scrolled %d/%d", screen.ScrollPosition, screen.HistorySize)
	}
	if sp.zoom == id {
		title += " · zoomed"
	}
	title += " "

	border, titleStyle := faint, tui.NewStyle().WithForeground(dim)
	if focused {
		border, titleStyle = accent, tui.NewStyle().WithForeground(accent).WithBold()
	}
	box := tui.Bordered(cells).
		Border(&tui.RoundedBorder).
		BorderFg(border).
		Title(title).
		TitleStyle(titleStyle)
	return a.hits.Area("pane:"+id, box)
}

// --- prompt ---

func (a *app) dialogView() tui.View {
	d := a.dialog
	if d == nil {
		return tui.Empty()
	}
	input := tui.InputField(&d.value).
		ID("dialog-input").
		OnSubmit(func(v string) {
			a.dialog = nil
			d.submit(strings.TrimSpace(v))
			a.saveNow()
		})
	content := tui.ZStack(
		tui.Fill(' '),
		tui.Stack(input, tui.Text("Enter to save · Esc to cancel").Fg(dim)),
	).Align(tui.AlignLeft)
	box := tui.Bordered(tui.PaddingHV(1, 0, content)).
		Border(&tui.RoundedBorder).
		BorderFg(accent).
		Title(" " + d.title + " ").
		TitleStyle(tui.NewStyle().WithForeground(accent).WithBold())
	return a.hits.Area("dialog", tui.Size(48, 4, box))
}
