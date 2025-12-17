# unidiff

Unified diff format parser. Parses git diff output and similar unified diff formats into structured data for display, analysis, or transformation.

## Usage Examples

### Parsing a Diff

```go
package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/unidiff"
)

func main() {
	diffText := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

+import "fmt"
+
 func main() {
-    println("Hello")
+    fmt.Println("Hello, World!")
 }`

	// Parse the diff
	diff, err := unidiff.Parse(diffText)
	if err != nil {
		log.Fatal(err)
	}

	// Iterate through files
	for _, file := range diff.Files {
		fmt.Printf("File: %s -> %s\n", file.OldPath, file.NewPath)

		// Iterate through hunks
		for _, hunk := range file.Hunks {
			fmt.Printf("  Hunk: %s\n", hunk.Header)
			fmt.Printf("  Old: lines %d-%d\n", hunk.OldStart, hunk.OldStart+hunk.OldCount-1)
			fmt.Printf("  New: lines %d-%d\n", hunk.NewStart, hunk.NewStart+hunk.NewCount-1)

			// Iterate through lines
			for _, line := range hunk.Lines {
				switch line.Type {
				case unidiff.LineAdded:
					fmt.Printf("  + %s\n", line.Content)
				case unidiff.LineRemoved:
					fmt.Printf("  - %s\n", line.Content)
				case unidiff.LineContext:
					fmt.Printf("    %s\n", line.Content)
				}
			}
		}
	}
}
```

### Getting Diff Statistics

```go
func analyzeDiff(diffText string) {
	diff, err := unidiff.Parse(diffText)
	if err != nil {
		log.Fatal(err)
	}

	// Get summary statistics
	stats := diff.Stats()
	fmt.Printf("Files changed: %d\n", stats.FilesChanged)
	fmt.Printf("Additions: %d\n", stats.Additions)
	fmt.Printf("Deletions: %d\n", stats.Deletions)
	fmt.Printf("Net change: %+d lines\n", stats.Additions-stats.Deletions)
}
```

### Filtering Lines by Type

```go
func findAddedLines(diff *unidiff.Diff) []string {
	var added []string

	for _, file := range diff.Files {
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				if line.Type == unidiff.LineAdded {
					added = append(added, line.Content)
				}
			}
		}
	}

	return added
}

func findRemovedLines(diff *unidiff.Diff) []string {
	var removed []string

	for _, file := range diff.Files {
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				if line.Type == unidiff.LineRemoved {
					removed = append(removed, line.Content)
				}
			}
		}
	}

	return removed
}
```

### Analyzing Line Numbers

```go
func showLineMapping(diff *unidiff.Diff) {
	for _, file := range diff.Files {
		fmt.Printf("\nFile: %s\n", file.NewPath)

		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				switch line.Type {
				case unidiff.LineAdded:
					fmt.Printf("  Line %d (added): %s\n", line.NewLineNum, line.Content)
				case unidiff.LineRemoved:
					fmt.Printf("  Line %d (removed): %s\n", line.OldLineNum, line.Content)
				case unidiff.LineContext:
					fmt.Printf("  Lines %d->%d: %s\n", line.OldLineNum, line.NewLineNum, line.Content)
				}
			}
		}
	}
}
```

### Parsing Git Diff Output

```go
package main

import (
	"log"
	"os/exec"

	"github.com/deepnoodle-ai/wonton/unidiff"
)

func main() {
	// Get git diff output
	cmd := exec.Command("git", "diff")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	// Parse the diff
	diff, err := unidiff.Parse(string(output))
	if err != nil {
		log.Fatal(err)
	}

	// Process the diff
	for _, file := range diff.Files {
		fmt.Printf("Modified: %s\n", file.NewPath)
	}
}
```

### Comparing Specific Files

```go
func findFileChanges(diff *unidiff.Diff, filename string) *unidiff.File {
	for _, file := range diff.Files {
		if file.NewPath == filename || file.OldPath == filename {
			return &file
		}
	}
	return nil
}

func main() {
	diff, _ := unidiff.Parse(diffText)

	file := findFileChanges(diff, "main.go")
	if file != nil {
		fmt.Printf("Changes to main.go:\n")
		for _, hunk := range file.Hunks {
			fmt.Printf("  %d lines changed\n", len(hunk.Lines))
		}
	}
}
```

### Detecting File Renames

```go
func detectRenames(diff *unidiff.Diff) []struct{ Old, New string } {
	var renames []struct{ Old, New string }

	for _, file := range diff.Files {
		if file.OldPath != file.NewPath {
			renames = append(renames, struct{ Old, New string }{
				Old: file.OldPath,
				New: file.NewPath,
			})
		}
	}

	return renames
}
```

### Generating Summary Report

```go
func generateReport(diff *unidiff.Diff) string {
	stats := diff.Stats()

	var report strings.Builder
	report.WriteString("Diff Summary\n")
	report.WriteString("============\n\n")

	report.WriteString(fmt.Sprintf("Files changed: %d\n", stats.FilesChanged))
	report.WriteString(fmt.Sprintf("Insertions: %d\n", stats.Additions))
	report.WriteString(fmt.Sprintf("Deletions: %d\n", stats.Deletions))
	report.WriteString("\n")

	report.WriteString("Modified Files:\n")
	for _, file := range diff.Files {
		report.WriteString(fmt.Sprintf("  - %s\n", file.NewPath))

		// Count changes per file
		adds, dels := 0, 0
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				switch line.Type {
				case unidiff.LineAdded:
					adds++
				case unidiff.LineRemoved:
					dels++
				}
			}
		}
		report.WriteString(fmt.Sprintf("    +%d -%d\n", adds, dels))
	}

	return report.String()
}
```

### Filtering Context Lines

```go
func getModifiedLinesOnly(diff *unidiff.Diff) *unidiff.Diff {
	filtered := &unidiff.Diff{}

	for _, file := range diff.Files {
		newFile := unidiff.File{
			OldPath: file.OldPath,
			NewPath: file.NewPath,
		}

		for _, hunk := range file.Hunks {
			newHunk := unidiff.Hunk{
				OldStart: hunk.OldStart,
				OldCount: hunk.OldCount,
				NewStart: hunk.NewStart,
				NewCount: hunk.NewCount,
				Header:   hunk.Header,
			}

			// Only include added and removed lines
			for _, line := range hunk.Lines {
				if line.Type != unidiff.LineContext {
					newHunk.Lines = append(newHunk.Lines, line)
				}
			}

			if len(newHunk.Lines) > 0 {
				newFile.Hunks = append(newFile.Hunks, newHunk)
			}
		}

		if len(newFile.Hunks) > 0 {
			filtered.Files = append(filtered.Files, newFile)
		}
	}

	return filtered
}
```

### Custom Diff Display

```go
func displayColoredDiff(diff *unidiff.Diff) {
	for _, file := range diff.Files {
		// File header
		fmt.Printf("\033[1m%s\033[0m\n", file.NewPath)

		for _, hunk := range file.Hunks {
			// Hunk header in cyan
			fmt.Printf("\033[36m%s\033[0m\n", hunk.Header)

			for _, line := range hunk.Lines {
				switch line.Type {
				case unidiff.LineAdded:
					// Green for additions
					fmt.Printf("\033[32m+%s\033[0m\n", line.Content)
				case unidiff.LineRemoved:
					// Red for deletions
					fmt.Printf("\033[31m-%s\033[0m\n", line.Content)
				case unidiff.LineContext:
					// Normal for context
					fmt.Printf(" %s\n", line.Content)
				}
			}
		}
		fmt.Println()
	}
}
```

## API Reference

### Types

| Type | Description |
|------|-------------|
| `Diff` | Complete diff containing multiple files |
| `File` | Changes to a single file |
| `Hunk` | Contiguous block of changes |
| `Line` | Single line in a diff |
| `Stats` | Diff statistics (files, additions, deletions) |
| `LineType` | Type of line (context, added, removed, header, hunk) |

### Line Type Constants

| Constant | Description |
|----------|-------------|
| `LineContext` | Unchanged line (context) |
| `LineAdded` | Added line (starts with +) |
| `LineRemoved` | Removed line (starts with -) |
| `LineHeader` | File header line |
| `LineHunkHeader` | Hunk header line (@@ ...) |

### Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Parse` | Parses unified diff format | `diffText string` | `*Diff, error` |
| `Diff.Stats` | Calculates diff statistics | none | `Stats` |
| `LineType.String` | Returns string representation | none | `string` |

### Diff Structure

```go
type Diff struct {
    Files []File
}

type File struct {
    OldPath string
    NewPath string
    Hunks   []Hunk
}

type Hunk struct {
    OldStart int    // Starting line in old file
    OldCount int    // Number of lines in old file
    NewStart int    // Starting line in new file
    NewCount int    // Number of lines in new file
    Header   string // The @@ line content
    Lines    []Line
}

type Line struct {
    Type       LineType
    OldLineNum int    // Line number in old file (0 if added)
    NewLineNum int    // Line number in new file (0 if removed)
    Content    string // Line content without +/- marker
    RawLine    string // Original line including marker
}

type Stats struct {
    FilesChanged int
    Additions    int
    Deletions    int
}
```

## Supported Formats

The parser supports standard unified diff format as produced by:

- `git diff`
- `diff -u`
- `svn diff`
- GitHub/GitLab/Bitbucket PR diffs

## Line Number Tracking

The parser tracks line numbers in both old and new files:

- **Context lines**: Both `OldLineNum` and `NewLineNum` are set
- **Added lines**: Only `NewLineNum` is set (OldLineNum is 0)
- **Removed lines**: Only `OldLineNum` is set (NewLineNum is 0)

## Related Packages

- [tui](../tui) - Can be used to display diffs in terminal UI
- [git](../git) - Wrapper for git operations that produce diffs
- [terminal](../terminal) - For colored diff output
