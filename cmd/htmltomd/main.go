// Command htmltomd converts HTML to Markdown from a URL or local file.
//
// Usage:
//
//	htmltomd <url-or-file>
//	htmltomd https://example.com
//	htmltomd ./page.html
//	cat page.html | htmltomd -
//
// Options:
//
//	-refs      Use referenced link style [text][n] instead of inline [text](url)
//	-setext    Use setext-style headings (underlined) for h1/h2
//	-indent    Use indented code blocks instead of fenced
//	-bullet    Bullet character for lists (default "-")
//	-skip      Comma-separated tags to skip (e.g. "nav,footer,aside")
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/htmltomd"
)

func main() {
	refs := flag.Bool("refs", false, "use referenced link style")
	setext := flag.Bool("setext", false, "use setext-style headings")
	indent := flag.Bool("indent", false, "use indented code blocks")
	bullet := flag.String("bullet", "-", "bullet character for lists")
	skip := flag.String("skip", "", "comma-separated tags to skip")
	timeout := flag.Duration("timeout", 30*time.Second, "HTTP request timeout")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <url-or-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Converts HTML to Markdown from a URL, local file, or stdin.\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  <url-or-file>  URL to fetch, path to local file, or - for stdin\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	input := flag.Arg(0)

	var body []byte
	var err error

	switch {
	case input == "-":
		// Read from stdin
		body, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}

	case isURL(input):
		// Fetch from URL
		body, err = fetchURL(input, *timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		// Read from local file
		body, err = os.ReadFile(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	}

	// Build options
	opts := htmltomd.DefaultOptions()

	if *refs {
		opts.LinkStyle = htmltomd.LinkStyleReferenced
	}
	if *setext {
		opts.HeadingStyle = htmltomd.HeadingStyleSetext
	}
	if *indent {
		opts.CodeBlockStyle = htmltomd.CodeBlockStyleIndented
	}
	if *bullet != "" {
		opts.BulletChar = *bullet
	}
	if *skip != "" {
		opts.SkipTags = strings.Split(*skip, ",")
	}

	// Convert and output
	md := htmltomd.ConvertWithOptions(string(body), opts)
	fmt.Println(md)
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

// fetchURL fetches content from a URL.
func fetchURL(url string, timeout time.Duration) ([]byte, error) {
	// Ensure URL has a scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}
