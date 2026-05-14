package biz

import (
	"context"
	"strings"
	"time"

	"github.com/satiu123/GoPalette/pkg/auth"
	"github.com/satiu123/GoPalette/pkg/pagination"

	pb "github.com/satiu123/GoPalette/api/user/v1"

	"github.com/satiu123/GoPalette/app/user/service/internal/pkg/util"

	"github.com/go-kratos/kratos/v2/log"
)

type User struct {
	ID          int64
	Username    string
	Email       string
	Password    string
	Role        int32
	AvatarURL   string
	Status      int32
	Bio         string
	SocialLinks map[string]string
	Location    string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepo interface {
	Create(context.Context, *User) (*User, error)
	Update(context.Context, *User, []string) (*User, error)
	Delete(context.Context, int64) error
	Get(context.Context, int64) (*User, error)
	ListByIDs(context.Context, []int64) ([]*User, error)
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
	if err := CheckAdmin(ctx); err != nil {
		return nil, err
	}

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

func (uc *UserUsecase) UpdateUser(ctx context.Context, u *User, fields []string) (*User, error) {
	if err := CheckOwner(ctx, u.ID); err != nil {
		return nil, err
	}

	// 如果不是管理员，强制移除敏感字段
	if err := CheckAdmin(ctx); err != nil {
		u.Password = ""
		u.Role = 0

		// 从更新字段中移除敏感字段
		fields = filterSensitiveFields(fields)
	}
	fields = normalizeUserUpdateFields(fields)

	if len(fields) == 0 {
		return nil, pb.ErrorInvalidArgument("没有可更新字段")
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

	user, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, pb.ErrorUserNotFound("用户 ID %d 不存在", id)
	}
	if !isSelf(ctx, id) {
		user.Email = ""
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

func (uc *UserUsecase) ListUsersByIDs(ctx context.Context, ids []int64) ([]*User, error) {
	if len(ids) == 0 {
		return []*User{}, nil
	}
	return uc.repo.ListByIDs(ctx, ids)
}

func filterSensitiveFields(fields []string) []string {
	filtered := make([]string, 0)
	for _, f := range fields {
		switch strings.TrimSpace(f) {
		case "password", "role":
			continue
		default:
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func normalizeUserUpdateFields(fields []string) []string {
	normalized := make([]string, 0, len(fields))
	for _, field := range fields {
		switch strings.TrimSpace(field) {
		case "username", "email", "role", "status", "bio", "location", "social_links", "socialLinks", "avatar_u_r_l", "avatarURL":
			normalized = append(normalized, field)
		}
	}
	return normalized
}

func isSelf(ctx context.Context, targetID int64) bool {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return false
	}
	return claims.UserID == targetID
}
