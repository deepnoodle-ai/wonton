package cli

import (
	"github.com/deepnoodle-ai/wonton/terminal"
	"github.com/deepnoodle-ai/wonton/tui"
)

// ViewFunc creates a Wonton view for interactive display.
type ViewFunc func(*Context) tui.View

// RunInteractive runs an interactive Wonton application within a CLI command.
func RunInteractive(ctx *Context, app tui.Application, opts ...tui.RunOption) error {
	if !ctx.Interactive() {
		return Error("interactive terminal required").
			Hint("Run in a terminal with TTY support")
	}
	return tui.Run(app, opts...)
}

// InteractiveApp wraps a CLI context with Wonton for rich TUI.
type InteractiveApp struct {
	ctx     *Context
	viewFn  ViewFunc
	handler func(tui.Event, *Context) []tui.Cmd
}

// NewInteractiveApp creates an interactive app from a CLI context.
func NewInteractiveApp(ctx *Context, viewFn ViewFunc) *InteractiveApp {
	return &InteractiveApp{
		ctx:    ctx,
		viewFn: viewFn,
	}
}

// OnEvent sets the event handler.
func (a *InteractiveApp) OnEvent(h func(tui.Event, *Context) []tui.Cmd) *InteractiveApp {
	a.handler = h
	return a
}

// View implements tui.Application.
func (a *InteractiveApp) View() tui.View {
	return a.viewFn(a.ctx)
}

// HandleEvent implements tui.EventHandler.
func (a *InteractiveApp) HandleEvent(event tui.Event) []tui.Cmd {
	// Handle quit by default
	if key, ok := event.(tui.KeyEvent); ok {
		if key.Key == tui.KeyCtrlC || key.Rune == 'q' {
			return []tui.Cmd{tui.Quit()}
		}
	}

	if a.handler != nil {
		return a.handler(event, a.ctx)
	}
	return nil
}

// Run starts the interactive application.
func (a *InteractiveApp) Run(opts ...tui.RunOption) error {
	return tui.Run(a, opts...)
}

// Prompt shows a simple prompt and returns the selected option.
type Prompt struct {
	title   string
	options []string
	ctx     *Context
}

// NewPrompt creates a new selection prompt.
func NewPrompt(ctx *Context, title string, options ...string) *Prompt {
	return &Prompt{
		title:   title,
		options: options,
		ctx:     ctx,
	}
}

// Show displays the prompt and returns the selected index.
func (p *Prompt) Show() (int, error) {
	if !p.ctx.Interactive() {
		return -1, Error("interactive terminal required for prompts")
	}

	selected := 0
	done := false

	app := &promptApp{
		prompt:   p,
		selected: &selected,
		done:     &done,
	}

	err := tui.Run(app, tui.WithMouseTracking(true), tui.WithAlternateScreen(false))
	if err != nil {
		return -1, err
	}

	if !done {
		return -1, Error("selection cancelled")
	}

	return selected, nil
}

type promptApp struct {
	prompt   *Prompt
	selected *int
	done     *bool
}

func (a *promptApp) View() tui.View {
	items := make([]tui.View, 0, len(a.prompt.options)+2)

	// Title
	items = append(items, tui.Text("%s", a.prompt.title).Bold())
	items = append(items, tui.Spacer().MinHeight(1))

	// Options
	for i, opt := range a.prompt.options {
		prefix := "  "
		style := terminal.NewStyle()
		if i == *a.selected {
			prefix = "> "
			style = style.WithForeground(terminal.ColorCyan).WithBold()
		}
		items = append(items, tui.Text("%s%s", prefix, opt).Style(style))
	}

	return tui.Stack(items...)
}

func (a *promptApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyArrowUp:
			if *a.selected > 0 {
				*a.selected--
			}
		case tui.KeyArrowDown:
			if *a.selected < len(a.prompt.options)-1 {
				*a.selected++
			}
		case tui.KeyEnter:
			*a.done = true
			return []tui.Cmd{tui.Quit()}
		case tui.KeyCtrlC, tui.KeyEscape:
			return []tui.Cmd{tui.Quit()}
		}
		switch e.Rune {
		case 'j':
			if *a.selected < len(a.prompt.options)-1 {
				*a.selected++
			}
		case 'k':
			if *a.selected > 0 {
				*a.selected--
			}
		case 'q':
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

// InputPrompt shows an input prompt and returns the entered text.
type InputPrompt struct {
	label       string
	placeholder string
	value       string
	ctx         *Context
}

// NewInput creates a new text input prompt.
func NewInput(ctx *Context, label string) *InputPrompt {
	return &InputPrompt{
		label: label,
		ctx:   ctx,
	}
}

// Placeholder sets the placeholder text.
func (i *InputPrompt) Placeholder(text string) *InputPrompt {
	i.placeholder = text
	return i
}

// Default sets the default value.
func (i *InputPrompt) Default(text string) *InputPrompt {
	i.value = text
	return i
}

// Show displays the input prompt and returns the entered text.
func (i *InputPrompt) Show() (string, error) {
	if !i.ctx.Interactive() {
		return "", Error("interactive terminal required for input prompts")
	}

	done := false
	value := i.value

	app := &inputApp{
		input: i,
		value: &value,
		done:  &done,
	}

	err := tui.Run(app, tui.WithMouseTracking(true), tui.WithAlternateScreen(false))
	if err != nil {
		return "", err
	}

	if !done {
		return "", Error("input cancelled")
	}

	return value, nil
}

type inputApp struct {
	input *InputPrompt
	value *string
	done  *bool
}

func (a *inputApp) View() tui.View {
	return tui.Stack(
		tui.Text("%s", a.input.label).Bold(),
		tui.Spacer().MinHeight(1),
		tui.Input(a.value).Placeholder(a.input.placeholder).Width(40),
		tui.Spacer().MinHeight(1),
		tui.Text("Press Enter to submit, Esc to cancel").Dim(),
	)
}

func (a *inputApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyEnter:
			*a.done = true
			return []tui.Cmd{tui.Quit()}
		case tui.KeyCtrlC, tui.KeyEscape:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

// Confirm shows a confirmation prompt.
func ConfirmPrompt(ctx *Context, message string) (bool, error) {
	idx, err := NewPrompt(ctx, message, "Yes", "No").Show()
	if err != nil {
		return false, err
	}
	return idx == 0, nil
}

// Select shows a selection prompt and returns the selected option.
func Select(ctx *Context, title string, options ...string) (string, error) {
	idx, err := NewPrompt(ctx, title, options...).Show()
	if err != nil {
		return "", err
	}
	if idx >= 0 && idx < len(options) {
		return options[idx], nil
	}
	return "", Error("invalid selection")
}
