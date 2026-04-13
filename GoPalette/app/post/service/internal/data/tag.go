package data

import (
	"GoPalette/app/post/service/internal/biz"
	"GoPalette/pkg/pagination"
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type tagRepo struct {
	data *Data
	log  *log.Helper
}

func NewTagRepo(data *Data, logger log.Logger) biz.TagRepo {
	return &tagRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "repo/tag")),
	}
}

func (r *tagRepo) CreateTag(ctx context.Context, t *biz.Tag) (*biz.Tag, error) {
	to := r.toDataTag(t)
	if err := r.data.db.WithContext(ctx).Create(to).Error; err != nil {
		return nil, err
	}
	return r.GetTagByID(ctx, int64(to.ID))
}

func (r *tagRepo) UpdateTag(ctx context.Context, t *biz.Tag, fields []string) (*biz.Tag, error) {
	db := r.data.db.WithContext(ctx).Model(&Tag{})

	if len(fields) > 0 {
		selected := r.filterTagFields(fields)
		if len(selected) == 0 {
			return r.GetTagByID(ctx, t.ID)
		}
		db = db.Select(selected)
	}

	to := r.toDataTag(t)
	if err := db.Where("id = ?", t.ID).Updates(to).Error; err != nil {
		return nil, err
	}
	return r.GetTagByID(ctx, t.ID)
}

func (r *tagRepo) DeleteTag(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&Tag{}, id).Error
}

func (r *tagRepo) GetTagByID(ctx context.Context, id int64) (*biz.Tag, error) {
	var to Tag
	if err := r.data.db.WithContext(ctx).First(&to, id).Error; err != nil {
		return nil, err
	}
	return r.toBizTag(&to), nil
}

func (r *tagRepo) ListTags(ctx context.Context, page, pageSize int64) ([]*biz.Tag, int64, error) {
	p := pagination.NewPagingParam(page, pageSize)

	var rows []Tag
	var total int64

	db := r.data.db.WithContext(ctx).Model(&Tag{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*biz.Tag{}, 0, nil
	}

	if err := db.Order("created_at DESC").
		Scopes(pagination.Paginate(p)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	res := make([]*biz.Tag, 0, len(rows))
	for i := range rows {
		res = append(res, r.toBizTag(&rows[i]))
	}
	return res, total, nil
}

func (r *tagRepo) toBizTag(to *Tag) *biz.Tag {
	return &biz.Tag{
		ID:        int64(to.ID),
		Name:      to.Name,
		Slug:      to.Slug,
		PostCount: to.PostCount,
		CreatedAt: to.CreatedAt,
		UpdatedAt: to.UpdatedAt,
	}
}

func (r *tagRepo) toDataTag(t *biz.Tag) *Tag {
	return &Tag{
		Model: gorm.Model{
			ID:        uint(t.ID),
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		},
		Name:      t.Name,
		Slug:      t.Slug,
		PostCount: t.PostCount,
	}
}

func (r *tagRepo) filterTagFields(fields []string) []string {
	allowed := map[string]struct{}{
		"name": {},
		"slug": {},
	}

	selected := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, ok := allowed[field]; ok {
			selected = append(selected, field)
		}
	}
	return selected
}
