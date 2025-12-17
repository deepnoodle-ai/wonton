//go:build unix

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

// Session represents an interactive PTY (pseudo-terminal) session.
//
// Session manages a command running in a PTY, handles terminal I/O,
// and optionally records the session to an asciinema v2 format file.
// It's designed for creating interactive terminal sessions that feel
// like a real terminal (with proper handling of colors, cursor movement, etc.).
//
// The session handles:
//   - PTY creation and management
//   - Terminal size synchronization (including SIGWINCH)
//   - Raw mode terminal setup
//   - Optional recording with the Recorder
//
// Use Start() to begin an interactive session, or Record() to start
// and record simultaneously.
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
	input   io.Reader
	output  io.Writer
}

// SessionOptions configures a new PTY session.
type SessionOptions struct {
	Command []string  // Command to run (default: user's shell from $SHELL)
	Dir     string    // Working directory (default: current directory)
	Env     []string  // Additional environment variables (added to inherited environment)
	Input   io.Reader // Input source (default: os.Stdin)
	Output  io.Writer // Output destination (default: os.Stdout)
}

// NewSession creates a new PTY session with the given options.
//
// The session is not started until Start() or Record() is called.
// If no command is specified, the user's shell ($SHELL or /bin/sh) is used.
//
// Example:
//
//	session, err := NewSession(SessionOptions{
//	    Command: []string{"bash", "-i"},
//	    Dir: "/tmp",
//	    Env: []string{"TERM=xterm-256color"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer session.Close()
func NewSession(opts SessionOptions) (*Session, error) {
	s := &Session{
		command: opts.Command,
		dir:     opts.Dir,
		env:     opts.Env,
		input:   opts.Input,
		output:  opts.Output,
		done:    make(chan struct{}),
	}

	return s, nil
}

// Record starts the PTY session and records it to the specified file.
//
// This is a convenience method that creates a Recorder, attaches it to the
// session, and calls Start(). The terminal size is detected automatically
// (or defaults to 80x24 if detection fails).
//
// The recording file is created immediately with the header written.
// Call Wait() to block until the session ends, then Close() to finalize.
//
// Example:
//
//	session, _ := NewSession(SessionOptions{
//	    Command: []string{"bash", "-c", "echo 'Hello, World!'"},
//	})
//	err := session.Record("demo.cast", RecordingOptions{
//	    Compress: true,
//	    Title: "Hello Demo",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	session.Wait()
//	session.Close()
func (s *Session) Record(filename string, opts RecordingOptions) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("session already started")
	}
	// Default input if not set
	input := s.input
	if input == nil {
		input = os.Stdin
	}
	s.mu.Unlock()

	// Get terminal size for recording
	var width, height int
	var err error
	fd := -1

	if f, ok := input.(*os.File); ok {
		fd = int(f.Fd())
	}

	if fd != -1 {
		width, height, err = term.GetSize(fd)
	} else {
		err = fmt.Errorf("not a terminal")
	}

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

	if err := s.Start(); err != nil {
		// Clean up recorder on start failure
		s.mu.Lock()
		s.recorder = nil
		s.mu.Unlock()
		recorder.Close()
		return err
	}

	return nil
}

// Start begins the PTY session without recording.
//
// This method:
//   - Creates a PTY and starts the command
//   - Sets the terminal to raw mode (if stdin is a terminal)
//   - Starts goroutines to handle I/O between stdin/stdout and the PTY
//   - Sets up terminal resize handling (SIGWINCH)
//
// The session runs in the background. Use Wait() to block until it completes.
//
// Example:
//
//	session, _ := NewSession(SessionOptions{
//	    Command: []string{"bash"},
//	})
//	err := session.Start()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Session is now interactive
//	session.Wait()
func (s *Session) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("session already started")
	}
	s.mu.Unlock()

	// Defaults
	if s.input == nil {
		s.input = os.Stdin
	}
	if s.output == nil {
		s.output = os.Stdout
	}

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

	// Set raw mode on input (if it's a terminal)
	if f, ok := s.input.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		oldState, err := term.MakeRaw(int(f.Fd()))
		if err != nil {
			s.pty.Close()
			s.pty = nil
			return fmt.Errorf("failed to enable raw mode: %w", err)
		}
		s.oldState = oldState
	}

	// Mark as started only after all setup succeeds
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	// Sync initial terminal size
	s.syncSize()

	// Handle SIGWINCH for resize
	go s.handleResize()

	// Start I/O copy goroutines
	go s.copyInput()
	go s.copyOutput()

	return nil
}

// Wait blocks until the session ends and returns any error.
//
// This waits for the command to exit. The exit code can be retrieved
// with ExitCode() after Wait returns.
func (s *Session) Wait() error {
	<-s.done

	s.mu.Lock()
	err := s.err
	s.mu.Unlock()

	return err
}

// ExitCode returns the exit code of the command.
//
// This is only valid after Wait() returns. Returns 0 before the session completes.
func (s *Session) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCode
}

// Close terminates the session and cleans up resources.
//
// This method:
//   - Restores the terminal to its original state
//   - Closes and finalizes any recording
//   - Closes the PTY
//
// It's safe to call Close multiple times or before the session completes.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Restore terminal state
	if s.oldState != nil {
		if f, ok := s.input.(*os.File); ok {
			term.Restore(int(f.Fd()), s.oldState)
		}
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

// Resize manually sets the terminal size.
//
// This is useful for programmatically resizing the terminal or when
// automatic resize detection isn't available (non-TTY scenarios).
// The recorder is also updated if recording is active.
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

// PauseRecording temporarily pauses recording.
//
// Terminal I/O continues normally, but events are not recorded while paused.
// Has no effect if the session is not being recorded.
func (s *Session) PauseRecording() {
	s.mu.Lock()
	recorder := s.recorder
	s.mu.Unlock()

	if recorder != nil {
		recorder.Pause()
	}
}

// ResumeRecording resumes a paused recording.
//
// Recording continues from where it was paused. Has no effect if
// the session is not being recorded or is not paused.
func (s *Session) ResumeRecording() {
	s.mu.Lock()
	recorder := s.recorder
	s.mu.Unlock()

	if recorder != nil {
		recorder.Resume()
	}
}

// IsRecording returns true if the session is being recorded.
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
	fd := -1
	if f, ok := s.input.(*os.File); ok {
		fd = int(f.Fd())
	}

	if fd == -1 || !term.IsTerminal(fd) {
		return nil
	}

	width, height, err := term.GetSize(fd)
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

// copyInput reads from input and writes to the PTY
func (s *Session) copyInput() {
	buf := make([]byte, 4096)
	for {
		n, err := s.input.Read(buf)
		if n > 0 {
			s.mu.Lock()
			ptmx := s.pty
			recorder := s.recorder
			s.mu.Unlock()

			if recorder != nil {
				recorder.RecordInput(string(buf[:n]))
			}

			if ptmx != nil {
				if _, werr := ptmx.Write(buf[:n]); werr != nil {
					break
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				s.mu.Lock()
				ptmx := s.pty
				s.mu.Unlock()
				if ptmx != nil {
					// Send EOT (Ctrl+D) to signal EOF to the PTY slave
					ptmx.Write([]byte{4})
				}
			}
			break
		}
	}
}

// copyOutput reads from PTY and writes to output, recording if enabled
func (s *Session) copyOutput() {
	defer func() {
		// Clean up when output ends (command exited)
		s.mu.Lock()
		if s.oldState != nil {
			if f, ok := s.input.(*os.File); ok {
				term.Restore(int(f.Fd()), s.oldState)
			}
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

		// Write to output
		s.output.Write(data)

		// Record if enabled
		if recorder != nil {
			recorder.RecordOutput(string(data))
		}
	}
}
