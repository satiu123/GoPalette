package service

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/repository"
)

type CategoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) *CategoryService {
	return &CategoryService{categoryRepo: categoryRepo}
}

func (s *CategoryService) CreateCategory(ctx context.Context, name string) (*model.Category, error) {
	category := &model.Category{Name: name}
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *CategoryService) ListCategories(ctx context.Context) ([]model.Category, error) {
	return s.categoryRepo.FindAll(ctx)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id int64) error {
	category, err := s.categoryRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if category == nil {
		return errors.New("分类不存在")
	}
	return s.categoryRepo.Delete(ctx, id)
}
