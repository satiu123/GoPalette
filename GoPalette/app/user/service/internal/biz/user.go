package biz

import (
	"context"
	"time"

	pb "GoPalette/api/user/v1"
	"GoPalette/app/user/service/internal/pkg/util"
	"GoPalette/pkg/pagination"

	"github.com/go-kratos/kratos/v2/log"
)

type User struct {
	ID        int64
	Username  string
	Email     string
	Password  string
	Role      int32
	AvatarURL string
	Status    int32

	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepo interface {
	Create(context.Context, *User) (*User, error)
	Update(context.Context, *User, []string) (*User, error)
	Delete(context.Context, int64) error
	Get(context.Context, int64) (*User, error)
	List(ctx context.Context, page, pageSize int64) ([]*User, int64, error)
	FindByEmail(context.Context, string) (*User, error)
}

type UserUsecase struct {
	repo   UserRepo
	logger *log.Helper
}

func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{
		repo:   repo,
		logger: log.NewHelper(log.With(logger, "module", "usecase/user")),
	}
}

func (uc *UserUsecase) CreateUser(ctx context.Context, u *User) (*User, error) {
	exist, _ := uc.repo.FindByEmail(ctx, u.Email)
	if exist != nil {
		return nil, pb.ErrorEmailConflict("邮箱 %s 已存在", u.Email)
	}

	// 密码加密
	hashedPassword, err := util.HashPassword(u.Password)
	if err != nil {
		return nil, pb.ErrorInternalServerError("%s", "服务器开小差了，请稍后再试")
	}
	u.Password = hashedPassword

	return uc.repo.Create(ctx, u)
}

func (uc *UserUsecase) Register(ctx context.Context, u *User) (*User, error) {
	exist, _ := uc.repo.FindByEmail(ctx, u.Email)
	if exist != nil {
		return nil, pb.ErrorEmailConflict("邮箱 %s 已存在", u.Email)
	}

	// 密码加密
	hashedPassword, err := util.HashPassword(u.Password)
	if err != nil {
		return nil, pb.ErrorInternalServerError("%s", "服务器开小差了，请稍后再试")
	}
	u.Password = hashedPassword

	return uc.repo.Create(ctx, u)
}

// UpdateUser 仅更新基本信息，不允许修改密码和角色等敏感字段
func (uc *UserUsecase) UpdateUser(ctx context.Context, u *User, fields []string) (*User, error) {
	if err := CheckOwner(ctx, u.ID); err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, u, fields)
}

// DeleteUser 只能删除自己的账号，管理员可以删除其他用户
func (uc *UserUsecase) DeleteUser(ctx context.Context, id int64) error {
	if err := CheckOwner(ctx, id); err != nil {
		return err
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *UserUsecase) GetUser(ctx context.Context, id int64) (*User, error) {
	if err := CheckOwner(ctx, id); err != nil {
		return nil, err
	}
	user, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, pb.ErrorUserNotFound("用户 ID %d 不存在", id)
	}
	return user, nil
}

// ListUser 只有管理员可以查看用户列表
func (uc *UserUsecase) ListUser(ctx context.Context, page, pageSize int64) (*pagination.PageResult[*User], error) {
	if err := CheckAdmin(ctx); err != nil {
		return nil, err
	}
	list, total, err := uc.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	return pagination.NewPageResult(total, list), nil
}

func (uc *UserUsecase) FindByEmail(ctx context.Context, email string) (*User, error) {
	return uc.repo.FindByEmail(ctx, email)
}
