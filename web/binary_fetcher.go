package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BinaryFetchInput contains parameters for fetching binary files.
type BinaryFetchInput struct {
	// URL is the address to fetch the binary file from.
	URL string `json:"url"`

	// Headers contains additional HTTP headers to include in the request.
	Headers map[string]string `json:"headers,omitempty"`

	// OutputPath is the destination file path or directory. If it's a directory,
	// the filename is derived from the URL or Content-Disposition header.
	// If empty, the file content is returned in BinaryFetchResult.Data.
	OutputPath string `json:"output_path,omitempty"`

	// CreateDirs creates parent directories if they don't exist.
	CreateDirs bool `json:"create_dirs,omitempty"`

	// MaxSizeBytes limits the maximum file size to download. A value of 0
	// means no limit.
	MaxSizeBytes int64 `json:"max_size_bytes,omitempty"`

	// ExpectedType is the expected MIME type (e.g., "application/pdf", "image/jpeg").
	// Only the media type is compared; parameters like charset are ignored.
	ExpectedType string `json:"expected_type,omitempty"`

	// VerifyMimeType enables MIME type verification against ExpectedType.
	VerifyMimeType bool `json:"verify_mime_type,omitempty"`
}

// BinaryFetchResult contains the result of a binary file fetch operation.
type BinaryFetchResult struct {
	// Filename is the name of the downloaded file.
	Filename string

	// Size is the number of bytes downloaded.
	Size int64

	// ContentType is the MIME type reported by the server.
	ContentType string

	// DownloadPath is the file path where content was saved (only set if
	// OutputPath was specified in the input).
	DownloadPath string

	// Data contains the file content (only populated if OutputPath was not
	// specified in the input).
	Data []byte
}

// BinaryFetcher defines the interface for fetching binary files from URLs.
//
// Implementations should handle HTTP redirects, respect size limits, and
// sanitize filenames to prevent path traversal attacks.
type BinaryFetcher interface {
	// FetchBinary downloads a binary file from the specified URL.
	// Returns the result containing either the file data or the path where
	// it was saved.
	FetchBinary(ctx context.Context, input *BinaryFetchInput) (*BinaryFetchResult, error)
}

// DefaultBinaryFetcher provides a standard implementation of BinaryFetcher
// with sensible defaults for production use.
type DefaultBinaryFetcher struct {
	// Client is the HTTP client used for requests. If nil, a default client
	// with a 30-second timeout is used.
	Client *http.Client
}

// NewDefaultBinaryFetcher creates a new binary fetcher with a default HTTP
// client configured with a 30-second timeout.
func NewDefaultBinaryFetcher() *DefaultBinaryFetcher {
	return &DefaultBinaryFetcher{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchBinary downloads a binary file from the specified URL.
//
// The function performs filename sanitization to prevent path traversal attacks
// when saving to disk. Filenames from Content-Disposition headers or URLs are
// cleaned to remove path separators and parent directory references.
func (f *DefaultBinaryFetcher) FetchBinary(ctx context.Context, input *BinaryFetchInput) (*BinaryFetchResult, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", input.URL, nil)
	if err != nil {
		return nil, err
	}

	// Add headers if specified
	for key, value := range input.Headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, NewFetchError(resp.StatusCode, fmt.Errorf("failed to fetch binary from %s", input.URL))
	}

	contentType := resp.Header.Get("Content-Type")

	// Verify content type if requested (using mime.ParseMediaType to handle parameters)
	if input.VerifyMimeType && input.ExpectedType != "" {
		expectedType, _, err := mime.ParseMediaType(input.ExpectedType)
		if err != nil {
			return nil, fmt.Errorf("invalid expected type %q: %w", input.ExpectedType, err)
		}
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			return nil, fmt.Errorf("invalid content type header: %w", err)
		}
		if mediaType != expectedType {
			return nil, fmt.Errorf("content type mismatch: expected %s, got %s", expectedType, mediaType)
		}
	}

	// Get content length if available
	contentLength := resp.ContentLength

	// Check size limits if specified
	if input.MaxSizeBytes > 0 && contentLength > 0 && contentLength > input.MaxSizeBytes {
		return nil, fmt.Errorf("file size exceeds maximum allowed size: %d > %d", contentLength, input.MaxSizeBytes)
	}

	// Determine if we need a filename from response
	// Only required when OutputPath is empty or is a directory
	var filename string
	var outputIsDir bool

	if input.OutputPath != "" {
		// Check if path ends with a separator (treat as directory even if doesn't exist)
		endsWithSep := strings.HasSuffix(input.OutputPath, string(filepath.Separator)) ||
			strings.HasSuffix(input.OutputPath, "/")

		if endsWithSep {
			outputIsDir = true
		} else {
			// Check if existing path is a directory
			fileInfo, statErr := os.Stat(input.OutputPath)
			outputIsDir = statErr == nil && fileInfo.IsDir()
		}
	}

	needsResponseFilename := input.OutputPath == "" || outputIsDir
	if needsResponseFilename {
		var err error
		filename, err = safeFilenameFromResponse(resp)
		if err != nil {
			return nil, err
		}
	} else {
		// OutputPath is an explicit file path; use its basename
		filename = filepath.Base(input.OutputPath)
	}

	result := &BinaryFetchResult{
		Filename:    filename,
		ContentType: contentType,
		Size:        contentLength,
	}

	// Apply size limit if specified
	var reader io.Reader = resp.Body
	if input.MaxSizeBytes > 0 {
		reader = io.LimitReader(resp.Body, input.MaxSizeBytes+1) // +1 to detect overflow
	}

	// If output path is specified, save to file
	if input.OutputPath != "" {
		outputPath := input.OutputPath

		// If output path is a directory, append the sanitized filename
		if outputIsDir {
			outputPath = filepath.Join(outputPath, filename)
		}

		// Verify the final path is safe (doesn't escape intended directory)
		if input.CreateDirs || outputIsDir {
			baseDir := input.OutputPath
			if !outputIsDir {
				baseDir = filepath.Dir(input.OutputPath)
			}
			absBase, _ := filepath.Abs(baseDir)
			absOutput, _ := filepath.Abs(outputPath)
			if !strings.HasPrefix(absOutput, absBase+string(filepath.Separator)) && absOutput != absBase {
				return nil, fmt.Errorf("path traversal detected: output path escapes base directory")
			}
		}

		// Create directories if requested and needed
		if input.CreateDirs {
			dir := filepath.Dir(outputPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory structure: %w", err)
			}
		}

		// Create the file
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		defer outputFile.Close()

		// Copy the response body to the file with size limit
		written, err := io.Copy(outputFile, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to write file contents: %w", err)
		}

		// Check if we exceeded the size limit
		if input.MaxSizeBytes > 0 && written > input.MaxSizeBytes {
			os.Remove(outputPath) // Clean up partial file
			return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", input.MaxSizeBytes)
		}

		result.Size = written
		result.DownloadPath = outputPath
	} else {
		// If no output path, read into memory with size limit
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Check if we exceeded the size limit
		if input.MaxSizeBytes > 0 && int64(len(data)) > input.MaxSizeBytes {
			return nil, fmt.Errorf("response size exceeds maximum allowed size of %d bytes", input.MaxSizeBytes)
		}

		result.Data = data
		result.Size = int64(len(data))
	}

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
		filename = extractFilenameFromContentDisposition(cd)
	}

	// Fall back to the URL path
	if filename == "" {
		path := resp.Request.URL.Path
		filename = filepath.Base(path)
	}

	// Sanitize the filename
	filename = sanitizeFilename(filename)

	if filename == "" {
		return "", errors.New("could not determine a valid filename from response")
	}

	return filename, nil
}

// sanitizeFilename removes path traversal attempts and invalid characters from
// a filename. Returns an empty string if the filename is invalid.
func sanitizeFilename(filename string) string {
	// Remove any path components - only keep the base name
	filename = filepath.Base(filename)

	// Reject if it's a directory reference or empty
	if filename == "." || filename == ".." || filename == "/" || filename == "" {
		return ""
	}

	// Remove any remaining path separators (shouldn't happen after Base, but defensive)
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	// Remove null bytes and other control characters
	var clean strings.Builder
	for _, r := range filename {
		if r >= 32 && r != 127 {
			clean.WriteRune(r)
		}
	}
	filename = clean.String()

	// Trim whitespace and dots from beginning and end
	filename = strings.Trim(filename, " .")

	if filename == "" {
		return ""
	}

	return filename
}

// extractFilenameFromContentDisposition extracts filename from Content-Disposition header.
// Uses mime.ParseMediaType for proper RFC 2231 parsing when possible.
func extractFilenameFromContentDisposition(cd string) string {
	// Try standard mime parsing first
	_, params, err := mime.ParseMediaType(cd)
	if err == nil {
		if filename, ok := params["filename"]; ok && filename != "" {
			return filename
		}
	}

	// Fallback: simple parsing for non-compliant headers
	const filenamePrefix = "filename="
	if idx := strings.Index(strings.ToLower(cd), filenamePrefix); idx >= 0 {
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

	return ""
}
