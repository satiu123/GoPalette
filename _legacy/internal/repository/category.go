package repository

import (
	"context"

	"github.com/satiu123/GoPalette/internal/model"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *model.Category) error
	FindAll(ctx context.Context) ([]model.Category, error)
	FindByID(ctx context.Context, id int64) (*model.Category, error)
	Delete(ctx context.Context, id int64) error
}
