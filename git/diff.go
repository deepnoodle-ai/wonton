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
	// ContextLines is the number of context lines to show. Zero or a
	// negative value uses git's default, which is 3 unless diff.context is
	// set. No DiffOptions value requests zero context lines.
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
	numstatArgs = append(numstatArgs, "--numstat", "-z")
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

	// Parse numstat for file details. With -z, each entry ends in NUL and
	// paths are not quoted. A rename or copy has an empty path field followed
	// by the old and new paths as separate fields.
	fileStats := make(map[string]*DiffFile)
	fields := strings.Split(string(numstatOut), "\x00")
	for i := 0; i < len(fields); i++ {
		parts := strings.SplitN(fields[i], "\t", 3)
		if len(parts) < 3 {
			continue
		}

		df := &DiffFile{Path: parts[2]}
		if df.Path == "" {
			if i+2 >= len(fields) {
				break
			}
			df.OldPath, df.Path = fields[i+1], fields[i+2]
			df.Status = "renamed"
			i += 2
		}

		if parts[0] == "-" {
			df.Binary = true
		} else {
			df.Additions, _ = strconv.Atoi(parts[0])
			df.Deletions, _ = strconv.Atoi(parts[1])
		}

		diff.TotalAdded += df.Additions
		diff.TotalRemoved += df.Deletions
		diff.Files = append(diff.Files, *df)
		fileStats[df.Path] = df
	}

	// Determine file status from name-status
	nameStatusArgs := append([]string{}, args...)
	nameStatusArgs = append(nameStatusArgs, "--name-status", "-z")
	nameStatusOut, err := r.run(ctx, withPath(nameStatusArgs)...)
	if err != nil {
		return nil, err
	}

	// With -z, the status and each path are separate NUL-terminated fields.
	// A rename or copy has two paths, old then new.
	fields = strings.Split(string(nameStatusOut), "\x00")
	for f := 0; f+1 < len(fields); f++ {
		status := fields[f]
		if status == "" {
			continue
		}
		path := fields[f+1]
		f++
		var oldPath string
		if status[0] == 'R' || status[0] == 'C' {
			if f+1 >= len(fields) {
				break
			}
			oldPath, path = path, fields[f+1]
			f++
		}

		// Find the file in our list by its new path. Matching OldPath too
		// would let a copy source's own entry overwrite the copy.
		for i := range diff.Files {
			if diff.Files[i].Path == path {
				switch status[0] {
				case 'A':
					diff.Files[i].Status = "added"
				case 'D':
					diff.Files[i].Status = "deleted"
				case 'M':
					diff.Files[i].Status = "modified"
				case 'R':
					diff.Files[i].Status = "renamed"
					diff.Files[i].OldPath = oldPath
				case 'C':
					diff.Files[i].Status = "copied"
				case 'T':
					diff.Files[i].Status = "typechange"
				default:
					diff.Files[i].Status = "modified"
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
