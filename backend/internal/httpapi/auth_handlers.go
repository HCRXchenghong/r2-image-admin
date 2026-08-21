package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"r2-image-admin/backend/internal/auth"
	"r2-image-admin/backend/internal/store"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if !validUsername(req.Username) || len(req.Password) == 0 || len(req.Password) > 256 {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	key := loginRateKey(r, req.Username)
	if allowed, wait := s.loginLimiter.allowed(key, time.Now()); !allowed {
		seconds := int(wait.Round(time.Second).Seconds())
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		s.recordAudit(r, nil, "LOGIN", "/api/auth/login", "blocked")
		writeError(w, http.StatusTooManyRequests, "登录尝试过于频繁，请稍后再试")
		return
	}
	userID, err := s.db.Authenticate(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			s.loginLimiter.failed(key, time.Now())
			s.recordAudit(r, nil, "LOGIN", "/api/auth/login", "failure")
			writeError(w, http.StatusUnauthorized, "用户名或密码错误")
			return
		}
		s.recordAudit(r, nil, "LOGIN", "/api/auth/login", "failure")
		writeError(w, http.StatusInternalServerError, "登录失败")
		return
	}
	s.loginLimiter.succeeded(key)
	token, err := auth.Sign(s.cfg.JWTSecret, s.cfg.JWTTTL, userID, req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成令牌失败")
		return
	}
	s.recordAudit(r, &auth.Claims{UserID: userID, Username: req.Username}, "LOGIN", "/api/auth/login", "success")
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "username": req.Username})
}

func validUsername(value string) bool {
	if len(value) < 3 || len(value) > 64 {
		return false
	}
	for _, c := range value {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"username": claims.Username})
}
