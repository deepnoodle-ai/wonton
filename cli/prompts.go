package cli

import (
	"github.com/deepnoodle-ai/wonton/tui"
)

// This file provides prompt helpers for interactive CLI commands.
// These wrap the tui package to provide simple blocking prompts.

// Select displays a selection prompt and returns the selected index.
//
// The user can navigate with arrow keys or j/k and select with Enter.
// Returns -1 and an error if cancelled or not interactive.
//
//	idx, err := ctx.Select("Choose environment:", "dev", "staging", "prod")
//	if err != nil {
//	    return err
//	}
//	env := []string{"dev", "staging", "prod"}[idx]
func (c *Context) Select(title string, options ...string) (int, error) {
	if !c.Interactive() {
		return -1, Error("interactive terminal required for selection prompts")
	}

	selected := 0
	done := false

	app := &selectPrompt{
		title:    title,
		options:  options,
		selected: &selected,
		done:     &done,
	}

	err := tui.Run(app,
		tui.WithAlternateScreen(false),
		tui.WithHideCursor(true),
	)
	if err != nil {
		return -1, err
	}

	if !done {
		return -1, Error("selection cancelled")
	}

	return selected, nil
}

// SelectString displays a selection prompt and returns the selected option.
//
// This is a convenience wrapper around Select that returns the string directly:
//
//	env, err := ctx.SelectString("Choose environment:", "dev", "staging", "prod")
func (c *Context) SelectString(title string, options ...string) (string, error) {
	idx, err := c.Select(title, options...)
	if err != nil {
		return "", err
	}
	if idx >= 0 && idx < len(options) {
		return options[idx], nil
	}
	return "", Error("invalid selection")
}

// Input displays a text input prompt and returns the entered text.
//
//	name, err := ctx.Input("Enter your name: ")
//	if err != nil {
//	    return err
//	}
func (c *Context) Input(prompt string) (string, error) {
	if !c.Interactive() {
		return "", Error("interactive terminal required for input prompts")
	}

	value := ""
	done := false

	app := &inputPrompt{
		prompt: prompt,
		value:  &value,
		done:   &done,
	}

	err := tui.Run(app,
		tui.WithAlternateScreen(false),
		tui.WithHideCursor(false),
	)
	if err != nil {
		return "", err
	}

	if !done {
		return "", Error("input cancelled")
	}

	return value, nil
}

// Confirm displays a yes/no confirmation prompt.
//
// Returns true if the user selected "Yes", false otherwise:
//
//	if confirmed, err := ctx.Confirm("Delete all files?"); err == nil && confirmed {
//	    // Proceed with deletion
//	}
func (c *Context) Confirm(message string) (bool, error) {
	idx, err := c.Select(message, "Yes", "No")
	if err != nil {
		return false, err
	}
	return idx == 0, nil
}

// selectPrompt implements tui.Application for selection prompts
type selectPrompt struct {
	title    string
	options  []string
	selected *int
	done     *bool
}

func (p *selectPrompt) View() tui.View {
	items := make([]tui.View, 0, len(p.options)+2)

	// Title
	items = append(items, tui.Text("%s", p.title).Bold())
	items = append(items, tui.Spacer().MinHeight(1))

	// Options
	for i, opt := range p.options {
		prefix := "  "
		if i == *p.selected {
			prefix = "> "
			items = append(items, tui.Text("%s%s", prefix, opt).Fg(tui.ColorCyan).Bold())
		} else {
			items = append(items, tui.Text("%s%s", prefix, opt))
		}
	}

	return tui.Stack(items...)
}

func (p *selectPrompt) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyArrowUp:
			if *p.selected > 0 {
				*p.selected--
			}
		case tui.KeyArrowDown:
			if *p.selected < len(p.options)-1 {
				*p.selected++
			}
		case tui.KeyEnter:
			*p.done = true
			return []tui.Cmd{tui.Quit()}
		case tui.KeyCtrlC, tui.KeyEscape:
			return []tui.Cmd{tui.Quit()}
		}
		switch e.Rune {
		case 'j':
			if *p.selected < len(p.options)-1 {
				*p.selected++
			}
		case 'k':
			if *p.selected > 0 {
				*p.selected--
			}
		case 'q':
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

// inputPrompt implements tui.Application for text input prompts
type inputPrompt struct {
	prompt string
	value  *string
	done   *bool
}

func (p *inputPrompt) View() tui.View {
	return tui.Stack(
		tui.Text("%s", p.prompt).Bold(),
		tui.Spacer().MinHeight(1),
		tui.Input(p.value).Width(40),
		tui.Spacer().MinHeight(1),
		tui.Text("Enter to submit, Esc to cancel").Dim(),
	)
}

func (p *inputPrompt) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyEnter:
			*p.done = true
			return []tui.Cmd{tui.Quit()}
		case tui.KeyCtrlC, tui.KeyEscape:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}
