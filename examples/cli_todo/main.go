// Example: Todo List Manager with TUI
//
// Demonstrates building a CLI app with a rich TUI for data management:
// - CLI commands for quick operations (add, list, complete)
// - Full TUI mode for interactive task management
// - Data persistence simulation
// - Keyboard navigation and mouse support
//
// Run with:
//
//	go run examples/cli_todo/main.go --help
//	go run examples/cli_todo/main.go add "Buy groceries"
//	go run examples/cli_todo/main.go add "Write docs" --priority high
//	go run examples/cli_todo/main.go list
//	go run examples/cli_todo/main.go list --filter pending
//	go run examples/cli_todo/main.go complete 1
//	go run examples/cli_todo/main.go tui          # Full interactive mode
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/tui"
)

// Task represents a todo item
type Task struct {
	ID        int
	Title     string
	Priority  string // low, medium, high
	Done      bool
	CreatedAt time.Time
}

// Global task store (in real app, this would be persistent)
var tasks = []Task{
	{ID: 1, Title: "Set up project structure", Priority: "high", Done: true, CreatedAt: time.Now().Add(-24 * time.Hour)},
	{ID: 2, Title: "Write documentation", Priority: "medium", Done: false, CreatedAt: time.Now().Add(-12 * time.Hour)},
	{ID: 3, Title: "Add unit tests", Priority: "high", Done: false, CreatedAt: time.Now().Add(-6 * time.Hour)},
	{ID: 4, Title: "Review pull request", Priority: "low", Done: false, CreatedAt: time.Now().Add(-1 * time.Hour)},
}

func main() {
	app := cli.New("todo").
		Description("A todo list manager with TUI support").
		Version("1.0.0")

	// Global flags
	app.GlobalFlags(
		&cli.BoolFlag{Name: "json", Help: "Output as JSON"},
	)

	// Add a new task (CLI mode)
	app.Command("add").
		Description("Add a new task").
		Args("title").
		Flags(
			&cli.StringFlag{
				Name:  "priority",
				Short: "p",
				Help:  "Task priority",
				Value: "medium",
				Enum:  []string{"low", "medium", "high"},
			},
		).
		Run(func(ctx *cli.Context) error {
			title := ctx.Arg(0)
			if title == "" {
				return cli.Error("task title required")
			}

			priority := ctx.String("priority")
			task := Task{
				ID:        len(tasks) + 1,
				Title:     title,
				Priority:  priority,
				Done:      false,
				CreatedAt: time.Now(),
			}
			tasks = append(tasks, task)

			if ctx.Bool("json") {
				ctx.Printf(`{"id": %d, "title": "%s", "priority": "%s"}`+"\n",
					task.ID, task.Title, task.Priority)
			} else {
				ctx.Printf("Added task #%d: %s [%s]\n", task.ID, task.Title, task.Priority)
			}
			return nil
		})

	// List tasks (CLI mode)
	app.Command("list").
		Description("List all tasks").
		Alias("ls").
		Flags(
			&cli.StringFlag{
				Name:  "filter",
				Short: "f",
				Help:  "Filter tasks",
				Value: "all",
				Enum:  []string{"all", "pending", "done"},
			},
			&cli.StringFlag{
				Name:  "priority",
				Short: "p",
				Help:  "Filter by priority",
				Enum:  []string{"", "low", "medium", "high"},
			},
		).
		Run(func(ctx *cli.Context) error {
			filter := ctx.String("filter")
			priorityFilter := ctx.String("priority")

			filtered := filterTasks(tasks, filter, priorityFilter)

			if ctx.Bool("json") {
				ctx.Println("[")
				for i, t := range filtered {
					comma := ","
					if i == len(filtered)-1 {
						comma = ""
					}
					status := "pending"
					if t.Done {
						status = "done"
					}
					ctx.Printf(`  {"id": %d, "title": "%s", "priority": "%s", "status": "%s"}%s`+"\n",
						t.ID, t.Title, t.Priority, status, comma)
				}
				ctx.Println("]")
			} else {
				if len(filtered) == 0 {
					ctx.Println("No tasks found")
					return nil
				}

				ctx.Println("Tasks:")
				ctx.Println(strings.Repeat("-", 50))
				for _, t := range filtered {
					checkbox := "[ ]"
					if t.Done {
						checkbox = "[x]"
					}
					priority := priorityBadge(t.Priority)
					ctx.Printf("%s #%-3d %s %s\n", checkbox, t.ID, priority, t.Title)
				}
				ctx.Println(strings.Repeat("-", 50))
				ctx.Printf("Total: %d tasks\n", len(filtered))
			}
			return nil
		})

	// Complete a task (CLI mode)
	app.Command("complete").
		Description("Mark a task as complete").
		Alias("done").
		Args("id").
		Run(func(ctx *cli.Context) error {
			id := ctx.Int("id")
			if id == 0 {
				// Try parsing from positional arg
				if ctx.NArg() > 0 {
					fmt.Sscanf(ctx.Arg(0), "%d", &id)
				}
			}
			if id == 0 {
				return cli.Error("task ID required")
			}

			for i := range tasks {
				if tasks[i].ID == id {
					tasks[i].Done = true
					if ctx.Bool("json") {
						ctx.Printf(`{"id": %d, "completed": true}`+"\n", id)
					} else {
						ctx.Printf("Completed task #%d: %s\n", id, tasks[i].Title)
					}
					return nil
				}
			}
			return cli.Errorf("task #%d not found", id)
		})

	// Delete a task (CLI mode)
	app.Command("delete").
		Description("Delete a task").
		Alias("rm").
		Args("id").
		Run(func(ctx *cli.Context) error {
			var id int
			if ctx.NArg() > 0 {
				fmt.Sscanf(ctx.Arg(0), "%d", &id)
			}
			if id == 0 {
				return cli.Error("task ID required")
			}

			for i := range tasks {
				if tasks[i].ID == id {
					title := tasks[i].Title
					tasks = append(tasks[:i], tasks[i+1:]...)
					ctx.Printf("Deleted task #%d: %s\n", id, title)
					return nil
				}
			}
			return cli.Errorf("task #%d not found", id)
		})

	// Interactive TUI mode
	app.Command("tui").
		Description("Open interactive TUI mode").
		Aliases("ui", "interactive").
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("TUI mode requires an interactive terminal").
					Hint("Run in a terminal with TTY support")
			}

			todoApp := &TodoApp{
				tasks:  tasks,
				cursor: 0,
			}
			return cli.RunInteractive(ctx, todoApp, tui.WithMouseTracking(true))
		})

	// Quick add with interactive prompt
	app.Command("quick-add").
		Description("Add task with interactive prompts").
		Alias("qa").
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("quick-add requires an interactive terminal")
			}

			// Get title
			title, err := cli.NewInput(ctx, "Task title:").
				Placeholder("Enter task description...").
				Show()
			if err != nil {
				return err
			}
			if title == "" {
				return cli.Error("task title cannot be empty")
			}

			// Get priority
			priority, err := cli.Select(ctx, "Priority:", "low", "medium", "high")
			if err != nil {
				return err
			}

			task := Task{
				ID:        len(tasks) + 1,
				Title:     title,
				Priority:  priority,
				Done:      false,
				CreatedAt: time.Now(),
			}
			tasks = append(tasks, task)

			ctx.Printf("\nAdded task #%d: %s [%s]\n", task.ID, task.Title, task.Priority)
			return nil
		})

	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func filterTasks(tasks []Task, filter, priority string) []Task {
	var result []Task
	for _, t := range tasks {
		// Filter by status
		if filter == "pending" && t.Done {
			continue
		}
		if filter == "done" && !t.Done {
			continue
		}
		// Filter by priority
		if priority != "" && t.Priority != priority {
			continue
		}
		result = append(result, t)
	}
	return result
}

func priorityBadge(priority string) string {
	switch priority {
	case "high":
		return "[HIGH]"
	case "medium":
		return "[MED] "
	case "low":
		return "[LOW] "
	default:
		return "      "
	}
}

// TodoApp is the full TUI application
type TodoApp struct {
	tasks   []Task
	cursor  int
	filter  string // all, pending, done
	message string
}

func (app *TodoApp) View() tui.View {
	views := []tui.View{
		// Header
		tui.HeaderBar("Todo Manager"),
		tui.Spacer().MinHeight(1),
	}

	// Filter tabs
	views = append(views, tui.HStack(
		app.filterButton("All", "all"),
		app.filterButton("Pending", "pending"),
		app.filterButton("Done", "done"),
	).Gap(2))
	views = append(views, tui.Divider())

	// Task list
	filtered := app.filteredTasks()
	if len(filtered) == 0 {
		views = append(views, tui.VStack(tui.Text("No tasks found").Dim()).Padding(1))
	} else {
		for i, task := range filtered {
			views = append(views, app.taskRow(task, i == app.cursor))
		}
	}

	views = append(views, tui.Spacer())

	// Message area
	if app.message != "" {
		views = append(views, tui.Text("%s", app.message).Fg(tui.ColorYellow))
	}

	// Footer with help
	views = append(views, tui.Divider())
	views = append(views, tui.Text("j/k: Navigate  Enter: Toggle  d: Delete  a: Add  1-3: Filter  q: Quit").Dim())

	return tui.VStack(views...).Padding(1)
}

func (app *TodoApp) filterButton(label, value string) tui.View {
	style := tui.NewStyle()
	if app.filter == value || (app.filter == "" && value == "all") {
		style = style.WithForeground(tui.ColorCyan).WithBold()
	}
	return tui.Clickable(fmt.Sprintf("[%s]", label), func() {
		if value == "all" {
			app.filter = ""
		} else {
			app.filter = value
		}
		app.cursor = 0
	}).Style(style)
}

func (app *TodoApp) taskRow(task Task, selected bool) tui.View {
	checkbox := "[ ]"
	if task.Done {
		checkbox = "[x]"
	}

	var priorityStyle tui.Style
	switch task.Priority {
	case "high":
		priorityStyle = tui.NewStyle().WithForeground(tui.ColorRed)
	case "medium":
		priorityStyle = tui.NewStyle().WithForeground(tui.ColorYellow)
	case "low":
		priorityStyle = tui.NewStyle().WithForeground(tui.ColorGreen)
	}

	titleStyle := tui.NewStyle()
	if task.Done {
		titleStyle = titleStyle.WithDim()
	}

	rowStyle := tui.NewStyle()
	prefix := "  "
	if selected {
		prefix = "> "
		rowStyle = rowStyle.WithForeground(tui.ColorCyan)
	}

	return tui.HStack(
		tui.Text("%s", prefix).Style(rowStyle),
		tui.Text("%s", checkbox).Style(rowStyle),
		tui.Text(" #%-3d ", task.ID).Dim(),
		tui.Text("[%s]", strings.ToUpper(task.Priority[:1])).Style(priorityStyle),
		tui.Text(" %s", task.Title).Style(titleStyle),
	)
}

func (app *TodoApp) filteredTasks() []Task {
	return filterTasks(app.tasks, app.filter, "")
}

func (app *TodoApp) HandleEvent(event tui.Event) []tui.Cmd {
	filtered := app.filteredTasks()

	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyArrowUp:
			if app.cursor > 0 {
				app.cursor--
			}
		case tui.KeyArrowDown:
			if app.cursor < len(filtered)-1 {
				app.cursor++
			}
		case tui.KeyEnter:
			// Toggle task completion
			if app.cursor < len(filtered) {
				taskID := filtered[app.cursor].ID
				for i := range app.tasks {
					if app.tasks[i].ID == taskID {
						app.tasks[i].Done = !app.tasks[i].Done
						if app.tasks[i].Done {
							app.message = fmt.Sprintf("Completed: %s", app.tasks[i].Title)
						} else {
							app.message = fmt.Sprintf("Reopened: %s", app.tasks[i].Title)
						}
						break
					}
				}
			}
		case tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}

		switch e.Rune {
		case 'q':
			return []tui.Cmd{tui.Quit()}
		case 'j':
			if app.cursor < len(filtered)-1 {
				app.cursor++
			}
		case 'k':
			if app.cursor > 0 {
				app.cursor--
			}
		case 'd':
			// Delete task
			if app.cursor < len(filtered) {
				taskID := filtered[app.cursor].ID
				for i := range app.tasks {
					if app.tasks[i].ID == taskID {
						app.message = fmt.Sprintf("Deleted: %s", app.tasks[i].Title)
						app.tasks = append(app.tasks[:i], app.tasks[i+1:]...)
						if app.cursor >= len(app.filteredTasks()) && app.cursor > 0 {
							app.cursor--
						}
						break
					}
				}
			}
		case '1':
			app.filter = ""
			app.cursor = 0
		case '2':
			app.filter = "pending"
			app.cursor = 0
		case '3':
			app.filter = "done"
			app.cursor = 0
		case 'a':
			// In a real app, this would open an input dialog
			app.message = "Press 'q' to quit and use 'todo add' to add tasks"
		}
	}
	return nil
}
