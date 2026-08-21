package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"r2-image-admin/backend/internal/config"
)

// configUpdateRequest 设置页提交的配置项，字段为 nil 表示不修改。
type configUpdateRequest struct {
	StorageDriver *string `json:"storage_driver"`
	R2AccountID   *string `json:"r2_account_id"`
	R2AccessKey   *string `json:"r2_access_key_id"`
	R2SecretKey   *string `json:"r2_secret_access_key"`
	R2Bucket      *string `json:"r2_bucket"`
	PublicBaseURL *string `json:"public_base_url"`
	AdminPassword *string `json:"admin_password"`
	JWTSecret     *string `json:"jwt_secret"`
	ImgSizes      *string `json:"img_sizes"`
	ImgFormats    *string `json:"img_formats"`
	ImgQuality    *int    `json:"img_quality"`
	KeepOriginal  *bool   `json:"img_keep_original"`
	MaxUploadMB   *int64  `json:"upload_max_mb"`
	AI_API_URL    *string `json:"ai_image_api_url"`
	AI_API_KEY    *string `json:"ai_image_api_key"`
	AI_MODEL      *string `json:"ai_image_model"`
	CorsOrigins   *string `json:"cors_allowed_origins"`
	AutoRestart   *bool   `json:"auto_restart"`
}

// handleUpdateConfig 保存配置到 .env 文件，重启服务后生效。
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req configUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	updates := map[string]string{}
	passwordChanged := false
	if req.StorageDriver != nil {
		updates["STORAGE_DRIVER"] = *req.StorageDriver
	}
	if req.R2AccountID != nil {
		updates["R2_ACCOUNT_ID"] = *req.R2AccountID
	}
	if req.R2AccessKey != nil {
		updates["R2_ACCESS_KEY_ID"] = *req.R2AccessKey
	}
	if req.R2SecretKey != nil {
		updates["R2_SECRET_ACCESS_KEY"] = *req.R2SecretKey
	}
	if req.R2Bucket != nil {
		updates["R2_BUCKET"] = *req.R2Bucket
	}
	if req.PublicBaseURL != nil {
		updates["PUBLIC_BASE_URL"] = *req.PublicBaseURL
	}
	if req.AdminPassword != nil && *req.AdminPassword != "" {
		if err := config.ValidateAdminPassword(*req.AdminPassword); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		updates["ADMIN_PASSWORD"] = *req.AdminPassword
		passwordChanged = true
	}
	if req.JWTSecret != nil && *req.JWTSecret != "" {
		if err := config.ValidateJWTSecret(*req.JWTSecret); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		updates["JWT_SECRET"] = *req.JWTSecret
	}
	if passwordChanged && updates["JWT_SECRET"] == "" {
		secret, err := newJWTSecret()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "生成会话密钥失败")
			return
		}
		updates["JWT_SECRET"] = secret
	}
	if req.ImgSizes != nil {
		updates["IMG_SIZES"] = *req.ImgSizes
	}
	if req.ImgFormats != nil {
		updates["IMG_FORMATS"] = *req.ImgFormats
	}
	if req.ImgQuality != nil {
		updates["IMG_QUALITY"] = strconv.Itoa(*req.ImgQuality)
	}
	if req.KeepOriginal != nil {
		updates["IMG_KEEP_ORIGINAL"] = strconv.FormatBool(*req.KeepOriginal)
	}
	if req.MaxUploadMB != nil {
		updates["UPLOAD_MAX_MB"] = strconv.FormatInt(*req.MaxUploadMB, 10)
	}
	if req.AI_API_URL != nil {
		updates["AI_IMAGE_API_URL"] = *req.AI_API_URL
	}
	if req.AI_API_KEY != nil && *req.AI_API_KEY != "" {
		updates["AI_IMAGE_API_KEY"] = *req.AI_API_KEY
	}
	if req.AI_MODEL != nil {
		updates["AI_IMAGE_MODEL"] = *req.AI_MODEL
	}
	if req.CorsOrigins != nil {
		updates["CORS_ALLOWED_ORIGINS"] = *req.CorsOrigins
	}
	if req.AutoRestart != nil {
		updates["AUTO_RESTART"] = strconv.FormatBool(*req.AutoRestart)
	}

	newCfg, err := config.LoadWith(updates)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := config.WriteEnvFile(config.ResolveEnvFile(), updates); err != nil {
		writeError(w, http.StatusInternalServerError, "写入配置文件失败："+err.Error())
		return
	}
	if passwordChanged {
		claims := claimsFrom(r)
		if claims == nil {
			writeError(w, http.StatusUnauthorized, "未登录")
			return
		}
		if err := s.db.UpdatePassword(claims.UserID, *req.AdminPassword); err != nil {
			writeError(w, http.StatusInternalServerError, "更新管理员密码失败")
			return
		}
	}

	restartNow := newCfg.AutoRestart || passwordChanged
	restartMsg := "配置已保存，服务将自动重启后生效"
	if passwordChanged {
		restartMsg = "密码已更新，所有会话将失效，服务正在重启"
	} else if !restartNow {
		restartMsg = "配置已保存，需手动重启服务后生效"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"message":         restartMsg,
		"storage_driver":  newCfg.StorageDriver,
		"bucket":          newCfg.R2Bucket,
		"public_base_url": newCfg.PublicBaseURL,
		"sizes":           newCfg.ImgSizes,
		"formats":         newCfg.ImgFormats,
		"quality":         newCfg.ImgQuality,
		"keep_original":   newCfg.KeepOriginal,
		"max_upload_mb":   newCfg.MaxUploadMB,
		"db_driver":       newCfg.DBDriver,
		"processing":      s.proc.Available(),
		"auto_restart":    restartNow,
	})

	if restartNow {
		go func() {
			time.Sleep(600 * time.Millisecond)
			if s.restart != nil {
				s.restart(config.MergeEnv(updates))
				return
			}
			os.Exit(0)
		}()
	}
}

func newJWTSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
