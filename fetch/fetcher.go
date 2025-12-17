// Package fetch provides HTTP page fetching with configurable options.
//
// This package offers a unified interface for retrieving web pages and processing
// their content. It supports multiple output formats (HTML, Markdown, links, images),
// content extraction options, and branding profile detection.
//
// # Basic Usage
//
// The simplest way to fetch a page is using HTTPFetcher:
//
//	fetcher := fetch.NewHTTPFetcher(fetch.HTTPFetcherOptions{})
//	resp, err := fetcher.Fetch(ctx, &fetch.Request{
//		URL:     "https://example.com",
//		Formats: []string{"html", "markdown"},
//	})
//
// # Output Formats
//
// The package supports multiple output formats via the Request.Formats field:
//
//   - "html" - Processed HTML with transformations applied
//   - "raw_html" - Original HTML without modifications
//   - "markdown" - HTML converted to Markdown format
//   - "links" - All hyperlinks found on the page
//   - "images" - All images found on the page
//   - "branding" - Brand identity information (colors, fonts, logos)
//
// # Content Processing
//
// Fetch requests support several content processing options:
//
//   - OnlyMainContent: Extract only the main content area
//   - IncludeTags/ExcludeTags: Filter HTML by tag names
//   - ExcludeFilters: Filter using custom element criteria
//   - Prettify: Format HTML output for readability
//
// # Fetcher Implementations
//
// The package provides two Fetcher implementations:
//
//   - HTTPFetcher: Direct HTTP fetching using Go's standard library
//   - MockFetcher: In-memory mock for testing
//
// HTTPFetcher supports basic options like custom headers, timeouts, and body size limits.
// For advanced browser automation features (screenshots, JavaScript execution, mobile
// emulation), use a remote fetching service that implements the Fetcher interface.
package fetch

import (
	"context"
	"time"

	"github.com/deepnoodle-ai/wonton/htmlparse"
)

// Link represents a hyperlink found on a page.
type Link = htmlparse.Link

// Image represents an image found on a page.
type Image = htmlparse.Image

// Metadata represents page metadata.
type Metadata = htmlparse.Metadata

// ElementFilter defines criteria for matching HTML elements.
type ElementFilter = htmlparse.ElementFilter

// StandardExcludeFilters contains commonly excluded elements for clean content extraction.
var StandardExcludeFilters = htmlparse.StandardExcludeFilters

// Request defines the payload for fetch requests.
//
// A Request configures how a web page should be fetched and processed. It includes
// the target URL, output formats, content filtering options, and optional browser
// automation features.
//
// Note: Some fields (MaxAge, WaitFor, Mobile, Actions, StorageState) require browser
// automation support and are not available with HTTPFetcher. Use these only with
// fetcher implementations that support advanced features.
type Request struct {
	// URL is the target web page to fetch (required).
	URL string `json:"url"`

	// OnlyMainContent extracts only the main content area, removing navigation,
	// headers, footers, and other peripheral elements.
	OnlyMainContent bool `json:"only_main_content,omitempty"`

	// IncludeTags specifies HTML tags to include. When set, only these tags
	// and their content are included in the output.
	IncludeTags []string `json:"include_tags,omitempty"`

	// ExcludeTags specifies HTML tags to exclude from the output.
	ExcludeTags []string `json:"exclude_tags,omitempty"`

	// ExcludeFilters provides custom element filtering using ElementFilter criteria.
	ExcludeFilters []ElementFilter `json:"exclude_filters,omitempty"`

	// MaxAge specifies the maximum cache age in milliseconds. Requires browser automation.
	MaxAge int `json:"max_age,omitempty"`

	// Timeout is the request timeout in milliseconds. If not specified, uses fetcher default.
	Timeout int `json:"timeout,omitempty"`

	// WaitFor specifies how long to wait in milliseconds before capturing content.
	// Requires browser automation.
	WaitFor int `json:"wait_for,omitempty"`

	// Fetcher specifies which fetcher implementation to use (for routing requests).
	Fetcher string `json:"fetcher,omitempty"`

	// Mobile enables mobile browser emulation. Requires browser automation.
	Mobile bool `json:"mobile,omitempty"`

	// Prettify formats the HTML output with proper indentation for readability.
	Prettify bool `json:"prettify,omitempty"`

	// Formats specifies the output formats to include in the response.
	// Supported values: "html", "raw_html", "markdown", "links", "images", "branding".
	// If empty, defaults to "html" only.
	Formats []string `json:"formats,omitempty"`

	// Actions specifies browser automation actions to perform before capturing content.
	// Requires browser automation.
	Actions []Action `json:"actions,omitempty"`

	// Headers are custom HTTP headers to include in the request.
	Headers map[string]string `json:"headers,omitempty"`

	// StorageState contains browser storage state (cookies, localStorage, etc.).
	// Requires browser automation.
	StorageState map[string]any `json:"storage_state,omitempty"`
}

// Response defines the payload for fetch responses.
//
// A Response contains the fetched page content in various formats along with
// metadata, headers, and any extracted information (links, images, branding).
// Which fields are populated depends on the Request.Formats specified.
type Response struct {
	// URL is the final URL after any redirects.
	URL string `json:"url"`

	// StatusCode is the HTTP status code (e.g., 200, 404).
	StatusCode int `json:"status_code"`

	// Headers contains the HTTP response headers.
	Headers map[string]string `json:"headers"`

	// HTML is the processed HTML content with transformations applied.
	// Populated when "html" format is requested.
	HTML string `json:"html,omitempty"`

	// RawHTML is the original HTML without any processing.
	// Populated when "raw_html" format is requested.
	RawHTML string `json:"raw_html,omitempty"`

	// Markdown is the HTML content converted to Markdown format.
	// Populated when "markdown" format is requested.
	Markdown string `json:"markdown,omitempty"`

	// Summary is an AI-generated summary of the page content.
	// Populated when "summary" format is requested (requires special fetcher support).
	Summary string `json:"summary,omitempty"`

	// Screenshot is a base64-encoded image of the rendered page.
	// Populated when "screenshot" format is requested (requires browser automation).
	Screenshot string `json:"screenshot,omitempty"`

	// JSON contains structured data extracted from the page.
	// Populated when "json" format is requested (requires special fetcher support).
	JSON map[string]any `json:"json,omitempty"`

	// Branding contains brand identity information extracted from the page.
	// Populated when "branding" format is requested.
	Branding *BrandingProfile `json:"branding,omitempty"`

	// Images contains all images found on the page.
	// Populated when "images" format is requested.
	Images []Image `json:"images,omitempty"`

	// Error contains any error message from the fetch operation.
	Error string `json:"error,omitempty"`

	// Metadata contains page metadata (title, description, author, etc.).
	// Always populated when HTML is successfully fetched.
	Metadata Metadata `json:"metadata,omitempty"`

	// Links contains all hyperlinks found on the page.
	// Populated when "links" format is requested.
	Links []Link `json:"links,omitempty"`

	// StorageState contains browser storage state captured after page load.
	// Populated when browser automation is used.
	StorageState map[string]any `json:"storage_state,omitempty"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp,omitzero"`
}

// BrandingProfile contains brand identity information extracted from a page.
//
// This includes colors, fonts, logos, and other visual design elements
// that define a website's brand identity.
type BrandingProfile struct {
	// ColorScheme describes the overall color scheme (e.g., "light", "dark").
	ColorScheme string `json:"color_scheme,omitempty"`

	// Logo is the URL of the primary logo image.
	Logo string `json:"logo,omitempty"`

	// Colors contains the brand's color palette.
	Colors *BrandingColors `json:"colors,omitempty"`

	// Fonts lists the fonts used on the page.
	Fonts []BrandingFont `json:"fonts,omitempty"`

	// Typography contains detailed font usage information.
	Typography *BrandingTypo `json:"typography,omitempty"`

	// Spacing contains information about spacing and border radius.
	Spacing *BrandingSpacing `json:"spacing,omitempty"`

	// Images contains URLs for brand images (logo, favicon, og:image).
	Images *BrandingImages `json:"images,omitempty"`
}

// BrandingColors contains brand color information extracted from a page.
type BrandingColors struct {
	// Primary is the primary brand color.
	Primary string `json:"primary,omitempty"`

	// Secondary is the secondary brand color.
	Secondary string `json:"secondary,omitempty"`

	// Accent is an accent color used for highlights.
	Accent string `json:"accent,omitempty"`

	// Background is the main background color.
	Background string `json:"background,omitempty"`

	// TextPrimary is the primary text color.
	TextPrimary string `json:"text_primary,omitempty"`

	// TextSecondary is the secondary text color.
	TextSecondary string `json:"text_secondary,omitempty"`
}

// BrandingFont represents a font family used on the page.
type BrandingFont struct {
	// Family is the font family name (e.g., "Helvetica", "Arial").
	Family string `json:"family,omitempty"`
}

// BrandingTypo contains detailed typography information.
type BrandingTypo struct {
	// FontFamilies contains font family assignments for different contexts.
	FontFamilies *TypoFamilies `json:"font_families,omitempty"`

	// FontSizes maps font size categories to CSS values.
	FontSizes map[string]string `json:"font_sizes,omitempty"`

	// FontWeights maps font weight categories to numeric values.
	FontWeights map[string]int `json:"font_weights,omitempty"`
}

// TypoFamilies contains font family assignments for different text contexts.
type TypoFamilies struct {
	// Primary is the font family used for primary content.
	Primary string `json:"primary,omitempty"`

	// Heading is the font family used for headings.
	Heading string `json:"heading,omitempty"`

	// Code is the font family used for code blocks.
	Code string `json:"code,omitempty"`
}

// BrandingSpacing contains spacing and layout information.
type BrandingSpacing struct {
	// BaseUnit is the base spacing unit in pixels.
	BaseUnit int `json:"base_unit,omitempty"`

	// BorderRadius is the border radius used for rounded corners.
	BorderRadius string `json:"border_radius,omitempty"`
}

// BrandingImages contains URLs for brand-related images.
type BrandingImages struct {
	// Logo is the URL of the primary logo image.
	Logo string `json:"logo,omitempty"`

	// Favicon is the URL of the favicon.
	Favicon string `json:"favicon,omitempty"`

	// OGImage is the URL of the Open Graph image (og:image meta tag).
	OGImage string `json:"og_image,omitempty"`
}

// Fetcher defines an interface for fetching web pages.
//
// Implementations can range from simple HTTP clients to sophisticated browser
// automation systems. The interface provides a unified way to fetch pages
// regardless of the underlying technology.
type Fetcher interface {
	// Fetch retrieves a web page according to the given request parameters
	// and returns the processed response. Returns an error if the fetch fails
	// or if any requested options are unsupported by this implementation.
	Fetch(ctx context.Context, request *Request) (*Response, error)
}
