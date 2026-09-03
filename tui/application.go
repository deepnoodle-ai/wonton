package tui

// Application is the main interface for declarative UI applications.
// Implementations provide a View that describes the current UI state.
//
// The View method is called after every event to render the UI. The framework
// efficiently diffs the view tree and only updates changed terminal regions.
//
// Thread Safety: View and HandleEvent (if implemented) are NEVER called concurrently.
// The Runtime ensures these methods run sequentially in a single goroutine,
// eliminating the need for locks in application code.
//
// Example:
//
//	type CounterApp struct {
//	    count int
//	}
//
//	func (a *CounterApp) View() tui.View {
//	    return tui.Stack(
//	        tui.Text("Count: %d", a.count),
//	        tui.Button("Increment", func() { a.count++ }),
//	    )
//	}
type Application interface {
	// View returns the declarative view tree representing the current UI.
	// This is called automatically after each event is processed.
	View() View
}

// EventHandler is an optional interface that applications can implement
// to handle events and trigger async operations.
//
// Applications implementing both Application and EventHandler get full control:
// HandleEvent processes the event (potentially mutating state and returning commands),
// then View is called to render the updated UI.
//
// Example:
//
//	func (a *App) HandleEvent(event Event) []Cmd {
//	    switch e := event.(type) {
//	    case KeyEvent:
//	        if e.Rune == 'q' {
//	            return []Cmd{Quit()}
//	        }
//	        if e.Rune == 'r' {
//	            a.loading = true
//	            return []Cmd{a.fetchData()}
//	        }
//	    case DataEvent:
//	        a.data = e.Data
//	        a.loading = false
//	    }
//	    return nil
//	}
type EventHandler interface {
	// HandleEvent processes an event and optionally returns commands for async execution.
	// This is called in a single-threaded event loop, so state can be mutated freely
	// without locks. Return commands for async operations (HTTP requests, timers, etc.).
	//
	// Key routing: KeyEvents are offered to the focused element (InputField,
	// Table, PromptChoice, etc.) first. If the focused element consumes the
	// key, it is NOT delivered to HandleEvent — so a plain-letter shortcut
	// like 'q' won't fire while the user is typing into an input. Ctrl+C is
	// the exception: it is always delivered, making it the reliable choice
	// for a global quit shortcut in apps with focusable inputs.
	//
	// Widgets only consume keys they act on. Keys an input doesn't use —
	// Up/Down when the cursor can't move (and no History is configured),
	// Tab when no completion or focus target wants it — fall through to
	// HandleEvent. To claim specific keys while an input is focused, use
	// the input's OnKey hook rather than handling them here.
	HandleEvent(event Event) []Cmd
}

// Initializable is an optional interface that applications can implement
// to perform initialization before the event loop starts.
type Initializable interface {
	Init() error
}

// Destroyable is an optional interface that applications can implement
// to perform cleanup when the runtime stops.
type Destroyable interface {
	Destroy()
}

// RuntimeAware is an optional interface an application can implement to be
// handed the Runtime driving it, before Init runs.
//
// Run builds the Terminal and the Runtime itself, so without this an
// application started that way has no route to either — which puts
// Runtime.Suspend and writing an escape sequence to the terminal out of reach
// unless the application forgoes Run and assembles a Runtime by hand.
//
//	func (a *App) SetRuntime(r *tui.Runtime) { a.runtime = r }
type RuntimeAware interface {
	SetRuntime(*Runtime)
}

// InputSource abstracts the source of input events.
type InputSource interface {
	ReadEvent() (Event, error)
	SetPasteTabWidth(int)
}

// ViewProvider is an alias for Application for backward compatibility.
//
// Deprecated: Use Application directly instead.
type ViewProvider = Application
