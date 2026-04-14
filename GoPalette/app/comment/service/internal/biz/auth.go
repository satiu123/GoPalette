package biz

import (
	"context"

	"github.com/satiu123/GoPalette/pkg/auth"

	pb "github.com/satiu123/GoPalette/api/comment/v1"
	userPb "github.com/satiu123/GoPalette/api/user/v1"
)

func CheckOwner(ctx context.Context, ownerID int64) error {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return pb.ErrorUnauthenticated("%s", "未认证")
	}

	isOwner := claims.UserID == ownerID
	isAdmin := claims.Role == int32(userPb.Role_ADMIN)
	if !isOwner && !isAdmin {
		return pb.ErrorAccessDenied("%s", "无权限操作该评论")
	}
	return nil
}
