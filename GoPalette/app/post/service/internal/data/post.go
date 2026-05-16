package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/satiu123/GoPalette/pkg/pagination"

	"github.com/satiu123/GoPalette/app/post/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model
	Title           string `gorm:"size:255;not null"`
	Summary         string `gorm:"size:255"`
	Content         string `gorm:"type:text"`
	OriginalContent string `gorm:"type:text"`
	CoverURL        string `gorm:"size:1024"`
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

type PostLike struct {
	PostID    int64     `gorm:"primaryKey;autoIncrement:false"`
	UserID    int64     `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
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
func (PostLike) TableName() string { return "post_likes" }

func (r *postRepo) Create(ctx context.Context, p *biz.Post) (*biz.Post, error) {
	po := r.toDataPost(p)
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 处理标签,如果标签不存在则创建
		var tags []Tag
		for _, tagName := range p.Tags {
			var tag Tag
			if err := tx.Where(Tag{Name: tagName}).FirstOrCreate(&tag, Tag{Name: tagName, Slug: tagName}).Error; err != nil {
				return err
			}
			tags = append(tags, tag)
		}
		po.Tags = tags

		// 创建文章
		return tx.Create(po).Error
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, int64(po.ID))
}

func (r *postRepo) Update(ctx context.Context, p *biz.Post, fields []string) (*biz.Post, error) {
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		po := r.toDataPost(p)
		db := tx.Model(&Post{})

		// 只更新指定字段
		if len(fields) > 0 {
			db = db.Select(fields)
		}

		if err := db.Where("id = ?", p.ID).Updates(po).Error; err != nil {
			return err
		}

		// 如果更新了 tags 字段，则需要更新关联的标签
		if r.contains(fields, "tags") {
			var tags []Tag
			for _, tagName := range p.Tags {
				var tag Tag
				tx.Where(Tag{Name: tagName}).FirstOrCreate(&tag, Tag{Name: tagName, Slug: tagName})
				tags = append(tags, tag)
			}
			// 更新文章与标签的关联关系
			if err := tx.Model(po).Association("Tags").Replace(tags); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, p.ID)
}

func (r *postRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&Post{}, id).Error
}

func (r *postRepo) GetByID(ctx context.Context, id int64) (*biz.Post, error) {
	var po Post
	err := r.data.db.WithContext(ctx).
		Preload("Category").
		Preload("Tags").
		First(&po, id).Error
	if err != nil {
		return nil, err
	}
	return r.toBizPost(&po), nil
}

func (r *postRepo) GetBySlug(ctx context.Context, slug string) (*biz.Post, error) {
	var po Post
	err := r.data.db.WithContext(ctx).
		Preload("Category").
		Preload("Tags").
		Where("slug = ?", slug).
		First(&po).Error
	if err != nil {
		return nil, err
	}
	return r.toBizPost(&po), nil
}

func (r *postRepo) List(ctx context.Context, page, pageSize int64, publishedOnly bool) ([]*biz.Post, int64, error) {
	// 初始化分页参数
	p := pagination.NewPagingParam(page, pageSize)

	var pos []Post
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Post{})
	if publishedOnly {
		db = db.Where("status = ?", 1)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*biz.Post{}, 0, nil
	}

	err := db.Preload("Category").
		Preload("Tags").
		Order("id DESC").
		Scopes(pagination.Paginate(p)).
		Find(&pos).Error
	if err != nil {
		return nil, 0, err
	}

	var posts []*biz.Post
	for _, po := range pos {
		posts = append(posts, r.toBizPost(&po))
	}
	return posts, total, nil
}

func (r *postRepo) ListByAuthor(ctx context.Context, authorID, page, pageSize int64, publishedOnly bool) ([]*biz.Post, int64, error) {
	p := pagination.NewPagingParam(page, pageSize)

	var pos []Post
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Post{}).Where("author_id = ?", authorID)
	if publishedOnly {
		db = db.Where("status = ?", 1)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*biz.Post{}, 0, nil
	}

	err := db.Preload("Category").
		Preload("Tags").
		Order("id DESC").
		Scopes(pagination.Paginate(p)).
		Find(&pos).Error
	if err != nil {
		return nil, 0, err
	}

	posts := make([]*biz.Post, 0, len(pos))
	for _, po := range pos {
		posts = append(posts, r.toBizPost(&po))
	}

	return posts, total, nil
}

func (r *postRepo) GetAuthorStats(ctx context.Context, authorID int64, publishedOnly bool) (*biz.AuthorPostStats, error) {
	type row struct {
		Posts     int64
		Published int64
		Drafts    int64
		Archived  int64
		Private   int64
		Offline   int64
		Views     int64
		Likes     int64
		Comments  int64
	}

	query := r.data.db.WithContext(ctx).Model(&Post{}).Where("author_id = ?", authorID)
	if publishedOnly {
		query = query.Where("status = ?", 1)
	}

	var out row
	err := query.Select(
		"COUNT(*) AS posts, " +
			"SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS published, " +
			"SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) AS drafts, " +
			"SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END) AS archived, " +
			"SUM(CASE WHEN status = 3 THEN 1 ELSE 0 END) AS private, " +
			"SUM(CASE WHEN status = 4 THEN 1 ELSE 0 END) AS offline, " +
			"COALESCE(SUM(view_count), 0) AS views, " +
			"COALESCE(SUM(like_count), 0) AS likes, " +
			"COALESCE(SUM(comment_count), 0) AS comments",
	).Scan(&out).Error
	if err != nil {
		return nil, err
	}

	return &biz.AuthorPostStats{
		Posts:     out.Posts,
		Published: out.Published,
		Drafts:    out.Drafts,
		Archived:  out.Archived,
		Private:   out.Private,
		Offline:   out.Offline,
		Views:     out.Views,
		Likes:     out.Likes,
		Comments:  out.Comments,
	}, nil
}

func (r *postRepo) ListTopByAuthor(ctx context.Context, authorID, limit int64, publishedOnly bool) ([]*biz.Post, error) {
	var pos []Post

	query := r.data.db.WithContext(ctx).Model(&Post{}).Where("author_id = ?", authorID)
	if publishedOnly {
		query = query.Where("status = ?", 1)
	}

	err := query.Preload("Category").
		Preload("Tags").
		Order("view_count DESC, like_count DESC, created_at DESC").
		Limit(int(limit)).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	posts := make([]*biz.Post, 0, len(pos))
	for _, po := range pos {
		posts = append(posts, r.toBizPost(&po))
	}

	return posts, nil
}

func (r *postRepo) ListForIndex(ctx context.Context, page, pageSize int64, publishedOnly bool) ([]*biz.Post, int64, error) {
	p := pagination.NewPagingParam(page, pageSize)

	var pos []Post
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Post{})
	if publishedOnly {
		db = db.Where("status = ?", 1)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*biz.Post{}, 0, nil
	}

	err := db.Preload("Category").
		Preload("Tags").
		Order("id DESC").
		Scopes(pagination.Paginate(p)).
		Find(&pos).Error
	if err != nil {
		return nil, 0, err
	}

	posts := make([]*biz.Post, 0, len(pos))
	for _, po := range pos {
		posts = append(posts, r.toBizPost(&po))
	}
	return posts, total, nil
}

func (r *postRepo) ListForIndexByCursor(ctx context.Context, cursorID, pageSize int64, publishedOnly bool) ([]*biz.Post, int64, int64, bool, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}
	if pageSize > 5000 {
		pageSize = 5000
	}

	db := r.data.db.WithContext(ctx).Model(&Post{})
	if publishedOnly {
		db = db.Where("status = ?", 1)
	}
	if cursorID > 0 {
		db = db.Where("id < ?", cursorID)
	}

	var total int64
	if cursorID == 0 {
		if err := db.Count(&total).Error; err != nil {
			return nil, 0, 0, false, err
		}
		if total == 0 {
			return []*biz.Post{}, 0, 0, false, nil
		}
	}

	var pos []Post
	err := db.Preload("Category").
		Preload("Tags").
		Order("id DESC").
		Limit(int(pageSize)).
		Find(&pos).Error
	if err != nil {
		return nil, 0, 0, false, err
	}
	if len(pos) == 0 {
		return []*biz.Post{}, total, 0, false, nil
	}

	posts := make([]*biz.Post, 0, len(pos))
	for _, po := range pos {
		posts = append(posts, r.toBizPost(&po))
	}

	nextCursorID := int64(pos[len(pos)-1].ID)
	hasMore := len(pos) == int(pageSize)
	return posts, total, nextCursorID, hasMore, nil
}

// 将数据层的 Post 转换为业务层的 Post
func (r *postRepo) toBizPost(po *Post) *biz.Post {
	return &biz.Post{
		ID:              int64(po.ID),
		Title:           po.Title,
		Summary:         po.Summary,
		Content:         po.Content,
		OriginalContent: po.OriginalContent,
		CoverURL:        po.CoverURL,
		Slug:            po.Slug,
		Status:          po.Status,

		ViewCount:    po.ViewCount,
		LikeCount:    po.LikeCount,
		CommentCount: po.CommentCount,

		AuthorID:     po.AuthorID,
		CategoryID:   po.CategoryID,
		CategoryName: po.Category.Name,
		Tags: func() []string {
			var tagNames []string
			for _, tag := range po.Tags {
				tagNames = append(tagNames, tag.Name)
			}
			return tagNames
		}(),

		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

// 将业务层的 Post 转换为数据层的 Post (不包含 tags)
func (r *postRepo) toDataPost(p *biz.Post) *Post {
	return &Post{
		Model: gorm.Model{
			ID:        uint(p.ID),
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		Title:           p.Title,
		Summary:         p.Summary,
		Content:         p.Content,
		OriginalContent: p.OriginalContent,
		CoverURL:        p.CoverURL,
		Slug:            p.Slug,
		Status:          p.Status,
		AuthorID:        p.AuthorID,
		CategoryID:      p.CategoryID,
	}
}

func (r *postRepo) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (r *postRepo) IncrCommentCount(ctx context.Context, id int64, delta int64) error {
	return r.data.db.WithContext(ctx).Model(&Post{}).
		Where("id = ?", id).
		Update("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error
}

func (r *postRepo) RecordView(ctx context.Context, id int64, viewerKey string, window time.Duration) (bool, int64, error) {
	key := fmt.Sprintf("post:view:%d:%s", id, viewerKey)
	counted, err := r.data.rdb.SetNX(ctx, key, "1", window).Result()
	if err != nil {
		return false, 0, err
	}

	if counted {
		result := r.data.db.WithContext(ctx).Model(&Post{}).
			Where("id = ? AND status = ?", id, 1).
			Update("view_count", gorm.Expr("view_count + 1"))
		if result.Error != nil {
			_ = r.data.rdb.Del(ctx, key).Err()
			return false, 0, result.Error
		}
		if result.RowsAffected == 0 {
			_ = r.data.rdb.Del(ctx, key).Err()
			counted = false
		}
	}

	var viewCount int64
	if err := r.data.db.WithContext(ctx).Model(&Post{}).
		Where("id = ?", id).
		Select("view_count").
		Scan(&viewCount).Error; err != nil {
		return counted, 0, err
	}

	return counted, viewCount, nil
}

func (r *postRepo) ToggleLike(ctx context.Context, postID, userID int64) (bool, int64, error) {
	liked := false
	var likeCount int64
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var relation PostLike
		err := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&relation).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Create(&PostLike{PostID: postID, UserID: userID}).Error; err != nil {
				return err
			}
			if err := tx.Model(&Post{}).
				Where("id = ?", postID).
				Update("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			liked = true
		} else {
			if err := tx.Where("post_id = ? AND user_id = ?", postID, userID).Delete(&PostLike{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&Post{}).
				Where("id = ?", postID).
				Update("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Post{}).Where("id = ?", postID).Select("like_count").Scan(&likeCount).Error
	})
	if err != nil {
		return false, 0, err
	}
	return liked, likeCount, nil
}

func (r *postRepo) GetLikeState(ctx context.Context, postID, userID int64) (bool, int64, error) {
	var likeCount int64
	if err := r.data.db.WithContext(ctx).Model(&Post{}).
		Where("id = ?", postID).
		Select("like_count").
		Scan(&likeCount).Error; err != nil {
		return false, 0, err
	}

	var total int64
	if err := r.data.db.WithContext(ctx).Model(&PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&total).Error; err != nil {
		return false, 0, err
	}

	return total > 0, likeCount, nil
}

func (r *postRepo) ListLikedByUser(ctx context.Context, userID, page, pageSize int64) ([]*biz.Post, int64, error) {
	p := pagination.NewPagingParam(page, pageSize)

	var total int64
	db := r.data.db.WithContext(ctx).Model(&Post{}).
		Joins("JOIN post_likes ON post_likes.post_id = posts.id").
		Where("post_likes.user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*biz.Post{}, 0, nil
	}

	var pos []Post
	err := db.Preload("Category").
		Preload("Tags").
		Order("post_likes.created_at DESC").
		Scopes(pagination.Paginate(p)).
		Find(&pos).Error
	if err != nil {
		return nil, 0, err
	}

	posts := make([]*biz.Post, 0, len(pos))
	for i := range pos {
		posts = append(posts, r.toBizPost(&pos[i]))
	}
	return posts, total, nil
}
