package data

import (
	"GoPalette/app/post/service/internal/biz"
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model
	Title           string `gorm:"size:255;not null"`
	Summary         string `gorm:"size:255"`
	Content         string `gorm:"type:text"`
	OriginalContent string `gorm:"type:text"`
	Slug            string `gorm:"size:255;not null;unique"`
	Status          int32  `gorm:"default:0"`

	ViewCount    int64 `gorm:"default:0"`
	LikeCount    int64 `gorm:"default:0"`
	CommentCount int64 `gorm:"default:0"`

	AuthorID   int64    `gorm:"index"`
	CategoryID int64    `gorm:"index"`
	Category   Category `gorm:"foreignKey:CategoryID"`
	Tags       []Tag    `gorm:"many2many:post_tags;"`
}

type Category struct {
	gorm.Model
	Name string `gorm:"size:255;not null"`
	Slug string `gorm:"size:255;not null;unique"`

	Description string `gorm:"size:255"`
	PostCount   int64  `gorm:"default:0"`
}

type Tag struct {
	gorm.Model
	Name string `gorm:"size:255;not null"`
	Slug string `gorm:"size:255;not null;unique"`

	PostCount int64 `gorm:"default:0"`
}

type postRepo struct {
	data *Data
	log  *log.Helper
}

func NewPostRepo(data *Data, logger log.Logger) biz.PostRepo {
	return &postRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "repo/post")),
	}
}

func (Post) TableName() string     { return "posts" }
func (Category) TableName() string { return "categories" }
func (Tag) TableName() string      { return "tags" }

func (r *postRepo) Create(ctx context.Context, p *biz.Post) (*biz.Post, error) {
	return nil, nil
}

func (r *postRepo) Update(ctx context.Context, p *biz.Post, fields []string) (*biz.Post, error) {
	return nil, nil
}

func (r *postRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

func (r *postRepo) GetByID(ctx context.Context, id int64) (*biz.Post, error) {
	return nil, nil
}

func (r *postRepo) GetBySlug(ctx context.Context, slug string) (*biz.Post, error) {
	return nil, nil
}

func (r *postRepo) List(ctx context.Context, page, pageSize int64) ([]*biz.Post, int64, error) {
	return nil, 0, nil
}
