package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ErrInvalidCredentials 用户名或密码错误。
var ErrInvalidCredentials = errors.New("用户名或密码错误")

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("记录不存在")

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AuditLog 保存安全相关操作，供审计追溯。记录中不保存口令、Token、API Key 或请求体。
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Username  string    `gorm:"index;size:64;not null" json:"username"`
	RemoteIP  string    `gorm:"size:64" json:"remote_ip"`
	Action    string    `gorm:"index;size:96;not null" json:"action"`
	Target    string    `gorm:"size:255" json:"target"`
	Outcome   string    `gorm:"index;size:16;not null" json:"outcome"`
	RequestID string    `gorm:"size:64" json:"request_id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// Variant 一张图片的某个尺寸/格式版本，序列化为 JSON 存进 Variants 字段。
type Variant struct {
	Label     string `json:"label"`
	Key       string `json:"key"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Format    string `json:"format"`
	SizeBytes int64  `json:"size_bytes"`
}

type Image struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"uniqueIndex;size:512;not null" json:"key"`
	Prefix      string    `gorm:"index;size:512;not null" json:"prefix"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Category    string    `gorm:"index;size:255;not null" json:"category"`
	ContentType string    `gorm:"size:64;not null" json:"content_type"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	SizeBytes   int64     `json:"size_bytes"`
	SHA256      string    `gorm:"size:64" json:"sha256"`
	Variants    string    `gorm:"type:text" json:"variants"`
	OriginalKey string    `gorm:"size:512" json:"original_key"`
	Direct      bool      `gorm:"default:false" json:"direct"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListFilter struct {
	Query    string
	Category string
	Page     int
	PageSize int
}

type CategoryStat struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
	Bytes    int64  `json:"bytes"`
}

type Stats struct {
	Images     int64          `json:"images"`
	Bytes      int64          `json:"bytes"`
	Categories []CategoryStat `json:"categories"`
}

// Store 封装数据库读写。
type Store struct {
	db *gorm.DB
}

// Open 按驱动打开数据库并自动建表，兼容 PostgreSQL / MySQL / SQLite。
func Open(driver, dsn string) (*Store, error) {
	var dialector gorm.Dialector
	switch driver {
	case "postgres":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	case "sqlite":
		if err := prepareSQLiteDir(dsn); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", driver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&User{}, &Image{}, &AuditLog{}); err != nil {
		return nil, err
	}
	if driver == "sqlite" {
		_ = os.Chmod(dsn, 0o600)
	}
	return &Store{db: db}, nil
}

func prepareSQLiteDir(dsn string) error {
	if dsn == ":memory:" || dsn == "" || len(dsn) >= 5 && dsn[:5] == "file:" {
		return nil
	}
	dir := filepath.Dir(dsn)
	if dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}

// Ping 检查数据库连通性。
func (s *Store) Ping() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// EnsureAdmin 首次启动时创建管理员账号，之后不会覆盖已有账号。
func (s *Store) EnsureAdmin(username, password string) error {
	var count int64
	if err := s.db.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.Create(&User{Username: username, PasswordHash: string(hash)}).Error
}

// Authenticate 校验用户名密码，成功返回用户 ID。
func (s *Store) Authenticate(username, password string) (uint, error) {
	var u User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		return 0, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return 0, ErrInvalidCredentials
	}
	return u.ID, nil
}

// UpdatePassword 更新当前管理员口令，并使用 bcrypt 重新散列。
func (s *Store) UpdatePassword(userID uint, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result := s.db.Model(&User{}).Where("id = ?", userID).Update("password_hash", string(hash))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateAudit(log *AuditLog) error {
	return s.db.Create(log).Error
}

func (s *Store) ListAudit(page, pageSize int) ([]AuditLog, int64, error) {
	var total int64
	if err := s.db.Model(&AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	logs := []AuditLog{}
	err := s.db.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func (s *Store) PruneAudit(before time.Time) error {
	return s.db.Where("created_at < ?", before).Delete(&AuditLog{}).Error
}

func (s *Store) CreateImage(img *Image) error {
	return s.db.Create(img).Error
}

func (s *Store) GetImage(id uint) (*Image, error) {
	var img Image
	if err := s.db.First(&img, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &img, nil
}

func (s *Store) GetImageByKey(key string) (*Image, error) {
	var img Image
	if err := s.db.Where("key = ?", key).First(&img).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &img, nil
}

func (s *Store) UpdateImage(img *Image) error {
	return s.db.Save(img).Error
}

func (s *Store) DeleteImage(id uint) error {
	return s.db.Delete(&Image{}, id).Error
}

func (s *Store) ListImages(f ListFilter) ([]Image, int64, error) {
	q := s.db.Model(&Image{})
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		q = q.Where("name LIKE ? OR key LIKE ? OR category LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []Image{}
	err := q.Order("created_at DESC, id DESC").
		Offset((f.Page - 1) * f.PageSize).
		Limit(f.PageSize).
		Find(&items).Error
	return items, total, err
}

func (s *Store) Categories() ([]CategoryStat, error) {
	rows := []CategoryStat{}
	err := s.db.Model(&Image{}).
		Select("category", "COUNT(*) AS count", "COALESCE(SUM(size_bytes),0) AS bytes").
		Group("category").
		Order("count DESC").
		Scan(&rows).Error
	return rows, err
}

func (s *Store) Stats() (Stats, error) {
	var st Stats
	row := s.db.Model(&Image{}).Select("COUNT(*)", "COALESCE(SUM(size_bytes),0)").Row()
	if err := row.Scan(&st.Images, &st.Bytes); err != nil {
		return st, err
	}
	var err error
	st.Categories, err = s.Categories()
	return st, err
}
