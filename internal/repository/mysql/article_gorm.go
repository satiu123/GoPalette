package mysql

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/repository"
	"gorm.io/gorm"
)

type articleGormRepository struct {
	db *gorm.DB
}

func NewArticleGormRepository(db *gorm.DB) *articleGormRepository {
	return &articleGormRepository{db: db}
}

func (r *articleGormRepository) Create(ctx context.Context, article *model.Article) error {
	return r.db.WithContext(ctx).Create(article).Error
}

func (r *articleGormRepository) FindByID(ctx context.Context, id int64) (*model.Article, error) {
	var article model.Article
	err := r.db.WithContext(ctx).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		First(&article, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &article, nil
}

func (r *articleGormRepository) FindAll(ctx context.Context, page, pageSize int, filter repository.ListArticlesFilter) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Article{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.CategoryID > 0 {
		query = query.Where("category_id = ?", filter.CategoryID)
	}
	if filter.TagID > 0 {
		// 用子查询避免 JOIN 产生重复行
		query = query.Where("id IN (?)",
			r.db.Table("article_tags").Select("article_id").Where("tag_id = ?", filter.TagID))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&articles).Error

	return articles, total, err
}

func (r *articleGormRepository) Update(ctx context.Context, article *model.Article) error {
	// 先更新基础字段（跳过关联），再替换标签关联
	if err := r.db.WithContext(ctx).Omit("Tags").Save(article).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(article).Association("Tags").Replace(article.Tags)
}

func (r *articleGormRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Article{}, id).Error
}

func (r *articleGormRepository) IncrReadCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", id).
		UpdateColumn("read_count", gorm.Expr("read_count + 1")).Error
}
