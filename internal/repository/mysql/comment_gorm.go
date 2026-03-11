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
