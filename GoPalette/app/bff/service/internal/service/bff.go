package service

import (
	"context"

	pb "github.com/satiu123/GoPalette/api/bff/v1"
	commentv1 "github.com/satiu123/GoPalette/api/comment/v1"
	postv1 "github.com/satiu123/GoPalette/api/post/v1"
	"github.com/satiu123/GoPalette/app/bff/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type BffService struct {
	pb.UnimplementedBlogBffServer

	uc  *biz.ProfileUsecase
	pc  *biz.PostUsecase
	cc  *biz.CommentUsecase
	log *log.Helper
}

func NewBffService(uc *biz.ProfileUsecase, pc *biz.PostUsecase, cc *biz.CommentUsecase, logger log.Logger) *BffService {
	return &BffService{
		uc:  uc,
		pc:  pc,
		cc:  cc,
		log: log.NewHelper(log.With(logger, "module", "service/bff")),
	}
}

func (s *BffService) ListPosts(ctx context.Context, req *postv1.ListPostsRequest) (*postv1.ListPostsReply, error) {
	return s.pc.ListPosts(ctx, req)
}

func (s *BffService) GetFullUserProfile(ctx context.Context, req *pb.GetFullUserProfileRequest) (*pb.GetFullUserProfileReply, error) {
	return s.uc.GetFullUserProfile(ctx, req)
}

func (s *BffService) GetPost(ctx context.Context, req *postv1.GetPostRequest) (*postv1.GetPostReply, error) {
	return s.pc.GetPost(ctx, req)
}

func (s *BffService) ListPostComments(ctx context.Context, req *commentv1.ListCommentsRequest) (*pb.ListPostCommentsReply, error) {
	return s.cc.ListPostComments(ctx, req)
}
