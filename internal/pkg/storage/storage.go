package storage

import (
	"mime/multipart"
)

// Storage 定义图片存储的标准接口，支持多种实现（本地、OSS 等）
type Storage interface {
	Save(file multipart.File, filename string) (url string, err error)
	Delete(url string) error
}
