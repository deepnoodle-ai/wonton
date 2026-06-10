package tui

import (
	"sync"
	"sync/atomic"
)

// registries bundles the state stores that views register into during render
// (buttons, clickable regions, input state, prompt choices, text areas).
// Each Runtime and InlineApp owns one instance, so multiple applications in
// a single process cannot observe or mutate each other's interactive state.
//
// Views obtain the active registries two ways:
//   - During render, from the RenderContext (ctx.registries()).
//   - At construction time via capturedRegistries(), so size() can consult
//     persistent state before the first render attaches a context.
type registries struct {
	buttons       *buttonRegistryImpl
	interactive   *interactiveRegistryImpl
	inputs        *inputRegistryImpl
	promptChoices *promptChoiceRegistryImpl
	textAreas     *textAreaRegistryImpl
}

func newRegistries() *registries {
	return &registries{
		buttons:       &buttonRegistryImpl{buttons: make(map[string]*buttonState)},
		interactive:   &interactiveRegistryImpl{regions: make([]interactiveRegion, 0)},
		inputs:        &inputRegistryImpl{inputs: make(map[string]*inputState)},
		promptChoices: &promptChoiceRegistryImpl{inputs: make(map[string]*TextInput)},
		textAreas: &textAreaRegistryImpl{
			states: make(map[string]*textAreaState),
			active: make(map[string]bool),
		},
	}
}

// clearForRender resets the per-frame registries. Called before each render
// pass; views re-register as they draw.
func (r *registries) clearForRender() {
	r.buttons.Clear()
	r.interactive.Clear()
	r.inputs.Clear()
	r.textAreas.Clear()
}

// defaultRegistries serves views rendered outside a Runtime or InlineApp:
// direct render calls in tests, and standalone Print/Sprint of views that
// were not built inside an application's View method.
var defaultRegistries = newRegistries()

// buildingReg holds the registries of the runtime currently constructing its
// view tree (inside app.View()/LiveView()). Stateful view constructors
// (InputField, Input, TextArea, PromptChoice) capture it so their size()
// method can consult persistent state before the first render pass.
var (
	viewBuildMu sync.Mutex
	buildingReg atomic.Pointer[registries]
)

// buildViews invokes fn (typically app.View or app.LiveView) with reg set as
// the capture target for stateful view constructors. Serialized across
// runtimes so concurrent applications can't capture each other's registries.
func buildViews(reg *registries, fn func() View) View {
	viewBuildMu.Lock()
	defer viewBuildMu.Unlock()
	buildingReg.Store(reg)
	defer buildingReg.Store(nil)
	return fn()
}

// capturedRegistries returns the registries of the runtime currently building
// its view tree, or defaultRegistries when views are constructed outside an
// application's View method.
func capturedRegistries() *registries {
	if reg := buildingReg.Load(); reg != nil {
		return reg
	}
	return defaultRegistries
}
