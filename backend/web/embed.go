package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// DistFS 返回前端构建产物文件系统。
func DistFS() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
