package git

import (
	"bufio"
	"bytes"
	"context"
	"strconv"
	"strings"
)

// DiffOptions configures the Diff command.
type DiffOptions struct {
	// Staged shows changes staged for commit (git diff --cached).
	Staged bool
	// Ref compares working directory against this ref.
	Ref string
	// From and To compare two refs (From..To).
	From string
	To   string
	// Path filters diff to this path.
	Path string
	// ContextLines is the number of context lines (default 3).
	ContextLines int
	// IncludePatch includes the actual diff content.
	IncludePatch bool
	// NameOnly returns only file names, not stats.
	NameOnly bool
}

// Diff returns the diff for the working directory or between refs.
func (r *Repository) Diff(ctx context.Context, opts DiffOptions) (*Diff, error) {
	// Build base args without path filter (path must come after options)
	args := []string{"diff"}

	if opts.ContextLines > 0 {
		args = append(args, "-U"+strconv.Itoa(opts.ContextLines))
	}

	if opts.Staged {
		args = append(args, "--cached")
	}

	if opts.From != "" && opts.To != "" {
		args = append(args, opts.From+".."+opts.To)
	} else if opts.Ref != "" {
		args = append(args, opts.Ref)
	}

	// Helper to append path filter after options
	withPath := func(a []string) []string {
		if opts.Path != "" {
			return append(a, "--", opts.Path)
		}
		return a
	}

	// First get the stat summary
	statArgs := append([]string{}, args...)
	statArgs = append(statArgs, "--stat", "--stat-width=1000")
	statOut, err := r.run(ctx, withPath(statArgs)...)
	if err != nil {
		return nil, err
	}

	// Get numstat for precise counts
	numstatArgs := append([]string{}, args...)
	numstatArgs = append(numstatArgs, "--numstat")
	numstatOut, err := r.run(ctx, withPath(numstatArgs)...)
	if err != nil {
		return nil, err
	}

	diff := &Diff{}

	// Parse stat output for the summary line
	statLines := strings.Split(strings.TrimSpace(string(statOut)), "\n")
	if len(statLines) > 0 {
		diff.Stats = statLines[len(statLines)-1]
	}

	// Parse numstat for file details
	fileStats := make(map[string]*DiffFile)
	scanner := bufio.NewScanner(bytes.NewReader(numstatOut))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		df := &DiffFile{Path: parts[2]}

		if parts[0] == "-" {
			df.Binary = true
		} else {
			df.Additions, _ = strconv.Atoi(parts[0])
			df.Deletions, _ = strconv.Atoi(parts[1])
		}

		// Check for rename (old -> new format in path)
		if strings.Contains(parts[2], " => ") {
			// Handle various rename formats
			df.Path, df.OldPath = parseRenamePath(parts[2])
			df.Status = "renamed"
		}

		diff.TotalAdded += df.Additions
		diff.TotalRemoved += df.Deletions
		diff.Files = append(diff.Files, *df)
		fileStats[df.Path] = df
	}

	// Determine file status from name-status
	nameStatusArgs := append([]string{}, args...)
	nameStatusArgs = append(nameStatusArgs, "--name-status")
	nameStatusOut, err := r.run(ctx, withPath(nameStatusArgs)...)
	if err != nil {
		return nil, err
	}

	scanner = bufio.NewScanner(bytes.NewReader(nameStatusOut))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		path := parts[len(parts)-1]

		// Find the file in our list
		for i := range diff.Files {
			if diff.Files[i].Path == path || diff.Files[i].OldPath == path {
				if diff.Files[i].Status == "" {
					switch status[0] {
					case 'A':
						diff.Files[i].Status = "added"
					case 'D':
						diff.Files[i].Status = "deleted"
					case 'M':
						diff.Files[i].Status = "modified"
					case 'R':
						diff.Files[i].Status = "renamed"
						if len(parts) >= 3 {
							diff.Files[i].OldPath = parts[1]
							diff.Files[i].Path = parts[2]
						}
					case 'C':
						diff.Files[i].Status = "copied"
					case 'T':
						diff.Files[i].Status = "typechange"
					default:
						diff.Files[i].Status = "modified"
					}
				}
				break
			}
		}
	}

	// Get actual patch content if requested
	if opts.IncludePatch {
		patchOut, err := r.run(ctx, withPath(args)...)
		if err != nil {
			return nil, err
		}
		patches := parsePatchContent(string(patchOut))
		for path, patch := range patches {
			for i := range diff.Files {
				if diff.Files[i].Path == path {
					diff.Files[i].Patch = patch
					break
				}
			}
		}
	}

	return diff, nil
}

// DiffFile returns the diff for a specific file.
func (r *Repository) DiffFile(ctx context.Context, path string, opts DiffOptions) (*DiffFile, error) {
	opts.Path = path
	opts.IncludePatch = true
	diff, err := r.Diff(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(diff.Files) == 0 {
		return nil, nil
	}
	return &diff.Files[0], nil
}

// parseRenamePath handles git's rename path formats like:
// - "old => new"
// - "{old => new}/file"
// - "dir/{old => new}"
func parseRenamePath(path string) (newPath, oldPath string) {
	if strings.Contains(path, "{") && strings.Contains(path, "}") {
		// Format: prefix{old => new}suffix
		start := strings.Index(path, "{")
		end := strings.Index(path, "}")
		if start < end {
			prefix := path[:start]
			suffix := path[end+1:]
			inner := path[start+1 : end]
			parts := strings.Split(inner, " => ")
			if len(parts) == 2 {
				oldPath = prefix + parts[0] + suffix
				newPath = prefix + parts[1] + suffix
				return
			}
		}
	}

	// Simple format: "old => new"
	parts := strings.Split(path, " => ")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[0])
	}

	return path, ""
}

// parsePatchContent splits a unified diff into per-file patches.
func parsePatchContent(patch string) map[string]string {
	result := make(map[string]string)
	if patch == "" {
		return result
	}

	// Split by "diff --git" markers
	parts := strings.Split(patch, "diff --git ")
	for _, part := range parts[1:] { // Skip first empty part
		lines := strings.SplitN(part, "\n", 2)
		if len(lines) < 2 {
			continue
		}

		// Extract filename from "a/file b/file" header
		header := lines[0]
		fields := strings.Fields(header)
		if len(fields) >= 2 {
			// Remove "a/" or "b/" prefix
			path := strings.TrimPrefix(fields[1], "b/")
			result[path] = "diff --git " + part
		}
	}

	return result
}
