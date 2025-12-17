// Command htmltomd converts HTML to Markdown from a URL or local file.
//
// Usage:
//
//	htmltomd [options] <url-or-file>
//	htmltomd https://example.com
//	htmltomd ./page.html
//	cat page.html | htmltomd -
//
// Options:
//
//	-r, --refs      Use referenced link style [text][n] instead of inline [text](url)
//	-s, --setext    Use setext-style headings (underlined) for h1/h2
//	-i, --indent    Use indented code blocks instead of fenced
//	-b, --bullet    Bullet character for lists (default "-")
//	--skip          Comma-separated tags to skip (e.g. "nav,footer,aside")
//	-t, --timeout   HTTP request timeout (default 30s)
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/htmltomd"
)

const version = "1.0.0"

func main() {
	app := cli.New("htmltomd").
		Description("Convert HTML to Markdown").
		Version(version)

	// Default command for conversion
	app.Command("convert").
		Description("Convert HTML to Markdown from a URL, file, or stdin").
		Args("source").
		Flags(
			cli.Bool("refs", "r").Help("Use referenced link style [text][n]"),
			cli.Bool("setext", "s").Help("Use setext-style headings (underlined)"),
			cli.Bool("indent", "i").Help("Use indented code blocks instead of fenced"),
			cli.String("bullet", "b").Default("-").Help("Bullet character for lists"),
			cli.String("skip", "").Help("Comma-separated tags to skip (e.g. nav,footer,aside)"),
			cli.Duration("timeout", "t").Default(30*time.Second).Help("HTTP request timeout"),
		).
		Run(runConvert)

	// Make convert the default command by handling args directly
	args := os.Args[1:]

	// If first arg doesn't start with - and isn't "help" or "version", treat as convert
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") &&
		args[0] != "help" && args[0] != "version" && args[0] != "convert" {
		args = append([]string{"convert"}, args...)
	} else if len(args) == 0 || (len(args) > 0 && strings.HasPrefix(args[0], "-")) {
		// If no args or first arg is a flag, prepend convert
		args = append([]string{"convert"}, args...)
	}

	if err := app.RunArgs(args); err != nil {
		// Don't print help errors (already shown)
		if _, ok := err.(*cli.HelpRequested); !ok {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func runConvert(ctx *cli.Context) error {
	source := ctx.Arg(0)
	if source == "" {
		return fmt.Errorf("missing source argument (URL, file path, or - for stdin)")
	}

	// Get options
	opts := htmltomd.DefaultOptions()

	if ctx.Bool("refs") {
		opts.LinkStyle = htmltomd.LinkStyleReferenced
	}
	if ctx.Bool("setext") {
		opts.HeadingStyle = htmltomd.HeadingStyleSetext
	}
	if ctx.Bool("indent") {
		opts.CodeBlockStyle = htmltomd.CodeBlockStyleIndented
	}
	if bullet := ctx.String("bullet"); bullet != "" {
		opts.BulletChar = bullet
	}
	if skip := ctx.String("skip"); skip != "" {
		opts.SkipTags = strings.Split(skip, ",")
	}

	timeout := 30 * time.Second
	if t := ctx.String("timeout"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	// Read input
	var body []byte
	var err error

	switch {
	case source == "-":
		body, err = io.ReadAll(ctx.Stdin())
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}

	case isURL(source):
		body, err = fetchURL(ctx, source, timeout)
		if err != nil {
			return err
		}

	default:
		body, err = os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
	}

	// Convert and output
	md := htmltomd.ConvertWithOptions(string(body), opts)
	ctx.Println(md)
	return nil
}

// isURL returns true if the input looks like a URL.
func isURL(input string) bool {
	return strings.HasPrefix(input, "http://") ||
		strings.HasPrefix(input, "https://") ||
		// Treat domain-like inputs as URLs (contains dot, no path separator at start)
		(strings.Contains(input, ".") &&
			!strings.HasPrefix(input, "/") &&
			!strings.HasPrefix(input, "./") &&
			!strings.HasPrefix(input, "../") &&
			!strings.Contains(input, string(os.PathSeparator)))
}

// fetchURL fetches content from a URL, showing a spinner in interactive mode.
func fetchURL(ctx *cli.Context, url string, timeout time.Duration) ([]byte, error) {
	// Ensure URL has a scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// In interactive mode, show a nice spinner
	if ctx.Interactive() {
		return fetchWithSpinner(ctx.Context(), url, timeout)
	}

	// Non-interactive: just fetch
	return fetchSimple(url, timeout)
}

// fetchSimple does a basic HTTP fetch without UI.
func fetchSimple(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return io.ReadAll(resp.Body)
}

// fetchWithSpinner fetches a URL while showing a spinner.
func fetchWithSpinner(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	// Create result channels
	type result struct {
		body []byte
		err  error
	}
	done := make(chan result, 1)

	// Start fetch in background
	go func() {
		body, err := fetchSimple(url, timeout)
		done <- result{body, err}
	}()

	// Spinner frames (dots style)
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := 0
	msg := "Fetching " + truncateURL(url, 50)

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	// Hide cursor and prepare for spinner
	fmt.Fprint(os.Stderr, "\033[?25l") // Hide cursor
	defer fmt.Fprint(os.Stderr, "\033[?25h") // Show cursor on exit

	for {
		select {
		case r := <-done:
			// Clear spinner line
			fmt.Fprint(os.Stderr, "\r\033[K")
			return r.body, r.err

		case <-ticker.C:
			frame = (frame + 1) % len(frames)
			fmt.Fprintf(os.Stderr, "\r%s %s", color.Cyan.Apply(frames[frame]), msg)

		case <-ctx.Done():
			fmt.Fprint(os.Stderr, "\r\033[K")
			return nil, ctx.Err()
		}
	}
}

// truncateURL shortens a URL for display.
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}
