package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"r2-image-admin/backend/internal/auth"
	"r2-image-admin/backend/internal/store"
)

const (
	loginWindow      = 15 * time.Minute
	loginBlockPeriod = 15 * time.Minute
	maxLoginFailures = 5
	jsonBodyLimit    = int64(1 << 20)
)

type loginAttempt struct {
	failures     int
	windowStart  time.Time
	blockedUntil time.Time
}

type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{attempts: make(map[string]loginAttempt)}
}

func (l *loginRateLimiter) allowed(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.attempts[key]
	if now.Before(a.blockedUntil) {
		return false, a.blockedUntil.Sub(now)
	}
	if !a.windowStart.IsZero() && now.Sub(a.windowStart) > loginWindow {
		delete(l.attempts, key)
	}
	return true, 0
}

func (l *loginRateLimiter) failed(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.attempts[key]
	if a.windowStart.IsZero() || now.Sub(a.windowStart) > loginWindow {
		a = loginAttempt{windowStart: now}
	}
	a.failures++
	if a.failures >= maxLoginFailures {
		a.blockedUntil = now.Add(loginBlockPeriod)
		a.failures = 0
		a.windowStart = time.Time{}
	}
	l.attempts[key] = a
	if len(l.attempts) > 4096 {
		for k, v := range l.attempts {
			if now.After(v.blockedUntil) && (v.windowStart.IsZero() || now.Sub(v.windowStart) > loginWindow) {
				delete(l.attempts, k)
			}
		}
	}
}

func (l *loginRateLimiter) succeeded(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func loginRateKey(r *http.Request, username string) string {
	return clientIP(r) + "\x00" + strings.ToLower(strings.TrimSpace(username))
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.Trim(strings.TrimSpace(r.RemoteAddr), "[]")
}

func safeAuditText(value string, max int) string {
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
	if len(value) > max {
		return value[:max]
	}
	return value
}

func (s *Server) recordAudit(r *http.Request, claims *auth.Claims, action, target, outcome string) {
	username := "anonymous"
	var userID uint
	if claims != nil {
		username = claims.Username
		userID = claims.UserID
	}
	_ = s.db.CreateAudit(&store.AuditLog{
		UserID:    userID,
		Username:  safeAuditText(username, 64),
		RemoteIP:  safeAuditText(clientIP(r), 64),
		Action:    safeAuditText(action, 96),
		Target:    safeAuditText(target, 255),
		Outcome:   outcome,
		RequestID: safeAuditText(middleware.GetReqID(r.Context()), 64),
	})
}

func (s *Server) auditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		outcome := "success"
		if ww.Status() >= http.StatusBadRequest {
			outcome = "failure"
		}
		s.recordAudit(r, claimsFrom(r), r.Method, r.URL.Path, outcome)
	})
}

func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; form-action 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; connect-src 'self'; font-src 'self' data:")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
			next.ServeHTTP(w, r)
			return
		}
		limit := jsonBodyLimit
		if r.URL.Path == "/api/images" || r.URL.Path == "/api/images/direct" || strings.HasPrefix(r.URL.Path, "/api/images/") {
			limit = (s.cfg.MaxUploadMB << 20) + (1 << 20)
		}
		if r.ContentLength > limit {
			writeError(w, http.StatusRequestEntityTooLarge, "请求体超过允许大小")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		allowed := false
		for _, configured := range s.cfg.CorsAllowedOrigins {
			if origin == configured {
				allowed = true
				break
			}
		}
		if !allowed {
			if r.Method == http.MethodOptions {
				writeError(w, http.StatusForbidden, "不允许的跨域来源")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
