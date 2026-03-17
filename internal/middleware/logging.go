package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

func newRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}

// RequestLogger 记录每个请求的关键字段，便于定位错误与性能瓶颈。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set(requestIDHeader, requestID)

		start := time.Now()
		c.Next()

		latency := time.Since(start)
		userID := c.GetInt("userID")
		role := c.GetString("role")

		slog.Info("http_request",
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_id", userID,
			"role", role,
		)
	}
}
