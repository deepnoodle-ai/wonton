package tui

import (
	"reflect"
	"time"
)

// Cmd is a function that performs async work and returns an event.
// Commands are executed in separate goroutines by the Runtime,
// and their results are sent back to the event loop as events.
//
// This pattern enables non-blocking async operations (HTTP requests,
// timers, file I/O) while maintaining the single-threaded event loop
// guarantee for application state.
//
// The built-in commands that only change runtime state (Quit, Focus,
// FocusNext and FocusPrev) are the exception: the runtime applies them as
// soon as HandleEvent returns, in the order returned, before handling the
// next event.
//
// Example:
//
//	func fetchData() Cmd {
//	    return func() Event {
//	        data, err := http.Get("https://api.example.com/data")
//	        if err != nil {
//	            return ErrorEvent{Err: err, Cause: "fetch"}
//	        }
//	        return DataReceivedEvent{Data: data}
//	    }
//	}
//
//	func (a *App) HandleEvent(event Event) []Cmd {
//	    if _, ok := event.(KeyEvent); ok {
//	        return []Cmd{fetchData()}
//	    }
//	    return nil
//	}
type Cmd func() Event

// Quit returns a command that triggers application shutdown.
// The application will clean up and exit gracefully. Quit takes effect as
// soon as HandleEvent returns: no further events are handled, and commands
// listed after it are not run. Quit does not wait for commands that are
// still running, and Run does not either. To do work once the UI has
// exited, such as playing a file the user picked, do it after Run returns.
//
// Example:
//
//	func (a *App) HandleEvent(event Event) []Cmd {
//	    if key, ok := event.(KeyEvent); ok && key.Rune == 'q' {
//	        return []Cmd{Quit()}
//	    }
//	    return nil
//	}
func Quit() Cmd {
	return quitCmd{}.event
}

type quitCmd struct{}

func (quitCmd) event() Event { return QuitEvent{Time: time.Now()} }

// runtimeCmds holds the code pointers of the built-in commands that only
// change runtime state. They need no I/O, so the runtime runs them inline
// instead of on a goroutine; otherwise an event already queued, such as the
// next key press, would be handled before they take effect.
var runtimeCmds = map[uintptr]bool{
	cmdPointer(Quit()):      true,
	cmdPointer(Focus("")):   true,
	cmdPointer(FocusNext()): true,
	cmdPointer(FocusPrev()): true,
}

func cmdPointer(c Cmd) uintptr { return reflect.ValueOf(c).Pointer() }

// isRuntimeCmd reports whether c was made by Quit, Focus, FocusNext or
// FocusPrev.
func isRuntimeCmd(c Cmd) bool { return c != nil && runtimeCmds[cmdPointer(c)] }

// After returns a command that waits for the specified duration and then
// delivers a TimerEvent carrying the given tag. Handle the TimerEvent in
// HandleEvent — which runs on the single-threaded event loop — to mutate
// application state safely:
//
//	// Schedule: clear a flash message after 2 seconds
//	return []Cmd{After(2*time.Second, "clear-flash")}
//
//	// Handle:
//	if timer, ok := event.(TimerEvent); ok && timer.Tag == "clear-flash" {
//	    a.flash = ""
//	}
//
// Note: earlier versions of After took a func() that ran on the command
// goroutine. That invited data races on application state, which the event
// loop otherwise makes impossible — mutate state in HandleEvent instead.
func After(duration time.Duration, tag string) Cmd {
	return func() Event {
		time.Sleep(duration)
		return TimerEvent{Time: time.Now(), Tag: tag}
	}
}

// Batch combines multiple commands into a single list.
// This is a convenience function for returning multiple commands from HandleEvent.
//
// Example:
//
//	return Batch(
//	    fetchUserData(),
//	    fetchSettings(),
//	    startTimer(),
//	)
func Batch(cmds ...Cmd) []Cmd {
	return cmds
}

// None returns an empty command list.
// This is a convenience function for when HandleEvent needs to return
// an empty slice explicitly (useful for readability).
func None() []Cmd {
	return nil
}

// Sequence returns a command that executes multiple commands in order,
// collecting their results into a BatchEvent.
//
// Unlike Batch (which runs commands in parallel), Sequence runs them
// sequentially in a single goroutine. Use this when commands depend
// on each other or when parallel execution would cause issues.
//
// Example:
//
//	return []Cmd{Sequence(
//	    authenticate(),
//	    fetchData(),
//	    processResults(),
//	)}
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
