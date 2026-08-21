//go:build !vips

package imaging

import "errors"

var errProcessingDisabled = errors.New("图片处理未启用：请使用 Docker 镜像部署，或安装 libvips 后用 -tags vips 编译；原图上传不受影响")

type noopProcessor struct{}

func NewProcessor() Processor { return noopProcessor{} }

func (noopProcessor) Available() bool { return false }

func (noopProcessor) DecodeSize([]byte) (int, int, error) {
	return 0, 0, errProcessingDisabled
}

func (noopProcessor) Render([]byte, int, int, int, string) ([]byte, error) {
	return nil, errProcessingDisabled
}
