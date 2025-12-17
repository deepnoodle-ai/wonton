package web

import (
	"net/url"
	"strings"
)

// mediaExtensions is the set of file extensions considered media files.
//
// This includes common file types that web crawlers typically want to skip when
// extracting text content or following links:
//   - Images: .jpg, .png, .gif, .svg, .webp, .bmp, .ico, etc.
//   - Videos: .mp4, .avi, .mov, .mkv, .flv, etc.
//   - Audio: .mp3, .wav, .aac, .ogg, .flac, etc.
//   - Documents: .pdf, .doc, .docx, .xls, .xlsx, .ppt, .pptx
//   - Archives: .zip, .tar, .gz, .rar, .7z, .iso
//   - Fonts: .ttf, .otf, .woff, .woff2, .eot
//   - Executables: .exe, .dmg, .apk, .deb, .rpm, .msi, etc.
//   - Other: .css, .torrent
//
// Extensions are stored in lowercase with the leading dot included.
// Use IsMediaURL to check if a URL points to a media file, or
// IsMediaExtension to check a file extension directly.
var mediaExtensions = map[string]struct{}{
	".7z":      {},
	".aac":     {},
	".apk":     {},
	".avi":     {},
	".bin":     {},
	".bmp":     {},
	".css":     {},
	".deb":     {},
	".dmg":     {},
	".doc":     {},
	".docx":    {},
	".eot":     {},
	".exe":     {},
	".flac":    {},
	".flv":     {},
	".gif":     {},
	".gz":      {},
	".ico":     {},
	".img":     {},
	".iso":     {},
	".jpeg":    {},
	".jpg":     {},
	".m4a":     {},
	".m4v":     {},
	".mkv":     {},
	".mov":     {},
	".mp3":     {},
	".mp4":     {},
	".msi":     {},
	".ogg":     {},
	".otf":     {},
	".pdf":     {},
	".pkg":     {},
	".png":     {},
	".ppt":     {},
	".pptx":    {},
	".rar":     {},
	".rpm":     {},
	".svg":     {},
	".tar":     {},
	".tif":     {},
	".tiff":    {},
	".torrent": {},
	".ttf":     {},
	".wav":     {},
	".webp":    {},
	".wmv":     {},
	".woff":    {},
	".woff2":   {},
	".xls":     {},
	".xlsx":    {},
	".zip":     {},
}

// IsMediaURL checks if a URL appears to point to a media file based on its
// file extension.
//
// The function extracts the file extension from the URL's path and performs
// a case-insensitive lookup against the known media extensions. Returns true if
// the extension is recognized as a media file type.
//
// This is useful for filtering out media files when crawling web pages or
// extracting links that point to HTML content.
//
// Example:
//
//	url, _ := url.Parse("https://example.com/image.jpg")
//	web.IsMediaURL(url) // true
//
//	url, _ = url.Parse("https://example.com/page.html")
//	web.IsMediaURL(url) // false
//
//	url, _ = url.Parse("https://example.com/VIDEO.MP4")
//	web.IsMediaURL(url) // true (case-insensitive)
func IsMediaURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	if idx := strings.LastIndex(u.Path, "."); idx > 0 {
		ext := strings.ToLower(u.Path[idx:])
		_, ok := mediaExtensions[ext]
		return ok
	}
	return false
}

// IsMediaExtension checks if a file extension is considered a media file extension.
// The extension should include the leading dot (e.g., ".jpg", ".mp4").
// The check is case-insensitive.
//
// Example:
//
//	web.IsMediaExtension(".jpg")  // true
//	web.IsMediaExtension(".JPG")  // true
//	web.IsMediaExtension(".html") // false
//	web.IsMediaExtension("jpg")   // false (missing dot)
func IsMediaExtension(ext string) bool {
	_, ok := mediaExtensions[strings.ToLower(ext)]
	return ok
}
