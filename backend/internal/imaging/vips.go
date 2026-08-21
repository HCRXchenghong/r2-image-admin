//go:build vips

package imaging

import (
	"fmt"

	"github.com/h2non/bimg"
)

var typeMap = map[string]bimg.ImageType{
	"webp": bimg.WEBP,
	"avif": bimg.AVIF,
	"jpeg": bimg.JPEG,
	"png":  bimg.PNG,
}

type vipsProcessor struct{}

func NewProcessor() Processor { return vipsProcessor{} }

func (vipsProcessor) Available() bool {
	return bimg.VipsVersion != ""
}

func (vipsProcessor) DecodeSize(data []byte) (int, int, error) {
	size, err := bimg.NewImage(data).Size()
	if err != nil {
		return 0, 0, err
	}
	return size.Width, size.Height, nil
}

// Render 输出指定尺寸与格式；width/height 为 0 时保持原尺寸。
func (vipsProcessor) Render(data []byte, width, height, quality int, format string) ([]byte, error) {
	typ, ok := typeMap[format]
	if !ok {
		return nil, fmt.Errorf("不支持的输出格式: %s", format)
	}
	opts := bimg.Options{
		Type:          typ,
		Quality:       quality,
		StripMetadata: true,
	}
	if width > 0 && height > 0 {
		opts.Width = width
		opts.Height = height
	}
	out, err := bimg.NewImage(data).Process(opts)
	if err != nil {
		return nil, err
	}
	return out, nil
}
