package service

import (
	"context"

	pb "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/post-service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type CategoryService struct {
	pb.UnimplementedCategoryServer

	uc     *biz.CategoryUsecase
	logger *log.Helper
}

func NewCategoryService(uc *biz.CategoryUsecase, logger log.Logger) *CategoryService {
	return &CategoryService{
		uc:     uc,
		logger: log.NewHelper(log.With(logger, "module", "service/category")),
	}
}

func (s *CategoryService) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryReply, error) {
	res, err := s.uc.CreateCategory(ctx, &biz.Category{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateCategoryReply{Category: s.toPBDetail(res)}, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.UpdateCategoryReply, error) {
	res, err := s.uc.UpdateCategory(ctx, &biz.Category{
		ID:          req.Id,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}, req.UpdateMask.GetPaths())
	if err != nil {
		return nil, err
	}
	return &pb.UpdateCategoryReply{Category: s.toPBDetail(res)}, nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*pb.DeleteCategoryReply, error) {
	if err := s.uc.DeleteCategory(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteCategoryReply{Success: true}, nil
}

func (s *CategoryService) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.GetCategoryReply, error) {
	res, err := s.uc.GetCategory(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.GetCategoryReply{Category: s.toPBDetail(res)}, nil
}

func (s *CategoryService) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesReply, error) {
	res, total, err := s.uc.ListCategories(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	categories := make([]*pb.CategoryInfo, 0, len(res))
	for _, c := range res {
		categories = append(categories, &pb.CategoryInfo{Id: c.ID, Name: c.Name})
	}

	return &pb.ListCategoriesReply{Categories: categories, Total: total}, nil
}

func (s *CategoryService) toPBDetail(c *biz.Category) *pb.CategoryDetail {
	return &pb.CategoryDetail{
		Info: &pb.CategoryInfo{
			Id:   c.ID,
			Name: c.Name,
		},
		Slug:        c.Slug,
		Description: c.Description,
		PostCount:   c.PostCount,
	}
}
