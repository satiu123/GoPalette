package service

import (
	"context"
	"testing"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/repository"
)

type fakeArticleRepo struct {
	createFn     func(ctx context.Context, article *model.Article) error
	findByIDFn   func(ctx context.Context, id int64) (*model.Article, error)
	findBySlugFn func(ctx context.Context, slug string) (*model.Article, error)
	findAllFn    func(ctx context.Context, page, pageSize int, filter repository.ListArticlesFilter) ([]model.Article, int64, error)
	searchFn     func(ctx context.Context, keyword string, page, pageSize int) ([]model.Article, int64, error)
	updateFn     func(ctx context.Context, article *model.Article) error
	deleteFn     func(ctx context.Context, id int64) error
	incrReadFn   func(ctx context.Context, id int64) error
	createdItem  *model.Article
}

func (f *fakeArticleRepo) Create(ctx context.Context, article *model.Article) error {
	f.createdItem = article
	if f.createFn != nil {
		return f.createFn(ctx, article)
	}
	article.ID = 100
	return nil
}

func (f *fakeArticleRepo) FindByID(ctx context.Context, id int64) (*model.Article, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeArticleRepo) FindBySlug(ctx context.Context, slug string) (*model.Article, error) {
	if f.findBySlugFn != nil {
		return f.findBySlugFn(ctx, slug)
	}
	return nil, nil
}

func (f *fakeArticleRepo) FindAll(ctx context.Context, page, pageSize int, filter repository.ListArticlesFilter) ([]model.Article, int64, error) {
	if f.findAllFn != nil {
		return f.findAllFn(ctx, page, pageSize, filter)
	}
	return nil, 0, nil
}

func (f *fakeArticleRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]model.Article, int64, error) {
	if f.searchFn != nil {
		return f.searchFn(ctx, keyword, page, pageSize)
	}
	return nil, 0, nil
}

func (f *fakeArticleRepo) Update(ctx context.Context, article *model.Article) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, article)
	}
	return nil
}

func (f *fakeArticleRepo) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeArticleRepo) IncrReadCount(ctx context.Context, id int64) error {
	if f.incrReadFn != nil {
		return f.incrReadFn(ctx, id)
	}
	return nil
}

func TestArticleServiceCreateArticleDefaultsAndSanitize(t *testing.T) {
	repo := &fakeArticleRepo{}
	svc := NewArticleService(repo, nil)

	article, err := svc.CreateArticle(context.Background(), 7, CreateArticleInput{
		Title:   "test",
		Summary: "sum",
		Content: `<p>ok</p><script>alert(1)</script>`,
		Status:  "",
	})
	if err != nil {
		t.Fatalf("create article failed: %v", err)
	}
	if article.Status != "draft" {
		t.Fatalf("expected default status draft, got %q", article.Status)
	}
	if repo.createdItem == nil {
		t.Fatal("repo should receive created article")
	}
	if repo.createdItem.Content == `<p>ok</p><script>alert(1)</script>` {
		t.Fatal("content should be sanitized")
	}
}

func TestArticleServiceDeleteArticlePermissionDenied(t *testing.T) {
	repo := &fakeArticleRepo{
		findByIDFn: func(ctx context.Context, id int64) (*model.Article, error) {
			return &model.Article{ID: id, AuthorID: 2}, nil
		},
	}
	svc := NewArticleService(repo, nil)

	err := svc.DeleteArticle(context.Background(), 10, 1, "user")
	if err == nil {
		t.Fatal("delete should fail for non-owner non-admin")
	}
}
