package repository

import (
	"context"

	"github.com/satiu123/GoPalette/internal/model"
)

type TagRepository interface {
	Create(ctx context.Context, tag *model.Tag) error
	FindAll(ctx context.Context) ([]model.Tag, error)
	FindByID(ctx context.Context, id int64) (*model.Tag, error)
	Delete(ctx context.Context, id int64) error
}
