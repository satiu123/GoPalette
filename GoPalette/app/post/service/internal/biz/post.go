package biz

import (
	"context"
	"time"

	pb "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type Post struct {
	ID              int64
	Title           string
	Summary         string
	Content         string
	OriginalContent string
	Slug            string
	Status          int32

	ViewCount    int64
	LikeCount    int64
	CommentCount int64

	AuthorID     int64
	CategoryID   int64
	CategoryName string
	Tags         []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type PostRepo interface {
	Create(context.Context, *Post) (*Post, error)
	Update(context.Context, *Post, []string) (*Post, error)
	Delete(context.Context, int64) error
	GetByID(context.Context, int64) (*Post, error)
	GetBySlug(context.Context, string) (*Post, error)
	List(ctx context.Context, page, pageSize int64) ([]*Post, int64, error)
	ListForIndex(ctx context.Context, page, pageSize int64, publishedOnly bool) ([]*Post, int64, error)
	IncrCommentCount(ctx context.Context, id int64, delta int64) error
}

type PostUsecase struct {
	repo   PostRepo
	logger *log.Helper
}

func NewPostUsecase(repo PostRepo, logger log.Logger) *PostUsecase {
	return &PostUsecase{
		repo:   repo,
		logger: log.NewHelper(log.With(logger, "module", "usecase/post")),
	}
}

func (uc *PostUsecase) CreatePost(ctx context.Context, p *Post) (*Post, error) {
	// 标题不能为空
	if p.Title == "" {
		return nil, pb.ErrorInvalidArgument("%s", "标题不能为空")
	}

	// SLUG 不能为空且必须唯一
	if p.Slug == "" {
		return nil, pb.ErrorInvalidArgument("%s", "Slug 不能为空")
	}
	exist, _ := uc.repo.GetBySlug(ctx, p.Slug)
	if exist != nil {
		return nil, pb.ErrorSlugConflict("Slug %s 已存在", p.Slug)
	}

	// 如果摘要为空，则自动生成摘要
	if p.Summary == "" {
		if len(p.Content) > 100 {
			p.Summary = p.Content[:100]
		} else {
			p.Summary = p.Content
		}
	}
	uc.logger.WithContext(ctx).Infof("%s 正在创建文章: %s", p.AuthorID, p.Title)
	return uc.repo.Create(ctx, p)
}

func (uc *PostUsecase) UpdatePost(ctx context.Context, p *Post, fields []string) (*Post, error) {
	oldPost, err := uc.repo.GetByID(ctx, p.ID)
	if err != nil {
		return nil, pb.ErrorPostNotFound("文章未找到")
	}

	if err := CheckOwner(ctx, oldPost.AuthorID); err != nil {
		return nil, err
	}
	uc.logger.WithContext(ctx).Infof("%s 正在更新文章: %s", p.AuthorID, p.Title)
	return uc.repo.Update(ctx, p, fields)
}

func (uc *PostUsecase) GetPost(ctx context.Context, id int64, slug string) (*Post, error) {
	var post *Post
	var err error
	if id > 0 {
		post, err = uc.repo.GetByID(ctx, id)
	} else if slug != "" {
		post, err = uc.repo.GetBySlug(ctx, slug)
	} else {
		return nil, pb.ErrorInvalidArgument("%s", "id 和 slug 不能同时为空")
	}
	if err != nil {
		return nil, pb.ErrorPostNotFound("文章未找到")
	}
	return post, nil
}

func (uc *PostUsecase) DeletePost(ctx context.Context, id int64) error {
	post, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return pb.ErrorPostNotFound("文章未找到")
	}

	if err := CheckOwner(ctx, post.AuthorID); err != nil {
		return err
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *PostUsecase) ListPosts(ctx context.Context, page, pageSize int64) ([]*Post, int64, error) {
	return uc.repo.List(ctx, page, pageSize)
}

func (uc *PostUsecase) ListPostsForIndex(ctx context.Context, page, pageSize int64, publishedOnly bool) ([]*Post, int64, error) {
	return uc.repo.ListForIndex(ctx, page, pageSize, publishedOnly)
}

func (uc *PostUsecase) IncrCommentCount(ctx context.Context, id int64, delta int64) error {
	if id <= 0 {
		return pb.ErrorInvalidArgument("%s", "文章ID无效")
	}
	if delta == 0 {
		return nil
	}
	if _, err := uc.repo.GetByID(ctx, id); err != nil {
		return pb.ErrorPostNotFound("文章未找到")
	}
	return uc.repo.IncrCommentCount(ctx, id, delta)
}
