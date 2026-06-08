package hydrator

import (
	"context"

	postv1 "github.com/satiu123/GoPalette/api/post/v1"
	userv1 "github.com/satiu123/GoPalette/api/user/v1"
	"google.golang.org/grpc"
)

type UserProvider interface {
	BatchGetUsers(ctx context.Context, req *userv1.BatchGetUsersRequest, opts ...grpc.CallOption) (*userv1.BatchGetUsersReply, error)
}

type AuthorHydrator struct {
	userClient UserProvider
}

func NewAuthorHydrator(userClient UserProvider) *AuthorHydrator {
	return &AuthorHydrator{
		userClient: userClient,
	}
}

func (h *AuthorHydrator) HydratePosts(ctx context.Context, posts []*postv1.PostInfo) error {

	if len(posts) == 0 {
		return nil
	}

	authorSet := make(map[int64]struct{})
	var ids []int64

	for _, post := range posts {
		if post == nil || post.Author == nil {
			continue
		}

		id := post.Author.Id
		if id <= 0 {
			continue
		}

		if _, ok := authorSet[id]; ok {
			continue
		}

		authorSet[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil
	}

	resp, err := h.userClient.BatchGetUsers(
		ctx,
		&userv1.BatchGetUsersRequest{
			Ids: ids,
		},
	)
	if err != nil {
		return err
	}

	users := make(map[int64]*userv1.UserProfile)

	for _, user := range resp.Users {
		users[user.Id] = user
	}

	for _, post := range posts {
		if post == nil || post.Author == nil {
			continue
		}

		if user, ok := users[post.Author.Id]; ok {
			post.Author.Name = user.Username
			post.Author.AvatarUrl = user.AvatarUrl
		}
	}

	return nil
}
