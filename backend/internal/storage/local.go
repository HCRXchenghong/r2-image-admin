package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Local 本地磁盘存储，用于开发和试跑，不进生产。
type Local struct {
	dir     string
	baseURL string
}

func NewLocal(dataDir, publicBaseURL string) (*Local, error) {
	if dataDir == "" {
		dataDir = "data/files"
	}
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, err
	}
	if err := os.Chmod(dataDir, 0o750); err != nil {
		return nil, err
	}
	return &Local{dir: dataDir, baseURL: strings.TrimRight(publicBaseURL, "/")}, nil
}

func (l *Local) Driver() string { return "local" }

// Dir 暴露文件根目录，供开发环境挂载静态文件服务。
func (l *Local) Dir() string { return l.dir }

func (l *Local) absPath(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	p := filepath.Join(l.dir, clean)
	if !strings.HasPrefix(p, l.dir+string(filepath.Separator)) && p != l.dir {
		return "", fmt.Errorf("非法的 key: %s", key)
	}
	return p, nil
}

func (l *Local) Put(_ context.Context, obj Object) error {
	p, err := l.absPath(obj.Key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	return os.WriteFile(p, obj.Body, 0o640)
}

func (l *Local) Get(_ context.Context, key string) (Object, error) {
	p, err := l.absPath(key)
	if err != nil {
		return Object{}, err
	}
	body, err := os.ReadFile(p)
	if err != nil {
		return Object{}, err
	}
	ct := "application/octet-stream"
	if ext := filepath.Ext(key); ext != "" {
		if m := mimeTypeByExt(ext); m != "" {
			ct = m
		}
	}
	return Object{Key: key, ContentType: ct, Body: body}, nil
}

func (l *Local) Delete(_ context.Context, keys []string) error {
	for _, k := range keys {
		p, err := l.absPath(k)
		if err != nil {
			continue
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (l *Local) PresignPut(_ context.Context, _, _ string, _ time.Duration) (string, error) {
	return "", errors.New("本地存储不支持预签名直传，请切换 STORAGE_DRIVER=r2")
}

func (l *Local) PublicURL(key string) string {
	if l.baseURL != "" {
		return l.baseURL + "/" + key
	}
	return "/files/" + key
}

func mimeTypeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return ""
	}
}
