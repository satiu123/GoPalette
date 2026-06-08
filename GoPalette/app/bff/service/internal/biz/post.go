package biz

import (
	"context"

	postv1 "github.com/satiu123/GoPalette/api/post/v1"
	"github.com/satiu123/GoPalette/app/bff/service/internal/biz/hydrator"
	"github.com/satiu123/GoPalette/app/bff/service/internal/data"
	"github.com/satiu123/GoPalette/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
)

type PostUsecase struct {
	data           *data.Data
	log            *log.Helper
	authorHydrator *hydrator.AuthorHydrator
}

func NewPostUsecase(d *data.Data, logger log.Logger) *PostUsecase {
	return &PostUsecase{
		data: d,
		log: log.NewHelper(
			log.With(logger, "module", "usecase/post"),
		),
		authorHydrator: hydrator.NewAuthorHydrator(
			d.UserClient(),
		),
	}
}

func (uc *PostUsecase) ListPosts(ctx context.Context, req *postv1.ListPostsRequest) (*postv1.ListPostsReply, error) {
	if req == nil {
		req = &postv1.ListPostsRequest{}
	}

	downstreamCtx := auth.ForwardMetadataToClientContext(ctx)

	res, err := uc.data.PostClient().ListPosts(
		downstreamCtx,
		req,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.authorHydrator.HydratePosts(
		downstreamCtx,
		res.Posts,
	); err != nil {
		return nil, err
	}

	return res, nil
}

func (uc *PostUsecase) GetPost(ctx context.Context, req *postv1.GetPostRequest) (*postv1.GetPostReply, error) {
	downstreamCtx := auth.ForwardMetadataToClientContext(ctx)

	res, err := uc.data.PostClient().GetPost(
		downstreamCtx,
		req,
	)
	if err != nil {
		return nil, err
	}

	if res == nil ||
		res.Post == nil ||
		res.Post.Info == nil {
		return res, nil
	}

	if err := uc.authorHydrator.HydratePosts(
		downstreamCtx,
		[]*postv1.PostInfo{
			res.Post.Info,
		},
	); err != nil {
		uc.log.Warn(err)
	}

	return res, nil
}
