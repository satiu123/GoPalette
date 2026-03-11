package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/service"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(s *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: s}
}

// List GET /api/articles/:id/comments（公开）
func (h *CommentHandler) List(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}
	comments, err := h.commentService.ListComments(c.Request.Context(), articleID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, comments)
}

// Create POST /api/articles/:id/comments（需要 JWT）
func (h *CommentHandler) Create(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}

	var req CreateCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID := int64(c.GetInt("userID"))
	comment, err := h.commentService.CreateComment(c.Request.Context(), articleID, userID, req.ParentID, req.Content)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, comment)
}

// Delete DELETE /api/comments/:id（需要 JWT，仅本人或 admin）
func (h *CommentHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的评论 ID")
		return
	}

	requesterID := int64(c.GetInt("userID"))
	role := c.GetString("role")

	if err := h.commentService.DeleteComment(c.Request.Context(), id, requesterID, role); err != nil {
		switch err.Error() {
		case "评论不存在":
			response.Error(c, http.StatusNotFound, err.Error())
		case "无权限删除此评论":
			response.Error(c, http.StatusForbidden, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	response.Success(c, gin.H{"message": "评论已删除"})
}
