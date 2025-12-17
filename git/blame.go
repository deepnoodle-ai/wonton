package git

import (
	"bufio"
	"bytes"
	"context"
	"strconv"
	"strings"
	"time"
)

// BlameOptions configures the Blame command, allowing you to specify
// a revision and line range to blame.
type BlameOptions struct {
	// Ref is the revision to blame (defaults to HEAD if empty).
	// Can be a commit hash, branch name, tag, or any valid git ref.
	Ref string
	// StartLine is the first line to blame (1-indexed, inclusive).
	// If 0, starts from the beginning of the file.
	StartLine int
	// EndLine is the last line to blame (1-indexed, inclusive).
	// If 0 (and StartLine is also 0), blames the entire file.
	EndLine int
}

// Blame returns line-by-line attribution for a file, showing which commit
// last modified each line.
//
// This is useful for understanding the history of a file and identifying
// who changed specific lines. Each line in the result includes the commit
// hash, author, timestamp, line number, and content.
//
// Example:
//
//	blame, err := repo.Blame(ctx, "main.go", git.BlameOptions{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, line := range blame {
//	    fmt.Printf("%s %s: %s\n", line.Hash[:7], line.Author, line.Content)
//	}
//
// To blame specific lines:
//
//	blame, err := repo.Blame(ctx, "main.go", git.BlameOptions{
//	    StartLine: 10,
//	    EndLine:   20,
//	})
//
// To blame at a specific revision:
//
//	blame, err := repo.Blame(ctx, "main.go", git.BlameOptions{
//	    Ref: "v1.0.0",
//	})
func (r *Repository) Blame(ctx context.Context, path string, opts BlameOptions) ([]BlameLine, error) {
	args := []string{"blame", "--porcelain"}

	if opts.StartLine > 0 && opts.EndLine > 0 {
		args = append(args, "-L", strconv.Itoa(opts.StartLine)+","+strconv.Itoa(opts.EndLine))
	} else if opts.StartLine > 0 {
		args = append(args, "-L", strconv.Itoa(opts.StartLine)+",")
	}

	if opts.Ref != "" {
		args = append(args, opts.Ref)
	}

	args = append(args, "--", path)

	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}

	return parseBlameOutput(out)
}

func parseBlameOutput(out []byte) ([]BlameLine, error) {
	var lines []BlameLine
	scanner := bufio.NewScanner(bytes.NewReader(out))

	var current BlameLine
	commitInfo := make(map[string]map[string]string) // cache commit info by hash

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}

		// Line starting with a hash is a new blame entry
		if len(line) >= 40 && isHexString(line[:40]) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				current = BlameLine{
					Hash: parts[0],
				}
				current.LineNumber, _ = strconv.Atoi(parts[2])
			}
			continue
		}

		// Parse metadata lines
		if strings.HasPrefix(line, "author ") {
			current.Author = strings.TrimPrefix(line, "author ")
			// Cache for this commit
			if commitInfo[current.Hash] == nil {
				commitInfo[current.Hash] = make(map[string]string)
			}
			commitInfo[current.Hash]["author"] = current.Author
		} else if strings.HasPrefix(line, "author-mail ") {
			current.Email = strings.Trim(strings.TrimPrefix(line, "author-mail "), "<>")
			if commitInfo[current.Hash] != nil {
				commitInfo[current.Hash]["email"] = current.Email
			}
		} else if strings.HasPrefix(line, "author-time ") {
			if ts, err := strconv.ParseInt(strings.TrimPrefix(line, "author-time "), 10, 64); err == nil {
				current.Timestamp = time.Unix(ts, 0)
			}
			if commitInfo[current.Hash] != nil {
				commitInfo[current.Hash]["time"] = strings.TrimPrefix(line, "author-time ")
			}
		} else if strings.HasPrefix(line, "\t") {
			// Content line starts with tab
			current.Content = line[1:]

			// Fill in cached info if missing
			if info, ok := commitInfo[current.Hash]; ok {
				if current.Author == "" {
					current.Author = info["author"]
				}
				if current.Email == "" {
					current.Email = info["email"]
				}
				if current.Timestamp.IsZero() {
					if ts, err := strconv.ParseInt(info["time"], 10, 64); err == nil {
						current.Timestamp = time.Unix(ts, 0)
					}
				}
			}

			lines = append(lines, current)
			current = BlameLine{}
		}
	}

	return lines, scanner.Err()
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
