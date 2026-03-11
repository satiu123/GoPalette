package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"
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
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("登录请求参数绑定失败", "error", err)
		response.Error(c, 400, "请求参数错误")
		return
	}
	accessToken, refreshToken, err := h.UserService.Login(c, req.Username, req.Password)
	if err != nil {
		slog.Error("登录失败", "username", req.Username, "error", err)
		response.Error(c, 401, "用户名或密码错误")
		return
	}
	response.Success(c, TokenResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("注册请求参数绑定失败", "error", err)
		response.Error(c, 400, "请求参数错误")
		return
	}
	if err := h.UserService.Register(c, req.Username, req.Password, req.Role); err != nil {
		slog.Error("注册失败", "username", req.Username, "error", err)
		response.Error(c, 500, "注册失败")
		return
	}
	response.Success(c, gin.H{"message": "注册成功"})
}

func (h *UserHandler) Refresh(c *gin.Context) {
	var req RefreshReq
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
	response.Success(c, TokenResp{
		AccessToken:  newAccess,
		RefreshToken: newRefresh,
	})
}
