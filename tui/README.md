# tui

Declarative terminal UI framework with layout engine, rich components, and event-driven runtime. Build responsive terminal applications using composable views, automatic layout, and single-threaded event handling.

## Usage Examples

### Basic Application

```go
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

type app struct {
	count int
}

func (a *app) View() tui.View {
	return tui.Stack(
		tui.Text("Counter: %d", a.count).Bold(),
		tui.Text("Press + to increment, - to decrement, q to quit").Dim(),
	).Gap(1).Padding(2)
}

func (a *app) HandleEvent(ev tui.Event) []tui.Cmd {
	switch e := ev.(type) {
	case tui.KeyEvent:
		switch e.Rune {
		case '+':
			a.count++
		case '-':
			a.count--
		case 'q':
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func main() {
	if err := tui.Run(&app{}); err != nil {
		log.Fatal(err)
	}
}
```

### List Selection

```go
type listApp struct {
	items    []string
	selected int
}

func (a *listApp) View() tui.View {
	return tui.Stack(
		tui.Text("Select an item:").Bold(),
		tui.Table([]tui.TableColumn{{Header: "Items"}}, &a.selected).
			Rows(func() [][]string {
				var rows [][]string
				for _, item := range a.items {
					rows = append(rows, []string{item})
				}
				return rows
			}()).
			Height(10),
		tui.Text("Selected: %s", a.items[a.selected]),
	).Gap(1)
}

func (a *listApp) HandleEvent(ev tui.Event) []tui.Cmd {
	if ev, ok := ev.(tui.KeyEvent); ok && ev.Rune == 'q' {
		return []tui.Cmd{tui.Quit()}
	}
	return nil
}
```

### Text Input Form

```go
type formApp struct {
	name     string
	email    string
	password string
	focused  int // 0=name, 1=email, 2=password
}

func (a *formApp) View() tui.View {
	return tui.Stack(
		tui.Text("Registration Form").Bold(),
		tui.Divider(),

		tui.Text("Name:"),
		tui.InputField(&a.name).
			ID("name").
			OnSubmit(func(s string) { a.focused = 1 }),

		tui.Text("Email:"),
		tui.InputField(&a.email).
			ID("email").
			OnSubmit(func(s string) { a.focused = 2 }),

		tui.Text("Password:"),
		tui.PasswordInput(&a.password).
			ID("password").
			OnSubmit(func(s string) { /* submit form */ }),

		tui.Spacer(),
		tui.Group(
			tui.Button("Submit", func() { /* submit */ }),
			tui.Button("Cancel", func() { /* cancel */ }),
		).Gap(2),
	).Gap(1).Padding(2)
}
```

### Table Display

```go
type tableApp struct {
	selected int
}

func (a *tableApp) View() tui.View {
	columns := []tui.TableColumn{
		{Header: "Name", Width: 15},
		{Header: "Age", Width: 5},
		{Header: "City", Width: 20},
	}
	rows := [][]string{
		{"Alice", "28", "New York"},
		{"Bob", "35", "San Francisco"},
		{"Carol", "42", "Boston"},
	}

	return tui.Stack(
		tui.Text("Employee Directory").Bold(),
		tui.Table(columns, &a.selected).
			Rows(rows).
			Bordered().
			Striped(),
	).Gap(1)
}
```

### Markdown Rendering

```go
type docApp struct {
	markdown string
	scrollY  int
}

func (a *docApp) View() tui.View {
	return tui.Padding(2,
		tui.Markdown(a.markdown, &a.scrollY),
	)
}

func main() {
	content := `# Welcome

## Features
- **Bold** and *italic* text
- Code blocks with syntax highlighting
- Lists and tables
- Links and images

## Code Example
\x60\x60\x60go
func main() {
    fmt.Println("Hello, World!")
}
\x60\x60\x60
`

	tui.Run(&docApp{markdown: content})
}
```

### Layout with Stack and Group

```go
func (a *app) View() tui.View {
	return tui.Stack(
		// Header
		tui.Text("Dashboard").Bold().Fg(tui.ColorCyan),
		tui.Divider(),

		// Main content area (horizontal)
		tui.Group(
			// Left sidebar
			tui.Width(20,
				tui.Stack(
					tui.Text("Menu").Bold(),
					tui.Table([]tui.TableColumn{{Header: ""}}, &a.menuIdx).
						Rows([][]string{{"Home"}, {"Profile"}, {"Settings"}}),
				).Bordered(),
			),

			// Main panel
			tui.Stack(
				tui.Text("Content Area"),
				a.renderContent(),
			).Flex(1).Bordered(),

			// Right sidebar
			tui.Width(20,
				tui.Stack(
					tui.Text("Info").Bold(),
					tui.Text("Status: Active").Dim(),
				).Bordered(),
			),
		).Gap(1).Flex(1),

		// Footer
		tui.Divider(),
		tui.Text("Press q to quit").Dim(),
	)
}
```

### Animated Text Effects

```go
func (a *app) View() tui.View {
	return tui.Stack(
		// Rainbow color cycling
		tui.Text("Welcome to the App!").Animate(Rainbow(3)),

		// Pulsing alert
		tui.Text("ALERT").Animate(Pulse(tui.NewRGB(255, 0, 0), 12)),

		// Wave effect
		tui.Text("Status: Connected").Animate(Wave(12,
			tui.NewRGB(50, 150, 255),
			tui.NewRGB(100, 200, 255),
		)),

		// Sliding highlight
		tui.Text("Processing...").Animate(Slide(2,
			tui.NewRGB(100, 100, 100),
			tui.NewRGB(255, 255, 255),
		)),

		// Sparkle effect
		tui.Text("✨ Special ✨").Animate(Sparkle(3,
			tui.NewRGB(180, 180, 220),
			tui.NewRGB(255, 255, 255),
		)),

		// Typewriter effect
		tui.Text("Loading data...").Animate(Typewriter(3,
			tui.NewRGB(0, 255, 100),
			tui.NewRGB(255, 255, 255),
		).WithLoop(true)),

		// Glitch effect
		tui.Text("SIGNAL_LOST").Animate(Glitch(2,
			tui.NewRGB(255, 0, 100),
			tui.NewRGB(0, 255, 255),
		)),

		// Reversed rainbow with custom animation configuration
		tui.Text("Reversed!").Animate(Rainbow(3).Reverse()),
	).Gap(1)
}
```

### Semantic Text Styles

```go
func (a *app) View() tui.View {
	return tui.Stack(
		tui.Text("Operation completed successfully").Success(),
		tui.Text("Failed to connect to server").Error(),
		tui.Text("Disk space is running low").Warning(),
		tui.Text("Indexing 1,234 files...").Info(),
		tui.Text("Optional configuration available").Muted(),
		tui.Text("Press Enter to continue").Hint(),
	).Gap(1)
}
```

### Custom Canvas Drawing

```go
func (a *app) View() tui.View {
	return tui.CanvasContext(func(ctx *tui.RenderContext) {
		w, h := ctx.Size()
		frame := ctx.Frame()

		// Animated moving block
		x := int(frame) % w
		y := h / 2
		ctx.SetCell(x, y, '█', tui.NewStyle().WithForeground(tui.ColorCyan))

		// Draw border
		for i := 0; i < w; i++ {
			ctx.SetCell(i, 0, '─', tui.NewStyle())
			ctx.SetCell(i, h-1, '─', tui.NewStyle())
		}
		for i := 0; i < h; i++ {
			ctx.SetCell(0, i, '│', tui.NewStyle())
			ctx.SetCell(w-1, i, '│', tui.NewStyle())
		}
	})
}
```

### Async Commands

```go
func (a *app) HandleEvent(ev tui.Event) []tui.Cmd {
	switch e := ev.(type) {
	case tui.KeyEvent:
		if e.Rune == 'r' {
			// Trigger async refresh
			return []tui.Cmd{a.fetchData()}
		}
	case dataLoadedEvent:
		// Handle async result
		a.data = e.data
	}
	return nil
}

type dataLoadedEvent struct {
	data []string
}

func (a *app) fetchData() tui.Cmd {
	return func() tui.Event {
		// This runs in a goroutine
		time.Sleep(1 * time.Second)
		data := []string{"Item 1", "Item 2", "Item 3"}
		return dataLoadedEvent{data: data}
	}
}
```

### Periodic Updates

```go
type clockApp struct {
	time string
}

func (a *clockApp) View() tui.View {
	return tui.Text("Current time: %s", a.time).Bold()
}

func (a *clockApp) HandleEvent(ev tui.Event) []tui.Cmd {
	switch ev.(type) {
	case tui.TickEvent:
		a.time = time.Now().Format("15:04:05")
	case tui.KeyEvent:
		return []tui.Cmd{tui.Quit()}
	}
	return nil
}

func main() {
	// Run at 1 FPS for clock updates
	tui.Run(&clockApp{}, tui.WithFPS(1))
}
```

### Progress Indicators

```go
type app struct {
	progress int
	frame    uint64
	status   string
}

func (a *app) HandleEvent(event tui.Event) []tui.Cmd {
	if tick, ok := event.(tui.TickEvent); ok {
		a.frame = tick.Frame
	}
	return nil
}

func (a *app) View() tui.View {
	return tui.Stack(
		tui.Text("Download Progress").Bold(),

		tui.Progress(a.progress, 100).
			Width(40).
			ShowPercent(),

		tui.Loading(a.frame).Label("Loading..."),

		tui.Text("Status: %s", a.status).Dim(),
	).Gap(1)
}
```

## API Reference

### Application Types

| Type                  | Description                                  |
| --------------------- | -------------------------------------------- |
| `Application`         | Main interface for full-screen UI apps       |
| `InlineApplication`   | Interface for inline apps (LiveView method)  |
| `EventHandler`        | Optional interface for handling events       |
| `Initializable`       | Optional interface for initialization        |
| `Destroyable`         | Optional interface for cleanup               |
| `View`                | Core interface for UI components             |
| `RenderContext`       | Drawing context with animation frame         |

### Runtime Functions

| Function            | Description                | Inputs                                   | Outputs         |
| ------------------- | -------------------------- | ---------------------------------------- | --------------- |
| `Run`               | Starts full-screen app     | `app Application, opts ...RuntimeOption` | `error`         |
| `NewInlineApp`      | Creates inline app runner  | `opts ...InlineOption`                   | `*InlineApp`    |
| `RunInline`         | Convenience inline runner  | `app any, opts ...InlineOption`          | `error`         |
| `WithFPS`           | Sets frame rate            | `fps int`                                | `RuntimeOption` |
| `WithMouseTracking` | Enables mouse support      | `enabled bool`                           | `RuntimeOption` |
| `Quit`              | Returns quit command       | none                                     | `Cmd`           |

### Layout Views

| Function | Description             | Inputs             | Outputs       |
| -------- | ----------------------- | ------------------ | ------------- |
| `Stack`  | Vertical stack layout   | `children ...View` | `*stack`      |
| `Group`  | Horizontal stack layout | `children ...View` | `*group`      |
| `ZStack` | Layered stack layout    | `children ...View` | `*zStack`     |
| `Spacer` | Flexible spacing        | none               | `*spacerView` |
| `Empty`  | Empty view              | none               | `View`        |

**Flex Inheritance**: Stack and Group containers automatically inherit flexibility from their children. If a container holds flexible views (like Canvas or Spacer), the container itself becomes flexible without needing an explicit `.Flex()` call. This enables intuitive nested layouts:

```go
// Canvas expands because Group inherits its flexibility
Stack(
    Text("Header"),
    Group(Canvas()),  // Group auto-inherits flex from Canvas
    Text("Footer"),
)
```

### Text Views

| Function   | Description       | Inputs                                        | Outputs          |
| ---------- | ----------------- | --------------------------------------------- | ---------------- |
| `Text`     | Formatted text    | `format string, args ...interface{}`          | `*textView`      |
| `Markdown` | Markdown renderer | `content string, scrollY *int`                | `*markdownView`  |
| `Code`     | Syntax highlight  | `code string, language string`                | `*codeView`      |
| `DiffView` | Diff display      | `diff *Diff, language string, scrollY *int`   | `*diffView`      |

### Input Views

| Function        | Description           | Inputs          | Outputs               |
| --------------- | --------------------- | --------------- | --------------------- |
| `InputField`    | Text input            | `value *string` | `*inputFieldView`     |
| `PasswordInput` | Password input        | `value *string` | `*passwordInputView`  |
| `TextArea`      | Multi-line text input | `value *string` | `*textAreaView`       |

### Interactive Views

| Function     | Description          | Inputs                         | Outputs          |
| ------------ | -------------------- | ------------------------------ | ---------------- |
| `Button`     | Keyboard button      | `label string, onClick func()` | `*buttonView`    |
| `Clickable`  | Mouse-only clickable | `label string, onClick func()` | `*clickableView` |

### Display Views

| Function   | Description        | Inputs                                       | Outputs          |
| ---------- | ------------------ | -------------------------------------------- | ---------------- |
| `Table`    | Data table         | `columns []TableColumn, selected *int`       | `*tableView`     |
| `Tree`     | Hierarchical tree  | `root *TreeNode`                             | `*treeView`      |
| `Progress` | Progress indicator | `current, total int`                         | `*progressView`  |
| `Loading`  | Loading spinner    | `frame uint64`                               | `*loadingView`   |
| `Divider`  | Horizontal line    | none                                         | `*dividerView`   |

### Container/Modifier Views

| Function    | Description          | Inputs                           | Outputs           |
| ----------- | -------------------- | -------------------------------- | ----------------- |
| `Bordered`  | Border container     | `inner View`                     | `*borderedView`   |
| `Padding`   | Padding container    | `n int, inner View`              | `View`            |
| `PaddingHV` | H/V padding          | `h, v int, inner View`           | `View`            |
| `Width`     | Fixed width          | `w int, inner View`              | `View`            |
| `Height`    | Fixed height         | `h int, inner View`              | `View`            |
| `MaxWidth`  | Maximum width        | `w int, inner View`              | `View`            |
| `MinWidth`  | Minimum width        | `w int, inner View`              | `View`            |
| `Scroll`    | Scrollable container | `inner View, scrollY *int`       | `*scrollView`     |

**borderedView methods**: `.Title(string)`, `.Border(*BorderStyle)`, `.BorderFg(Color)`, `.FocusBorderFg(Color)`, `.TitleStyle(Style)`

### Custom Drawing

| Function        | Description           | Inputs                                               | Outputs       |
| --------------- | --------------------- | ---------------------------------------------------- | ------------- |
| `Canvas`        | Custom drawing area   | `draw func(frame RenderFrame, bounds Rectangle)`     | `*canvasView` |
| `CanvasContext` | Canvas with context   | `draw func(ctx *RenderContext)`                      | `*canvasView` |

### Collection Views

| Function   | Description             | Inputs                                        | Outputs          |
| ---------- | ----------------------- | --------------------------------------------- | ---------------- |
| `ForEach`  | Map items to views      | `items []T, mapper func(T, int) View`         | `*forEachView`   |
| `HForEach` | Horizontal map          | `items []T, mapper func(T, int) View`         | `*hForEachView`  |

### Conditional Views

| Function  | Description             | Inputs                                | Outputs |
| --------- | ----------------------- | ------------------------------------- | ------- |
| `If`      | Conditional rendering   | `condition bool, view View`           | `View`  |
| `IfElse`  | Conditional with else   | `condition bool, then, else View`     | `View`  |
| `Switch`  | Multi-way conditional   | `value T, cases ...CaseView[T]`       | `View`  |

### View Modifiers

Views support fluent modifier methods:

| Modifier          | Description                    | Example                                     |
| ----------------- | ------------------------------ | ------------------------------------------- |
| `.Width(int)`     | Sets fixed width (on TextView) | `tui.Text("Hello").Width(20)`               |
| `.Height(int)`    | Sets fixed height              | `tui.Text("Hi").Height(10)`                 |
| `.MaxWidth(int)`  | Sets maximum width             | `tui.Text(long).MaxWidth(80)`               |
| `.Flex(int)`      | Sets flex factor               | `tui.Stack(...).Flex(1)`                    |
| `.Bordered()`     | Adds border                    | `tui.Stack(...).Bordered()`                 |
| `.Padding(int)`   | Adds padding (method on stack) | `tui.Stack(...).Padding(2)`                 |
| `.Gap(int)`       | Sets spacing (Stack/Group)     | `tui.Stack(...).Gap(1)`                     |
| `.Align(align)`   | Sets alignment (Stack/Group)   | `tui.Stack(...).Align(tui.AlignCenter)`     |
| `.ID(string)`     | Sets focus ID (inputs)         | `tui.InputField(&s).ID("name")`             |

### Text Style Modifiers

| Modifier            | Description                 |
| ------------------- | --------------------------- |
| `.Bold()`           | Bold text                   |
| `.Dim()`            | Dimmed text                 |
| `.Italic()`         | Italic text                 |
| `.Underline()`      | Underlined text             |
| `.Strikethrough()`  | Strikethrough text          |
| `.Blink()`          | Blinking text               |
| `.Reverse()`        | Reverse video (swap fg/bg)  |
| `.Fg(Color)`        | Sets foreground color       |
| `.Bg(Color)`        | Sets background color       |
| `.FgRGB(r, g, b)`   | Sets foreground RGB color   |
| `.BgRGB(r, g, b)`   | Sets background RGB color   |
| `.Wrap()`           | Enable text wrapping        |
| `.Center()`         | Center align text           |
| `.Right()`          | Right align text            |
| `.FillBg()`         | Fill background with color  |

### Semantic Style Modifiers

| Modifier     | Description      |
| ------------ | ---------------- |
| `.Success()` | Green bold text  |
| `.Error()`   | Red bold text    |
| `.Warning()` | Yellow bold text |
| `.Info()`    | Cyan text        |
| `.Muted()`   | Dim gray text    |
| `.Hint()`    | Dim italic text  |

### Animation Modifiers

Apply animations using `.Animate(animation)` with animation constructors:

| Animation Constructor             | Description         | Parameters                         | Chainable Methods                          |
| --------------------------------- | ------------------- | ---------------------------------- | ------------------------------------------ |
| `Rainbow(speed)`                  | Rainbow color cycle | `speed int`                        | `.Reverse()`, `.WithLength(int)`           |
| `Pulse(color, speed)`             | Pulsing brightness  | `color RGB, speed int`             | `.Brightness(min, max float64)`            |
| `Wave(speed, colors...)`          | Wave color effect   | `speed int, colors ...RGB`         | `.WithAmplitude(float64)`                  |
| `Slide(speed, base, highlight)`   | Sliding highlight   | `speed int, base, highlight RGB`   | `.Reversed()`, `.WithWidth(int)`           |
| `Sparkle(speed, base, spark)`     | Sparkle effect      | `speed int, base, spark RGB`       | `.WithDensity(int)`                        |
| `Typewriter(speed, text, cursor)` | Typewriter reveal   | `speed int, text, cursor RGB`      | `.WithLoop(bool)`, `.WithHoldFrames(int)`  |
| `Glitch(speed, base, glitch)`     | Glitch effect       | `speed int, base, glitch RGB`      | `.WithIntensity(int)`                      |

Example:
```go
tui.Text("Hello").Animate(Rainbow(3))
tui.Text("Hello").Animate(Rainbow(3).Reverse())
tui.Text("Alert").Animate(Pulse(tui.NewRGB(255, 0, 0), 10).Brightness(0.3, 1.0))
```

### Event Types

| Event         | Description     | Fields                                             |
| ------------- | --------------- | -------------------------------------------------- |
| `KeyEvent`    | Keyboard input  | `Rune rune, Key Key, Modifiers KeyModifier`        |
| `MouseEvent`  | Mouse input     | `X, Y int, Button MouseButton, Action MouseAction` |
| `TickEvent`   | Frame tick      | `Frame uint64`                                     |
| `ResizeEvent` | Terminal resize | `Width, Height int`                                |
| `ErrorEvent`  | Error occurred  | `Err error`                                        |
| `QuitEvent`   | Quit requested  | none                                               |

### Command Types

| Type         | Description                                  |
| ------------ | -------------------------------------------- |
| `Cmd`        | Function returning Event (runs async)        |
| `Quit()`     | Exits application                            |
| `Tick(dur)`  | Creates a timer command                      |
| `After(dur)` | Executes function after duration             |
| `Batch(...)` | Executes multiple commands                   |
| `Sequence()` | Executes commands sequentially               |

## Architecture

The TUI framework uses a declarative, single-threaded event model:

1. **View Tree**: `Application.View()` returns an immutable view tree describing the UI
2. **Event Loop**: Events are processed sequentially in a single goroutine
3. **No Locks**: Application code never needs synchronization
4. **Commands**: Async operations run in background goroutines and send results as events
5. **Render**: After each event, `View()` is called and the tree is rendered

## Runtime Options

```go
tui.Run(app,
	tui.WithFPS(60),                    // 60 frames per second
	tui.WithMouseTracking(true),        // Enable mouse support
	tui.WithAlternateScreen(true),      // Use alternate screen buffer (default)
	tui.WithHideCursor(true),           // Hide cursor during rendering (default)
	tui.WithBracketedPaste(true),       // Enable bracketed paste mode
	tui.WithPasteTabWidth(4),           // Convert tabs to spaces in paste
)
```

## Snapshot Testing

The tui package includes a comprehensive snapshot (golden) testing system for verifying rendered output. This approach captures the exact visual output of views and compares against saved snapshots.

### Writing Snapshot Tests

```go
func TestGolden_MyComponent(t *testing.T) {
    // Build your view
    view := Stack(
        Text("Header").Bold(),
        Divider(),
        Text("Content"),
    )

    // Render to a virtual screen with specific dimensions
    screen := SprintScreen(view, WithWidth(30), WithHeight(10))

    // Assert against saved snapshot
    termtest.AssertScreen(t, screen)
}
```

### Test Organization

Tests are organized by feature in `golden_test.go` with clear section headers:

```go
// =============================================================================
// MY COMPONENT TESTS - Description of what's being tested
// =============================================================================

func TestGolden_MyComponent_BasicUsage(t *testing.T) { ... }
func TestGolden_MyComponent_EdgeCase(t *testing.T) { ... }
```

### Running Tests

```bash
# Run all golden tests
go test ./tui -run "TestGolden"

# Run specific test category
go test ./tui -run "TestGolden_UI"

# Run with verbose output
go test ./tui -run "TestGolden" -v
```

### Creating and Updating Snapshots

When you add new tests or intentionally change rendering behavior:

```bash
# Create/update snapshots (flag must come AFTER package)
go test ./tui -run "TestGolden_MyComponent" -update

# Update all snapshots
go test ./tui -run "TestGolden" -update
```

Snapshots are stored in `testdata/snapshots/` as `.snap` files.

### Reviewing Tests

Use the `reviewtests` tool to review test code alongside snapshots:

```bash
# Review all tests containing "Flex"
go run ./tui/cmd/reviewtests Flex

# Review specific test
go run ./tui/cmd/reviewtests MyComponent

# Review all UI tests
go run ./tui/cmd/reviewtests UI
```

The tool displays each test's code and its expected snapshot output for easy verification.

### Best Practices

1. **Descriptive Names**: Use `TestGolden_Category_Scenario` naming convention
2. **Explicit Dimensions**: Always specify `WithWidth()` and `WithHeight()` for consistent snapshots
3. **Comments**: Add comments explaining what the test verifies
4. **Edge Cases**: Test narrow widths, empty containers, and boundary conditions
5. **Real-World Patterns**: Create tests inspired by actual UI patterns from `examples/`

### Example: Complex UI Test

```go
func TestGolden_UI_Dashboard(t *testing.T) {
    // Dashboard with header, panels, and footer
    view := Stack(
        HeaderBar("Dashboard"),
        Group(
            Bordered(Text("Panel A")).Border(&RoundedBorder).Title("Left"),
            Bordered(Text("Panel B")).Border(&RoundedBorder).Title("Right"),
        ).Gap(1),
        StatusBar("Ready"),
    )
    screen := SprintScreen(view, WithWidth(50), WithHeight(10))
    termtest.AssertScreen(t, screen)
}
```

## Non-Interactive Printing

For CLI tools that want to display styled output without taking over the screen, use `Print()`:

```go
view := tui.Stack(
	tui.Text("Success!").Success(),
	tui.Text("Operation completed").Dim(),
).Gap(1)

// Print to stdout inline (no alternate screen, no event loop)
tui.Print(view)

// Or render to a string
output := tui.Sprint(view, tui.WithWidth(80))
fmt.Println(output)
```

The `Print` family of functions renders views without:
- Enabling alternate screen mode
- Starting an event loop
- Handling keyboard input
- Clearing the screen

This is perfect for command-line tools that want rich formatting without a full TUI.

## Inline Applications

For applications that need both scrollback output and live updating regions, use `InlineApp`. This is ideal for chat interfaces, build tools with logs, REPLs, and similar applications.

### Basic InlineApp

```go
type CounterApp struct {
    runner *tui.InlineApp
    count  int
}

func (app *CounterApp) LiveView() tui.View {
    return tui.Stack(
        tui.Divider(),
        tui.Text(" Count: %d", app.count).Bold(),
        tui.Text(" Press +/- to change, q to quit").Dim(),
        tui.Divider(),
    )
}

func (app *CounterApp) HandleEvent(event tui.Event) []tui.Cmd {
    if key, ok := event.(tui.KeyEvent); ok {
        switch key.Rune {
        case '+':
            app.count++
            app.runner.Printf("Incremented to %d", app.count)
        case '-':
            app.count--
            app.runner.Printf("Decremented to %d", app.count)
        case 'q':
            return []tui.Cmd{tui.Quit()}
        }
    }
    return nil
}

func main() {
    app := &CounterApp{}
    app.runner = tui.NewInlineApp()
    if err := app.runner.Run(app); err != nil {
        log.Fatal(err)
    }
}
```

### InlineApp vs Run

| Feature | `tui.Run()` | `tui.InlineApp` |
|---------|-------------|-----------------|
| Screen mode | Alternate (full screen) | Inline (coexists with scrollback) |
| Interface | `View()` | `LiveView()` |
| Output | View-only | `Print()` to scrollback + live region |
| Use case | Full TUI applications | Chat, logs, REPLs |
| Terminal history | Cleared on exit | Preserved |

### InlineApp Options

```go
runner := tui.NewInlineApp(
    tui.WithInlineWidth(80),              // Set rendering width
    tui.WithInlineFPS(30),                // Enable tick events for animations
    tui.WithInlineMouseTracking(true),    // Enable mouse events
    tui.WithInlineBracketedPaste(true),   // Enable bracketed paste mode
    tui.WithInlineKittyKeyboard(true),    // Enable Kitty keyboard protocol
    tui.WithInlinePasteTabWidth(4),       // Convert tabs in pasted text
)
```

### InlineApp Architecture

InlineApp uses the same three-goroutine architecture as `Run()`:

1. **Event loop**: Processes events sequentially, calls `HandleEvent` and `LiveView`
2. **Input reader**: Reads from stdin, sends key/mouse events
3. **Command executor**: Runs async `Cmd` functions

This design ensures:
- No race conditions in application code
- `HandleEvent` and `LiveView` are never called concurrently
- No locks needed in your application

### Rendering Optimization

InlineApp minimizes flicker using:

- **Line-level diffing**: Only changed lines are redrawn
- **Synchronized output mode**: Terminal buffers changes and renders atomically (DEC 2026)
- **Atomic Print**: Scrollback output and live region re-render happen as one operation

## Related Packages

- [terminal](../terminal) - Low-level terminal control and ANSI sequences
- [termsession](../termsession) - Terminal session recording and playback
- [termtest](../termtest) - Terminal output testing with screen simulation
