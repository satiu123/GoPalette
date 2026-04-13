package biz

import (
	"context"
	"strings"
	"time"

	pb "GoPalette/api/post/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type Category struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	PostCount   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CategoryRepo interface {
	CreateCategory(context.Context, *Category) (*Category, error)
	UpdateCategory(context.Context, *Category, []string) (*Category, error)
	DeleteCategory(context.Context, int64) error
	GetCategoryByID(context.Context, int64) (*Category, error)
	ListCategories(context.Context, int64, int64) ([]*Category, int64, error)
}

type CategoryUsecase struct {
	repo   CategoryRepo
	logger *log.Helper
}

func NewCategoryUsecase(repo CategoryRepo, logger log.Logger) *CategoryUsecase {
	return &CategoryUsecase{
		repo:   repo,
		logger: log.NewHelper(log.With(logger, "module", "usecase/category")),
	}
}

func (uc *CategoryUsecase) CreateCategory(ctx context.Context, c *Category) (*Category, error) {
	if err := CheckAdmin(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.Name) == "" {
		return nil, pb.ErrorInvalidArgument("%s", "分类名称不能为空")
	}
	if strings.TrimSpace(c.Slug) == "" {
		return nil, pb.ErrorInvalidArgument("%s", "分类 slug 不能为空")
	}
	return uc.repo.CreateCategory(ctx, c)
}

func (uc *CategoryUsecase) UpdateCategory(ctx context.Context, c *Category, fields []string) (*Category, error) {
	if err := CheckAdmin(ctx); err != nil {
		return nil, err
	}
	if _, err := uc.repo.GetCategoryByID(ctx, c.ID); err != nil {
		return nil, pb.ErrorCategoryNotFound("分类未找到")
	}
	if len(fields) == 0 {
		return nil, pb.ErrorInvalidArgument("%s", "没有可更新字段")
	}
	validatedFields, err := validateCategoryUpdateFields(fields)
	if err != nil {
		return nil, err
	}
	return uc.repo.UpdateCategory(ctx, c, validatedFields)
}

func (uc *CategoryUsecase) DeleteCategory(ctx context.Context, id int64) error {
	if err := CheckAdmin(ctx); err != nil {
		return err
	}
	if _, err := uc.repo.GetCategoryByID(ctx, id); err != nil {
		return pb.ErrorCategoryNotFound("分类未找到")
	}
	return uc.repo.DeleteCategory(ctx, id)
}

func (uc *CategoryUsecase) GetCategory(ctx context.Context, id int64) (*Category, error) {
	res, err := uc.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, pb.ErrorCategoryNotFound("分类未找到")
	}
	return res, nil
}

func (uc *CategoryUsecase) ListCategories(ctx context.Context, page, pageSize int64) ([]*Category, int64, error) {
	return uc.repo.ListCategories(ctx, page, pageSize)
}

func validateCategoryUpdateFields(fields []string) ([]string, error) {
	allowed := map[string]struct{}{
		"name":        {},
		"slug":        {},
		"description": {},
	}

	seen := make(map[string]struct{}, len(fields))
	validated := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, ok := allowed[field]; !ok {
			return nil, pb.ErrorInvalidArgument("%s", "不支持更新字段: "+field)
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		validated = append(validated, field)
	}

	if len(validated) == 0 {
		return nil, pb.ErrorInvalidArgument("%s", "没有可更新字段")
	}

	return validated, nil
}
