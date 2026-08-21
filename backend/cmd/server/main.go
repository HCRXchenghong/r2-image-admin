package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"r2-image-admin/backend/internal/config"
	"r2-image-admin/backend/internal/httpapi"
	"r2-image-admin/backend/internal/imaging"
	"r2-image-admin/backend/internal/storage"
	"r2-image-admin/backend/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("配置加载失败", "err", err)
		os.Exit(1)
	}
	warnDefaults(cfg)

	db, err := store.Open(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		slog.Error("数据库连接失败", "err", err)
		os.Exit(1)
	}
	if err := db.EnsureAdmin(cfg.AdminUsername, cfg.AdminPassword); err != nil {
		slog.Error("初始化管理员失败", "err", err)
		os.Exit(1)
	}
	if err := db.PruneAudit(time.Now().AddDate(0, 0, -cfg.AuditRetentionDays)); err != nil {
		slog.Warn("清理过期审计日志失败", "err", err)
	}

	var obj storage.Storage
	switch cfg.StorageDriver {
	case "r2":
		obj, err = storage.NewR2(cfg.R2AccountID, cfg.R2AccessKey, cfg.R2SecretKey, cfg.R2Bucket, cfg.PublicBaseURL)
	case "local":
		obj, err = storage.NewLocal(cfg.LocalDataDir, cfg.PublicBaseURL)
	}
	if err != nil {
		slog.Error("存储初始化失败", "err", err)
		os.Exit(1)
	}

	proc := imaging.NewProcessor()
	api := httpapi.New(cfg, db, obj, proc)
	api.SetRestartFunc(restartProcess)
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      3 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		slog.Info("服务已启动",
			"addr", cfg.Addr,
			"storage", cfg.StorageDriver,
			"db", cfg.DBDriver,
			"processing", proc.Available(),
			"public_base_url", cfg.PublicBaseURL,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("服务异常退出", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	slog.Info("服务已停止")
}

func warnDefaults(cfg *config.Config) {
	if cfg.AdminPassword == "admin123" {
		slog.Warn("使用默认管理员密码 admin123，请通过 ADMIN_PASSWORD 环境变量修改")
	}
	if cfg.JWTSecret == "dev-insecure-secret-change-me" {
		slog.Warn("使用开发默认 JWT_SECRET，生产环境请设置强随机值")
	}
	if cfg.StorageDriver == "local" {
		slog.Warn("当前使用本地磁盘存储，仅用于开发试跑；生产请设置 STORAGE_DRIVER=r2")
	}
	if cfg.DBDriver == "sqlite" {
		slog.Warn("当前使用 SQLite，仅用于开发试跑；生产请设置 DB_DRIVER=postgres 或 mysql")
	}
}

// restartProcess 以相同参数与环境拉起新进程后退出，实现“保存配置后自动重启”。
func restartProcess(env []string) {
	exe, err := os.Executable()
	if err != nil {
		slog.Error("自动重启失败：无法获取可执行文件路径", "err", err)
		os.Exit(1)
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		slog.Error("自动重启失败：无法启动新进程", "err", err)
		os.Exit(1)
	}
	_ = cmd.Process.Release()
	slog.Info("自动重启：新进程已启动，当前进程退出")
	os.Exit(0)
}
