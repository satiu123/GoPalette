package data

import (
	"context"

	"github.com/satiu123/GoPalette/pkg/pagination"

	"github.com/satiu123/GoPalette/app/post/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type categoryRepo struct {
	data *Data
	log  *log.Helper
}

func NewCategoryRepo(data *Data, logger log.Logger) biz.CategoryRepo {
	return &categoryRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "repo/category")),
	}
}

func (r *categoryRepo) CreateCategory(ctx context.Context, c *biz.Category) (*biz.Category, error) {
	co := r.toDataCategory(c)
	if err := r.data.db.WithContext(ctx).Create(co).Error; err != nil {
		return nil, err
	}
	return r.GetCategoryByID(ctx, int64(co.ID))
}

func (r *categoryRepo) UpdateCategory(ctx context.Context, c *biz.Category, fields []string) (*biz.Category, error) {
	db := r.data.db.WithContext(ctx).Model(&Category{})

	if len(fields) > 0 {
		selected := r.filterCategoryFields(fields)
		if len(selected) == 0 {
			return r.GetCategoryByID(ctx, c.ID)
		}
		db = db.Select(selected)
	}

	po := r.toDataCategory(c)

	if err := db.Where("id = ?", c.ID).Updates(po).Error; err != nil {
		return nil, err
	}
	return r.GetCategoryByID(ctx, c.ID)
}

func (r *categoryRepo) DeleteCategory(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&Category{}, id).Error
}

func (r *categoryRepo) GetCategoryByID(ctx context.Context, id int64) (*biz.Category, error) {
	var co Category
	if err := r.data.db.WithContext(ctx).First(&co, id).Error; err != nil {
		return nil, err
	}
	return r.toBizCategory(&co), nil
}

func (r *categoryRepo) ListCategories(ctx context.Context, page, pageSize int64) ([]*biz.Category, int64, error) {
	p := pagination.NewPagingParam(page, pageSize)

	var rows []Category
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Category{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*biz.Category{}, 0, nil
	}

	if err := db.Order("created_at DESC").
		Scopes(pagination.Paginate(p)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	res := make([]*biz.Category, 0, len(rows))
	for i := range rows {
		res = append(res, r.toBizCategory(&rows[i]))
	}
	return res, total, nil
}

func (r *categoryRepo) toBizCategory(co *Category) *biz.Category {
	return &biz.Category{
		ID:          int64(co.ID),
		Name:        co.Name,
		Slug:        co.Slug,
		Description: co.Description,
		PostCount:   co.PostCount,
		CreatedAt:   co.CreatedAt,
		UpdatedAt:   co.UpdatedAt,
	}
}

func (r *categoryRepo) toDataCategory(c *biz.Category) *Category {
	return &Category{
		Model: gorm.Model{
			ID:        uint(c.ID),
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		},
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		PostCount:   c.PostCount,
	}
}

func (r *categoryRepo) filterCategoryFields(fields []string) []string {
	allowed := map[string]struct{}{
		"name":        {},
		"slug":        {},
		"description": {},
	}

	selected := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, ok := allowed[field]; ok {
			selected = append(selected, field)
		}
	}
	return selected
}
