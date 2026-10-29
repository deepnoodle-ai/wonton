package main

import (
	"image"
	"slices"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// openMenu opens the context menu for whatever was right-clicked: the hit
// area key under the pointer, or a gap in the layout.
func (a *app) openMenu(key string, x, y int) {
	sp := a.space()
	if sp.zoom == "" {
		if branch, index, ok := sp.split.DividerAt(x, y); ok {
			a.menu.Open(x, y, "edge", a.gapMenu(sp, branch, index))
			return
		}
	}
	kind, id, _ := strings.Cut(key, ":")
	switch kind {
	case "pane", "item":
		if p := a.panes[id]; p != nil {
			a.menu.Open(x, y, p.label(), a.paneMenu(p, kind == "item"))
			return
		}
	case "space":
		if i := a.spaceIndex(id); i >= 0 {
			sp := a.spaces[i]
			a.menu.Open(x, y, "space "+sp.name, a.spaceMenu(sp))
			return
		}
	case "spaces":
		a.menu.Open(x, y, "spaces", []tui.MenuItem{
			{Label: "New space here", Hint: "^B s", Action: a.run(func() []tui.Cmd {
				return a.newSpace(a.startDir(a.space(), a.focused()))
			})},
		})
		return
	}
	a.menu.Open(x, y, "corral", a.workspaceMenu())
}

// run adapts an action that returns commands to a menu action. The menu
// runs actions inside HandleEvent, which hands the commands to the runtime
// once it returns.
func (a *app) run(fn func() []tui.Cmd) func() {
	return func() { a.pending = append(a.pending, fn()...) }
}

func (a *app) paneMenu(p *pane, fromSidebar bool) []tui.MenuItem {
	zoom := "Zoom"
	if p.space.zoom == p.id {
		zoom = "Unzoom"
	}
	screen := p.term.Screen()
	items := []tui.MenuItem{}
	if fromSidebar {
		items = append(items, tui.MenuItem{Label: "Focus", Action: func() { a.focusPane(p.id) }})
	}
	items = append(items,
		tui.MenuItem{Label: "Split right", Hint: "^B |", Action: a.run(func() []tui.Cmd {
			return a.splitPane(p.id, tui.SplitColumns)
		})},
		tui.MenuItem{Label: "Split down", Hint: "^B -", Action: a.run(func() []tui.Cmd {
			return a.splitPane(p.id, tui.SplitRows)
		})},
		tui.MenuItem{Label: "New column", Hint: "^B c", Action: a.run(func() []tui.Cmd {
			return a.newColumn(p.space)
		})},
		tui.MenuSeparator(),
		tui.MenuItem{Label: zoom, Hint: "^B z", Action: func() {
			a.focusPane(p.id)
			a.toggleZoom(p.id)
		}},
		tui.MenuItem{Label: "Rename…", Hint: "^B ,", Action: a.run(func() []tui.Cmd {
			return a.renamePane(p.id)
		})},
		tui.MenuItem{Label: "Copy screen", Action: func() { a.copyScreen(p.id) }},
		tui.MenuItem{Label: "Paste", Action: func() { a.pasteClipboard(p.id) }},
		tui.MenuItem{Label: "Back to live output", Disabled: !screen.InMode, Action: func() {
			_ = p.term.SendKey(tui.KeyEvent{Key: tui.KeyEscape})
		}},
		tui.MenuSeparator(),
		tui.MenuItem{Label: "Close pane", Hint: "^B x", Danger: true, Action: a.run(func() []tui.Cmd {
			return a.closePane(p.id)
		})},
	)
	return items
}

func (a *app) spaceMenu(sp *space) []tui.MenuItem {
	return []tui.MenuItem{
		{Label: "Switch to", Disabled: sp == a.space(), Action: func() { a.switchSpace(a.spaceIndex(sp.id)) }},
		{Label: "New pane", Action: a.run(func() []tui.Cmd {
			a.switchSpace(a.spaceIndex(sp.id))
			return a.newColumn(sp)
		})},
		{Label: "Rename…", Action: a.run(func() []tui.Cmd { return a.renameSpace(sp) })},
		{Label: "Equalize panes", Action: a.run(func() []tui.Cmd {
			sp.split.Equalize()
			a.save()
			return nil
		})},
		tui.MenuSeparator(),
		{Label: "Close space", Danger: true, Action: a.run(func() []tui.Cmd { return a.closeSpace(sp) })},
	}
}

// gapMenu acts on the two neighbors a gap separates, and on the run of
// panes it belongs to.
func (a *app) gapMenu(sp *space, branch *tui.SplitNode, index int) []tui.MenuItem {
	rotate := "Stack as rows"
	if branch.Axis == tui.SplitRows {
		rotate = "Arrange as columns"
	}
	left, right := branch.Children[index], branch.Children[index+1]
	// A pane can exit while the menu is open and reshape the layout, so an
	// action runs only if the gap is still between the same two neighbors.
	onGap := func(act func(c []*tui.SplitNode)) func() {
		return func() {
			c := branch.Children
			if index+1 >= len(c) || c[index] != left || c[index+1] != right || !inLayout(sp.split.Root, branch) {
				a.setStatus("that gap is gone")
				return
			}
			act(c)
			a.save()
		}
	}
	return []tui.MenuItem{
		{Label: "Even out these two", Action: onGap(func(c []*tui.SplitNode) {
			w := (c[index].Weight + c[index+1].Weight) / 2
			c[index].Weight, c[index+1].Weight = w, w
		})},
		{Label: "Swap sides", Action: onGap(func(c []*tui.SplitNode) {
			c[index], c[index+1] = c[index+1], c[index]
		})},
		{Label: rotate, Action: onGap(func([]*tui.SplitNode) {
			if branch.Axis == tui.SplitColumns {
				branch.Axis = tui.SplitRows
			} else {
				branch.Axis = tui.SplitColumns
			}
		})},
		tui.MenuSeparator(),
		{Label: "Equalize all", Hint: "^B =", Action: a.run(func() []tui.Cmd {
			a.space().split.Equalize()
			a.save()
			return nil
		})},
	}
}

// inLayout reports whether n is in the layout under root.
func inLayout(root, n *tui.SplitNode) bool {
	if root == n {
		return true
	}
	return root != nil && slices.ContainsFunc(root.Children, func(c *tui.SplitNode) bool {
		return inLayout(c, n)
	})
}

func (a *app) workspaceMenu() []tui.MenuItem {
	sp := a.space()
	return []tui.MenuItem{
		{Label: "New column", Hint: "^B c", Action: a.run(func() []tui.Cmd { return a.newColumn(sp) })},
		{Label: "New space", Hint: "^B s", Action: a.run(func() []tui.Cmd {
			return a.newSpace(a.startDir(sp, a.focused()))
		})},
		{Label: "Equalize panes", Hint: "^B =", Action: a.run(func() []tui.Cmd {
			sp.split.Equalize()
			a.save()
			return nil
		})},
		{Label: "Keys…", Hint: "^B ?", Action: a.openHelp},
		tui.MenuSeparator(),
		{Label: "Quit, keep sessions", Hint: "^B d", Action: a.run(a.quit)},
		{Label: "Quit and end all sessions", Danger: true, Action: a.run(a.endAll)},
	}
}

// openHelp lists corral's keys as a menu: choosing one runs it.
func (a *app) openHelp() {
	key := func(r rune) func() {
		return a.run(func() []tui.Cmd { return a.command(tui.KeyEvent{Rune: r}) })
	}
	items := []tui.MenuItem{
		{Label: "Split right", Hint: "|", Action: key('|')},
		{Label: "Split down", Hint: "-", Action: key('-')},
		{Label: "New column", Hint: "c", Action: key('c')},
		{Label: "Zoom pane", Hint: "z", Action: key('z')},
		{Label: "Rename pane", Hint: ",", Action: key(',')},
		{Label: "Pane menu", Hint: "m", Action: key('m')},
		{Label: "Focus neighbor", Hint: "←↑↓→ hjkl"},
		{Label: "Next pane", Hint: "o", Action: key('o')},
		{Label: "Equalize", Hint: "=", Action: key('=')},
		tui.MenuSeparator(),
		{Label: "New space", Hint: "s", Action: key('s')},
		{Label: "Next / previous space", Hint: "n p", Action: key('n')},
		tui.MenuSeparator(),
		{Label: "Send Ctrl+B to pane", Hint: "^B"},
		{Label: "Close pane", Hint: "x", Danger: true, Action: key('x')},
		{Label: "Quit, keep sessions", Hint: "d", Action: key('d')},
	}
	w, h := a.screenSize()
	a.menu.Open(w/2-16, max(1, h/2-len(items)/2-2), "Ctrl+B, then…", items)
}

func (a *app) screenSize() (int, int) {
	if b, ok := a.hits.Bounds("root"); ok {
		return b.Dx(), b.Dy()
	}
	return 120, 40
}

func inRect(x, y int, r image.Rectangle) bool {
	return image.Pt(x, y).In(r)
}
