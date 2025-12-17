package web

import (
	"net/url"
	"strings"
)

// MediaExtensions is a map of file extensions that are considered media files.
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
// Use IsMediaURL to check if a URL points to a media file.
var MediaExtensions = map[string]bool{
	".7z":      true,
	".aac":     true,
	".apk":     true,
	".avi":     true,
	".bin":     true,
	".bmp":     true,
	".css":     true,
	".deb":     true,
	".dmg":     true,
	".doc":     true,
	".docx":    true,
	".eot":     true,
	".exe":     true,
	".flac":    true,
	".flv":     true,
	".gif":     true,
	".gz":      true,
	".ico":     true,
	".img":     true,
	".iso":     true,
	".jpeg":    true,
	".jpg":     true,
	".m4a":     true,
	".m4v":     true,
	".mkv":     true,
	".mov":     true,
	".mp3":     true,
	".mp4":     true,
	".msi":     true,
	".ogg":     true,
	".otf":     true,
	".pdf":     true,
	".pkg":     true,
	".png":     true,
	".ppt":     true,
	".pptx":    true,
	".rar":     true,
	".rpm":     true,
	".svg":     true,
	".tar":     true,
	".tif":     true,
	".tiff":    true,
	".torrent": true,
	".ttf":     true,
	".wav":     true,
	".webp":    true,
	".wmv":     true,
	".woff":    true,
	".woff2":   true,
	".xls":     true,
	".xlsx":    true,
	".zip":     true,
}

// IsMediaURL checks if a URL appears to point to a media file based on its
// file extension.
//
// The function extracts the file extension from the URL's path and performs
// a case-insensitive lookup against the MediaExtensions map. Returns true if
// the extension is found in the map.
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
	if idx := strings.LastIndex(u.Path, "."); idx > 0 {
		ext := strings.ToLower(u.Path[idx:])
		if MediaExtensions[ext] {
			return true
		}
	}
	return false
}
