package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/git"
)

func TestOpen(t *testing.T) {
	// Test opening the current repo
	repo, err := git.Open(".")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if repo.Path == "" {
		t.Error("expected non-empty path")
	}
}

func TestOpenNotRepo(t *testing.T) {
	_, err := git.Open("/tmp")
	if err != git.ErrNotRepository {
		t.Errorf("expected ErrNotRepository, got %v", err)
	}
}

func setupTestRepo(t *testing.T) (*git.Repository, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	// Initialize repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatal(err)
	}

	// Configure user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	cmd.Run()

	repo, err := git.Open(dir)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}

	return repo, cleanup
}

func addFile(t *testing.T, repo *git.Repository, name, content string) {
	t.Helper()
	path := filepath.Join(repo.Path, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", name)
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}

func commit(t *testing.T, repo *git.Repository, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestStatus(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Empty repo should have no status items
	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Add an untracked file
	if err := os.WriteFile(filepath.Join(repo.Path, "untracked.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	status, err = repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(status.Untracked) != 1 {
		t.Errorf("expected 1 untracked file, got %d", len(status.Untracked))
	}
	if status.IsClean {
		t.Error("expected not clean")
	}
	if !status.HasUntracked {
		t.Error("expected HasUntracked")
	}

	// Stage the file
	addFile(t, repo, "untracked.txt", "hello")

	status, err = repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(status.Staged) != 1 {
		t.Errorf("expected 1 staged file, got %d", len(status.Staged))
	}
}

func TestLog(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create some commits
	addFile(t, repo, "file1.txt", "content1")
	commit(t, repo, "First commit")

	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Second commit")

	addFile(t, repo, "file3.txt", "content3")
	commit(t, repo, "Third commit")

	// Get all commits
	commits, err := repo.Log(ctx, git.LogOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 3 {
		t.Errorf("expected 3 commits, got %d", len(commits))
	}

	// Newest first
	if commits[0].Subject != "Third commit" {
		t.Errorf("expected 'Third commit', got %q", commits[0].Subject)
	}

	// Test limit
	commits, err = repo.Log(ctx, git.LogOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(commits))
	}

	// Test path filter
	commits, err = repo.Log(ctx, git.LogOptions{Path: "file1.txt"})
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 1 {
		t.Errorf("expected 1 commit for file1.txt, got %d", len(commits))
	}
}

func TestBranches(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Need at least one commit
	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Create a branch
	cmd := exec.Command("git", "branch", "feature")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	branches, err := repo.LocalBranches(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(branches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(branches))
	}

	// Check current branch
	current, err := repo.CurrentBranch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Should be main, master, or whatever default
	if current == "" {
		t.Error("expected non-empty current branch")
	}
}

func TestTags(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Need at least one commit
	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Create tags
	cmd := exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("git", "tag", "-a", "v2.0.0", "-m", "Version 2")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	tags, err := repo.Tags(ctx, git.TagOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	// Test latest tag
	latest, err := repo.LatestTag(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if latest == nil {
		t.Error("expected latest tag")
	}
}

func TestDiff(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create initial commit
	addFile(t, repo, "file.txt", "line1\nline2\nline3\n")
	commit(t, repo, "Initial commit")

	// Modify the file
	if err := os.WriteFile(filepath.Join(repo.Path, "file.txt"), []byte("line1\nmodified\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get diff
	diff, err := repo.Diff(ctx, git.DiffOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(diff.Files) != 1 {
		t.Errorf("expected 1 file in diff, got %d", len(diff.Files))
	}

	if diff.Files[0].Status != "modified" {
		t.Errorf("expected modified status, got %q", diff.Files[0].Status)
	}
}

func TestConfig(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Get user name (we set it in setup)
	name, err := repo.UserName(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if name != "Test User" {
		t.Errorf("expected 'Test User', got %q", name)
	}

	email, err := repo.UserEmail(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if email != "test@example.com" {
		t.Errorf("expected 'test@example.com', got %q", email)
	}
}

func TestBlame(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create file with multiple lines
	addFile(t, repo, "file.txt", "line1\nline2\nline3\n")
	commit(t, repo, "Initial commit")

	blame, err := repo.Blame(ctx, "file.txt", git.BlameOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(blame) != 3 {
		t.Errorf("expected 3 blame lines, got %d", len(blame))
	}

	if blame[0].Author != "Test User" {
		t.Errorf("expected 'Test User', got %q", blame[0].Author)
	}

	if blame[0].Content != "line1" {
		t.Errorf("expected 'line1', got %q", blame[0].Content)
	}
}

func TestTrackedFiles(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create and commit files
	addFile(t, repo, "file1.txt", "content1")
	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Initial commit")

	files, err := repo.TrackedFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 tracked files, got %d", len(files))
	}
}

func TestHeadAndRefs(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Need a commit
	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	head, err := repo.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(head) != 40 {
		t.Errorf("expected 40-char hash, got %d chars", len(head))
	}

	// Short hash
	short, err := repo.ShortHash(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if len(short) < 7 {
		t.Errorf("expected at least 7-char short hash, got %d chars", len(short))
	}
}

func TestIsClean(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Empty repo is clean
	clean, err := repo.IsClean(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !clean {
		t.Error("expected clean repo")
	}

	// Add untracked file
	if err := os.WriteFile(filepath.Join(repo.Path, "new.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	clean, err = repo.IsClean(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if clean {
		t.Error("expected not clean")
	}
}

func TestLogWithTime(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create commits
	addFile(t, repo, "file1.txt", "content1")
	commit(t, repo, "First commit")

	time.Sleep(100 * time.Millisecond)

	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Second commit")

	// Get commits and verify timestamps are set
	commits, err := repo.Log(ctx, git.LogOptions{})
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range commits {
		if c.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}
	}
}
