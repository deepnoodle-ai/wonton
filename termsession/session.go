package termsession

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Session represents an interactive PTY session that can be recorded
type Session struct {
	cmd      *exec.Cmd
	pty      *os.File
	oldState *term.State
	recorder *Recorder

	mu       sync.Mutex
	done     chan struct{}
	err      error
	started  bool
	exitCode int

	// Configuration
	command []string
	env     []string
	dir     string
}

// SessionOptions configures a new session
type SessionOptions struct {
	Command []string // Command to run (default: user's shell from $SHELL)
	Dir     string   // Working directory (default: current directory)
	Env     []string // Additional environment variables
}

// NewSession creates a new PTY session with the given options.
// The session is not started until Start() or Record() is called.
func NewSession(opts SessionOptions) (*Session, error) {
	s := &Session{
		command: opts.Command,
		dir:     opts.Dir,
		env:     opts.Env,
		done:    make(chan struct{}),
	}

	return s, nil
}

// Record starts the session and records it to the specified file.
// This is a convenience method that combines Start() with recording setup.
func (s *Session) Record(filename string, opts RecordingOptions) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("session already started")
	}
	s.mu.Unlock()

	// Get terminal size for recording
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		// Default to 80x24 if we can't get size
		width, height = 80, 24
	}

	// Create recorder
	recorder, err := NewRecorder(filename, width, height, opts)
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}

	s.mu.Lock()
	s.recorder = recorder
	s.mu.Unlock()

	return s.Start()
}

// Start begins the PTY session without recording
func (s *Session) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("session already started")
	}
	s.started = true
	s.mu.Unlock()

	// Get user's shell if no command specified
	command := s.command
	if len(command) == 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		command = []string{shell}
	}

	s.cmd = exec.Command(command[0], command[1:]...)
	s.cmd.Env = append(os.Environ(), s.env...)
	if s.dir != "" {
		s.cmd.Dir = s.dir
	}

	// Start command in PTY
	ptmx, err := pty.Start(s.cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}
	s.pty = ptmx

	// Set raw mode on stdin (if it's a terminal)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			s.pty.Close()
			return fmt.Errorf("failed to enable raw mode: %w", err)
		}
		s.oldState = oldState
	}

	// Sync initial terminal size
	s.syncSize()

	// Handle SIGWINCH for resize
	go s.handleResize()

	// Start I/O copy goroutines
	go s.copyInput()
	go s.copyOutput()

	return nil
}

// Wait blocks until the session ends and returns any error
func (s *Session) Wait() error {
	<-s.done

	s.mu.Lock()
	err := s.err
	s.mu.Unlock()

	return err
}

// ExitCode returns the exit code of the command (only valid after Wait returns)
func (s *Session) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCode
}

// Close terminates the session and cleans up resources
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Restore terminal state
	if s.oldState != nil {
		term.Restore(int(os.Stdin.Fd()), s.oldState)
		s.oldState = nil
	}

	// Close recorder
	if s.recorder != nil {
		s.recorder.Close()
		s.recorder = nil
	}

	// Close PTY
	if s.pty != nil {
		s.pty.Close()
		s.pty = nil
	}

	return nil
}

// Resize manually sets the terminal size (useful for non-TTY scenarios)
func (s *Session) Resize(width, height int) error {
	s.mu.Lock()
	ptmx := s.pty
	recorder := s.recorder
	s.mu.Unlock()

	if ptmx == nil {
		return fmt.Errorf("session not started")
	}

	if err := pty.Setsize(ptmx, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	}); err != nil {
		return err
	}

	if recorder != nil {
		recorder.UpdateSize(width, height)
	}

	return nil
}

// PauseRecording temporarily pauses recording
func (s *Session) PauseRecording() {
	s.mu.Lock()
	recorder := s.recorder
	s.mu.Unlock()

	if recorder != nil {
		recorder.Pause()
	}
}

// ResumeRecording resumes a paused recording
func (s *Session) ResumeRecording() {
	s.mu.Lock()
	recorder := s.recorder
	s.mu.Unlock()

	if recorder != nil {
		recorder.Resume()
	}
}

// IsRecording returns true if the session is being recorded
func (s *Session) IsRecording() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recorder != nil
}

// handleResize listens for SIGWINCH and propagates size changes
func (s *Session) handleResize() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	defer signal.Stop(ch)

	for {
		select {
		case <-ch:
			s.syncSize()
		case <-s.done:
			return
		}
	}
}

// syncSize synchronizes PTY size with the controlling terminal
func (s *Session) syncSize() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}

	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	s.mu.Lock()
	ptmx := s.pty
	recorder := s.recorder
	s.mu.Unlock()

	if ptmx != nil {
		if err := pty.Setsize(ptmx, &pty.Winsize{
			Rows: uint16(height),
			Cols: uint16(width),
		}); err != nil {
			return err
		}
	}

	if recorder != nil {
		recorder.UpdateSize(width, height)
	}

	return nil
}

// copyInput reads from stdin and writes to the PTY
func (s *Session) copyInput() {
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}

		s.mu.Lock()
		ptmx := s.pty
		s.mu.Unlock()

		if ptmx == nil {
			break
		}

		if _, err := ptmx.Write(buf[:n]); err != nil {
			break
		}
	}
}

// copyOutput reads from PTY and writes to stdout, recording if enabled
func (s *Session) copyOutput() {
	defer func() {
		// Clean up when output ends (command exited)
		s.mu.Lock()
		if s.oldState != nil {
			term.Restore(int(os.Stdin.Fd()), s.oldState)
			s.oldState = nil
		}
		recorder := s.recorder
		s.mu.Unlock()

		if recorder != nil {
			recorder.Flush()
			recorder.Close()
		}

		// Wait for the process to exit and capture exit code
		if s.cmd != nil {
			err := s.cmd.Wait()
			s.mu.Lock()
			s.err = err
			if exitErr, ok := err.(*exec.ExitError); ok {
				s.exitCode = exitErr.ExitCode()
			} else if err == nil {
				s.exitCode = 0
			}
			s.mu.Unlock()
		}

		close(s.done)
	}()

	buf := make([]byte, 4096)
	for {
		s.mu.Lock()
		ptmx := s.pty
		recorder := s.recorder
		s.mu.Unlock()

		if ptmx == nil {
			break
		}

		n, err := ptmx.Read(buf)
		if err != nil {
			if err != io.EOF {
				s.mu.Lock()
				s.err = err
				s.mu.Unlock()
			}
			break
		}

		data := buf[:n]

		// Write to stdout
		os.Stdout.Write(data)

		// Record if enabled
		if recorder != nil {
			recorder.RecordOutput(string(data))
		}
	}
}
