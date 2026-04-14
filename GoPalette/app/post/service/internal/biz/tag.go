package biz

import (
	"context"
	"strings"
	"time"

	pb "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type Tag struct {
	ID        int64
	Name      string
	Slug      string
	PostCount int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TagRepo interface {
	CreateTag(context.Context, *Tag) (*Tag, error)
	UpdateTag(context.Context, *Tag, []string) (*Tag, error)
	DeleteTag(context.Context, int64) error
	GetTagByID(context.Context, int64) (*Tag, error)
	ListTags(context.Context, int64, int64) ([]*Tag, int64, error)
}

type TagUsecase struct {
	repo   TagRepo
	logger *log.Helper
}

func NewTagUsecase(repo TagRepo, logger log.Logger) *TagUsecase {
	return &TagUsecase{
		repo:   repo,
		logger: log.NewHelper(log.With(logger, "module", "usecase/tag")),
	}
}

func (uc *TagUsecase) CreateTag(ctx context.Context, t *Tag) (*Tag, error) {
	if err := CheckAdmin(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(t.Name) == "" {
		return nil, pb.ErrorInvalidArgument("%s", "标签名称不能为空")
	}
	if strings.TrimSpace(t.Slug) == "" {
		return nil, pb.ErrorInvalidArgument("%s", "标签 slug 不能为空")
	}
	return uc.repo.CreateTag(ctx, t)
}

func (uc *TagUsecase) UpdateTag(ctx context.Context, t *Tag, fields []string) (*Tag, error) {
	if err := CheckAdmin(ctx); err != nil {
		return nil, err
	}
	if _, err := uc.repo.GetTagByID(ctx, t.ID); err != nil {
		return nil, pb.ErrorTagNotFound("标签未找到")
	}
	if len(fields) == 0 {
		return nil, pb.ErrorInvalidArgument("%s", "没有可更新字段")
	}
	validatedFields, err := validateTagUpdateFields(fields)
	if err != nil {
		return nil, err
	}
	return uc.repo.UpdateTag(ctx, t, validatedFields)
}

func (uc *TagUsecase) DeleteTag(ctx context.Context, id int64) error {
	if err := CheckAdmin(ctx); err != nil {
		return err
	}
	if _, err := uc.repo.GetTagByID(ctx, id); err != nil {
		return pb.ErrorTagNotFound("标签未找到")
	}
	return uc.repo.DeleteTag(ctx, id)
}

func (uc *TagUsecase) GetTag(ctx context.Context, id int64) (*Tag, error) {
	res, err := uc.repo.GetTagByID(ctx, id)
	if err != nil {
		return nil, pb.ErrorTagNotFound("标签未找到")
	}
	return res, nil
}

func (uc *TagUsecase) ListTags(ctx context.Context, page, pageSize int64) ([]*Tag, int64, error) {
	return uc.repo.ListTags(ctx, page, pageSize)
}

func validateTagUpdateFields(fields []string) ([]string, error) {
	allowed := map[string]struct{}{
		"name": {},
		"slug": {},
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
