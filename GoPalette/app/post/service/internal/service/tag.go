package service

import (
	"context"

	pb "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/post-service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type TagService struct {
	pb.UnimplementedTagServer

	uc     *biz.TagUsecase
	logger *log.Helper
}

func NewTagService(uc *biz.TagUsecase, logger log.Logger) *TagService {
	return &TagService{
		uc:     uc,
		logger: log.NewHelper(log.With(logger, "module", "service/tag")),
	}
}

func (s *TagService) CreateTag(ctx context.Context, req *pb.CreateTagRequest) (*pb.CreateTagReply, error) {
	res, err := s.uc.CreateTag(ctx, &biz.Tag{
		Name: req.Name,
		Slug: req.Slug,
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateTagReply{Tag: s.toPBDetail(res)}, nil
}

func (s *TagService) UpdateTag(ctx context.Context, req *pb.UpdateTagRequest) (*pb.UpdateTagReply, error) {
	res, err := s.uc.UpdateTag(ctx, &biz.Tag{
		ID:   req.Id,
		Name: req.Name,
		Slug: req.Slug,
	}, req.UpdateMask.GetPaths())
	if err != nil {
		return nil, err
	}
	return &pb.UpdateTagReply{Tag: s.toPBDetail(res)}, nil
}

func (s *TagService) DeleteTag(ctx context.Context, req *pb.DeleteTagRequest) (*pb.DeleteTagReply, error) {
	if err := s.uc.DeleteTag(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteTagReply{Success: true}, nil
}

func (s *TagService) GetTag(ctx context.Context, req *pb.GetTagRequest) (*pb.GetTagReply, error) {
	res, err := s.uc.GetTag(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.GetTagReply{Tag: s.toPBDetail(res)}, nil
}

func (s *TagService) ListTags(ctx context.Context, req *pb.ListTagsRequest) (*pb.ListTagsReply, error) {
	res, total, err := s.uc.ListTags(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	tags := make([]*pb.TagInfo, 0, len(res))
	for _, t := range res {
		tags = append(tags, &pb.TagInfo{Id: t.ID, Name: t.Name})
	}

	return &pb.ListTagsReply{Tags: tags, Total: total}, nil
}

func (s *TagService) toPBDetail(t *biz.Tag) *pb.TagDetail {
	return &pb.TagDetail{
		Info: &pb.TagInfo{
			Id:   t.ID,
			Name: t.Name,
		},
		Slug:      t.Slug,
		PostCount: t.PostCount,
	}
}
