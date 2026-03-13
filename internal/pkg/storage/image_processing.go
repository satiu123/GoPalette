package storage

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ImageProcessingStorage 作为存储中间件，对图片做压缩并转成 webp。
type ImageProcessingStorage struct {
	next      Storage
	maxWidth  int
	maxHeight int
	quality   float32
}

func NewImageProcessingStorage(next Storage, maxWidth, maxHeight int, quality float32) *ImageProcessingStorage {
	if maxWidth <= 0 {
		maxWidth = 1920
	}
	if maxHeight <= 0 {
		maxHeight = 1920
	}
	if quality <= 0 || quality > 100 {
		quality = 80
	}
	return &ImageProcessingStorage{
		next:      next,
		maxWidth:  maxWidth,
		maxHeight: maxHeight,
		quality:   quality,
	}
}

func (s *ImageProcessingStorage) Save(file multipart.File, filename string) (string, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("解析图片失败: %w", err)
	}

	processed := s.resizeToFit(img, s.maxWidth, s.maxHeight)

	buf := bytes.NewBuffer(nil)
	if err := webp.Encode(buf, processed, &webp.Options{Quality: s.quality}); err != nil {
		return "", fmt.Errorf("编码 webp 失败: %w", err)
	}

	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	if base == "" {
		base = "image"
	}
	newName := base + ".webp"

	return s.next.Save(newReadSeekCloser(buf.Bytes()), newName)
}

func (s *ImageProcessingStorage) Delete(url string) error {
	return s.next.Delete(url)
}

func (s *ImageProcessingStorage) resizeToFit(src image.Image, maxW, maxH int) image.Image {
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= maxW && h <= maxH {
		return src
	}

	ratioW := float64(maxW) / float64(w)
	ratioH := float64(maxH) / float64(h)
	ratio := ratioW
	if ratioH < ratioW {
		ratio = ratioH
	}

	nw := int(float64(w) * ratio)
	nh := int(float64(h) * ratio)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

type readSeekCloser struct {
	*bytes.Reader
}

func newReadSeekCloser(data []byte) *readSeekCloser {
	return &readSeekCloser{Reader: bytes.NewReader(data)}
}

func (r *readSeekCloser) Close() error { return nil }
