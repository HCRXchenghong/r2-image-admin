package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"r2-image-admin/backend/internal/auth"
	"r2-image-admin/backend/internal/config"
	"r2-image-admin/backend/internal/imaging"
	"r2-image-admin/backend/internal/storage"
	"r2-image-admin/backend/internal/store"
	"r2-image-admin/backend/web"
)

// Server 汇总各依赖并组装路由。
type Server struct {
	cfg          *config.Config
	db           *store.Store
	storage      storage.Storage
	proc         imaging.Processor
	started      time.Time
	restart      func(env []string)
	loginLimiter *loginRateLimiter
}

func New(cfg *config.Config, st *store.Store, obj storage.Storage, proc imaging.Processor) *Server {
	return &Server{cfg: cfg, db: st, storage: obj, proc: proc, started: time.Now(), loginLimiter: newLoginRateLimiter()}
}

// SetRestartFunc 注入“保存配置后自动重启”的实现。
func (s *Server) SetRestartFunc(fn func(env []string)) {
	s.restart = fn
}

type ctxKey struct{}

var claimsKey = ctxKey{}

func claimsFrom(r *http.Request) *auth.Claims {
	c, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return c
}

// Handler 组装完整 HTTP 路由。
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(s.securityHeadersMiddleware)
	r.Use(s.requestLimitMiddleware)
	r.Use(s.logMiddleware)
	r.Use(s.corsMiddleware)

	r.Get("/api/health", s.handleHealth)
	r.Post("/api/auth/login", s.handleLogin)

	r.Group(func(prot chi.Router) {
		prot.Use(s.authMiddleware)
		prot.Use(s.auditMiddleware)
		prot.Get("/api/auth/me", s.handleMe)
		prot.Post("/api/images", s.handleUpload)
		prot.Post("/api/images/direct", s.handleDirectUpload)
		prot.Get("/api/images", s.handleListImages)
		prot.Get("/api/images/categories", s.handleCategories)
		prot.Get("/api/images/{id}", s.handleGetImage)
		prot.Put("/api/images/{id}", s.handleReplaceImage)
		prot.Delete("/api/images/{id}", s.handleDeleteImage)
		prot.Post("/api/presign", s.handlePresign)
		prot.Post("/api/presign/confirm", s.handlePresignConfirm)
		prot.Post("/api/ai/generate", s.handleAIGenerate)
		prot.Post("/api/ai/models/sync", s.handleAISyncModels)
		prot.Get("/api/stats", s.handleStats)
		prot.Get("/api/config", s.handleConfig)
		prot.Get("/api/resources", s.handleResources)
		prot.Get("/api/audit-logs", s.handleListAudit)
		prot.Get("/api/guides/r2", s.handleR2Guide)
		prot.Put("/api/config", s.handleUpdateConfig)
	})

	if s.cfg.StorageDriver == "local" {
		if l, ok := s.storage.(interface{ Dir() string }); ok {
			fileServer := http.StripPrefix("/files/", http.FileServer(http.Dir(l.Dir())))
			r.Get("/files/*", func(w http.ResponseWriter, req *http.Request) {
				if strings.HasSuffix(strings.ToLower(req.URL.Path), ".svg") {
					w.Header().Set("Content-Disposition", "attachment")
					w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
				}
				fileServer.ServeHTTP(w, req)
			})
		}
	}

	s.mountSPA(r)
	return r
}

func (s *Server) mountSPA(r chi.Router) {
	dist, err := web.DistFS()
	if err != nil {
		return
	}
	fileServer := http.FileServer(http.FS(dist))
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/files/") {
			http.NotFound(w, req)
			return
		}
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := dist.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, req)
			return
		}
		http.ServeFileFS(w, req, dist, "index.html")
	})
}

func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"dur", time.Since(start).String(),
		)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		token := ""
		if strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		claims, err := auth.Parse(s.cfg.JWTSecret, token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
