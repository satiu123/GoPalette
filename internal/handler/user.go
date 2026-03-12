package handler

import (
	"log/slog"
	"net/http"

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

// GetMe 获取当前登录用户信息
// @Summary      获取当前用户信息
// @Description  需要携带 Access Token，返回用户基本信息
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=UserResp}
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := int64(c.GetInt("userID"))
	u, err := h.UserService.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "用户不存在")
		return
	}
	response.Success(c, UserResp{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
	})
}

// Login 登录
// @Summary      用户登录
// @Description  返回 Access Token 和 Refresh Token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body LoginReq true "登录信息"
// @Success      200  {object}  response.Response{data=TokenResp}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /login [post]
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

// Register 注册
// @Summary      用户注册
// @Description  创建新用户账号
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body RegisterReq true "注册信息"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /register [post]
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

// Refresh 刷新令牌
// @Summary      刷新 Token
// @Description  使用 Refresh Token 获取新的双令牌（运用轮转机制）
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body RefreshReq true "Refresh Token"
// @Success      200  {object}  response.Response{data=TokenResp}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /refresh [post]
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
