package util

import (
	"strings"
)

// 清洗UpdateMask
func CleanUpdateMask(paths []string, prefix string) []string {
	var updateFields []string
	for _, path := range paths {
		updateFields = append(updateFields, strings.TrimPrefix(path, prefix))
	}
	return updateFields
}
