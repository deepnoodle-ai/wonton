package git

import (
	"context"
	"strings"
)

// LsFilesOptions configures the LsFiles command.
type LsFilesOptions struct {
	// Cached lists cached/staged files (default).
	Cached bool
	// Modified lists files with unstaged modifications.
	Modified bool
	// Others lists untracked files.
	Others bool
	// Deleted lists deleted files.
	Deleted bool
	// Ignored lists ignored files (requires Others).
	Ignored bool
	// Stage includes stage information.
	Stage bool
	// Path filters to files matching this path/pattern.
	Path string
}

// LsFiles returns a list of tracked files.
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
func (r *Repository) TrackedFiles(ctx context.Context) ([]string, error) {
	return r.LsFiles(ctx, LsFilesOptions{Cached: true})
}

// UntrackedFiles returns untracked files (excluding ignored).
func (r *Repository) UntrackedFiles(ctx context.Context) ([]string, error) {
	return r.LsFiles(ctx, LsFilesOptions{Others: true})
}

// IgnoredFiles returns ignored files.
func (r *Repository) IgnoredFiles(ctx context.Context) ([]string, error) {
	return r.LsFiles(ctx, LsFilesOptions{Others: true, Ignored: true})
}

// ModifiedFiles returns files with unstaged modifications.
func (r *Repository) ModifiedFiles(ctx context.Context) ([]string, error) {
	return r.LsFiles(ctx, LsFilesOptions{Modified: true})
}

// FileExists checks if a file is tracked in the repository.
func (r *Repository) FileExists(ctx context.Context, path string) (bool, error) {
	files, err := r.LsFiles(ctx, LsFilesOptions{Cached: true, Path: path})
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// ShowFile returns the contents of a file at a specific revision.
func (r *Repository) ShowFile(ctx context.Context, ref, path string) ([]byte, error) {
	return r.run(ctx, "show", ref+":"+path)
}

// FileAtRef returns the contents of a file at a specific ref.
func (r *Repository) FileAtRef(ctx context.Context, ref, path string) (string, error) {
	out, err := r.ShowFile(ctx, ref, path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Contributors returns unique contributors to a file based on git history.
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
