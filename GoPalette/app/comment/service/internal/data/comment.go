package data

import (
	"context"
	"errors"

	pb "GoPalette/api/comment/v1"
	"GoPalette/app/comment/service/internal/biz"
	"GoPalette/pkg/pagination"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Comment struct {
	gorm.Model
	PostID    int64  `gorm:"index;not null"`
	UserID    int64  `gorm:"index;not null"`
	Content   string `gorm:"type:text;not null"`
	ParentID  int64  `gorm:"index;default:0"`
	RootID    int64  `gorm:"index;default:0"`
	LikeCount int64  `gorm:"default:0"`
	Status    int32  `gorm:"default:1"`
}

func (Comment) TableName() string { return "comments" }

type commentRepo struct {
	data *Data
	log  *log.Helper
}

func NewCommentRepo(data *Data, logger log.Logger) biz.CommentRepo {
	return &commentRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "repo/comment")),
	}
}

func (r *commentRepo) Create(ctx context.Context, c *biz.Comment) (*biz.Comment, error) {
	po := &Comment{
		PostID:    c.PostID,
		UserID:    c.UserID,
		Content:   c.Content,
		ParentID:  c.ParentID,
		RootID:    c.RootID,
		LikeCount: c.LikeCount,
		Status:    c.Status,
	}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	return r.GetByID(ctx, int64(po.ID))
}

func (r *commentRepo) GetByID(ctx context.Context, id int64) (*biz.Comment, error) {
	var po Comment
	if err := r.data.db.WithContext(ctx).First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toBizComment(&po), nil
}

func (r *commentRepo) ListRootByPost(ctx context.Context, postID, page, pageSize int64) ([]*biz.Comment, int64, error) {
	p := pagination.NewPagingParam(page, pageSize)
	var total int64
	db := r.data.db.WithContext(ctx).Model(&Comment{}).
		Where("post_id = ? AND parent_id = 0 AND status IN ?", postID, []int32{biz.CommentStatusNormal, biz.CommentStatusDeleted})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*biz.Comment{}, 0, nil
	}

	var pos []Comment
	if err := db.Order("created_at DESC").Scopes(pagination.Paginate(p)).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	out := make([]*biz.Comment, 0, len(pos))
	for i := range pos {
		out = append(out, toBizComment(&pos[i]))
	}
	return out, total, nil
}

func (r *commentRepo) ListRepliesByRootIDs(ctx context.Context, rootIDs []int64) ([]*biz.Comment, error) {
	if len(rootIDs) == 0 {
		return []*biz.Comment{}, nil
	}
	var pos []Comment
	if err := r.data.db.WithContext(ctx).
		Where("root_id IN ? AND parent_id <> 0 AND status IN ?", rootIDs, []int32{biz.CommentStatusNormal, biz.CommentStatusDeleted}).
		Order("created_at ASC").
		Find(&pos).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Comment, 0, len(pos))
	for i := range pos {
		out = append(out, toBizComment(&pos[i]))
	}
	return out, nil
}

func (r *commentRepo) UpdateRootID(ctx context.Context, id, rootID int64) error {
	return r.data.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", id).Update("root_id", rootID).Error
}

func (r *commentRepo) SoftDelete(ctx context.Context, id int64) error {
	updates := map[string]any{
		"content": "[该评论已删除]",
		"status":  biz.CommentStatusDeleted,
	}
	res := r.data.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return pb.ErrorCommentNotFound("%s", "评论不存在")
	}
	return nil
}

func toBizComment(po *Comment) *biz.Comment {
	return &biz.Comment{
		ID:        int64(po.ID),
		PostID:    po.PostID,
		UserID:    po.UserID,
		Content:   po.Content,
		ParentID:  po.ParentID,
		RootID:    po.RootID,
		LikeCount: po.LikeCount,
		Status:    po.Status,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}
