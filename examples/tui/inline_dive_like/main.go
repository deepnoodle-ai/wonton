package main

import (
	"fmt"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

type diveLikeApp struct {
	runner *tui.InlineApp

	frame uint64

	processing bool
	showTodos  bool
	showDialog bool

	showAutocomplete   bool
	showCompactionInfo bool
	showExitHint       bool

	toolCount   int
	expandedMsg bool

	inputText string
}

func (a *diveLikeApp) LiveView() tui.View {
	views := make([]tui.View, 0)

	if a.showDialog {
		views = append(views, tui.Text(""))
		views = append(views, a.dialogView())
		return tui.Stack(views...)
	}

	if a.processing {
		views = append(views, tui.Text(""))
		liveContent := a.buildLiveView()
		if liveContent != nil {
			views = append(views, liveContent)
		}
	}

	if !a.processing && a.showTodos {
		views = append(views, tui.Text(""))
		views = append(views, a.todoListView())
	}

	views = append(views, tui.Text(""))
	views = append(views, tui.Divider())
	views = append(views,
		tui.InputField(&a.inputText).
			ID("main-input").
			Prompt(" > ").
			PromptStyle(tui.NewStyle().WithForeground(tui.ColorCyan)).
			Placeholder("Type a message...").
			Multiline(true).
			MaxHeight(10).
			OnSubmit(func(value string) {
				if a.runner != nil {
					a.runner.Printf("You entered: %s", value)
				}
			}),
	)
	views = append(views, tui.Divider())

	footerViews := make([]tui.View, 0, 8)
	if a.showAutocomplete {
		matches := []string{"file.txt", "folder/", "README.md", "notes.md", "todo.md"}
		for i, match := range matches {
			if i >= 8 {
				break
			}
			prefix := "   "
			if i == 1 {
				prefix = " \u275f "
			}
			footerViews = append(footerViews, tui.Text("%s@%s", prefix, match).Hint())
		}
	} else if a.showCompactionInfo {
		footerViews = append(footerViews,
			tui.Text(" \u26a1").Fg(tui.ColorYellow),
			tui.Text(" Context compacted:").Hint(),
			tui.Text(" 12400 \u2192 8200 tokens"),
			tui.Text(" (12 messages summarized)").Hint(),
		)
	} else if a.showExitHint {
		footerViews = append(footerViews, tui.Text(" Press Ctrl+C again to exit").Hint())
	}

	// Pad footer to consistent height to prevent layout shifts
	for len(footerViews) < 8 {
		footerViews = append(footerViews, tui.Text(""))
	}

	views = append(views, footerViews...)

	if len(views) == 0 {
		return tui.Text("")
	}

	return tui.Stack(views...).Gap(0)
}

func (a *diveLikeApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Rune {
		case 'q':
			return []tui.Cmd{tui.Quit()}
		case 'p':
			a.processing = !a.processing
		case 'd':
			a.showDialog = !a.showDialog
		case 't':
			a.showTodos = !a.showTodos
		case 'a':
			a.showAutocomplete = !a.showAutocomplete
			if a.showAutocomplete {
				a.showCompactionInfo = false
				a.showExitHint = false
			}
		case 'c':
			a.showCompactionInfo = !a.showCompactionInfo
			if a.showCompactionInfo {
				a.showAutocomplete = false
				a.showExitHint = false
			}
		case 'e':
			a.showExitHint = !a.showExitHint
			if a.showExitHint {
				a.showAutocomplete = false
				a.showCompactionInfo = false
			}
		case '+':
			a.toolCount++
		case '-':
			if a.toolCount > 0 {
				a.toolCount--
			}
		case 'x':
			a.expandedMsg = !a.expandedMsg
		}
	case tui.TickEvent:
		a.frame = e.Frame
	}
	return nil
}

func (a *diveLikeApp) buildLiveView() tui.View {
	views := make([]tui.View, 0)

	for i := 0; i < a.toolCount; i++ {
		label := fmt.Sprintf("Tool call %d", i+1)
		if a.expandedMsg {
			label = fmt.Sprintf("Tool call %d: longer description line", i+1)
		}
		views = append(views, tui.Group(
			tui.Text("\u23fa").Fg(tui.ColorCyan),
			tui.Text(" %s", label),
		))
	}

	views = append(views, tui.Group(
		tui.Loading(a.frame).CharSet(tui.SpinnerBounce.Frames).Speed(6).Fg(tui.ColorCyan),
		tui.Text(" thinking").Animate(tui.Slide(3, tui.NewRGB(80, 80, 80), tui.NewRGB(80, 200, 220))),
		tui.Text(" (%s)", formatDuration(time.Second*23)).Hint(),
		tui.Text("  ").Hint(),
		tui.Text("esc to interrupt").Hint(),
	))

	if a.showTodos {
		views = append(views, a.todoListView())
	}

	if len(views) == 0 {
		return tui.Text("")
	}

	return tui.PaddingLTRB(1, 0, 1, 0, tui.Stack(views...).Gap(1))
}

func (a *diveLikeApp) todoListView() tui.View {
	return tui.Stack(
		tui.Text("Todos").Bold(),
		tui.Text("- Run tests"),
		tui.Text("- Update docs"),
		tui.Text("- Ship release"),
	).Gap(0)
}

func (a *diveLikeApp) dialogView() tui.View {
	return tui.Stack(
		tui.Text("Confirm action").Bold(),
		tui.Text("This is a dialog-like view that replaces the normal UI."),
		tui.Text("Press d to close.").Hint(),
	).Gap(0)
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	return fmt.Sprintf("%ds", seconds)
}

func main() {
	app := &diveLikeApp{
		showExitHint: true,
	}
	app.runner = tui.NewInlineApp(tui.InlineAppConfig{
		FPS: 20,
	})
	if err := app.runner.Run(app); err != nil {
		panic(err)
	}
}
