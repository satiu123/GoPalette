package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/config"
	"github.com/satiu123/GoPalette/internal/pkg/jwt"
	"github.com/satiu123/GoPalette/internal/pkg/response"
)

// JWTOptionalMiddleware 尝试解析 Access Token，有效则写入 userID/role，无 token 或无效则跳过（不中断请求）
func JWTOptionalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.Next()
			return
		}
		claims, err := jwt.ParseToken(tokenString, config.GlobalConfig.JWT.AccessTokenSecret)
		if err == nil {
			c.Set("userID", claims.UserID)
			c.Set("role", claims.Role)
		}
		c.Next()
	}
}

// JWTAuthMiddleware 校验 Access Token 的中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中获取 JWT Token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			response.Unauthorized(c, "缺少 Authorization 头")
			c.Abort()
			return
		}

		// 解析 Access Token
		claims, err := jwt.ParseToken(tokenString, config.GlobalConfig.JWT.AccessTokenSecret)
		if err != nil {
			response.Unauthorized(c, "无效的 Token")
			c.Abort()
			return
		}

		// 将用户信息存储在上下文中，供后续处理使用
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}
