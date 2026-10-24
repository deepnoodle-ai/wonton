package git

import (
	"context"
	"strconv"
	"strings"
)

// DiffOptions configures the Diff command with various filtering and formatting options.
type DiffOptions struct {
	// Staged shows changes staged for commit (equivalent to git diff --cached).
	Staged bool
	// Ref compares working directory against this ref.
	// Cannot be used with From/To.
	Ref string
	// From and To compare two refs (From..To range syntax).
	// Cannot be used with Ref or Staged.
	From string
	To   string
	// Path filters diff to this file or directory path.
	Path string
	// ContextLines is the number of context lines to show (default is 3).
	// Set to 0 to show only changed lines.
	ContextLines int
	// IncludePatch includes the actual unified diff content in the Patch field.
	IncludePatch bool
	// NameOnly returns only file names without stats (not currently used).
	NameOnly bool
}

// Diff returns the diff for the working directory or between refs.
//
// By default, shows unstaged changes in the working directory. Use DiffOptions
// to compare different revisions or show staged changes.
//
// Examples:
//
//	// Unstaged changes
//	diff, err := repo.Diff(ctx, git.DiffOptions{})
//
//	// Staged changes
//	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
//
//	// Compare working directory to a commit
//	diff, err := repo.Diff(ctx, git.DiffOptions{Ref: "HEAD~1"})
//
//	// Compare two commits
//	diff, err := repo.Diff(ctx, git.DiffOptions{
//	    From: "v1.0.0",
//	    To:   "v2.0.0",
//	})
//
//	// Get diff with patch content
//	diff, err := repo.Diff(ctx, git.DiffOptions{
//	    Staged:       true,
//	    IncludePatch: true,
//	})
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
		if err := checkArgs(opts.From, opts.To); err != nil {
			return nil, err
		}
		args = append(args, opts.From+".."+opts.To)
	} else if opts.Ref != "" {
		if err := checkArgs(opts.Ref); err != nil {
			return nil, err
		}
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
	scanner := newScanner(numstatOut)
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

	scanner = newScanner(nameStatusOut)
	for scanner.Scan() {
		line := scanner.Text()
		// Fields are tab-separated; paths may contain spaces.
		parts := strings.Split(line, "\t")
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

// DiffFile returns the diff for a specific file, including patch content.
//
// This is a convenience method that automatically sets IncludePatch to true
// and filters to the specified path. Returns nil if the file has no changes.
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
				oldPath = joinRenamePart(prefix, parts[0], suffix)
				newPath = joinRenamePart(prefix, parts[1], suffix)
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

// patchHeaderPath returns the new path from the "a/<old> b/<new>" part of a
// "diff --git" header. Paths may contain spaces, so it first tries the common
// case where both sides name the same path, then falls back to the last " b/".
func patchHeaderPath(header string) string {
	if n := len(header) - len("a/ b/"); n > 0 && n%2 == 0 {
		path := header[2 : 2+n/2]
		if header == "a/"+path+" b/"+path {
			return path
		}
	}
	if i := strings.LastIndex(header, " b/"); i >= 0 {
		return header[i+len(" b/"):]
	}
	return ""
}

// joinRenamePart rebuilds one side of a "prefix{old => new}suffix" rename.
// When the braced part is empty (a move into or out of a directory, such as
// "dir/{sub => }/file"), the separator it would have carried is dropped so the
// result has no leading or doubled slash.
func joinRenamePart(prefix, mid, suffix string) string {
	if mid == "" && strings.HasPrefix(suffix, "/") && (prefix == "" || strings.HasSuffix(prefix, "/")) {
		suffix = suffix[1:]
	}
	return prefix + mid + suffix
}

// parsePatchContent splits a unified diff into per-file patches.
func parsePatchContent(patch string) map[string]string {
	result := make(map[string]string)
	if patch == "" {
		return result
	}

	var currentPath string
	var currentPatch strings.Builder

	scanner := newScanner([]byte(patch))
	for scanner.Scan() {
		line := scanner.Text()

		// specific check for start of new file diff
		if strings.HasPrefix(line, "diff --git ") {
			// Save previous patch
			if currentPath != "" {
				result[currentPath] = currentPatch.String()
			}

			// Start new patch
			currentPatch.Reset()
			currentPatch.WriteString(line)
			currentPatch.WriteString("\n")

			currentPath = patchHeaderPath(strings.TrimPrefix(line, "diff --git "))
		} else {
			if currentPath != "" { // Only append if we're inside a patch
				currentPatch.WriteString(line)
				currentPatch.WriteString("\n")
			}
		}
	}

	// Save last patch
	if currentPath != "" {
		result[currentPath] = currentPatch.String()
	}

	return result
}
