package service

import (
	"context"

	"github.com/satiu123/GoPalette/pkg/auth"

	pb "github.com/satiu123/GoPalette/api/comment/v1"

	"github.com/satiu123/GoPalette/app/comment/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CommentService struct {
	pb.UnimplementedCommentServer

	uc     *biz.CommentUsecase
	logger *log.Helper
}

func NewCommentService(uc *biz.CommentUsecase, logger log.Logger) *CommentService {
	return &CommentService{
		uc:     uc,
		logger: log.NewHelper(log.With(logger, "module", "service/comment")),
	}
}

func (s *CommentService) CreateComment(ctx context.Context, req *pb.CreateCommentRequest) (*pb.CommentInfo, error) {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return nil, pb.ErrorUnauthenticated("%s", "未认证的用户")
	}

	created, err := s.uc.Create(ctx, &biz.Comment{
		PostID:   req.PostId,
		UserID:   claims.UserID,
		Content:  req.Content,
		ParentID: req.ParentId,
	})
	if err != nil {
		return nil, err
	}
	return s.toPB(created), nil
}

func (s *CommentService) ListComments(ctx context.Context, req *pb.ListCommentsRequest) (*pb.ListCommentsReply, error) {
	if req.PostId > 0 {

		// 获取根评论列表和对应的子评论列表
		roots, replies, total, err := s.uc.ListByPost(ctx, req.PostId, req.Page, req.PageSize)
		if err != nil {
			return nil, err
		}

		// 映射根评论到 PB 结构体，并建立 ID 索引
		rootMap := make(map[int64]*pb.CommentInfo, len(roots))
		commentMap := make(map[int64]*biz.Comment, len(roots)+len(replies))
		items := make([]*pb.CommentInfo, 0, len(roots))
		for _, r := range roots {
			pbRoot := s.toPB(r)
			items = append(items, pbRoot)
			rootMap[r.ID] = pbRoot
			commentMap[r.ID] = r
		}
		for _, rep := range replies {
			commentMap[rep.ID] = rep
		}

		// 将子评论根据 RootID 分组归档到对应的根评论下
		for _, rep := range replies {
			rootID := rep.RootID
			if rootID == 0 {
				rootID = rep.ParentID
			}
			if pbRoot, ok := rootMap[rootID]; ok {
				pbReply := s.toPB(rep)
				if parent, ok := commentMap[rep.ParentID]; ok && parent != nil {
					pbReply.ReplyToUserId = parent.UserID
				}
				pbRoot.Replies = append(pbRoot.Replies, pbReply)
			}
		}

		return &pb.ListCommentsReply{Comments: items, Total: total}, nil
	}
	// 全量后台列表查询
	comments, total, err := s.uc.ListAll(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.CommentInfo, 0, len(comments))
	for _, c := range comments {
		items = append(items, s.toPB(c))
	}

	return &pb.ListCommentsReply{Comments: items, Total: total}, nil
}

func (s *CommentService) DeleteComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.DeleteCommentReply, error) {
	if err := s.uc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteCommentReply{Success: true}, nil
}

func (s *CommentService) ReviewComment(ctx context.Context, req *pb.ReviewCommentRequest) (*pb.CommentInfo, error) {
	comment, err := s.uc.Review(ctx, req.Id, int32(req.Status))
	if err != nil {
		return nil, err
	}
	return s.toPB(comment), nil
}

func (s *CommentService) GetUserCommentStats(ctx context.Context, req *pb.GetUserCommentStatsRequest) (*pb.GetUserCommentStatsReply, error) {
	total, err := s.uc.GetUserCommentStats(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &pb.GetUserCommentStatsReply{Total: total}, nil
}

func (s *CommentService) ListUserRecentComments(ctx context.Context, req *pb.ListUserRecentCommentsRequest) (*pb.ListUserRecentCommentsReply, error) {
	comments, err := s.uc.ListUserRecentComments(ctx, req.UserId, req.Limit)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.CommentInfo, 0, len(comments))
	for _, item := range comments {
		items = append(items, s.toPB(item))
	}

	return &pb.ListUserRecentCommentsReply{Comments: items}, nil
}

func (s *CommentService) toPB(c *biz.Comment) *pb.CommentInfo {
	if c == nil {
		return nil
	}
	res := &pb.CommentInfo{
		Id:        c.ID,
		PostId:    c.PostID,
		UserId:    c.UserID,
		Content:   biz.SanitizeCommentContent(c.Content),
		ParentId:  c.ParentID,
		RootId:    c.RootID,
		LikeCount: c.LikeCount,
		Status:    pb.CommentStatus(c.Status),
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
	return res
}
