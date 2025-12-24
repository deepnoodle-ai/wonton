package tui

import (
	"github.com/deepnoodle-ai/wonton/unidiff"
)

// Re-export diff types from unidiff package for backward compatibility
type (
	// DiffLineType represents the type of a diff line
	DiffLineType = unidiff.LineType
)

// Re-export DiffLineType constants with tui naming convention
const (
	DiffLineContext    = unidiff.LineContext
	DiffLineAdded      = unidiff.LineAdded
	DiffLineRemoved    = unidiff.LineRemoved
	DiffLineHeader     = unidiff.LineHeader
	DiffLineHunkHeader = unidiff.LineHunkHeader
)

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type       DiffLineType
	OldLineNum int    // Line number in old file (0 if added)
	NewLineNum int    // Line number in new file (0 if removed)
	Content    string // Line content without the leading +/- marker
	RawLine    string // Original line including marker
}

// DiffHunk represents a contiguous block of changes
type DiffHunk struct {
	OldStart int    // Starting line in old file
	OldCount int    // Number of lines in old file
	NewStart int    // Starting line in new file
	NewCount int    // Number of lines in new file
	Header   string // The @@ line content
	Lines    []DiffLine
}

// DiffFile represents changes to a single file
type DiffFile struct {
	OldPath string // Path to old file (before change)
	NewPath string // Path to new file (after change)
	Hunks   []DiffHunk
}

// Diff represents a complete diff (may contain multiple files)
type Diff struct {
	Files []DiffFile
}

// ParseUnifiedDiff parses a unified diff format string into a Diff structure
func ParseUnifiedDiff(diffText string) (*Diff, error) {
	parsed, err := unidiff.Parse(diffText)
	if err != nil {
		return nil, err
	}

	// Convert unidiff types to tui types
	diff := &Diff{
		Files: make([]DiffFile, len(parsed.Files)),
	}

	for i, file := range parsed.Files {
		diff.Files[i] = DiffFile{
			OldPath: file.OldPath,
			NewPath: file.NewPath,
			Hunks:   make([]DiffHunk, len(file.Hunks)),
		}
		for j, hunk := range file.Hunks {
			diff.Files[i].Hunks[j] = DiffHunk{
				OldStart: hunk.OldStart,
				OldCount: hunk.OldCount,
				NewStart: hunk.NewStart,
				NewCount: hunk.NewCount,
				Header:   hunk.Header,
				Lines:    make([]DiffLine, len(hunk.Lines)),
			}
			for k, line := range hunk.Lines {
				diff.Files[i].Hunks[j].Lines[k] = DiffLine{
					Type:       line.Type,
					OldLineNum: line.OldLineNum,
					NewLineNum: line.NewLineNum,
					Content:    line.Content,
					RawLine:    line.RawLine,
				}
			}
		}
	}

	return diff, nil
}
