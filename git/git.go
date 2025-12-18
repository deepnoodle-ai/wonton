// Package git provides a wrapper around git commands for common read operations.
//
// This package is designed for building tools and applications that need to
// inspect git repositories. It wraps the git command-line tool and returns
// structured Go types instead of raw command output, making it particularly
// suitable for integration with AI agents, code analysis tools, and development
// utilities.
//
// # Features
//
//   - Repository inspection: status, branches, tags, remotes
//   - Commit history: log, show, blame
//   - Diff operations: staged, unstaged, between refs
//   - File operations: tracked files, untracked files, file contents at refs
//   - Reference operations: resolve refs, check ancestry, merge base
//   - Configuration access: read git config values
//
// # Basic Usage
//
// Open a repository and query its status:
//
//	repo, err := git.Open(".")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	status, err := repo.Status(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Branch: %s\n", status.Branch)
//	fmt.Printf("Clean: %v\n", status.IsClean)
//
// Get recent commit history:
//
//	commits, err := repo.Log(ctx, git.LogOptions{Limit: 10})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, commit := range commits {
//	    fmt.Printf("%s - %s\n", commit.ShortHash, commit.Subject)
//	}
//
// View staged changes:
//
//	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, file := range diff.Files {
//	    fmt.Printf("%s: +%d -%d\n", file.Path, file.Additions, file.Deletions)
//	}
//
// # Requirements
//
// This package requires the git command-line tool to be installed and available
// in the system PATH. All operations are read-only; the package does not modify
// repository state.
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

// Repository represents a git repository and provides methods for
// querying its state and history.
//
// All operations are context-aware and support cancellation. Methods
// return structured data types rather than raw command output.
type Repository struct {
	// Path is the root directory of the repository (the working tree).
	Path string
	// GitDir is the .git directory path (or the repository directory for bare repos).
	GitDir string
}

// Open opens a git repository at the given path.
//
// The path can be any directory within the repository; Open will automatically
// find the repository root. Returns ErrNotRepository if the path is not within
// a git repository.
//
// Example:
//
//	repo, err := git.Open(".")
//	if err == git.ErrNotRepository {
//	    fmt.Println("Not a git repository")
//	    return
//	}
//	if err != nil {
//	    log.Fatal(err)
//	}
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

// Commit represents a git commit with its metadata and message.
//
// The Body field is only populated when using LogOptions.IncludeBody.
// ParentHashes contains the full hashes of parent commits (empty for
// initial commit, multiple for merge commits).
type Commit struct {
	Hash         string    `json:"hash"`          // Full 40-character SHA-1 hash
	ShortHash    string    `json:"short_hash"`    // Abbreviated hash (typically 7 characters)
	Author       Person    `json:"author"`        // Person who authored the commit
	Committer    Person    `json:"committer"`     // Person who committed (may differ from author)
	Subject      string    `json:"subject"`       // First line of commit message
	Body         string    `json:"body,omitempty"` // Remaining lines of commit message
	ParentHashes []string  `json:"parent_hashes,omitempty"` // Parent commit hashes
	Timestamp    time.Time `json:"timestamp"`     // Commit timestamp
}

// Person represents an author or committer in git.
type Person struct {
	Name  string `json:"name"`  // Display name (e.g., "John Doe")
	Email string `json:"email"` // Email address (e.g., "john@example.com")
}

// Status represents the working directory status, including staged and
// unstaged changes, branch information, and tracking status.
//
// IsClean is true when there are no staged changes, unstaged changes,
// untracked files, or conflicts. HasUntracked is true when there are
// untracked files (even if otherwise clean).
type Status struct {
	Branch       string       `json:"branch"`              // Current branch name (empty if detached HEAD)
	Upstream     string       `json:"upstream,omitempty"`  // Upstream tracking branch (e.g., "origin/main")
	Ahead        int          `json:"ahead"`               // Commits ahead of upstream
	Behind       int          `json:"behind"`              // Commits behind upstream
	Staged       []FileStatus `json:"staged,omitempty"`    // Files staged for commit
	Unstaged     []FileStatus `json:"unstaged,omitempty"`  // Files with unstaged changes
	Untracked    []string     `json:"untracked,omitempty"` // Untracked file paths
	Conflicts    []string     `json:"conflicts,omitempty"` // Files with merge conflicts
	IsClean      bool         `json:"is_clean"`            // True if no changes in working tree
	HasUntracked bool         `json:"has_untracked"`       // True if there are untracked files
}

// FileStatus represents the status of a single file in the working tree.
type FileStatus struct {
	Path    string `json:"path"`               // Current file path
	OldPath string `json:"old_path,omitempty"` // Original path (for renames)
	Status  string `json:"status"`             // Status: "modified", "added", "deleted", "renamed", "copied"
}

// Branch represents a git branch (local or remote).
type Branch struct {
	Name      string `json:"name"`               // Branch name (e.g., "main" or "origin/main")
	Hash      string `json:"hash"`               // Short hash of the branch tip
	Upstream  string `json:"upstream,omitempty"` // Upstream branch (for local branches)
	IsCurrent bool   `json:"is_current"`         // True if this is the currently checked out branch
	IsRemote  bool   `json:"is_remote"`          // True if this is a remote-tracking branch
}

// Tag represents a git tag (lightweight or annotated).
//
// For lightweight tags, only Name, Hash, and Commit are populated.
// For annotated tags, Tagger, Message, and Date contain additional metadata.
type Tag struct {
	Name    string     `json:"name"`              // Tag name (e.g., "v1.0.0")
	Hash    string     `json:"hash"`              // Hash of the tag object (or commit for lightweight tags)
	Commit  string     `json:"commit,omitempty"`  // Hash of the tagged commit
	Tagger  *Person    `json:"tagger,omitempty"`  // Person who created the tag (annotated tags only)
	Message string     `json:"message,omitempty"` // Tag message (annotated tags only)
	Date    *time.Time `json:"date,omitempty"`    // Tag creation date (annotated tags only)
}

// Remote represents a git remote repository.
//
// FetchURL and PushURL may be the same or different depending on
// repository configuration.
type Remote struct {
	Name     string `json:"name"`      // Remote name (e.g., "origin", "upstream")
	FetchURL string `json:"fetch_url"` // URL used for fetching
	PushURL  string `json:"push_url"`  // URL used for pushing
}

// DiffFile represents changes to a single file in a diff.
//
// For binary files, Additions and Deletions are 0 and Binary is true.
// The Patch field is only populated when DiffOptions.IncludePatch is true.
type DiffFile struct {
	Path      string `json:"path"`               // Current file path
	OldPath   string `json:"old_path,omitempty"` // Original path (for renames)
	Status    string `json:"status"`             // Status: "added", "modified", "deleted", "renamed"
	Additions int    `json:"additions"`          // Number of lines added
	Deletions int    `json:"deletions"`          // Number of lines deleted
	Binary    bool   `json:"binary"`             // True for binary files
	Patch     string `json:"patch,omitempty"`    // Unified diff content (when IncludePatch is true)
}

// Diff represents a git diff result containing changes across multiple files.
//
// TotalAdded and TotalRemoved sum the line changes across all files.
// Stats contains a human-readable summary line (e.g., "3 files changed, 15 insertions(+), 2 deletions(-)").
type Diff struct {
	Files        []DiffFile `json:"files"`               // Changed files
	TotalAdded   int        `json:"total_added"`         // Total lines added across all files
	TotalRemoved int        `json:"total_removed"`       // Total lines deleted across all files
	Stats        string     `json:"stats,omitempty"`     // Human-readable summary
}

// BlameLine represents a single line of git blame output, showing which
// commit last modified the line.
type BlameLine struct {
	Hash       string    `json:"hash"`        // Commit hash that last modified this line
	Author     string    `json:"author"`      // Author name from that commit
	Email      string    `json:"email"`       // Author email from that commit
	Timestamp  time.Time `json:"timestamp"`   // Timestamp of the commit
	LineNumber int       `json:"line_number"` // Line number in the file (1-indexed)
	Content    string    `json:"content"`     // The actual line content
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

// Head returns the current HEAD commit hash (full 40-character SHA-1).
// Returns ErrNoCommits if the repository has no commits yet.
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
//
// Returns an empty string if HEAD is detached (not pointing to a branch).
// This can happen after checking out a specific commit or tag.
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
//
// A clean repository has no staged changes, unstaged modifications,
// untracked files, or merge conflicts. This is equivalent to
// "git status" reporting "nothing to commit, working tree clean".
func (r *Repository) IsClean(ctx context.Context) (bool, error) {
	out, err := r.run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) == 0, nil
}

// Status returns the working directory status including branch information,
// staged and unstaged changes, untracked files, and tracking status.
//
// This provides a comprehensive view of the repository state, similar to
// running "git status" but with structured data.
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
