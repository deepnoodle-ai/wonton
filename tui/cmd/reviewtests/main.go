// reviewtests displays golden test code alongside their snapshots for LLM review.
//
// Usage:
//
//	go run ./tui/cmd/reviewtests                     # Show all tests
//	go run ./tui/cmd/reviewtests Flex                # Show tests matching "Flex"
//	go run ./tui/cmd/reviewtests Flex Size           # Show tests matching "Flex" OR "Size"
//	go run ./tui/cmd/reviewtests -compact            # Compact output (less whitespace)
//	go run ./tui/cmd/reviewtests -list               # Just list test names
//	go run ./tui/cmd/reviewtests -stats              # Show category statistics
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	compact = flag.Bool("compact", false, "Compact output format")
	list    = flag.Bool("list", false, "List test names only")
	stats   = flag.Bool("stats", false, "Show category statistics")
)

type TestCase struct {
	Name     string
	Category string
	Code     string
	Snapshot string
}

func main() {
	flag.Parse()
	patterns := flag.Args()

	// Find the tui directory
	tuiDir := findTuiDir()
	if tuiDir == "" {
		fmt.Fprintln(os.Stderr, "Error: could not find tui directory")
		os.Exit(1)
	}

	// Parse tests from golden_test.go
	testFile := filepath.Join(tuiDir, "golden_test.go")
	tests, err := parseTests(testFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing tests: %v\n", err)
		os.Exit(1)
	}

	// Load snapshots
	snapshotDir := filepath.Join(tuiDir, "testdata", "snapshots")
	for i := range tests {
		snapshotPath := filepath.Join(snapshotDir, sanitizeName(tests[i].Name)+".snap")
		data, err := os.ReadFile(snapshotPath)
		if err == nil {
			tests[i].Snapshot = string(data)
		} else {
			tests[i].Snapshot = "(snapshot not found)"
		}
	}

	// Filter by patterns
	if len(patterns) > 0 {
		filtered := []TestCase{}
		for _, t := range tests {
			for _, p := range patterns {
				if strings.Contains(t.Name, p) || strings.Contains(t.Category, p) {
					filtered = append(filtered, t)
					break
				}
			}
		}
		tests = filtered
	}

	if len(tests) == 0 {
		fmt.Println("No tests match the given patterns.")
		return
	}

	// Output based on mode
	if *stats {
		showStats(tests)
	} else if *list {
		showList(tests)
	} else {
		showFull(tests, *compact)
	}
}

func findTuiDir() string {
	// Try relative paths
	candidates := []string{
		"tui",
		".",
		"../tui",
		"../../tui",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "golden_test.go")); err == nil {
			return c
		}
	}
	return ""
}

func parseTests(filename string) ([]TestCase, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tests []TestCase
	var currentTest *TestCase
	var codeLines []string
	var inTest bool
	braceCount := 0

	// Regex to match test function declarations
	testFuncRe := regexp.MustCompile(`^func (TestGolden_\w+)\(t \*testing\.T\)`)
	// Regex to match category comments
	categoryRe := regexp.MustCompile(`^// =+$`)

	scanner := bufio.NewScanner(file)
	currentCategory := ""
	prevLine := ""

	for scanner.Scan() {
		line := scanner.Text()

		// Track category headers (lines like "// FLEX BEHAVIOR TESTS")
		if categoryRe.MatchString(line) && strings.HasPrefix(prevLine, "//") {
			// Extract category from the previous comment line
			cat := strings.TrimPrefix(prevLine, "//")
			cat = strings.TrimSpace(cat)
			if idx := strings.Index(cat, " - "); idx > 0 {
				cat = cat[:idx]
			}
			if idx := strings.Index(cat, " TESTS"); idx > 0 {
				cat = cat[:idx]
			}
			currentCategory = strings.TrimSpace(cat)
		}

		// Check for test function start
		if matches := testFuncRe.FindStringSubmatch(line); matches != nil {
			if currentTest != nil && len(codeLines) > 0 {
				currentTest.Code = strings.Join(codeLines, "\n")
				tests = append(tests, *currentTest)
			}
			currentTest = &TestCase{
				Name:     matches[1],
				Category: currentCategory,
			}
			codeLines = []string{line}
			inTest = true
			braceCount = 1 // Opening brace is on this line
			prevLine = line
			continue
		}

		if inTest {
			codeLines = append(codeLines, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				// End of function
				currentTest.Code = strings.Join(codeLines, "\n")
				tests = append(tests, *currentTest)
				currentTest = nil
				codeLines = nil
				inTest = false
			}
		}

		prevLine = line
	}

	// Handle last test if file doesn't end with newline
	if currentTest != nil && len(codeLines) > 0 {
		currentTest.Code = strings.Join(codeLines, "\n")
		tests = append(tests, *currentTest)
	}

	return tests, scanner.Err()
}

func sanitizeName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}

func showStats(tests []TestCase) {
	categories := make(map[string]int)
	for _, t := range tests {
		cat := t.Category
		if cat == "" {
			cat = "(uncategorized)"
		}
		categories[cat]++
	}

	// Sort categories
	var cats []string
	for c := range categories {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	fmt.Printf("Golden Test Statistics (%d tests total)\n", len(tests))
	fmt.Println(strings.Repeat("=", 50))
	for _, c := range cats {
		fmt.Printf("%-35s %3d tests\n", c, categories[c])
	}
}

func showList(tests []TestCase) {
	currentCat := ""
	for _, t := range tests {
		if t.Category != currentCat {
			if currentCat != "" {
				fmt.Println()
			}
			currentCat = t.Category
			if currentCat != "" {
				fmt.Printf("## %s\n", currentCat)
			}
		}
		fmt.Printf("  %s\n", t.Name)
	}
}

func showFull(tests []TestCase, compact bool) {
	fmt.Println("# Golden Tests Review")
	fmt.Printf("# %d tests to review\n", len(tests))
	fmt.Println("#")
	fmt.Println("# For each test, review:")
	fmt.Println("#   1. Does the test code correctly set up the scenario?")
	fmt.Println("#   2. Does the snapshot match expected visual output?")
	fmt.Println("#   3. Are edge cases properly handled?")
	fmt.Println()

	currentCat := ""
	for i, t := range tests {
		// Category header
		if t.Category != currentCat && t.Category != "" {
			currentCat = t.Category
			fmt.Printf("## %s\n", currentCat)
			if !compact {
				fmt.Println()
			}
		}

		// Test header
		fmt.Printf("### [%d/%d] %s\n", i+1, len(tests), t.Name)
		if !compact {
			fmt.Println()
		}

		// Code section
		fmt.Println("<code>")
		fmt.Println(t.Code)
		fmt.Println("</code>")
		if !compact {
			fmt.Println()
		}

		// Snapshot section
		snapshot := t.Snapshot
		// Trim trailing newlines for cleaner display
		snapshot = strings.TrimRight(snapshot, "\n")
		fmt.Println("<snapshot>")
		if snapshot == "" {
			fmt.Println("(empty)")
		} else {
			fmt.Println(snapshot)
		}
		fmt.Println("</snapshot>")
		fmt.Println()
	}

	fmt.Println("# END OF REVIEW")
}
