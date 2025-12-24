# Golden Test Review Tool

A tool for reviewing TUI golden tests with their snapshots, optimized for LLM/AI agent review.

## Usage

```bash
# From the wonton repository root:

# Show all tests (full output with code + snapshots)
go run ./tui/cmd/reviewtests

# Filter by pattern (matches test name or category)
go run ./tui/cmd/reviewtests Flex           # Tests containing "Flex"
go run ./tui/cmd/reviewtests Flex Size      # Tests matching "Flex" OR "Size"
go run ./tui/cmd/reviewtests Table          # All table-related tests

# Compact output (less whitespace)
go run ./tui/cmd/reviewtests -compact Border

# List test names only (no code/snapshots)
go run ./tui/cmd/reviewtests -list
go run ./tui/cmd/reviewtests -list Stack

# Show category statistics
go run ./tui/cmd/reviewtests -stats
```

## Output Formats

### Full Output (default)

Shows each test with:
- Markdown headers for category and test name
- Code in `<code>` tags
- Snapshot in `<snapshot>` tags

```
## ZSTACK

### [1/2] TestGolden_ZStack_Layers

<code>
func TestGolden_ZStack_Layers(t *testing.T) {
    view := ZStack(
        Fill('.'),
        Text("Overlay"),
    )
    screen := SprintScreen(view, WithWidth(15), WithHeight(3))
    termtest.AssertScreen(t, screen)
}
</code>

<snapshot>
....Overlay....
...............
</snapshot>
```

### Statistics (-stats)

```
Golden Test Statistics (113 tests total)
==================================================
AUTO-WIDTH SCENARIOS                  4 tests
FLEX BEHAVIOR                        11 tests
SIZE CONSTRAINT                      17 tests
...
```

### List (-list)

```
## FLEX BEHAVIOR
  TestGolden_Flex_EqualDistribution
  TestGolden_Flex_UnequalFactors
  ...
```

## LLM Review Workflow

### Review All Tests

```bash
go run ./tui/cmd/reviewtests > review.txt
# Then provide review.txt to the LLM
```

### Review Specific Category

```bash
# Review all flex-related tests
go run ./tui/cmd/reviewtests Flex

# Review sizing behavior
go run ./tui/cmd/reviewtests Size Parent Auto

# Review table column handling
go run ./tui/cmd/reviewtests TableWidth
```

### Quick Audit

```bash
# First check stats
go run ./tui/cmd/reviewtests -stats

# Then review specific categories that need attention
go run ./tui/cmd/reviewtests -compact Edge
```

## What to Look For

When reviewing tests, check:

1. **Test Setup** - Does the code correctly create the scenario being tested?
2. **Snapshot Accuracy** - Does the visual output match expectations?
3. **Edge Cases** - Are boundary conditions handled (empty, very small, overflow)?
4. **Flex Behavior** - Do spacers distribute space correctly?
5. **Size Constraints** - Do Min/Max/Fixed constraints apply properly?
6. **Nesting** - Do nested layouts compose correctly?

## Test Categories

| Category | Description |
|----------|-------------|
| TEXT VIEW | Basic text rendering, styles, unicode |
| STACK LAYOUT | Vertical layout, gaps, alignment |
| GROUP LAYOUT | Horizontal layout |
| ZSTACK | Layered/overlay layout |
| BORDER & PADDING | Borders, padding modifiers |
| FLEX BEHAVIOR | Spacer distribution, flex factors |
| SIZE CONSTRAINT | Min/Max/Fixed width/height |
| PARENT/CHILD SIZE | Auto-sizing, constraint propagation |
| TABLE | Table rendering, column widths |
| COMPLEX NESTED | Multi-level nested layouts |
| EDGE CASES | Boundary conditions |
