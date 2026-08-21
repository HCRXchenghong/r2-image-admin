package keygen

import (
	"regexp"
	"testing"
)

func TestNewID(t *testing.T) {
	idRe := regexp.MustCompile(`^[0-9a-f]{8}$`)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := NewID()
		if !idRe.MatchString(id) {
			t.Fatalf("NewID 返回非法格式: %q", id)
		}
		if seen[id] {
			t.Fatalf("NewID 出现重复: %q", id)
		}
		seen[id] = true
	}
}

func TestCategory(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Products/Summer 2026", "products/summer-2026"},
		{" 产品 / 春夏 ", "general"},
		{"", "general"},
		{"a//b/", "a/b"},
		{"UPPER.Case_File", "upper-case-file"},
		{"products/sku-001", "products/sku-001"},
	}
	for _, c := range cases {
		if got := Category(c.in); got != c.want {
			t.Errorf("Category(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKeyFormats(t *testing.T) {
	if got := MainKey("products/a1b2c3d4", "webp"); got != "products/a1b2c3d4/main.webp" {
		t.Errorf("MainKey = %q", got)
	}
	if got := SizeKey("products/a1b2c3d4", 400, "webp"); got != "products/a1b2c3d4/400.webp" {
		t.Errorf("SizeKey = %q", got)
	}
	if got := OriginalKey("products/a1b2c3d4", ".jpg"); got != "products/a1b2c3d4/original.jpg" {
		t.Errorf("OriginalKey = %q", got)
	}
}
