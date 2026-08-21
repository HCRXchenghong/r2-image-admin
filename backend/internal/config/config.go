package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 汇总所有运行配置，全部来自环境变量（可选 .env 文件）。
type Config struct {
	Addr   string
	AppEnv string

	// CorsAllowedOrigins 为空时仅允许同源请求，不会返回 CORS 响应头。
	CorsAllowedOrigins []string

	DBDriver string
	DBDSN    string

	StorageDriver string
	R2AccountID   string
	R2AccessKey   string
	R2SecretKey   string
	R2Bucket      string
	PublicBaseURL string
	LocalDataDir  string

	AdminUsername string
	AdminPassword string
	JWTSecret     string
	JWTTTL        time.Duration

	ImgSizes     []int
	ImgFormats   []string
	ImgQuality   int
	KeepOriginal bool
	MaxUploadMB  int64

	AI_IMAGE_API_URL string
	AI_IMAGE_API_KEY string
	AI_IMAGE_MODEL   string
	AIAllowedHosts   []string

	AutoRestart        bool
	AuditRetentionDays int
}

// Load 读取环境变量并做基本校验。
func Load() (*Config, error) {
	envFile := ResolveEnvFile()
	if os.Getenv("ENV_FILE") != "" {
		// 明确指定的配置文件是设置页的持久化来源，优先于容器初始环境变量。
		_ = godotenv.Overload(envFile)
	} else {
		_ = godotenv.Load(envFile)
	}
	return load(os.Getenv)
}

// LoadWith 在现有环境变量基础上叠加 updates 后校验，返回合并后的配置。
// 用于“设置页保存配置”时先校验，避免写入非法值。
func LoadWith(updates map[string]string) (*Config, error) {
	merged := map[string]string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			merged[kv[:i]] = kv[i+1:]
		}
	}
	for k, v := range updates {
		merged[k] = v
	}
	return load(func(k string) string { return merged[k] })
}

func load(get func(string) string) (*Config, error) {
	cfg := &Config{
		Addr:               getEnv(get, "ADDR", ":8080"),
		AppEnv:             strings.ToLower(getEnv(get, "APP_ENV", "development")),
		CorsAllowedOrigins: parseStringList(get("CORS_ALLOWED_ORIGINS")),
		DBDriver:           strings.ToLower(getEnv(get, "DB_DRIVER", "sqlite")),
		DBDSN:              getEnv(get, "DB_DSN", "data/r2admin.db"),
		StorageDriver:      strings.ToLower(getEnv(get, "STORAGE_DRIVER", "local")),
		R2AccountID:        get("R2_ACCOUNT_ID"),
		R2AccessKey:        get("R2_ACCESS_KEY_ID"),
		R2SecretKey:        get("R2_SECRET_ACCESS_KEY"),
		R2Bucket:           getEnv(get, "R2_BUCKET", "site-images"),
		PublicBaseURL:      strings.TrimRight(get("PUBLIC_BASE_URL"), "/"),
		LocalDataDir:       getEnv(get, "LOCAL_DATA_DIR", "data/files"),
		AdminUsername:      getEnv(get, "ADMIN_USERNAME", "admin"),
		AdminPassword:      get("ADMIN_PASSWORD"),
		JWTSecret:          get("JWT_SECRET"),
		JWTTTL:             time.Duration(getEnvInt(get, "JWT_TTL_HOURS", 12)) * time.Hour,
		ImgSizes:           parseIntList(getEnv(get, "IMG_SIZES", "400,800,1200,1600")),
		ImgFormats:         parseStringList(getEnv(get, "IMG_FORMATS", "webp")),
		ImgQuality:         getEnvInt(get, "IMG_QUALITY", 82),
		KeepOriginal:       getEnvBool(get, "IMG_KEEP_ORIGINAL", true),
		MaxUploadMB:        int64(getEnvInt(get, "UPLOAD_MAX_MB", 20)),
		AI_IMAGE_API_URL:   getEnv(get, "AI_IMAGE_API_URL", "https://api.openai.com/v1/images/generations"),
		AI_IMAGE_API_KEY:   get("AI_IMAGE_API_KEY"),
		AI_IMAGE_MODEL:     getEnv(get, "AI_IMAGE_MODEL", "gpt-image-1"),
		AIAllowedHosts:     parseStringList(get("AI_IMAGE_ALLOWED_HOSTS")),
		AutoRestart:        getEnvBool(get, "AUTO_RESTART", true),
		AuditRetentionDays: getEnvInt(get, "AUDIT_RETENTION_DAYS", 180),
	}
	if cfg.AppEnv != "development" && cfg.AppEnv != "production" && cfg.AppEnv != "test" {
		return nil, fmt.Errorf("APP_ENV 必须是 development / production / test，当前值: %q", cfg.AppEnv)
	}

	switch cfg.DBDriver {
	case "postgres", "mysql", "sqlite":
	default:
		return nil, fmt.Errorf("DB_DRIVER 必须是 postgres / mysql / sqlite，当前值: %q", cfg.DBDriver)
	}

	switch cfg.StorageDriver {
	case "r2", "local":
	default:
		return nil, fmt.Errorf("STORAGE_DRIVER 必须是 r2 / local，当前值: %q", cfg.StorageDriver)
	}

	if cfg.StorageDriver == "r2" {
		if cfg.R2AccountID == "" || cfg.R2AccessKey == "" || cfg.R2SecretKey == "" || cfg.R2Bucket == "" {
			return nil, fmt.Errorf("使用 r2 存储时必须配置 R2_ACCOUNT_ID / R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY / R2_BUCKET")
		}
		if cfg.PublicBaseURL == "" {
			return nil, fmt.Errorf("使用 r2 存储时必须配置 PUBLIC_BASE_URL（例如 https://img.example.com）")
		}
	}

	if cfg.AdminPassword == "" {
		cfg.AdminPassword = "admin123"
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "dev-insecure-secret-change-me"
	}
	if cfg.JWTTTL < time.Hour || cfg.JWTTTL > 24*time.Hour {
		return nil, fmt.Errorf("JWT_TTL_HOURS 必须在 1-24 之间")
	}
	if cfg.MaxUploadMB < 1 || cfg.MaxUploadMB > 200 {
		return nil, fmt.Errorf("UPLOAD_MAX_MB 必须在 1-200 之间")
	}
	if cfg.AuditRetentionDays < 30 || cfg.AuditRetentionDays > 3650 {
		return nil, fmt.Errorf("AUDIT_RETENTION_DAYS 必须在 30-3650 之间")
	}
	if err := validateOrigins(cfg.CorsAllowedOrigins); err != nil {
		return nil, err
	}
	if err := validateAIURL(cfg.AI_IMAGE_API_URL, cfg.AppEnv); err != nil {
		return nil, err
	}
	if err := validateHostnames(cfg.AIAllowedHosts); err != nil {
		return nil, err
	}
	if len(cfg.ImgFormats) == 0 {
		cfg.ImgFormats = []string{"webp"}
	}
	for _, f := range cfg.ImgFormats {
		if f != "webp" && f != "avif" && f != "jpeg" && f != "png" {
			return nil, fmt.Errorf("IMG_FORMATS 仅支持 webp / avif / jpeg / png，当前值: %q", f)
		}
	}
	if cfg.ImgQuality < 1 || cfg.ImgQuality > 100 {
		return nil, fmt.Errorf("IMG_QUALITY 必须在 1-100 之间")
	}
	if cfg.AppEnv == "production" {
		if err := ValidateAdminPassword(cfg.AdminPassword); err != nil {
			return nil, err
		}
		if err := ValidateJWTSecret(cfg.JWTSecret); err != nil {
			return nil, err
		}
		if cfg.StorageDriver != "r2" {
			return nil, fmt.Errorf("生产环境必须使用 STORAGE_DRIVER=r2")
		}
		if !strings.HasPrefix(strings.ToLower(cfg.PublicBaseURL), "https://") {
			return nil, fmt.Errorf("生产环境的 PUBLIC_BASE_URL 必须使用 HTTPS")
		}
	}
	return cfg, nil
}

// WriteEnvFile 把 updates 合并写入 .env 文件，保留未涉及的行和注释。
func WriteEnvFile(path string, updates map[string]string) error {
	lines := []string{}
	handled := map[string]bool{}

	for key, value := range updates {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "=\r\n\x00") {
			return fmt.Errorf("配置项名称非法")
		}
		if strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf("配置值不能包含换行或 NUL 字符")
		}
	}

	if b, err := os.ReadFile(path); err == nil {
		content := strings.ReplaceAll(string(b), "\r\n", "\n")
		for _, line := range strings.Split(content, "\n") {
			trim := strings.TrimSpace(line)
			if trim == "" || strings.HasPrefix(trim, "#") {
				lines = append(lines, line)
				continue
			}
			eq := strings.IndexByte(line, '=')
			if eq < 0 {
				lines = append(lines, line)
				continue
			}
			key := strings.TrimSpace(line[:eq])
			if key == "" {
				lines = append(lines, line)
				continue
			}
			if v, ok := updates[key]; ok {
				lines = append(lines, key+"="+quoteEnvValue(v))
				handled[key] = true
			} else {
				lines = append(lines, line)
			}
		}
	}

	for k, v := range updates {
		if !handled[k] {
			lines = append(lines, k+"="+quoteEnvValue(v))
		}
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func quoteEnvValue(value string) string {
	if value == "" {
		return ""
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return r == '#' || r == ' ' || r == '\t' || r == '\\' || r == '"' || r == '\''
	}) < 0 {
		return value
	}
	return `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`
}

// ResolveEnvFile 返回配置文件的读取/写入路径：
// 优先使用 ENV_FILE，否则按 .env、../.env 顺序取第一个存在的文件，
// 都不存在时默认使用当前目录的 .env。
func ResolveEnvFile() string {
	if f := os.Getenv("ENV_FILE"); f != "" {
		return f
	}
	for _, c := range []string{".env", "../.env"} {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ".env"
}

// MergeEnv 基于当前进程环境变量叠加 updates，返回适合传给重启子进程的完整环境。
func MergeEnv(updates map[string]string) []string {
	out := append([]string{}, os.Environ()...)
	for k, v := range updates {
		kv := k + "=" + v
		replaced := false
		for i, existing := range out {
			if strings.HasPrefix(existing, k+"=") {
				out[i] = kv
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, kv)
		}
	}
	return out
}

func getEnv(get func(string) string, key, fallback string) string {
	if v := get(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(get func(string) string, key string, fallback int) int {
	if v := get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(get func(string) string, key string, fallback bool) bool {
	if v := get(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func parseIntList(s string) []int {
	var out []int
	seen := map[int]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n <= 0 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func parseStringList(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" && !seen[part] {
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}

func isWeakPassword(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) < 12 || value == "admin123" || value == "change-me-please"
}

func isWeakJWTSecret(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) < 32 || value == "dev-insecure-secret-change-me" || value == "please-change-me-to-a-long-random-string"
}

// ValidateAdminPassword 校验设置页写入的管理员口令。开发与生产均不接受弱口令。
func ValidateAdminPassword(value string) error {
	value = strings.TrimSpace(value)
	if len(value) < 12 || len(value) > 256 || isWeakPassword(value) {
		return fmt.Errorf("管理员密码至少需要 12 位，且不能使用默认值")
	}
	return nil
}

// ValidateJWTSecret 校验用于签发会话的对称密钥。
func ValidateJWTSecret(value string) error {
	value = strings.TrimSpace(value)
	if len(value) < 32 || len(value) > 1024 || isWeakJWTSecret(value) {
		return fmt.Errorf("JWT_SECRET 至少需要 32 位，且不能使用默认值")
	}
	return nil
}

func validateOrigins(origins []string) error {
	for _, raw := range origins {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS 包含无效来源: %q", raw)
		}
	}
	return nil
}

func validateAIURL(raw, appEnv string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("AI_IMAGE_API_URL 必须是有效的 HTTP(S) URL")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && appEnv != "production") {
		return fmt.Errorf("生产环境的 AI_IMAGE_API_URL 必须使用 HTTPS")
	}
	return nil
}

func validateHostnames(hosts []string) error {
	for _, host := range hosts {
		if host == "" || strings.ContainsAny(host, "/\\@:\t\r\n") {
			return fmt.Errorf("AI_IMAGE_ALLOWED_HOSTS 包含无效主机名: %q", host)
		}
	}
	return nil
}
