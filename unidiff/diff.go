// Package unidiff provides parsing of unified diff format.
//
// This package parses unified diff output (like from git diff) into structured
// data that can be used for display, analysis, or transformation.
//
// Example usage:
//
//	diff, err := unidiff.Parse(diffText)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, file := range diff.Files {
//	    fmt.Printf("File: %s -> %s\n", file.OldPath, file.NewPath)
//	    for _, hunk := range file.Hunks {
//	        fmt.Printf("Hunk: %s\n", hunk.Header)
//	        for _, line := range hunk.Lines {
//	            fmt.Printf("%s: %s\n", line.Type, line.Content)
//	        }
//	    }
//	}
package unidiff

import (
	"fmt"
	"strings"
)

// LineType represents the type of a diff line
type LineType int

const (
	LineContext    LineType = iota // Unchanged line
	LineAdded                      // Added line
	LineRemoved                    // Removed line
	LineHeader                     // File header
	LineHunkHeader                 // Hunk header
)

// String returns a string representation of the line type
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

// Line represents a single line in a diff
type Line struct {
	Type       LineType
	OldLineNum int    // Line number in old file (0 if added)
	NewLineNum int    // Line number in new file (0 if removed)
	Content    string // Line content without the leading +/- marker
	RawLine    string // Original line including marker
}

// Hunk represents a contiguous block of changes
type Hunk struct {
	OldStart int    // Starting line in old file
	OldCount int    // Number of lines in old file
	NewStart int    // Starting line in new file
	NewCount int    // Number of lines in new file
	Header   string // The @@ line content
	Lines    []Line
}

// File represents changes to a single file
type File struct {
	OldPath string // Path to old file (before change)
	NewPath string // Path to new file (after change)
	Hunks   []Hunk
}

// Diff represents a complete diff (may contain multiple files)
type Diff struct {
	Files []File
}

// Parse parses a unified diff format string into a Diff structure.
// It handles standard unified diff format as produced by git diff, diff -u, etc.
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

// Stats returns statistics about the diff
type Stats struct {
	FilesChanged int
	Additions    int
	Deletions    int
}

// Stats calculates statistics for the diff
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
