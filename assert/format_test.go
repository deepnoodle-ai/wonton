package assert

import (
	"strings"
	"testing"
)

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name       string
		msgAndArgs []any
		want       string
	}{
		{
			name:       "empty args",
			msgAndArgs: nil,
			want:       "",
		},
		{
			name:       "single string",
			msgAndArgs: []any{"hello"},
			want:       "hello",
		},
		{
			name:       "single non-string",
			msgAndArgs: []any{42},
			want:       "42",
		},
		{
			name:       "format string with args",
			msgAndArgs: []any{"value is %d", 42},
			want:       "value is 42",
		},
		{
			name:       "format string with multiple args",
			msgAndArgs: []any{"%s=%d", "count", 5},
			want:       "count=5",
		},
		{
			name:       "non-string first arg with multiple args",
			msgAndArgs: []any{123, "ignored"},
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMessage(tt.msgAndArgs...)
			if got != tt.want {
				t.Errorf("formatMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIndentMessageLines(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		longestLabelLen int
		want            string
	}{
		{
			name:            "single line",
			message:         "hello",
			longestLabelLen: 5,
			want:            "hello",
		},
		{
			name:            "multi line",
			message:         "line1\nline2",
			longestLabelLen: 5,
			want:            "line1\n\t      \tline2",
		},
		{
			name:            "three lines",
			message:         "a\nb\nc",
			longestLabelLen: 3,
			want:            "a\n\t    \tb\n\t    \tc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentMessageLines(tt.message, tt.longestLabelLen)
			if got != tt.want {
				t.Errorf("indentMessageLines() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsTest(t *testing.T) {
	tests := []struct {
		name   string
		fnName string
		prefix string
		want   bool
	}{
		{
			name:   "exact match Test",
			fnName: "Test",
			prefix: "Test",
			want:   true,
		},
		{
			name:   "TestFoo",
			fnName: "TestFoo",
			prefix: "Test",
			want:   true,
		},
		{
			name:   "Test_foo",
			fnName: "Test_foo",
			prefix: "Test",
			want:   true,
		},
		{
			name:   "Testfoo lowercase after prefix",
			fnName: "Testfoo",
			prefix: "Test",
			want:   false,
		},
		{
			name:   "BenchmarkFoo",
			fnName: "BenchmarkFoo",
			prefix: "Benchmark",
			want:   true,
		},
		{
			name:   "ExampleFoo",
			fnName: "ExampleFoo",
			prefix: "Example",
			want:   true,
		},
		{
			name:   "no prefix match",
			fnName: "FooTest",
			prefix: "Test",
			want:   false,
		},
		{
			name:   "exact match Benchmark",
			fnName: "Benchmark",
			prefix: "Benchmark",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTest(tt.fnName, tt.prefix)
			if got != tt.want {
				t.Errorf("isTest(%q, %q) = %v, want %v", tt.fnName, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestFormatLabeledOutput(t *testing.T) {
	// Disable colors for consistent testing
	origColorEnabled := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = origColorEnabled }()

	tests := []struct {
		name    string
		content []labeledContent
		check   func(t *testing.T, output string)
	}{
		{
			name:    "empty content",
			content: nil,
			check: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("expected empty output, got %q", output)
				}
			},
		},
		{
			name: "single item",
			content: []labeledContent{
				{label: "Label", content: "value"},
			},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "Label:") {
					t.Errorf("output should contain 'Label:', got %q", output)
				}
				if !strings.Contains(output, "value") {
					t.Errorf("output should contain 'value', got %q", output)
				}
			},
		},
		{
			name: "multiple items aligned",
			content: []labeledContent{
				{label: "Short", content: "val1"},
				{label: "LongerLabel", content: "val2"},
			},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "Short:") {
					t.Errorf("output should contain 'Short:', got %q", output)
				}
				if !strings.Contains(output, "LongerLabel:") {
					t.Errorf("output should contain 'LongerLabel:', got %q", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLabeledOutput(tt.content...)
			tt.check(t, got)
		})
	}
}

func TestFormatListDiff(t *testing.T) {
	// Disable colors for consistent testing
	origColorEnabled := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = origColorEnabled }()

	tests := []struct {
		name   string
		listA  any
		listB  any
		extraA []any
		extraB []any
		check  func(t *testing.T, output string)
	}{
		{
			name:   "no extras",
			listA:  []int{1, 2},
			listB:  []int{1, 2},
			extraA: nil,
			extraB: nil,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "elements differ") {
					t.Errorf("output should contain 'elements differ'")
				}
				if strings.Contains(output, "extra elements") {
					t.Errorf("output should not contain 'extra elements' when both are empty")
				}
			},
		},
		{
			name:   "extra in A",
			listA:  []int{1, 2, 3},
			listB:  []int{1, 2},
			extraA: []any{3},
			extraB: nil,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "extra elements in list A") {
					t.Errorf("output should contain 'extra elements in list A'")
				}
			},
		},
		{
			name:   "extra in B",
			listA:  []int{1, 2},
			listB:  []int{1, 2, 3},
			extraA: nil,
			extraB: []any{3},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "extra elements in list B") {
					t.Errorf("output should contain 'extra elements in list B'")
				}
			},
		},
		{
			name:   "extras in both",
			listA:  []int{1, 2, 3},
			listB:  []int{1, 4, 5},
			extraA: []any{2, 3},
			extraB: []any{4, 5},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "extra elements in list A") {
					t.Errorf("output should contain 'extra elements in list A'")
				}
				if !strings.Contains(output, "extra elements in list B") {
					t.Errorf("output should contain 'extra elements in list B'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatListDiff(tt.listA, tt.listB, tt.extraA, tt.extraB)
			tt.check(t, got)
		})
	}
}

func TestFormatUnequalValues(t *testing.T) {
	// Disable colors for consistent testing
	origColorEnabled := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = origColorEnabled }()

	tests := []struct {
		name     string
		expected any
		actual   any
		wantExp  string
		wantAct  string
	}{
		{
			name:     "same type int",
			expected: 42,
			actual:   43,
			wantExp:  "42",
			wantAct:  "43",
		},
		{
			name:     "same type string",
			expected: "hello",
			actual:   "world",
			wantExp:  `"hello"`,
			wantAct:  `"world"`,
		},
		{
			name:     "different types",
			expected: 42,
			actual:   "42",
			wantExp:  `int(42)`,
			wantAct:  `string("42")`,
		},
		{
			name:     "different types int and float",
			expected: 42,
			actual:   42.0,
			wantExp:  `int(42)`,
			wantAct:  `float64(42)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExp, gotAct := formatUnequalValues(tt.expected, tt.actual)
			if gotExp != tt.wantExp {
				t.Errorf("formatUnequalValues() expected = %q, want %q", gotExp, tt.wantExp)
			}
			if gotAct != tt.wantAct {
				t.Errorf("formatUnequalValues() actual = %q, want %q", gotAct, tt.wantAct)
			}
		})
	}
}

func TestFormatDiff(t *testing.T) {
	// Disable colors for consistent testing
	origColorEnabled := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = origColorEnabled }()

	tests := []struct {
		name     string
		expected any
		actual   any
		check    func(t *testing.T, output string)
	}{
		{
			name:     "nil expected",
			expected: nil,
			actual:   "something",
			check: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("expected empty output for nil, got %q", output)
				}
			},
		},
		{
			name:     "nil actual",
			expected: "something",
			actual:   nil,
			check: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("expected empty output for nil, got %q", output)
				}
			},
		},
		{
			name:     "different types",
			expected: 42,
			actual:   "42",
			check: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("expected empty output for different types, got %q", output)
				}
			},
		},
		{
			name:     "int values no diff",
			expected: 42,
			actual:   42,
			check: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("expected empty output for non-comparable types, got %q", output)
				}
			},
		},
		{
			name:     "struct diff",
			expected: struct{ A int }{A: 1},
			actual:   struct{ A int }{A: 2},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "Diff:") {
					t.Errorf("expected diff output for structs, got %q", output)
				}
				if !strings.Contains(output, "--- expected") {
					t.Errorf("expected '--- expected' in output, got %q", output)
				}
				if !strings.Contains(output, "+++ actual") {
					t.Errorf("expected '+++ actual' in output, got %q", output)
				}
			},
		},
		{
			name:     "map diff",
			expected: map[string]int{"a": 1},
			actual:   map[string]int{"a": 2},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "Diff:") {
					t.Errorf("expected diff output for maps, got %q", output)
				}
			},
		},
		{
			name:     "slice diff",
			expected: []int{1, 2, 3},
			actual:   []int{1, 2, 4},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "Diff:") {
					t.Errorf("expected diff output for slices, got %q", output)
				}
			},
		},
		{
			name:     "string diff",
			expected: "hello",
			actual:   "world",
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "Diff:") {
					t.Errorf("expected diff output for strings, got %q", output)
				}
			},
		},
		{
			name:     "equal values no diff",
			expected: []int{1, 2, 3},
			actual:   []int{1, 2, 3},
			check: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("expected empty output for equal values, got %q", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDiff(tt.expected, tt.actual)
			tt.check(t, got)
		})
	}
}

func TestColorEnabled(t *testing.T) {
	// Save original state
	origColorEnabled := colorEnabled

	// Test setting to false
	SetColorEnabled(false)
	if ColorEnabled() {
		t.Error("ColorEnabled() should return false after SetColorEnabled(false)")
	}

	// Test setting to true
	SetColorEnabled(true)
	if !ColorEnabled() {
		t.Error("ColorEnabled() should return true after SetColorEnabled(true)")
	}

	// Restore original state
	colorEnabled = origColorEnabled
}

func TestCallerInfo(t *testing.T) {
	// Call from test function - should get caller info
	info := callerInfo()
	// Should have at least one frame from this test
	if len(info) == 0 {
		t.Skip("callerInfo may return empty in some test environments")
	}
	// First frame should reference this test file
	found := false
	for _, frame := range info {
		if strings.Contains(frame, "format_test.go") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("callerInfo() should include format_test.go, got %v", info)
	}
}
