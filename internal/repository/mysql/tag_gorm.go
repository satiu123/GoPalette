package mysql

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/internal/model"
	"gorm.io/gorm"
)

type tagGormRepository struct {
	db *gorm.DB
}

func NewTagGormRepository(db *gorm.DB) *tagGormRepository {
	return &tagGormRepository{db: db}
}

func (r *tagGormRepository) Create(ctx context.Context, tag *model.Tag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}

func (r *tagGormRepository) FindAll(ctx context.Context) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.WithContext(ctx).Order("id ASC").Find(&tags).Error
	return tags, err
}

func (r *tagGormRepository) FindByID(ctx context.Context, id int64) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.WithContext(ctx).First(&tag, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tag, nil
}

func (r *tagGormRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Tag{}, id).Error
}
