package tui

import "time"

// Cmd is a function that performs async work and returns an event.
// Commands are executed in separate goroutines by the Runtime,
// and their results are sent back to the event loop as events.
type Cmd func() Event

// Quit returns a command that triggers application shutdown.
func Quit() Cmd {
	return func() Event {
		return QuitEvent{Time: time.Now()}
	}
}

// Tick returns a command that waits for the specified duration
// and then returns a TickEvent.
func Tick(duration time.Duration) Cmd {
	return func() Event {
		time.Sleep(duration)
		return TickEvent{Time: time.Now()}
	}
}

// After returns a command that waits for the specified duration,
// executes the provided function, and returns a TickEvent.
// If fn is nil, it simply waits and returns a TickEvent.
func After(duration time.Duration, fn func()) Cmd {
	return func() Event {
		time.Sleep(duration)
		if fn != nil {
			fn()
		}
		return TickEvent{Time: time.Now()}
	}
}

// Batch combines multiple commands into a single list.
// This is a convenience function for returning multiple commands from HandleEvent.
func Batch(cmds ...Cmd) []Cmd {
	return cmds
}

// None returns an empty command list.
// This is a convenience function for when HandleEvent needs to return
// an empty slice explicitly.
func None() []Cmd {
	return nil
}

// Sequence returns a command that executes multiple commands in order,
// collecting their results into a BatchEvent.
func Sequence(cmds ...Cmd) Cmd {
	return func() Event {
		events := make([]Event, len(cmds))
		for i, cmd := range cmds {
			events[i] = cmd()
		}
		return BatchEvent{
			Time:   time.Now(),
			Events: events,
		}
	}
}
