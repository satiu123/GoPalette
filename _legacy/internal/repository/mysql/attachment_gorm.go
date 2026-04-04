package mysql

import (
	"context"
	"time"

	"github.com/satiu123/GoPalette/internal/model"
	"gorm.io/gorm"
)

type attachmentGormRepository struct {
	db *gorm.DB
}

func NewAttachmentGormRepository(db *gorm.DB) *attachmentGormRepository {
	return &attachmentGormRepository{db: db}
}

func (r *attachmentGormRepository) Create(ctx context.Context, attachment *model.Attachment) error {
	return r.db.WithContext(ctx).Create(attachment).Error
}

func (r *attachmentGormRepository) BindTemporaryByURLs(ctx context.Context, userID, postID int64, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.Attachment{}).
		Where("user_id = ? AND post_id = 0 AND url IN ?", userID, urls).
		Update("post_id", postID).Error
}

func (r *attachmentGormRepository) FindExpiredTemporary(ctx context.Context, createdBefore time.Time) ([]model.Attachment, error) {
	var attachments []model.Attachment
	err := r.db.WithContext(ctx).
		Where("post_id = 0 AND created_at < ?", createdBefore).
		Find(&attachments).Error
	return attachments, err
}

func (r *attachmentGormRepository) DeleteByID(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Attachment{}, id).Error
}
