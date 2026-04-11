package service

import (
	"context"

	pb "GoPalette/api/post/v1"
	"GoPalette/app/post/service/internal/biz"
	"GoPalette/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return nil, pb.ErrorUnauthenticated("%s", "未认证的用户")
	}

	p := &biz.Post{
		Title:   req.Title,
		Summary: req.Summary,
		Content: req.Content,
		Slug:    req.Slug,
		Status:  int32(req.Status),

		AuthorID:   claims.UserID,
		CategoryID: req.CategoryId,
		Tags:       req.Tags,
	}
	createdPost, err := s.uc.CreatePost(ctx, p)
	if err != nil {
		return nil, err
	}
	return &pb.CreatePostReply{
		Post: s.toPBDetail(createdPost),
	}, nil
}
func (s *PostService) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.UpdatePostReply, error) {
	p := &biz.Post{
		ID:      req.Id,
		Title:   req.Title,
		Summary: req.Summary,
		Content: req.Content,
		Slug:    req.Slug,
		Status:  int32(req.Status),

		CategoryID: req.CategoryId,
		Tags:       req.Tags,
	}
	updatedPost, err := s.uc.UpdatePost(ctx, p, req.UpdateMask.GetPaths())
	if err != nil {
		return nil, err
	}
	return &pb.UpdatePostReply{
		Post: s.toPBDetail(updatedPost),
	}, nil
}
func (s *PostService) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostReply, error) {
	err := s.uc.DeletePost(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.DeletePostReply{Success: true}, nil
}
func (s *PostService) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.GetPostReply, error) {
	var id int64
	var slug string
	switch x := req.Query.(type) {
	case *pb.GetPostRequest_Id:
		id = x.Id
	case *pb.GetPostRequest_Slug:
		slug = x.Slug
	default:
		return nil, pb.ErrorInvalidArgument("%s", "必须提供文章ID或Slug")
	}
	post, err := s.uc.GetPost(ctx, id, slug)
	if err != nil {
		return nil, err
	}
	return &pb.GetPostReply{
		Post: s.toPBDetail(post),
	}, nil
}
func (s *PostService) ListPost(ctx context.Context, req *pb.ListPostRequest) (*pb.ListPostReply, error) {
	res, total, err := s.uc.ListPosts(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.PostInfo, len(res))
	for i, p := range res {
		posts[i] = s.toPBInfo(p)
	}

	return &pb.ListPostReply{
		Posts: posts,
		Total: total,
	}, nil

}

func (s *PostService) toPBInfo(p *biz.Post) *pb.PostInfo {
	return &pb.PostInfo{
		Id:           p.ID,
		Title:        p.Title,
		Summary:      p.Summary,
		Slug:         p.Slug,
		Status:       pb.PostStatus(p.Status),
		ViewCount:    p.ViewCount,
		LikeCount:    p.LikeCount,
		CommentCount: p.CommentCount,
		Author: &pb.AuthorInfo{
			Id: p.AuthorID,
			// 此处后续通过rpc调用用户服务获取作者名称，目前先使用占位符
		},
		Category: &pb.CategoryInfo{
			Id:   p.CategoryID,
			Name: p.CategoryName,
		},
		Tags:      p.Tags,
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func (s *PostService) toPBDetail(p *biz.Post) *pb.PostDetail {
	return &pb.PostDetail{
		Info:            s.toPBInfo(p),
		Content:         p.Content,
		OriginalContent: p.OriginalContent,
	}
}
