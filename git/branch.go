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
	if err := checkArgs(opts.Contains, opts.Merged, opts.NoMerged); err != nil {
		return nil, err
	}

	// Format: full refname, objectname, upstream, HEAD indicator. The full
	// refname tells local branches (refs/heads/) from remote-tracking ones
	// (refs/remotes/); short names can't, since local names may contain "/".
	format := "%(refname)%00%(objectname:short)%00%(upstream:short)%00%(HEAD)"

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
			Hash:      parts[1],
			Upstream:  parts[2],
			IsCurrent: parts[3] == "*",
		}

		switch {
		case strings.HasPrefix(parts[0], "refs/heads/"):
			branch.Name = strings.TrimPrefix(parts[0], "refs/heads/")
		case strings.HasPrefix(parts[0], "refs/remotes/"):
			branch.Name = strings.TrimPrefix(parts[0], "refs/remotes/")
			branch.IsRemote = true
		default:
			// Not a branch, such as "(HEAD detached at abc1234)".
			continue
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
//
// Tags, commit hashes, and other refs that are not branches return false.
func (r *Repository) BranchExists(ctx context.Context, name string) (bool, error) {
	if err := checkArgs(name); err != nil {
		return false, err
	}
	for _, ref := range []string{"refs/heads/" + name, "refs/remotes/" + name} {
		_, err := r.run(ctx, "show-ref", "--verify", "--quiet", ref)
		if err == nil {
			return true, nil
		}
		if !strings.Contains(err.Error(), "exit status 1") {
			return false, err
		}
	}
	return false, nil
}
