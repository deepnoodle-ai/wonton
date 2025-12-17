// Package webfetch provides interfaces and implementations for fetching web pages.
//
// It includes a simple HTTP fetcher for direct requests and a client for
// fetching through a remote proxy service.
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
type Request struct {
	URL             string            `json:"url"`
	OnlyMainContent bool              `json:"only_main_content,omitempty"`
	IncludeTags     []string          `json:"include_tags,omitempty"`
	ExcludeTags     []string          `json:"exclude_tags,omitempty"`
	ExcludeFilters  []ElementFilter   `json:"exclude_filters,omitempty"`
	MaxAge          int               `json:"max_age,omitempty"`  // milliseconds
	Timeout         int               `json:"timeout,omitempty"`  // milliseconds
	WaitFor         int               `json:"wait_for,omitempty"` // milliseconds
	Fetcher         string            `json:"fetcher,omitempty"`
	Mobile          bool              `json:"mobile,omitempty"`
	Prettify        bool              `json:"prettify,omitempty"`
	Formats         []string          `json:"formats,omitempty"`
	Actions         []Action          `json:"actions,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	StorageState    map[string]any    `json:"storage_state,omitempty"`
}

// Response defines the payload for fetch responses.
type Response struct {
	URL          string            `json:"url"`
	StatusCode   int               `json:"status_code"`
	Headers      map[string]string `json:"headers"`
	HTML         string            `json:"html,omitempty"`
	RawHTML      string            `json:"raw_html,omitempty"`
	Markdown     string            `json:"markdown,omitempty"`
	Summary      string            `json:"summary,omitempty"`
	Screenshot   string            `json:"screenshot,omitempty"`
	JSON         map[string]any    `json:"json,omitempty"`
	Branding     *BrandingProfile  `json:"branding,omitempty"`
	Images       []Image           `json:"images,omitempty"`
	Error        string            `json:"error,omitempty"`
	Metadata     Metadata          `json:"metadata,omitempty"`
	Links        []Link            `json:"links,omitempty"`
	StorageState map[string]any    `json:"storage_state,omitempty"`
	Timestamp    time.Time         `json:"timestamp,omitzero"`
}

// BrandingProfile contains brand identity information extracted from a page.
type BrandingProfile struct {
	ColorScheme string           `json:"color_scheme,omitempty"`
	Logo        string           `json:"logo,omitempty"`
	Colors      *BrandingColors  `json:"colors,omitempty"`
	Fonts       []BrandingFont   `json:"fonts,omitempty"`
	Typography  *BrandingTypo    `json:"typography,omitempty"`
	Spacing     *BrandingSpacing `json:"spacing,omitempty"`
	Images      *BrandingImages  `json:"images,omitempty"`
}

// BrandingColors contains brand color information.
type BrandingColors struct {
	Primary       string `json:"primary,omitempty"`
	Secondary     string `json:"secondary,omitempty"`
	Accent        string `json:"accent,omitempty"`
	Background    string `json:"background,omitempty"`
	TextPrimary   string `json:"text_primary,omitempty"`
	TextSecondary string `json:"text_secondary,omitempty"`
}

// BrandingFont represents a font used on the page.
type BrandingFont struct {
	Family string `json:"family,omitempty"`
}

// BrandingTypo contains typography information.
type BrandingTypo struct {
	FontFamilies *TypoFamilies     `json:"font_families,omitempty"`
	FontSizes    map[string]string `json:"font_sizes,omitempty"`
	FontWeights  map[string]int    `json:"font_weights,omitempty"`
}

// TypoFamilies contains font family assignments.
type TypoFamilies struct {
	Primary string `json:"primary,omitempty"`
	Heading string `json:"heading,omitempty"`
	Code    string `json:"code,omitempty"`
}

// BrandingSpacing contains spacing information.
type BrandingSpacing struct {
	BaseUnit     int    `json:"base_unit,omitempty"`
	BorderRadius string `json:"border_radius,omitempty"`
}

// BrandingImages contains brand image URLs.
type BrandingImages struct {
	Logo    string `json:"logo,omitempty"`
	Favicon string `json:"favicon,omitempty"`
	OGImage string `json:"og_image,omitempty"`
}

// Fetcher defines an interface for fetching pages.
type Fetcher interface {
	// Fetch a webpage and return the response.
	Fetch(ctx context.Context, request *Request) (*Response, error)
}
