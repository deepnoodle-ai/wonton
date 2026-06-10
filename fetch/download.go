package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// DownloadOptions configures a [Download] call. The zero value (or nil)
// downloads the file into memory with no size limit.
type DownloadOptions struct {
	// Headers contains additional HTTP headers to include in the request.
	Headers map[string]string `json:"headers,omitempty"`

	// OutputPath is the destination file path or directory. If it's a
	// directory (or ends with a path separator), the filename is derived
	// from the Content-Disposition header or the URL. If empty, the file
	// content is returned in DownloadResult.Data instead. An existing file
	// at the destination is overwritten.
	OutputPath string `json:"output_path,omitempty"`

	// CreateDirs creates parent directories if they don't exist.
	CreateDirs bool `json:"create_dirs,omitempty"`

	// MaxSizeBytes limits the maximum file size to download. A value of 0
	// means no limit.
	MaxSizeBytes int64 `json:"max_size_bytes,omitempty"`

	// ExpectedType, if set, requires the response Content-Type to match
	// this MIME type (e.g. "application/pdf"). Only the media type is
	// compared; parameters like charset are ignored.
	ExpectedType string `json:"expected_type,omitempty"`

	// Client is the HTTP client used for the request. If nil,
	// DefaultDownloadClient is used.
	Client *http.Client `json:"-"`
}

// DownloadResult contains the result of a [Download] call.
type DownloadResult struct {
	// Filename is the name of the downloaded file.
	Filename string

	// Size is the number of bytes downloaded.
	Size int64

	// ContentType is the MIME type reported by the server.
	ContentType string

	// Path is the file path where content was saved (only set if
	// OutputPath was specified in the options).
	Path string

	// Data contains the file content (only populated if OutputPath was not
	// specified in the options).
	Data []byte
}

// DefaultDownloadClient is the HTTP client used by [Download] when no
// client is provided. It has no overall timeout — a wall-clock cap on the
// whole transfer would make large downloads on slow connections fail — but
// it does bound how long the server may take to start responding. Use the
// context to cancel a download or to impose a deadline.
var DefaultDownloadClient = &http.Client{Transport: newDownloadTransport()}

func newDownloadTransport() http.RoundTripper {
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		t := base.Clone()
		t.ResponseHeaderTimeout = 30 * time.Second
		return t
	}
	return http.DefaultTransport
}

// downloadUserAgent is sent when the caller does not provide a User-Agent
// header. The Go default ("Go-http-client/1.1") is rejected by some CDNs.
const downloadUserAgent = "wonton/1.0 (+https://github.com/deepnoodle-ai/wonton)"

// Download fetches a file from a URL, optionally saving it to disk.
//
// With no options (nil is fine), the file content is returned in
// [DownloadResult].Data. Set [DownloadOptions].OutputPath to save to disk
// instead — filenames derived from the response (Content-Disposition header
// or URL path) are sanitized to prevent path traversal attacks.
//
// HTTP error responses are returned as a *[Error] carrying the status code,
// so callers can test failures with [errors.As] or classify them with
// [IsRetryable].
//
// Example:
//
//	// Download to memory
//	result, err := fetch.Download(ctx, "https://example.com/doc.pdf", nil)
//
//	// Download to a directory, with limits
//	result, err = fetch.Download(ctx, "https://example.com/doc.pdf", &fetch.DownloadOptions{
//	    OutputPath:   "/tmp/downloads/", // filename from response
//	    MaxSizeBytes: 10 * 1024 * 1024,
//	    ExpectedType: "application/pdf",
//	})
func Download(ctx context.Context, url string, opts *DownloadOptions) (*DownloadResult, error) {
	if opts == nil {
		opts = &DownloadOptions{}
	}
	if url == "" {
		return nil, errors.New("url cannot be empty")
	}

	client := opts.Client
	if client == nil {
		client = DefaultDownloadClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", downloadUserAgent)
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &Error{StatusCode: resp.StatusCode, URL: url}
	}

	contentType := resp.Header.Get("Content-Type")

	if opts.ExpectedType != "" {
		expectedType, _, err := mime.ParseMediaType(opts.ExpectedType)
		if err != nil {
			return nil, fmt.Errorf("invalid expected type %q: %w", opts.ExpectedType, err)
		}
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			return nil, fmt.Errorf("invalid content type header: %w", err)
		}
		if mediaType != expectedType {
			return nil, fmt.Errorf("content type mismatch: expected %s, got %s", expectedType, mediaType)
		}
	}

	// Reject early when the server declares a size over the limit
	if opts.MaxSizeBytes > 0 && resp.ContentLength > opts.MaxSizeBytes {
		return nil, fmt.Errorf("file size exceeds maximum allowed size: %d > %d", resp.ContentLength, opts.MaxSizeBytes)
	}

	// Determine whether OutputPath names a directory. A trailing separator
	// counts as a directory even if it doesn't exist yet.
	var outputIsDir bool
	if opts.OutputPath != "" {
		if strings.HasSuffix(opts.OutputPath, string(filepath.Separator)) ||
			strings.HasSuffix(opts.OutputPath, "/") {
			outputIsDir = true
		} else if fileInfo, err := os.Stat(opts.OutputPath); err == nil {
			outputIsDir = fileInfo.IsDir()
		}
	}

	// A filename from the response is only required when we aren't given an
	// explicit destination file path.
	var filename string
	if opts.OutputPath == "" || outputIsDir {
		filename, err = safeFilenameFromResponse(resp)
		if err != nil {
			return nil, err
		}
	} else {
		filename = filepath.Base(opts.OutputPath)
	}

	result := &DownloadResult{
		Filename:    filename,
		ContentType: contentType,
	}

	var reader io.Reader = resp.Body
	if opts.MaxSizeBytes > 0 {
		reader = io.LimitReader(resp.Body, opts.MaxSizeBytes+1) // +1 to detect overflow
	}

	if opts.OutputPath == "" {
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		if opts.MaxSizeBytes > 0 && int64(len(data)) > opts.MaxSizeBytes {
			return nil, fmt.Errorf("response size exceeds maximum allowed size of %d bytes", opts.MaxSizeBytes)
		}
		result.Data = data
		result.Size = int64(len(data))
		return result, nil
	}

	outputPath := opts.OutputPath
	if outputIsDir {
		outputPath = filepath.Join(outputPath, filename)
	}

	// Verify the final path stays inside the intended directory
	if opts.CreateDirs || outputIsDir {
		baseDir := opts.OutputPath
		if !outputIsDir {
			baseDir = filepath.Dir(opts.OutputPath)
		}
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			return nil, fmt.Errorf("invalid output path: %w", err)
		}
		absOutput, err := filepath.Abs(outputPath)
		if err != nil {
			return nil, fmt.Errorf("invalid output path: %w", err)
		}
		if !strings.HasPrefix(absOutput, absBase+string(filepath.Separator)) && absOutput != absBase {
			return nil, errors.New("path traversal detected: output path escapes base directory")
		}
	}

	if opts.CreateDirs {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory structure: %w", err)
		}
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	written, err := io.Copy(outputFile, reader)
	if closeErr := outputFile.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(outputPath) // Clean up partial file
		return nil, fmt.Errorf("failed to write file contents: %w", err)
	}
	if opts.MaxSizeBytes > 0 && written > opts.MaxSizeBytes {
		os.Remove(outputPath) // Clean up partial file
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", opts.MaxSizeBytes)
	}

	result.Size = written
	result.Path = outputPath
	return result, nil
}

// safeFilenameFromResponse extracts a filename from the response and sanitizes
// it to prevent path traversal attacks.
//
// Returns an error if no valid filename can be determined.
func safeFilenameFromResponse(resp *http.Response) (string, error) {
	var filename string

	// Try Content-Disposition header first
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		filename = filenameFromContentDisposition(cd)
	}

	// Fall back to the URL path (of the final URL, after redirects)
	if filename == "" {
		filename = filepath.Base(resp.Request.URL.Path)
	}

	filename = sanitizeFilename(filename)
	if filename == "" {
		return "", errors.New("could not determine a valid filename from response")
	}
	return filename, nil
}

// sanitizeFilename removes path traversal attempts and invalid characters from
// a filename. Returns an empty string if the filename is invalid.
func sanitizeFilename(filename string) string {
	// Normalize separators so the logic is OS-independent. path.Base treats
	// only "/" as a separator, so filepath.Base("/") returning "\" on Windows
	// would otherwise leak through.
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = path.Base(filename)

	// Reject if it's a directory reference or empty
	if filename == "." || filename == ".." || filename == "/" || filename == "" {
		return ""
	}

	// Remove null bytes and other control characters
	var clean strings.Builder
	for _, r := range filename {
		if r >= 32 && r != 127 {
			clean.WriteRune(r)
		}
	}
	filename = clean.String()

	// Trim whitespace and dots from beginning and end
	return strings.Trim(filename, " .")
}

// filenameFromContentDisposition extracts the filename from a
// Content-Disposition header. Uses mime.ParseMediaType for proper RFC 2231
// parsing when possible, with a lenient fallback for non-compliant headers.
func filenameFromContentDisposition(cd string) string {
	// Try standard mime parsing first
	_, params, err := mime.ParseMediaType(cd)
	if err == nil {
		if filename, ok := params["filename"]; ok && filename != "" {
			return trimQuotes(filename)
		}
	}

	// Fallback: simple parsing for non-compliant headers. Header values are
	// effectively ASCII, but guard against multi-byte case folding shifting
	// byte offsets between the lowered copy and the original.
	lower := strings.ToLower(cd)
	if len(lower) != len(cd) {
		lower = cd
	}
	const filenamePrefix = "filename="
	idx := strings.Index(lower, filenamePrefix)
	if idx < 0 {
		return ""
	}
	filename := cd[idx+len(filenamePrefix):]

	// Handle quoted filenames
	if len(filename) > 0 && (filename[0] == '"' || filename[0] == '\'') {
		quote := filename[0]
		if endIdx := strings.IndexByte(filename[1:], quote); endIdx >= 0 {
			return filename[1 : endIdx+1]
		}
	}

	// Handle unquoted filenames (ending at first semicolon or end of string)
	if endIdx := strings.IndexByte(filename, ';'); endIdx >= 0 {
		return strings.TrimSpace(filename[:endIdx])
	}
	return strings.TrimSpace(filename)
}

// trimQuotes removes a matching pair of surrounding single quotes.
// mime.ParseMediaType handles double quotes itself but treats single quotes
// as literal characters, which would otherwise end up in the filename.
func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}
