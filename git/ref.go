package git

import (
	"context"
	"strings"
)

// ResolveRef resolves a ref (branch, tag, HEAD, etc.) to a commit hash.
func (r *Repository) ResolveRef(ctx context.Context, ref string) (string, error) {
	out, err := r.run(ctx, "rev-parse", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ShortHash returns the short hash for a ref.
func (r *Repository) ShortHash(ctx context.Context, ref string) (string, error) {
	out, err := r.run(ctx, "rev-parse", "--short", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsAncestor checks if ancestor is an ancestor of descendant.
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
func (r *Repository) MergeBase(ctx context.Context, a, b string) (string, error) {
	out, err := r.run(ctx, "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// AbbrevRef returns the abbreviated ref name for a symbolic ref.
func (r *Repository) AbbrevRef(ctx context.Context, ref string) (string, error) {
	out, err := r.run(ctx, "rev-parse", "--abbrev-ref", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RefExists checks if a ref exists.
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

// CommitCount returns the number of commits in the history.
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
