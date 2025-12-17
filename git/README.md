# git

The git package provides a wrapper around git commands for common read operations. It returns structured data instead of raw command output, making it ideal for integration with LLM tools and automated workflows.

## Usage Examples

### Opening a Repository

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/deepnoodle-ai/wonton/git"
)

func main() {
    // Open repository (works from any directory within the repo)
    repo, err := git.Open(".")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Repository path:", repo.Path)
    fmt.Println("Git directory:", repo.GitDir)
}
```

### Getting Repository Status

```go
ctx := context.Background()

status, err := repo.Status(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Branch: %s\n", status.Branch)
fmt.Printf("Upstream: %s\n", status.Upstream)
fmt.Printf("Ahead: %d, Behind: %d\n", status.Ahead, status.Behind)

// Check for changes
if status.IsClean {
    fmt.Println("Working directory is clean")
} else {
    for _, f := range status.Staged {
        fmt.Printf("Staged: %s (%s)\n", f.Path, f.Status)
    }
    for _, f := range status.Unstaged {
        fmt.Printf("Modified: %s (%s)\n", f.Path, f.Status)
    }
    for _, path := range status.Untracked {
        fmt.Printf("Untracked: %s\n", path)
    }
}
```

### Getting Commit History

```go
// Get last 10 commits
commits, err := repo.Log(ctx, git.LogOptions{
    Limit: 10,
})
if err != nil {
    log.Fatal(err)
}

for _, commit := range commits {
    fmt.Printf("%s - %s (%s)\n",
        commit.ShortHash,
        commit.Subject,
        commit.Author.Name,
    )
}
```

### Filtering Commit History

```go
// Get commits from specific author
commits, err := repo.Log(ctx, git.LogOptions{
    Author: "john@example.com",
    Limit:  20,
})

// Get commits in date range
since := time.Now().AddDate(0, -1, 0) // Last month
commits, err = repo.Log(ctx, git.LogOptions{
    Since: since,
})

// Search commit messages
commits, err = repo.Log(ctx, git.LogOptions{
    Grep: "fix bug",
})

// Get commits affecting specific path
commits, err = repo.Log(ctx, git.LogOptions{
    Path:  "src/main.go",
    Limit: 10,
})
```

### Getting Commit Details

```go
// Get specific commit with full message body
commit, err := repo.Show(ctx, "abc123")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Commit: %s\n", commit.Hash)
fmt.Printf("Author: %s <%s>\n", commit.Author.Name, commit.Author.Email)
fmt.Printf("Date: %s\n", commit.Timestamp)
fmt.Printf("\n%s\n\n%s\n", commit.Subject, commit.Body)
fmt.Printf("Parents: %v\n", commit.ParentHashes)
```

### Comparing Commits

```go
// Get commits between two refs
commits, err := repo.CommitsBetween(ctx, "main", "feature-branch")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Commits in feature-branch not in main: %d\n", len(commits))
```

### Getting Diffs

```go
// Get unstaged changes
diff, err := repo.Diff(ctx, git.DiffOptions{})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Files changed: %d\n", len(diff.Files))
fmt.Printf("Total: +%d -%d\n", diff.TotalAdded, diff.TotalRemoved)

for _, file := range diff.Files {
    fmt.Printf("%s: +%d -%d (%s)\n",
        file.Path,
        file.Additions,
        file.Deletions,
        file.Status,
    )
}
```

### Staged Changes

```go
// Get staged changes (ready to commit)
diff, err := repo.Diff(ctx, git.DiffOptions{
    Staged: true,
})
```

### Comparing Branches or Commits

```go
// Compare two branches
diff, err := repo.Diff(ctx, git.DiffOptions{
    From: "main",
    To:   "feature",
})

// Compare against specific commit
diff, err = repo.Diff(ctx, git.DiffOptions{
    Ref: "HEAD~3",
})
```

### Getting Diff with Patches

```go
// Include actual patch content
diff, err := repo.Diff(ctx, git.DiffOptions{
    Staged:       true,
    IncludePatch: true,
})

for _, file := range diff.Files {
    fmt.Printf("\n%s:\n%s\n", file.Path, file.Patch)
}
```

### Diff for Specific File

```go
// Get diff for single file
fileDiff, err := repo.DiffFile(ctx, "src/main.go", git.DiffOptions{})
if fileDiff != nil {
    fmt.Printf("Status: %s\n", fileDiff.Status)
    fmt.Printf("Changes: +%d -%d\n", fileDiff.Additions, fileDiff.Deletions)
    fmt.Printf("\n%s\n", fileDiff.Patch)
}
```

### Branch Operations

```go
// List all branches
branches, err := repo.Branches(ctx)
if err != nil {
    log.Fatal(err)
}

for _, branch := range branches {
    marker := " "
    if branch.IsCurrent {
        marker = "*"
    }
    fmt.Printf("%s %s -> %s\n", marker, branch.Name, branch.Hash[:7])
    if branch.Upstream != "" {
        fmt.Printf("  tracks: %s\n", branch.Upstream)
    }
}

// Get current branch
currentBranch, err := repo.CurrentBranch(ctx)
```

### Remote Operations

```go
// List remotes
remotes, err := repo.Remotes(ctx)
if err != nil {
    log.Fatal(err)
}

for _, remote := range remotes {
    fmt.Printf("%s:\n", remote.Name)
    fmt.Printf("  Fetch: %s\n", remote.FetchURL)
    fmt.Printf("  Push:  %s\n", remote.PushURL)
}
```

### Tag Operations

```go
// List all tags
tags, err := repo.Tags(ctx)
if err != nil {
    log.Fatal(err)
}

for _, tag := range tags {
    fmt.Printf("%s -> %s\n", tag.Name, tag.Hash[:7])
    if tag.Message != "" {
        // Annotated tag
        fmt.Printf("  Tagger: %s\n", tag.Tagger.Name)
        fmt.Printf("  Date: %s\n", tag.Date)
        fmt.Printf("  Message: %s\n", tag.Message)
    }
}
```

### Blame Information

```go
// Get blame for file
lines, err := repo.Blame(ctx, "src/main.go")
if err != nil {
    log.Fatal(err)
}

for _, line := range lines {
    fmt.Printf("%s (%s) %s\n",
        line.Hash[:7],
        line.Author,
        line.Content,
    )
}
```

### File Content at Revision

```go
// Get file content from specific commit
content, err := repo.ShowFile(ctx, "HEAD~2", "README.md")
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(content))
```

### List Files in Repository

```go
// List all tracked files
files, err := repo.ListFiles(ctx)
if err != nil {
    log.Fatal(err)
}

for _, file := range files {
    fmt.Println(file)
}
```

### Configuration

```go
// Get config value
email, err := repo.Config(ctx, "user.email")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User email: %s\n", email)

// Get all config
config, err := repo.ConfigAll(ctx)
for key, value := range config {
    fmt.Printf("%s = %s\n", key, value)
}
```

### Working with HEAD

```go
// Get current HEAD commit hash
head, err := repo.Head(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("HEAD: %s\n", head)

// Check if working directory is clean
clean, err := repo.IsClean(ctx)
if clean {
    fmt.Println("No uncommitted changes")
}
```

## API Reference

### Repository Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `Open(path)` | Opens git repository | `string` | `(*Repository, error)` |

### Repository Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `Status(ctx)` | Gets working directory status | `context.Context` | `(*Status, error)` |
| `Log(ctx, opts)` | Gets commit history | `context.Context`, `LogOptions` | `([]Commit, error)` |
| `Show(ctx, ref)` | Gets single commit details | `context.Context`, `string` | `(*Commit, error)` |
| `CommitsBetween(ctx, from, to)` | Gets commits between refs | `context.Context`, `string`, `string` | `([]Commit, error)` |
| `Diff(ctx, opts)` | Gets diff | `context.Context`, `DiffOptions` | `(*Diff, error)` |
| `DiffFile(ctx, path, opts)` | Gets diff for file | `context.Context`, `string`, `DiffOptions` | `(*DiffFile, error)` |
| `Branches(ctx)` | Lists branches | `context.Context` | `([]Branch, error)` |
| `CurrentBranch(ctx)` | Gets current branch name | `context.Context` | `(string, error)` |
| `Tags(ctx)` | Lists tags | `context.Context` | `([]Tag, error)` |
| `Remotes(ctx)` | Lists remotes | `context.Context` | `([]Remote, error)` |
| `Blame(ctx, path)` | Gets blame for file | `context.Context`, `string` | `([]BlameLine, error)` |
| `ShowFile(ctx, ref, path)` | Gets file content at ref | `context.Context`, `string`, `string` | `([]byte, error)` |
| `ListFiles(ctx)` | Lists tracked files | `context.Context` | `([]string, error)` |
| `Config(ctx, key)` | Gets config value | `context.Context`, `string` | `(string, error)` |
| `ConfigAll(ctx)` | Gets all config | `context.Context` | `(map[string]string, error)` |
| `Head(ctx)` | Gets HEAD commit hash | `context.Context` | `(string, error)` |
| `IsClean(ctx)` | Checks if working dir clean | `context.Context` | `(bool, error)` |

### Log Options

| Field | Type | Description |
|-------|------|-------------|
| `Limit` | `int` | Max commits to return |
| `Since` | `time.Time` | Commits after this date |
| `Until` | `time.Time` | Commits before this date |
| `Author` | `string` | Filter by author (substring) |
| `Grep` | `string` | Filter by message (substring) |
| `Path` | `string` | Filter by file path |
| `Ref` | `string` | Starting ref (default: HEAD) |
| `FirstParent` | `bool` | Follow only first parent of merges |
| `All` | `bool` | Include all refs |
| `IncludeBody` | `bool` | Include full commit message |

### Diff Options

| Field | Type | Description |
|-------|------|-------------|
| `Staged` | `bool` | Show staged changes |
| `Ref` | `string` | Compare against ref |
| `From` | `string` | Compare from ref |
| `To` | `string` | Compare to ref |
| `Path` | `string` | Filter to path |
| `ContextLines` | `int` | Context lines (default 3) |
| `IncludePatch` | `bool` | Include patch content |
| `NameOnly` | `bool` | Only file names |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `Branch` | `string` | Current branch name |
| `Upstream` | `string` | Upstream branch |
| `Ahead` | `int` | Commits ahead of upstream |
| `Behind` | `int` | Commits behind upstream |
| `Staged` | `[]FileStatus` | Staged files |
| `Unstaged` | `[]FileStatus` | Modified files |
| `Untracked` | `[]string` | Untracked files |
| `Conflicts` | `[]string` | Conflicted files |
| `IsClean` | `bool` | No changes present |
| `HasUntracked` | `bool` | Has untracked files |

### Commit Fields

| Field | Type | Description |
|-------|------|-------------|
| `Hash` | `string` | Full commit hash |
| `ShortHash` | `string` | Abbreviated hash |
| `Author` | `Person` | Commit author |
| `Committer` | `Person` | Committer |
| `Subject` | `string` | First line of message |
| `Body` | `string` | Full message body |
| `ParentHashes` | `[]string` | Parent commit hashes |
| `Timestamp` | `time.Time` | Commit timestamp |

### Diff Fields

| Field | Type | Description |
|-------|------|-------------|
| `Files` | `[]DiffFile` | Changed files |
| `TotalAdded` | `int` | Total lines added |
| `TotalRemoved` | `int` | Total lines removed |
| `Stats` | `string` | Summary line |

### DiffFile Fields

| Field | Type | Description |
|-------|------|-------------|
| `Path` | `string` | File path |
| `OldPath` | `string` | Old path (for renames) |
| `Status` | `string` | Change status |
| `Additions` | `int` | Lines added |
| `Deletions` | `int` | Lines removed |
| `Binary` | `bool` | Is binary file |
| `Patch` | `string` | Diff content |

### File Status Values

- `"modified"` - File modified
- `"added"` - File added
- `"deleted"` - File deleted
- `"renamed"` - File renamed
- `"copied"` - File copied
- `"typechange"` - File type changed

### Branch Fields

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Branch name |
| `Hash` | `string` | Commit hash |
| `Upstream` | `string` | Upstream branch |
| `IsCurrent` | `bool` | Is current branch |
| `IsRemote` | `bool` | Is remote branch |

### Tag Fields

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Tag name |
| `Hash` | `string` | Tag/commit hash |
| `Commit` | `string` | Commit (annotated tags) |
| `Tagger` | `*Person` | Tagger (annotated tags) |
| `Message` | `string` | Message (annotated tags) |
| `Date` | `*time.Time` | Date (annotated tags) |

### Remote Fields

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Remote name |
| `FetchURL` | `string` | Fetch URL |
| `PushURL` | `string` | Push URL |

### BlameLine Fields

| Field | Type | Description |
|-------|------|-------------|
| `Hash` | `string` | Commit hash |
| `Author` | `string` | Author name |
| `Email` | `string` | Author email |
| `Timestamp` | `time.Time` | Commit time |
| `LineNumber` | `int` | Line number |
| `Content` | `string` | Line content |

### Error Constants

| Constant | Description |
|----------|-------------|
| `ErrNotRepository` | Not a git repository |
| `ErrNoCommits` | No commits in repository |

## Related Packages

- [unidiff](../unidiff/) - Parse and display unified diffs
- [cli](../cli/) - Build CLI tools with git operations

## Implementation Notes

- All operations are read-only; no write operations supported
- Requires git binary in PATH
- Works with any directory within a git repository
- Status uses porcelain v2 format for stable parsing
- Timestamps are in UTC
- Empty slices returned for no results (never nil)
- Context cancellation supported for all operations
- Detached HEAD returns empty string for current branch
