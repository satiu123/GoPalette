package repository

import (
	"context"

	"github.com/satiu123/GoPalette/internal/model"
)

type ListArticlesFilter struct {
	CategoryID int64
	TagID      int64
	Status     string
}

type ArticleRepository interface {
	Create(ctx context.Context, article *model.Article) error
	FindByID(ctx context.Context, id int64) (*model.Article, error)
	FindAll(ctx context.Context, page, pageSize int, filter ListArticlesFilter) ([]model.Article, int64, error)
	Update(ctx context.Context, article *model.Article) error
	Delete(ctx context.Context, id int64) error
	IncrReadCount(ctx context.Context, id int64) error
}
