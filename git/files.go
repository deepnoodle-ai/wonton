package git

import (
	"context"
	"strings"
)

// LsFilesOptions configures the LsFiles command for querying file status.
type LsFilesOptions struct {
	// Cached lists cached/staged files (this is the default if no other option is set).
	Cached bool
	// Modified lists files with unstaged modifications.
	Modified bool
	// Others lists untracked files (not in the index).
	Others bool
	// Deleted lists deleted files.
	Deleted bool
	// Ignored lists ignored files (requires Others to be true).
	// Respects .gitignore rules.
	Ignored bool
	// Stage includes stage information (not currently implemented).
	Stage bool
	// Path filters to files matching this path or pattern.
	Path string
}

// LsFiles returns a list of files based on the specified options.
//
// By default (with no options set), returns tracked files in the index.
// Use options to filter for modified, untracked, or ignored files.
func (r *Repository) LsFiles(ctx context.Context, opts LsFilesOptions) ([]string, error) {
	args := []string{"ls-files"}

	// Default to cached if nothing specified
	if !opts.Modified && !opts.Others && !opts.Deleted {
		opts.Cached = true
	}

	if opts.Cached {
		args = append(args, "--cached")
	}
	if opts.Modified {
		args = append(args, "--modified")
	}
	if opts.Others {
		args = append(args, "--others")
		if opts.Ignored {
			args = append(args, "--ignored", "--exclude-standard")
		} else {
			args = append(args, "--exclude-standard")
		}
	}
	if opts.Deleted {
		args = append(args, "--deleted")
	}
	if opts.Stage {
		args = append(args, "--stage")
	}

	if opts.Path != "" {
		args = append(args, "--", opts.Path)
	}

	return r.runLines(ctx, args...)
}

// TrackedFiles returns all tracked files in the repository.
// This includes all files in the git index.
func (r *Repository) TrackedFiles(ctx context.Context) ([]string, error) {
	return r.LsFiles(ctx, LsFilesOptions{Cached: true})
}

// UntrackedFiles returns untracked files (excluding files ignored by .gitignore).
// These are files that exist in the working directory but are not in the git index.
func (r *Repository) UntrackedFiles(ctx context.Context) ([]string, error) {
	return r.LsFiles(ctx, LsFilesOptions{Others: true})
}

// IgnoredFiles returns files that are ignored according to .gitignore rules.
// This includes files explicitly ignored and files in ignored directories.
func (r *Repository) IgnoredFiles(ctx context.Context) ([]string, error) {
	return r.LsFiles(ctx, LsFilesOptions{Others: true, Ignored: true})
}

// ModifiedFiles returns files with unstaged modifications.
// These are tracked files that have been modified but not staged.
func (r *Repository) ModifiedFiles(ctx context.Context) ([]string, error) {
	return r.LsFiles(ctx, LsFilesOptions{Modified: true})
}

// FileExists checks if a file is tracked in the repository (exists in the git index).
// Returns false for untracked files, even if they exist in the working directory.
func (r *Repository) FileExists(ctx context.Context, path string) (bool, error) {
	files, err := r.LsFiles(ctx, LsFilesOptions{Cached: true, Path: path})
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// ShowFile returns the contents of a file at a specific revision.
//
// The ref can be a commit hash, branch, tag, or any valid git revision.
// The path is relative to the repository root.
func (r *Repository) ShowFile(ctx context.Context, ref, path string) ([]byte, error) {
	return r.run(ctx, "show", ref+":"+path)
}

// FileAtRef returns the contents of a file at a specific ref as a string.
// This is a convenience wrapper around ShowFile that returns a string instead of bytes.
func (r *Repository) FileAtRef(ctx context.Context, ref, path string) (string, error) {
	out, err := r.ShowFile(ctx, ref, path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Contributors returns unique contributors to a file based on git history.
//
// This examines the commit history for the file and returns a list of unique
// authors who have modified it. If path is empty, returns contributors to the
// entire repository.
func (r *Repository) Contributors(ctx context.Context, path string) ([]Person, error) {
	args := []string{"log", "--format=%an%x00%ae", "--follow"}
	if path != "" {
		args = append(args, "--", path)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var contributors []Person

	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, "\x00")
		if len(parts) < 2 {
			continue
		}
		key := parts[0] + "<" + parts[1] + ">"
		if !seen[key] {
			seen[key] = true
			contributors = append(contributors, Person{
				Name:  parts[0],
				Email: parts[1],
			})
		}
	}

	return contributors, nil
}
