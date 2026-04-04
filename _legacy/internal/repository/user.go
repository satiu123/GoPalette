package repository

import (
	"context"

	"github.com/satiu123/GoPalette/internal/model"
)

// UserRepository 定义了用户数据操作的标准接口
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id int64) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
}
