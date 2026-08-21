package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionRejectsWeakSecrets(t *testing.T) {
	values := map[string]string{
		"APP_ENV": "production", "DB_DRIVER": "sqlite", "STORAGE_DRIVER": "r2",
		"R2_ACCOUNT_ID": "account", "R2_ACCESS_KEY_ID": "access", "R2_SECRET_ACCESS_KEY": "secret", "R2_BUCKET": "images",
		"PUBLIC_BASE_URL": "https://img.example.com", "ADMIN_PASSWORD": "admin123", "JWT_SECRET": "dev-insecure-secret-change-me",
		"JWT_TTL_HOURS": "12", "AI_IMAGE_API_URL": "https://api.openai.com/v1/images/generations",
	}
	_, err := load(func(key string) string { return values[key] })
	if err == nil || !strings.Contains(err.Error(), "管理员密码") {
		t.Fatalf("expected weak password rejection, got %v", err)
	}
}

func TestWriteEnvFileRejectsLineInjection(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := WriteEnvFile(path, map[string]string{"JWT_SECRET": "valid\nADMIN_PASSWORD=attacker"}); err == nil {
		t.Fatal("expected line injection rejection")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected config file after rejected write: %v", err)
	}
}

func TestWriteEnvFileQuotesSpecialCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := WriteEnvFile(path, map[string]string{"ADMIN_PASSWORD": `long password # with "quotes"`}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `ADMIN_PASSWORD="long password # with \"quotes\""`) {
		t.Fatalf("unexpected encoded config: %s", content)
	}
}
