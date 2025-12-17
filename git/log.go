package git

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// LogOptions configures the Log command.
type LogOptions struct {
	// Limit is the maximum number of commits to return.
	Limit int
	// Since returns commits after this date.
	Since time.Time
	// Until returns commits before this date.
	Until time.Time
	// Author filters commits by author (substring match).
	Author string
	// Grep filters commits by message (substring match).
	Grep string
	// Path filters commits affecting this path.
	Path string
	// Ref is the starting point (branch, tag, or commit). Defaults to HEAD.
	Ref string
	// FirstParent follows only the first parent of merge commits.
	FirstParent bool
	// All includes all refs.
	All bool
	// IncludeBody includes the full commit message body.
	IncludeBody bool
}

// Log returns the commit history.
func (r *Repository) Log(ctx context.Context, opts LogOptions) ([]Commit, error) {
	// Use a custom format for parsing with record separator (0x1e) at end
	// Format: hash|short|author_name|author_email|committer_name|committer_email|subject|body|timestamp|parents<RS>
	// Using %x1f (unit separator) between fields and %x1e (record separator) between records
	format := "%H%x1f%h%x1f%an%x1f%ae%x1f%cn%x1f%ce%x1f%s%x1f%b%x1f%ct%x1f%P%x1e"

	args := []string{"log", "--format=" + format}

	if opts.Limit > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Limit))
	}
	if !opts.Since.IsZero() {
		args = append(args, "--since="+opts.Since.Format(time.RFC3339))
	}
	if !opts.Until.IsZero() {
		args = append(args, "--until="+opts.Until.Format(time.RFC3339))
	}
	if opts.Author != "" {
		args = append(args, "--author="+opts.Author)
	}
	if opts.Grep != "" {
		args = append(args, "--grep="+opts.Grep)
	}
	if opts.FirstParent {
		args = append(args, "--first-parent")
	}
	if opts.All {
		args = append(args, "--all")
	}

	ref := opts.Ref
	if ref == "" {
		ref = "HEAD"
	}
	args = append(args, ref)

	if opts.Path != "" {
		args = append(args, "--", opts.Path)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		if strings.Contains(err.Error(), "unknown revision") {
			return nil, ErrNoCommits
		}
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	// Split by record separator (0x1e)
	records := strings.Split(string(out), "\x1e")
	var commits []Commit

	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}

		// Split by unit separator (0x1f)
		parts := strings.Split(record, "\x1f")
		if len(parts) < 9 {
			continue
		}

		timestamp, _ := strconv.ParseInt(parts[8], 10, 64)
		commit := Commit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Author: Person{
				Name:  parts[2],
				Email: parts[3],
			},
			Committer: Person{
				Name:  parts[4],
				Email: parts[5],
			},
			Subject:   parts[6],
			Timestamp: time.Unix(timestamp, 0),
		}

		if opts.IncludeBody && parts[7] != "" {
			commit.Body = strings.TrimSpace(parts[7])
		}

		if len(parts) > 9 && parts[9] != "" {
			commit.ParentHashes = strings.Fields(parts[9])
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// Show returns details for a specific commit.
func (r *Repository) Show(ctx context.Context, ref string) (*Commit, error) {
	commits, err := r.Log(ctx, LogOptions{
		Ref:         ref,
		Limit:       1,
		IncludeBody: true,
	})
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, ErrNoCommits
	}
	return &commits[0], nil
}

// CommitsBetween returns commits between two refs (exclusive of from, inclusive of to).
func (r *Repository) CommitsBetween(ctx context.Context, from, to string) ([]Commit, error) {
	return r.Log(ctx, LogOptions{
		Ref:         from + ".." + to,
		IncludeBody: true,
	})
}
