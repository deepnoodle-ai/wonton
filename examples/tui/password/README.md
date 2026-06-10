# Password Input Demo

This example demonstrates masked password input using the declarative View
style: an `Input` bound to a string with a mask character, so typed
characters render as `*`.

## Features Demonstrated

- **Masked input**: `tui.Input(&app.password).Mask('*')` displays a mask
  character instead of what was typed
- **Placeholder text**: shown before typing begins
- **Fixed input width**: `.Width(30)`
- **Submit/quit handling**: Enter submits, Escape or Ctrl+C quits

## Running the Demo

```bash
go run ./examples/tui/password
```

## Code Example

```go
func (app *PasswordDemoApp) View() tui.View {
    return tui.Stack(
        tui.Text("Password Input Demo"),
        tui.Group(
            tui.Text("Password: "),
            tui.Input(&app.password).Mask('*').Placeholder("enter password").Width(30),
        ),
        tui.Text("Enter to submit, Esc to quit"),
    )
}
```

The application holds the password in a plain `string` field bound to the
input. Submission is detected in `HandleEvent` on `tui.KeyEnter`; the view
then switches to a confirmation screen showing the password length without
echoing its contents.

## Notes

- Use a different mask rune (e.g. `.Mask('•')`) to change the displayed
  character.
- Because the focused input consumes printable keys, app-level key handling
  should rely on special keys (Enter, Escape, Ctrl+C) rather than letters.

## Related Examples

- `examples/tui/input_forms/` - Form with multiple inputs
- `examples/tui/input_styles/` - Input styling options
