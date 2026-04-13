package biz

import (
	pb "GoPalette/api/comment/v1"
	userPb "GoPalette/api/user/v1"
	"GoPalette/pkg/auth"
	"context"
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
