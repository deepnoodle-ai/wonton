package git_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/git"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Caller-supplied refs, names, and patterns that start with "-" must never
// reach git as options (BUG-009).
func TestOptionInjectionRejected(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "README.md", "hello\n")
	commit(t, repo, "initial")

	outDir := t.TempDir()
	target := filepath.Join(outDir, "pwned")
	inject := "--output=" + target

	calls := map[string]func() error{
		"Show": func() error { _, err := repo.Show(ctx, inject); return err },
		"Log":  func() error { _, err := repo.Log(ctx, git.LogOptions{Ref: inject}); return err },
		"CommitsBetween": func() error {
			_, err := repo.CommitsBetween(ctx, inject, "HEAD")
			return err
		},
		"Diff.Ref":  func() error { _, err := repo.Diff(ctx, git.DiffOptions{Ref: inject}); return err },
		"Diff.From": func() error { _, err := repo.Diff(ctx, git.DiffOptions{From: inject, To: "HEAD"}); return err },
		"DiffFile": func() error {
			_, err := repo.DiffFile(ctx, "README.md", git.DiffOptions{Ref: inject})
			return err
		},
		"ShowFile": func() error { _, err := repo.ShowFile(ctx, inject, "README.md"); return err },
		"Blame": func() error {
			_, err := repo.Blame(ctx, "README.md", git.BlameOptions{Ref: "--contents=/etc/hosts"})
			return err
		},
		"Tags.Pattern":  func() error { _, err := repo.Tags(ctx, git.TagOptions{Pattern: inject}); return err },
		"Tags.Contains": func() error { _, err := repo.Tags(ctx, git.TagOptions{Contains: inject}); return err },
		"Describe":      func() error { _, err := repo.Describe(ctx, inject); return err },
		"DescribeLong":  func() error { _, err := repo.DescribeLong(ctx, inject); return err },
		"Branches.Contains": func() error {
			_, err := repo.Branches(ctx, git.BranchOptions{Contains: inject})
			return err
		},
		"BranchExists": func() error { _, err := repo.BranchExists(ctx, inject); return err },
		"RemoteURL":    func() error { _, err := repo.RemoteURL(ctx, inject); return err },
		"Config":       func() error { _, err := repo.Config(ctx, inject); return err },
		"ConfigWithScope": func() error {
			_, err := repo.ConfigWithScope(ctx, "user.name", git.ConfigScope("file="+target))
			return err
		},
		"ResolveRef": func() error { _, err := repo.ResolveRef(ctx, inject); return err },
		"ShortHash":  func() error { _, err := repo.ShortHash(ctx, inject); return err },
		"IsAncestor": func() error { _, err := repo.IsAncestor(ctx, inject, "HEAD"); return err },
		"MergeBase":  func() error { _, err := repo.MergeBase(ctx, "HEAD", inject); return err },
		"AbbrevRef":  func() error { _, err := repo.AbbrevRef(ctx, inject); return err },
		"RefExists":  func() error { _, err := repo.RefExists(ctx, inject); return err },
		"CommitCount": func() error {
			_, err := repo.CommitCount(ctx, inject)
			return err
		},
		"CommitCountBetween": func() error {
			_, err := repo.CommitCountBetween(ctx, inject, "HEAD")
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call()
			if !errors.Is(err, git.ErrInvalidArgument) {
				t.Errorf("expected ErrInvalidArgument, got %v", err)
			}
			if _, statErr := os.Stat(target); statErr == nil {
				os.Remove(target)
				t.Errorf("git wrote %s: option was injected", target)
			}
		})
	}
}

// A rename in porcelain v2 output carries a score field before the path, and
// a file can be both staged and unstaged with different states (BUG-033).
func TestStatusRenameAndDualState(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "old.txt", "content\n")
	commit(t, repo, "initial")
	runGit(t, repo.Path, "mv", "old.txt", "new.txt")

	// Staged add + unstaged modify.
	addFile(t, repo, "both.txt", "one\n")
	if err := os.WriteFile(filepath.Join(repo.Path, "both.txt"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}

	staged := map[string]git.FileStatus{}
	for _, fs := range status.Staged {
		staged[fs.Path] = fs
	}
	if fs, ok := staged["new.txt"]; !ok || fs.OldPath != "old.txt" || fs.Status != "renamed" {
		t.Errorf("rename entry = %+v (all staged: %+v)", fs, status.Staged)
	}
	if fs := staged["both.txt"]; fs.Status != "added" {
		t.Errorf("staged both.txt status = %q, want added", fs.Status)
	}
	var unstaged string
	for _, fs := range status.Unstaged {
		if fs.Path == "both.txt" {
			unstaged = fs.Status
		}
	}
	if unstaged != "modified" {
		t.Errorf("unstaged both.txt status = %q, want modified", unstaged)
	}
}

func TestStatusPathWithSpaces(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "my file.txt", "content\n")
	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Staged) != 1 || status.Staged[0].Path != "my file.txt" {
		t.Errorf("staged = %+v", status.Staged)
	}
}

// Local branches containing "/" are not remote, and BranchExists only
// matches branches (BUG-068).
func TestLocalBranchWithSlashAndBranchExists(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "initial")
	runGit(t, repo.Path, "branch", "feature/foo")
	runGit(t, repo.Path, "tag", "v1.0.0")

	branches, err := repo.LocalBranches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, b := range branches {
		if b.Name == "feature/foo" {
			found = true
			if b.IsRemote {
				t.Error("local branch feature/foo marked IsRemote")
			}
		}
	}
	if !found {
		t.Errorf("feature/foo not in %+v", branches)
	}

	for _, name := range []string{"HEAD", "v1.0.0"} {
		exists, err := repo.BranchExists(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Errorf("BranchExists(%q) = true, want false", name)
		}
	}
	exists, err := repo.BranchExists(ctx, "feature/foo")
	if err != nil || !exists {
		t.Errorf("BranchExists(feature/foo) = %v, %v", exists, err)
	}
}

func TestRemoteBranchExists(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "initial")
	runGit(t, repo.Path, "update-ref", "refs/remotes/origin/main", "HEAD")

	exists, err := repo.BranchExists(ctx, "origin/main")
	if err != nil || !exists {
		t.Errorf("BranchExists(origin/main) = %v, %v", exists, err)
	}
	branches, err := repo.RemoteBranches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 1 || !branches[0].IsRemote {
		t.Errorf("remote branches = %+v", branches)
	}
}

func TestBranchesDetachedHead(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "initial")
	runGit(t, repo.Path, "checkout", "-q", "--detach")

	branches, err := repo.LocalBranches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range branches {
		if strings.HasPrefix(b.Name, "(") {
			t.Errorf("detached HEAD reported as branch %q", b.Name)
		}
	}
}

func TestStatusDetachedHead(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "initial")
	runGit(t, repo.Path, "checkout", "-q", "--detach")

	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Branch != "" {
		t.Errorf("Branch = %q, want empty for detached HEAD", status.Branch)
	}
}

// Diff must match status and patch for paths with spaces, handle renames out
// of subdirectories, and not drop long lines (BUG-069).
func TestDiffPathsWithSpaces(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "my file.txt", "one\n")
	commit(t, repo, "initial")
	if err := os.WriteFile(filepath.Join(repo.Path, "my file.txt"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff, err := repo.Diff(ctx, git.DiffOptions{IncludePatch: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Files) != 1 {
		t.Fatalf("files = %+v", diff.Files)
	}
	f := diff.Files[0]
	if f.Path != "my file.txt" || f.Status != "modified" || !strings.Contains(f.Patch, "+two") {
		t.Errorf("file = %+v", f)
	}
}

func TestDiffRenameOutOfSubdirectory(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	if err := os.MkdirAll(filepath.Join(repo.Path, "dir", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	addFile(t, repo, "dir/sub/file.txt", "a\nb\nc\nd\n")
	commit(t, repo, "initial")
	runGit(t, repo.Path, "mv", "dir/sub/file.txt", "dir/file.txt")

	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Files) != 1 {
		t.Fatalf("files = %+v", diff.Files)
	}
	f := diff.Files[0]
	if f.Path != "dir/file.txt" || f.OldPath != "dir/sub/file.txt" || f.Status != "renamed" {
		t.Errorf("file = %+v", f)
	}
}

// With diff.renames=copies, a copy is reported as copied, and the modified
// copy source keeps its own status.
func TestDiffCopiedFile(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	runGit(t, repo.Path, "config", "diff.renames", "copies")
	addFile(t, repo, "z.txt", "a\nb\nc\nd\n")
	commit(t, repo, "initial")
	addFile(t, repo, "b.txt", "a\nb\nc\nd\n")
	addFile(t, repo, "z.txt", "a\nb\nc\nd\ne\n")

	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]git.DiffFile{}
	for _, f := range diff.Files {
		files[f.Path] = f
	}
	if f := files["b.txt"]; f.Status != "copied" || f.OldPath != "z.txt" {
		t.Errorf("b.txt = %+v, want copied from z.txt (all: %+v)", f, diff.Files)
	}
	if f := files["z.txt"]; f.Status != "modified" {
		t.Errorf("z.txt status = %q, want modified (all: %+v)", f.Status, diff.Files)
	}
}

// A rename or copy inside a directory whose name has braces keeps its paths
// and status. git prints such a numstat path as "routes/{id}/{z.txt => b.txt}".
func TestDiffRenameAndCopyInBraceDirectory(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	runGit(t, repo.Path, "config", "diff.renames", "copies")
	if err := os.MkdirAll(filepath.Join(repo.Path, "routes", "{id}"), 0o755); err != nil {
		t.Fatal(err)
	}
	addFile(t, repo, "routes/{id}/z.txt", "a\nb\nc\nd\n")
	addFile(t, repo, "routes/{id}/old.txt", "1\n2\n3\n4\n")
	commit(t, repo, "initial")
	addFile(t, repo, "routes/{id}/b.txt", "a\nb\nc\nd\n")
	addFile(t, repo, "routes/{id}/z.txt", "a\nb\nc\nd\ne\n")
	runGit(t, repo.Path, "mv", "routes/{id}/old.txt", "routes/{id}/new.txt")

	diff, err := repo.Diff(ctx, git.DiffOptions{Staged: true})
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]git.DiffFile{}
	for _, f := range diff.Files {
		files[f.Path] = f
	}
	if f := files["routes/{id}/b.txt"]; f.Status != "copied" || f.OldPath != "routes/{id}/z.txt" {
		t.Errorf("b.txt = %+v, want copied from routes/{id}/z.txt (all: %+v)", f, diff.Files)
	}
	if f := files["routes/{id}/new.txt"]; f.Status != "renamed" || f.OldPath != "routes/{id}/old.txt" {
		t.Errorf("new.txt = %+v, want renamed from routes/{id}/old.txt (all: %+v)", f, diff.Files)
	}
	if f := files["routes/{id}/z.txt"]; f.Status != "modified" {
		t.Errorf("z.txt status = %q, want modified (all: %+v)", f.Status, diff.Files)
	}
}

// Diff returns a non-ASCII path as the path itself, not git's quoted form.
func TestDiffNonASCIIPath(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "café.txt", "one\n")
	commit(t, repo, "initial")
	if err := os.WriteFile(filepath.Join(repo.Path, "café.txt"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff, err := repo.Diff(ctx, git.DiffOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Files) != 1 || diff.Files[0].Path != "café.txt" || diff.Files[0].Status != "modified" {
		t.Errorf("files = %+v, want café.txt modified", diff.Files)
	}
}

// git C-quotes a "diff --git" header path that has a non-ASCII byte, '"', '\',
// or a control character. The patch must still match its file.
func TestDiffPatchQuotedPath(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	names := []string{"café.txt", "plain.txt"}
	if runtime.GOOS != "windows" {
		names = append(names, "tab\there.txt", `quo"te.txt`, `back\slash.txt`)
	}
	for _, name := range names {
		addFile(t, repo, name, "one\n")
	}
	commit(t, repo, "initial")
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(repo.Path, name), []byte("two\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	diff, err := repo.Diff(ctx, git.DiffOptions{IncludePatch: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Files) != len(names) {
		t.Fatalf("got %d files, want %d: %+v", len(diff.Files), len(names), diff.Files)
	}
	for _, f := range diff.Files {
		if !strings.Contains(f.Patch, "+two") {
			t.Errorf("%q: patch = %q, want the +two hunk", f.Path, f.Patch)
		}
	}

	f, err := repo.DiffFile(ctx, "café.txt", git.DiffOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if f == nil || !strings.Contains(f.Patch, "+two") {
		t.Errorf("DiffFile(café.txt) = %+v, want a patch with +two", f)
	}
}

func TestDiffLongLinePatch(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "a.txt", "a\n")
	addFile(t, repo, "b.txt", "b\n")
	commit(t, repo, "initial")
	long := strings.Repeat("x", 100*1024)
	os.WriteFile(filepath.Join(repo.Path, "a.txt"), []byte(long+"\n"), 0o644)
	os.WriteFile(filepath.Join(repo.Path, "b.txt"), []byte("changed\n"), 0o644)

	diff, err := repo.Diff(ctx, git.DiffOptions{IncludePatch: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range diff.Files {
		if f.Patch == "" {
			t.Errorf("%s: empty patch", f.Path)
		}
		if f.Path == "a.txt" && !strings.Contains(f.Patch, long) {
			t.Errorf("a.txt patch truncated")
		}
	}

	blame, err := repo.Blame(ctx, "a.txt", git.BlameOptions{})
	if err == nil && len(blame) != 1 {
		t.Errorf("blame lines = %d", len(blame))
	}
	runGit(t, repo.Path, "commit", "-qam", "long")
	blame, err = repo.Blame(ctx, "a.txt", git.BlameOptions{})
	if err != nil {
		t.Fatalf("blame long line: %v", err)
	}
	if len(blame) != 1 || blame[0].Content != long {
		t.Errorf("blame = %d lines", len(blame))
	}
}

// ErrNoCommits means an empty repository, not a bad ref (BUG-070).
func TestErrNoCommitsSemantics(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	if _, err := repo.Log(ctx, git.LogOptions{}); !errors.Is(err, git.ErrNoCommits) {
		t.Errorf("Log on empty repo: got %v, want ErrNoCommits", err)
	}
	if _, err := repo.Show(ctx, "HEAD"); !errors.Is(err, git.ErrNoCommits) {
		t.Errorf("Show HEAD on empty repo: got %v, want ErrNoCommits", err)
	}

	addFile(t, repo, "file.txt", "content")
	commit(t, repo, "initial")

	_, err := repo.Show(ctx, "nonexistent")
	if err == nil || errors.Is(err, git.ErrNoCommits) {
		t.Errorf("Show(nonexistent) on non-empty repo: got %v, want non-ErrNoCommits error", err)
	}
}

func TestOpenAbsoluteGitDirAndBare(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	if !filepath.IsAbs(repo.GitDir) {
		t.Errorf("GitDir = %q, want absolute", repo.GitDir)
	}

	bare := t.TempDir()
	runGit(t, bare, "init", "-q", "--bare")
	r, err := git.Open(bare)
	if err != nil {
		t.Fatalf("Open bare: %v", err)
	}
	if !filepath.IsAbs(r.GitDir) {
		t.Errorf("bare GitDir = %q, want absolute", r.GitDir)
	}
}

func TestBlameEndLineOnly(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	addFile(t, repo, "f.txt", "1\n2\n3\n4\n")
	commit(t, repo, "initial")

	lines, err := repo.Blame(ctx, "f.txt", git.BlameOptions{EndLine: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Errorf("got %d lines, want 2", len(lines))
	}
}
