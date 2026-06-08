package biz

import (
	"context"
	"strconv"

	bffv1 "github.com/satiu123/GoPalette/api/bff/v1"
	commentv1 "github.com/satiu123/GoPalette/api/comment/v1"
	postv1 "github.com/satiu123/GoPalette/api/post/v1"
	userv1 "github.com/satiu123/GoPalette/api/user/v1"
	"github.com/satiu123/GoPalette/app/bff/service/internal/biz/hydrator"
	"github.com/satiu123/GoPalette/app/bff/service/internal/data"
	"github.com/satiu123/GoPalette/pkg/auth"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/sync/errgroup"
)

type ProfileUsecase struct {
	data *data.Data
	log  *log.Helper

	authorHydrator *hydrator.AuthorHydrator
}

func NewProfileUsecase(d *data.Data, logger log.Logger) *ProfileUsecase {
	return &ProfileUsecase{
		data: d,
		log: log.NewHelper(
			log.With(logger, "module", "usecase/profile"),
		),
		authorHydrator: hydrator.NewAuthorHydrator(
			d.UserClient(),
		),
	}
}

func (uc *ProfileUsecase) GetFullUserProfile(ctx context.Context, req *bffv1.GetFullUserProfileRequest) (*bffv1.GetFullUserProfileReply, error) {
	if req == nil || req.UserId == "" {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "user_id 不能为空")
	}

	userID, err := strconv.ParseInt(req.UserId, 10, 64)
	if err != nil || userID <= 0 {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "user_id 必须是正整数")
	}

	var (
		userInfo       *userv1.UserInfo
		postStats      *postv1.GetAuthorPostStatsReply
		topPosts       []*postv1.PostInfo
		authorPosts    []*postv1.PostInfo
		recentComments []*commentv1.CommentInfo
	)
	claims, hasClaims := auth.FromContext(ctx)
	includeNonPublished := hasClaims && claims.UserID == userID

	downstreamCtx := auth.ForwardMetadataToClientContext(ctx)
	g, gctx := errgroup.WithContext(downstreamCtx)

	g.Go(func() error {
		if includeNonPublished {
			res, err := uc.data.UserClient().GetUser(gctx, &userv1.GetUserRequest{Id: userID})
			if err != nil {
				return err
			}
			userInfo = res.User
			return nil
		}

		res, err := uc.data.UserClient().BatchGetUsers(gctx, &userv1.BatchGetUsersRequest{Ids: []int64{userID}})
		if err != nil {
			return err
		}
		if len(res.Users) == 0 {
			return errors.NotFound("USER_NOT_FOUND", "用户不存在")
		}
		profile := res.Users[0]
		userInfo = &userv1.UserInfo{
			Id:        profile.Id,
			Username:  profile.Username,
			AvatarURL: profile.AvatarUrl,
		}
		return nil
	})

	g.Go(func() error {
		res, err := uc.data.PostClient().GetAuthorPostStats(gctx, &postv1.GetAuthorPostStatsRequest{
			AuthorId:            userID,
			IncludeNonPublished: includeNonPublished,
		})
		if err != nil {
			return err
		}
		postStats = res
		return nil
	})

	g.Go(func() error {
		res, err := uc.data.PostClient().ListAuthorPosts(gctx, &postv1.ListAuthorPostsRequest{
			AuthorId:            userID,
			Page:                1,
			PageSize:            20,
			IncludeNonPublished: includeNonPublished,
		})
		if err != nil {
			return err
		}
		authorPosts = res.Posts
		return nil
	})

	g.Go(func() error {
		res, err := uc.data.PostClient().ListTopAuthorPosts(gctx, &postv1.ListTopAuthorPostsRequest{
			AuthorId:            userID,
			Limit:               3,
			IncludeNonPublished: false,
		})
		if err != nil {
			return err
		}
		topPosts = res.Posts
		return nil
	})

	// 近期评论为可降级能力，不阻断主页主链路。
	g.Go(func() error {
		res, err := uc.data.CommentClient().ListUserRecentComments(gctx, &commentv1.ListUserRecentCommentsRequest{
			UserId: userID,
			Limit:  5,
		})
		if err != nil {
			uc.log.WithContext(ctx).Warnf("拉取近期评论失败 user_id=%d err=%v", userID, err)
			return nil
		}
		recentComments = res.Comments
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	hydrateAuthorPostsByUserInfo(topPosts, userInfo)
	hydrateAuthorPostsByUserInfo(authorPosts, userInfo)

	return &bffv1.GetFullUserProfileReply{
		UserInfo:       userInfo,
		PostStats:      postStats,
		RecentComments: recentComments,
		TopPosts:       topPosts,
		AuthorPosts:    authorPosts,
	}, nil
}

func hydrateAuthorPostsByUserInfo(posts []*postv1.PostInfo, userInfo *userv1.UserInfo) {
	if userInfo == nil {
		return
	}
	for _, post := range posts {
		if post == nil {
			continue
		}
		if post.Author == nil {
			post.Author = &postv1.AuthorInfo{}
		}
		post.Author.Id = userInfo.Id
		post.Author.Name = userInfo.Username
		post.Author.AvatarUrl = userInfo.AvatarURL
	}
}
