package tui

import (
	"bytes"
	"testing"
)

// renderWithRegistries renders a view through a context bound to the given
// registries, the way a Runtime or InlineApp does.
func renderWithRegistries(t *testing.T, view View, reg *registries) {
	t.Helper()
	var buf bytes.Buffer
	terminal := NewTestTerminal(30, 5, &buf)
	frame, err := terminal.BeginFrame()
	if err != nil {
		t.Fatalf("BeginFrame: %v", err)
	}
	ctx := NewRenderContext(frame, 0).withRegistries(reg)
	view.size(30, 5)
	view.render(ctx)
	terminal.EndFrame(frame)
}

// TestRegistriesClickIsolation verifies that clickable regions registered by
// two different runtimes don't leak into each other: a click routed through
// runtime A's registries can never fire runtime B's callback.
func TestRegistriesClickIsolation(t *testing.T) {
	regA := newRegistries()
	regB := newRegistries()

	clickedA, clickedB := false, false
	renderWithRegistries(t, Clickable("AAAA", func() { clickedA = true }), regA)
	renderWithRegistries(t, Clickable("BBBB", func() { clickedB = true }), regB)

	// Both clickables occupy the same screen coordinates.
	regA.interactive.HandleClick(1, 0)
	if !clickedA {
		t.Error("click through registries A should fire A's callback")
	}
	if clickedB {
		t.Error("click through registries A must not fire B's callback")
	}

	regB.interactive.HandleClick(1, 0)
	if !clickedB {
		t.Error("click through registries B should fire B's callback")
	}
}

// TestRegistriesInputStateIsolation verifies that two inputs with the SAME
// explicit ID, owned by different runtimes, keep separate persistent state.
func TestRegistriesInputStateIsolation(t *testing.T) {
	regA := newRegistries()
	regB := newRegistries()

	bindingA := "alpha"
	bindingB := "beta"

	viewA := buildViews(regA, func() View { return InputField(&bindingA).ID("shared-id") })
	viewB := buildViews(regB, func() View { return InputField(&bindingB).ID("shared-id") })

	renderWithRegistries(t, viewA, regA)
	renderWithRegistries(t, viewB, regB)

	stateA, okA := regA.inputs.lookup("shared-id")
	stateB, okB := regB.inputs.lookup("shared-id")
	if !okA || !okB {
		t.Fatalf("expected both registries to hold state for shared-id (A: %v, B: %v)", okA, okB)
	}
	if stateA == stateB {
		t.Error("the two runtimes must not share input state for the same ID")
	}
	if got := stateA.input.Value(); got != "alpha" {
		t.Errorf("runtime A input state = %q, want %q", got, "alpha")
	}
	if got := stateB.input.Value(); got != "beta" {
		t.Errorf("runtime B input state = %q, want %q", got, "beta")
	}
}

// TestBuildViewsCapture verifies that stateful view constructors capture the
// registries of the runtime building the view tree, and fall back to the
// process default outside buildViews.
func TestBuildViewsCapture(t *testing.T) {
	reg := newRegistries()
	binding := ""

	view := buildViews(reg, func() View { return InputField(&binding) })
	field, ok := view.(*inputFieldView)
	if !ok {
		t.Fatalf("expected *inputFieldView, got %T", view)
	}
	if field.reg != reg {
		t.Error("constructor inside buildViews should capture the runtime's registries")
	}

	outside := InputField(&binding)
	if outside.reg != defaultRegistries {
		t.Error("constructor outside buildViews should capture defaultRegistries")
	}
}
