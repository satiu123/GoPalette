package server

import "strings"

var publicOperations = map[string]struct{}{
	"/health/live":    {},
	"/health/ready":   {},
	"/health/startup": {},

	"/grpc.health.v1.Health/Check": {},

	"/api.user.v1.User/Register":      {},
	"/api.user.v1.User/Login":         {},
	"/api.user.v1.User/RefreshToken":  {},
	"/api.user.v1.User/BatchGetUsers": {},
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
