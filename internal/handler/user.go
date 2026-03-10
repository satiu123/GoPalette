package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/service"
)

type UserHandler struct {
	UserService *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{
		UserService: s,
	}
}

func (h *UserHandler) Login(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		slog.Error("登录请求参数绑定失败", "error", err)
		response.Error(c, 400, "请求参数错误")
		return
	}
	accessToken, refreshToken, err := h.UserService.Login(c, user.Username, user.Password)
	if err != nil {
		slog.Error("登录失败", "username", user.Username, "error", err)
		response.Error(c, 401, "用户名或密码错误")
		return
	}
	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *UserHandler) Register(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		slog.Error("注册请求参数绑定失败", "error", err)
		response.Error(c, 400, "请求参数错误")
		return
	}
	if err := h.UserService.Register(c, user.Username, user.Password, user.Role); err != nil {
		slog.Error("注册失败", "username", user.Username, "error", err)
		response.Error(c, 500, "注册失败")
		return
	}
	response.Success(c, gin.H{
		"message": "注册成功",
	})
}

func (h *UserHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("刷新请求参数绑定失败", "error", err)
		response.Error(c, 400, "请求参数错误")
		return
	}
	newAccess, newRefresh, err := h.UserService.Refresh(c, req.RefreshToken)
	if err != nil {
		slog.Error("刷新 Token 失败", "error", err)
		response.Error(c, 401, "无效的 Refresh Token")
		return
	}
	response.Success(c, gin.H{
		"access_token":  newAccess,
		"refresh_token": newRefresh,
	})
}
