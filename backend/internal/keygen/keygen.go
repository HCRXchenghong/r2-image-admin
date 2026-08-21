package keygen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var segRe = regexp.MustCompile(`[^a-z0-9-]+`)

const (
	maxCategorySegments = 8
	maxCategorySegment  = 48
)

// NewID 返回 8 位十六进制随机 ID，用作文件路径中的唯一目录。
func NewID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Category 把用户输入规范化为安全的路径，只保留小写字母、数字和短横线，例如 "Products/Summer 2026" -> "products/summer-2026"。
func Category(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	segs := strings.Split(input, "/")
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if len(out) >= maxCategorySegments {
			break
		}
		s = segRe.ReplaceAllString(s, "-")
		s = strings.Trim(s, "-")
		if len(s) > maxCategorySegment {
			s = strings.Trim(s[:maxCategorySegment], "-")
		}
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return "general"
	}
	return strings.Join(out, "/")
}

// MainKey 主图（原尺寸）路径。
func MainKey(prefix, format string) string {
	return fmt.Sprintf("%s/main.%s", prefix, format)
}

// SizeKey 指定宽度的变体路径。
func SizeKey(prefix string, width int, format string) string {
	return fmt.Sprintf("%s/%d.%s", prefix, width, format)
}

// OriginalKey 原文件路径，ext 需要带点，例如 ".jpg"。
func OriginalKey(prefix, ext string) string {
	return fmt.Sprintf("%s/original%s", prefix, ext)
}
