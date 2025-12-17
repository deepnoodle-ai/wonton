// Package unidiff provides parsing and analysis of unified diff format.
//
// This package parses unified diff output (like from git diff, diff -u, or
// version control systems) into structured data that can be used for display,
// analysis, or transformation. It handles the standard unified diff format
// with support for multiple files, multiple hunks per file, and proper line
// number tracking.
//
// # Basic Usage
//
// Parse a diff and iterate through the changes:
//
//	diff, err := unidiff.Parse(diffText)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, file := range diff.Files {
//	    fmt.Printf("File: %s -> %s\n", file.OldPath, file.NewPath)
//	    for _, hunk := range file.Hunks {
//	        for _, line := range hunk.Lines {
//	            switch line.Type {
//	            case unidiff.LineAdded:
//	                fmt.Printf("+%s\n", line.Content)
//	            case unidiff.LineRemoved:
//	                fmt.Printf("-%s\n", line.Content)
//	            }
//	        }
//	    }
//	}
//
// # Statistics
//
// Get summary statistics about changes:
//
//	stats := diff.Stats()
//	fmt.Printf("Files: %d, Additions: %d, Deletions: %d\n",
//	    stats.FilesChanged, stats.Additions, stats.Deletions)
//
// # Line Numbers
//
// Track line numbers in both old and new versions:
//
//	for _, line := range hunk.Lines {
//	    if line.Type == unidiff.LineAdded {
//	        fmt.Printf("Added at line %d: %s\n", line.NewLineNum, line.Content)
//	    } else if line.Type == unidiff.LineRemoved {
//	        fmt.Printf("Removed from line %d: %s\n", line.OldLineNum, line.Content)
//	    }
//	}
package unidiff

import (
	"fmt"
	"strings"
)

// LineType represents the type of a diff line.
//
// Line types are used to distinguish between different kinds of lines in a
// unified diff: context lines (unchanged), added lines (prefixed with +),
// removed lines (prefixed with -), and header lines.
type LineType int

const (
	// LineContext represents an unchanged line that appears in both versions.
	// These lines provide context around changes and are prefixed with a space
	// in the diff format.
	LineContext LineType = iota

	// LineAdded represents a line that was added in the new version.
	// These lines are prefixed with + in the diff format and only have
	// a NewLineNum set (OldLineNum is 0).
	LineAdded

	// LineRemoved represents a line that was removed from the old version.
	// These lines are prefixed with - in the diff format and only have
	// an OldLineNum set (NewLineNum is 0).
	LineRemoved

	// LineHeader represents a file header line.
	// These are metadata lines at the start of each file section.
	LineHeader

	// LineHunkHeader represents a hunk header line.
	// These lines start with @@ and indicate the location and size of changes.
	LineHunkHeader
)

// String returns a human-readable string representation of the line type.
//
// Returns "context", "added", "removed", "header", "hunk", or "unknown".
func (t LineType) String() string {
	switch t {
	case LineContext:
		return "context"
	case LineAdded:
		return "added"
	case LineRemoved:
		return "removed"
	case LineHeader:
		return "header"
	case LineHunkHeader:
		return "hunk"
	default:
		return "unknown"
	}
}

// Line represents a single line in a diff with its content and metadata.
//
// Each line tracks its type (added, removed, or context), line numbers in both
// the old and new versions of the file, and the line content. The Content field
// contains the text without the leading marker (+, -, or space), while RawLine
// preserves the original line exactly as it appeared in the diff.
type Line struct {
	// Type indicates whether this line was added, removed, or is context.
	Type LineType

	// OldLineNum is the line number in the old file. This is 0 for added lines.
	OldLineNum int

	// NewLineNum is the line number in the new file. This is 0 for removed lines.
	NewLineNum int

	// Content is the line text without the leading +/- marker.
	Content string

	// RawLine is the original line including the marker (useful for debugging).
	RawLine string
}

// Hunk represents a contiguous block of changes within a file.
//
// A hunk corresponds to a section in the diff that starts with a @@ header line.
// It contains the location in both old and new files where the changes occur,
// along with the actual line-by-line changes. Multiple hunks may exist in a
// single file when changes are scattered across different parts of the file.
type Hunk struct {
	// OldStart is the starting line number in the old file (1-indexed).
	OldStart int

	// OldCount is the number of lines from the old file included in this hunk.
	// This includes both removed lines and context lines.
	OldCount int

	// NewStart is the starting line number in the new file (1-indexed).
	NewStart int

	// NewCount is the number of lines from the new file included in this hunk.
	// This includes both added lines and context lines.
	NewCount int

	// Header is the raw @@ line, e.g., "@@ -1,3 +1,4 @@ function main".
	Header string

	// Lines contains all the lines in this hunk (added, removed, and context).
	Lines []Line
}

// File represents all changes to a single file in a diff.
//
// A file may contain multiple hunks if changes are scattered throughout the file.
// The OldPath and NewPath are typically the same for modified files, but differ
// for renamed files. For new files, OldPath may be "/dev/null", and for deleted
// files, NewPath may be "/dev/null".
type File struct {
	// OldPath is the file path before changes (may include a/ prefix from git).
	OldPath string

	// NewPath is the file path after changes (may include b/ prefix from git).
	NewPath string

	// Hunks contains all the change blocks for this file.
	Hunks []Hunk
}

// Diff represents a complete diff, which may contain changes to multiple files.
//
// This is the top-level structure returned by Parse. It contains all files that
// were modified, added, or deleted in the diff.
type Diff struct {
	// Files contains all changed files in the diff.
	Files []File
}

// Parse parses a unified diff format string into a structured Diff.
//
// This function handles standard unified diff format as produced by git diff,
// diff -u, svn diff, and similar tools. It supports:
//   - Multiple files in a single diff
//   - Multiple hunks per file
//   - Line number tracking for old and new files
//   - Detection of added, removed, and context lines
//   - File renames (when old and new paths differ)
//
// The parser strips a/ and b/ prefixes from file paths (common in git diffs)
// and preserves both the raw line (with markers) and cleaned content.
//
// Example:
//
//	diffText := `diff --git a/main.go b/main.go
//	--- a/main.go
//	+++ b/main.go
//	@@ -1,2 +1,3 @@
//	 package main
//	+import "fmt"
//	 func main() {`
//
//	diff, err := unidiff.Parse(diffText)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Files changed: %d\n", len(diff.Files))
func Parse(diffText string) (*Diff, error) {
	lines := strings.Split(diffText, "\n")
	diff := &Diff{}

	var currentFile *File
	var currentHunk *Hunk
	oldLineNum := 0
	newLineNum := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "diff --git") {
			// Start of a new file
			if currentFile != nil && currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}
			if currentFile != nil {
				diff.Files = append(diff.Files, *currentFile)
			}

			currentFile = &File{}
			currentHunk = nil

		} else if strings.HasPrefix(line, "--- ") {
			// Old file path
			if currentFile != nil {
				path := strings.TrimPrefix(line, "--- ")
				// Remove a/ prefix if present
				path = strings.TrimPrefix(path, "a/")
				currentFile.OldPath = path
			}

		} else if strings.HasPrefix(line, "+++ ") {
			// New file path
			if currentFile != nil {
				path := strings.TrimPrefix(line, "+++ ")
				// Remove b/ prefix if present
				path = strings.TrimPrefix(path, "b/")
				currentFile.NewPath = path
			}

		} else if strings.HasPrefix(line, "@@") {
			// Start of a new hunk
			if currentFile != nil && currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			currentHunk = &Hunk{
				Header: line,
			}

			// Parse hunk header: @@ -oldStart,oldCount +newStart,newCount @@
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Parse old range
				oldRange := strings.TrimPrefix(parts[1], "-")
				fmt.Sscanf(oldRange, "%d,%d", &currentHunk.OldStart, &currentHunk.OldCount)

				// Parse new range
				newRange := strings.TrimPrefix(parts[2], "+")
				fmt.Sscanf(newRange, "%d,%d", &currentHunk.NewStart, &currentHunk.NewCount)
			}

			oldLineNum = currentHunk.OldStart
			newLineNum = currentHunk.NewStart

		} else if currentHunk != nil {
			// Process diff line
			var diffLine Line
			diffLine.RawLine = line

			if strings.HasPrefix(line, "+") {
				// Added line
				diffLine.Type = LineAdded
				diffLine.Content = strings.TrimPrefix(line, "+")
				diffLine.OldLineNum = 0
				diffLine.NewLineNum = newLineNum
				newLineNum++

			} else if strings.HasPrefix(line, "-") {
				// Removed line
				diffLine.Type = LineRemoved
				diffLine.Content = strings.TrimPrefix(line, "-")
				diffLine.OldLineNum = oldLineNum
				diffLine.NewLineNum = 0
				oldLineNum++

			} else if strings.HasPrefix(line, " ") || line == "" {
				// Context line (unchanged)
				diffLine.Type = LineContext
				diffLine.Content = strings.TrimPrefix(line, " ")
				diffLine.OldLineNum = oldLineNum
				diffLine.NewLineNum = newLineNum
				oldLineNum++
				newLineNum++

			} else {
				// Other lines (might be metadata, skip)
				continue
			}

			currentHunk.Lines = append(currentHunk.Lines, diffLine)
		}
	}

	// Append last hunk and file
	if currentFile != nil && currentHunk != nil {
		currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
	}
	if currentFile != nil {
		diff.Files = append(diff.Files, *currentFile)
	}

	return diff, nil
}

// Stats contains summary statistics about changes in a diff.
//
// This provides a high-level overview similar to the summary shown by
// git diff --stat or GitHub pull request summaries.
type Stats struct {
	// FilesChanged is the total number of files modified in the diff.
	FilesChanged int

	// Additions is the total number of lines added across all files.
	Additions int

	// Deletions is the total number of lines removed across all files.
	Deletions int
}

// Stats calculates and returns summary statistics for the diff.
//
// This counts the total number of files changed, lines added, and lines
// removed across all files and hunks in the diff. Context lines (unchanged)
// are not counted in additions or deletions.
//
// Example:
//
//	diff, _ := unidiff.Parse(diffText)
//	stats := diff.Stats()
//	fmt.Printf("%d files changed, %d insertions(+), %d deletions(-)\n",
//	    stats.FilesChanged, stats.Additions, stats.Deletions)
func (d *Diff) Stats() Stats {
	stats := Stats{
		FilesChanged: len(d.Files),
	}
	for _, file := range d.Files {
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				switch line.Type {
				case LineAdded:
					stats.Additions++
				case LineRemoved:
					stats.Deletions++
				}
			}
		}
	}
	return stats
}
