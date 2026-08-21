package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"r2-image-admin/backend/internal/imaging"
	"r2-image-admin/backend/internal/keygen"
	"r2-image-admin/backend/internal/storage"
	"r2-image-admin/backend/internal/store"
)

// imageDTO 把数据库记录转成前端友好的结构，并拼好公开 URL。
func imageDTO(rec *store.Image, st storage.Storage) map[string]any {
	var variants []store.Variant
	_ = json.Unmarshal([]byte(rec.Variants), &variants)

	vs := make([]map[string]any, 0, len(variants))
	mainURL := ""
	thumbURL := ""
	for _, v := range variants {
		u := st.PublicURL(v.Key)
		vs = append(vs, map[string]any{
			"label":      v.Label,
			"key":        v.Key,
			"url":        u,
			"width":      v.Width,
			"height":     v.Height,
			"format":     v.Format,
			"size_bytes": v.SizeBytes,
		})
		if v.Label == "main" {
			mainURL = u
		} else if thumbURL == "" {
			thumbURL = u
		}
	}
	if thumbURL == "" {
		thumbURL = mainURL
	}
	if mainURL == "" {
		mainURL = st.PublicURL(rec.Key)
		thumbURL = mainURL
	}

	dto := map[string]any{
		"id":           rec.ID,
		"key":          rec.Key,
		"url":          mainURL,
		"thumb_url":    thumbURL,
		"name":         rec.Name,
		"category":     rec.Category,
		"content_type": rec.ContentType,
		"width":        rec.Width,
		"height":       rec.Height,
		"size_bytes":   rec.SizeBytes,
		"sha256":       rec.SHA256,
		"direct":       rec.Direct,
		"variants":     vs,
		"created_at":   rec.CreatedAt,
		"updated_at":   rec.UpdatedAt,
	}
	if rec.OriginalKey != "" {
		dto["original_url"] = st.PublicURL(rec.OriginalKey)
	}
	return dto
}

func imageKeys(rec *store.Image) []string {
	var variants []store.Variant
	_ = json.Unmarshal([]byte(rec.Variants), &variants)
	keys := make([]string, 0, len(variants)+2)
	seen := map[string]bool{}
	if rec.Key != "" {
		keys = append(keys, rec.Key)
		seen[rec.Key] = true
	}
	for _, v := range variants {
		if v.Key != "" && !seen[v.Key] {
			keys = append(keys, v.Key)
			seen[v.Key] = true
		}
	}
	if rec.OriginalKey != "" && !seen[rec.OriginalKey] {
		keys = append(keys, rec.OriginalKey)
	}
	return keys
}

func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 24
	}
	items, total, err := s.db.ListImages(store.ListFilter{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询失败："+err.Error())
		return
	}
	dtos := make([]map[string]any, 0, len(items))
	for i := range items {
		dtos = append(dtos, imageDTO(&items[i], s.storage))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":     dtos,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (s *Server) handleGetImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	img, err := s.db.GetImage(uint(id))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "图片不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	writeJSON(w, http.StatusOK, imageDTO(img, s.storage))
}

func (s *Server) handleDeleteImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	img, err := s.db.GetImage(uint(id))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "图片不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	if err := s.storage.Delete(r.Context(), imageKeys(img)); err != nil {
		writeError(w, http.StatusBadGateway, "删除存储对象失败："+err.Error())
		return
	}
	if err := s.db.DeleteImage(img.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "删除记录失败："+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleReplaceImage 替换图片内容：处理过的图保持原 URL 不变，老尺寸自动清理。
func (s *Server) handleReplaceImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	img, err := s.db.GetImage(uint(id))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "图片不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	data, filename, err := s.readUploadFile(r)
	if err != nil {
		writeUploadError(w, err)
		return
	}
	ct := imaging.SniffContentType(data, filename)
	if err := validateUploadContentType(ct); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 原图直传记录：直接覆盖原 key。
	if img.Direct {
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
		if err := s.storage.Put(r.Context(), storage.Object{Key: img.Key, ContentType: ct, Body: data}); err != nil {
			writeError(w, http.StatusBadGateway, "上传到存储失败："+err.Error())
			return
		}
		img.Name = filename
		img.ContentType = ct
		img.Width = width
		img.Height = height
		img.SizeBytes = int64(len(data))
		img.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		if err := s.db.UpdateImage(img); err != nil {
			writeError(w, http.StatusInternalServerError, "保存记录失败："+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, imageDTO(img, s.storage))
		return
	}

	if !s.proc.Available() {
		writeError(w, http.StatusNotImplemented, "服务器未启用图片处理（libvips）")
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

	// 先在内存里生成全部变体，再统一上传，避免替换中途失败留下半成品。
	prefix := img.Prefix
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

	for _, up := range uploads {
		if err := s.storage.Put(r.Context(), up); err != nil {
			writeError(w, http.StatusBadGateway, "上传到存储失败："+err.Error())
			return
		}
	}

	newKeys := make(map[string]bool, len(uploads))
	for _, up := range uploads {
		newKeys[up.Key] = true
	}
	var stale []string
	for _, k := range imageKeys(img) {
		if !newKeys[k] {
			stale = append(stale, k)
		}
	}
	if len(stale) > 0 {
		if err := s.storage.Delete(r.Context(), stale); err != nil {
			// 清理失败不阻断替换，只记录日志。
			slog.Warn("清理旧变体失败", "keys", stale, "err", err)
		}
	}

	img.Key = mainKey
	img.Name = filename
	img.ContentType = "image/" + mainFormat
	img.Width = origW
	img.Height = origH
	img.SizeBytes = totalBytes
	img.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
	img.Variants = mustJSON(variants)
	img.OriginalKey = originalKey
	img.Direct = false
	if err := s.db.UpdateImage(img); err != nil {
		writeError(w, http.StatusInternalServerError, "保存记录失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, imageDTO(img, s.storage))
}
