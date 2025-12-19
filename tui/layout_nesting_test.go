package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNestedLayout_Collapsing(t *testing.T) {
	// Scenario: A Stack containing a nested Stack with a Spacer.
	// We want the inner Stack to expand to fill the available height.
	//
	// Parent Stack (Height 10)
	// ├── Text "Top" (Height 1)
	// ├── Nested Stack (Should take 8)
	// │   └── Spacer
	// └── Text "Bottom" (Height 1)

	nestedStack := Stack(
		Spacer(),
	).Flex(1)

	parent := Stack(
		Text("Top"),
		nestedStack,
		Text("Bottom"),
	)

	// Measure with constrained height of 10
	w, h := parent.size(20, 10)

	// With Flex(1), the nested stack should participate in flex distribution.
	// 1. Parent Stack iterates children.
	// 2. nestedStack implements Flexible (flex=1).
	// 3. Parent measures fixed children ("Top", "Bottom") -> height 2.
	// 4. Remaining height = 10 - 2 = 8.
	// 5. Parent measures nestedStack with size(width, 8).
	// 6. nestedStack measures Spacer with size(width, 8).
	// 7. Spacer returns (width, 8).
	// 8. nestedStack returns (width, 8).
	// 9. Parent total height = 2 + 8 = 10.

	assert.Equal(t, 10, h, "Nested stack should expand to fill height")
	_ = w
}

func TestNestedGroupLayout(t *testing.T) {
	// Scenario: A Group containing a nested Group with a Spacer.
	// Parent Group (Width 20)
	// ├── Text "L" (Width 1)
	// ├── Nested Group (Should take 18)
	// │   └── Spacer
	// └── Text "R" (Width 1)

	nestedGroup := Group(
		Spacer(),
	).Flex(1)

	parent := Group(
		Text("L"),
		nestedGroup,
		Text("R"),
	)

	// Measure with constrained width of 20
	w, h := parent.size(20, 10)

	assert.Equal(t, 20, w, "Nested group should expand to fill width")
	assert.Equal(t, 1, h)
}
