package mysql

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/internal/model"
	"gorm.io/gorm"
)

type categoryGormRepository struct {
	db *gorm.DB
}

func NewCategoryGormRepository(db *gorm.DB) *categoryGormRepository {
	return &categoryGormRepository{db: db}
}

func (r *categoryGormRepository) Create(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *categoryGormRepository) FindAll(ctx context.Context) ([]model.Category, error) {
	var categories []model.Category
	err := r.db.WithContext(ctx).Order("id ASC").Find(&categories).Error
	return categories, err
}

func (r *categoryGormRepository) FindByID(ctx context.Context, id int64) (*model.Category, error) {
	var category model.Category
	err := r.db.WithContext(ctx).First(&category, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *categoryGormRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Category{}, id).Error
}
