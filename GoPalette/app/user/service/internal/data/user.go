package data

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/satiu123/GoPalette/pkg/pagination"

	"github.com/satiu123/GoPalette/app/user/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username    string `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email       string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Password    string `gorm:"type:varchar(255);not null"`
	Role        int32  `gorm:"type:int;default:0"` // 0: user, 1: admin
	AvatarURL   string `gorm:"type:varchar(255)"`
	Status      int32  `gorm:"type:int;default:0"` // 0: active, 1: inactive
	Bio         string `gorm:"type:varchar(512)"`
	SocialLinks string `gorm:"type:json"`
	Location    string `gorm:"type:varchar(255)"`
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
	socialLinks, err := marshalSocialLinks(u.SocialLinks)
	if err != nil {
		return nil, err
	}
	po := &User{
		Username:    u.Username,
		Password:    u.Password,
		Email:       u.Email,
		Role:        u.Role,
		Status:      u.Status,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		SocialLinks: socialLinks,
		Location:    u.Location,
	}

	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	return r.Get(ctx, int64(po.ID))
}

func (r *userRepo) Update(ctx context.Context, u *biz.User, fields []string) (*biz.User, error) {
	updates := make(map[string]any, len(fields))
	for _, field := range fields {
		switch field {
		case "username":
			updates["username"] = u.Username
		case "email":
			updates["email"] = u.Email
		case "avatar_u_r_l", "avatarURL":
			updates["avatar_url"] = u.AvatarURL
		case "status":
			updates["status"] = u.Status
		case "bio":
			updates["bio"] = u.Bio
		case "location":
			updates["location"] = u.Location
		case "social_links", "socialLinks":
			socialLinks, err := marshalSocialLinks(u.SocialLinks)
			if err != nil {
				return nil, err
			}
			updates["social_links"] = socialLinks
		}
	}

	if len(updates) == 0 {
		return nil, errors.New("no valid update fields")
	}

	if err := r.data.db.WithContext(ctx).Model(&User{}).Where("id = ?", u.ID).Updates(updates).Error; err != nil {
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
	return r.toBizUser(&po), nil
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
		users[i] = r.toBizUser(&po)
	}
	return users, total, nil
}

func (r *userRepo) ListByIDs(ctx context.Context, ids []int64) ([]*biz.User, error) {
	if len(ids) == 0 {
		return []*biz.User{}, nil
	}

	var pos []User
	if err := r.data.db.WithContext(ctx).Where("id IN ?", ids).Find(&pos).Error; err != nil {
		return nil, err
	}

	users := make([]*biz.User, 0, len(pos))
	for i := range pos {
		users = append(users, r.toBizUser(&pos[i]))
	}
	return users, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*biz.User, error) {
	var po User
	if err := r.data.db.WithContext(ctx).Where("email = ?", email).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toBizUser(&po), nil
}

func (r *userRepo) toBizUser(po *User) *biz.User {
	socialLinks, _ := unmarshalSocialLinks(po.SocialLinks)
	return &biz.User{
		ID:          int64(po.ID),
		Username:    po.Username,
		Email:       po.Email,
		Password:    po.Password,
		Role:        po.Role,
		AvatarURL:   po.AvatarURL,
		Status:      po.Status,
		Bio:         po.Bio,
		SocialLinks: socialLinks,
		Location:    po.Location,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func marshalSocialLinks(links map[string]string) (string, error) {
	if links == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(links)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func unmarshalSocialLinks(raw string) (map[string]string, error) {
	if raw == "" {
		return map[string]string{}, nil
	}
	links := make(map[string]string)
	if err := json.Unmarshal([]byte(raw), &links); err != nil {
		return map[string]string{}, err
	}
	return links, nil
}
