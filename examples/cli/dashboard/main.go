// Example: Real-time Dashboard with TUI
//
// Demonstrates building a monitoring dashboard with the CLI framework:
// - Real-time data updates using Wonton tick events
// - Progress bars and meters for system metrics
// - Table views for process/connection lists
// - Multiple dashboard views with keyboard switching
//
// Run with:
//
//	go run examples/cli_dashboard/main.go --help
//	go run examples/cli_dashboard/main.go status           # Quick status check
//	go run examples/cli_dashboard/main.go status --json    # JSON output
//	go run examples/cli_dashboard/main.go monitor          # Live dashboard
//	go run examples/cli_dashboard/main.go processes        # Process list
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/tui"
)

// SystemMetrics represents system performance data
type SystemMetrics struct {
	CPUUsage    float64
	MemoryUsed  int
	MemoryTotal int
	DiskUsed    int
	DiskTotal   int
	NetworkIn   int
	NetworkOut  int
	Uptime      time.Duration
	LoadAvg     [3]float64
}

// Process represents a running process
type Process struct {
	PID    int
	Name   string
	CPU    float64
	Memory float64
	Status string
}

func main() {
	app := cli.New("dashboard").
		Description("System monitoring dashboard").
		Version("1.0.0")

	// Global flags
	app.GlobalFlags(
		&cli.BoolFlag{Name: "json", Help: "Output as JSON"},
	)

	// Quick status check (CLI mode)
	app.Command("status").
		Description("Show current system status").
		Flags(
			&cli.BoolFlag{Name: "verbose", Short: "v", Help: "Show detailed status"},
		).
		Run(func(ctx *cli.Context) error {
			metrics := getMetrics()
			verbose := ctx.Bool("verbose")

			if ctx.Bool("json") {
				data, _ := json.MarshalIndent(metrics, "", "  ")
				ctx.Println(string(data))
				return nil
			}

			ctx.Println("System Status")
			ctx.Println("=============")
			ctx.Printf("CPU Usage:    %.1f%%\n", metrics.CPUUsage)
			ctx.Printf("Memory:       %d / %d MB (%.1f%%)\n",
				metrics.MemoryUsed, metrics.MemoryTotal,
				float64(metrics.MemoryUsed)/float64(metrics.MemoryTotal)*100)
			ctx.Printf("Disk:         %d / %d GB (%.1f%%)\n",
				metrics.DiskUsed, metrics.DiskTotal,
				float64(metrics.DiskUsed)/float64(metrics.DiskTotal)*100)

			if verbose {
				ctx.Println("")
				ctx.Printf("Network In:   %d KB/s\n", metrics.NetworkIn)
				ctx.Printf("Network Out:  %d KB/s\n", metrics.NetworkOut)
				ctx.Printf("Load Average: %.2f %.2f %.2f\n",
					metrics.LoadAvg[0], metrics.LoadAvg[1], metrics.LoadAvg[2])
				ctx.Printf("Uptime:       %s\n", formatDuration(metrics.Uptime))
			}

			return nil
		})

	// Process list (CLI mode)
	app.Command("processes").
		Description("List running processes").
		Alias("ps").
		Flags(
			&cli.IntFlag{Name: "limit", Short: "n", Help: "Number of processes to show", Value: 10},
			&cli.StringFlag{Name: "sort", Short: "s", Help: "Sort by field", Value: "cpu", Enum: []string{"cpu", "memory", "pid"}},
		).
		Run(func(ctx *cli.Context) error {
			limit := ctx.Int("limit")
			processes := getProcesses(limit)

			if ctx.Bool("json") {
				data, _ := json.MarshalIndent(processes, "", "  ")
				ctx.Println(string(data))
				return nil
			}

			ctx.Println("Running Processes")
			ctx.Println("-----------------")
			ctx.Printf("%-8s %-20s %8s %8s %10s\n", "PID", "NAME", "CPU%", "MEM%", "STATUS")
			for _, p := range processes {
				ctx.Printf("%-8d %-20s %8.1f %8.1f %10s\n",
					p.PID, truncate(p.Name, 20), p.CPU, p.Memory, p.Status)
			}

			return nil
		})

	// Live monitoring dashboard (TUI mode)
	app.Command("monitor").
		Description("Open live monitoring dashboard").
		Aliases("mon", "live").
		Flags(
			&cli.IntFlag{Name: "refresh", Short: "r", Help: "Refresh interval in seconds", Value: 1},
		).
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("monitor requires an interactive terminal").
					Hint("Run in a terminal with TTY support")
			}

			refresh := ctx.Int("refresh")
			if refresh < 1 {
				refresh = 1
			}

			dashApp := &DashboardApp{
				metrics:   getMetrics(),
				history:   make([]float64, 30),
				refresh:   time.Duration(refresh) * time.Second,
				view:      "overview",
				processes: getProcesses(10),
			}
			return cli.RunInteractive(ctx, dashApp,
				tui.WithMouseTracking(true),
				tui.WithFPS(30),
			)
		})

	// Watch a specific metric with progress indicator
	app.Command("watch").
		Description("Watch a specific metric").
		Flags(
			&cli.StringFlag{Name: "metric", Short: "m", Help: "Metric to watch", Value: "cpu", Enum: []string{"cpu", "memory", "disk", "network"}},
			&cli.IntFlag{Name: "duration", Short: "d", Help: "Watch duration in seconds", Value: 10},
		).
		Run(func(ctx *cli.Context) error {
			metric := ctx.String("metric")
			duration := ctx.Int("duration")

			return ctx.WithProgress(fmt.Sprintf("Watching %s...", metric), func(p *cli.Progress) error {
				for i := 0; i < duration; i++ {
					metrics := getMetrics()
					var value float64
					switch metric {
					case "cpu":
						value = metrics.CPUUsage
					case "memory":
						value = float64(metrics.MemoryUsed) / float64(metrics.MemoryTotal) * 100
					case "disk":
						value = float64(metrics.DiskUsed) / float64(metrics.DiskTotal) * 100
					case "network":
						value = float64(metrics.NetworkIn + metrics.NetworkOut)
					}

					p.SetProgress(i+1, duration)
					p.Append(fmt.Sprintf("[%s] %s: %.1f\n", time.Now().Format("15:04:05"), metric, value))
					time.Sleep(1 * time.Second)
				}
				p.Complete()
				return nil
			})
		})

	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

// DashboardApp is the full TUI dashboard
type DashboardApp struct {
	metrics   SystemMetrics
	processes []Process
	history   []float64
	refresh   time.Duration
	view      string // overview, processes, network
	frame     uint64
}

func (app *DashboardApp) View() tui.View {
	views := []tui.View{
		// Header
		tui.Group(
			tui.Text("System Dashboard").Bold().Fg(tui.ColorCyan),
			tui.Spacer(),
			tui.Text("[%s]", time.Now().Format("15:04:05")).Dim(),
		),
		tui.Divider(),
	}

	// View tabs
	views = append(views, tui.Group(
		app.tabButton("Overview", "overview"),
		app.tabButton("Processes", "processes"),
		app.tabButton("Network", "network"),
	).Gap(2))
	views = append(views, tui.Spacer().MinHeight(1))

	// Content based on view
	switch app.view {
	case "overview":
		views = append(views, app.overviewView()...)
	case "processes":
		views = append(views, app.processesView()...)
	case "network":
		views = append(views, app.networkView()...)
	}

	views = append(views, tui.Spacer())

	// Footer
	views = append(views, tui.Divider())
	views = append(views, tui.Text("1-3: Switch Views  r: Refresh  q: Quit").Dim())

	return tui.Stack(views...).Padding(1)
}

func (app *DashboardApp) tabButton(label, value string) tui.View {
	style := tui.NewStyle()
	if app.view == value {
		style = style.WithForeground(tui.ColorCyan).WithBold().WithUnderline()
	}
	return tui.Clickable(label, func() {
		app.view = value
	}).Style(style)
}

func (app *DashboardApp) overviewView() []tui.View {
	m := app.metrics

	cpuColor := tui.ColorGreen
	if m.CPUUsage > 80 {
		cpuColor = tui.ColorRed
	} else if m.CPUUsage > 50 {
		cpuColor = tui.ColorYellow
	}

	memPercent := float64(m.MemoryUsed) / float64(m.MemoryTotal) * 100
	memColor := tui.ColorGreen
	if memPercent > 80 {
		memColor = tui.ColorRed
	} else if memPercent > 50 {
		memColor = tui.ColorYellow
	}

	diskPercent := float64(m.DiskUsed) / float64(m.DiskTotal) * 100
	diskColor := tui.ColorGreen
	if diskPercent > 80 {
		diskColor = tui.ColorRed
	} else if diskPercent > 50 {
		diskColor = tui.ColorYellow
	}

	return []tui.View{
		// CPU
		tui.Group(
			tui.Text("CPU").Bold().Width(10),
			tui.Progress(int(m.CPUUsage), 100).Width(40).Fg(cpuColor),
			tui.Text(" %5.1f%%", m.CPUUsage),
		),
		tui.Spacer().MinHeight(1),

		// Memory
		tui.Group(
			tui.Text("Memory").Bold().Width(10),
			tui.Progress(m.MemoryUsed, m.MemoryTotal).Width(40).Fg(memColor),
			tui.Text(" %d/%d MB", m.MemoryUsed, m.MemoryTotal),
		),
		tui.Spacer().MinHeight(1),

		// Disk
		tui.Group(
			tui.Text("Disk").Bold().Width(10),
			tui.Progress(m.DiskUsed, m.DiskTotal).Width(40).Fg(diskColor),
			tui.Text(" %d/%d GB", m.DiskUsed, m.DiskTotal),
		),
		tui.Spacer().MinHeight(1),

		// Load average
		tui.Group(
			tui.Text("Load Avg").Bold().Width(10),
			tui.Text("%.2f  %.2f  %.2f", m.LoadAvg[0], m.LoadAvg[1], m.LoadAvg[2]),
		),

		// Uptime
		tui.Group(
			tui.Text("Uptime").Bold().Width(10),
			tui.Text("%s", formatDuration(m.Uptime)),
		),

		tui.Spacer().MinHeight(1),

		// CPU History (sparkline-like)
		tui.Text("CPU History:").Bold(),
		app.historyChart(),
	}
}

func (app *DashboardApp) historyChart() tui.View {
	// Simple ASCII chart of CPU history
	bars := []rune{'_', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var chart string
	for _, v := range app.history {
		idx := int(v / 100 * float64(len(bars)-1))
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		if idx < 0 {
			idx = 0
		}
		chart += string(bars[idx])
	}
	return tui.Text("%s", chart).Fg(tui.ColorCyan)
}

func (app *DashboardApp) processesView() []tui.View {
	views := []tui.View{
		tui.Group(
			tui.Text("PID").Bold().Width(8),
			tui.Text("NAME").Bold().Width(20),
			tui.Text("CPU%%").Bold().Width(8),
			tui.Text("MEM%%").Bold().Width(8),
			tui.Text("STATUS").Bold().Width(10),
		),
		tui.Divider(),
	}

	for _, p := range app.processes {
		cpuView := tui.Text("%.1f", p.CPU)
		if p.CPU > 50 {
			cpuView = cpuView.Fg(tui.ColorRed)
		} else if p.CPU > 20 {
			cpuView = cpuView.Fg(tui.ColorYellow)
		}

		statusView := tui.Text("%s", p.Status)
		if p.Status == "running" {
			statusView = statusView.Fg(tui.ColorGreen)
		}

		views = append(views, tui.Group(
			tui.Text("%d", p.PID).Width(8),
			tui.Text("%s", truncate(p.Name, 18)).Width(20),
			cpuView.Width(8),
			tui.Text("%.1f", p.Memory).Width(8),
			statusView.Width(10),
		))
	}

	return views
}

func (app *DashboardApp) networkView() []tui.View {
	m := app.metrics
	return []tui.View{
		tui.Text("Network Traffic").Bold(),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
		tui.Group(
			tui.Text("Inbound:").Width(12),
			tui.Text("%d KB/s", m.NetworkIn).Fg(tui.ColorGreen),
			tui.Text("  ").Width(4),
			tui.Loading(app.frame),
		),
		tui.Group(
			tui.Text("Outbound:").Width(12),
			tui.Text("%d KB/s", m.NetworkOut).Fg(tui.ColorCyan),
		),
		tui.Spacer().MinHeight(1),
		tui.Text("Active Connections: 42").Dim(),
		tui.Text("Packets/sec: %d", (m.NetworkIn+m.NetworkOut)*10).Dim(),
	}
}

func (app *DashboardApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.TickEvent:
		app.frame++
		// Refresh metrics
		app.metrics = getMetrics()
		app.processes = getProcesses(10)
		// Update history
		app.history = append(app.history[1:], app.metrics.CPUUsage)
		return []tui.Cmd{tui.Tick(app.refresh)}

	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}
		switch e.Rune {
		case 'q':
			return []tui.Cmd{tui.Quit()}
		case 'r':
			app.metrics = getMetrics()
			app.processes = getProcesses(10)
		case '1':
			app.view = "overview"
		case '2':
			app.view = "processes"
		case '3':
			app.view = "network"
		}
	}
	return nil
}

// Simulated data functions

func getMetrics() SystemMetrics {
	return SystemMetrics{
		CPUUsage:    30 + rand.Float64()*40,
		MemoryUsed:  4000 + rand.Intn(4000),
		MemoryTotal: 16000,
		DiskUsed:    200 + rand.Intn(50),
		DiskTotal:   500,
		NetworkIn:   100 + rand.Intn(500),
		NetworkOut:  50 + rand.Intn(200),
		Uptime:      time.Duration(rand.Intn(1000000)) * time.Second,
		LoadAvg:     [3]float64{1.5 + rand.Float64(), 1.2 + rand.Float64(), 0.8 + rand.Float64()},
	}
}

func getProcesses(limit int) []Process {
	names := []string{"node", "python", "go", "postgres", "redis", "nginx", "docker", "chrome", "code", "bash"}
	statuses := []string{"running", "sleeping", "running", "running", "sleeping"}

	processes := make([]Process, limit)
	for i := 0; i < limit; i++ {
		processes[i] = Process{
			PID:    1000 + rand.Intn(9000),
			Name:   names[rand.Intn(len(names))],
			CPU:    rand.Float64() * 50,
			Memory: rand.Float64() * 20,
			Status: statuses[rand.Intn(len(statuses))],
		}
	}
	return processes
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "~"
}
