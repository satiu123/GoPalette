package service

import (
	"context"
	"testing"

	"github.com/satiu123/GoPalette/internal/model"
)

type fakeCommentRepo struct {
	createFn          func(ctx context.Context, comment *model.Comment) error
	findByArticleIDFn func(ctx context.Context, articleID int64) ([]model.Comment, error)
	findByIDFn        func(ctx context.Context, id int64) (*model.Comment, error)
	deleteFn          func(ctx context.Context, id int64) error
	findAllFn         func(ctx context.Context, page, pageSize int) ([]model.Comment, int64, error)
	findByUserIDFn    func(ctx context.Context, userID int64, page, pageSize int) ([]model.Comment, int64, error)
	findReceivedFn    func(ctx context.Context, authorID int64, page, pageSize int) ([]model.Comment, int64, error)
}

func (f *fakeCommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	if f.createFn != nil {
		return f.createFn(ctx, comment)
	}
	return nil
}

func (f *fakeCommentRepo) FindByArticleID(ctx context.Context, articleID int64) ([]model.Comment, error) {
	if f.findByArticleIDFn != nil {
		return f.findByArticleIDFn(ctx, articleID)
	}
	return nil, nil
}

func (f *fakeCommentRepo) FindByID(ctx context.Context, id int64) (*model.Comment, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeCommentRepo) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeCommentRepo) FindAll(ctx context.Context, page, pageSize int) ([]model.Comment, int64, error) {
	if f.findAllFn != nil {
		return f.findAllFn(ctx, page, pageSize)
	}
	return nil, 0, nil
}

func (f *fakeCommentRepo) FindByUserID(ctx context.Context, userID int64, page, pageSize int) ([]model.Comment, int64, error) {
	if f.findByUserIDFn != nil {
		return f.findByUserIDFn(ctx, userID, page, pageSize)
	}
	return nil, 0, nil
}

func (f *fakeCommentRepo) FindReceivedByAuthorID(ctx context.Context, authorID int64, page, pageSize int) ([]model.Comment, int64, error) {
	if f.findReceivedFn != nil {
		return f.findReceivedFn(ctx, authorID, page, pageSize)
	}
	return nil, 0, nil
}

func TestCommentServiceAnonymousDeleteNeedsAdmin(t *testing.T) {
	repo := &fakeCommentRepo{
		findByIDFn: func(ctx context.Context, id int64) (*model.Comment, error) {
			return &model.Comment{ID: id, UserID: nil}, nil
		},
	}
	svc := NewCommentService(repo)

	err := svc.DeleteComment(context.Background(), 1, 99, "user")
	if err == nil {
		t.Fatal("anonymous comment should require admin to delete")
	}
}

func TestCommentServiceOwnerCanDelete(t *testing.T) {
	uid := int64(9)
	deleted := false
	repo := &fakeCommentRepo{
		findByIDFn: func(ctx context.Context, id int64) (*model.Comment, error) {
			return &model.Comment{ID: id, UserID: &uid}, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			deleted = true
			return nil
		},
	}
	svc := NewCommentService(repo)

	err := svc.DeleteComment(context.Background(), 1, 9, "user")
	if err != nil {
		t.Fatalf("owner delete should succeed: %v", err)
	}
	if !deleted {
		t.Fatal("delete should be called")
	}
}
