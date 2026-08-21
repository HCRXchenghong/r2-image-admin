package storage

import (
	"context"
	"time"
)

// Object 一个待存储的对象。
type Object struct {
	Key         string
	ContentType string
	Body        []byte
}

// Storage 抽象存储后端：R2 与本地磁盘都实现该接口。
type Storage interface {
	Put(ctx context.Context, obj Object) error
	Get(ctx context.Context, key string) (Object, error)
	Delete(ctx context.Context, keys []string) error
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error)
	PublicURL(key string) string
	Driver() string
}
