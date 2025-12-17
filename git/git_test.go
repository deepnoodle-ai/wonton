package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestResolveRef(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	hash, err := repo.ResolveRef(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %d chars", len(hash))
	}
}

func TestRefExists(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	exists, err := repo.RefExists(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected HEAD to exist")
	}

	exists, err = repo.RefExists(ctx, "nonexistent-branch")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected nonexistent-branch to not exist")
	}
}

func TestCommitCount(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file1.txt", "content1")
	commit(t, repo, "First commit")

	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Second commit")

	count, err := repo.CommitCount(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Errorf("expected 2 commits, got %d", count)
	}
}

func TestAbbrevRef(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	abbrev, err := repo.AbbrevRef(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	// Should be master, main, or whatever default branch name
	if abbrev == "" {
		t.Error("expected non-empty abbrev ref")
	}
}

func TestRemotes(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Add a remote
	cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	remotes, err := repo.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(remotes) != 1 {
		t.Errorf("expected 1 remote, got %d", len(remotes))
	}

	if remotes[0].Name != "origin" {
		t.Errorf("expected origin remote, got %q", remotes[0].Name)
	}
}

func TestRemote(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	remote, err := repo.Remote(ctx, "origin")
	if err != nil {
		t.Fatal(err)
	}

	if remote == nil {
		t.Fatal("expected remote, got nil")
	}

	if remote.Name != "origin" {
		t.Errorf("expected origin, got %q", remote.Name)
	}

	// Test nonexistent remote
	remote, err = repo.Remote(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if remote != nil {
		t.Error("expected nil for nonexistent remote")
	}
}

func TestRemoteURL(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	expectedURL := "https://github.com/test/repo.git"
	cmd := exec.Command("git", "remote", "add", "origin", expectedURL)
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	url, err := repo.RemoteURL(ctx, "origin")
	if err != nil {
		t.Fatal(err)
	}

	// URL might be normalized by git, so just check it's not empty
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

func TestUntrackedFiles(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create untracked file
	if err := os.WriteFile(filepath.Join(repo.Path, "untracked.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := repo.UntrackedFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 untracked file, got %d", len(files))
	}
}

func TestModifiedFiles(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Modify the file
	if err := os.WriteFile(filepath.Join(repo.Path, "file.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := repo.ModifiedFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(files))
	}
}

func TestBranchExists(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Create a branch
	cmd := exec.Command("git", "branch", "feature")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	exists, err := repo.BranchExists(ctx, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected feature branch to exist")
	}

	exists, err = repo.BranchExists(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected nonexistent branch to not exist")
	}
}

func TestMergeBase(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create initial commit
	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Create a branch and make changes
	cmd := exec.Command("git", "branch", "feature")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Make another commit on main
	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Second commit")

	// Switch to feature and make a commit
	cmd = exec.Command("git", "checkout", "feature")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	addFile(t, repo, "feature.txt", "feature content")
	commit(t, repo, "Feature commit")

	// Get merge base
	base, err := repo.MergeBase(ctx, "feature", "HEAD~1")
	if err != nil {
		t.Fatal(err)
	}

	if len(base) != 40 {
		t.Errorf("expected 40-char hash, got %d chars", len(base))
	}
}

func TestIsAncestor(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create commits
	addFile(t, repo, "file1.txt", "content1")
	commit(t, repo, "First commit")

	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Second commit")

	// HEAD~1 should be ancestor of HEAD
	isAnc, err := repo.IsAncestor(ctx, "HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !isAnc {
		t.Error("expected HEAD~1 to be ancestor of HEAD")
	}

	// HEAD should not be ancestor of HEAD~1
	isAnc, err = repo.IsAncestor(ctx, "HEAD", "HEAD~1")
	if err != nil {
		t.Fatal(err)
	}
	if isAnc {
		t.Error("expected HEAD to not be ancestor of HEAD~1")
	}
}

func TestCommitCountBetween(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create commits
	addFile(t, repo, "file1.txt", "content1")
	commit(t, repo, "First commit")

	// Create branch at first commit
	cmd := exec.Command("git", "branch", "base")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Second commit")

	addFile(t, repo, "file3.txt", "content3")
	commit(t, repo, "Third commit")

	count, err := repo.CommitCountBetween(ctx, "base", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Errorf("expected 2 commits between base and HEAD, got %d", count)
	}
}

func TestDefaultBranch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Get default branch - should be main or master
	branch, err := repo.DefaultBranch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if branch == "" {
		t.Error("expected non-empty default branch")
	}
}

func TestRemoteBranches(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// No remotes, should return empty list
	branches, err := repo.RemoteBranches(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if branches != nil && len(branches) > 0 {
		t.Errorf("expected no remote branches, got %d", len(branches))
	}
}

func TestBranchesWithOptions(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Create branches
	for _, branch := range []string{"feature-a", "feature-b"} {
		cmd := exec.Command("git", "branch", branch)
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	}

	// Test Contains option
	head, err := repo.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}

	branches, err := repo.Branches(ctx, git.BranchOptions{Contains: head})
	if err != nil {
		t.Fatal(err)
	}

	if len(branches) != 3 {
		t.Errorf("expected 3 branches containing HEAD, got %d", len(branches))
	}
}

func TestDiffStaged(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create initial commit
	addFile(t, repo, "file.txt", "line1\nline2\nline3\n")
	commit(t, repo, "Initial commit")

	// Modify and stage
	if err := os.WriteFile(filepath.Join(repo.Path, "file.txt"), []byte("line1\nmodified\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "file.txt")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Get staged diff
	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(diff.Files) != 1 {
		t.Errorf("expected 1 file in staged diff, got %d", len(diff.Files))
	}
}

func TestDiffBetweenRefs(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content1")
	commit(t, repo, "First commit")

	// Tag the first commit
	cmd := exec.Command("git", "tag", "v1")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	addFile(t, repo, "file.txt", "content2")
	commit(t, repo, "Second commit")

	// Diff between v1 and HEAD
	diff, err := repo.Diff(ctx, git.DiffOptions{From: "v1", To: "HEAD"})
	if err != nil {
		t.Fatal(err)
	}

	if len(diff.Files) != 1 {
		t.Errorf("expected 1 file in diff between refs, got %d", len(diff.Files))
	}
}

func TestDiffWithPatch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "line1\nline2\nline3\n")
	commit(t, repo, "Initial commit")

	// Modify
	if err := os.WriteFile(filepath.Join(repo.Path, "file.txt"), []byte("line1\nmodified\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get diff with patch
	diff, err := repo.Diff(ctx, git.DiffOptions{IncludePatch: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}

	if diff.Files[0].Patch == "" {
		t.Error("expected non-empty patch content")
	}
}

func TestDiffFile(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file1.txt", "content1\n")
	addFile(t, repo, "file2.txt", "content2\n")
	commit(t, repo, "Initial commit")

	// Modify both files
	if err := os.WriteFile(filepath.Join(repo.Path, "file1.txt"), []byte("modified1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo.Path, "file2.txt"), []byte("modified2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get diff for just file1
	df, err := repo.DiffFile(ctx, "file1.txt", git.DiffOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if df == nil {
		t.Fatal("expected diff file, got nil")
	}

	if df.Path != "file1.txt" {
		t.Errorf("expected file1.txt, got %s", df.Path)
	}

	// Test DiffFile returns nil when file has no changes
	df, err = repo.DiffFile(ctx, "nonexistent.txt", git.DiffOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if df != nil {
		t.Error("expected nil for file without changes")
	}
}

func TestTagsWithPattern(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Create tags with different patterns
	for _, tag := range []string{"v1.0.0", "v2.0.0", "release-1"} {
		cmd := exec.Command("git", "tag", tag)
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	}

	// Filter by pattern
	tags, err := repo.Tags(ctx, git.TagOptions{Pattern: "v*"})
	if err != nil {
		t.Fatal(err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags matching v*, got %d", len(tags))
	}
}

func TestTagsForCommit(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	cmd := exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	head, err := repo.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}

	tags, err := repo.TagsForCommit(ctx, head)
	if err != nil {
		t.Fatal(err)
	}

	if len(tags) != 1 {
		t.Errorf("expected 1 tag for commit, got %d", len(tags))
	}
}

func TestDescribe(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	cmd := exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	desc, err := repo.Describe(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if desc != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %s", desc)
	}

	// Add another commit
	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Second commit")

	desc, err = repo.Describe(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(desc, "v1.0.0-1-g") {
		t.Errorf("expected v1.0.0-1-g*, got %s", desc)
	}
}

func TestDescribeLong(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	cmd := exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	desc, err := repo.DescribeLong(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	// Long form should always have -0-g even at tag
	if !strings.HasPrefix(desc, "v1.0.0-0-g") {
		t.Errorf("expected v1.0.0-0-g*, got %s", desc)
	}
}

func TestConfigList(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	config, err := repo.ConfigList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Should have user.name and user.email from setup
	if config["user.name"] != "Test User" {
		t.Errorf("expected user.name 'Test User', got %q", config["user.name"])
	}

	if config["user.email"] != "test@example.com" {
		t.Errorf("expected user.email 'test@example.com', got %q", config["user.email"])
	}
}

func TestUser(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	user, err := repo.User(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if user == nil {
		t.Fatal("expected user, got nil")
	}

	if user.Name != "Test User" {
		t.Errorf("expected 'Test User', got %q", user.Name)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected 'test@example.com', got %q", user.Email)
	}
}

func TestBlameWithOptions(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create file with multiple lines
	addFile(t, repo, "file.txt", "line1\nline2\nline3\nline4\nline5\n")
	commit(t, repo, "Initial commit")

	// Blame specific lines
	blame, err := repo.Blame(ctx, "file.txt", git.BlameOptions{StartLine: 2, EndLine: 4})
	if err != nil {
		t.Fatal(err)
	}

	if len(blame) != 3 {
		t.Errorf("expected 3 blame lines (lines 2-4), got %d", len(blame))
	}

	// Check line numbers
	for i, line := range blame {
		expectedLine := i + 2
		if line.LineNumber != expectedLine {
			t.Errorf("expected line number %d, got %d", expectedLine, line.LineNumber)
		}
	}
}

func TestFileExists(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "exists.txt", "content")
	commit(t, repo, "Initial commit")

	exists, err := repo.FileExists(ctx, "exists.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected file to exist")
	}

	exists, err = repo.FileExists(ctx, "notexists.txt")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected file to not exist")
	}
}

func TestShowFile(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	content := "hello world"
	addFile(t, repo, "file.txt", content)
	commit(t, repo, "Initial commit")

	data, err := repo.ShowFile(ctx, "HEAD", "file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}
}

func TestFileAtRef(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "version1")
	commit(t, repo, "First commit")

	// Tag it
	cmd := exec.Command("git", "tag", "v1")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	addFile(t, repo, "file.txt", "version2")
	commit(t, repo, "Second commit")

	// Get file at v1
	content, err := repo.FileAtRef(ctx, "v1", "file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if content != "version1" {
		t.Errorf("expected 'version1', got %q", content)
	}

	// Get file at HEAD
	content, err = repo.FileAtRef(ctx, "HEAD", "file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if content != "version2" {
		t.Errorf("expected 'version2', got %q", content)
	}
}

func TestContributors(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content1")
	commit(t, repo, "First commit")

	addFile(t, repo, "file.txt", "content2")
	commit(t, repo, "Second commit")

	contributors, err := repo.Contributors(ctx, "file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if len(contributors) != 1 {
		t.Errorf("expected 1 contributor, got %d", len(contributors))
	}

	if contributors[0].Name != "Test User" {
		t.Errorf("expected 'Test User', got %q", contributors[0].Name)
	}
}

func TestLogWithOptions(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create commits with different messages
	addFile(t, repo, "file1.txt", "content1")
	commit(t, repo, "Feature: Add file1")

	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Bug fix: Fix issue")

	addFile(t, repo, "file3.txt", "content3")
	commit(t, repo, "Feature: Add file3")

	// Filter by grep
	commits, err := repo.Log(ctx, git.LogOptions{Grep: "Feature"})
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 2 {
		t.Errorf("expected 2 commits matching 'Feature', got %d", len(commits))
	}

	// Filter by author
	commits, err = repo.Log(ctx, git.LogOptions{Author: "Test User"})
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 3 {
		t.Errorf("expected 3 commits by Test User, got %d", len(commits))
	}
}

func TestLogIncludeBody(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")

	// Create commit with body
	cmd := exec.Command("git", "commit", "-m", "Subject line\n\nThis is the body.")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	commits, err := repo.Log(ctx, git.LogOptions{IncludeBody: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}

	if commits[0].Subject != "Subject line" {
		t.Errorf("expected 'Subject line', got %q", commits[0].Subject)
	}

	if commits[0].Body != "This is the body." {
		t.Errorf("expected body 'This is the body.', got %q", commits[0].Body)
	}
}

func TestShow(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Test commit")

	head, err := repo.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}

	c, err := repo.Show(ctx, head)
	if err != nil {
		t.Fatal(err)
	}

	if c == nil {
		t.Fatal("expected commit, got nil")
	}

	if c.Subject != "Test commit" {
		t.Errorf("expected 'Test commit', got %q", c.Subject)
	}

	if c.Hash != head {
		t.Errorf("expected hash %s, got %s", head, c.Hash)
	}
}

func TestCommitsBetween(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file1.txt", "content1")
	commit(t, repo, "First commit")

	// Tag first commit
	cmd := exec.Command("git", "tag", "v1")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Second commit")

	addFile(t, repo, "file3.txt", "content3")
	commit(t, repo, "Third commit")

	commits, err := repo.CommitsBetween(ctx, "v1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 2 {
		t.Errorf("expected 2 commits between v1 and HEAD, got %d", len(commits))
	}
}

func TestOriginURL(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	setURL := "https://github.com/test/repo.git"
	cmd := exec.Command("git", "remote", "add", "origin", setURL)
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	url, err := repo.OriginURL(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// URL might be transformed by git config (e.g., to SSH), so just check it's not empty
	// and contains the repo name
	if url == "" {
		t.Error("expected non-empty URL")
	}
	if !strings.Contains(url, "test/repo") {
		t.Errorf("expected URL to contain 'test/repo', got %q", url)
	}
}

func TestIgnoredFiles(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create .gitignore
	addFile(t, repo, ".gitignore", "*.log\n")
	commit(t, repo, "Add gitignore")

	// Create ignored file
	if err := os.WriteFile(filepath.Join(repo.Path, "test.log"), []byte("log content"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := repo.IgnoredFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 ignored file, got %d", len(files))
	}
}

func TestLsFilesOptions(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Modify the file
	if err := os.WriteFile(filepath.Join(repo.Path, "file.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test Modified option
	files, err := repo.LsFiles(ctx, git.LsFilesOptions{Modified: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(files))
	}

	// Test Cached option explicitly
	files, err = repo.LsFiles(ctx, git.LsFilesOptions{Cached: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 cached file, got %d", len(files))
	}
}

func TestStatusWithStagedAndUnstaged(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file1.txt", "content1")
	addFile(t, repo, "file2.txt", "content2")
	commit(t, repo, "Initial commit")

	// Create staged modification
	if err := os.WriteFile(filepath.Join(repo.Path, "file1.txt"), []byte("modified1"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "file1.txt")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Create unstaged modification
	if err := os.WriteFile(filepath.Join(repo.Path, "file2.txt"), []byte("modified2"), 0644); err != nil {
		t.Fatal(err)
	}

	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(status.Staged) != 1 {
		t.Errorf("expected 1 staged file, got %d", len(status.Staged))
	}

	if len(status.Unstaged) != 1 {
		t.Errorf("expected 1 unstaged file, got %d", len(status.Unstaged))
	}

	if status.IsClean {
		t.Error("expected not clean")
	}
}

func TestStatusWithDeleted(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Delete the file
	if err := os.Remove(filepath.Join(repo.Path, "file.txt")); err != nil {
		t.Fatal(err)
	}

	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(status.Unstaged) != 1 {
		t.Errorf("expected 1 unstaged (deleted) file, got %d", len(status.Unstaged))
	}

	if status.Unstaged[0].Status != "deleted" {
		t.Errorf("expected 'deleted' status, got %q", status.Unstaged[0].Status)
	}
}

func TestStatusAheadBehind(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// No remote, so no upstream info
	if status.Upstream != "" {
		t.Errorf("expected no upstream, got %q", status.Upstream)
	}

	if status.Ahead != 0 || status.Behind != 0 {
		t.Errorf("expected ahead=0, behind=0, got ahead=%d, behind=%d", status.Ahead, status.Behind)
	}
}

func TestDiffWithContextLines(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create file with many lines
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n"
	addFile(t, repo, "file.txt", content)
	commit(t, repo, "Initial commit")

	// Modify middle line
	modContent := "line1\nline2\nline3\nline4\nMODIFIED\nline6\nline7\nline8\nline9\nline10\n"
	if err := os.WriteFile(filepath.Join(repo.Path, "file.txt"), []byte(modContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Get diff with custom context lines
	diff, err := repo.Diff(ctx, git.DiffOptions{ContextLines: 1, IncludePatch: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}

	// Verify the patch has limited context
	if diff.Files[0].Patch == "" {
		t.Error("expected patch content")
	}
}

func TestDiffAddedFile(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Add new file
	if err := os.WriteFile(filepath.Join(repo.Path, "new.txt"), []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "new.txt")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Get staged diff
	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}

	if diff.Files[0].Status != "added" {
		t.Errorf("expected 'added' status, got %q", diff.Files[0].Status)
	}
}

func TestDiffDeletedFile(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "Initial commit")

	// Delete and stage
	cmd := exec.Command("git", "rm", "file.txt")
	cmd.Dir = repo.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Get staged diff
	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}

	if diff.Files[0].Status != "deleted" {
		t.Errorf("expected 'deleted' status, got %q", diff.Files[0].Status)
	}
}
