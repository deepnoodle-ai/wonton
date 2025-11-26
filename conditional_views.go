package gooey

// If returns the view if condition is true, otherwise returns Empty.
// Use this for conditional rendering in declarative UI.
//
// Example:
//
//	VStack(
//	    Text("Always shown"),
//	    If(app.showWarning, Text("Warning!").Fg(ColorYellow)),
//	)
func If(condition bool, view View) View {
	if condition {
		return view
	}
	return Empty()
}

// IfElse returns thenView if condition is true, otherwise returns elseView.
// Use this for conditional rendering with an alternative.
//
// Example:
//
//	IfElse(app.isLoggedIn,
//	    Text("Welcome back!"),
//	    Text("Please log in"),
//	)
func IfElse(condition bool, thenView, elseView View) View {
	if condition {
		return thenView
	}
	return elseView
}

// CaseView represents a case in a Switch statement.
type CaseView[T comparable] struct {
	value     T
	view      View
	isDefault bool
}

// Case creates a case for use with Switch.
func Case[T comparable](value T, view View) CaseView[T] {
	return CaseView[T]{value: value, view: view}
}

// Default creates a default case for use with Switch.
func Default[T comparable](view View) CaseView[T] {
	return CaseView[T]{view: view, isDefault: true}
}

// Switch returns the view associated with the matching case value.
// If no case matches and no Default is provided, returns Empty.
//
// Example:
//
//	Switch(app.status,
//	    Case("loading", Text("Loading...").Fg(ColorYellow)),
//	    Case("error", Text("Error!").Fg(ColorRed)),
//	    Case("ready", Text("Ready").Fg(ColorGreen)),
//	    Default[string](Text("Unknown")),
//	)
func Switch[T comparable](value T, cases ...CaseView[T]) View {
	var defaultView View
	for _, c := range cases {
		if c.isDefault {
			defaultView = c.view
			continue
		}
		if c.value == value {
			return c.view
		}
	}
	if defaultView != nil {
		return defaultView
	}
	return Empty()
}
