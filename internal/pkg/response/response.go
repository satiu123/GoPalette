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

// 错误返回：429 用真实 HTTP 状态码，其他业务错误统一用 200 + code 区分
func Error(c *gin.Context, code int, msg string) {
	httpStatus := 200
	if code == 429 {
		httpStatus = 429
	}
	c.JSON(httpStatus, Response{Code: code, Msg: msg})
}
