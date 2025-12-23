// Example: gitgif generates an animated GIF visualizing recent git commits.
//
// This tool creates a fun visualization showing:
// - Recent commits with their messages
// - File changes with color-coded additions/deletions
// - Statistics for each commit
//
// Run with:
//
//	go run ./examples/gitgif
//	go run ./examples/gitgif --commits 10 --output my-week.gif
//	go run ./examples/gitgif --repo /path/to/repo
package main

import (
	"context"
	"fmt"
	"image/color"
	"os"

	"github.com/deepnoodle-ai/wonton/cli"
	wontoncolor "github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/gif"
	"github.com/deepnoodle-ai/wonton/git"
	"github.com/deepnoodle-ai/wonton/humanize"
)

const (
	width  = 800
	height = 600
)

func main() {
	app := cli.New("gitgif").
		Description("Generate an animated GIF visualizing recent git commits").
		Version("0.1.0")

	app.Main().
		Flags(
			cli.Int("commits", "c").
				Default(5).
				Help("Number of recent commits to visualize"),
			cli.String("output", "o").
				Default("commits.gif").
				Help("Output GIF filename"),
			cli.Int("delay", "d").
				Default(100).
				Help("Delay between frames in 100ths of a second"),
			cli.String("repo", "r").
				Default(".").
				Help("Path to git repository"),
		).
		Run(generateGIF)

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func generateGIF(ctx *cli.Context) error {
	numCommits := ctx.Int("commits")
	output := ctx.String("output")
	delay := ctx.Int("delay")
	repoPath := ctx.String("repo")

	// Load the TTF font
	font, err := gif.LoadDefaultFont(14)
	if err != nil {
		return fmt.Errorf("failed to load font: %w", err)
	}
	defer font.Close()

	ctx.Printf("Opening git repository at %s...\n", repoPath)
	repo, err := git.Open(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	ctx.Printf("Fetching last %d commits...\n", numCommits)
	commits, err := repo.Log(context.Background(), git.LogOptions{
		Limit:       numCommits,
		IncludeBody: false,
	})
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	if len(commits) == 0 {
		return fmt.Errorf("no commits found")
	}

	ctx.Printf("Found %d commits, generating frames...\n", len(commits))

	// Create color palette with gradients for visualization
	palette := buildPalette()
	g := gif.NewWithPalette(width, height, palette)
	g.SetLoopCount(0) // Loop forever

	// Generate frames for each commit
	for i, commit := range commits {
		ctx.Printf("  [%d/%d] %s: %s\n", i+1, len(commits), commit.ShortHash, commit.Subject)

		// Get diff for this commit
		var diff *git.Diff
		if len(commit.ParentHashes) > 0 {
			diff, err = repo.Diff(context.Background(), git.DiffOptions{
				From:         commit.ParentHashes[0],
				To:           commit.Hash,
				IncludePatch: true,
			})
		} else {
			// First commit - show all files as added
			diff, err = repo.Diff(context.Background(), git.DiffOptions{
				From:         "",
				To:           commit.Hash,
				IncludePatch: false,
			})
		}
		if err != nil {
			// If diff fails, just skip this commit
			ctx.Printf("    Warning: failed to get diff: %v\n", err)
			continue
		}

		// Create frame showing this commit
		g.AddFrameWithDelay(func(f *gif.Frame) {
			renderCommitFrame(f, font, commit, diff, i+1, len(commits))
		}, delay)
	}

	ctx.Printf("Saving to %s...\n", output)
	if err := g.Save(output); err != nil {
		return fmt.Errorf("failed to save GIF: %w", err)
	}

	ctx.Printf("Successfully created %s with %d frames\n", output, len(commits))
	return nil
}

func buildPalette() gif.Palette {
	// Start with basic colors
	p := gif.Palette{
		color.RGBA{15, 20, 30, 255},    // Dark background
		color.RGBA{240, 240, 245, 255}, // White text
		color.RGBA{100, 100, 110, 255}, // Gray text
		color.RGBA{80, 200, 120, 255},  // Green for additions
		color.RGBA{220, 80, 100, 255},  // Red for deletions
		color.RGBA{100, 150, 230, 255}, // Blue accent
		color.RGBA{200, 160, 100, 255}, // Gold accent
	}

	// Add gradient colors for visualization
	greenGradient := wontoncolor.Gradient(
		wontoncolor.NewRGB(40, 100, 60),
		wontoncolor.NewRGB(120, 255, 150),
		20,
	)
	for _, c := range greenGradient {
		p = append(p, color.RGBA{c.R, c.G, c.B, 255})
	}

	redGradient := wontoncolor.Gradient(
		wontoncolor.NewRGB(100, 40, 40),
		wontoncolor.NewRGB(255, 100, 120),
		20,
	)
	for _, c := range redGradient {
		p = append(p, color.RGBA{c.R, c.G, c.B, 255})
	}

	return p
}

func renderCommitFrame(f *gif.Frame, font *gif.FontFace, commit git.Commit, diff *git.Diff, num, total int) {
	bg := color.RGBA{15, 20, 30, 255}
	fg := color.RGBA{240, 240, 245, 255}
	dimFg := color.RGBA{100, 100, 110, 255}
	green := color.RGBA{80, 200, 120, 255}
	red := color.RGBA{220, 80, 100, 255}
	accent := color.RGBA{100, 150, 230, 255}

	img := f.Image()
	lineHeight := font.CellHeight() + 4

	// Fill background
	f.Fill(bg)

	// Draw header bar
	f.FillRect(0, 0, width, 60, color.RGBA{20, 30, 45, 255})
	font.DrawString(img, 20, 15, fmt.Sprintf("Commit %d of %d", num, total), accent)
	font.DrawString(img, 20, 38, commit.ShortHash, dimFg)

	// Draw commit metadata
	y := 75
	font.DrawString(img, 20, y, "Author: "+commit.Author.Name, fg)
	y += lineHeight
	font.DrawString(img, 20, y, humanize.Time(commit.Timestamp), dimFg)
	y += lineHeight + 10

	// Draw commit message (wrapped if needed)
	subject := commit.Subject
	if len(subject) > 80 {
		subject = subject[:77] + "..."
	}
	font.DrawString(img, 20, y, subject, fg)
	y += lineHeight + 15

	// Draw stats summary
	if diff != nil {
		statsText := fmt.Sprintf("%s | +%s -%s",
			humanize.PluralWord(len(diff.Files), "file", "files"),
			humanize.Number(int64(diff.TotalAdded)),
			humanize.Number(int64(diff.TotalRemoved)),
		)
		font.DrawString(img, 20, y, statsText, accent)
		y += lineHeight + 10

		// Draw file changes (up to 12 files)
		maxFiles := 12
		for i, file := range diff.Files {
			if i >= maxFiles {
				remaining := len(diff.Files) - maxFiles
				font.DrawString(img, 40, y, fmt.Sprintf("... and %d more files", remaining), dimFg)
				break
			}

			// File name
			fileName := file.Path
			if len(fileName) > 50 {
				fileName = "..." + fileName[len(fileName)-47:]
			}
			font.DrawString(img, 40, y, fileName, fg)

			// Draw bar chart showing additions/deletions
			barX := 480
			barY := y
			barMaxWidth := 250
			total := file.Additions + file.Deletions
			if total > 0 {
				scale := float64(barMaxWidth) / float64(max(total, 50))
				addWidth := int(float64(file.Additions) * scale)
				delWidth := int(float64(file.Deletions) * scale)

				if addWidth > 0 {
					f.FillRect(barX, barY, addWidth, 14, green)
				}
				if delWidth > 0 {
					f.FillRect(barX+addWidth, barY, delWidth, 14, red)
				}

				// Draw stats text
				statsText := fmt.Sprintf("+%d -%d", file.Additions, file.Deletions)
				font.DrawString(img, barX+addWidth+delWidth+10, y, statsText, dimFg)
			}

			y += lineHeight + 4
			if y > height-50 {
				break
			}
		}
	}

	// Draw footer
	f.FillRect(0, height-30, width, 30, color.RGBA{20, 30, 45, 255})
	font.DrawString(img, width-210, height-22, "Generated with gitgif", dimFg)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
