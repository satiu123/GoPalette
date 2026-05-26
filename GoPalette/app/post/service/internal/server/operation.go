package server

import "strings"

var publicOperations = map[string]struct{}{
	"/health/live":    {},
	"/health/ready":   {},
	"/health/startup": {},

	"/grpc.health.v1.Health/Check": {},

	"/api.post.v1.Post/GetPost":            {},
	"/api.post.v1.Post/ListPosts":          {},
	"/api.post.v1.Post/ListPostsForIndex":  {},
	"/api.post.v1.Post/ListAuthorPosts":    {},
	"/api.post.v1.Post/GetAuthorPostStats": {},
	"/api.post.v1.Post/ListTopAuthorPosts": {},
	"/api.post.v1.Post/IncrCommentCount":   {},
	"/api.post.v1.Post/RecordPostView":     {},
	"/api.post.v1.Category/GetCategory":    {},
	"/api.post.v1.Category/ListCategories": {},
	"/api.post.v1.Tag/GetTag":              {},
	"/api.post.v1.Tag/ListTags":            {},
}

func IsPublicOperation(operation string) bool {
	_, ok := publicOperations[operation]
	return ok
}

func IsHealthOperation(operation string) bool {
	if strings.HasPrefix(
		operation,
		"/health/",
	) {
		return true
	}

	return operation == "/grpc.health.v1.Health/Check"
}
