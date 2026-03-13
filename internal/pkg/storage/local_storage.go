package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// LocalStorage 将文件存储到本地磁盘
type LocalStorage struct {
	dir     string
	baseURL string
}

// NewLocalStorage 创建本地存储实例，dir 为存储目录，baseURL 为访问前缀
func NewLocalStorage(dir, baseURL string) *LocalStorage {
	_ = os.MkdirAll(dir, 0o755)
	return &LocalStorage{dir: dir, baseURL: baseURL}
}

// Save 将上传文件保存到本地目录，以文件内容 MD5 作为文件名实现去重，返回可访问的相对 URL
func (s *LocalStorage) Save(file multipart.File, filename string) (string, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	ext := filepath.Ext(filename)
	hash := md5.Sum(content)
	newName := fmt.Sprintf("%x%s", hash, ext)
	dst := filepath.Join(s.dir, newName)
	if _, err := os.Stat(dst); err == nil {
		return s.baseURL + "/" + newName, nil
	}
	if err := os.WriteFile(dst, content, 0o644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return s.baseURL + "/" + newName, nil
}

func (s *LocalStorage) Delete(url string) error {
	name := filepath.Base(url)
	path := filepath.Join(s.dir, name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}
