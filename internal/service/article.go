package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/repository"
)

// ugcPolicy 允许富文本常见 HTML 标签，阻止 XSS 注入
var ugcPolicy = bluemonday.UGCPolicy()

type ArticleService struct {
	articleRepo       repository.ArticleRepository
	attachmentService *AttachmentService
	cacheRepo         repository.CacheRepository
}

func NewArticleService(articleRepo repository.ArticleRepository, attachmentService *AttachmentService, cacheRepo ...repository.CacheRepository) *ArticleService {
	svc := &ArticleService{articleRepo: articleRepo, attachmentService: attachmentService}
	if len(cacheRepo) > 0 {
		svc.cacheRepo = cacheRepo[0]
	}
	return svc
}

const (
	articleDetailTTLSeconds = 60
	articleListTTLSeconds   = 60
	articleSearchTTLSeconds = 45
	listVersionKey          = "article:cache:version:list"
	searchVersionKey        = "article:cache:version:search"
)

type articleListCacheData struct {
	Total    int64           `json:"total"`
	Articles []model.Article `json:"articles"`
}

func hashKeyword(keyword string) string {
	sum := sha256.Sum256([]byte(keyword))
	return hex.EncodeToString(sum[:8])
}

func articleDetailCacheKey(id int64) string {
	return fmt.Sprintf("article:detail:%d", id)
}

func articleListCacheKey(version int64, page, pageSize int, filter repository.ListArticlesFilter) string {
	return fmt.Sprintf("article:list:v%d:p%d:s%d:c%d:t%d:a%d:st:%s",
		version, page, pageSize, filter.CategoryID, filter.TagID, filter.AuthorID, filter.Status)
}

func articleSearchCacheKey(version int64, keyword string, page, pageSize int) string {
	return fmt.Sprintf("article:search:v%d:q:%s:p%d:s%d", version, hashKeyword(keyword), page, pageSize)
}

func (s *ArticleService) getCacheVersion(ctx context.Context, key string) int64 {
	if s.cacheRepo == nil {
		return 1
	}
	v, ok, err := s.cacheRepo.GetInt64(ctx, key)
	if err != nil {
		slog.Warn("读取缓存版本失败", "key", key, "error", err)
		return 1
	}
	if ok {
		return v
	}
	if err := s.cacheRepo.SetInt64(ctx, key, 1, 0); err != nil {
		slog.Warn("初始化缓存版本失败", "key", key, "error", err)
	}
	return 1
}

func (s *ArticleService) bumpCacheVersion(ctx context.Context, key string) {
	if s.cacheRepo == nil {
		return
	}
	if _, err := s.cacheRepo.Increment(ctx, key); err != nil {
		slog.Warn("更新缓存版本失败", "key", key, "error", err)
	}
}

func (s *ArticleService) invalidateArticleCaches(ctx context.Context, articleID int64) {
	if s.cacheRepo == nil {
		return
	}
	if articleID > 0 {
		if err := s.cacheRepo.Delete(ctx, articleDetailCacheKey(articleID)); err != nil {
			slog.Warn("删除文章详情缓存失败", "article_id", articleID, "error", err)
		}
	}
	s.bumpCacheVersion(ctx, listVersionKey)
	s.bumpCacheVersion(ctx, searchVersionKey)
}

type CreateArticleInput struct {
	Title      string
	Slug       string
	Summary    string
	Content    string
	CategoryID int64
	TagIDs     []int64
	Status     string
}

type UpdateArticleInput struct {
	Title      string
	Slug       string
	Summary    string
	Content    string
	CategoryID int64
	TagIDs     []int64
	Status     string
}

func sanitizeSlug(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}

func (s *ArticleService) resolveUniqueSlug(ctx context.Context, title, preferred string, excludeID int64) (string, error) {
	base := sanitizeSlug(preferred)
	if base == "" {
		base = sanitizeSlug(title)
	}
	if base == "" {
		base = "article"
	}

	candidate := base
	for i := 2; i <= 9999; i++ {
		existing, err := s.articleRepo.FindBySlug(ctx, candidate)
		if err != nil {
			return "", err
		}
		if existing == nil || existing.ID == excludeID {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}

	return "", errors.New("生成文章别名失败")
}

func (s *ArticleService) CreateArticle(ctx context.Context, authorID int64, input CreateArticleInput) (*model.Article, error) {
	slug, err := s.resolveUniqueSlug(ctx, input.Title, input.Slug, 0)
	if err != nil {
		return nil, err
	}

	tags := make([]model.Tag, len(input.TagIDs))
	for i, id := range input.TagIDs {
		tags[i] = model.Tag{ID: id}
	}

	status := input.Status
	if status == "" {
		status = "draft"
	}

	var categoryID *int64
	if input.CategoryID > 0 {
		categoryID = &input.CategoryID
	}

	article := &model.Article{
		Title:      input.Title,
		Slug:       slug,
		Summary:    input.Summary,
		Content:    ugcPolicy.Sanitize(input.Content),
		AuthorID:   authorID,
		CategoryID: categoryID,
		Status:     status,
		Tags:       tags,
	}

	if err := s.articleRepo.Create(ctx, article); err != nil {
		return nil, err
	}
	if s.attachmentService != nil {
		if err := s.attachmentService.BindFromContent(ctx, authorID, article.ID, article.Content); err != nil {
			return nil, err
		}
	}
	s.invalidateArticleCaches(ctx, 0)
	return article, nil
}

func (s *ArticleService) GetArticle(ctx context.Context, identifier string) (*model.Article, error) {
	var (
		id      int64
		article *model.Article
		err     error
	)

	if numericID, parseErr := strconv.ParseInt(identifier, 10, 64); parseErr == nil && numericID > 0 {
		id = numericID
	} else {
		article, err = s.articleRepo.FindBySlug(ctx, sanitizeSlug(identifier))
		if err != nil {
			return nil, err
		}
		if article == nil {
			return nil, errors.New("文章不存在")
		}
		id = article.ID
	}

	if s.cacheRepo != nil {
		var cached model.Article
		hit, err := s.cacheRepo.GetJSON(ctx, articleDetailCacheKey(id), &cached)
		if err != nil {
			slog.Warn("读取文章详情缓存失败", "article_id", id, "error", err)
		} else if hit {
			go s.articleRepo.IncrReadCount(context.Background(), id)
			return &cached, nil
		}
	}

	if article == nil {
		article, err = s.articleRepo.FindByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, errors.New("文章不存在")
	}
	if s.cacheRepo != nil {
		if err := s.cacheRepo.SetJSON(ctx, articleDetailCacheKey(id), article, articleDetailTTLSeconds); err != nil {
			slog.Warn("写入文章详情缓存失败", "article_id", id, "error", err)
		}
	}
	// 异步自增阅读数，不阻塞响应
	go s.articleRepo.IncrReadCount(context.Background(), id)
	return article, nil
}

func (s *ArticleService) ListArticles(ctx context.Context, page, pageSize int, filter repository.ListArticlesFilter) ([]model.Article, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 10
	}

	if s.cacheRepo != nil {
		version := s.getCacheVersion(ctx, listVersionKey)
		cacheKey := articleListCacheKey(version, page, pageSize, filter)
		var cached articleListCacheData
		hit, err := s.cacheRepo.GetJSON(ctx, cacheKey, &cached)
		if err != nil {
			slog.Warn("读取文章列表缓存失败", "key", cacheKey, "error", err)
		} else if hit {
			return cached.Articles, cached.Total, nil
		}

		articles, total, err := s.articleRepo.FindAll(ctx, page, pageSize, filter)
		if err != nil {
			return nil, 0, err
		}
		payload := articleListCacheData{Total: total, Articles: articles}
		if err := s.cacheRepo.SetJSON(ctx, cacheKey, payload, articleListTTLSeconds); err != nil {
			slog.Warn("写入文章列表缓存失败", "key", cacheKey, "error", err)
		}
		return articles, total, nil
	}

	return s.articleRepo.FindAll(ctx, page, pageSize, filter)
}

func (s *ArticleService) UpdateArticle(ctx context.Context, id, requesterID int64, requesterRole string, input UpdateArticleInput) (*model.Article, error) {
	article, err := s.articleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, errors.New("文章不存在")
	}
	if article.AuthorID != requesterID && requesterRole != "admin" {
		return nil, errors.New("无权限修改此文章")
	}

	var categoryID *int64
	if input.CategoryID > 0 {
		categoryID = &input.CategoryID
	}

	article.Title = input.Title
	if input.Slug != "" || article.Slug == "" {
		slug, err := s.resolveUniqueSlug(ctx, input.Title, input.Slug, article.ID)
		if err != nil {
			return nil, err
		}
		article.Slug = slug
	}
	article.Summary = input.Summary
	article.Content = ugcPolicy.Sanitize(input.Content)
	article.CategoryID = categoryID
	article.Status = input.Status

	tags := make([]model.Tag, len(input.TagIDs))
	for i, tagID := range input.TagIDs {
		tags[i] = model.Tag{ID: tagID}
	}
	article.Tags = tags

	if err := s.articleRepo.Update(ctx, article); err != nil {
		return nil, err
	}
	if s.attachmentService != nil {
		if err := s.attachmentService.BindFromContent(ctx, article.AuthorID, article.ID, article.Content); err != nil {
			return nil, err
		}
	}
	s.invalidateArticleCaches(ctx, article.ID)
	return article, nil
}

func (s *ArticleService) DeleteArticle(ctx context.Context, id, requesterID int64, requesterRole string) error {
	article, err := s.articleRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if article == nil {
		return errors.New("文章不存在")
	}
	if article.AuthorID != requesterID && requesterRole != "admin" {
		return errors.New("无权限删除此文章")
	}
	if err := s.articleRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateArticleCaches(ctx, id)
	return nil
}

func (s *ArticleService) SearchArticles(ctx context.Context, keyword string, page, pageSize int) ([]model.Article, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 10
	}

	if s.cacheRepo != nil {
		version := s.getCacheVersion(ctx, searchVersionKey)
		cacheKey := articleSearchCacheKey(version, keyword, page, pageSize)
		var cached articleListCacheData
		hit, err := s.cacheRepo.GetJSON(ctx, cacheKey, &cached)
		if err != nil {
			slog.Warn("读取搜索缓存失败", "key", cacheKey, "error", err)
		} else if hit {
			return cached.Articles, cached.Total, nil
		}

		articles, total, err := s.articleRepo.Search(ctx, keyword, page, pageSize)
		if err != nil {
			return nil, 0, err
		}
		payload := articleListCacheData{Total: total, Articles: articles}
		if err := s.cacheRepo.SetJSON(ctx, cacheKey, payload, articleSearchTTLSeconds); err != nil {
			slog.Warn("写入搜索缓存失败", "key", cacheKey, "error", err)
		}
		return articles, total, nil
	}

	return s.articleRepo.Search(ctx, keyword, page, pageSize)
}
