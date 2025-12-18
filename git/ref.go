package git

import (
	"context"
	"strings"
)

// ResolveRef resolves a ref (branch, tag, HEAD, etc.) to a full commit hash.
//
// The ref can be any valid git reference including symbolic names like HEAD,
// branch names, tag names, or relative references like HEAD~1 or main^.
func (r *Repository) ResolveRef(ctx context.Context, ref string) (string, error) {
	out, err := r.run(ctx, "rev-parse", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ShortHash returns the abbreviated hash for a ref.
// Typically returns a 7-character hash, but may be longer to ensure uniqueness.
func (r *Repository) ShortHash(ctx context.Context, ref string) (string, error) {
	out, err := r.run(ctx, "rev-parse", "--short", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsAncestor checks if ancestor is an ancestor of descendant in the commit graph.
//
// Returns true if ancestor is reachable by following parent links from descendant.
// This is useful for determining if a branch has been merged or checking if
// one commit is older than another.
func (r *Repository) IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	_, err := r.run(ctx, "merge-base", "--is-ancestor", ancestor, descendant)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// MergeBase returns the best common ancestor of two commits.
//
// This is the most recent commit that is an ancestor of both commits.
// Useful for finding where two branches diverged.
func (r *Repository) MergeBase(ctx context.Context, a, b string) (string, error) {
	out, err := r.run(ctx, "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// AbbrevRef returns the abbreviated ref name for a symbolic ref.
// For example, converts "refs/heads/main" to "main".
func (r *Repository) AbbrevRef(ctx context.Context, ref string) (string, error) {
	out, err := r.run(ctx, "rev-parse", "--abbrev-ref", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RefExists checks if a ref exists.
// Returns true if the ref can be resolved to a commit.
func (r *Repository) RefExists(ctx context.Context, ref string) (bool, error) {
	_, err := r.run(ctx, "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") ||
			strings.Contains(err.Error(), "unknown revision") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CommitCount returns the total number of commits reachable from a ref.
// Defaults to HEAD if ref is empty.
func (r *Repository) CommitCount(ctx context.Context, ref string) (int, error) {
	if ref == "" {
		ref = "HEAD"
	}
	out, err := r.run(ctx, "rev-list", "--count", ref)
	if err != nil {
		return 0, err
	}

	var count int
	_, err = strings.NewReader(strings.TrimSpace(string(out))).Read([]byte{})
	if err != nil {
		return 0, err
	}

	// Parse the count
	for _, c := range strings.TrimSpace(string(out)) {
		if c >= '0' && c <= '9' {
			count = count*10 + int(c-'0')
		}
	}

	return count, nil
}

// CommitCountBetween returns the number of commits between two refs.
// Uses range syntax (from..to) to count commits reachable from "to" but not from "from".
func (r *Repository) CommitCountBetween(ctx context.Context, from, to string) (int, error) {
	out, err := r.run(ctx, "rev-list", "--count", from+".."+to)
	if err != nil {
		return 0, err
	}

	var count int
	for _, c := range strings.TrimSpace(string(out)) {
		if c >= '0' && c <= '9' {
			count = count*10 + int(c-'0')
		}
	}

	return count, nil
}
