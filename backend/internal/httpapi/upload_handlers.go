package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"r2-image-admin/backend/internal/imaging"
	"r2-image-admin/backend/internal/keygen"
	"r2-image-admin/backend/internal/storage"
	"r2-image-admin/backend/internal/store"
)

const multipartMemory = 32 << 20

var errUploadTooLarge = errors.New("上传文件超过大小限制")

// readUploadFile 读取 multipart 中的 file 字段并做大小限制。
func (s *Server) readUploadFile(r *http.Request) ([]byte, string, error) {
	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, "", errUploadTooLarge
		}
		return nil, "", fmt.Errorf("表单解析失败，文件可能过大")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, "", fmt.Errorf("缺少 file 文件字段")
	}
	defer file.Close()
	maxBytes := s.cfg.MaxUploadMB << 20
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, "", errUploadTooLarge
		}
		return nil, "", fmt.Errorf("读取文件失败")
	}
	if int64(len(data)) > maxBytes {
		return nil, "", errUploadTooLarge
	}
	return data, safeUploadFilename(header.Filename), nil
}

func writeUploadError(w http.ResponseWriter, err error) {
	if errors.Is(err, errUploadTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "文件超过大小限制")
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}

func safeUploadFilename(filename string) string {
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	filename = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, strings.TrimSpace(filename))
	if filename == "" {
		return "upload"
	}
	if len(filename) > 240 {
		return filename[:240]
	}
	return filename
}

func validateUploadContentType(contentType string) error {
	if !imaging.IsRaster(contentType) {
		return fmt.Errorf("仅支持 JPEG、PNG、WebP、AVIF、GIF、TIFF、BMP、HEIC 等位图图片")
	}
	return nil
}

// handleUpload 智能上传：自动压缩并生成多尺寸 WebP/AVIF。
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	data, filename, err := s.readUploadFile(r)
	if err != nil {
		writeUploadError(w, err)
		return
	}
	category := keygen.Category(r.FormValue("category"))
	ct := imaging.SniffContentType(data, filename)
	if err := validateUploadContentType(ct); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !s.proc.Available() {
		writeError(w, http.StatusNotImplemented, "服务器未启用图片处理（libvips）：请使用 Docker 部署或加 -tags vips 编译；原图上传请使用 /api/images/direct")
		return
	}
	origW, origH, err := s.proc.DecodeSize(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无法解析图片，文件可能损坏或格式不受支持")
		return
	}
	if !validImageDimensions(origW, origH) {
		writeError(w, http.StatusBadRequest, "图片像素过大，最大支持 4000 万像素")
		return
	}

	prefix := category + "/" + keygen.NewID()
	mainFormat := s.cfg.ImgFormats[0]
	mainData, err := s.proc.Render(data, 0, 0, s.cfg.ImgQuality, mainFormat)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "图片处理失败："+err.Error())
		return
	}
	mainKey := keygen.MainKey(prefix, mainFormat)

	variants := []store.Variant{{Label: "main", Key: mainKey, Width: origW, Height: origH, Format: mainFormat, SizeBytes: int64(len(mainData))}}
	uploads := []storage.Object{{Key: mainKey, ContentType: "image/" + mainFormat, Body: mainData}}
	totalBytes := int64(len(mainData))

	for _, targetW := range s.cfg.ImgSizes {
		if targetW <= 0 || targetW >= origW {
			continue
		}
		targetH := int(math.Round(float64(targetW) * float64(origH) / float64(origW)))
		for _, f := range s.cfg.ImgFormats {
			out, err := s.proc.Render(data, targetW, targetH, s.cfg.ImgQuality, f)
			if err != nil {
				if f == "avif" {
					slog.Warn("AVIF 生成失败，已跳过", "err", err)
					continue
				}
				writeError(w, http.StatusInternalServerError, "生成 "+f+" 尺寸失败："+err.Error())
				return
			}
			k := keygen.SizeKey(prefix, targetW, f)
			uploads = append(uploads, storage.Object{Key: k, ContentType: "image/" + f, Body: out})
			variants = append(variants, store.Variant{Label: fmt.Sprintf("%d", targetW), Key: k, Width: targetW, Height: targetH, Format: f, SizeBytes: int64(len(out))})
			totalBytes += int64(len(out))
		}
	}

	originalKey := ""
	if s.cfg.KeepOriginal {
		ext := imaging.ExtFromContentType(ct, filename)
		originalKey = keygen.OriginalKey(prefix, ext)
		uploads = append(uploads, storage.Object{Key: originalKey, ContentType: ct, Body: data})
		totalBytes += int64(len(data))
	}

	uploaded := make([]string, 0, len(uploads))
	for _, up := range uploads {
		if err := s.storage.Put(r.Context(), up); err != nil {
			_ = s.storage.Delete(r.Context(), uploaded)
			writeError(w, http.StatusBadGateway, "上传到存储失败："+err.Error())
			return
		}
		uploaded = append(uploaded, up.Key)
	}

	rec := &store.Image{
		Key:         mainKey,
		Prefix:      prefix,
		Name:        filename,
		Category:    category,
		ContentType: "image/" + mainFormat,
		Width:       origW,
		Height:      origH,
		SizeBytes:   totalBytes,
		SHA256:      fmt.Sprintf("%x", sha256.Sum256(data)),
		Variants:    mustJSON(variants),
		OriginalKey: originalKey,
		Direct:      false,
	}
	if err := s.db.CreateImage(rec); err != nil {
		_ = s.storage.Delete(r.Context(), uploaded)
		writeError(w, http.StatusInternalServerError, "保存记录失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, imageDTO(rec, s.storage))
}

// handleDirectUpload 原图上传：不做任何处理，也支持 PDF/ZIP 等静态文件。
func (s *Server) handleDirectUpload(w http.ResponseWriter, r *http.Request) {
	data, filename, err := s.readUploadFile(r)
	if err != nil {
		writeUploadError(w, err)
		return
	}
	category := keygen.Category(r.FormValue("category"))
	ct := imaging.SniffContentType(data, filename)
	if err := validateUploadContentType(ct); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.saveDirect(r, w, data, filename, category, ct)
}

func (s *Server) saveDirect(r *http.Request, w http.ResponseWriter, data []byte, filename, category, ct string) {
	prefix := category + "/" + keygen.NewID()
	ext := imaging.ExtFromContentType(ct, filename)
	key := keygen.OriginalKey(prefix, ext)

	width, height := 0, 0
	if imaging.IsRaster(ct) && s.proc.Available() {
		if ww, hh, err := s.proc.DecodeSize(data); err == nil {
			if !validImageDimensions(ww, hh) {
				writeError(w, http.StatusBadRequest, "图片像素过大，最大支持 4000 万像素")
				return
			}
			width, height = ww, hh
		}
	}
	if err := s.storage.Put(r.Context(), storage.Object{Key: key, ContentType: ct, Body: data}); err != nil {
		writeError(w, http.StatusBadGateway, "上传到存储失败："+err.Error())
		return
	}
	rec := &store.Image{
		Key:         key,
		Prefix:      prefix,
		Name:        filename,
		Category:    category,
		ContentType: ct,
		Width:       width,
		Height:      height,
		SizeBytes:   int64(len(data)),
		SHA256:      fmt.Sprintf("%x", sha256.Sum256(data)),
		Direct:      true,
	}
	if err := s.db.CreateImage(rec); err != nil {
		_ = s.storage.Delete(r.Context(), []string{key})
		writeError(w, http.StatusInternalServerError, "保存记录失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, imageDTO(rec, s.storage))
}

func validImageDimensions(width, height int) bool {
	return width > 0 && height > 0 && int64(width)*int64(height) <= 40_000_000
}

type presignRequest struct {
	Category    string `json:"category"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Key         string `json:"key"`
}

// handlePresign 生成浏览器直传 R2 的预签名链接（需在 R2 桶配置 CORS）。
func (s *Server) handlePresign(w http.ResponseWriter, r *http.Request) {
	var req presignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Filename == "" {
		writeError(w, http.StatusBadRequest, "请求格式错误，需要 filename")
		return
	}
	if s.storage.Driver() != "r2" {
		writeError(w, http.StatusBadRequest, "仅 R2 存储支持直传链接")
		return
	}
	ct := strings.ToLower(strings.TrimSpace(req.ContentType))
	if ct == "" {
		ct = mime.TypeByExtension(filepath.Ext(req.Filename))
	}
	if err := validateUploadContentType(ct); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	category := keygen.Category(req.Category)
	ext := imaging.ExtFromContentType(ct, req.Filename)
	key := keygen.OriginalKey(category+"/"+keygen.NewID(), ext)
	ttl := 15 * time.Minute
	uploadURL, err := s.storage.PresignPut(r.Context(), key, ct, ttl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成直传链接失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key":          key,
		"upload_url":   uploadURL,
		"content_type": ct,
		"expires_in":   int(ttl.Seconds()),
		"public_url":   s.storage.PublicURL(key),
	})
}

// handlePresignConfirm 直传完成后登记记录。
func (s *Server) handlePresignConfirm(w http.ResponseWriter, r *http.Request) {
	var req presignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		writeError(w, http.StatusBadRequest, "请求格式错误，需要 key")
		return
	}
	obj, err := s.storage.Get(r.Context(), req.Key)
	if err != nil {
		writeError(w, http.StatusBadGateway, "读取已上传文件失败："+err.Error())
		return
	}
	category := keygen.Category(req.Category)
	ct := imaging.SniffContentType(obj.Body, req.Filename)
	if err := validateUploadContentType(ct); err != nil {
		_ = s.storage.Delete(r.Context(), []string{req.Key})
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := req.Filename
	if name == "" {
		name = filepath.Base(req.Key)
	}
	prefix := strings.TrimSuffix(req.Key, filepath.Ext(req.Key))
	prefix = strings.TrimSuffix(prefix, "/original")
	width, height := 0, 0
	if imaging.IsRaster(ct) && s.proc.Available() {
		if ww, hh, err := s.proc.DecodeSize(obj.Body); err == nil {
			if !validImageDimensions(ww, hh) {
				_ = s.storage.Delete(r.Context(), []string{req.Key})
				writeError(w, http.StatusBadRequest, "图片像素过大，最大支持 4000 万像素")
				return
			}
			width, height = ww, hh
		}
	}
	rec := &store.Image{
		Key:         req.Key,
		Prefix:      prefix,
		Name:        name,
		Category:    category,
		ContentType: ct,
		Width:       width,
		Height:      height,
		SizeBytes:   int64(len(obj.Body)),
		SHA256:      fmt.Sprintf("%x", sha256.Sum256(obj.Body)),
		Direct:      true,
	}
	if err := s.db.CreateImage(rec); err != nil {
		writeError(w, http.StatusInternalServerError, "保存记录失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, imageDTO(rec, s.storage))
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
