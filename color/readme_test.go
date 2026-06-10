package color_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestReadmeSnippetsCompile extracts every ```go block from README.md and
// compiles it, so documented APIs cannot drift from the real ones. Every
// snippet in this package's README is a complete program ("package main");
// keep new snippets in that form.
func TestReadmeSnippetsCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping README compile check in short mode")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go binary not available")
	}

	source, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	blocks := extractGoBlocks(string(source))
	if len(blocks) < 5 {
		t.Fatalf("expected to find several ```go blocks in README.md, got %d", len(blocks))
	}

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}

	// Mirror the parent module's go directive so the snippets build with the
	// same language version as wonton itself.
	goDirective := "go 1.21"
	if data, err := os.ReadFile(filepath.Join(repoRoot, "go.mod")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "go ") {
				goDirective = strings.TrimSpace(line)
				break
			}
		}
	}

	dir := t.TempDir()
	gomod := fmt.Sprintf(`module readme.test/snippets

%s

require github.com/deepnoodle-ai/wonton v0.0.0

replace github.com/deepnoodle-ai/wonton => %s
`, goDirective, repoRoot)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}

	for i, b := range blocks {
		if !strings.Contains(b.code, "package main") {
			t.Errorf("README snippet at line %d is not a complete program; "+
				"snippets in this README must include 'package main'", b.line)
			continue
		}
		name := fmt.Sprintf("snippet%02d_line%04d", i+1, b.line)
		snippetDir := filepath.Join(dir, name)
		if err := os.MkdirAll(snippetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(snippetDir, "snippet.go"), []byte(b.code), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command(goBin, "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("README snippets failed to compile (snippet dirs are named after their README line):\n%s", out)
	}
}

type goBlock struct {
	line int
	code string
}

var goFenceRe = regexp.MustCompile("(?ms)^```go\n(.*?)^```$")

// extractGoBlocks returns each ```go fenced block in the markdown source
// along with the 1-based line number where its code begins.
func extractGoBlocks(source string) []goBlock {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	var blocks []goBlock
	for _, m := range goFenceRe.FindAllStringSubmatchIndex(source, -1) {
		code := source[m[2]:m[3]]
		line := strings.Count(source[:m[2]], "\n") + 1
		blocks = append(blocks, goBlock{line: line, code: code})
	}
	return blocks
}
