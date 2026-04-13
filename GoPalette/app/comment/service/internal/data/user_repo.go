package data

import (
	"context"

	userv1 "GoPalette/api/user/v1"
	"GoPalette/app/comment/service/internal/biz"
)

type userRepo struct {
	data *Data
}

func NewUserRepo(data *Data) biz.UserRepo {
	return &userRepo{data: data}
}

func (r *userRepo) BatchGetProfiles(ctx context.Context, ids []int64) (map[int64]*biz.UserProfile, error) {
	if len(ids) == 0 {
		return map[int64]*biz.UserProfile{}, nil
	}
	resp, err := r.data.userClient.BatchGetUsers(ctx, &userv1.BatchGetUsersRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	out := make(map[int64]*biz.UserProfile, len(resp.Users))
	for _, u := range resp.Users {
		out[u.Id] = &biz.UserProfile{ID: u.Id, Name: u.Username, AvatarURL: u.AvatarUrl}
	}
	return out, nil
}
