package tui

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/deepnoodle-ai/wonton/internal/terminalinput"
	"github.com/deepnoodle-ai/wonton/internal/terminalstate"
	"golang.org/x/term"
)

var (
	ErrHandoffUnsupported = errors.New("tui: terminal handoff is unsupported for this input or output")
	ErrHandoffNotRunning  = errors.New("tui: terminal handoff requires a running event loop")
	ErrHandoffReentrant   = errors.New("tui: a terminal transition is already in progress")
)

func qualifiedTerminalFiles(input io.Reader, output io.Writer) bool {
	if input != os.Stdin || output != os.Stdout || !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	in, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	out, err := os.Stdout.Stat()
	return err == nil && os.SameFile(in, out)
}

func (r *Runtime) initializeManagedInput() error {
	access, ok := terminalstate.Inspect(r.terminal)
	if !ok || r.inputSource != nil || access.FD != int(os.Stdout.Fd()) || !qualifiedTerminalFiles(os.Stdin, access.Output) {
		return nil
	}
	owner, err := terminalinput.New(os.Stdin)
	if errors.Is(err, terminalinput.ErrUnsupported) {
		return nil
	}
	if err != nil {
		return err
	}
	r.managedInput = newManagedInput(owner, r.events, r.done, r.pasteTabWidth, r.backslashEnter, r.processMouseEvent, func(event Event) bool {
		if keys := r.suspendChannel(); keys != nil {
			select {
			case keys <- event:
			default:
			}
			return true
		}
		return false
	})
	r.managedInput.capturePanic = r.capturePanic
	return nil
}
func (r *InlineApp) initializeManagedInput() error {
	if !qualifiedTerminalFiles(r.config.Input, r.output) {
		return nil
	}
	owner, err := terminalinput.New(os.Stdin)
	if errors.Is(err, terminalinput.ErrUnsupported) {
		return nil
	}
	if err != nil {
		return err
	}
	r.managedInput = newManagedInput(owner, r.events, r.done, r.config.PasteTabWidth, r.config.BackslashEnter, r.processMouseEvent, nil)
	r.managedInput.capturePanic = r.capturePanic
	return nil
}

// Handoff gives a child exclusive terminal input and output until fn returns.
// Call synchronously from HandleEvent. The callback must wait for its child and
// must not call runner printing, rendering, Suspend, or Handoff. Darwin/Linux
// native stdin/stdout on the same terminal are supported; custom sources and
// outputs are not. A nil callback does nothing. operationErr reports release
// or callback failure; restoreErr is fatal and also becomes Run's error.
func (r *Runtime) Handoff(fn func() error) (operationErr, restoreErr error) {
	if fn == nil {
		return nil, nil
	}
	r.mu.Lock()
	running := r.running
	r.mu.Unlock()
	if !running || r.stopping() {
		return ErrHandoffNotRunning, nil
	}
	r.suspendMu.Lock()
	if r.handoffActive || r.suspendKeys != nil {
		r.suspendMu.Unlock()
		return ErrHandoffReentrant, nil
	}
	r.handoffActive = true
	r.suspendMu.Unlock()
	defer func() { r.suspendMu.Lock(); r.handoffActive = false; r.suspendMu.Unlock() }()
	access, ok := terminalstate.Inspect(r.terminal)
	if !ok || r.managedInput == nil || access.FD != int(os.Stdout.Fd()) || !qualifiedTerminalFiles(os.Stdin, access.Output) {
		return ErrHandoffUnsupported, nil
	}
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	return runTerminalHandoff(fn, r.managedInput, access.Snapshot, r.stopping, func() error {
		if err := r.terminal.RefreshSize(); err != nil {
			return err
		}
		r.terminal.Invalidate()
		return errors.Join(r.renderChecked(), terminalstate.Flush(access.Output))
	}, func(err error) { r.mu.Lock(); r.handoffErr = errors.Join(r.handoffErr, err); r.mu.Unlock(); r.Stop() })
}

// Handoff has the same ownership and error contract as Runtime.Handoff.
// Call it synchronously from HandleEvent, and treat restoreErr as fatal.
func (r *InlineApp) Handoff(fn func() error) (operationErr, restoreErr error) {
	if fn == nil {
		return nil, nil
	}
	r.mu.Lock()
	if !r.running || r.stopping() {
		r.mu.Unlock()
		return ErrHandoffNotRunning, nil
	}
	if r.handoffActive {
		r.mu.Unlock()
		return ErrHandoffReentrant, nil
	}
	r.handoffActive = true
	r.mu.Unlock()
	defer func() { r.mu.Lock(); r.handoffActive = false; r.mu.Unlock() }()
	if r.managedInput == nil || !qualifiedTerminalFiles(r.config.Input, r.output) {
		return ErrHandoffUnsupported, nil
	}
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	return runTerminalHandoff(fn, r.managedInput, r.handoffSnapshot, r.stopping, func() error {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return err
		}
		r.config.Width = width
		r.live.SetWidth(width)
		r.live.SetMaxHeight(height)
		return errors.Join(r.renderChecked(), terminalstate.Flush(r.output))
	}, func(err error) { r.mu.Lock(); r.handoffErr = errors.Join(r.handoffErr, err); r.mu.Unlock(); r.Stop() })
}

func runTerminalHandoff(fn func() error, input *managedInput, snapshot func() (terminalstate.Transition, error), stopped func() bool, repaint func() error, failed func(error)) (operationErr, restoreErr error) {
	transition, err := snapshot()
	if err != nil {
		return fmt.Errorf("tui: snapshot terminal: %w", err), nil
	}
	if err = input.pause(); err != nil {
		operationErr = fmt.Errorf("tui: release input: %w", err)
		restoreErr = errors.Join(fmt.Errorf("tui: input ownership could not be restored: %w", err), transition.Restore(true))
		failed(restoreErr)
		return
	}
	defer func() {
		restoreErr = errors.Join(restoreErr, input.owner.DiscardResidual())
		stopping := stopped() || restoreErr != nil
		restoreErr = errors.Join(restoreErr, transition.Restore(stopping))
		if !stopping && restoreErr == nil {
			if stopped() {
				restoreErr = errors.Join(restoreErr, transition.Restore(true))
			} else {
				restoreErr = errors.Join(restoreErr, repaint())
				if restoreErr == nil && !stopped() {
					restoreErr = input.resume()
				} else {
					restoreErr = errors.Join(restoreErr, transition.Restore(true))
				}
			}
		}
		if restoreErr != nil {
			restoreErr = fmt.Errorf("tui: restore terminal: %w", errors.Join(restoreErr, transition.Restore(true)))
			failed(restoreErr)
		}
	}()
	if err = transition.Release(); err != nil {
		return fmt.Errorf("tui: release terminal: %w", err), nil
	}
	if stopped() {
		return nil, nil
	}
	return fn(), nil
}

func (r *InlineApp) handoffSnapshot() (terminalstate.Transition, error) {
	attributes, err := term.GetState(r.stdinFd)
	if err != nil {
		return terminalstate.Transition{}, err
	}
	hidden := r.live.hiddenCursor
	paste, kitty, mouse := r.pasteEnabled, r.kittyEnabled, r.mouseEnabled
	plain := func() error {
		clearErr := r.live.clearChecked()
		sequence := "\x1b[?25h"
		if mouse {
			sequence += "\x1b[?1006l\x1b[?1000l"
		}
		if kitty {
			sequence += "\x1b[<u"
		}
		if paste {
			sequence += "\x1b[?2004l"
		}
		outputErr := checkedHandoffWrite(r.output, sequence)
		if outputErr == nil {
			r.live.hiddenCursor = false
			r.pasteEnabled, r.kittyEnabled, r.mouseEnabled = false, false, false
		}
		return errors.Join(clearErr, outputErr, term.Restore(r.stdinFd, r.oldState))
	}
	return terminalstate.Transition{Release: plain, Restore: func(stopping bool) error {
		if stopping {
			return plain()
		}
		attributeErr := term.Restore(r.stdinFd, attributes)
		sequence := ""
		if paste {
			sequence += "\x1b[?2004h"
		}
		if kitty {
			sequence += "\x1b[>1u"
		}
		if mouse {
			sequence += "\x1b[?1000h\x1b[?1006h"
		}
		if hidden {
			sequence += "\x1b[?25l"
		} else {
			sequence += "\x1b[?25h"
		}
		outputErr := checkedHandoffWrite(r.output, sequence)
		if outputErr == nil {
			r.live.hiddenCursor = hidden
			r.pasteEnabled, r.kittyEnabled, r.mouseEnabled = paste, kitty, mouse
		}
		return errors.Join(attributeErr, outputErr)
	}}, nil
}

func checkedHandoffWrite(out io.Writer, sequence string) error {
	return terminalstate.Write(out, sequence)
}
