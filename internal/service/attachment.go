package service

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"time"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/pkg/storage"
	"github.com/satiu123/GoPalette/internal/repository"
)

var imgSrcRegexp = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)

type AttachmentService struct {
	repo  repository.AttachmentRepository
	store storage.Storage
}

func NewAttachmentService(repo repository.AttachmentRepository, store storage.Storage) *AttachmentService {
	return &AttachmentService{repo: repo, store: store}
}

func (s *AttachmentService) CreateTemporary(ctx context.Context, userID int64, url string) error {
	if userID <= 0 || url == "" {
		return errors.New("invalid attachment data")
	}
	return s.repo.Create(ctx, &model.Attachment{
		UserID: userID,
		PostID: 0,
		URL:    url,
	})
}

func (s *AttachmentService) BindFromContent(ctx context.Context, userID, postID int64, content string) error {
	if userID <= 0 || postID <= 0 || content == "" {
		return nil
	}

	matches := imgSrcRegexp.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	urlSet := make(map[string]struct{}, len(matches))
	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		u := match[1]
		if _, exists := urlSet[u]; exists {
			continue
		}
		urlSet[u] = struct{}{}
		urls = append(urls, u)
	}

	if len(urls) == 0 {
		return nil
	}
	return s.repo.BindTemporaryByURLs(ctx, userID, postID, urls)
}

func (s *AttachmentService) CleanupExpiredTemporary(ctx context.Context, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	cutoff := time.Now().Add(-ttl)
	rows, err := s.repo.FindExpiredTemporary(ctx, cutoff)
	if err != nil {
		return err
	}

	for _, item := range rows {
		if err := s.store.Delete(item.URL); err != nil {
			slog.Warn("清理临时附件: 删除文件失败", "attachment_id", item.ID, "url", item.URL, "error", err)
			continue
		}
		if err := s.repo.DeleteByID(ctx, item.ID); err != nil {
			slog.Warn("清理临时附件: 删除记录失败", "attachment_id", item.ID, "url", item.URL, "error", err)
		}
	}

	slog.Debug("临时附件清理完成", "count", len(rows))
	return nil
}
