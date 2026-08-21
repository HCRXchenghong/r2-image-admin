package imaging

import (
	"bytes"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// Processor 图片解码与渲染能力。启用 libvips 时由 vips.go 实现，否则由 novips.go 兜底。
type Processor interface {
	Available() bool
	DecodeSize(data []byte) (width, height int, err error)
	Render(data []byte, width, height, quality int, format string) ([]byte, error)
}

var rasterTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/avif": true,
	"image/gif":  true,
	"image/tiff": true,
	"image/bmp":  true,
	"image/heic": true,
	"image/heif": true,
}

func IsRaster(contentType string) bool {
	return rasterTypes[strings.ToLower(strings.TrimSpace(contentType))]
}

// SniffContentType 通过魔数嗅探 + 扩展名兜底判断文件类型。
func SniffContentType(data []byte, filename string) string {
	if len(data) >= 12 && data[4] == 'f' && data[5] == 't' && data[6] == 'y' && data[7] == 'p' {
		switch string(data[8:12]) {
		case "avif", "avis":
			return "image/avif"
		case "heic", "heix", "hevc", "heim", "heis", "mif1":
			return "image/heic"
		}
	}
	if isSVG(data, filename) {
		return "image/svg+xml"
	}
	if ct := http.DetectContentType(data); isAcceptableDetectedType(ct) {
		return ct
	}
	if filename != "" {
		if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); ct != "" && isAcceptableDetectedType(ct) {
			return ct
		}
	}
	return "application/octet-stream"
}

// isAcceptableDetectedType 过滤 DetectContentType 的冷门误报（如 chemical/x-xyz），
// 只保留常见静态文件类型，其余走扩展名/octet-stream 兜底。
func isAcceptableDetectedType(ct string) bool {
	switch {
	case strings.HasPrefix(ct, "image/"),
		strings.HasPrefix(ct, "video/"),
		strings.HasPrefix(ct, "audio/"),
		strings.HasPrefix(ct, "text/"),
		strings.HasPrefix(ct, "application/vnd."):
		return true
	}
	switch ct {
	case "application/pdf", "application/zip", "application/gzip", "application/x-gzip",
		"application/json", "application/xml", "application/javascript",
		"application/x-7z-compressed", "application/x-rar-compressed", "application/x-tar",
		"application/x-bzip2", "application/msword":
		return true
	}
	return false
}

func isSVG(data []byte, filename string) bool {
	if strings.HasSuffix(strings.ToLower(filename), ".svg") {
		return true
	}
	head := strings.ToLower(string(bytes.TrimSpace(data)))
	if strings.HasPrefix(head, "<?xml") || strings.HasPrefix(head, "<svg") {
		return strings.Contains(head, "<svg")
	}
	return false
}

// ExtFromContentType 根据类型返回带点的扩展名。
func ExtFromContentType(contentType, filename string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/avif":
		return ".avif"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/tiff":
		return ".tiff"
	case "image/bmp":
		return ".bmp"
	case "image/heic", "image/heif":
		return ".heic"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	}
	if filename != "" {
		ext := strings.ToLower(filepath.Ext(filename))
		ext = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' {
				return r
			}
			return -1
		}, ext)
		if len(ext) > 1 && len(ext) <= 8 {
			return ext
		}
	}
	return ".bin"
}
