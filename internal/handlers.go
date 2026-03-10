package internal

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheckHandler 健康检查接口
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "服务运行正常",
	})
}

// PingHandler ping 接口
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// HelloHandler 欢迎接口
func HelloHandler(c *gin.Context) {
	name := c.DefaultQuery("name", "World")
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello, " + name + "!",
	})
}
