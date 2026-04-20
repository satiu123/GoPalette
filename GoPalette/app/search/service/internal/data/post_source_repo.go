package data

import (
	"context"

	postv1 "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/app/search/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type postSourceRepo struct {
	data *Data
	log  *log.Helper
}

func NewPostSourceRepo(data *Data, logger log.Logger) biz.PostSourceRepo {
	return &postSourceRepo{data: data, log: log.NewHelper(log.With(logger, "module", "repo/post-source"))}
}

func (r *postSourceRepo) ListPostsByCursor(ctx context.Context, cursorID, pageSize int64, includeNonPublished bool) ([]*biz.SyncPost, int64, int64, bool, error) {
	resp, err := r.data.postClient.ListPostsForIndex(ctx, &postv1.ListPostsForIndexRequest{
		CursorId:            cursorID,
		PageSize:            pageSize,
		IncludeNonPublished: includeNonPublished,
	})
	if err != nil {
		return nil, 0, 0, false, err
	}
	items := make([]*biz.SyncPost, 0, len(resp.Posts))
	for _, p := range resp.Posts {
		sp := &biz.SyncPost{
			ID:           p.Id,
			Title:        p.Title,
			Summary:      p.Summary,
			Content:      p.Content,
			Slug:         p.Slug,
			CategoryName: p.CategoryName,
			Tags:         p.Tags,
		}
		if p.CreatedAt != nil {
			sp.CreatedAt = p.CreatedAt.AsTime()
		}
		items = append(items, sp)
	}
	return items, resp.Total, resp.NextCursorId, resp.HasMore, nil
}
