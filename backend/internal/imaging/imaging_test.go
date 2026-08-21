package imaging

import "testing"

func TestSniffContentType(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
	if got := SniffContentType(png, "a.png"); got != "image/png" {
		t.Errorf("png 嗅探 = %q", got)
	}
	jpg := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0, 0, 0}
	if got := SniffContentType(jpg, "a.jpg"); got != "image/jpeg" {
		t.Errorf("jpeg 嗅探 = %q", got)
	}
	avif := []byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p', 'a', 'v', 'i', 'f'}
	if got := SniffContentType(avif, "a.avif"); got != "image/avif" {
		t.Errorf("avif 嗅探 = %q", got)
	}
	heic := []byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c'}
	if got := SniffContentType(heic, "a.heic"); got != "image/heic" {
		t.Errorf("heic 嗅探 = %q", got)
	}
	svg := []byte("<svg xmlns='http://www.w3.org/2000/svg'></svg>")
	if got := SniffContentType(svg, "a.svg"); got != "image/svg+xml" {
		t.Errorf("svg 嗅探 = %q", got)
	}
	unknown := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if got := SniffContentType(unknown, "file.xyz"); got != "application/octet-stream" {
		t.Errorf("未知文件嗅探 = %q", got)
	}
}

func TestExtFromContentType(t *testing.T) {
	cases := []struct {
		ct       string
		filename string
		want     string
	}{
		{"image/webp", "a.bin", ".webp"},
		{"application/octet-stream", "photo.jpg", ".jpg"},
		{"application/octet-stream", "noext", ".bin"},
		{"image/svg+xml", "", ".svg"},
	}
	for _, c := range cases {
		if got := ExtFromContentType(c.ct, c.filename); got != c.want {
			t.Errorf("ExtFromContentType(%q,%q) = %q, want %q", c.ct, c.filename, got, c.want)
		}
	}
}

func TestIsRaster(t *testing.T) {
	if !IsRaster("image/png") || !IsRaster("IMAGE/JPEG") {
		t.Error("常见位图格式应识别为 raster")
	}
	if IsRaster("image/svg+xml") || IsRaster("application/pdf") {
		t.Error("SVG/PDF 不应识别为 raster")
	}
}
