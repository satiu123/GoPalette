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
	return s.toPB(created, nil, nil), nil
}

func (s *CommentService) ListComments(ctx context.Context, req *pb.ListCommentsRequest) (*pb.ListCommentsReply, error) {
	var (
		views []*biz.CommentView
		total int64
		err   error
	)
	if req.PostId > 0 {
		views, total, err = s.uc.ListByPost(ctx, req.PostId, req.Page, req.PageSize)
	} else {
		views, total, err = s.uc.ListAll(ctx, req.Page, req.PageSize)
	}
	if err != nil {
		return nil, err
	}

	items := make([]*pb.CommentInfo, 0, len(views))
	for _, v := range views {
		root := s.toPB(v.Comment, v.Author, v.ReplyToAuthor)
		replies := make([]*pb.CommentInfo, 0, len(v.Replies))
		for _, rv := range v.Replies {
			replies = append(replies, s.toPB(rv.Comment, rv.Author, rv.ReplyToAuthor))
		}
		root.Replies = replies
		items = append(items, root)
	}

	return &pb.ListCommentsReply{Comments: items, Total: total}, nil
}

func (s *CommentService) DeleteComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.DeleteCommentReply, error) {
	if err := s.uc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteCommentReply{Success: true}, nil
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
		items = append(items, s.toPB(item, nil, nil))
	}

	return &pb.ListUserRecentCommentsReply{Comments: items}, nil
}

func (s *CommentService) toPB(c *biz.Comment, author *biz.UserProfile, replyTo *biz.UserProfile) *pb.CommentInfo {
	if c == nil {
		return nil
	}
	res := &pb.CommentInfo{
		Id:        c.ID,
		PostId:    c.PostID,
		UserId:    c.UserID,
		Content:   c.Content,
		ParentId:  c.ParentID,
		RootId:    c.RootID,
		LikeCount: c.LikeCount,
		Status:    pb.CommentStatus(c.Status),
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
	if author != nil {
		res.Author = &pb.AuthorInfo{Id: author.ID, Name: author.Name, AvatarUrl: author.AvatarURL}
	}
	if replyTo != nil {
		res.ReplyToAuthor = &pb.AuthorInfo{Id: replyTo.ID, Name: replyTo.Name, AvatarUrl: replyTo.AvatarURL}
	}
	return res
}
