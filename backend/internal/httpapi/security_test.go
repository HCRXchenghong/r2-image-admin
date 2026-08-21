package httpapi

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"r2-image-admin/backend/internal/auth"
	"r2-image-admin/backend/internal/config"
	"r2-image-admin/backend/internal/imaging"
	"r2-image-admin/backend/internal/storage"
	"r2-image-admin/backend/internal/store"
)

func newSecurityTestServer(t *testing.T) (*Server, *store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open("sqlite", filepath.Join(dir, "r2admin.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureAdmin("admin", "a-long-test-password"); err != nil {
		t.Fatal(err)
	}
	obj, err := storage.NewLocal(filepath.Join(dir, "files"), "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		AppEnv:             "test",
		CorsAllowedOrigins: []string{"https://console.example.com"},
		StorageDriver:      "local",
		JWTSecret:          "test-jwt-secret-with-at-least-thirty-two-characters",
		JWTTTL:             time.Hour,
		ImgFormats:         []string{"webp"},
		ImgQuality:         82,
		MaxUploadMB:        1,
		AI_IMAGE_API_URL:   "https://api.openai.com/v1/images/generations",
		AutoRestart:        false,
		AuditRetentionDays: 180,
	}
	s := New(cfg, db, obj, imaging.NewProcessor())
	token, err := auth.Sign(cfg.JWTSecret, cfg.JWTTTL, 1, "admin")
	if err != nil {
		t.Fatal(err)
	}
	return s, db, token
}

func TestSecurityHeadersAndCORSAllowlist(t *testing.T) {
	s, _, _ := newSecurityTestServer(t)
	h := s.Handler()

	evil := httptest.NewRequest(http.MethodOptions, "/api/config", nil)
	evil.Header.Set("Origin", "https://evil.example")
	evil.Header.Set("Access-Control-Request-Method", http.MethodPut)
	evilResult := httptest.NewRecorder()
	h.ServeHTTP(evilResult, evil)
	if evilResult.Code != http.StatusForbidden {
		t.Fatalf("unexpected disallowed preflight status: %d", evilResult.Code)
	}
	if got := evilResult.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin received CORS header: %q", got)
	}

	allowed := httptest.NewRequest(http.MethodOptions, "/api/config", nil)
	allowed.Header.Set("Origin", "https://console.example.com")
	allowed.Header.Set("Access-Control-Request-Method", http.MethodPut)
	allowedResult := httptest.NewRecorder()
	h.ServeHTTP(allowedResult, allowed)
	if allowedResult.Code != http.StatusNoContent {
		t.Fatalf("unexpected allowed preflight status: %d", allowedResult.Code)
	}
	if got := allowedResult.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("unexpected allowed origin: %q", got)
	}
	if got := allowedResult.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("missing Content-Security-Policy")
	}
	if got := allowedResult.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("unexpected X-Content-Type-Options: %q", got)
	}
}

func TestLoginIsRateLimited(t *testing.T) {
	s, db, _ := newSecurityTestServer(t)
	h := s.Handler()
	for i := 0; i < maxLoginFailures; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"incorrect"}`))
		r.Header.Set("Content-Type", "application/json")
		r.RemoteAddr = "198.51.100.9:3333"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"incorrect"}`))
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "198.51.100.9:3333"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after repeated failures, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
	logs, total, err := db.ListAudit(1, 20)
	if err != nil || total != int64(maxLoginFailures+1) || logs[0].Outcome != "blocked" {
		t.Fatalf("unexpected login audit result: total=%d logs=%+v err=%v", total, logs, err)
	}
}

func TestDirectUploadRejectsSVGAndAuditsFailure(t *testing.T) {
	s, db, token := newSecurityTestServer(t)
	h := s.Handler()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	part, err := mw.CreateFormFile("file", "unsafe.svg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/images/direct", body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected SVG rejection, got %d: %s", w.Code, w.Body.String())
	}
	logs, total, err := db.ListAudit(1, 10)
	if err != nil || total != 1 || logs[0].Action != http.MethodPost || logs[0].Outcome != "failure" {
		t.Fatalf("unexpected upload audit log: total=%d logs=%+v err=%v", total, logs, err)
	}
}

func TestR2GuideRequiresAuthenticatedSession(t *testing.T) {
	s, _, token := newSecurityTestServer(t)
	h := s.Handler()

	anonymous := httptest.NewRecorder()
	h.ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "/api/guides/r2", nil))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated guide request to be rejected, got %d", anonymous.Code)
	}

	authenticatedRequest := httptest.NewRequest(http.MethodGet, "/api/guides/r2", nil)
	authenticatedRequest.Header.Set("Authorization", "Bearer "+token)
	authenticated := httptest.NewRecorder()
	h.ServeHTTP(authenticated, authenticatedRequest)
	if authenticated.Code != http.StatusOK || !bytes.Contains(authenticated.Body.Bytes(), []byte("Cloudflare R2 接入指南")) {
		t.Fatalf("expected authenticated guide response, got %d: %s", authenticated.Code, authenticated.Body.String())
	}
}
