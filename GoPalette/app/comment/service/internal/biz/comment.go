package biz

import (
	"context"
	"html"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	pb "github.com/satiu123/GoPalette/api/comment/v1"
)

const (
	CommentStatusNormal  int32 = int32(pb.CommentStatus_COMMENT_STATUS_NORMAL)
	CommentStatusPending int32 = int32(pb.CommentStatus_COMMENT_STATUS_PENDING)
	CommentStatusDeleted int32 = int32(pb.CommentStatus_COMMENT_STATUS_DELETED)
)

type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Content   string
	ParentID  int64
	RootID    int64
	LikeCount int64
	Status    int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CommentRepo interface {
	Create(ctx context.Context, c *Comment) (*Comment, error)
	GetByID(ctx context.Context, id int64) (*Comment, error)
	ListAll(ctx context.Context, page, pageSize int64) ([]*Comment, int64, error)
	ListRootByPost(ctx context.Context, postID, page, pageSize int64) ([]*Comment, int64, error)
	ListRepliesByRootIDs(ctx context.Context, rootIDs []int64) ([]*Comment, error)
	CountByUser(ctx context.Context, userID int64) (int64, error)
	ListRecentByUser(ctx context.Context, userID, limit int64) ([]*Comment, error)
	UpdateRootID(ctx context.Context, id, rootID int64) error
	UpdateStatus(ctx context.Context, id int64, status int32) error
	SoftDelete(ctx context.Context, id int64) error
}

type RateLimitRepo interface {
	AllowCreate(ctx context.Context, userID int64, limit int64, window time.Duration) (bool, error)
}

type CommentEventPublisher interface {
	PublishCommentCountChanged(ctx context.Context, postID, commentID, delta int64) error
}

type CommentUsecase struct {
	repo      CommentRepo
	rateRepo  RateLimitRepo
	publisher CommentEventPublisher
	logger    *log.Helper
}

func NewCommentUsecase(repo CommentRepo, rateRepo RateLimitRepo, publisher CommentEventPublisher, logger log.Logger) *CommentUsecase {
	return &CommentUsecase{
		repo:      repo,
		rateRepo:  rateRepo,
		publisher: publisher,
		logger:    log.NewHelper(log.With(logger, "module", "usecase/comment")),
	}
}

func (uc *CommentUsecase) Create(ctx context.Context, c *Comment) (*Comment, error) {
	if c.PostID <= 0 || c.UserID <= 0 {
		return nil, pb.ErrorInvalidArgument("%s", "post_id 或 user_id 无效")
	}
	c.Content = strings.TrimSpace(c.Content)
	if c.Content == "" {
		return nil, pb.ErrorInvalidArgument("%s", "评论内容不能为空")
	}
	c.Content = SanitizeCommentContent(c.Content)

	ok, err := uc.rateRepo.AllowCreate(ctx, c.UserID, 3, time.Minute)
	if err != nil {
		uc.logger.WithContext(ctx).Warnf("rate limit check failed: %v", err)
	} else if !ok {
		return nil, pb.ErrorRateLimited("%s", "评论过于频繁，请稍后再试")
	}

	if c.ParentID > 0 {
		parent, err := uc.repo.GetByID(ctx, c.ParentID)
		if err != nil || parent == nil || parent.Status == CommentStatusDeleted {
			return nil, pb.ErrorParentNotFound("%s", "回复的评论不存在")
		}
		if parent.PostID != c.PostID {
			return nil, pb.ErrorInvalidArgument("%s", "父评论不属于该文章")
		}
		if parent.RootID > 0 {
			c.RootID = parent.RootID
		} else {
			c.RootID = parent.ID
		}
	}

	if hasSensitiveWord(c.Content) {
		c.Status = CommentStatusPending
	} else {
		c.Status = CommentStatusNormal
	}

	created, err := uc.repo.Create(ctx, c)
	if err != nil {
		return nil, err
	}

	if created.ParentID == 0 {
		if err := uc.repo.UpdateRootID(ctx, created.ID, created.ID); err == nil {
			created.RootID = created.ID
		}
	}

	if err := uc.publisher.PublishCommentCountChanged(ctx, created.PostID, created.ID, 1); err != nil {
		uc.logger.WithContext(ctx).Warnf("publish comment_count increment failed, post_id=%d comment_id=%d err=%v", created.PostID, created.ID, err)
	}

	return created, nil
}

func (uc *CommentUsecase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return pb.ErrorInvalidArgument("%s", "评论ID无效")
	}
	comment, err := uc.repo.GetByID(ctx, id)
	if err != nil || comment == nil {
		return pb.ErrorCommentNotFound("%s", "评论不存在")
	}
	if err := CheckOwner(ctx, comment.UserID); err != nil {
		return err
	}
	if comment.Status == CommentStatusDeleted {
		return nil
	}
	if err := uc.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	if err := uc.publisher.PublishCommentCountChanged(ctx, comment.PostID, comment.ID, -1); err != nil {
		uc.logger.WithContext(ctx).Warnf("publish comment_count decrement failed, post_id=%d comment_id=%d err=%v", comment.PostID, comment.ID, err)
	}
	return nil
}

func (uc *CommentUsecase) Review(ctx context.Context, id int64, status int32) (*Comment, error) {
	if err := CheckAdmin(ctx); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, pb.ErrorInvalidArgument("%s", "评论ID无效")
	}
	if status != CommentStatusNormal && status != CommentStatusDeleted {
		return nil, pb.ErrorInvalidArgument("%s", "审核状态仅支持通过或删除")
	}

	comment, err := uc.repo.GetByID(ctx, id)
	if err != nil || comment == nil {
		return nil, pb.ErrorCommentNotFound("%s", "评论不存在")
	}
	if comment.Status == status {
		return comment, nil
	}

	if status == CommentStatusDeleted {
		if err := uc.repo.SoftDelete(ctx, id); err != nil {
			return nil, err
		}
		if comment.Status != CommentStatusDeleted {
			if err := uc.publisher.PublishCommentCountChanged(ctx, comment.PostID, comment.ID, -1); err != nil {
				uc.logger.WithContext(ctx).Warnf("publish comment_count decrement failed, post_id=%d comment_id=%d err=%v", comment.PostID, comment.ID, err)
			}
		}
		return uc.repo.GetByID(ctx, id)
	}

	if err := uc.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}
	return uc.repo.GetByID(ctx, id)
}

// ListByPost 返回当前页的根评论及它们对应的子评论
func (uc *CommentUsecase) ListByPost(ctx context.Context, postID, page, pageSize int64) (roots []*Comment, replies []*Comment, total int64, err error) {
	if postID <= 0 {
		return nil, nil, 0, pb.ErrorInvalidArgument("%s", "post_id 无效")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	roots, total, err = uc.repo.ListRootByPost(ctx, postID, page, pageSize)
	if err != nil {
		return nil, nil, 0, err
	}
	if len(roots) == 0 {
		return []*Comment{}, []*Comment{}, total, nil
	}

	rootIDs := make([]int64, 0, len(roots))
	for _, c := range roots {
		rootID := c.RootID
		if rootID == 0 {
			rootID = c.ID
		}
		rootIDs = append(rootIDs, rootID)
	}

	replies, err = uc.repo.ListRepliesByRootIDs(ctx, rootIDs)
	if err != nil {
		return nil, nil, 0, err
	}

	return roots, replies, total, nil
}

// ListAll 仅返回纯评论数据列表
func (uc *CommentUsecase) ListAll(ctx context.Context, page, pageSize int64) ([]*Comment, int64, error) {
	if err := CheckAdmin(ctx); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return uc.repo.ListAll(ctx, page, pageSize)
}

func (uc *CommentUsecase) GetUserCommentStats(ctx context.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, pb.ErrorInvalidArgument("%s", "user_id 无效")
	}
	return uc.repo.CountByUser(ctx, userID)
}

// ListUserRecentComments 仅返回纯评论数据列表
func (uc *CommentUsecase) ListUserRecentComments(ctx context.Context, userID, limit int64) ([]*Comment, error) {
	if userID <= 0 {
		return nil, pb.ErrorInvalidArgument("%s", "user_id 无效")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	return uc.repo.ListRecentByUser(ctx, userID, limit)
}

func hasSensitiveWord(content string) bool {
	hitWords := []string{"spam", "赌博", "博彩", "违禁词"}
	lower := strings.ToLower(content)
	for _, w := range hitWords {
		if strings.Contains(lower, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

func SanitizeCommentContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	return html.EscapeString(html.UnescapeString(trimmed))
}
