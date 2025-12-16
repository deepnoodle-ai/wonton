// Package git provides a wrapper around git commands for common read operations.
//
// It's designed for integration with LLM tools, providing structured data
// from git operations instead of raw command output.
//
// Basic usage:
//
//	repo, err := git.Open(".")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	status, err := repo.Status(ctx)
//	commits, err := repo.Log(ctx, git.LogOptions{Limit: 10})
//	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Common errors.
var (
	ErrNotRepository = errors.New("not a git repository")
	ErrNoCommits     = errors.New("no commits yet")
)

// Repository represents a git repository.
type Repository struct {
	// Path is the root directory of the repository.
	Path string
	// GitDir is the .git directory path.
	GitDir string
}

// Open opens a git repository at the given path.
// The path can be any directory within the repository.
func Open(path string) (*Repository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Use git rev-parse to find the repository root
	cmd := exec.Command("git", "-C", absPath, "rev-parse", "--show-toplevel", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		return nil, ErrNotRepository
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return nil, ErrNotRepository
	}

	return &Repository{
		Path:   lines[0],
		GitDir: lines[1],
	}, nil
}

// Commit represents a git commit.
type Commit struct {
	Hash        string    `json:"hash"`
	ShortHash   string    `json:"short_hash"`
	Author      Person    `json:"author"`
	Committer   Person    `json:"committer"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body,omitempty"`
	ParentHashes []string `json:"parent_hashes,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// Person represents an author or committer.
type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Status represents the working directory status.
type Status struct {
	Branch        string       `json:"branch"`
	Upstream      string       `json:"upstream,omitempty"`
	Ahead         int          `json:"ahead"`
	Behind        int          `json:"behind"`
	Staged        []FileStatus `json:"staged,omitempty"`
	Unstaged      []FileStatus `json:"unstaged,omitempty"`
	Untracked     []string     `json:"untracked,omitempty"`
	Conflicts     []string     `json:"conflicts,omitempty"`
	IsClean       bool         `json:"is_clean"`
	HasUntracked  bool         `json:"has_untracked"`
}

// FileStatus represents the status of a single file.
type FileStatus struct {
	Path    string `json:"path"`
	OldPath string `json:"old_path,omitempty"` // For renames
	Status  string `json:"status"`             // modified, added, deleted, renamed, copied
}

// Branch represents a git branch.
type Branch struct {
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	Upstream  string `json:"upstream,omitempty"`
	IsCurrent bool   `json:"is_current"`
	IsRemote  bool   `json:"is_remote"`
}

// Tag represents a git tag.
type Tag struct {
	Name    string     `json:"name"`
	Hash    string     `json:"hash"`
	Commit  string     `json:"commit,omitempty"`  // For annotated tags
	Tagger  *Person    `json:"tagger,omitempty"`  // For annotated tags
	Message string     `json:"message,omitempty"` // For annotated tags
	Date    *time.Time `json:"date,omitempty"`    // For annotated tags
}

// Remote represents a git remote.
type Remote struct {
	Name     string `json:"name"`
	FetchURL string `json:"fetch_url"`
	PushURL  string `json:"push_url"`
}

// DiffFile represents changes to a single file in a diff.
type DiffFile struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"` // For renames
	Status    string `json:"status"`             // added, modified, deleted, renamed
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
	Patch     string `json:"patch,omitempty"` // The actual diff content
}

// Diff represents a git diff result.
type Diff struct {
	Files       []DiffFile `json:"files"`
	TotalAdded  int        `json:"total_added"`
	TotalRemoved int       `json:"total_removed"`
	Stats       string     `json:"stats,omitempty"` // Summary line
}

// BlameLine represents a single line of git blame output.
type BlameLine struct {
	Hash       string    `json:"hash"`
	Author     string    `json:"author"`
	Email      string    `json:"email"`
	Timestamp  time.Time `json:"timestamp"`
	LineNumber int       `json:"line_number"`
	Content    string    `json:"content"`
}

// run executes a git command and returns the output.
func (r *Repository) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.Path}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return out, nil
}

// runLines executes a git command and returns lines of output.
func (r *Repository) runLines(ctx context.Context, args ...string) ([]string, error) {
	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n"), nil
}

// Head returns the current HEAD commit hash.
func (r *Repository) Head(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		if strings.Contains(err.Error(), "unknown revision") {
			return "", ErrNoCommits
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CurrentBranch returns the name of the current branch.
// Returns empty string if in detached HEAD state.
func (r *Repository) CurrentBranch(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		// Detached HEAD state
		if strings.Contains(err.Error(), "not a symbolic ref") {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsClean returns true if the working directory has no changes.
func (r *Repository) IsClean(ctx context.Context) (bool, error) {
	out, err := r.run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) == 0, nil
}

// Status returns the working directory status.
func (r *Repository) Status(ctx context.Context) (*Status, error) {
	out, err := r.run(ctx, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return nil, err
	}

	status := &Status{}
	scanner := bufio.NewScanner(bytes.NewReader(out))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		switch {
		case strings.HasPrefix(line, "# branch.oid"):
			// Skip commit hash
		case strings.HasPrefix(line, "# branch.head"):
			status.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.upstream"):
			status.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab"):
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				status.Ahead, _ = strconv.Atoi(strings.TrimPrefix(parts[2], "+"))
				status.Behind, _ = strconv.Atoi(strings.TrimPrefix(parts[3], "-"))
			}
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 "):
			// Changed entry
			fs := parseStatusEntry(line)
			if fs != nil {
				if fs.staged {
					status.Staged = append(status.Staged, fs.FileStatus)
				}
				if fs.unstaged {
					status.Unstaged = append(status.Unstaged, fs.FileStatus)
				}
			}
		case strings.HasPrefix(line, "? "):
			// Untracked
			status.Untracked = append(status.Untracked, line[2:])
			status.HasUntracked = true
		case strings.HasPrefix(line, "u "):
			// Unmerged (conflict)
			parts := strings.Fields(line)
			if len(parts) > 10 {
				status.Conflicts = append(status.Conflicts, parts[10])
			}
		}
	}

	status.IsClean = len(status.Staged) == 0 &&
		len(status.Unstaged) == 0 &&
		len(status.Untracked) == 0 &&
		len(status.Conflicts) == 0

	return status, scanner.Err()
}

type statusEntry struct {
	FileStatus
	staged   bool
	unstaged bool
}

func parseStatusEntry(line string) *statusEntry {
	parts := strings.Fields(line)
	if len(parts) < 9 {
		return nil
	}

	xy := parts[1]
	path := parts[8]

	entry := &statusEntry{
		FileStatus: FileStatus{Path: path},
	}

	// First character is staged status
	if xy[0] != '.' {
		entry.staged = true
		entry.FileStatus.Status = statusCodeToString(xy[0])
	}
	// Second character is unstaged status
	if xy[1] != '.' {
		entry.unstaged = true
		entry.FileStatus.Status = statusCodeToString(xy[1])
	}

	// Handle renames (line type "2")
	if parts[0] == "2" && len(parts) >= 10 {
		entry.FileStatus.OldPath = parts[9]
		entry.FileStatus.Status = "renamed"
	}

	return entry
}

func statusCodeToString(c byte) string {
	switch c {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'T':
		return "typechange"
	default:
		return "unknown"
	}
}
