package tui

import "time"

// Cmd is a function that performs async work and returns an event.
// Commands are executed in separate goroutines by the Runtime,
// and their results are sent back to the event loop as events.
//
// This pattern enables non-blocking async operations (HTTP requests,
// timers, file I/O) while maintaining the single-threaded event loop
// guarantee for application state.
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
// The application will clean up and exit gracefully.
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
	return func() Event {
		return QuitEvent{Time: time.Now()}
	}
}

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
