package biz

import (
	postPb "GoPalette/api/post/v1"
	userPb "GoPalette/api/user/v1"
	"GoPalette/pkg/auth"
	"context"
)

// CheckOwner 检查当前用户是否是文章的作者或者管理员

func CheckOwner(ctx context.Context, authorID int64) error {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return postPb.ErrorUnauthenticated("未认证")
	}

	// 作者本人，或者是管理员
	isOwner := claims.UserID == authorID
	isAdmin := claims.Role == int32(userPb.Role_ADMIN)

	if !isOwner && !isAdmin {
		return postPb.ErrorAccessDenied("无权限操作该文章")
	}
	return nil
}

func CheckAdmin(ctx context.Context) error {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return postPb.ErrorUnauthenticated("未认证")
	}

	if claims.Role != int32(userPb.Role_ADMIN) {
		return postPb.ErrorAccessDenied("无权限操作")
	}
	return nil
}
