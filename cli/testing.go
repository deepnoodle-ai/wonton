package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestResult contains the results of a test run.
type TestResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// TestOption configures a test run.
type TestOption func(*testConfig)

type testConfig struct {
	args  []string
	stdin io.Reader
	env   map[string]string
}

// TestArgs sets the command-line arguments for the test.
func TestArgs(args ...string) TestOption {
	return func(c *testConfig) {
		c.args = args
	}
}

// TestStdin sets the stdin input for the test.
func TestStdin(input string) TestOption {
	return func(c *testConfig) {
		c.stdin = strings.NewReader(input)
	}
}

// TestEnv sets an environment variable for the test.
func TestEnv(key, value string) TestOption {
	return func(c *testConfig) {
		if c.env == nil {
			c.env = make(map[string]string)
		}
		c.env[key] = value
	}
}

// Test runs the application with the given options and returns the result.
// This is useful for testing CLI commands in unit tests.
func (a *App) Test(t *testing.T, opts ...TestOption) *TestResult {
	t.Helper()

	cfg := &testConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Set up I/O capture
	var stdout, stderr bytes.Buffer
	origStdout, origStderr := a.stdout, a.stderr
	origStdin := a.stdin
	a.stdout = &stdout
	a.stderr = &stderr
	if cfg.stdin != nil {
		a.stdin = cfg.stdin
	}

	// Set environment variables
	for key, value := range cfg.env {
		t.Setenv(key, value)
	}

	// Run the command
	err := a.RunArgs(cfg.args)

	// Restore I/O
	a.stdout = origStdout
	a.stderr = origStderr
	a.stdin = origStdin

	return &TestResult{
		ExitCode: GetExitCode(err),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Err:      err,
	}
}

// Contains checks if the stdout contains the given substring.
func (r *TestResult) Contains(s string) bool {
	return strings.Contains(r.Stdout, s)
}

// StderrContains checks if the stderr contains the given substring.
func (r *TestResult) StderrContains(s string) bool {
	return strings.Contains(r.Stderr, s)
}

// Success returns true if the command succeeded (exit code 0).
func (r *TestResult) Success() bool {
	return r.ExitCode == 0
}

// Failed returns true if the command failed (exit code != 0).
func (r *TestResult) Failed() bool {
	return r.ExitCode != 0
}

// TestApp creates a new app configured for testing.
// It disables interactive mode and captures all output.
func TestApp(name string) *App {
	app := New(name)
	app.isInteractive = false
	app.stdin = strings.NewReader("")
	app.stdout = &bytes.Buffer{}
	app.stderr = &bytes.Buffer{}
	return app
}

// CaptureOutput is a helper that captures stdout/stderr during a function call.
func CaptureOutput(fn func()) (stdout, stderr string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	return bufOut.String(), bufErr.String()
}
