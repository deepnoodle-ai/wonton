package tui

import (
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// These tests demonstrate shortcomings in the current flex layout system.
// The issue: fixed children are measured with maxHeight=0 (unconstrained),
// which causes views that need constraints (like Canvas) to get 0 size.
//
// In CSS Flexbox, Flutter, and SwiftUI, ALL children receive the container's
// constraints. Fixed children report their intrinsic size; flexible children
// expand to fill remaining space. The key difference is that fixed children
// are NOT measured with 0 constraints.
//
// Reference: https://www.w3.org/TR/css-flexbox-1/

// Test: Canvas inside Stack should get space when Stack has constraints
//
// Current behavior: Canvas gets height=0 because it's wrapped in a Group
// that has flex()=0, so Group is measured with maxHeight=0.
//
// Expected behavior: Canvas should fill the available space because it
// has flex()=1. The container should pass constraints to all children.
func TestFlexConstraints_CanvasInNestedContainer(t *testing.T) {
	// Canvas has flex()=1 by default (when no explicit size)
	// But when wrapped in Group (flex=0), the Group is measured unconstrained
	var canvasHeight int
	canvas := CanvasContext(func(ctx *RenderContext) {
		_, canvasHeight = ctx.Size()
	})

	// Wrap canvas in a Group (horizontal container)
	// Group has flex()=0 by default
	inner := Group(canvas)

	// Put that in a Stack with another element
	view := Stack(
		Text("Header"),
		inner, // This should expand to fill remaining space
	)

	// Render with constraints to trigger measurement
	screen := SprintScreen(view, WithWidth(80), WithHeight(24))
	_ = screen

	// BUG: Canvas gets 0 height because Group has flex()=0, so it's
	// treated as a fixed child and measured with maxHeight=0
	// Expected: Canvas should fill the remaining 23 lines (24 - 1 for header)
	assert.True(t, canvasHeight > 0, "canvas should receive height")
	assert.Equal(t, 23, canvasHeight, "canvas should fill remaining height after header")
}

// Test: Nested flex containers should propagate constraints
//
// This is the exact scenario from the user feedback:
// Stack(Group(panels...)).Flex(7)
//
// The outer Stack has Flex(7), so it gets space from its parent.
// But Group inside has flex()=0, so it's measured with maxHeight=0.
func TestFlexConstraints_NestedFlexContainers(t *testing.T) {
	// Create a canvas that tracks what size it received
	var canvasWidth, canvasHeight int
	canvas := CanvasContext(func(ctx *RenderContext) {
		canvasWidth, canvasHeight = ctx.Size()
	})

	// Nest it: Stack -> Group -> Canvas
	// Canvas.flex() = 1
	// Group.flex() = 0 (default)
	// Stack.Flex(1) explicitly set
	view := Stack(
		Group(canvas),
	).Flex(1)

	// Render with constraints
	screen := SprintScreen(view, WithWidth(40), WithHeight(20))
	_ = screen // trigger render

	// BUG: Canvas currently gets 0 height because Group is measured unconstrained
	// Expected: Canvas should get the full height (minus any fixed content)
	assert.True(t, canvasHeight > 0, "canvas should receive height constraints")
	assert.Equal(t, 20, canvasHeight, "canvas should fill available height")
	assert.Equal(t, 40, canvasWidth, "canvas should fill available width")
}

// Test: Fixed children measurement behavior
//
// Currently, fixed children (flex=0) are measured with maxHeight=0 (unconstrained).
// This is fine for views that return intrinsic sizes (like Text), but differs
// from CSS Flexbox which passes constraints to all children.
//
// The auto-derive flex fix handles most cases by making containers with
// flexible children themselves flexible. For truly fixed views that return
// intrinsic sizes, unconstrained measurement works correctly.
func TestFlexConstraints_FixedChildrenMeasurement(t *testing.T) {
	// Track what constraints the inner view receives
	var receivedMaxWidth, receivedMaxHeight int

	// Create a custom view that records its measurement constraints
	measuringView := &constraintTracker{
		onSize: func(maxW, maxH int) (int, int) {
			receivedMaxWidth = maxW
			receivedMaxHeight = maxH
			return 10, 5 // return fixed size
		},
	}

	// Put it in a Stack - it's treated as "fixed" (no flex)
	view := Stack(
		Text("Header"),
		measuringView,
		Text("Footer"),
	)

	// Measure with constraints
	view.size(80, 24)

	// Current behavior: fixed children get maxWidth but maxHeight=0
	// This is acceptable because fixed views should return intrinsic sizes
	assert.Equal(t, 80, receivedMaxWidth, "fixed child receives width constraint")
	assert.Equal(t, 0, receivedMaxHeight, "fixed child receives unconstrained height (current behavior)")

	// The returned size is still used correctly
	w, h := view.size(80, 24)
	// Total height: Header(1) + measuringView(5) + Footer(1) = 7
	assert.Equal(t, 10, w, "width should be max of children")
	assert.Equal(t, 7, h, "height should sum fixed children")
}

// Test: Group with flexible children should report appropriate size
//
// When a Group contains only flexible children (like Canvas), and it's
// measured with constraints, the flexible children should expand.
func TestFlexConstraints_GroupWithFlexibleChildren(t *testing.T) {
	// Canvas has flex()=1 by default
	canvas := Canvas(func(frame RenderFrame, bounds image.Rectangle) {})

	// Group containing only the canvas
	g := Group(canvas)

	// When measured WITH constraints, canvas should expand
	w, h := g.size(50, 30)

	// BUG: Currently w=50, h=0 because Canvas gets maxHeight=0
	// and returns 0 height when unconstrained
	assert.Equal(t, 50, w, "group width should match constraint")
	assert.Equal(t, 30, h, "group height should expand to constraint when child is flexible")
}

// Test: Stack with single flexible child should fill space
//
// A Stack containing only a Canvas (flex=1) should fill available space.
func TestFlexConstraints_StackWithSingleFlexibleChild(t *testing.T) {
	canvas := Canvas(func(frame RenderFrame, bounds image.Rectangle) {})

	s := Stack(canvas)

	w, h := s.size(60, 40)

	// Canvas has flex()=1, so it should expand to fill the Stack
	assert.Equal(t, 60, w, "stack should fill width")
	assert.Equal(t, 40, h, "stack should fill height when child is flexible")
}

// Test: Mixed fixed and flexible in nested containers - WITHOUT explicit Flex
//
// This demonstrates the bug: Group without .Flex() is treated as fixed,
// so its children (even if they are flexible) get 0 height.
func TestFlexConstraints_MixedNestedLayout_Bug(t *testing.T) {
	var canvas1H, canvas2H int

	canvas1 := CanvasContext(func(ctx *RenderContext) {
		_, canvas1H = ctx.Size()
	})
	canvas2 := CanvasContext(func(ctx *RenderContext) {
		_, canvas2H = ctx.Size()
	})

	// BUG: Group has flex()=0 by default, so it's treated as fixed
	// and measured with maxHeight=0, starving its children
	view := Stack(
		Text("Header"),          // fixed, 1 line
		Group(canvas1, canvas2), // NO .Flex() - this is the bug!
		Text("Footer"),          // fixed, 1 line
	)

	screen := SprintScreen(view, WithWidth(80), WithHeight(24))
	_ = screen

	// Header and footer take 2 lines, leaving 22 for the Group
	expectedCanvasHeight := 22

	// BUG: Currently canvases get 0 height because Group is fixed
	// Expected: Canvas should fill the remaining space
	assert.True(t, canvas1H > 0, "canvas1 should have height (currently 0 due to bug)")
	assert.True(t, canvas2H > 0, "canvas2 should have height (currently 0 due to bug)")
	assert.Equal(t, expectedCanvasHeight, canvas1H,
		"canvas1 should get remaining height after fixed children")
	assert.Equal(t, expectedCanvasHeight, canvas2H,
		"canvas2 should get remaining height after fixed children")
}

// Test: Workaround - explicit .Flex(1) on Group makes it work
//
// This test shows the current workaround: adding .Flex(1) to the Group
// makes it a flex child, so it gets measured with constraints.
func TestFlexConstraints_MixedNestedLayout_Workaround(t *testing.T) {
	var canvas1H, canvas2H int

	canvas1 := CanvasContext(func(ctx *RenderContext) {
		_, canvas1H = ctx.Size()
	})
	canvas2 := CanvasContext(func(ctx *RenderContext) {
		_, canvas2H = ctx.Size()
	})

	// WORKAROUND: Add .Flex(1) to make Group a flex child
	view := Stack(
		Text("Header"),                  // fixed, 1 line
		Group(canvas1, canvas2).Flex(1), // .Flex(1) makes it work
		Text("Footer"),                  // fixed, 1 line
	)

	screen := SprintScreen(view, WithWidth(80), WithHeight(24))
	_ = screen

	// Header and footer take 2 lines, leaving 22 for the Group
	expectedCanvasHeight := 22

	// This works because Group has .Flex(1), so it's treated as flexible
	assert.Equal(t, expectedCanvasHeight, canvas1H,
		"canvas1 should get remaining height (workaround works)")
	assert.Equal(t, expectedCanvasHeight, canvas2H,
		"canvas2 should get remaining height (workaround works)")
}

// Test: Verify CSS-like behavior - flex-basis equivalent
//
// In CSS, flex-basis:auto means use content size as starting point.
// Views with flex-grow:0 don't expand but DO get measured with constraints.
func TestFlexConstraints_FlexBasisBehavior(t *testing.T) {
	// Text has intrinsic size and flex()=0
	// It should NOT expand, but should be measured with constraints
	text := Text("Hello")

	// Put it in a Stack with a Spacer
	view := Stack(
		text,
		Spacer(),
	)

	w, h := view.size(100, 50)

	// Text should take its intrinsic size, Spacer fills the rest
	assert.Equal(t, 5, w, "width should be text width (intrinsic)")
	assert.Equal(t, 50, h, "total height should fill constraint")
}

// Test: Intrinsic size calculation for containers
//
// A container's intrinsic size should be computed from children's
// intrinsic sizes, not be 0 when children are flexible.
func TestFlexConstraints_ContainerIntrinsicSize(t *testing.T) {
	// Canvas with no explicit size - it's flexible
	canvas := Canvas(func(frame RenderFrame, bounds image.Rectangle) {})

	// When measured WITHOUT constraints (intrinsic sizing)
	g := Group(canvas)
	w, h := g.size(0, 0)

	// A flexible view's intrinsic size can be 0 - that's fine
	// But when measured WITH constraints, it should expand
	assert.Equal(t, 0, w, "intrinsic width can be 0 for flexible view")
	assert.Equal(t, 0, h, "intrinsic height can be 0 for flexible view")

	// Now measure WITH constraints - this is the key test
	w, h = g.size(50, 30)

	// BUG: Still returns 0 height
	assert.Equal(t, 50, w, "should expand to width constraint")
	assert.Equal(t, 30, h, "should expand to height constraint")
}

// constraintTracker is a helper view that records measurement constraints
type constraintTracker struct {
	onSize func(maxWidth, maxHeight int) (int, int)
}

func (c *constraintTracker) size(maxWidth, maxHeight int) (int, int) {
	if c.onSize != nil {
		return c.onSize(maxWidth, maxHeight)
	}
	return 0, 0
}

func (c *constraintTracker) render(ctx *RenderContext) {}

// Test: Render verification - Canvas should actually render content
//
// When Canvas is nested inside Group inside Stack, it should still render.
// With the auto-derive flex fix, Group inherits flex from Canvas.
func TestFlexConstraints_CanvasRenders(t *testing.T) {
	var canvasRendered bool
	var canvasWidth, canvasHeight int

	view := Stack(
		Text("Top"),
		Group(
			CanvasContext(func(ctx *RenderContext) {
				canvasRendered = true
				canvasWidth, canvasHeight = ctx.Size()
			}),
		), // Group now has flex()=1 (auto-derived from Canvas)
		Text("Bottom"),
	)

	// Trigger rendering
	screen := SprintScreen(view, WithWidth(20), WithHeight(10))
	_ = screen

	// Core behavior: Canvas should now render because Group inherits flex from Canvas
	assert.True(t, canvasRendered, "canvas should be rendered")
	assert.True(t, canvasHeight > 0, "canvas should have height")
	assert.Equal(t, 20, canvasWidth, "canvas should have full width")

	// Height should be the remaining space after fixed text views
	// 10 total - 1 (Top) - 1 (Bottom) = 8
	assert.Equal(t, 8, canvasHeight, "canvas should fill remaining height")
}

// Test: The exact user feedback scenario
//
// tui.Stack(tui.Group(panels...)).Flex(7)
//
// User expects: Stack gets 70% of space, Group fills Stack, panels fill Group
// Actual: Group measured with maxHeight=0, panels get no space
func TestFlexConstraints_UserFeedbackScenario(t *testing.T) {
	var panelHeight int

	panel := CanvasContext(func(ctx *RenderContext) {
		_, panelHeight = ctx.Size()
	})

	// This is exactly the pattern from user feedback
	view := Stack(
		Stack(Group(panel)).Flex(7), // 70% of space
		Spacer().Flex(3),            // 30% of space (placeholder)
	)

	screen := SprintScreen(view, WithWidth(80), WithHeight(100))
	_ = screen

	// Stack with Flex(7) should get 70 lines (70% of 100)
	// Group inside should fill that space
	// Panel inside Group should fill Group
	expectedHeight := 70

	// BUG: panelHeight is 0
	assert.True(t, panelHeight > 0, "panel should receive height")
	assert.Equal(t, expectedHeight, panelHeight,
		"panel should get 70%% of container height")
}
