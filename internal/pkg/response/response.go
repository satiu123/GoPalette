package response

import "github.com/gin-gonic/gin"

// 统一的返回结构
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// 成功返回
func Success(c *gin.Context, data any) {
	c.JSON(200, Response{Code: 200, Msg: "success", Data: data})
}

// 错误返回
func Error(c *gin.Context, code int, msg string) {
	c.JSON(200, Response{Code: code, Msg: msg}) // 注意：业务报错HTTP状态码也可统一用200，靠内部Code区分
}
