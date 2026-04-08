package biz

import (
	"context"
	"time"

	pb "GoPalette/api/user/v1"
	"GoPalette/internal/pkg/util"

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
	Update(context.Context, *User) (*User, error)
	Delete(context.Context, int64) error
	Get(context.Context, int64) (*User, error)
	List(ctx context.Context, page, pageSize int32) ([]*User, int32, error)
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

func (uc *UserUsecase) UpdateUser(ctx context.Context, u *User) (*User, error) {
	// 鉴权，暂时留空

	return uc.repo.Update(ctx, u)
}

func (uc *UserUsecase) DeleteUser(ctx context.Context, id int64) error {
	// 鉴权，暂时留空
	return uc.repo.Delete(ctx, id)
}

func (uc *UserUsecase) GetUser(ctx context.Context, id int64) (*User, error) {
	user, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, pb.ErrorUserNotFound("用户 ID %d 不存在", id)
	}
	return user, nil
}

func (uc *UserUsecase) ListUser(ctx context.Context, page, pageSize int32) ([]*User, int32, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return uc.repo.List(ctx, page, pageSize)
}

func (uc *UserUsecase) FindByEmail(ctx context.Context, email string) (*User, error) {
	return uc.repo.FindByEmail(ctx, email)
}
