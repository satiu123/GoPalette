package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/satiu123/GoPalette/internal/service"
)

func StartAttachmentCleanup(attSvc *service.AttachmentService, spec string, ttl time.Duration) (*cron.Cron, error) {
	if attSvc == nil {
		return nil, nil
	}
	if spec == "" {
		spec = "@every 1h"
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	c := cron.New()
	_, err := c.AddFunc(spec, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := attSvc.CleanupExpiredTemporary(ctx, ttl); err != nil {
			slog.Error("临时附件清理任务失败", "error", err)
		}
	})
	if err != nil {
		return nil, err
	}
	c.Start()
	slog.Info("临时附件清理任务已启动", "spec", spec, "ttl", ttl.String())
	return c, nil
}
