package git

import (
	"context"
	"strings"
)

// BranchOptions configures the Branches command with filtering and selection options.
type BranchOptions struct {
	// Remote includes only remote-tracking branches.
	Remote bool
	// All includes both local and remote-tracking branches.
	All bool
	// Contains filters to branches that contain the specified commit.
	// Can be a commit hash, tag, or branch name.
	Contains string
	// Merged filters to branches that have been merged into the specified ref.
	// Defaults to HEAD if value is "HEAD".
	Merged string
	// NoMerged filters to branches that have not been merged into the specified ref.
	// Defaults to HEAD if value is "HEAD".
	NoMerged string
}

// Branches returns the list of branches matching the specified options.
//
// By default, returns only local branches. Use BranchOptions.Remote or
// BranchOptions.All to include remote branches.
//
// Example:
//
//	// Get all local branches
//	branches, err := repo.Branches(ctx, git.BranchOptions{})
//
//	// Get all branches including remotes
//	branches, err := repo.Branches(ctx, git.BranchOptions{All: true})
//
//	// Get branches containing a specific commit
//	branches, err := repo.Branches(ctx, git.BranchOptions{
//	    Contains: "abc1234",
//	})
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

// LocalBranches returns only local branches (excluding remote-tracking branches).
// This is a convenience method equivalent to Branches(ctx, BranchOptions{}).
func (r *Repository) LocalBranches(ctx context.Context) ([]Branch, error) {
	return r.Branches(ctx, BranchOptions{})
}

// RemoteBranches returns only remote-tracking branches.
// This is a convenience method equivalent to Branches(ctx, BranchOptions{Remote: true}).
func (r *Repository) RemoteBranches(ctx context.Context) ([]Branch, error) {
	return r.Branches(ctx, BranchOptions{Remote: true})
}

// DefaultBranch returns the default branch name (typically "main" or "master").
//
// This attempts to determine the default branch by first checking the
// remote's HEAD, then checking for common default branch names, and
// finally falling back to the current branch.
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

// BranchExists checks if a branch with the given name exists.
//
// The name can be a local branch (e.g., "main") or a remote branch
// (e.g., "origin/main").
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
