# gitgif - Git Commit Visualizer

Generate an animated GIF that visualizes recent git commits, showing file changes, insertions/deletions with color-coded bars, and commit messages.

## Features

- Visualizes multiple commits in an animated GIF
- Shows commit metadata (hash, author, timestamp)
- Displays file changes with color-coded addition/deletion bars
- Includes statistics for each commit
- Customizable number of commits and frame delay

## Usage

```bash
# Generate GIF with default settings (5 commits)
go run ./examples/gitgif generate

# Generate with custom number of commits
go run ./examples/gitgif generate --commits 10

# Specify output filename
go run ./examples/gitgif generate --output my-week.gif

# Customize frame delay (in 100ths of a second)
go run ./examples/gitgif generate --delay 150

# Use a different repository
go run ./examples/gitgif generate --repo /path/to/repo
```

## Options

- `-c, --commits` - Number of recent commits to visualize (default: 5)
- `-o, --output` - Output GIF filename (default: commits.gif)
- `-d, --delay` - Delay between frames in 100ths of a second (default: 100)
- `-r, --repo` - Path to git repository (default: .)

## Example Output

The generated GIF shows:

1. **Header**: Commit number and hash
2. **Metadata**: Author name and relative timestamp
3. **Commit Message**: Subject line (truncated if too long)
4. **Statistics**: Files changed, additions, deletions
5. **File List**: Each file with a visual bar showing additions (green) and deletions (red)

## Packages Used

- `cli` - Command-line interface framework
- `git` - Git repository operations and commit history
- `gif` - Animated GIF creation with drawing primitives
- `color` - Color gradients for visualization
- `humanize` - Human-readable formatting for numbers and times

## Implementation Highlights

- Simple bitmap font rendering for text display
- Color-coded bar charts showing file changes
- Gradient palette for smooth color transitions
- Handles commits with no parents (initial commit)
- Truncates long file names and messages for display
