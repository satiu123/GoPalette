package service

import (
	"context"

	pb "GoPalette/api/post/v1"
	"GoPalette/app/post/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type PostService struct {
	pb.UnimplementedPostServer

	uc     *biz.PostUsecase
	logger *log.Helper
}

func NewPostService(uc *biz.PostUsecase, logger log.Logger) *PostService {
	return &PostService{
		uc:     uc,
		logger: log.NewHelper(log.With(logger, "module", "service/post")),
	}
}

func (s *PostService) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostReply, error) {
	return &pb.CreatePostReply{}, nil
}
func (s *PostService) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.UpdatePostReply, error) {
	return &pb.UpdatePostReply{}, nil
}
func (s *PostService) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostReply, error) {
	return &pb.DeletePostReply{}, nil
}
func (s *PostService) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.GetPostReply, error) {
	return &pb.GetPostReply{}, nil
}
func (s *PostService) ListPost(ctx context.Context, req *pb.ListPostRequest) (*pb.ListPostReply, error) {
	return &pb.ListPostReply{}, nil
}
