package repository

import (
	"context"

	"github.com/satiu123/GoPalette/internal/model"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	FindByArticleID(ctx context.Context, articleID int64) ([]model.Comment, error)
	FindByID(ctx context.Context, id int64) (*model.Comment, error)
	Delete(ctx context.Context, id int64) error
}
