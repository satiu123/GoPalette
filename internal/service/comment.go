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

func (s *CommentService) CreateComment(ctx context.Context, articleID int64, userID *int64, parentID int64, content string) (*model.Comment, error) {
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
	// 匿名评论（UserID == nil）只有 admin 可删；有 UserID 的评论本人或 admin 可删
	if (comment.UserID == nil || *comment.UserID != requesterID) && requesterRole != "admin" {
		return errors.New("无权限删除此评论")
	}
	return s.commentRepo.Delete(ctx, id)
}

// ListAllComments 管理员分页查询全部评论（含文章信息）
func (s *CommentService) ListAllComments(ctx context.Context, page, pageSize int) ([]model.Comment, int64, error) {
	return s.commentRepo.FindAll(ctx, page, pageSize)
}

// ListMyComments 分页查询当前用户发表的评论
func (s *CommentService) ListMyComments(ctx context.Context, userID int64, page, pageSize int) ([]model.Comment, int64, error) {
	return s.commentRepo.FindByUserID(ctx, userID, page, pageSize)
}

// ListReceivedComments 分页查询当前用户文章收到的评论
func (s *CommentService) ListReceivedComments(ctx context.Context, authorID int64, page, pageSize int) ([]model.Comment, int64, error) {
	return s.commentRepo.FindReceivedByAuthorID(ctx, authorID, page, pageSize)
}
