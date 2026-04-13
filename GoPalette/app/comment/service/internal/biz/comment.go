package biz

import (
	"context"
	"strings"
	"time"

	pb "GoPalette/api/comment/v1"

	"github.com/go-kratos/kratos/v2/log"
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

type UserProfile struct {
	ID        int64
	Name      string
	AvatarURL string
}

type CommentView struct {
	Comment       *Comment
	Author        *UserProfile
	ReplyToAuthor *UserProfile
	Replies       []*CommentView
}

type CommentRepo interface {
	Create(ctx context.Context, c *Comment) (*Comment, error)
	GetByID(ctx context.Context, id int64) (*Comment, error)
	ListRootByPost(ctx context.Context, postID, page, pageSize int64) ([]*Comment, int64, error)
	ListRepliesByRootIDs(ctx context.Context, rootIDs []int64) ([]*Comment, error)
	UpdateRootID(ctx context.Context, id, rootID int64) error
	SoftDelete(ctx context.Context, id int64) error
}

type PostRepo interface {
	Exists(ctx context.Context, postID int64) (bool, error)
	IncrCommentCount(ctx context.Context, postID, delta int64) error
}

type UserRepo interface {
	BatchGetProfiles(ctx context.Context, ids []int64) (map[int64]*UserProfile, error)
}

type RateLimitRepo interface {
	AllowCreate(ctx context.Context, userID int64, limit int64, window time.Duration) (bool, error)
}

type CommentUsecase struct {
	repo     CommentRepo
	postRepo PostRepo
	userRepo UserRepo
	rateRepo RateLimitRepo
	logger   *log.Helper
}

func NewCommentUsecase(repo CommentRepo, postRepo PostRepo, userRepo UserRepo, rateRepo RateLimitRepo, logger log.Logger) *CommentUsecase {
	return &CommentUsecase{
		repo:     repo,
		postRepo: postRepo,
		userRepo: userRepo,
		rateRepo: rateRepo,
		logger:   log.NewHelper(log.With(logger, "module", "usecase/comment")),
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

	ok, err := uc.rateRepo.AllowCreate(ctx, c.UserID, 3, time.Minute)
	if err != nil {
		uc.logger.WithContext(ctx).Warnf("rate limit check failed: %v", err)
	} else if !ok {
		return nil, pb.ErrorRateLimited("%s", "评论过于频繁，请稍后再试")
	}

	exists, err := uc.postRepo.Exists(ctx, c.PostID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, pb.ErrorPostNotFound("%s", "文章不存在")
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

	if err := uc.postRepo.IncrCommentCount(ctx, created.PostID, 1); err != nil {
		uc.logger.WithContext(ctx).Warnf("incr post comment_count failed, post_id=%d err=%v", created.PostID, err)
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
	if err := uc.postRepo.IncrCommentCount(ctx, comment.PostID, -1); err != nil {
		uc.logger.WithContext(ctx).Warnf("decr post comment_count failed, post_id=%d err=%v", comment.PostID, err)
	}
	return nil
}

func (uc *CommentUsecase) ListByPost(ctx context.Context, postID, page, pageSize int64) ([]*CommentView, int64, error) {
	if postID <= 0 {
		return nil, 0, pb.ErrorInvalidArgument("%s", "post_id 无效")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	roots, total, err := uc.repo.ListRootByPost(ctx, postID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if len(roots) == 0 {
		return []*CommentView{}, total, nil
	}

	rootIDs := make([]int64, 0, len(roots))
	userIDSet := make(map[int64]struct{})
	commentMap := make(map[int64]*Comment)
	for _, c := range roots {
		rootID := c.RootID
		if rootID == 0 {
			rootID = c.ID
		}
		rootIDs = append(rootIDs, rootID)
		userIDSet[c.UserID] = struct{}{}
		commentMap[c.ID] = c
	}

	replies, err := uc.repo.ListRepliesByRootIDs(ctx, rootIDs)
	if err != nil {
		return nil, 0, err
	}
	for _, c := range replies {
		userIDSet[c.UserID] = struct{}{}
		commentMap[c.ID] = c
	}

	userIDs := make([]int64, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}
	profiles, err := uc.userRepo.BatchGetProfiles(ctx, userIDs)
	if err != nil {
		uc.logger.WithContext(ctx).Warnf("batch get users failed: %v", err)
		profiles = map[int64]*UserProfile{}
	}

	rootViews := make([]*CommentView, 0, len(roots))
	rootMap := make(map[int64]*CommentView, len(roots))
	for _, c := range roots {
		v := &CommentView{Comment: c, Author: profiles[c.UserID], Replies: []*CommentView{}}
		rootViews = append(rootViews, v)
		rootMap[c.ID] = v
	}

	for _, r := range replies {
		rootID := r.RootID
		if rootID == 0 {
			rootID = r.ParentID
		}
		target := rootMap[rootID]
		if target == nil {
			continue
		}
		var replyTo *UserProfile
		if parent, ok := commentMap[r.ParentID]; ok {
			replyTo = profiles[parent.UserID]
		}
		target.Replies = append(target.Replies, &CommentView{
			Comment:       r,
			Author:        profiles[r.UserID],
			ReplyToAuthor: replyTo,
		})
	}

	return rootViews, total, nil
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
