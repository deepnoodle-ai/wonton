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
	).Gap(1)
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
	if err := tui.Run(&app{}, tui.WithFPS(30)); err != nil {
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
		tui.SelectListStrings(a.items, &a.selected).
			OnSelect(func(idx int) {
				a.selected = idx
			}).
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
			Focused(a.focused == 0).
			OnSubmit(func(s string) { a.focused = 1 }),

		tui.Text("Email:"),
		tui.InputField(&a.email).
			Focused(a.focused == 1).
			OnSubmit(func(s string) { a.focused = 2 }),

		tui.Text("Password:"),
		tui.PasswordInput(&a.password).
			Focused(a.focused == 2).
			OnSubmit(func(s string) { /* submit form */ }),

		tui.Spacer(),
		tui.HStack(
			tui.Button("Submit", func() { /* submit */ }).Primary(),
			tui.Button("Cancel", func() { /* cancel */ }),
		).Gap(2),
	).Gap(1).Padding(2)
}
```

### Table Display

```go
type tableApp struct{}

func (a *tableApp) View() tui.View {
	headers := []string{"Name", "Age", "City"}
	rows := [][]string{
		{"Alice", "28", "New York"},
		{"Bob", "35", "San Francisco"},
		{"Carol", "42", "Boston"},
	}

	return tui.Stack(
		tui.Text("Employee Directory").Bold(),
		tui.Table(headers, rows).
			Border(true).
			Striped(true),
	).Gap(1)
}
```

### Markdown Rendering

```go
type docApp struct {
	markdown string
}

func (a *docApp) View() tui.View {
	return tui.Stack(
		tui.Markdown(a.markdown).Padding(2),
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
		tui.Text("Dashboard").Bold().Color(tui.ColorCyan),
		tui.Divider(),

		// Main content area (horizontal)
		tui.Group(
			// Left sidebar
			tui.Stack(
				tui.Text("Menu").Bold(),
				tui.SelectListStrings([]string{"Home", "Profile", "Settings"}, &a.menuIdx),
			).Width(20).Border(true),

			// Main panel
			tui.Stack(
				tui.Text("Content Area"),
				a.renderContent(),
			).Flex(1).Border(true),

			// Right sidebar
			tui.Stack(
				tui.Text("Info").Bold(),
				tui.Text("Status: Active").Dim(),
			).Width(20).Border(true),
		).Flex(1),

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
		tui.Text("Welcome to the App!").Rainbow(3),

		// Pulsing alert
		tui.Text("ALERT").Pulse(tui.NewRGB(255, 0, 0), 12),

		// Wave effect
		tui.Text("Status: Connected").Wave(12,
			tui.NewRGB(50, 150, 255),
			tui.NewRGB(100, 200, 255),
		),

		// Sliding highlight
		tui.Text("Processing...").Slide(2,
			tui.NewRGB(100, 100, 100),
			tui.NewRGB(255, 255, 255),
		),

		// Sparkle effect
		tui.Text("✨ Special ✨").Sparkle(3,
			tui.NewRGB(180, 180, 220),
			tui.NewRGB(255, 255, 255),
		),

		// Typewriter effect
		tui.Text("Loading data...").Typewriter(3,
			tui.NewRGB(0, 255, 100),
			tui.NewRGB(255, 255, 255),
		),

		// Glitch effect
		tui.Text("SIGNAL_LOST").Glitch(2,
			tui.NewRGB(255, 0, 100),
			tui.NewRGB(0, 255, 255),
		),
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

| Type            | Description                            |
| --------------- | -------------------------------------- |
| `Application`   | Main interface for declarative UI apps |
| `EventHandler`  | Optional interface for handling events |
| `Initializable` | Optional interface for initialization  |
| `Destroyable`   | Optional interface for cleanup         |
| `View`          | Core interface for UI components       |
| `RenderContext` | Drawing context with animation frame   |

### Runtime Functions

| Function            | Description                | Inputs                                   | Outputs         |
| ------------------- | -------------------------- | ---------------------------------------- | --------------- |
| `Run`               | Starts application runtime | `app Application, opts ...RuntimeOption` | `error`         |
| `WithFPS`           | Sets frame rate            | `fps int`                                | `RuntimeOption` |
| `WithMouseTracking` | Enables mouse support      | `enabled bool`                           | `RuntimeOption` |
| `Quit`              | Returns quit command       | none                                     | `Cmd`           |

### Layout Views

| Function | Description             | Inputs             | Outputs       |
| -------- | ----------------------- | ------------------ | ------------- |
| `Stack`  | Vertical stack layout   | `children ...View` | `*StackView`  |
| `Group`  | Horizontal stack layout | `children ...View` | `*GroupView`  |
| `ZStack` | Layered stack layout    | `children ...View` | `*ZStackView` |
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

| Function   | Description       | Inputs                               | Outputs         |
| ---------- | ----------------- | ------------------------------------ | --------------- |
| `Text`     | Formatted text    | `format string, args ...interface{}` | `*TextView`     |
| `Markdown` | Markdown renderer | `content string`                     | `*MarkdownView` |

### Input Views

| Function        | Description           | Inputs          | Outputs              |
| --------------- | --------------------- | --------------- | -------------------- |
| `InputField`    | Text input            | `value *string` | `*InputFieldView`    |
| `PasswordInput` | Password input        | `value *string` | `*PasswordInputView` |
| `TextArea`      | Multi-line text input | `value *string` | `*TextAreaView`      |

### Selection Views

| Function            | Description      | Inputs                               | Outputs           |
| ------------------- | ---------------- | ------------------------------------ | ----------------- |
| `SelectList`        | Selectable list  | `items []interface{}, selected *int` | `*SelectListView` |
| `SelectListStrings` | String list      | `items []string, selected *int`      | `*SelectListView` |
| `Button`            | Clickable button | `label string, onClick func()`       | `*ButtonView`     |

### Display Views

| Function         | Description        | Inputs                              | Outputs         |
| ---------------- | ------------------ | ----------------------------------- | --------------- |
| `Table`          | Data table         | `headers []string, rows [][]string` | `*TableView`    |
| `FilterableList` | Scrollable list    | `items []string`                    | `*ListView`     |
| `Tree`           | Hierarchical tree  | `root *TreeNode`                    | `*TreeView`     |
| `Progress`       | Progress indicator | `value, max int`                    | `*ProgressView` |
| `Loading`        | Loading spinner    | none                                | `*LoadingView`  |
| `Divider`        | Horizontal line    | none                                | `*DividerView`  |

### Container Views

| Function     | Description          | Inputs                    | Outputs        |
| ------------ | -------------------- | ------------------------- | -------------- |
| `Border`     | Border container     | `child View`              | `*BorderView`  |
| `Padding`    | Padding container    | `child View, padding int` | `*PaddingView` |
| `ScrollView` | Scrollable container | `child View`              | `*ScrollView`  |

### Custom Drawing

| Function        | Description         | Inputs                          | Outputs       |
| --------------- | ------------------- | ------------------------------- | ------------- |
| `Canvas`        | Custom drawing area | `draw func(ctx *RenderContext)` | `*CanvasView` |
| `CanvasContext` | Canvas with context | `draw func(ctx *RenderContext)` | `*CanvasView` |

### View Modifiers

Most views support fluent modifier methods:

| Modifier         | Description                | Example                            |
| ---------------- | -------------------------- | ---------------------------------- |
| `.Width(int)`    | Sets fixed width           | `tui.Text("Hello").Width(20)`      |
| `.Height(int)`   | Sets fixed height          | `tui.Stack(...).Height(10)`        |
| `.MinWidth(int)` | Sets minimum width         | `tui.InputField(&s).MinWidth(30)`  |
| `.MaxWidth(int)` | Sets maximum width         | `tui.Text(long).MaxWidth(80)`      |
| `.Flex(int)`     | Sets flex factor           | `tui.Stack(...).Flex(1)`           |
| `.Border(bool)`  | Adds border                | `tui.Stack(...).Border(true)`      |
| `.Padding(int)`  | Adds padding               | `tui.Text("Hi").Padding(2)`        |
| `.Gap(int)`      | Sets spacing (Stack/Group) | `tui.Stack(...).Gap(1)`            |
| `.Centered()`    | Centers content            | `tui.Text("Title").Centered()`     |
| `.Focused(bool)` | Sets focus state           | `tui.InputField(&s).Focused(true)` |

### Text Style Modifiers

| Modifier          | Description           |
| ----------------- | --------------------- |
| `.Bold()`         | Bold text             |
| `.Dim()`          | Dimmed text           |
| `.Italic()`       | Italic text           |
| `.Underline()`    | Underlined text       |
| `.Color(Color)`   | Sets foreground color |
| `.BgColor(Color)` | Sets background color |

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

| Modifier                                | Description         | Parameters                         |
| --------------------------------------- | ------------------- | ---------------------------------- |
| `.Rainbow(speed)`                       | Rainbow color cycle | `speed int`                        |
| `.RainbowReverse(speed)`                | Reverse rainbow     | `speed int`                        |
| `.Pulse(color, speed)`                  | Pulsing brightness  | `color Color, speed int`           |
| `.Wave(speed, colors...)`               | Wave color effect   | `speed int, colors ...Color`       |
| `.Slide(speed, base, highlight)`        | Sliding highlight   | `speed int, base, highlight Color` |
| `.SlideReverse(speed, base, highlight)` | Reverse slide       | `speed int, base, highlight Color` |
| `.Sparkle(speed, base, highlight)`      | Sparkle effect      | `speed int, base, highlight Color` |
| `.Typewriter(speed, base, cursor)`      | Typewriter reveal   | `speed int, base, cursor Color`    |
| `.Glitch(speed, color1, color2)`        | Glitch effect       | `speed int, color1, color2 Color`  |

### Event Types

| Event             | Description     | Fields                                             |
| ----------------- | --------------- | -------------------------------------------------- |
| `KeyEvent`        | Keyboard input  | `Rune rune, Key Key, Modifiers KeyModifier`        |
| `MouseEvent`      | Mouse input     | `X, Y int, Button MouseButton, Action MouseAction` |
| `TickEvent`       | Frame tick      | `Frame uint64`                                     |
| `WindowSizeEvent` | Terminal resize | `Width, Height int`                                |

### Command Types

| Type     | Description                           |
| -------- | ------------------------------------- |
| `Cmd`    | Function returning Event (runs async) |
| `Quit()` | Exits application                     |
| `Tick`   | Periodic timer                        |

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
	tui.WithAlternateScreen(false),     // Disable alternate screen
	tui.WithRawMode(true),              // Enable raw mode
)
```

## Related Packages

- [terminal](../terminal) - Low-level terminal control and ANSI sequences
- [termsession](../termsession) - Terminal session recording and playback
- [termtest](../termtest) - Terminal output testing with screen simulation
