package server

import "strings"

var publicOperations = map[string]struct{}{
	"/health/live":    {},
	"/health/ready":   {},
	"/health/startup": {},

	"/grpc.health.v1.Health/Check": {},
	"/grpc.health.v1.Health/Watch": {},

	"/api.bff.v1.BlogBff/ListPosts":          {},
	"/api.bff.v1.BlogBff/GetFullUserProfile": {},
	"/api.bff.v1.BlogBff/GetPost":            {},
	"/api.bff.v1.BlogBff/ListPostComments":   {},
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

	return strings.HasPrefix(operation, "/grpc.health.v1.Health/")
}
