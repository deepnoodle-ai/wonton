// Package tui provides a declarative terminal user interface framework
// for building interactive command-line applications in Go.
//
// # Philosophy
//
// Wonton TUI is designed around declarative UI principles similar to SwiftUI
// and React. Applications describe what the UI should look like in terms of
// the current state, and the framework handles rendering and updates efficiently.
// This eliminates manual terminal manipulation and focus on application logic.
//
// # Quick Start
//
// A minimal TUI application implements the Application interface with a View()
// method that returns the UI tree:
//
//	type App struct {
//	    count int
//	}
//
//	func (a *App) View() tui.View {
//	    return tui.Stack(
//	        tui.Text("Count: %d", a.count),
//	        tui.Button("Increment", func() { a.count++ }),
//	    )
//	}
//
//	func (a *App) HandleEvent(event tui.Event) []tui.Cmd {
//	    if key, ok := event.(tui.KeyEvent); ok && key.Rune == 'q' {
//	        return []tui.Cmd{tui.Quit()}
//	    }
//	    return nil
//	}
//
//	func main() {
//	    tui.Run(&App{})
//	}
//
// # Core Concepts
//
// View Tree: The UI is built as a tree of View components. Containers like
// Stack, HStack, and ZStack arrange children, while leaf views like Text,
// Button, and Input display content.
//
// Event Loop: The Runtime manages a single-threaded event loop that processes
// events sequentially. This eliminates race conditions in application code -
// no locks needed for state management.
//
// Commands: Async operations are handled through the Cmd system. Commands
// execute in separate goroutines and send their results back as events.
//
// # Layout System
//
// Views are arranged in two phases:
//
//  1. Measurement: Each view calculates its preferred size given constraints
//  2. Rendering: Each view draws itself within its allocated space
//
// Layout containers (Stack, HStack, etc.) coordinate child sizing:
//
//	Stack(              // Vertical stack
//	    Text("Header"),
//	    Spacer(),       // Expands to fill space
//	    Text("Footer"),
//	).Gap(1)            // Space between children
//
// # View Components
//
// Text and Styling:
//
//	Text("Hello").Fg(ColorGreen).Bold()
//	Text("Error: %s", err).Error()  // Semantic styling
//
// Layout Containers:
//
//	Stack(children...)   // Vertical layout
//	HStack(children...)  // Horizontal layout
//	ZStack(children...)  // Layered (z-axis) layout
//
// Interactive Elements:
//
//	Button("Submit", func() { ... })      // Keyboard + mouse
//	Clickable("Link", func() { ... })     // Mouse-only
//	InputField(&app.name).Placeholder("Enter name")
//
// Modifiers:
//
//	Padding(2, content)              // Add space around view
//	Bordered(content).Title("Box")   // Draw border with optional title
//	Width(40, content)               // Fixed width
//	MaxWidth(80, content)            // Maximum width constraint
//
// # Event Handling
//
// Applications can optionally implement EventHandler to process events:
//
//	func (a *App) HandleEvent(event tui.Event) []tui.Cmd {
//	    switch e := event.(type) {
//	    case tui.KeyEvent:
//	        if e.Rune == 'q' {
//	            return []tui.Cmd{tui.Quit()}
//	        }
//	    case tui.ResizeEvent:
//	        // Handle terminal resize
//	    }
//	    return nil
//	}
//
// Event Types:
//   - KeyEvent: Keyboard input with modifiers
//   - MouseEvent: Mouse clicks, movement, scrolling
//   - TickEvent: Periodic timer for animations (based on FPS)
//   - ResizeEvent: Terminal size changed
//   - QuitEvent: Application should exit
//
// # Async Operations
//
// Long-running operations use the Cmd system to avoid blocking the UI:
//
//	func fetchData() tui.Cmd {
//	    return func() tui.Event {
//	        data, err := http.Get("...")
//	        return DataEvent{data, err}
//	    }
//	}
//
//	func (a *App) HandleEvent(event tui.Event) []tui.Cmd {
//	    if _, ok := event.(tui.KeyEvent); ok {
//	        return []tui.Cmd{fetchData()}
//	    }
//	    if data, ok := event.(DataEvent); ok {
//	        a.data = data
//	    }
//	    return nil
//	}
//
// # Focus Management
//
// Interactive elements like Button and InputField are automatically focusable.
// Users navigate with Tab/Shift+Tab, activate with Enter/Space:
//
//	Stack(
//	    InputField(&app.name).ID("name"),
//	    InputField(&app.email).ID("email"),
//	    Button("Submit", func() { app.submit() }),
//	)
//
// Views can respond to focus state:
//
//	Bordered(content).
//	    FocusID("name").
//	    FocusBorderFg(ColorCyan)
//
// # Mouse Support
//
// Mouse tracking is opt-in via WithMouseTracking:
//
//	tui.Run(&App{}, tui.WithMouseTracking(true))
//
// Applications receive MouseEvent events for clicks, movement, and scrolling.
// Interactive elements (Button, Clickable) automatically handle mouse clicks.
//
// # Animations
//
// Animations are driven by TickEvent, which fires at the configured FPS:
//
//	type App struct {
//	    frame uint64
//	}
//
//	func (a *App) HandleEvent(event tui.Event) []tui.Cmd {
//	    if tick, ok := event.(tui.TickEvent); ok {
//	        a.frame = tick.Frame
//	    }
//	    return nil
//	}
//
//	func (a *App) View() tui.View {
//	    return Text("Frame %d", a.frame)
//	}
//
// Text views support built-in animations:
//
//	Text("Rainbow").Rainbow(5)
//	Text("Pulse").Pulse(tui.NewRGB(255, 0, 0), 10)
//	Text("Type").Typewriter(3, textColor, cursorColor)
//
// # Advanced Features
//
// Collections: ForEach renders slices of data:
//
//	ForEach(app.items, func(item Item) tui.View {
//	    return Text(item.Name)
//	})
//
// Conditional Rendering: If/IfElse for dynamic UI:
//
//	If(app.showHelp, func() tui.View {
//	    return Text("Help text...")
//	})
//
// Scrolling: ScrollView for content larger than viewport:
//
//	ScrollView(&app.scrollY, content).Height(20)
//
// Tables: TableView for tabular data:
//
//	TableView(headers, rows).Border(&tui.SingleBorder)
//
// Markdown: MarkdownView for rich formatted text:
//
//	MarkdownView(mdContent).Width(80)
//
// Code Highlighting: CodeView with syntax highlighting:
//
//	CodeView(code, "go").Width(80)
//
// # Testing
//
// The terminal package provides TestTerminal for unit testing TUI applications:
//
//	func TestApp(t *testing.T) {
//	    term := tui.NewTestTerminal(80, 24)
//	    runtime := tui.NewRuntime(term, &App{}, 30)
//	    // Send events and verify output
//	}
//
// # Thread Safety
//
// The Runtime guarantees that View() and HandleEvent() are NEVER called
// concurrently. All state mutations happen in the single-threaded event loop,
// eliminating the need for locks in application code.
//
// Commands execute in separate goroutines but communicate back via events,
// maintaining the single-threaded guarantee for application logic.
//
// # Performance
//
// The framework uses double-buffering and dirty region tracking to minimize
// terminal I/O. Only changed cells are sent to the terminal, enabling smooth
// 60 FPS animations even in large UIs.
//
// # Package Organization
//
// This package re-exports types from the terminal package (Style, Color, etc.)
// for convenience. Applications typically only need to import "tui".
//
// Related packages:
//   - terminal: Low-level terminal control and rendering
//   - termtest: Testing utilities for TUI applications
//
// For detailed examples, see the examples directory in the repository.
package tui
