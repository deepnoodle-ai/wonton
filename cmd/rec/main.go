// Command rec provides terminal session recording and playback.
//
// Usage:
//
//	rec record [-o output.cast] [command...]
//	rec play [-s speed] <file.cast>
//
// Record a session:
//
//	rec record                         # Record interactive shell
//	rec record -o demo.cast            # Record to specific file
//	rec record bash -c "echo hello"    # Record a specific command
//
// Play back a recording:
//
//	rec play demo.cast                 # Play at normal speed
//	rec play -s 2 demo.cast            # Play at 2x speed
//	rec play -s 0.5 demo.cast          # Play at half speed
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/termsession"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "record", "rec":
		recordCmd(os.Args[2:])
	case "play":
		playCmd(os.Args[2:])
	case "info":
		infoCmd(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func recordCmd(args []string) {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	output := fs.String("o", "session.cast", "Output file")
	title := fs.String("t", "", "Recording title")
	compress := fs.Bool("z", true, "Enable gzip compression")
	redact := fs.Bool("redact", true, "Redact secrets (passwords, tokens, etc.)")
	idleLimit := fs.Float64("idle", 0, "Max idle time between events in seconds (0=unlimited)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: rec record [options] [command...]")
		fmt.Fprintln(os.Stderr, "\nRecord a terminal session to an asciinema v2 file.")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  rec record                         # Record interactive shell")
		fmt.Fprintln(os.Stderr, "  rec record -o demo.cast            # Record to specific file")
		fmt.Fprintln(os.Stderr, "  rec record -t \"My Demo\" bash       # Record bash with title")
		fmt.Fprintln(os.Stderr, "  rec record python script.py        # Record command execution")
	}
	fs.Parse(args)

	command := fs.Args()

	session, err := termsession.NewSession(termsession.SessionOptions{
		Command: command,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := termsession.RecordingOptions{
		Compress:      *compress,
		RedactSecrets: *redact,
		Title:         *title,
		IdleTimeLimit: *idleLimit,
		Env: map[string]string{
			"SHELL": os.Getenv("SHELL"),
			"TERM":  os.Getenv("TERM"),
		},
	}

	fmt.Fprintf(os.Stderr, "Recording to %s (press Ctrl+D or type 'exit' to stop)\n", *output)

	if err := session.Record(*output, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := session.Wait(); err != nil {
		// Exit errors are expected when the user exits the shell
		// Only report unexpected errors
		if session.ExitCode() != 0 {
			os.Exit(session.ExitCode())
		}
	}

	fmt.Fprintf(os.Stderr, "\nRecording saved to %s\n", *output)
}

func playCmd(args []string) {
	fs := flag.NewFlagSet("play", flag.ExitOnError)
	speed := fs.Float64("s", 1.0, "Playback speed (e.g., 2.0 for 2x, 0.5 for half speed)")
	loop := fs.Bool("l", false, "Loop playback")
	maxIdle := fs.Float64("idle", 2.0, "Max idle time between events in seconds (0=preserve original)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: rec play [options] <file.cast>")
		fmt.Fprintln(os.Stderr, "\nPlay back a recorded terminal session.")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  rec play demo.cast                 # Play at normal speed")
		fmt.Fprintln(os.Stderr, "  rec play -s 2 demo.cast            # Play at 2x speed")
		fmt.Fprintln(os.Stderr, "  rec play -s 0.5 -l demo.cast       # Play at half speed, looping")
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing recording file")
		fs.Usage()
		os.Exit(1)
	}

	filename := fs.Arg(0)

	player, err := termsession.NewPlayer(filename, termsession.PlayerOptions{
		Speed:   *speed,
		Loop:    *loop,
		MaxIdle: *maxIdle,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := player.Play(); err != nil {
		fmt.Fprintf(os.Stderr, "Playback error: %v\n", err)
		os.Exit(1)
	}
}

func infoCmd(args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: rec info <file.cast>")
		fmt.Fprintln(os.Stderr, "\nShow information about a recording.")
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing recording file")
		fs.Usage()
		os.Exit(1)
	}

	filename := fs.Arg(0)

	player, err := termsession.NewPlayer(filename, termsession.PlayerOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	header := player.GetHeader()
	duration := player.GetDuration()

	fmt.Printf("File:       %s\n", filename)
	if header.Title != "" {
		fmt.Printf("Title:      %s\n", header.Title)
	}
	fmt.Printf("Size:       %dx%d\n", header.Width, header.Height)
	fmt.Printf("Duration:   %.1fs\n", duration)
	fmt.Printf("Events:     %d\n", player.EventCount())
	if len(header.Env) > 0 {
		fmt.Println("Environment:")
		for k, v := range header.Env {
			fmt.Printf("  %s=%s\n", k, v)
		}
	}
}

func usage() {
	fmt.Println(`rec - Terminal session recording and playback

Usage:
    rec <command> [options] [arguments]

Commands:
    record, rec   Record a terminal session
    play          Play back a recorded session
    info          Show information about a recording
    help          Show this help message

Record Options:
    -o <file>     Output file (default: session.cast)
    -t <title>    Recording title
    -z            Enable gzip compression (default: true)
    -redact       Redact secrets (default: true)
    -idle <sec>   Max idle time between events (0=unlimited)

Play Options:
    -s <speed>    Playback speed multiplier (default: 1.0)
    -l            Loop playback
    -idle <sec>   Max idle time between events (default: 2.0)

Examples:
    rec record                          # Record interactive shell
    rec record -o demo.cast bash        # Record bash session
    rec record python script.py         # Record command execution
    rec play demo.cast                  # Play recording
    rec play -s 2 demo.cast             # Play at 2x speed
    rec info demo.cast                  # Show recording info

The recording format is asciinema v2 (.cast), compatible with asciinema.org.`)
}
