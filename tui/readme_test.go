package tui

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
// compiles it, so documented APIs cannot drift from the real ones.
//
// Each block is compiled as its own package. Blocks containing "package main"
// are compiled verbatim; other blocks are wrapped: top-level declarations get
// a package clause and imports, and bare statements are additionally wrapped
// in a function body. A small scaffold supplies the surrounding context some
// snippets assume (an `app` type, free variables like `selected`), driven by
// simple content probes. If you add a snippet that needs new context, extend
// the scaffolding here rather than bloating the README.
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
	if len(blocks) < 10 {
		t.Fatalf("expected to find many ```go blocks in README.md, got %d", len(blocks))
	}

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	gomod := fmt.Sprintf(`module readme.test/snippets

go 1.21

require github.com/deepnoodle-ai/wonton v0.0.0

replace github.com/deepnoodle-ai/wonton => %s
`, repoRoot)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}

	for i, b := range blocks {
		name := fmt.Sprintf("snippet%02d_line%04d", i+1, b.line)
		snippetDir := filepath.Join(dir, name)
		if err := os.MkdirAll(snippetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		file := renderSnippet(b.code)
		if err := os.WriteFile(filepath.Join(snippetDir, "snippet.go"), []byte(file), 0o644); err != nil {
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

type readmeBlock struct {
	line int // 1-based README line of the ```go fence
	code string
}

func extractGoBlocks(readme string) []readmeBlock {
	var blocks []readmeBlock
	lines := strings.Split(readme, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") != "```go" {
			continue
		}
		start := i + 1
		j := start
		for j < len(lines) && strings.TrimRight(lines[j], " \t") != "```" {
			j++
		}
		blocks = append(blocks, readmeBlock{
			line: i + 1,
			code: strings.Join(lines[start:j], "\n"),
		})
		i = j
	}
	return blocks
}

var topLevelKeyword = regexp.MustCompile(`^(func|type|var|const|import|package)\b`)
var shortVarDecl = regexp.MustCompile(`^([A-Za-z_]\w*(?:\s*,\s*[A-Za-z_]\w*)*)\s*:=`)

// renderSnippet turns one README code block into a compilable Go file.
func renderSnippet(code string) string {
	if strings.Contains(code, "package main") {
		return code
	}

	var sb strings.Builder
	sb.WriteString("package snippet\n\n")

	// Imports, anchored with a blank use so an import a snippet doesn't
	// reference never reports as unused.
	type imp struct{ probe, spec, anchor string }
	imports := []imp{
		{"tui.", "tui \"github.com/deepnoodle-ai/wonton/tui\"", "var _ = tui.Text"},
		{"fmt.", "\"fmt\"", "var _ = fmt.Println"},
		{"log.", "\"log\"", "var _ = log.Println"},
		{"time.", "\"time\"", "var _ = time.Now"},
		{"strings.", "\"strings\"", "var _ = strings.Join"},
		{"testing.", "\"testing\"", "var _ testing.TB"},
		{"termtest.", "\"github.com/deepnoodle-ai/wonton/termtest\"", "var _ = termtest.AssertScreen"},
	}
	var specs, anchors []string
	for _, im := range imports {
		if strings.Contains(code, im.probe) {
			specs = append(specs, im.spec)
			anchors = append(anchors, im.anchor)
		}
	}
	// Blocks that never qualify identifiers with "tui." are written from the
	// package's own point of view (golden tests, layout sketches); give them
	// a dot import. Blocks that do use "tui." must qualify everything, so
	// they deliberately do not get one.
	if !strings.Contains(code, "tui.") {
		specs = append(specs, ". \"github.com/deepnoodle-ai/wonton/tui\"")
		anchors = append(anchors, "var _ = Empty")
	}
	if len(specs) > 0 {
		sb.WriteString("import (\n")
		for _, s := range specs {
			sb.WriteString("\t" + s + "\n")
		}
		sb.WriteString(")\n\n")
	}
	for _, a := range anchors {
		sb.WriteString(a + "\n")
	}
	if len(anchors) > 0 {
		sb.WriteString("\n")
	}

	// Context scaffolding for snippets that assume surrounding app code.
	usesAppReceiver := strings.Contains(code, "*app)")
	definesAppType := strings.Contains(code, "type app struct")
	if usesAppReceiver && !definesAppType {
		sb.WriteString("type app struct {\n\tmenuIdx int\n\tdata []string\n}\n\n")
		if !strings.Contains(code, "func (a *app) renderContent()") {
			sb.WriteString("func (a *app) renderContent() tui.View { return tui.Empty() }\n\n")
		}
	}
	if strings.Contains(code, "tui.Run(app") && !usesAppReceiver && !definesAppType {
		sb.WriteString("var app tui.Application\n\n")
	}
	if strings.Contains(code, "&selected") && !strings.Contains(code, "selected :=") && !strings.Contains(code, "selected int") {
		sb.WriteString("var selected int\n\nvar input string\n\n")
	}
	if strings.Contains(code, "a.printResults()") {
		sb.WriteString("type printerApp struct{ running bool }\n\nfunc (p *printerApp) printResults() {}\n\nvar a printerApp\n\n")
	}

	if snippetIsTopLevel(code) {
		sb.WriteString(code)
		sb.WriteString("\n")
		return sb.String()
	}

	// Statement snippet: wrap in a function, then blank-assign every
	// top-level short variable declaration so unused variables (fine in
	// documentation) don't fail the compile.
	sb.WriteString("func snippetBody() {\n")
	sb.WriteString(code)
	sb.WriteString("\n")
	for _, line := range strings.Split(code, "\n") {
		m := shortVarDecl.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		for _, name := range strings.Split(m[1], ",") {
			name = strings.TrimSpace(name)
			if name != "_" && name != "" {
				sb.WriteString("\t_ = " + name + "\n")
			}
		}
	}
	sb.WriteString("}\n\nvar _ = snippetBody\n")
	return sb.String()
}

// snippetIsTopLevel reports whether the block's first significant line
// starts a top-level declaration (as opposed to bare statements).
func snippetIsTopLevel(code string) bool {
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		return topLevelKeyword.MatchString(trimmed)
	}
	return false
}
