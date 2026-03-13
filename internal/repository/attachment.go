package repository

import (
	"context"
	"time"

	"github.com/satiu123/GoPalette/internal/model"
)

// AttachmentRepository 定义上传附件元数据的存取接口
type AttachmentRepository interface {
	Create(ctx context.Context, attachment *model.Attachment) error
	BindTemporaryByURLs(ctx context.Context, userID, postID int64, urls []string) error
	FindExpiredTemporary(ctx context.Context, createdBefore time.Time) ([]model.Attachment, error)
	DeleteByID(ctx context.Context, id int64) error
}
