package service

import (
	"context"

	pb "github.com/satiu123/GoPalette/api/bff/v1"
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

func (s *BffService) GetFullUserProfile(ctx context.Context, req *pb.GetFullUserProfileRequest) (*pb.GetFullUserProfileReply, error) {
	return s.uc.GetFullUserProfile(ctx, req)
}

