package data

import (
	"context"

	postv1 "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/app/comment/service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

type postRepo struct {
	data *Data
}

func NewPostRepo(data *Data) biz.PostRepo {
	return &postRepo{data: data}
}

func (r *postRepo) Exists(ctx context.Context, postID int64) (bool, error) {
	resp, err := r.data.postClient.GetPost(ctx, &postv1.GetPostRequest{Query: &postv1.GetPostRequest_Id{Id: postID}})
	if err != nil {
		e := kerrors.FromError(err)
		if e != nil && e.Reason == "POST_NOT_FOUND" {
			return false, nil
		}
		return false, err
	}
	return resp != nil && resp.Post != nil && resp.Post.Info != nil, nil
}

func (r *postRepo) IncrCommentCount(ctx context.Context, postID, delta int64) error {
	_, err := r.data.postClient.IncrCommentCount(ctx, &postv1.IncrCommentCountRequest{Id: postID, Delta: delta})
	return err
}
