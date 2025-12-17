package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/htmlparse"
	"github.com/deepnoodle-ai/wonton/htmltomd"
)

const (
	// DefaultMaxBodySize is the maximum response body size (10 MB).
	// Responses larger than this will be rejected to prevent memory issues.
	DefaultMaxBodySize = 10 * 1024 * 1024

	// DefaultTimeout is the default HTTP request timeout (30 seconds).
	DefaultTimeout = 30 * time.Second
)

var (
	// DefaultHTTPClient is the default HTTP client used when none is specified.
	DefaultHTTPClient = &http.Client{Timeout: DefaultTimeout}

	// DefaultHeaders are the default HTTP headers sent with requests.
	// Empty by default; populate to set global headers.
	DefaultHeaders = map[string]string{}
)

// HTTPFetcherOptions defines configuration options for HTTPFetcher.
//
// All fields are optional. When not specified, sensible defaults are used.
type HTTPFetcherOptions struct {
	// Timeout is the HTTP request timeout. Defaults to DefaultTimeout (30s).
	Timeout time.Duration

	// Headers are default HTTP headers sent with all requests.
	// Request-specific headers override these. Defaults to DefaultHeaders.
	Headers map[string]string

	// Client is the HTTP client to use for requests.
	// Defaults to DefaultHTTPClient.
	Client *http.Client

	// MaxBodySize is the maximum response body size in bytes.
	// Responses larger than this are rejected. Defaults to DefaultMaxBodySize (10 MB).
	MaxBodySize int64
}

// HTTPFetcher implements the Fetcher interface using Go's standard HTTP client.
//
// HTTPFetcher provides basic page fetching without browser automation features.
// It supports HTML fetching, custom headers, timeouts, and content processing,
// but does not support JavaScript execution, screenshots, or browser emulation.
//
// For advanced features like screenshots or mobile emulation, use a Fetcher
// implementation that supports browser automation.
type HTTPFetcher struct {
	timeout     time.Duration
	headers     map[string]string
	client      *http.Client
	maxBodySize int64
}

// validateRequest checks for unsupported options and returns an error if any are set.
func (f *HTTPFetcher) validateRequest(req *Request) error {
	if req.MaxAge > 0 {
		return ErrUnsupportedOption("MaxAge")
	}
	if req.WaitFor > 0 {
		return ErrUnsupportedOption("WaitFor")
	}
	if req.Mobile {
		return ErrUnsupportedOption("Mobile")
	}
	if len(req.Actions) > 0 {
		return ErrUnsupportedOption("Actions")
	}
	if len(req.StorageState) > 0 {
		return ErrUnsupportedOption("StorageState")
	}
	// Check for unsupported formats
	for _, format := range req.Formats {
		switch format {
		case "markdown", "html", "raw_html", "links", "images", "branding":
			// supported
		case "screenshot", "json", "summary":
			return ErrUnsupportedOption("format " + format)
		}
	}
	return nil
}

// NewHTTPFetcher creates a new HTTPFetcher with the given options.
//
// All options are optional and will use sensible defaults if not specified.
// Returns a configured HTTPFetcher ready to use.
//
// Example:
//
//	fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{
//		Timeout: 10 * time.Second,
//		Headers: map[string]string{
//			"User-Agent": "MyApp/1.0",
//		},
//	})
func NewHTTPFetcher(options HTTPFetcherOptions) *HTTPFetcher {
	if options.Timeout == 0 {
		options.Timeout = DefaultTimeout
	}
	if options.Headers == nil {
		options.Headers = DefaultHeaders
	}
	if options.Client == nil {
		options.Client = DefaultHTTPClient
	}
	if options.MaxBodySize == 0 {
		options.MaxBodySize = DefaultMaxBodySize
	}
	return &HTTPFetcher{
		timeout:     options.Timeout,
		headers:     options.Headers,
		client:      options.Client,
		maxBodySize: options.MaxBodySize,
	}
}

// Fetch implements the Fetcher interface for HTTP requests.
//
// Fetches the page at req.URL and processes it according to the request options.
// Returns an error if the URL is invalid, the request fails, the content type
// is not HTML, the response exceeds MaxBodySize, or any unsupported options are set.
//
// Supported request options:
//   - URL (required)
//   - Formats (html, raw_html, markdown, links, images, branding)
//   - Headers
//   - Timeout
//   - OnlyMainContent
//   - IncludeTags/ExcludeTags
//   - ExcludeFilters
//   - Prettify
//
// Unsupported options that will return ErrUnsupported:
//   - MaxAge, WaitFor, Mobile, Actions, StorageState
//   - Formats: screenshot, json, summary
func (f *HTTPFetcher) Fetch(ctx context.Context, req *Request) (*Response, error) {
	// Check for unsupported options
	if err := f.validateRequest(req); err != nil {
		return nil, err
	}

	// Apply timeout: per-request timeout takes precedence, otherwise use fetcher default
	var cancel context.CancelFunc
	if req.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Millisecond)
	} else {
		ctx, cancel = context.WithTimeout(ctx, f.timeout)
	}
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return nil, err
	}

	// Apply default headers
	for key, value := range f.headers {
		if httpReq.Header.Get(key) == "" {
			httpReq.Header.Set(key, value)
		}
	}

	// Apply custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := f.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Confirm the content type indicates HTML
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}

	// Use LimitReader to prevent reading excessive data
	limitedReader := io.LimitReader(resp.Body, f.maxBodySize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	// Check if the body is too large
	if len(body) > int(f.maxBodySize) {
		return nil, fmt.Errorf("response size exceeds limit of %d bytes", f.maxBodySize)
	}

	// Convert response headers to map[string]string
	headers := make(map[string]string)
	for name, values := range resp.Header {
		if len(values) > 0 {
			headers[name] = values[0] // Use first value if multiple
		}
	}

	// Apply processing options
	response, err := ProcessRequest(req, string(body))
	if err != nil {
		return nil, err
	}

	// Set other response fields
	// Use the final URL after any redirects
	response.URL = resp.Request.URL.String()
	response.StatusCode = resp.StatusCode
	response.Headers = headers
	return response, nil
}

// ProcessRequest applies request options to HTML content and builds a response.
//
// This function processes raw HTML according to the request options, applying
// transformations (filtering, formatting) and generating requested output formats.
// It can be used standalone to process HTML content without fetching.
//
// The function parses the HTML, applies any include/exclude filters, extracts
// metadata, and generates the requested output formats (HTML, Markdown, links,
// images, branding).
//
// Returns a Response with the processed content and metadata, or an error if
// HTML parsing fails.
func ProcessRequest(request *Request, htmlContent string) (*Response, error) {
	htmlContent = strings.TrimSpace(htmlContent)
	if htmlContent == "" {
		return &Response{
			URL:        request.URL,
			StatusCode: 200,
		}, nil
	}

	// Parse the HTML
	doc, err := htmlparse.Parse(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %w", err)
	}
	metadata := doc.Metadata()

	// Render transformed HTML with options
	transformOpts := &htmlparse.TransformOptions{
		PrettyPrint:     request.Prettify,
		Include:         request.IncludeTags,
		Exclude:         request.ExcludeTags,
		ExcludeFilters:  request.ExcludeFilters,
		OnlyMainContent: request.OnlyMainContent,
	}
	renderedHTML := doc.Transform(transformOpts)

	// By default, return the HTML but not markdown
	includeHTML := true
	includeRawHTML := false
	includeMarkdown := false
	includeLinks := false
	includeImages := false
	includeBranding := false

	// Specified formats were requested
	if len(request.Formats) > 0 {
		includeHTML = false
		for _, format := range request.Formats {
			switch format {
			case "markdown":
				includeMarkdown = true
			case "html":
				includeHTML = true
			case "raw_html":
				includeRawHTML = true
			case "links":
				includeLinks = true
			case "images":
				includeImages = true
			case "branding":
				includeBranding = true
			}
		}
	}

	// Generate markdown if requested
	var markdownContent string
	if includeMarkdown {
		markdownContent = htmltomd.Convert(renderedHTML)
	}

	// Get links from document if requested
	var links []Link
	if includeLinks {
		links = doc.Links()
	}

	// Get images from document if requested
	var images []Image
	if includeImages {
		images = doc.Images()
	}

	// Extract branding if requested
	var branding *BrandingProfile
	if includeBranding {
		b := doc.Branding()
		branding = &BrandingProfile{
			ColorScheme: b.ColorScheme,
			Logo:        b.Logo,
			Images: &BrandingImages{
				Logo:    b.Logo,
				Favicon: b.Favicon,
			},
		}
		// Set primary color from theme-color if available
		if b.ThemeColor != "" {
			branding.Colors = &BrandingColors{
				Primary: b.ThemeColor,
			}
		}
		// Use apple icon as logo fallback
		if branding.Logo == "" && b.AppleIcon != "" {
			branding.Logo = b.AppleIcon
			branding.Images.Logo = b.AppleIcon
		}
	}

	// Build response
	resp := &Response{
		URL:        request.URL,
		StatusCode: 200,
		Headers:    map[string]string{},
		Markdown:   markdownContent,
		Metadata:   metadata,
		Links:      links,
		Images:     images,
		Branding:   branding,
		Timestamp:  time.Now().UTC(),
	}

	// Include HTML formats as requested
	if includeHTML {
		resp.HTML = renderedHTML
	}
	if includeRawHTML {
		resp.RawHTML = htmlContent
	}

	return resp, nil
}
