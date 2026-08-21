package httpapi

import (
	"net/http"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ok := true
	dbStatus := "up"
	if err := s.db.Ping(); err != nil {
		ok = false
		dbStatus = "down: " + err.Error()
	}
	code := http.StatusOK
	if !ok {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{
		"ok":         ok,
		"db":         dbStatus,
		"storage":    s.storage.Driver(),
		"processing": s.proc.Available(),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.db.Stats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "统计查询失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Categories()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "分类查询失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": rows})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"storage_driver":       s.cfg.StorageDriver,
		"bucket":               s.cfg.R2Bucket,
		"public_base_url":      s.cfg.PublicBaseURL,
		"sizes":                s.cfg.ImgSizes,
		"formats":              s.cfg.ImgFormats,
		"quality":              s.cfg.ImgQuality,
		"keep_original":        s.cfg.KeepOriginal,
		"max_upload_mb":        s.cfg.MaxUploadMB,
		"db_driver":            s.cfg.DBDriver,
		"processing":           s.proc.Available(),
		"ai_image_api_url":     s.cfg.AI_IMAGE_API_URL,
		"ai_image_model":       s.cfg.AI_IMAGE_MODEL,
		"ai_image_configured":  s.cfg.AI_IMAGE_API_KEY != "",
		"cors_allowed_origins": s.cfg.CorsAllowedOrigins,
		"jwt_ttl_hours":        int(s.cfg.JWTTTL.Hours()),
		"audit_retention_days": s.cfg.AuditRetentionDays,
		"app_env":              s.cfg.AppEnv,
		"auto_restart":         s.cfg.AutoRestart,
	})
}

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	items, total, err := s.db.ListAudit(page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询审计日志失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items, "total": total, "page": page, "page_size": pageSize,
	})
}

// handleR2Guide 返回 R2 接入说明。该内容通过受保护 API 提供，未登录用户无法读取。
func (s *Server) handleR2Guide(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"title":       "Cloudflare R2 接入指南",
		"subtitle":    "按下面步骤创建 R2 存储、最小权限 API Token 与图片公开域名，再将信息填写到本后台。",
		"console_url": "https://dash.cloudflare.com/",
		"steps": []map[string]any{
			{
				"number":      1,
				"title":       "创建或选择 Bucket",
				"path":        "Cloudflare 控制台 → R2 对象存储 → Overview",
				"description": "点击 Create bucket 创建桶，或选择已有桶。桶名就是后台要填写的 R2 Bucket。建议使用专用桶，不要与其他业务混用。",
				"tips":        []string{"Bucket 名称创建后不可直接修改", "生产环境建议按业务和环境分桶，例如 prod-images"},
			},
			{
				"number":      2,
				"title":       "获取 Account ID",
				"path":        "Cloudflare 控制台任意账户页面 → 右侧栏 Account ID",
				"description": "复制当前 Cloudflare 账户的 Account ID，填写到后台的 R2 Account ID。它不是邮箱、Zone ID 或 Bucket 名称。",
				"tips":        []string{"确认当前账户与 Bucket 所在账户一致", "不要把 Account ID 当作 Access Key"},
			},
			{
				"number":      3,
				"title":       "创建 R2 API Token",
				"path":        "R2 对象存储 → Manage R2 API Tokens → Create API Token",
				"description": "创建仅供此图床使用的 API Token，权限选择 Object Read & Write，并限定到本次使用的 Bucket。创建后复制 Access Key ID 和 Secret Access Key；Secret 只显示一次。",
				"tips":        []string{"不要使用 Cloudflare 全局 API Key", "Secret Access Key 不要发送到聊天、截图或提交到 Git"},
			},
			{
				"number":      4,
				"title":       "配置图片公开域名",
				"path":        "目标 Bucket → Settings → Custom Domains",
				"description": "绑定自有子域名，例如 img.example.com，并等待 DNS 与证书生效。将完整 HTTPS 地址填写到公开域名；不要填 Bucket 的 S3 API Endpoint。",
				"tips":        []string{"推荐使用 Custom Domain，便于长期稳定访问", "生产环境必须是 https:// 开头，末尾无需加 /"},
			},
		},
		"fields": []map[string]string{
			{"field": "R2 Account ID", "where": "Cloudflare 控制台右侧栏的 Account ID", "example": "32 位账户标识", "note": "必填，不是 Zone ID"},
			{"field": "Access Key ID", "where": "Manage R2 API Tokens 创建后显示", "example": "R2 API Token 的 Access Key ID", "note": "必填，建议最小权限"},
			{"field": "Secret Access Key", "where": "与 Access Key ID 同时显示，仅一次", "example": "R2 API Token 的 Secret Access Key", "note": "必填；留空表示不修改已保存值"},
			{"field": "Bucket", "where": "R2 → Overview 的 Bucket 列表", "example": "site-images", "note": "必填，名称须完全一致"},
			{"field": "公开域名", "where": "Bucket → Settings → Custom Domains", "example": "https://img.example.com", "note": "必填，使用 HTTPS，不填 S3 Endpoint"},
		},
		"cors": map[string]any{
			"description": "只有使用“预签名直传”时才需要配置。普通后台上传由服务端完成，不依赖 R2 CORS。",
			"example":     "[\n  {\n    \"AllowedOrigins\": [\"https://admin.example.com\"],\n    \"AllowedMethods\": [\"PUT\"],\n    \"AllowedHeaders\": [\"Content-Type\"],\n    \"ExposeHeaders\": [\"ETag\"],\n    \"MaxAgeSeconds\": 300\n  }\n]",
		},
		"verify": []string{
			"在本后台选择 Cloudflare R2，填入以上五项并保存。",
			"上传一张测试图片，确认图片管理出现记录。",
			"在无登录状态的浏览器窗口打开公开图片 URL，确认能访问且为 HTTPS。",
			"测试替换和删除，确认 R2 Bucket 中对象同步变化。",
		},
	})
}

// handleResources 返回服务器运行资源与磁盘占用信息。
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	var diskTotal, diskFree uint64
	if dir, ok := s.storage.(interface{ Dir() string }); ok {
		var st syscall.Statfs_t
		if err := syscall.Statfs(dir.Dir(), &st); err == nil {
			bsize := uint64(st.Bsize)
			diskTotal = st.Blocks * bsize
			diskFree = st.Bavail * bsize
		}
	}
	diskUsed := uint64(0)
	if diskTotal > diskFree {
		diskUsed = diskTotal - diskFree
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"uptime_seconds": int64(time.Since(s.started).Seconds()),
		"go_version":     runtime.Version(),
		"num_cpu":        runtime.NumCPU(),
		"num_goroutine":  runtime.NumGoroutine(),
		"mem_alloc":      ms.Alloc,
		"mem_sys":        ms.Sys,
		"num_gc":         ms.NumGC,
		"disk_total":     diskTotal,
		"disk_free":      diskFree,
		"disk_used":      diskUsed,
	})
}
