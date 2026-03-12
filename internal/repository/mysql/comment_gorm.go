package mysql

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/internal/model"
	"gorm.io/gorm"
)

type commentGormRepository struct {
	db *gorm.DB
}

func NewCommentGormRepository(db *gorm.DB) *commentGormRepository {
	return &commentGormRepository{db: db}
}

func (r *commentGormRepository) Create(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *commentGormRepository) FindByArticleID(ctx context.Context, articleID int64) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("article_id = ?", articleID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *commentGormRepository) FindByID(ctx context.Context, id int64) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.WithContext(ctx).First(&comment, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &comment, nil
}

func (r *commentGormRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Comment{}, id).Error
}

// FindAll 分页查询全部评论（含用户信息、文章标题）
func (r *commentGormRepository) FindAll(ctx context.Context, page, pageSize int) ([]model.Comment, int64, error) {
	var comments []model.Comment
	var total int64
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Model(&model.Comment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Article").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&comments).Error
	return comments, total, err
}
