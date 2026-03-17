package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一的返回结构
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

const (
	MsgSuccess         = "success"
	MsgBadRequest      = "请求参数错误"
	MsgUnauthorized    = "未授权访问"
	MsgForbidden       = "无权限访问"
	MsgNotFound        = "资源不存在"
	MsgInternal        = "服务器内部错误"
	MsgTooManyRequests = "请求过于频繁，请稍后再试"
)

// 成功返回
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: http.StatusOK, Msg: MsgSuccess, Data: data})
}

// 错误返回：HTTP 状态码与业务 code 保持一致。
func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, Response{Code: status, Msg: msg})
}

func BadRequest(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgBadRequest
	}
	Error(c, http.StatusBadRequest, msg)
}

func Unauthorized(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgUnauthorized
	}
	Error(c, http.StatusUnauthorized, msg)
}

func Forbidden(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgForbidden
	}
	Error(c, http.StatusForbidden, msg)
}

func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgNotFound
	}
	Error(c, http.StatusNotFound, msg)
}

func Internal(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgInternal
	}
	Error(c, http.StatusInternalServerError, msg)
}

func TooManyRequests(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgTooManyRequests
	}
	Error(c, http.StatusTooManyRequests, msg)
}
