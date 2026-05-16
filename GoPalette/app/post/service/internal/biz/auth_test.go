package biz

import (
	"context"
	"strconv"
	"testing"

	postPb "github.com/satiu123/GoPalette/api/post/v1"
	userPb "github.com/satiu123/GoPalette/api/user/v1"
	"github.com/satiu123/GoPalette/pkg/auth"

	"github.com/go-kratos/kratos/v2/metadata"
)

func TestCanViewPost(t *testing.T) {
	t.Parallel()

	const authorID int64 = 12

	tests := []struct {
		name string
		ctx  context.Context
		post *Post
		want bool
	}{
		{
			name: "published post is public",
			ctx:  context.Background(),
			post: &Post{AuthorID: authorID, Status: int32(postPb.PostStatus_PUBLISHED)},
			want: true,
		},
		{
			name: "draft post is hidden from anonymous users",
			ctx:  context.Background(),
			post: &Post{AuthorID: authorID, Status: int32(postPb.PostStatus_DRAFT)},
			want: false,
		},
		{
			name: "private post is hidden from anonymous users",
			ctx:  context.Background(),
			post: &Post{AuthorID: authorID, Status: int32(postPb.PostStatus_PRIVATE)},
			want: false,
		},
		{
			name: "offline post is hidden from anonymous users",
			ctx:  context.Background(),
			post: &Post{AuthorID: authorID, Status: int32(postPb.PostStatus_OFFLINE)},
			want: false,
		},
		{
			name: "author can view own non-published post",
			ctx:  contextWithAuth(authorID, int32(userPb.Role_USER)),
			post: &Post{AuthorID: authorID, Status: int32(postPb.PostStatus_PRIVATE)},
			want: true,
		},
		{
			name: "admin can view non-published post",
			ctx:  contextWithAuth(99, int32(userPb.Role_ADMIN)),
			post: &Post{AuthorID: authorID, Status: int32(postPb.PostStatus_OFFLINE)},
			want: true,
		},
		{
			name: "other user cannot view non-published post",
			ctx:  contextWithAuth(99, int32(userPb.Role_USER)),
			post: &Post{AuthorID: authorID, Status: int32(postPb.PostStatus_DRAFT)},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := CanViewPost(tt.ctx, tt.post); got != tt.want {
				t.Fatalf("CanViewPost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func contextWithAuth(userID int64, role int32) context.Context {
	md := metadata.Metadata{}
	md.Set(auth.MetadataUserIDKey, strconv.FormatInt(userID, 10))
	md.Set(auth.MetadataRoleKey, strconv.FormatInt(int64(role), 10))
	return metadata.NewServerContext(context.Background(), md)
}
