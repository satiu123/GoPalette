package service

import (
	"context"

	pb "github.com/satiu123/GoPalette/api/bff/v1"
	postv1 "github.com/satiu123/GoPalette/api/post/v1"
	"github.com/satiu123/GoPalette/app/bff/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type BffService struct {
	pb.UnimplementedBlogBffServer

	uc  *biz.ProfileUsecase
	log *log.Helper
}

func NewBffService(uc *biz.ProfileUsecase, logger log.Logger) *BffService {
	return &BffService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/bff")),
	}
}

func (s *BffService) ListPosts(ctx context.Context, req *postv1.ListPostsRequest) (*postv1.ListPostsReply, error) {
	return s.uc.ListPosts(ctx, req)
}

func (s *BffService) GetFullUserProfile(ctx context.Context, req *pb.GetFullUserProfileRequest) (*pb.GetFullUserProfileReply, error) {
	return s.uc.GetFullUserProfile(ctx, req)
}
