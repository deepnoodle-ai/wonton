// Example: Streaming and Progress
//
// Demonstrates streaming output and progress indicators:
// - Stream function for chunked output
// - Progress indicator with spinner
// - Different output modes (TTY, pipe, JSON)
//
// Run with:
//
//	go run examples/cli_streaming/main.go generate "Write a poem"
//	go run examples/cli_streaming/main.go download
//	go run examples/cli_streaming/main.go process
//	go run examples/cli_streaming/main.go generate --json "Hello"
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
)

func main() {
	app := cli.New("streamdemo", "Demonstrates streaming and progress")
	app.Version("1.0.0")

	// Add global --json flag for JSON output mode
	app.AddGlobalFlag(&cli.Flag{
		Name:        "json",
		Description: "Output as JSON events",
		Default:     false,
	})

	// Streaming text generation
	app.Command("generate", "Generate text with streaming", cli.WithArgs("prompt")).
		Run(func(ctx *cli.Context) error {
			prompt := ctx.Arg(0)
			if prompt == "" {
				prompt = "Tell me something interesting"
			}

			ctx.Printf("Generating response for: %s\n\n", prompt)

			// Use Stream for chunked output
			// - In TTY: shows spinner + streaming text
			// - In pipe: raw chunks
			// - With --json: newline-delimited JSON events
			return ctx.Stream(func(yield func(string)) error {
				// Simulate streaming generation
				response := "The quick brown fox jumps over the lazy dog. " +
					"This is a simulated streaming response that demonstrates " +
					"how text can be output in chunks, similar to how LLM APIs " +
					"stream their responses token by token."

				words := splitIntoWords(response)
				for i, word := range words {
					if i > 0 {
						yield(" ")
					}
					yield(word)
					time.Sleep(50 * time.Millisecond)
				}
				yield("\n")
				return nil
			})
		})

	// Download with progress
	app.Command("download", "Simulate a download with progress").
		AddFlag(&cli.Flag{
			Name:        "size",
			Short:       "s",
			Description: "File size in MB",
			Default:     10,
		}).
		Run(func(ctx *cli.Context) error {
			size := ctx.Int("size")

			return ctx.WithProgress("Downloading file...", func(p *cli.Progress) error {
				total := size * 10 // 10 chunks per MB
				for i := 0; i <= total; i++ {
					p.SetProgress(i, total)
					p.SetMessage(fmt.Sprintf("Downloading... %d%%", (i*100)/total))
					time.Sleep(50 * time.Millisecond)
				}
				p.Complete()
				p.Append(fmt.Sprintf("\nDownloaded %d MB successfully!\n", size))
				return nil
			})
		})

	// Multi-step process
	app.Command("process", "Run a multi-step process").
		Run(func(ctx *cli.Context) error {
			steps := []string{
				"Initializing...",
				"Loading configuration...",
				"Connecting to server...",
				"Fetching data...",
				"Processing records...",
				"Generating report...",
				"Cleaning up...",
			}

			return ctx.WithProgress(steps[0], func(p *cli.Progress) error {
				for i, step := range steps {
					p.SetMessage(step)
					p.SetProgress(i+1, len(steps))
					p.Append(fmt.Sprintf("Step %d/%d: %s\n", i+1, len(steps), step))
					time.Sleep(500 * time.Millisecond)
				}
				p.Complete()
				p.Append("\nAll steps completed successfully!\n")
				return nil
			})
		})

	// Streaming with progress messages
	app.Command("analyze", "Analyze with streaming output", cli.WithArgs("input")).
		Run(func(ctx *cli.Context) error {
			input := ctx.Arg(0)
			if input == "" {
				input = "sample data"
			}

			return ctx.WithProgress("Analyzing...", func(p *cli.Progress) error {
				// Phase 1: Preparation
				p.SetMessage("Preparing analysis...")
				time.Sleep(500 * time.Millisecond)

				// Phase 2: Processing
				p.SetMessage("Processing input...")
				p.Append(fmt.Sprintf("Input: %s\n", input))
				time.Sleep(500 * time.Millisecond)

				// Phase 3: Analysis
				p.SetMessage("Running analysis...")
				p.Append("Finding patterns...\n")
				time.Sleep(500 * time.Millisecond)

				p.Append("Computing statistics...\n")
				time.Sleep(500 * time.Millisecond)

				// Results
				p.Complete()
				p.Append("\n")
				p.Append("=== Analysis Results ===\n")
				p.Append(fmt.Sprintf("Input length: %d characters\n", len(input)))
				p.Append(fmt.Sprintf("Word count: %d words\n", len(splitIntoWords(input))))
				p.Append("Status: Complete\n")

				return nil
			})
		})

	// Batch operation with progress
	app.Command("batch", "Process items in batch").
		AddFlag(&cli.Flag{
			Name:        "count",
			Short:       "n",
			Description: "Number of items to process",
			Default:     20,
		}).
		Run(func(ctx *cli.Context) error {
			count := ctx.Int("count")

			return ctx.WithProgress("Processing batch...", func(p *cli.Progress) error {
				for i := 1; i <= count; i++ {
					p.SetMessage(fmt.Sprintf("Processing item %d/%d", i, count))
					p.SetProgress(i, count)

					// Simulate work
					time.Sleep(100 * time.Millisecond)

					// Report progress for some items
					if i%5 == 0 {
						p.Append(fmt.Sprintf("Processed %d items...\n", i))
					}
				}

				p.Complete()
				p.Append(fmt.Sprintf("\nBatch complete! Processed %d items.\n", count))
				return nil
			})
		})

	// Simple spinner
	app.Command("wait", "Show a simple wait spinner").
		AddFlag(&cli.Flag{
			Name:        "seconds",
			Short:       "s",
			Description: "Seconds to wait",
			Default:     3,
		}).
		Run(func(ctx *cli.Context) error {
			seconds := ctx.Int("seconds")

			return ctx.WithProgress("Please wait...", func(p *cli.Progress) error {
				for i := 0; i < seconds; i++ {
					p.SetMessage(fmt.Sprintf("Waiting... %d seconds remaining", seconds-i))
					time.Sleep(1 * time.Second)
				}
				p.Complete()
				p.Append("Done waiting!\n")
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

func splitIntoWords(s string) []string {
	var words []string
	var current string
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}
