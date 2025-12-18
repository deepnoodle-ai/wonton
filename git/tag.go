package git

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// TagOptions configures the Tags command with filtering and sorting options.
type TagOptions struct {
	// Pattern filters tags by glob pattern (e.g., "v*" for version tags).
	Pattern string
	// Sort specifies the sort order (e.g., "-creatordate" for newest first).
	// Use a minus prefix for descending order.
	Sort string
	// Contains filters to tags that contain the specified commit.
	// Useful for finding which tags include a particular commit.
	Contains string
	// PointsAt filters to tags that point directly at the specified commit.
	// Unlike Contains, this only matches tags pointing at the exact commit.
	PointsAt string
}

// Tags returns the list of tags matching the specified options.
//
// By default, returns all tags. Use TagOptions to filter by pattern or commit.
//
// Examples:
//
//	// Get all tags
//	tags, err := repo.Tags(ctx, git.TagOptions{})
//
//	// Get version tags sorted by date
//	tags, err := repo.Tags(ctx, git.TagOptions{
//	    Pattern: "v*",
//	    Sort:    "-creatordate",
//	})
//
//	// Get tags containing a specific commit
//	tags, err := repo.Tags(ctx, git.TagOptions{
//	    Contains: "abc1234",
//	})
func (r *Repository) Tags(ctx context.Context, opts TagOptions) ([]Tag, error) {
	// Format: refname, objectname, type, taggername, taggeremail, taggerdate, subject
	format := "%(refname:short)%00%(objectname:short)%00%(*objectname:short)%00%(objecttype)%00%(taggername)%00%(taggeremail)%00%(creatordate:unix)%00%(subject)"

	args := []string{"tag", "-l", "--format=" + format}

	if opts.Pattern != "" {
		args = append(args, opts.Pattern)
	}
	if opts.Sort != "" {
		args = append(args, "--sort="+opts.Sort)
	}
	if opts.Contains != "" {
		args = append(args, "--contains", opts.Contains)
	}
	if opts.PointsAt != "" {
		args = append(args, "--points-at", opts.PointsAt)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	var tags []Tag
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\x00")
		if len(parts) < 8 {
			continue
		}

		tag := Tag{
			Name: parts[0],
			Hash: parts[1],
		}

		// Annotated tags have a dereferenced commit
		if parts[2] != "" {
			tag.Commit = parts[2]
		} else {
			tag.Commit = parts[1] // Lightweight tag points directly to commit
		}

		// Annotated tags have tagger info
		if parts[3] == "tag" && parts[4] != "" {
			tag.Tagger = &Person{
				Name:  parts[4],
				Email: strings.Trim(parts[5], "<>"),
			}
			if ts, err := strconv.ParseInt(parts[6], 10, 64); err == nil && ts > 0 {
				t := time.Unix(ts, 0)
				tag.Date = &t
			}
			tag.Message = parts[7]
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

// LatestTag returns the most recent tag by creation date.
// Returns nil if the repository has no tags.
func (r *Repository) LatestTag(ctx context.Context) (*Tag, error) {
	tags, err := r.Tags(ctx, TagOptions{Sort: "-creatordate"})
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, nil
	}
	return &tags[0], nil
}

// TagsForCommit returns tags pointing directly at a specific commit.
// The ref can be a commit hash, branch, or any valid git reference.
func (r *Repository) TagsForCommit(ctx context.Context, ref string) ([]Tag, error) {
	return r.Tags(ctx, TagOptions{PointsAt: ref})
}

// Describe returns a human-readable name for a commit based on tags.
//
// The format is "tag-distance-ghash" where distance is the number of commits
// since the tag. If the commit is tagged, returns just the tag name.
// Returns something like "v1.0.0-3-g1234567" or "v1.0.0".
func (r *Repository) Describe(ctx context.Context, ref string) (string, error) {
	args := []string{"describe", "--tags", "--always"}
	if ref != "" {
		args = append(args, ref)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// DescribeLong returns the long form description.
//
// Unlike Describe, this always includes the distance and hash, even for
// tagged commits. Returns something like "v1.0.0-0-g1234567" for tagged
// commits and "v1.0.0-3-g1234567" for commits after a tag.
func (r *Repository) DescribeLong(ctx context.Context, ref string) (string, error) {
	args := []string{"describe", "--tags", "--long", "--always"}
	if ref != "" {
		args = append(args, ref)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
