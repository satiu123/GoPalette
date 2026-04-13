package biz

import (
	"context"
	"math"
	"strings"
	"time"

	pb "GoPalette/api/search/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type PostSearch struct {
	ID           int64
	Title        string
	Summary      string
	Slug         string
	CategoryName string
	Tags         []string
	CreatedAt    time.Time
}

type SyncPost struct {
	ID           int64
	Title        string
	Summary      string
	Content      string
	Slug         string
	CategoryName string
	Tags         []string
	CreatedAt    time.Time
}

type SearchRepo interface {
	SearchPosts(ctx context.Context, query string, offset, limit int64, category string) ([]*PostSearch, int64, error)
	SyncPost(ctx context.Context, p *SyncPost) error
	DeletePost(ctx context.Context, postID int64) error
	ResetIndex(ctx context.Context) error
	SyncPostsBatch(ctx context.Context, posts []*SyncPost) error
}

type PostSourceRepo interface {
	ListPosts(ctx context.Context, page, pageSize int64, includeNonPublished bool) ([]*SyncPost, int64, error)
}

type SearchUsecase struct {
	repo       SearchRepo
	postSource PostSourceRepo
	logger     *log.Helper
}

func NewSearchUsecase(repo SearchRepo, postSource PostSourceRepo, logger log.Logger) *SearchUsecase {
	return &SearchUsecase{
		repo:       repo,
		postSource: postSource,
		logger:     log.NewHelper(log.With(logger, "module", "usecase/search")),
	}
}

func (uc *SearchUsecase) SearchPosts(ctx context.Context, query string, page, pageSize int64, category string) ([]*PostSearch, int64, int64, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, 0, 0, pb.ErrorInvalidArgument("%s", "搜索关键词不能为空")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	items, total, err := uc.repo.SearchPosts(ctx, query, offset, pageSize, category)
	if err != nil {
		return nil, 0, 0, err
	}
	totalPages := int64(math.Ceil(float64(total) / float64(pageSize)))
	if total == 0 {
		totalPages = 0
	}
	return items, total, totalPages, nil
}

func (uc *SearchUsecase) SyncPost(ctx context.Context, p *SyncPost) error {
	if p == nil || p.ID <= 0 {
		return pb.ErrorInvalidArgument("%s", "post 数据无效")
	}
	if strings.TrimSpace(p.Title) == "" {
		return pb.ErrorInvalidArgument("%s", "标题不能为空")
	}
	return uc.repo.SyncPost(ctx, p)
}

func (uc *SearchUsecase) DeleteIndex(ctx context.Context, postID int64) error {
	if postID <= 0 {
		return pb.ErrorInvalidArgument("%s", "post_id 无效")
	}
	return uc.repo.DeletePost(ctx, postID)
}

func (uc *SearchUsecase) RebuildIndex(ctx context.Context, resetFirst bool, includeNonPublished bool) (int64, error) {
	if resetFirst {
		if err := uc.repo.ResetIndex(ctx); err != nil {
			return 0, err
		}
	}

	const pageSize int64 = 100
	var page int64 = 1
	var indexed int64
	for {
		posts, total, err := uc.postSource.ListPosts(ctx, page, pageSize, includeNonPublished)
		if err != nil {
			return indexed, err
		}
		if len(posts) == 0 {
			break
		}
		if err := uc.repo.SyncPostsBatch(ctx, posts); err != nil {
			return indexed, err
		}
		indexed += int64(len(posts))
		if page*pageSize >= total {
			break
		}
		page++
	}
	return indexed, nil
}
