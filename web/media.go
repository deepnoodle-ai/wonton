package web

import (
	"net/url"
	"strings"
)

// MediaExtensions is a map of file extensions that are considered media files.
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

// IsMediaURL returns true if the URL appears to point to a media file.
func IsMediaURL(u *url.URL) bool {
	if idx := strings.LastIndex(u.Path, "."); idx > 0 {
		ext := strings.ToLower(u.Path[idx:])
		if MediaExtensions[ext] {
			return true
		}
	}
	return false
}
