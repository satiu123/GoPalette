package service

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/repository"
)

type CommentService struct {
	commentRepo repository.CommentRepository
}

func NewCommentService(commentRepo repository.CommentRepository) *CommentService {
	return &CommentService{commentRepo: commentRepo}
}

func (s *CommentService) CreateComment(ctx context.Context, articleID, userID, parentID int64, content string) (*model.Comment, error) {
	comment := &model.Comment{
		ArticleID: articleID,
		UserID:    userID,
		Content:   content,
		ParentID:  parentID,
	}
	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *CommentService) ListComments(ctx context.Context, articleID int64) ([]model.Comment, error) {
	return s.commentRepo.FindByArticleID(ctx, articleID)
}

func (s *CommentService) DeleteComment(ctx context.Context, id, requesterID int64, requesterRole string) error {
	comment, err := s.commentRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if comment == nil {
		return errors.New("评论不存在")
	}
	if comment.UserID != requesterID && requesterRole != "admin" {
		return errors.New("无权限删除此评论")
	}
	return s.commentRepo.Delete(ctx, id)
}
