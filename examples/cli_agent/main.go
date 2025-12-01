// Example: AI Agent Tools
//
// Demonstrates building CLI commands as AI-callable tools:
// - Marking commands as tools
// - Generating MCP-compatible tool schemas
// - Tool parameter definitions
// - Structured tool output
//
// Run with:
//
//	go run examples/cli_agent/main.go --help
//	go run examples/cli_agent/main.go tools
//	go run examples/cli_agent/main.go read-file ./go.mod
//	go run examples/cli_agent/main.go search --pattern "func main"
//	go run examples/cli_agent/main.go execute ls -la
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/deepnoodle-ai/wonton/cli"
)

// Tool parameter structs with proper annotations

type ReadFileParams struct {
	Path string `flag:"path,p" help:"File path to read" required:"true"`
}

type WriteFileParams struct {
	Path    string `flag:"path,p" help:"File path to write"`
	Content string `flag:"content,c" help:"Content to write"`
}

type SearchParams struct {
	Pattern     string `flag:"pattern,p" help:"Search pattern (regex)"`
	Path        string `flag:"path" help:"Directory to search" default:"."`
	FilePattern string `flag:"glob,g" help:"File glob pattern" default:"*"`
	MaxResults  int    `flag:"max,m" help:"Maximum results" default:"10"`
}

func main() {
	app := cli.New("agent", "An AI agent CLI with tool support")
	app.Version("1.0.0")

	// Add global --json flag for structured output
	app.AddGlobalFlag(&cli.Flag{
		Name:        "json",
		Description: "Output as JSON",
		Default:     false,
	})

	// Built-in command to output tool schemas
	app.Command("tools", "Output tool schemas as JSON").
		Run(cli.PrintToolsJSON)

	// File operations group
	files := app.Group("files", "File operations")

	// Read file tool
	files.Command("read", "Read a file", cli.WithArgs("path"), cli.WithTool()).
		Long("Read the contents of a file. Returns the file content as text.").
		Run(func(ctx *cli.Context) error {
			path := ctx.Arg(0)
			if path == "" {
				return cli.Error("path is required")
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return cli.Errorf("failed to read file: %v", err).
					Hint("Check that the file exists and is readable")
			}

			if ctx.Bool("json") {
				output := map[string]any{
					"path":    path,
					"content": string(content),
					"size":    len(content),
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				ctx.Println(string(data))
			} else {
				ctx.Print(string(content))
			}
			return nil
		})

	// Write file tool
	files.Command("write", "Write to a file", cli.WithTool()).
		AddArg(&cli.Arg{Name: "path", Description: "File path", Required: true}).
		AddArg(&cli.Arg{Name: "content", Description: "Content to write", Required: true}).
		Run(func(ctx *cli.Context) error {
			path := ctx.Arg(0)
			content := ctx.Arg(1)

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return cli.Errorf("failed to write file: %v", err)
			}

			if ctx.Bool("json") {
				output := map[string]any{
					"path":    path,
					"written": len(content),
					"success": true,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				ctx.Println(string(data))
			} else {
				ctx.Printf("Wrote %d bytes to %s\n", len(content), path)
			}
			return nil
		})

	// List directory tool
	files.Command("list", "List directory contents", cli.WithTool()).
		AddArg(&cli.Arg{Name: "path", Description: "Directory path", Required: false}).
		AddFlag(&cli.Flag{
			Name:        "recursive",
			Short:       "r",
			Description: "List recursively",
			Default:     false,
		}).
		Run(func(ctx *cli.Context) error {
			path := ctx.Arg(0)
			if path == "" {
				path = "."
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				return cli.Errorf("failed to read directory: %v", err)
			}

			if ctx.Bool("json") {
				var items []map[string]any
				for _, e := range entries {
					info, _ := e.Info()
					item := map[string]any{
						"name":  e.Name(),
						"isDir": e.IsDir(),
					}
					if info != nil {
						item["size"] = info.Size()
						item["mode"] = info.Mode().String()
					}
					items = append(items, item)
				}
				data, _ := json.MarshalIndent(items, "", "  ")
				ctx.Println(string(data))
			} else {
				for _, e := range entries {
					info, _ := e.Info()
					if e.IsDir() {
						ctx.Printf("%s/\n", e.Name())
					} else if info != nil {
						ctx.Printf("%s (%d bytes)\n", e.Name(), info.Size())
					} else {
						ctx.Println(e.Name())
					}
				}
			}
			return nil
		})

	// Search tool
	app.Command("search", "Search for patterns in files", cli.WithTool()).
		AddFlag(&cli.Flag{
			Name:        "pattern",
			Short:       "p",
			Description: "Search pattern",
			Required:    true,
		}).
		AddFlag(&cli.Flag{
			Name:        "path",
			Description: "Search path",
			Default:     ".",
		}).
		AddFlag(&cli.Flag{
			Name:        "glob",
			Short:       "g",
			Description: "File pattern",
			Default:     "*.go",
		}).
		Run(func(ctx *cli.Context) error {
			pattern := ctx.String("pattern")
			searchPath := ctx.String("path")
			glob := ctx.String("glob")

			var results []map[string]any

			err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}

				matched, _ := filepath.Match(glob, filepath.Base(path))
				if !matched {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				lines := strings.Split(string(content), "\n")
				for i, line := range lines {
					if strings.Contains(line, pattern) {
						results = append(results, map[string]any{
							"file":    path,
							"line":    i + 1,
							"content": strings.TrimSpace(line),
						})
					}
				}
				return nil
			})

			if err != nil {
				return cli.Errorf("search failed: %v", err)
			}

			if ctx.Bool("json") {
				data, _ := json.MarshalIndent(results, "", "  ")
				ctx.Println(string(data))
			} else {
				if len(results) == 0 {
					ctx.Println("No matches found")
				} else {
					for _, r := range results {
						ctx.Printf("%s:%d: %s\n", r["file"], r["line"], r["content"])
					}
				}
			}
			return nil
		})

	// Execute command tool (with safety restrictions)
	app.Command("execute", "Execute a shell command", cli.WithTool()).
		AddArg(&cli.Arg{Name: "command", Description: "Command to execute", Required: true}).
		AddFlag(&cli.Flag{
			Name:        "timeout",
			Short:       "t",
			Description: "Timeout in seconds",
			Default:     30,
		}).
		Run(func(ctx *cli.Context) error {
			args := ctx.Args()
			if len(args) == 0 {
				return cli.Error("command required")
			}

			// Safety: only allow certain commands
			allowedCommands := map[string]bool{
				"ls": true, "cat": true, "echo": true, "pwd": true,
				"date": true, "whoami": true, "uname": true, "env": true,
			}

			cmdName := args[0]
			if !allowedCommands[cmdName] {
				return cli.Errorf("command '%s' not allowed", cmdName).
					Hint("Allowed commands: ls, cat, echo, pwd, date, whoami, uname, env")
			}

			cmd := exec.Command(args[0], args[1:]...)
			output, err := cmd.CombinedOutput()

			if ctx.Bool("json") {
				result := map[string]any{
					"command": strings.Join(args, " "),
					"output":  string(output),
					"success": err == nil,
				}
				if err != nil {
					result["error"] = err.Error()
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				ctx.Println(string(data))
			} else {
				ctx.Print(string(output))
				if err != nil {
					return cli.Errorf("command failed: %v", err)
				}
			}
			return nil
		})

	// Demo: Generate tool schema from struct
	app.Command("schema", "Generate schema from struct type").
		Run(func(ctx *cli.Context) error {
			// Generate schema from struct definition
			schema := cli.GenerateToolSchemaFromStruct[SearchParams]("search", "Search for patterns in files")

			data, _ := json.MarshalIndent(schema, "", "  ")
			ctx.Println(string(data))
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
