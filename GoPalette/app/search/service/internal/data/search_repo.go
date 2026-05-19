package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "github.com/satiu123/GoPalette/api/search/v1"

	"github.com/satiu123/GoPalette/app/search/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/meilisearch/meilisearch-go"
)

type searchRepo struct {
	data *Data
	log  *log.Helper
}

type indexedPost struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	Content      string    `json:"content"`
	Slug         string    `json:"slug"`
	CategoryName string    `json:"category_name"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewSearchRepo(data *Data, logger log.Logger) biz.SearchRepo {
	return &searchRepo{data: data, log: log.NewHelper(log.With(logger, "module", "repo/search"))}
}

func initIndexSettings(d *Data) error {
	if err := ensureIndex(d); err != nil {
		return err
	}
	index := d.meili.Index(d.indexName)
	if _, err := index.UpdateSearchableAttributes(&[]string{"title", "summary", "content", "tags"}); err != nil {
		return err
	}
	if _, err := index.UpdateFilterableAttributes(&[]any{"category_name", "tags"}); err != nil {
		return err
	}
	if _, err := index.UpdateSortableAttributes(&[]string{"created_at"}); err != nil {
		return err
	}
	_, err := index.UpdateRankingRules(&[]string{
		"words",
		"typo",
		"proximity",
		"attribute",
		"sort",
		"exactness",
		"created_at:desc",
	})
	return err
}

func ensureIndex(d *Data) error {
	if _, err := d.meili.GetIndex(d.indexName); err == nil {
		return nil
	} else if !isMeiliIndexNotFound(err) {
		return err
	}

	if _, err := d.meili.CreateIndex(&meilisearch.IndexConfig{Uid: d.indexName, PrimaryKey: "id"}); err != nil && !isMeiliIndexAlreadyExists(err) {
		return err
	}
	return nil
}

func isMeiliIndexNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "index_not_found")
}

func isMeiliIndexAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "index_already_exists")
}

func (r *searchRepo) SearchPosts(ctx context.Context, query string, offset, limit int64, category string) ([]*biz.PostSearch, int64, error) {
	_ = ctx
	index := r.data.meili.Index(r.data.indexName)
	searchReq := &meilisearch.SearchRequest{
		Offset:                offset,
		Limit:                 limit,
		AttributesToHighlight: []string{"title", "summary"},
		HighlightPreTag:       "<em>",
		HighlightPostTag:      "</em>",
	}
	if category != "" {
		searchReq.Filter = fmt.Sprintf("category_name = \"%s\"", category)
	}
	res, err := index.Search(query, searchReq)
	if err != nil {
		return nil, 0, pb.ErrorSearchFailed("%v", err)
	}

	items := make([]*biz.PostSearch, 0, len(res.Hits))
	for _, hit := range res.Hits {
		b, _ := json.Marshal(hit)
		var raw map[string]any
		if err := json.Unmarshal(b, &raw); err != nil {
			continue
		}
		ps := &biz.PostSearch{Tags: []string{}}
		if v, ok := raw["id"].(float64); ok {
			ps.ID = int64(v)
		}
		if v, ok := raw["slug"].(string); ok {
			ps.Slug = v
		}
		if v, ok := raw["category_name"].(string); ok {
			ps.CategoryName = v
		}
		if tags, ok := raw["tags"].([]any); ok {
			for _, t := range tags {
				if s, ok := t.(string); ok {
					ps.Tags = append(ps.Tags, s)
				}
			}
		}
		ps.CreatedAt = parseIndexedTime(raw, "created_at", "createdAt")
		if f, ok := raw["_formatted"].(map[string]any); ok {
			if t, ok := f["title"].(string); ok {
				ps.Title = t
			}
			if s, ok := f["summary"].(string); ok {
				ps.Summary = s
			}
		}
		if ps.Title == "" {
			if t, ok := raw["title"].(string); ok {
				ps.Title = t
			}
		}
		if ps.Summary == "" {
			if s, ok := raw["summary"].(string); ok {
				ps.Summary = s
			}
		}
		items = append(items, ps)
	}

	var total int64
	if res.EstimatedTotalHits > 0 {
		total = res.EstimatedTotalHits
	}
	if total == 0 {
		total = int64(len(items))
	}
	return items, total, nil
}

func parseIndexedTime(raw map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if parsed, ok := parseIndexedTimeValue(value); ok {
			return parsed
		}
	}
	return time.Time{}
}

func parseIndexedTimeValue(value any) (time.Time, bool) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return time.Time{}, false
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if parsed, err := time.Parse(layout, v); err == nil {
				return parsed, true
			}
		}
	case float64:
		if v <= 0 {
			return time.Time{}, false
		}
		return time.Unix(int64(v), 0).UTC(), true
	case json.Number:
		if unix, err := v.Int64(); err == nil && unix > 0 {
			return time.Unix(unix, 0).UTC(), true
		}
	}
	return time.Time{}, false
}

func (r *searchRepo) SyncPost(ctx context.Context, p *biz.SyncPost) error {
	_ = ctx
	doc := indexedPost{
		ID:           p.ID,
		Title:        p.Title,
		Summary:      p.Summary,
		Content:      p.Content,
		Slug:         p.Slug,
		CategoryName: p.CategoryName,
		Tags:         p.Tags,
		CreatedAt:    p.CreatedAt,
	}
	_, err := r.data.meili.Index(r.data.indexName).AddDocuments([]indexedPost{doc}, nil)
	if err != nil {
		return pb.ErrorIndexSyncFailed("%v", err)
	}
	return nil
}

func (r *searchRepo) DeletePost(ctx context.Context, postID int64) error {
	_ = ctx
	_, err := r.data.meili.Index(r.data.indexName).DeleteDocument(strconv.FormatInt(postID, 10), nil)
	if err != nil {
		return pb.ErrorIndexSyncFailed("%v", err)
	}
	return nil
}

func (r *searchRepo) ResetIndex(ctx context.Context) error {
	_ = ctx
	_, err := r.data.meili.Index(r.data.indexName).DeleteAllDocuments(nil)
	if err != nil {
		return pb.ErrorIndexSyncFailed("%v", err)
	}
	return nil
}

func (r *searchRepo) SyncPostsBatch(ctx context.Context, posts []*biz.SyncPost) error {
	_ = ctx
	if len(posts) == 0 {
		return nil
	}
	docs := make([]indexedPost, 0, len(posts))
	for _, p := range posts {
		docs = append(docs, indexedPost{
			ID:           p.ID,
			Title:        p.Title,
			Summary:      p.Summary,
			Content:      p.Content,
			Slug:         p.Slug,
			CategoryName: p.CategoryName,
			Tags:         p.Tags,
			CreatedAt:    p.CreatedAt,
		})
	}
	_, err := r.data.meili.Index(r.data.indexName).AddDocuments(docs, nil)
	if err != nil {
		return pb.ErrorIndexSyncFailed("%v", err)
	}
	return nil
}
