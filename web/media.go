package web

import (
	"net/url"
	"path"
	"strings"
)

// ExtensionSet is a set of file extensions supporting case-insensitive
// lookups. Extensions are stored lowercase with a leading dot; Add and
// Contains normalize their inputs, so ".JPG", "jpg", and ".jpg" are all
// equivalent.
type ExtensionSet map[string]struct{}

// NewExtensionSet creates an ExtensionSet from the given extensions.
// Extensions may be given with or without a leading dot.
func NewExtensionSet(exts ...string) ExtensionSet {
	s := make(ExtensionSet, len(exts))
	s.Add(exts...)
	return s
}

// Add inserts extensions into the set. Extensions may be given with or
// without a leading dot.
func (s ExtensionSet) Add(exts ...string) {
	for _, ext := range exts {
		if ext = normalizeExtension(ext); ext != "." {
			s[ext] = struct{}{}
		}
	}
}

// Remove deletes extensions from the set.
func (s ExtensionSet) Remove(exts ...string) {
	for _, ext := range exts {
		delete(s, normalizeExtension(ext))
	}
}

// Contains reports whether the extension is in the set. The check is
// case-insensitive and accepts extensions with or without a leading dot.
func (s ExtensionSet) Contains(ext string) bool {
	if ext == "" {
		return false
	}
	_, ok := s[normalizeExtension(ext)]
	return ok
}

// ContainsURL reports whether the URL's path has a file extension that is
// in the set. Returns false for nil URLs and paths without an extension.
func (s ExtensionSet) ContainsURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	ext := path.Ext(u.Path)
	if ext == "" {
		return false
	}
	_, ok := s[strings.ToLower(ext)]
	return ok
}

// Clone returns a copy of the set. Useful for customizing a default set
// without mutating it:
//
//	exts := web.BinaryExtensions.Clone()
//	exts.Remove(".pdf") // crawl PDFs too
func (s ExtensionSet) Clone() ExtensionSet {
	clone := make(ExtensionSet, len(s))
	for ext := range s {
		clone[ext] = struct{}{}
	}
	return clone
}

func normalizeExtension(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

// BinaryExtensions is the default set of file extensions for URLs that
// point to file downloads and subresources rather than web pages — the
// things a crawler extracting text or following links typically skips:
//
//   - Images: .jpg, .png, .gif, .svg, .webp, .avif, .heic, .ico, etc.
//   - Video: .mp4, .webm, .mkv, .mov, .avi, .mpeg, etc.
//   - Audio: .mp3, .wav, .aac, .ogg, .opus, .flac, etc.
//   - Documents: .pdf, .doc(x), .xls(x), .ppt(x), .odt, .ods, .odp, .epub
//   - Archives: .zip, .tar, .gz, .tgz, .bz2, .xz, .zst, .rar, .7z, .iso
//   - Fonts: .ttf, .otf, .woff, .woff2, .eot
//   - Executables: .exe, .dmg, .apk, .deb, .rpm, .msi, .jar, etc.
//   - Page subresources: .css, .js, .mjs
//
// Some of these (.css, .js, .svg) are text rather than binary, but they are
// not pages, which is the distinction that matters when crawling. Use
// [IsBinaryURL] and [IsBinaryExtension] to check against this set, or Clone
// it to customize.
var BinaryExtensions = NewExtensionSet(
	".7z", ".aac", ".apk", ".avi", ".avif", ".bin", ".bmp", ".bz2",
	".css", ".deb", ".dmg", ".doc", ".docx", ".eot", ".epub", ".exe",
	".flac", ".flv", ".gif", ".gz", ".heic", ".heif", ".ico", ".img",
	".iso", ".jar", ".jpeg", ".jpg", ".js", ".m4a", ".m4v", ".mid",
	".midi", ".mjs", ".mkv", ".mov", ".mp3", ".mp4", ".mpeg", ".mpg",
	".msi", ".odp", ".ods", ".odt", ".ogg", ".ogv", ".opus", ".otf",
	".pdf", ".pkg", ".png", ".ppt", ".pptx", ".rar", ".rpm", ".svg",
	".swf", ".tar", ".tgz", ".tif", ".tiff", ".torrent", ".ttf", ".wav",
	".weba", ".webm", ".webp", ".wmv", ".woff", ".woff2", ".xls", ".xlsx",
	".xz", ".zip", ".zst",
)

// IsBinaryURL checks if a URL appears to point to a file download or page
// subresource rather than a web page, based on its file extension. The
// lookup is case-insensitive against [BinaryExtensions].
//
// This is useful for filtering out non-page resources when crawling or
// extracting links that point to HTML content.
//
// Example:
//
//	u, _ := url.Parse("https://example.com/image.jpg")
//	web.IsBinaryURL(u) // true
//
//	u, _ = url.Parse("https://example.com/page.html")
//	web.IsBinaryURL(u) // false
//
//	u, _ = url.Parse("https://example.com/VIDEO.MP4")
//	web.IsBinaryURL(u) // true (case-insensitive)
func IsBinaryURL(u *url.URL) bool {
	return BinaryExtensions.ContainsURL(u)
}

// IsBinaryExtension checks if a file extension is in [BinaryExtensions].
// The check is case-insensitive and accepts extensions with or without a
// leading dot.
//
// Example:
//
//	web.IsBinaryExtension(".jpg")  // true
//	web.IsBinaryExtension("JPG")   // true
//	web.IsBinaryExtension(".html") // false
func IsBinaryExtension(ext string) bool {
	return BinaryExtensions.Contains(ext)
}
