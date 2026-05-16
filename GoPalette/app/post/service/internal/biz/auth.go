package biz

import (
	"context"

	"github.com/satiu123/GoPalette/pkg/auth"

	postPb "github.com/satiu123/GoPalette/api/post/v1"
	userPb "github.com/satiu123/GoPalette/api/user/v1"
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

func CheckIncludeNonPublishedAccess(ctx context.Context, authorID int64, includeNonPublished bool) error {
	if !includeNonPublished {
		return nil
	}
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return postPb.ErrorUnauthenticated("未认证")
	}
	if claims.UserID != authorID {
		return postPb.ErrorAccessDenied("无权查看非公开文章")
	}
	return nil
}

func CanViewPost(ctx context.Context, post *Post) bool {
	if post == nil {
		return false
	}
	if post.Status == int32(postPb.PostStatus_PUBLISHED) {
		return true
	}

	claims, ok := auth.FromContext(ctx)
	if !ok {
		return false
	}

	return claims.UserID == post.AuthorID || claims.Role == int32(userPb.Role_ADMIN)
}
