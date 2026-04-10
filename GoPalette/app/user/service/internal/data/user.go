package data

import (
	"GoPalette/app/user/service/internal/biz"
	"GoPalette/pkg/pagination"
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username  string `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email     string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Password  string `gorm:"type:varchar(255);not null"`
	Role      int32  `gorm:"type:int;default:1"` // 0: admin, 1: user
	AvatarURL string `gorm:"type:varchar(255)"`
	Status    int32  `gorm:"type:int;default:0"` // 0: active, 1: inactive
}

func (User) TableName() string { return "users" }

type userRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "repo/user")),
	}
}

func (r *userRepo) Create(ctx context.Context, u *biz.User) (*biz.User, error) {
	po := &User{
		Username: u.Username,
		Password: u.Password,
		Email:    u.Email,
		Role:     u.Role,
		Status:   u.Status,
	}

	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	u.ID = int64(po.ID)
	u.CreatedAt = po.CreatedAt
	u.UpdatedAt = po.UpdatedAt
	return u, nil
}

func (r *userRepo) Update(ctx context.Context, u *biz.User, fields []string) (*biz.User, error) {
	db := r.data.db.WithContext(ctx).Model(&User{})
	po := &User{
		Model:     gorm.Model{ID: uint(u.ID)},
		Username:  u.Username,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		Status:    u.Status,
	}
	// 只更新指定字段
	if len(fields) > 0 {
		db = db.Select(fields)
	}

	if err := db.Where("id = ?", u.ID).Updates(po).Error; err != nil {
		return nil, err
	}
	return r.Get(ctx, u.ID)
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&User{}, id).Error
}

func (r *userRepo) Get(ctx context.Context, id int64) (*biz.User, error) {
	var po User
	if err := r.data.db.WithContext(ctx).First(&po, id).Error; err != nil {
		return nil, err
	}
	return &biz.User{
		ID:        int64(po.ID),
		Username:  po.Username,
		Email:     po.Email,
		Password:  po.Password,
		Role:      po.Role,
		AvatarURL: po.AvatarURL,
		Status:    po.Status,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}, nil
}

func (r *userRepo) List(ctx context.Context, page, pageSize int64) ([]*biz.User, int64, error) {
	// 初始化分页参数
	p := pagination.NewPagingParam(page, pageSize)

	var pos []User
	var total int64

	db := r.data.db.WithContext(ctx).Model(&User{})

	// 统计总记录数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 如果没有记录，直接返回空列表和总数0
	if total == 0 {
		return []*biz.User{}, 0, nil
	}

	if err := db.Scopes(pagination.Paginate(p)).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	// 将数据转换为 biz.User 切片
	users := make([]*biz.User, len(pos))
	for i, po := range pos {
		users[i] = &biz.User{
			ID:        int64(po.ID),
			Username:  po.Username,
			Email:     po.Email,
			Password:  po.Password,
			Role:      po.Role,
			AvatarURL: po.AvatarURL,
			Status:    po.Status,
			CreatedAt: po.CreatedAt,
			UpdatedAt: po.UpdatedAt,
		}
	}
	return users, total, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*biz.User, error) {
	var po User
	if err := r.data.db.WithContext(ctx).Where("email = ?", email).First(&po).Error; err != nil {
		return nil, err
	}
	return &biz.User{
		ID:        int64(po.ID),
		Username:  po.Username,
		Email:     po.Email,
		Password:  po.Password,
		Role:      po.Role,
		AvatarURL: po.AvatarURL,
		Status:    po.Status,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}, nil
}
