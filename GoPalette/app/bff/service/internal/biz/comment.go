package biz

import (
	"context"

	bffv1 "github.com/satiu123/GoPalette/api/bff/v1"
	commentv1 "github.com/satiu123/GoPalette/api/comment/v1"
	userv1 "github.com/satiu123/GoPalette/api/user/v1"
	"github.com/satiu123/GoPalette/app/bff/service/internal/data"
	"github.com/satiu123/GoPalette/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
)

type CommentUsecase struct {
	data *data.Data
	log  *log.Helper
}

func NewCommentUsecase(d *data.Data, logger log.Logger) *CommentUsecase {
	return &CommentUsecase{
		data: d,
		log:  log.NewHelper(log.With(logger, "module", "usecase/comment")),
	}
}

func (uc *CommentUsecase) ListPostComments(ctx context.Context, req *commentv1.ListCommentsRequest) (*bffv1.ListPostCommentsReply, error) {
	if req == nil {
		req = &commentv1.ListCommentsRequest{}
	}

	downstreamCtx := auth.ForwardMetadataToClientContext(ctx)
	res, err := uc.data.CommentClient().ListComments(downstreamCtx, req)
	if err != nil {
		return nil, err
	}
	if res == nil || len(res.Comments) == 0 {
		return &bffv1.ListPostCommentsReply{Comments: []*bffv1.CommentView{}, Total: res.GetTotal()}, nil
	}

	profiles, err := uc.loadProfiles(downstreamCtx, res.Comments)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate comment authors failed: %v", err)
		profiles = map[int64]*userv1.UserProfile{}
	}

	comments := make([]*bffv1.CommentView, 0, len(res.Comments))
	for _, item := range res.Comments {
		comments = append(comments, toCommentView(item, profiles))
	}

	return &bffv1.ListPostCommentsReply{Comments: comments, Total: res.Total}, nil
}

func (uc *CommentUsecase) loadProfiles(ctx context.Context, comments []*commentv1.CommentInfo) (map[int64]*userv1.UserProfile, error) {
	ids := make([]int64, 0)
	seen := make(map[int64]struct{})
	add := func(id int64) {
		if id <= 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	var walk func(items []*commentv1.CommentInfo)
	walk = func(items []*commentv1.CommentInfo) {
		for _, item := range items {
			if item == nil {
				continue
			}
			add(item.UserId)
			add(item.ReplyToUserId)
			walk(item.Replies)
		}
	}
	walk(comments)

	if len(ids) == 0 {
		return map[int64]*userv1.UserProfile{}, nil
	}

	resp, err := uc.data.UserClient().BatchGetUsers(ctx, &userv1.BatchGetUsersRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	profiles := make(map[int64]*userv1.UserProfile, len(resp.Users))
	for _, user := range resp.Users {
		if user == nil {
			continue
		}
		profiles[user.Id] = user
	}
	return profiles, nil
}

func toCommentView(item *commentv1.CommentInfo, profiles map[int64]*userv1.UserProfile) *bffv1.CommentView {
	if item == nil {
		return nil
	}
	view := &bffv1.CommentView{
		Id:        item.Id,
		PostId:    item.PostId,
		Content:   item.Content,
		LikeCount: item.LikeCount,
		Author:    toBffAuthor(item.UserId, profiles),
		CreatedAt: item.CreatedAt,
	}
	if item.ReplyToUserId > 0 {
		view.ReplyToAuthor = toBffAuthor(item.ReplyToUserId, profiles)
	}
	if len(item.Replies) > 0 {
		view.Replies = make([]*bffv1.CommentView, 0, len(item.Replies))
		for _, reply := range item.Replies {
			view.Replies = append(view.Replies, toCommentView(reply, profiles))
		}
	}
	return view
}

func toBffAuthor(userID int64, profiles map[int64]*userv1.UserProfile) *bffv1.AuthorInfo {
	author := &bffv1.AuthorInfo{Id: userID}
	if profile, ok := profiles[userID]; ok && profile != nil {
		author.Name = profile.Username
		author.AvatarUrl = profile.AvatarUrl
	}
	return author
}
