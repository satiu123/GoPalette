package service

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/repository"
)

type ArticleService struct {
	articleRepo repository.ArticleRepository
}

func NewArticleService(articleRepo repository.ArticleRepository) *ArticleService {
	return &ArticleService{articleRepo: articleRepo}
}

type CreateArticleInput struct {
	Title      string
	Summary    string
	Content    string
	CategoryID int64
	TagIDs     []int64
	Status     string
}

type UpdateArticleInput struct {
	Title      string
	Summary    string
	Content    string
	CategoryID int64
	TagIDs     []int64
	Status     string
}

func (s *ArticleService) CreateArticle(ctx context.Context, authorID int64, input CreateArticleInput) (*model.Article, error) {
	tags := make([]model.Tag, len(input.TagIDs))
	for i, id := range input.TagIDs {
		tags[i] = model.Tag{ID: id}
	}

	status := input.Status
	if status == "" {
		status = "draft"
	}

	article := &model.Article{
		Title:      input.Title,
		Summary:    input.Summary,
		Content:    input.Content,
		AuthorID:   authorID,
		CategoryID: input.CategoryID,
		Status:     status,
		Tags:       tags,
	}

	if err := s.articleRepo.Create(ctx, article); err != nil {
		return nil, err
	}
	return article, nil
}

func (s *ArticleService) GetArticle(ctx context.Context, id int64) (*model.Article, error) {
	article, err := s.articleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, errors.New("文章不存在")
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

	article.Title = input.Title
	article.Summary = input.Summary
	article.Content = input.Content
	article.CategoryID = input.CategoryID
	article.Status = input.Status

	tags := make([]model.Tag, len(input.TagIDs))
	for i, tagID := range input.TagIDs {
		tags[i] = model.Tag{ID: tagID}
	}
	article.Tags = tags

	if err := s.articleRepo.Update(ctx, article); err != nil {
		return nil, err
	}
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
	return s.articleRepo.Delete(ctx, id)
}
