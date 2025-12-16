package git

import (
	"context"
	"strings"
)

// BranchOptions configures the Branches command.
type BranchOptions struct {
	// Remote includes remote branches.
	Remote bool
	// All includes both local and remote branches.
	All bool
	// Contains filters branches containing this commit.
	Contains string
	// Merged filters branches merged into HEAD (or specified ref).
	Merged string
	// NoMerged filters branches not merged into HEAD (or specified ref).
	NoMerged string
}

// Branches returns the list of branches.
func (r *Repository) Branches(ctx context.Context, opts BranchOptions) ([]Branch, error) {
	// Format: refname, objectname, upstream, HEAD indicator
	format := "%(refname:short)%00%(objectname:short)%00%(upstream:short)%00%(HEAD)"

	args := []string{"branch", "--format=" + format}

	if opts.All {
		args = append(args, "-a")
	} else if opts.Remote {
		args = append(args, "-r")
	}

	if opts.Contains != "" {
		args = append(args, "--contains", opts.Contains)
	}
	if opts.Merged != "" {
		args = append(args, "--merged", opts.Merged)
	}
	if opts.NoMerged != "" {
		args = append(args, "--no-merged", opts.NoMerged)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, "\x00")
		if len(parts) < 4 {
			continue
		}

		branch := Branch{
			Name:      parts[0],
			Hash:      parts[1],
			Upstream:  parts[2],
			IsCurrent: parts[3] == "*",
			IsRemote:  strings.HasPrefix(parts[0], "remotes/") || strings.Contains(parts[0], "/"),
		}

		// Clean up remote branch names
		if strings.HasPrefix(branch.Name, "remotes/") {
			branch.Name = strings.TrimPrefix(branch.Name, "remotes/")
			branch.IsRemote = true
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// LocalBranches returns only local branches.
func (r *Repository) LocalBranches(ctx context.Context) ([]Branch, error) {
	return r.Branches(ctx, BranchOptions{})
}

// RemoteBranches returns only remote branches.
func (r *Repository) RemoteBranches(ctx context.Context) ([]Branch, error) {
	return r.Branches(ctx, BranchOptions{Remote: true})
}

// DefaultBranch returns the default branch name (main or master).
func (r *Repository) DefaultBranch(ctx context.Context) (string, error) {
	// Try to get from remote
	out, err := r.run(ctx, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		ref := strings.TrimSpace(string(out))
		return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
	}

	// Check for common defaults
	for _, name := range []string{"main", "master"} {
		_, err := r.run(ctx, "rev-parse", "--verify", name)
		if err == nil {
			return name, nil
		}
	}

	// Return the current branch as fallback
	return r.CurrentBranch(ctx)
}

// BranchExists checks if a branch exists.
func (r *Repository) BranchExists(ctx context.Context, name string) (bool, error) {
	_, err := r.run(ctx, "rev-parse", "--verify", name)
	if err != nil {
		if strings.Contains(err.Error(), "unknown revision") ||
			strings.Contains(err.Error(), "Needed a single revision") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
