package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/git"
	"github.com/deepnoodle-ai/wonton/tmux"
	"github.com/deepnoodle-ai/wonton/tui"
)

type app struct {
	server *tmux.Server
	saver  *saver
	rt     *tui.Runtime

	spaces  []*space
	current int
	panes   map[string]*pane
	seq     int // last number used for a pane or space ID

	navWidth int
	navDrag  bool   // the sidebar's edge is being dragged
	capture  string // the pane receiving a forwarded mouse drag
	swallow  bool   // a press closed the prompt; ignore the rest of it

	// A text selection in one pane, made by dragging across it.
	selPane   string            // the pane holding it, or ""
	sel       tui.GridSelection // in the pane's cells
	selecting bool              // the drag is still going

	hits   tui.HitMap
	menu   tui.MenuState
	dialog *dialog
	prefix bool // Ctrl+B was pressed; the next key is corral's

	// pending holds commands from menu actions, which run inside the
	// menu's event handling, until HandleEvent returns them.
	pending []tui.Cmd

	status   string
	statusAt time.Time

	branches   map[string]string // git branch by directory
	branchScan time.Time

	frame   uint64
	waking  atomic.Bool
	ended   bool // the user ended every session on the way out
	initErr error
}

// dialog is a one-line text prompt, used for renaming.
type dialog struct {
	title  string
	value  string
	submit func(string)
}

// wakeEvent asks the runtime to redraw: a pane has a new screen or ended.
type wakeEvent struct{}

func (wakeEvent) Timestamp() time.Time { return time.Now() }

// branchesEvent carries the result of a background git scan.
type branchesEvent map[string]string

func (branchesEvent) Timestamp() time.Time { return time.Now() }

func newApp(server *tmux.Server) *app {
	return &app{
		server:   server,
		saver:    newSaver(func(data string) { _ = server.SetOption(context.Background(), optionName, data) }),
		panes:    map[string]*pane{},
		navWidth: 28,
		branches: map[string]string{},
	}
}

func (a *app) SetRuntime(rt *tui.Runtime) { a.rt = rt }

func (a *app) Init() error {
	return a.restore()
}

// Destroy detaches from every session and leaves it running, unless the
// user chose to end them.
func (a *app) Destroy() {
	if a.ended {
		return
	}
	a.saveNow()
	for _, p := range a.panes {
		_ = p.term.Detach()
	}
}

// wake is every terminal's OnUpdate. It runs on a tmux goroutine, so it only
// queues an event, and never more than one at a time.
func (a *app) wake() {
	if a.rt != nil && a.waking.CompareAndSwap(false, true) {
		go a.rt.SendEvent(wakeEvent{})
	}
}

func (a *app) space() *space { return a.spaces[a.current] }

// focused returns the pane with focus in the current space, or nil.
func (a *app) focused() *pane {
	return a.panes[a.space().focus]
}

func (a *app) setStatus(format string, args ...any) {
	a.status = fmt.Sprintf(format, args...)
	a.statusAt = time.Now()
}

func (a *app) HandleEvent(ev tui.Event) []tui.Cmd {
	switch e := ev.(type) {
	case wakeEvent:
		a.waking.Store(false)
		return a.reap()
	case branchesEvent:
		a.branches = e
	case tui.TickEvent:
		a.frame = e.Frame
		a.syncSizes()
		cmds := a.reap()
		if time.Since(a.branchScan) > 3*time.Second {
			a.branchScan = time.Now()
			cmds = append(cmds, a.scanBranches())
		}
		return cmds
	case tui.KeyEvent:
		return a.flush(a.handleKey(e))
	case tui.MouseEvent:
		return a.flush(a.handleMouse(e))
	}
	return nil
}

// flush appends the commands menu actions left behind.
func (a *app) flush(cmds []tui.Cmd) []tui.Cmd {
	cmds = append(cmds, a.pending...)
	a.pending = nil
	return cmds
}

// reap drops panes whose sessions have ended.
func (a *app) reap() []tui.Cmd {
	var gone []string
	for id, p := range a.panes {
		if p.term.Exited() {
			gone = append(gone, id)
		}
	}
	if len(gone) == 0 {
		return nil
	}
	for _, id := range gone {
		a.removePane(id)
	}
	a.save()
	return nil
}

// syncSizes fits every visible terminal to the box it was drawn in. tmux
// resizes are cheap, but each one makes the program inside redraw, so while
// a gap is being dragged they go out a few times a second rather than on
// every frame.
func (a *app) syncSizes() {
	if (a.space().split.Dragging() || a.navDrag) && a.frame%6 != 0 {
		return
	}
	for _, p := range a.panesIn(a.space()) {
		if b, ok := a.hits.Bounds("pane:" + p.id); ok {
			p.term.Resize(b.Dx()-2, b.Dy()-2)
		}
	}
}

// scanBranches looks up the git branch of every directory a pane is in.
func (a *app) scanBranches() tui.Cmd {
	dirs := map[string]bool{}
	for _, sp := range a.spaces {
		dirs[sp.dir] = true
	}
	for _, p := range a.panes {
		if path := p.term.Screen().Path; path != "" {
			dirs[path] = true
		}
	}
	return func() tui.Event {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		out := branchesEvent{}
		for dir := range dirs {
			repo, err := git.Open(dir)
			if err != nil {
				continue
			}
			if branch, err := repo.CurrentBranch(ctx); err == nil {
				out[dir] = branch
			}
		}
		return out
	}
}

// --- actions ---

func (a *app) focusPane(id string) {
	p := a.panes[id]
	if p == nil {
		return
	}
	for i, sp := range a.spaces {
		if sp == p.space {
			a.current = i
		}
	}
	p.space.focus = id
	if p.space.zoom != "" && p.space.zoom != id {
		p.space.zoom = id
	}
}

func (a *app) switchSpace(i int) {
	if i >= 0 && i < len(a.spaces) {
		cancelDrag(a.space())
		a.current = i
	}
}

// cancelDrag ends a divider drag in progress. Zooming or switching spaces
// hides the split, so it would never see the button release.
func cancelDrag(sp *space) {
	if sp.split.Dragging() {
		sp.split.HandleMouse(tui.MouseEvent{Type: tui.MouseDragCancel})
	}
}

// paneSize estimates a new pane's size from its neighbor's, for the moment
// before the first render measures it.
func (a *app) paneSize(near string, axis tui.SplitAxis) (int, int) {
	w, h := 80, 24
	if b, ok := a.hits.Bounds("pane:" + near); ok {
		w, h = b.Dx()-2, b.Dy()-2
		if axis == tui.SplitColumns {
			w /= 2
		} else {
			h /= 2
		}
	}
	return w, h
}

// startDir is where a pane opened next to p starts: wherever p's shell is.
func (a *app) startDir(sp *space, p *pane) string {
	if p != nil {
		if path := p.term.Screen().Path; path != "" {
			return path
		}
	}
	return sp.dir
}

func (a *app) splitPane(id string, axis tui.SplitAxis) []tui.Cmd {
	target := a.panes[id]
	if target == nil {
		return a.newColumn(a.space())
	}
	sp := target.space
	w, h := a.paneSize(id, axis)
	p, err := a.openPane(sp, a.startDir(sp, target), w, h)
	if err != nil {
		a.setStatus("could not open a pane: %v", err)
		return nil
	}
	sp.split.Split(id, p.id, axis)
	sp.zoom = ""
	a.focusPane(p.id)
	a.save()
	return nil
}

func (a *app) newColumn(sp *space) []tui.Cmd {
	w, h := 80, 24
	if b, ok := a.hits.Bounds("main"); ok {
		w, h = b.Dx()/(len(sp.split.Leaves())+1)-2, b.Dy()-2
	}
	p, err := a.openPane(sp, a.startDir(sp, a.panes[sp.focus]), w, h)
	if err != nil {
		a.setStatus("could not open a pane: %v", err)
		return nil
	}
	sp.split.Append(p.id, tui.SplitColumns)
	sp.zoom = ""
	a.focusPane(p.id)
	a.save()
	return nil
}

func (a *app) closePane(id string) []tui.Cmd {
	p := a.panes[id]
	if p == nil {
		return nil
	}
	_ = p.term.Kill()
	a.removePane(id)
	a.save()
	return nil
}

// removePane takes a pane out of its space; its session is already gone.
func (a *app) removePane(id string) {
	p := a.panes[id]
	if p == nil {
		return
	}
	delete(a.panes, id)
	sp := p.space
	// Hand focus to the pane before the one going, or after it if it was
	// first. The layout's order is enough: screen positions may be stale
	// for a space that is not showing.
	if sp.focus == id {
		leaves := sp.split.Leaves()
		if i := slices.Index(leaves, id); i > 0 {
			sp.focus = leaves[i-1]
		} else if i == 0 && len(leaves) > 1 {
			sp.focus = leaves[1]
		}
	}
	sp.split.Remove(id)
	if a.capture == id {
		a.capture = ""
	}
	if a.selPane == id {
		a.selPane, a.selecting = "", false
	}
	a.fixFocus(sp)
}

func (a *app) toggleZoom(id string) {
	p := a.panes[id]
	if p == nil {
		return
	}
	cancelDrag(p.space)
	if p.space.zoom == id {
		p.space.zoom = ""
	} else {
		p.space.zoom = id
		p.space.focus = id
	}
}

func (a *app) moveFocus(dx, dy int) {
	sp := a.space()
	if sp.zoom != "" {
		return
	}
	if next, ok := sp.split.Neighbor(sp.focus, dx, dy); ok {
		sp.focus = next
	}
}

func (a *app) cycleFocus(dir int) {
	sp := a.space()
	leaves := sp.split.Leaves()
	if len(leaves) == 0 {
		return
	}
	i := slices.Index(leaves, sp.focus)
	next := leaves[(i+dir+len(leaves))%len(leaves)]
	a.focusPane(next)
}

func (a *app) newSpace(dir string) []tui.Cmd {
	sp := a.newSpaceValue(dir)
	p, err := a.openPane(sp, sp.dir, 80, 24)
	if err != nil {
		a.setStatus("could not open a pane: %v", err)
		return nil
	}
	sp.split.Append(p.id, tui.SplitColumns)
	sp.focus = p.id
	a.spaces = append(a.spaces, sp)
	a.current = len(a.spaces) - 1
	a.save()
	return nil
}

// closeSpace ends every pane in a space and removes it. The last space is
// replaced with a fresh one rather than leaving nothing.
func (a *app) closeSpace(sp *space) []tui.Cmd {
	for _, p := range a.panesIn(sp) {
		_ = p.term.Kill()
		delete(a.panes, p.id)
	}
	i := slices.Index(a.spaces, sp)
	if i < 0 {
		return nil
	}
	a.spaces = slices.Delete(a.spaces, i, i+1)
	if len(a.spaces) == 0 {
		a.newSpace(sp.dir)
		if len(a.spaces) == 0 {
			// No pane would open, but there must still be a space to show.
			a.spaces = append(a.spaces, a.newSpaceValue(sp.dir))
		}
	}
	if a.current >= len(a.spaces) {
		a.current = len(a.spaces) - 1
	} else if a.current > i {
		a.current--
	}
	a.save()
	return nil
}

func (a *app) renamePane(id string) []tui.Cmd {
	p := a.panes[id]
	if p == nil {
		return nil
	}
	return a.prompt("Rename pane", p.label(), func(v string) {
		p.name = v
	})
}

func (a *app) renameSpace(sp *space) []tui.Cmd {
	return a.prompt("Rename space", sp.name, func(v string) {
		if v != "" {
			sp.name = v
		}
	})
}

func (a *app) prompt(title, value string, submit func(string)) []tui.Cmd {
	a.dialog = &dialog{title: title, value: value, submit: submit}
	return []tui.Cmd{tui.Focus("dialog-input")}
}

// copySelection copies the text just selected in a pane to the clipboard.
// The selection stays highlighted until the next click or key.
func (a *app) copySelection() {
	p := a.panes[a.selPane]
	if p == nil || a.sel.Empty() {
		a.selPane = ""
		return
	}
	// Rows the terminal wrapped copy as the one line they were. If tmux
	// cannot say which those are, copy row by row.
	wrapped, _ := p.term.WrappedRows()
	text := a.sel.Text(p.term.Screen().Cells, wrapped)
	if err := clipboard.Write(text); err != nil {
		a.setStatus("copy failed: %v", err)
		return
	}
	if n := strings.Count(text, "\n") + 1; n > 1 {
		a.setStatus("copied %d lines", n)
	} else {
		a.setStatus("copied %d characters", len([]rune(text)))
	}
}

func (a *app) copyScreen(id string) {
	p := a.panes[id]
	if p == nil {
		return
	}
	if err := clipboard.Write(p.term.Screen().Text()); err != nil {
		a.setStatus("copy failed: %v", err)
		return
	}
	a.setStatus("copied %s's screen to the clipboard", p.label())
}

func (a *app) pasteClipboard(id string) {
	p := a.panes[id]
	if p == nil {
		return
	}
	text, err := clipboard.Read()
	if err != nil {
		a.setStatus("paste failed: %v", err)
		return
	}
	_ = p.term.Paste(text)
}

// quit leaves every session running.
func (a *app) quit() []tui.Cmd {
	return []tui.Cmd{tui.Quit()}
}

// endAll ends every session and quits.
func (a *app) endAll() []tui.Cmd {
	a.ended = true
	for _, p := range a.panes {
		_ = p.term.Detach()
	}
	_ = a.server.Kill(context.Background())
	a.panes = map[string]*pane{}
	return []tui.Cmd{tui.Quit()}
}
