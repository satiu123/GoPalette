package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/casbin"
	"github.com/satiu123/GoPalette/internal/pkg/response"
)

// RBACMiddleware 使用 Casbin 对已认证请求进行路由级授权校验。
func RBACMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if casbin.Enforcer == nil {
			slog.Error("Casbin enforcer 未初始化")
			response.Error(c, http.StatusInternalServerError, "权限系统未初始化")
			c.Abort()
			return
		}

		subject := c.GetString("role")
		if subject == "" {
			subject = "anonymous"
		}

		object := c.FullPath()
		if object == "" {
			object = c.Request.URL.Path
		}
		action := c.Request.Method

		ok, err := casbin.Enforcer.Enforce(subject, object, action)
		if err != nil {
			slog.Error("Casbin 鉴权失败", "error", err, "sub", subject, "obj", object, "act", action)
			response.Error(c, http.StatusInternalServerError, "权限校验失败")
			c.Abort()
			return
		}
		if !ok {
			response.Forbidden(c, response.MsgForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
